package main

import (
	"context"
	"fmt"

	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type permCheck struct {
	group    string
	resource string
	verb     string
}

// criticalPerms is the minimum set of permissions checked at startup.
// One representative verb per functional capability, derived from deploy/rbac.yaml.
// A denied check means the bridge will fail at that operation — surface it early.
var criticalPerms = []permCheck{
	{"", "namespaces", "create"},
	{"", "secrets", "create"},
	{"", "pods", "list"},
	{"", "services", "create"},
	{"apps", "deployments", "create"},
	{"networking.k8s.io", "networkpolicies", "create"},
	{"traefik.io", "ingressroutes", "create"},
	{"traefik.io", "middlewares", "create"},
}

// checkPermissions runs SelfSubjectAccessReview for each critical permission and
// returns the denied ones as "group/resource:verb" strings (empty group omitted).
func (b *Bridge) checkPermissions(ctx context.Context) []string {
	var missing []string
	for _, p := range criticalPerms {
		allowed, err := b.canI(ctx, p)
		if err != nil {
			b.Logger.Printf("[RBAC] SSAR error for %s: %v", p.key(), err)
		}
		if !allowed {
			missing = append(missing, p.key())
		}
	}
	return missing
}

func (p permCheck) key() string {
	if p.group == "" {
		return fmt.Sprintf("%s:%s", p.resource, p.verb)
	}
	return fmt.Sprintf("%s/%s:%s", p.group, p.resource, p.verb)
}

func (b *Bridge) canI(ctx context.Context, p permCheck) (bool, error) {
	sar := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Verb:     p.verb,
				Group:    p.group,
				Resource: p.resource,
			},
		},
	}
	result, err := b.KubeClient.AuthorizationV1().SelfSubjectAccessReviews().Create(
		ctx, sar, metav1.CreateOptions{},
	)
	if err != nil {
		return false, err
	}
	return result.Status.Allowed, nil
}
