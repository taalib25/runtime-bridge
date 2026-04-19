package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	runtimev1alpha1 "github.com/hermeshq/hermes-runtime-operator/api/v1alpha1"
	"github.com/hermeshq/hermes-runtime-operator/internal/runtimeutil"
)

const (
	sharedSecretHeader = "X-Controller-Secret"
	maxSecretBytes     = 1 << 20
)

var secretKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Server exposes a compatibility API that an external backend can call.
type Server struct {
	Client       client.Client
	Log          logr.Logger
	SharedSecret string
	Namespace    string
	Scheme       *k8sruntime.Scheme
	HTTPClient   *http.Client
}

// CreateRuntimeRequest is the POST /runtimes request body.
type CreateRuntimeRequest struct {
	TenantID       string                                   `json:"tenantId"`
	RuntimeID      string                                   `json:"runtimeId"`
	Image          string                                   `json:"image"`
	Plan           runtimev1alpha1.RuntimePlan              `json:"plan"`
	Storage        runtimev1alpha1.RuntimeStorage           `json:"storage"`
	Network        runtimev1alpha1.RuntimeNetwork           `json:"network"`
	Config         json.RawMessage                          `json:"config,omitempty"`
	Soul           string                                   `json:"soul,omitempty"`
	Env            []runtimev1alpha1.RuntimeEnvVar          `json:"env,omitempty"`
	EnvFromSecrets []runtimev1alpha1.RuntimeSecretReference `json:"envFromSecrets,omitempty"`
	RequestID      string                                   `json:"requestId,omitempty"`
	ActorID        string                                   `json:"actorId,omitempty"`
}

// RuntimeResponse is returned from runtime lifecycle endpoints.
type RuntimeResponse struct {
	Name               string                       `json:"name"`
	RuntimeID          string                       `json:"runtimeId"`
	TenantID           string                       `json:"tenantId"`
	Phase              runtimev1alpha1.RuntimePhase `json:"phase,omitempty"`
	URL                string                       `json:"url,omitempty"`
	Namespace          string                       `json:"namespace,omitempty"`
	DeploymentName     string                       `json:"deploymentName,omitempty"`
	ServiceName        string                       `json:"serviceName,omitempty"`
	IngressName        string                       `json:"ingressName,omitempty"`
	ReadyReplicas      int32                        `json:"readyReplicas,omitempty"`
	DesiredReplicas    int32                        `json:"desiredReplicas,omitempty"`
	Message            string                       `json:"message,omitempty"`
	ObservedGeneration int64                        `json:"observedGeneration,omitempty"`
	CreatedAt          metav1.Time                  `json:"createdAt,omitempty"`
}

// RuntimeHealthResponse is returned from GET /runtimes/{id}/health.
type RuntimeHealthResponse struct {
	RuntimeID       string                       `json:"runtimeId"`
	Phase           runtimev1alpha1.RuntimePhase `json:"phase,omitempty"`
	Healthy         bool                         `json:"healthy"`
	Namespace       string                       `json:"namespace,omitempty"`
	ServiceURL      string                       `json:"serviceUrl,omitempty"`
	ReadyReplicas   int32                        `json:"readyReplicas,omitempty"`
	DesiredReplicas int32                        `json:"desiredReplicas,omitempty"`
	AppStatusCode   int                          `json:"appStatusCode,omitempty"`
	Message         string                       `json:"message,omitempty"`
}

type UpsertRuntimeSecretsRequest struct {
	Data      map[string]string `json:"data"`
	RequestID string            `json:"requestId,omitempty"`
	ActorID   string            `json:"actorId,omitempty"`
}

type RuntimeSecretsResponse struct {
	RuntimeID       string   `json:"runtimeId"`
	SecretName      string   `json:"secretName"`
	Exists          bool     `json:"exists"`
	Keys            []string `json:"keys"`
	EnvFromAttached bool     `json:"envFromAttached"`
}

// NewServer returns a configured compatibility API server.
func NewServer(k8sClient client.Client, scheme *k8sruntime.Scheme, logger logr.Logger, sharedSecret string, namespace string) *Server {
	return &Server{
		Client:       k8sClient,
		Log:          logger,
		SharedSecret: sharedSecret,
		Namespace:    namespace,
		Scheme:       scheme,
		HTTPClient:   &http.Client{Timeout: 3 * time.Second},
	}
}

// Handler returns the API handler tree.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("POST /runtimes", s.auth(s.createRuntime))
	mux.HandleFunc("GET /runtimes", s.auth(s.listRuntimes))
	mux.HandleFunc("GET /runtimes/{id}", s.auth(s.getRuntime))
	mux.HandleFunc("PUT /runtimes/{id}", s.auth(s.updateRuntime))
	mux.HandleFunc("GET /runtimes/{id}/secrets", s.auth(s.getRuntimeSecrets))
	mux.HandleFunc("POST /runtimes/{id}/secrets", s.auth(s.upsertRuntimeSecrets))
	mux.HandleFunc("DELETE /runtimes/{id}/secrets/{key}", s.auth(s.deleteRuntimeSecretKey))
	mux.HandleFunc("DELETE /runtimes/{id}", s.auth(s.deleteRuntime))
	mux.HandleFunc("GET /runtimes/{id}/health", s.auth(s.getRuntimeHealth))
	return mux
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provided := r.Header.Get(sharedSecretHeader)
		if s.SharedSecret == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(s.SharedSecret)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r)
	}
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) createRuntime(w http.ResponseWriter, r *http.Request) {
	var req CreateRuntimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if strings.TrimSpace(req.TenantID) == "" || strings.TrimSpace(req.RuntimeID) == "" || strings.TrimSpace(req.Image) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tenantId, runtimeId, and image are required"})
		return
	}
	if err := validateExternalSecretRefs(req.RuntimeID, req.EnvFromSecrets); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	name := req.RuntimeID
	namespace := s.runtimeNamespace()
	existing := &runtimev1alpha1.Runtime{}
	if err := s.Client.Get(r.Context(), types.NamespacedName{Name: name, Namespace: namespace}, existing); err == nil {
		// Runtime already exists, return it
		writeJSON(w, http.StatusOK, runtimeResponse(existing))
		return
	} else if !apierrors.IsNotFound(err) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("lookup runtime: %v", err)})
		return
	}

	obj := &runtimev1alpha1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":    "runtime",
				"hermescloud.io/runtime-id": req.RuntimeID,
			},
		},
		Spec: runtimev1alpha1.RuntimeSpec{
			TenantID:       req.TenantID,
			RuntimeID:      req.RuntimeID,
			Image:          req.Image,
			Plan:           req.Plan,
			Storage:        req.Storage,
			Network:        req.Network,
			Config:         rawExtension(req.Config),
			Soul:           req.Soul,
			Env:            req.Env,
			EnvFromSecrets: withManagedSecretRefIfPresent(req.RuntimeID, req.EnvFromSecrets, nil),
		},
	}
	if req.RequestID != "" {
		obj.Annotations = map[string]string{"hermescloud.io/request-id": req.RequestID}
	}
	if req.ActorID != "" {
		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}
		obj.Annotations["hermescloud.io/actor-id"] = req.ActorID
	}

	if err := s.Client.Create(r.Context(), obj); err != nil {
		status := http.StatusInternalServerError
		if apierrors.IsInvalid(err) {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, runtimeResponse(obj))
}

func (s *Server) updateRuntime(w http.ResponseWriter, r *http.Request) {
	runtimeID := r.PathValue("id")
	rt, err := s.fetchRuntime(r.Context(), runtimeID)
	if err != nil {
		writeLookupError(w, err)
		return
	}

	var req CreateRuntimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if strings.TrimSpace(req.RuntimeID) != "" && req.RuntimeID != rt.Spec.RuntimeID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "runtimeId is immutable"})
		return
	}
	if strings.TrimSpace(req.TenantID) != "" && req.TenantID != rt.Spec.TenantID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tenantId is immutable"})
		return
	}
	if err := validateExternalSecretRefs(rt.Spec.RuntimeID, req.EnvFromSecrets); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	managedSecret, err := s.fetchManagedSecret(r.Context(), rt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	rt.Spec.Image = req.Image
	rt.Spec.Plan = req.Plan
	rt.Spec.Storage = req.Storage
	rt.Spec.Network = req.Network
	rt.Spec.Config = rawExtension(req.Config)
	rt.Spec.Soul = req.Soul
	rt.Spec.Env = req.Env
	rt.Spec.EnvFromSecrets = withManagedSecretRefIfPresent(rt.Spec.RuntimeID, req.EnvFromSecrets, managedSecret)

	if req.RequestID != "" {
		if rt.Annotations == nil {
			rt.Annotations = make(map[string]string)
		}
		rt.Annotations["hermescloud.io/request-id"] = req.RequestID
	}
	if req.ActorID != "" {
		if rt.Annotations == nil {
			rt.Annotations = make(map[string]string)
		}
		rt.Annotations["hermescloud.io/actor-id"] = req.ActorID
	}

	if err := s.Client.Update(r.Context(), rt); err != nil {
		status := http.StatusInternalServerError
		if apierrors.IsInvalid(err) {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, runtimeResponse(rt))
}

func (s *Server) getRuntimeSecrets(w http.ResponseWriter, r *http.Request) {
	rt, err := s.fetchRuntime(r.Context(), r.PathValue("id"))
	if err != nil {
		writeLookupError(w, err)
		return
	}
	secret, err := s.fetchManagedSecret(r.Context(), rt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, runtimeSecretsResponse(rt, secret))
}

func (s *Server) upsertRuntimeSecrets(w http.ResponseWriter, r *http.Request) {
	rt, err := s.fetchRuntime(r.Context(), r.PathValue("id"))
	if err != nil {
		writeLookupError(w, err)
		return
	}
	var req UpsertRuntimeSecretsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if len(req.Data) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "data must include at least one key"})
		return
	}
	for key := range req.Data {
		if !secretKeyPattern.MatchString(key) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid secret key %q", key)})
			return
		}
	}
	secret, err := s.fetchManagedSecret(r.Context(), rt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if secret == nil {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: runtimeutil.ManagedSecretName(rt.Spec.RuntimeID), Namespace: rt.Namespace},
			Type:       corev1.SecretTypeOpaque,
		}
	}
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	for key, value := range req.Data {
		secret.Data[key] = []byte(value)
	}
	if totalSecretBytes(secret.Data) > maxSecretBytes {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "secret data exceeds 1 MiB limit"})
		return
	}
	secret.Labels = runtimeutil.Labels(rt)
	if err := controllerutil.SetControllerReference(rt, secret, s.Scheme); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if secret.CreationTimestamp.IsZero() {
		if err := s.Client.Create(r.Context(), secret); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	} else {
		if err := s.Client.Update(r.Context(), secret); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	if err := s.ensureManagedSecretRef(r.Context(), rt, secret); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	rt, err = s.fetchRuntime(r.Context(), rt.Spec.RuntimeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	updated, err := s.fetchManagedSecret(r.Context(), rt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	response := runtimeSecretsResponse(rt, updated)
	response.EnvFromAttached = true
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) deleteRuntimeSecretKey(w http.ResponseWriter, r *http.Request) {
	rt, err := s.fetchRuntime(r.Context(), r.PathValue("id"))
	if err != nil {
		writeLookupError(w, err)
		return
	}
	key := r.PathValue("key")
	if !secretKeyPattern.MatchString(key) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid secret key %q", key)})
		return
	}
	secret, err := s.fetchManagedSecret(r.Context(), rt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if secret == nil {
		writeJSON(w, http.StatusOK, runtimeSecretsResponse(rt, nil))
		return
	}
	delete(secret.Data, key)
	if len(secret.Data) == 0 {
		if err := s.Client.Delete(r.Context(), secret); err != nil && !apierrors.IsNotFound(err) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := s.removeManagedSecretRef(r.Context(), rt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		rt, err = s.fetchRuntime(r.Context(), rt.Spec.RuntimeID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, runtimeSecretsResponse(rt, nil))
		return
	}
	if err := s.Client.Update(r.Context(), secret); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	rt, err = s.fetchRuntime(r.Context(), rt.Spec.RuntimeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, runtimeSecretsResponse(rt, secret))
}

func (s *Server) getRuntime(w http.ResponseWriter, r *http.Request) {
	rt, err := s.fetchRuntime(r.Context(), r.PathValue("id"))
	if err != nil {
		writeLookupError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, runtimeResponse(rt))
}

func (s *Server) deleteRuntime(w http.ResponseWriter, r *http.Request) {
	rt, err := s.fetchRuntime(r.Context(), r.PathValue("id"))
	if err != nil {
		writeLookupError(w, err)
		return
	}
	if err := s.Client.Delete(r.Context(), rt); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"runtimeId": rt.Spec.RuntimeID,
		"phase":     string(runtimev1alpha1.RuntimePhaseDeleting),
		"message":   "Runtime deletion requested",
	})
}

func (s *Server) listRuntimes(w http.ResponseWriter, r *http.Request) {
	tenantFilter := r.URL.Query().Get("tenantId")
	list := &runtimev1alpha1.RuntimeList{}
	if err := s.Client.List(r.Context(), list, client.InNamespace(s.runtimeNamespace())); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	items := make([]RuntimeResponse, 0, len(list.Items))
	for _, item := range list.Items {
		if tenantFilter != "" && item.Spec.TenantID != tenantFilter {
			continue
		}
		items = append(items, runtimeResponse(&item))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].RuntimeID < items[j].RuntimeID })
	writeJSON(w, http.StatusOK, map[string]any{"runtimes": items})
}

func (s *Server) getRuntimeHealth(w http.ResponseWriter, r *http.Request) {
	rt, err := s.fetchRuntime(r.Context(), r.PathValue("id"))
	if err != nil {
		writeLookupError(w, err)
		return
	}

	namespace := rt.Status.Namespace
	if namespace == "" {
		namespace = rt.Namespace
	}

	response := RuntimeHealthResponse{
		RuntimeID:  rt.Spec.RuntimeID,
		Phase:      rt.Status.Phase,
		Namespace:  namespace,
		ServiceURL: fmt.Sprintf("http://%s.%s.svc.cluster.local/health", generateRuntimeServiceName(rt.Spec.RuntimeID), namespace),
		Message:    rt.Status.Message,
	}

	deployment := &appsv1.Deployment{}
	if err := s.Client.Get(r.Context(), types.NamespacedName{Name: generateRuntimeDeploymentName(rt.Spec.RuntimeID), Namespace: namespace}, deployment); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSON(w, http.StatusOK, response)
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	response.ReadyReplicas = deployment.Status.ReadyReplicas
	response.DesiredReplicas = *deployment.Spec.Replicas
	response.Healthy = deployment.Status.ReadyReplicas > 0 && deployment.Status.AvailableReplicas > 0

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, response.ServiceURL, nil)
	if err == nil {
		httpResp, httpErr := s.HTTPClient.Do(req)
		if httpErr == nil {
			response.AppStatusCode = httpResp.StatusCode
			response.Healthy = response.Healthy && httpResp.StatusCode == http.StatusOK
			_ = httpResp.Body.Close()
		} else if response.Message == "" {
			response.Message = httpErr.Error()
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) fetchRuntime(ctx context.Context, runtimeID string) (*runtimev1alpha1.Runtime, error) {
	rt := &runtimev1alpha1.Runtime{}
	err := s.Client.Get(ctx, types.NamespacedName{Name: runtimeID, Namespace: s.runtimeNamespace()}, rt)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func (s *Server) fetchManagedSecret(ctx context.Context, rt *runtimev1alpha1.Runtime) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := s.Client.Get(ctx, types.NamespacedName{Name: runtimeutil.ManagedSecretName(rt.Spec.RuntimeID), Namespace: rt.Namespace}, secret)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (s *Server) ensureManagedSecretRef(ctx context.Context, rt *runtimev1alpha1.Runtime, secret *corev1.Secret) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		current := &runtimev1alpha1.Runtime{}
		if err := s.Client.Get(ctx, types.NamespacedName{Name: rt.Name, Namespace: rt.Namespace}, current); err != nil {
			return err
		}
		managedSecretName := runtimeutil.ManagedSecretName(current.Spec.RuntimeID)
		updatedRefs := withManagedSecretRefIfPresent(current.Spec.RuntimeID, current.Spec.EnvFromSecrets, secret)
		if runtimeutil.ContainsSecretRef(current.Spec.EnvFromSecrets, managedSecretName) && len(updatedRefs) == len(current.Spec.EnvFromSecrets) {
			return nil
		}
		current.Spec.EnvFromSecrets = updatedRefs
		return s.Client.Update(ctx, current)
	})
}

func (s *Server) removeManagedSecretRef(ctx context.Context, rt *runtimev1alpha1.Runtime) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		current := &runtimev1alpha1.Runtime{}
		if err := s.Client.Get(ctx, types.NamespacedName{Name: rt.Name, Namespace: rt.Namespace}, current); err != nil {
			return err
		}
		managedSecretName := runtimeutil.ManagedSecretName(current.Spec.RuntimeID)
		updatedRefs := runtimeutil.WithoutSecretRef(current.Spec.EnvFromSecrets, managedSecretName)
		if len(updatedRefs) == len(current.Spec.EnvFromSecrets) {
			return nil
		}
		current.Spec.EnvFromSecrets = updatedRefs
		return s.Client.Update(ctx, current)
	})
}

func (s *Server) runtimeNamespace() string {
	if strings.TrimSpace(s.Namespace) == "" {
		return "default"
	}
	return s.Namespace
}

func rawExtension(raw json.RawMessage) k8sruntime.RawExtension {
	if len(raw) == 0 {
		return k8sruntime.RawExtension{}
	}
	return k8sruntime.RawExtension{Raw: raw}
}

func validateExternalSecretRefs(runtimeID string, secretRefs []runtimev1alpha1.RuntimeSecretReference) error {
	reserved := runtimeutil.ManagedSecretName(runtimeID)
	for _, item := range secretRefs {
		if item.Name == reserved {
			return fmt.Errorf("envFromSecrets may not include reserved managed secret %q", reserved)
		}
	}
	return nil
}

func withManagedSecretRefIfPresent(runtimeID string, secretRefs []runtimev1alpha1.RuntimeSecretReference, secret *corev1.Secret) []runtimev1alpha1.RuntimeSecretReference {
	managedSecretName := runtimeutil.ManagedSecretName(runtimeID)
	refs := runtimeutil.WithoutSecretRef(secretRefs, managedSecretName)
	if secret == nil || len(secret.Data) == 0 {
		return refs
	}
	return runtimeutil.WithSecretRef(refs, managedSecretName)
}

func runtimeSecretsResponse(rt *runtimev1alpha1.Runtime, secret *corev1.Secret) RuntimeSecretsResponse {
	response := RuntimeSecretsResponse{
		RuntimeID:       rt.Spec.RuntimeID,
		SecretName:      runtimeutil.ManagedSecretName(rt.Spec.RuntimeID),
		Exists:          secret != nil,
		EnvFromAttached: runtimeutil.ContainsSecretRef(rt.Spec.EnvFromSecrets, runtimeutil.ManagedSecretName(rt.Spec.RuntimeID)),
	}
	if secret == nil {
		return response
	}
	keys := make([]string, 0, len(secret.Data))
	for key := range secret.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	response.Keys = keys
	return response
}

func totalSecretBytes(data map[string][]byte) int {
	total := 0
	for key, value := range data {
		total += len(key) + len(value)
	}
	return total
}

func runtimeResponse(rt *runtimev1alpha1.Runtime) RuntimeResponse {
	return RuntimeResponse{
		Name:               rt.Name,
		RuntimeID:          rt.Spec.RuntimeID,
		TenantID:           rt.Spec.TenantID,
		Phase:              rt.Status.Phase,
		URL:                rt.Status.URL,
		Namespace:          rt.Status.Namespace,
		DeploymentName:     rt.Status.DeploymentName,
		ServiceName:        rt.Status.ServiceName,
		IngressName:        rt.Status.IngressName,
		ReadyReplicas:      rt.Status.ReadyReplicas,
		DesiredReplicas:    rt.Status.DesiredReplicas,
		Message:            rt.Status.Message,
		ObservedGeneration: rt.Status.ObservedGeneration,
		CreatedAt:          rt.CreationTimestamp,
	}
}

func generateRuntimeNamespace(runtimeID string) string {
	return runtimeID
}

func generateRuntimeDeploymentName(runtimeID string) string {
	return runtimeID
}

func generateRuntimeServiceName(runtimeID string) string {
	return runtimeID
}

func generateRuntimeIngressName(runtimeID string) string {
	return fmt.Sprintf("%s-ingress", runtimeID)
}

func writeLookupError(w http.ResponseWriter, err error) {
	if apierrors.IsNotFound(err) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "runtime not found"})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
