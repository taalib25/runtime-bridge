package main

import (
	"context"
	"testing"

	"k8s.io/client-go/kubernetes/fake"
)

// helmTestBridge builds a Bridge wired to the real charts/hermes-agent chart on disk
// (loader.Load needs a real chart) but with a fake KubeClient and a FakeHelmRunner, so
// CreateInstance/UpdateInstance/DeleteInstance can run without a live cluster or Helm
// release storage.
func helmTestBridge(t *testing.T) (*Bridge, *FakeHelmRunner) {
	t.Helper()
	b := newTestBridge("s")
	b.ChartPath = "../charts/hermes-agent"
	b.KubeClient = fake.NewSimpleClientset()
	fakeHelm := &FakeHelmRunner{}
	b.Helm = fakeHelm
	return b, fakeHelm
}

func minimalSpec(b *Bridge, instanceID string) InstanceSpec {
	spec := InstanceSpec{
		InstanceID: instanceID,
		TenantID:   "t1",
		Image:      b.Config.RuntimeNodeCoreImage,
		ImageTag:   b.Config.RuntimeNodeCoreImageTag,
		Network:    NetworkSpec{Host: instanceID + ".hermeshq.net"},
		Secrets:    map[string]string{"API_SERVER_KEY": "test-key"},
	}
	// Mirrors decodeInstanceRequest: every real call site normalizes before
	// CreateInstance/UpdateInstance ever see the spec.
	return b.normalizeInstanceSpec(spec)
}

func TestCreateInstance_RoutesThroughHelmRunner(t *testing.T) {
	b, fakeHelm := helmTestBridge(t)
	spec := minimalSpec(b, "ws-create0000001")

	rel, err := b.CreateInstance(context.Background(), spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel == nil {
		t.Fatal("expected a release, got nil")
	}

	if len(fakeHelm.InstallCalls) != 1 {
		t.Fatalf("expected exactly 1 Install call, got %d", len(fakeHelm.InstallCalls))
	}
	call := fakeHelm.InstallCalls[0]
	if call.Opts.ReleaseName != b.releaseName(spec.InstanceID) {
		t.Errorf("ReleaseName = %q, want %q", call.Opts.ReleaseName, b.releaseName(spec.InstanceID))
	}
	if call.Opts.Namespace != spec.InstanceID {
		t.Errorf("Namespace = %q, want %q", call.Opts.Namespace, spec.InstanceID)
	}
	if !call.Opts.CreateNamespace {
		t.Error("expected CreateNamespace=true for a namespace-per-instance spec")
	}
	if !call.Opts.SkipCRDs {
		t.Error("expected SkipCRDs=true")
	}
	if call.Opts.Wait {
		t.Error("expected Wait=false")
	}
	if len(fakeHelm.UninstallCalls) != 0 {
		t.Errorf("expected no Uninstall calls on a clean create, got %d", len(fakeHelm.UninstallCalls))
	}
}

// On the Helm "cannot re-use a name that is still in use" error, CreateInstance
// cleans up the stale release via Uninstall and retries Install exactly once more —
// this verifies that wiring survives the move to HelmRunner. FakeHelmRunner has no
// per-call sequencing, so both Install attempts return the same error here; what's
// under test is the retry/cleanup shape, not Helm's own retry-then-succeed behavior.
func TestCreateInstance_NameReuseError_TriggersUninstallCleanupAndRetry(t *testing.T) {
	b, fakeHelm := helmTestBridge(t)
	spec := minimalSpec(b, "ws-create0000002")
	fakeHelm.InstallErr = errInstallNameReuse

	rel, err := b.CreateInstance(context.Background(), spec)
	if err == nil {
		t.Fatal("expected the (still-failing) retried Install to propagate an error")
	}
	if rel != nil {
		t.Errorf("expected nil release on error, got %+v", rel)
	}
	if len(fakeHelm.InstallCalls) != 2 {
		t.Fatalf("expected the name-reuse error to trigger exactly 1 retry (2 Install calls total), got %d", len(fakeHelm.InstallCalls))
	}
	if len(fakeHelm.UninstallCalls) != 1 {
		t.Fatalf("expected exactly 1 Uninstall cleanup call between the two Install attempts, got %d", len(fakeHelm.UninstallCalls))
	}
	if !fakeHelm.UninstallCalls[0].Opts.IgnoreNotFound {
		t.Error("expected cleanup Uninstall to set IgnoreNotFound=true")
	}
	if fakeHelm.UninstallCalls[0].Opts.KeepHistory {
		t.Error("expected cleanup Uninstall to set KeepHistory=false")
	}
}

func TestUpdateInstance_RoutesThroughHelmRunner(t *testing.T) {
	b, fakeHelm := helmTestBridge(t)
	spec := minimalSpec(b, "ws-update0000001")

	rel, err := b.UpdateInstance(context.Background(), spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel == nil {
		t.Fatal("expected a release, got nil")
	}
	if len(fakeHelm.UpgradeCalls) != 1 {
		t.Fatalf("expected exactly 1 Upgrade call, got %d", len(fakeHelm.UpgradeCalls))
	}
	call := fakeHelm.UpgradeCalls[0]
	if call.ReleaseName != b.releaseName(spec.InstanceID) {
		t.Errorf("ReleaseName = %q, want %q", call.ReleaseName, b.releaseName(spec.InstanceID))
	}
	if call.Opts.Namespace != spec.InstanceID {
		t.Errorf("Namespace = %q, want %q", call.Opts.Namespace, spec.InstanceID)
	}
	if !call.Opts.SkipCRDs {
		t.Error("expected SkipCRDs=true")
	}
	if call.Opts.Wait {
		t.Error("expected Wait=false")
	}
}

func TestUpdateInstance_HelmErrorPropagates(t *testing.T) {
	b, fakeHelm := helmTestBridge(t)
	spec := minimalSpec(b, "ws-update0000002")
	fakeHelm.UpgradeErr = errBoom

	_, err := b.UpdateInstance(context.Background(), spec)
	if err == nil {
		t.Fatal("expected error from Helm upgrade to propagate")
	}
}

func TestDeleteInstance_RoutesThroughHelmRunner_KeepHistory(t *testing.T) {
	b, fakeHelm := helmTestBridge(t)
	instanceID := "ws-delete0000001"

	if err := b.DeleteInstance(context.Background(), instanceID, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fakeHelm.UninstallCalls) != 1 {
		t.Fatalf("expected exactly 1 Uninstall call, got %d", len(fakeHelm.UninstallCalls))
	}
	call := fakeHelm.UninstallCalls[0]
	if call.ReleaseName != b.releaseName(instanceID) {
		t.Errorf("ReleaseName = %q, want %q", call.ReleaseName, b.releaseName(instanceID))
	}
	if !call.Opts.KeepHistory {
		t.Error("expected KeepHistory=true when purge=false")
	}
	if call.Opts.Wait {
		t.Error("expected Wait=false")
	}
}

func TestDeleteInstance_Purge_DropsHistory(t *testing.T) {
	b, fakeHelm := helmTestBridge(t)
	instanceID := "ws-delete0000002"

	if err := b.DeleteInstance(context.Background(), instanceID, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fakeHelm.UninstallCalls) != 1 {
		t.Fatalf("expected exactly 1 Uninstall call, got %d", len(fakeHelm.UninstallCalls))
	}
	if fakeHelm.UninstallCalls[0].Opts.KeepHistory {
		t.Error("expected KeepHistory=false when purge=true")
	}
}

func TestDeleteInstance_AlreadyUninstalled_IsNotAnError(t *testing.T) {
	b, fakeHelm := helmTestBridge(t)
	fakeHelm.UninstallErr = errReleaseNotFound

	if err := b.DeleteInstance(context.Background(), "ws-delete0000003", false); err != nil {
		t.Fatalf("expected 'release: not found' to be treated as already-deleted, got error: %v", err)
	}
}

// TODO: RollbackInstance and ListInstances are routed through HelmRunner (see
// lifecycle.go / helm.go) but have no dedicated unit tests yet — they need a fake
// release lookup (lookupRelease → b.Helm.List) wired up similarly to the above.

var errInstallNameReuse = fakeHelmError("cannot re-use a name that is still in use")
var errReleaseNotFound = fakeHelmError("release: not found")
var errBoom = fakeHelmError("boom")

type fakeHelmError string

func (e fakeHelmError) Error() string { return string(e) }
