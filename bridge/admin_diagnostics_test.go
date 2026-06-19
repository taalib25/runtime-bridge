package main

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func boolPtr(b bool) *bool { return &b }

func discardLogger() *log.Logger { return log.New(io.Discard, "", 0) }

func TestEndpointReadiness_PrefersEndpointSlice(t *testing.T) {
	client := fake.NewSimpleClientset(&discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-1-abcde",
			Namespace: "ws-1",
			Labels:    map[string]string{"kubernetes.io/service-name": "ws-1"},
		},
		Endpoints: []discoveryv1.Endpoint{
			{Addresses: []string{"10.42.0.1"}, Conditions: discoveryv1.EndpointConditions{Ready: boolPtr(true)}},
			{Addresses: []string{"10.42.0.2"}, Conditions: discoveryv1.EndpointConditions{Ready: boolPtr(false)}},
		},
	})
	b := &Bridge{KubeClient: client, Logger: discardLogger()}

	source, ready, notReady, err := b.endpointReadiness(context.Background(), "ws-1", "ws-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "endpointslice" {
		t.Errorf("expected source=endpointslice, got %q", source)
	}
	if ready != 1 || notReady != 1 {
		t.Errorf("expected ready=1 notReady=1, got ready=%d notReady=%d", ready, notReady)
	}
}

func TestEndpointReadiness_FallsBackToLegacyEndpoints(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: "ws-1", Namespace: "ws-1"},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses:         []corev1.EndpointAddress{{IP: "10.42.0.1"}},
				NotReadyAddresses: []corev1.EndpointAddress{{IP: "10.42.0.2"}},
			},
		},
	})
	b := &Bridge{KubeClient: client, Logger: discardLogger()}

	source, ready, notReady, err := b.endpointReadiness(context.Background(), "ws-1", "ws-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "endpoints" {
		t.Errorf("expected source=endpoints (legacy fallback), got %q", source)
	}
	if ready != 1 || notReady != 1 {
		t.Errorf("expected ready=1 notReady=1, got ready=%d notReady=%d", ready, notReady)
	}
}

// This is the regression test for the actual bug: a lookup failure (e.g. the bridge's
// ClusterRole missing the endpoints/endpointslices grant, see deploy/rbac.yaml) must
// come back as an error, never silently coerced into "0 endpoints, not ready".
func TestEndpointReadiness_LookupFailureIsAnErrorNotFalse(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("get", "endpoints", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("forbidden")
	})
	client.PrependReactor("list", "endpointslices", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("forbidden")
	})
	b := &Bridge{KubeClient: client, Logger: discardLogger()}

	_, ready, notReady, err := b.endpointReadiness(context.Background(), "ws-1", "ws-1")
	if err == nil {
		t.Fatal("expected an error when both lookups fail, got nil")
	}
	if ready != 0 || notReady != 0 {
		t.Errorf("expected zero counts alongside the error, got ready=%d notReady=%d", ready, notReady)
	}
}

func TestBuildProvisionTimeline_FullHappyPath(t *testing.T) {
	created := time.Date(2026, 6, 18, 18, 29, 7, 0, time.UTC)
	scheduled := created.Add(2 * time.Second)
	pullStart := created.Add(13 * time.Second)
	pulled := created.Add(14 * time.Second) // 1s pull, image was cached
	containerCreated := created.Add(34 * time.Second)
	containerStarted := created.Add(35 * time.Second)
	podReady := created.Add(50 * time.Second)
	serviceReady := created.Add(57 * time.Second)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.NewTime(created)},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "hermes-agent"}}},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodScheduled, Status: corev1.ConditionTrue, LastTransitionTime: metav1.NewTime(scheduled)},
				{Type: corev1.PodReady, Status: corev1.ConditionTrue, LastTransitionTime: metav1.NewTime(podReady)},
			},
		},
	}
	events := []InstanceEvent{
		{Reason: "Pulling", Message: "Pulling image \"nousresearch/hermes-agent:latest\"", FirstTime: pullStart, LastTime: pullStart},
		{Reason: "Pulled", Message: "Successfully pulled image", FirstTime: pulled, LastTime: pulled},
		{Reason: "Created", Message: "Created container: hermes-agent", FirstTime: containerCreated, LastTime: containerCreated},
		{Reason: "Started", Message: "Started container hermes-agent", FirstTime: containerStarted, LastTime: containerStarted},
		// A different container's events must not bleed into the main container's timing.
		{Reason: "Created", Message: "Created container: bootstrap-config", FirstTime: created.Add(20 * time.Second), LastTime: created.Add(20 * time.Second)},
	}

	tl := buildProvisionTimeline(pod, events, &serviceReady)

	check := func(name string, got *time.Time, want time.Time) {
		if got == nil {
			t.Errorf("%s: expected %v, got nil", name, want)
			return
		}
		if !got.Equal(want) {
			t.Errorf("%s: expected %v, got %v", name, want, *got)
		}
	}
	check("PodCreatedAt", tl.PodCreatedAt, created)
	check("PodScheduledAt", tl.PodScheduledAt, scheduled)
	check("ImagePullStartedAt", tl.ImagePullStartedAt, pullStart)
	check("ImagePulledAt", tl.ImagePulledAt, pulled)
	check("ContainerCreatedAt", tl.ContainerCreatedAt, containerCreated)
	check("ContainerStartedAt", tl.ContainerStartedAt, containerStarted)
	check("PodReadyAt", tl.PodReadyAt, podReady)
	check("FirstAPIStatusSuccessAt", tl.FirstAPIStatusSuccessAt, podReady)
	check("ServiceEndpointsReadyAt", tl.ServiceEndpointsReadyAt, serviceReady)

	checkSeconds := func(name string, got *float64, want float64) {
		if got == nil {
			t.Errorf("%s: expected %vs, got nil", name, want)
			return
		}
		if *got != want {
			t.Errorf("%s: expected %vs, got %vs", name, want, *got)
		}
	}
	checkSeconds("ScheduleSeconds", tl.ScheduleSeconds, 2)
	checkSeconds("ImagePullSeconds", tl.ImagePullSeconds, 1)
	checkSeconds("ContainerStartSeconds", tl.ContainerStartSeconds, 1)
	checkSeconds("AppBootSeconds", tl.AppBootSeconds, 15)
	checkSeconds("ServiceReadySeconds", tl.ServiceReadySeconds, 7)
	checkSeconds("TotalProvisionSeconds", tl.TotalProvisionSeconds, 50)
}

// A pod that's still pulling its image must report what HAS happened (created,
// scheduled, pull started) without fabricating timestamps for phases that haven't
// happened yet — nil, not zero/now, is the honest answer.
func TestBuildProvisionTimeline_StillPullingImage(t *testing.T) {
	created := time.Date(2026, 6, 18, 18, 29, 7, 0, time.UTC)
	scheduled := created.Add(2 * time.Second)
	pullStart := created.Add(3 * time.Second)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.NewTime(created)},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "hermes-agent"}}},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodScheduled, Status: corev1.ConditionTrue, LastTransitionTime: metav1.NewTime(scheduled)},
			},
		},
	}
	events := []InstanceEvent{
		{Reason: "Pulling", Message: "Pulling image \"nousresearch/hermes-agent:latest\"", FirstTime: pullStart, LastTime: pullStart},
	}

	tl := buildProvisionTimeline(pod, events, nil)

	if tl.ImagePullStartedAt == nil || !tl.ImagePullStartedAt.Equal(pullStart) {
		t.Errorf("expected ImagePullStartedAt=%v, got %v", pullStart, tl.ImagePullStartedAt)
	}
	if tl.ImagePulledAt != nil {
		t.Errorf("expected ImagePulledAt=nil (still pulling), got %v", *tl.ImagePulledAt)
	}
	if tl.PodReadyAt != nil {
		t.Errorf("expected PodReadyAt=nil, got %v", *tl.PodReadyAt)
	}
	if tl.TotalProvisionSeconds != nil {
		t.Errorf("expected TotalProvisionSeconds=nil (not ready yet), got %v", *tl.TotalProvisionSeconds)
	}
}
