IMAGE       ?= ghcr.io/taalib25/runtime-node-core
TAG         ?= 0.1.0
AGENT_IMAGE ?= nousresearch/hermes-agent:sha-839cdd1b054a75ff1b581199a83488c8e0f2f788
WEBUI_REF   ?= master
PLATFORM    ?= linux/amd64
NAMESPACE   ?= zamo-workspace
RELEASE     ?= runtime-node-core
CHART       ?= ../charts/runtime-node-core
KIND_CLUSTER ?= hermes-test

.PHONY: verify-agent build push release kind-load \
        k8s-apply k8s-port-forward diagnose \
        helm-lint helm-template helm-install helm-rollback

verify-agent:
	./scripts/verify-agent-source.sh $(AGENT_IMAGE)

build:
	docker buildx build \
	  --platform $(PLATFORM) \
	  --build-arg HERMES_AGENT_IMAGE=$(AGENT_IMAGE) \
	  --build-arg HERMES_WEBUI_REF=$(WEBUI_REF) \
	  --build-arg BUILD_DATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ) \
	  --build-arg VCS_REF=$$(git rev-parse --short HEAD 2>/dev/null || echo local) \
	  --load \
	  -t $(IMAGE):$(TAG) \
	  .

push:
	docker push $(IMAGE):$(TAG)

release: build push

kind-load:
	kind load docker-image $(IMAGE):$(TAG) --name $(KIND_CLUSTER)

k8s-apply:
	kubectl apply -f k8s/

k8s-port-forward:
	kubectl -n $(NAMESPACE) port-forward svc/$(RELEASE) 8787:8787

diagnose:
	@POD=$$(kubectl -n $(NAMESPACE) get pod -l app.kubernetes.io/instance=$(RELEASE) -o jsonpath='{.items[0].metadata.name}'); \
	kubectl -n $(NAMESPACE) exec -it $$POD -- hermescloud-diagnose

helm-lint:
	helm lint $(CHART)

helm-template:
	helm template $(RELEASE) $(CHART) \
	  --namespace $(NAMESPACE) \
	  --set image.repository=$(IMAGE) \
	  --set image.tag=$(TAG)

helm-install:
	helm upgrade --install $(RELEASE) $(CHART) \
	  --namespace $(NAMESPACE) --create-namespace \
	  --set image.repository=$(IMAGE) \
	  --set image.tag=$(TAG)

helm-rollback:
	helm rollback $(RELEASE) --namespace $(NAMESPACE)
