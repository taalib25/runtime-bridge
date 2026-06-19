package main

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeriveDetailedPhase_TerminalPhasesPassThrough(t *testing.T) {
	for _, coarse := range []string{"failed", "error", "deleted", "deleting"} {
		got := deriveDetailedPhase(coarse, nil, nil)
		if got != coarse {
			t.Errorf("coarse=%q: expected pass-through, got %q", coarse, got)
		}
	}
	if got := deriveDetailedPhase("ready", nil, nil); got != "runtime_healthy" {
		t.Errorf("coarse=ready: expected runtime_healthy, got %q", got)
	}
}

func TestDeriveDetailedPhase_ProgressesThroughMilestones(t *testing.T) {
	base := time.Date(2026, 6, 18, 18, 29, 7, 0, time.UTC)
	mkPod := func(scheduled, ready bool) *corev1.Pod {
		conditions := []corev1.PodCondition{}
		if scheduled {
			conditions = append(conditions, corev1.PodCondition{Type: corev1.PodScheduled, Status: corev1.ConditionTrue, LastTransitionTime: metav1.NewTime(base.Add(time.Second))})
		}
		if ready {
			conditions = append(conditions, corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue, LastTransitionTime: metav1.NewTime(base.Add(30 * time.Second))})
		}
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.NewTime(base)},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "hermes-agent"}}},
			Status:     corev1.PodStatus{Conditions: conditions},
		}
	}

	cases := []struct {
		name    string
		pod     *corev1.Pod
		events  []InstanceEvent
		want    string
	}{
		{"nothing yet", mkPod(false, false), nil, "pod_pending"},
		{"scheduled only", mkPod(true, false), nil, "pod_scheduled"},
		{
			"pulling image",
			mkPod(true, false),
			[]InstanceEvent{{Reason: "Pulling", Message: "Pulling image", FirstTime: base.Add(2 * time.Second), LastTime: base.Add(2 * time.Second)}},
			"pulling_image",
		},
		{
			"image pulled",
			mkPod(true, false),
			[]InstanceEvent{
				{Reason: "Pulling", FirstTime: base.Add(2 * time.Second), LastTime: base.Add(2 * time.Second)},
				{Reason: "Pulled", FirstTime: base.Add(3 * time.Second), LastTime: base.Add(3 * time.Second)},
			},
			"image_pulled",
		},
		{
			"container creating",
			mkPod(true, false),
			[]InstanceEvent{
				{Reason: "Pulled", FirstTime: base.Add(3 * time.Second), LastTime: base.Add(3 * time.Second)},
				{Reason: "Created", Message: "Created container: hermes-agent", FirstTime: base.Add(4 * time.Second), LastTime: base.Add(4 * time.Second)},
			},
			"container_creating",
		},
		{
			"container started",
			mkPod(true, false),
			[]InstanceEvent{
				{Reason: "Created", Message: "Created container: hermes-agent", FirstTime: base.Add(4 * time.Second), LastTime: base.Add(4 * time.Second)},
				{Reason: "Started", Message: "Started container hermes-agent", FirstTime: base.Add(5 * time.Second), LastTime: base.Add(5 * time.Second)},
			},
			"container_started",
		},
		{"app starting (pod ready)", mkPod(true, true), nil, "app_starting"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveDetailedPhase("starting", tc.pod, tc.events)
			if got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestLatestEvent_PicksMostRecentLastTime(t *testing.T) {
	base := time.Date(2026, 6, 18, 18, 29, 7, 0, time.UTC)
	events := []InstanceEvent{
		{Reason: "Scheduled", LastTime: base},
		{Reason: "Started", LastTime: base.Add(5 * time.Second)},
		{Reason: "Pulling", LastTime: base.Add(2 * time.Second)},
	}
	got := latestEvent(events)
	if got == nil || got.Reason != "Started" {
		t.Errorf("expected Started (most recent), got %v", got)
	}
	if latestEvent(nil) != nil {
		t.Error("expected nil for empty event list")
	}
}
