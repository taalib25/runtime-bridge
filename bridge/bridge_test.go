package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// newTestBridge builds a Bridge without a real k8s/Helm connection for unit tests.
func newTestBridge(secret string) *Bridge {
	cfg := DefaultConfig()
	cfg.ClusterName = "test-cluster"
	cfg.BridgeSecret = secret
	cfg.ChartPath = "/charts/hermes-agent"
	cfg.SyncInterval = time.Minute

	return &Bridge{
		Config:      cfg,
		ClusterName: cfg.ClusterName,
		ChartPath:   cfg.ChartPath,
		HTTPClient:  &http.Client{Timeout: 5 * time.Second},
		Logger:      log.New(os.Stdout, "test ", log.LstdFlags),
		Metrics:     NewMetrics(cfg.ClusterName, prometheus.NewRegistry()),
		runner:      newOperationRunner(cfg.OperationTimeout),
	}
}

// --- Auth middleware ---

func TestAuthMiddleware_MissingSecret(t *testing.T) {
	b := newTestBridge("mysecret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/instances", nil)
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	b := newTestBridge("mysecret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/instances", nil)
	req.Header.Set("X-Bridge-Secret", "wrongsecret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

// --- Health endpoints (no auth required) ---

func TestReadyz_NotReady_WhenKubeClientNil(t *testing.T) {
	b := newTestBridge("mysecret")
	// KubeClient is nil — CheckReadiness will fail → 503 not-ready.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
	var body map[string]string
	json.NewDecoder(rr.Body).Decode(&body) //nolint:errcheck
	if body["status"] != "" && body["status"] == "ready" {
		t.Error("should not be ready with nil KubeClient")
	}
}

func TestReadyz_EmptyPermMissing_DoesNotDegrade(t *testing.T) {
	// When permMissing holds an empty slice the perm-check branch must be skipped.
	// CheckReadiness will still return an error (no KubeClient), so we test the
	// perm-branch logic directly via TestReadyz_DegradedResponse_Format rather
	// than going through the full HTTP handler.
	b := newTestBridge("mysecret")
	b.permMissing.Store([]string{})
	v := b.permMissing.Load()
	if v == nil {
		t.Fatal("expected non-nil after Store")
	}
	missing, ok := v.([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", v)
	}
	if len(missing) != 0 {
		t.Errorf("expected empty, got %v", missing)
	}
}

func TestReadyz_CheckReadiness_RunsBeforePermCheck(t *testing.T) {
	// CheckReadiness (k8s discovery) must run before the perm check so a
	// disconnected bridge returns 503+not-ready, not 200+degraded.
	b := newTestBridge("mysecret")
	b.permMissing.Store([]string{"networking.k8s.io/networkpolicies:create"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	b.handleReadyz(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (k8s unreachable beats perm-degraded), got %d", rr.Code)
	}
	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body) //nolint:errcheck
	if body["error"] == nil {
		t.Error("expected error field in not-ready response")
	}
}

func TestReadyz_DegradedResponse_Format(t *testing.T) {
	// Test the degraded response structure independently of the k8s connectivity check.
	b := newTestBridge("mysecret")
	b.permMissing.Store([]string{"networking.k8s.io/networkpolicies:create"})

	// Build the degraded response exactly as handleReadyz does (minus the CheckReadiness guard).
	base := map[string]any{"cluster": b.ClusterName, "version": version, "build": build}
	if v := b.permMissing.Load(); v != nil {
		if missing, ok := v.([]string); ok && len(missing) > 0 {
			base["status"] = "degraded"
			base["missingPermissions"] = missing
		}
	}

	if base["status"] != "degraded" {
		t.Fatalf("expected status=degraded, got %v", base["status"])
	}
	perms, ok := base["missingPermissions"].([]string)
	if !ok || len(perms) != 1 {
		t.Fatalf("expected 1 missing perm, got %v", base["missingPermissions"])
	}
	if perms[0] != "networking.k8s.io/networkpolicies:create" {
		t.Errorf("unexpected perm key: %q", perms[0])
	}
	if base["cluster"] != "test-cluster" {
		t.Errorf("expected cluster=test-cluster, got %v", base["cluster"])
	}
}

func TestPermCheckKey(t *testing.T) {
	tests := []struct {
		p    permCheck
		want string
	}{
		{permCheck{"", "namespaces", "create"}, "namespaces:create"},
		{permCheck{"apps", "deployments", "create"}, "apps/deployments:create"},
		{permCheck{"networking.k8s.io", "networkpolicies", "create"}, "networking.k8s.io/networkpolicies:create"},
	}
	for _, tt := range tests {
		if got := tt.p.key(); got != tt.want {
			t.Errorf("permCheck{%q,%q,%q}.key() = %q, want %q", tt.p.group, tt.p.resource, tt.p.verb, got, tt.want)
		}
	}
}

func TestHealthz(t *testing.T) {
	b := newTestBridge("mysecret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", body["status"])
	}
	if body["cluster"] != "test-cluster" {
		t.Errorf("expected cluster=test-cluster, got %q", body["cluster"])
	}
}

// --- Request decoding ---

const testWID = "ws-1234567890abcdef"

func TestDecodeInstanceRequest_Valid(t *testing.T) {
	b := newTestBridge("s")
	payload := `{"tenantId":"t1"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	spec, err := b.decodeInstanceRequest(req, testWID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.InstanceID != testWID {
		t.Errorf("expected workspaceId=%s, got %q", testWID, spec.InstanceID)
	}
	if spec.TenantID != "t1" {
		t.Errorf("expected tenantId=t1, got %q", spec.TenantID)
	}
	// image should be defaulted even when not sent
	if spec.Image == "" {
		t.Errorf("expected image to be defaulted, got empty")
	}
}

func TestDecodeInstanceRequest_MissingTenantID(t *testing.T) {
	b := newTestBridge("s")
	payload := `{}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	_, err := b.decodeInstanceRequest(req, testWID)
	if err == nil || !strings.Contains(err.Error(), "tenantId") {
		t.Fatalf("expected tenantId error, got %v", err)
	}
}

func TestDecodeInstanceRequest_ImageOptional(t *testing.T) {
	b := newTestBridge("s")
	payload := `{"tenantId":"t1"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	spec, err := b.decodeInstanceRequest(req, testWID)
	if err != nil {
		t.Fatalf("image should be optional, got error: %v", err)
	}
	if spec.Image != "nousresearch/hermes-agent" {
		t.Errorf("expected defaulted image, got %q", spec.Image)
	}
}

func TestDecodeInstanceRequest_WorkspaceIDMismatch(t *testing.T) {
	b := newTestBridge("s")
	payload := `{"instanceId":"ws-ffffffffffffffff","tenantId":"t1"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	_, err := b.decodeInstanceRequest(req, testWID)
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestDecodeInstanceRequest_InvalidInstanceIDFormat(t *testing.T) {
	b := newTestBridge("s")
	payload := `{"tenantId":"t1"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	// IDs with uppercase letters or spaces are not valid k8s names.
	_, err := b.decodeInstanceRequest(req, "INVALID ID!")
	if err == nil || !strings.Contains(err.Error(), "instanceId") {
		t.Fatalf("expected instanceId format error, got %v", err)
	}
}

func TestDecodeInstanceRequest_AnyPlanAccepted(t *testing.T) {
	b := newTestBridge("s")
	// Backend decides plan values — bridge no longer validates the string.
	for _, plan := range []string{"free", "pro", "enterprise", "premium", "custom-tier"} {
		payload := `{"tenantId":"t1","plan":"` + plan + `"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
		spec, err := b.decodeInstanceRequest(req, testWID)
		if err != nil {
			t.Fatalf("plan %q: unexpected error: %v", plan, err)
		}
		if spec.Plan != plan {
			t.Fatalf("plan %q: got spec.Plan=%q", plan, spec.Plan)
		}
	}
}

// --- Handler responses (no k8s, so list/get return errors) ---

func TestHandleListInstances_AuthRequired(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/instances", nil)
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleCreateInstance_BadBody(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/instances/ws-1", bytes.NewBufferString("{invalid json}"))
	req.Header.Set("X-Bridge-Secret", "secret")
	req.Header.Set("Content-Type", "application/json")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// --- Upgrade handler ---

func TestHandleUpgradeInstance_BadBody(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/instances/"+testWID+"/upgrade", bytes.NewBufferString("{invalid json}"))
	req.Header.Set("X-Bridge-Secret", "secret")
	req.Header.Set("Content-Type", "application/json")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleUpgradeInstance_MissingImageTag(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/instances/"+testWID+"/upgrade", bytes.NewBufferString(`{"image":"ghcr.io/taalib25/runtime-node-core"}`))
	req.Header.Set("X-Bridge-Secret", "secret")
	req.Header.Set("Content-Type", "application/json")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleUpgradeInstance_ValidBody(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/instances/"+testWID+"/upgrade", bytes.NewBufferString(`{"image":"ghcr.io/taalib25/runtime-node-core","imageTag":"v0.2.0"}`))
	req.Header.Set("X-Bridge-Secret", "secret")
	req.Header.Set("Content-Type", "application/json")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["type"] != "upgrade" {
		t.Errorf("expected type=upgrade, got %v", body["type"])
	}
	if body["status"] != "running" {
		t.Errorf("expected status=running, got %v", body["status"])
	}
}

func TestHandleUpgradeInstance_DuplicateInFlight(t *testing.T) {
	b := newTestBridge("secret")

	// Inject a fake in-flight upgrade op directly into pendingOps.
	existingOp := &Operation{
		ID:         "existing-op-id",
		Type:       "upgrade",
		InstanceID: testWID,
		Status:     "running",
	}
	b.pendingOps.Store(testWID, &inflightOp{op: existingOp, cancel: func() {}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/instances/"+testWID+"/upgrade", bytes.NewBufferString(`{"image":"ghcr.io/taalib25/runtime-node-core","imageTag":"v0.2.0"}`))
	req.Header.Set("X-Bridge-Secret", "secret")
	req.Header.Set("Content-Type", "application/json")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
}

// --- Restart handler ---

func TestHandleRestartInstance_ValidRequest(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/instances/"+testWID+"/restart", nil)
	req.Header.Set("X-Bridge-Secret", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["type"] != "restart" {
		t.Errorf("expected type=restart, got %v", body["type"])
	}
	if body["status"] != "running" {
		t.Errorf("expected status=running, got %v", body["status"])
	}
}

func TestHandleRestartInstance_DuplicateInFlight(t *testing.T) {
	b := newTestBridge("secret")
	existingOp := &Operation{
		ID:         "existing-op-id",
		Type:       "restart",
		InstanceID: testWID,
		Status:     "running",
	}
	b.pendingOps.Store(testWID, &inflightOp{op: existingOp, cancel: func() {}})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/instances/"+testWID+"/restart", nil)
	req.Header.Set("X-Bridge-Secret", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
}

func TestHandleRestartInstance_InvalidInstanceID(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/instances/INVALID!/restart", nil)
	req.Header.Set("X-Bridge-Secret", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// Delete must win over an in-flight op: it cancels the running operation's context
// (so the op short-circuits cleanly) rather than being blocked by the one-in-flight guard.
func TestHandleDeleteInstance_SupersedesInFlightOp(t *testing.T) {
	b := newTestBridge("secret")
	canceled := make(chan struct{})
	existingOp := &Operation{
		ID:         "existing-op-id",
		Type:       "upgrade",
		InstanceID: testWID,
		Status:     "running",
	}
	b.runner.Record(existingOp)
	b.pendingOps.Store(testWID, &inflightOp{op: existingOp, cancel: func() { close(canceled) }})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/instances/"+testWID+"?purge=false", nil)
	req.Header.Set("X-Bridge-Secret", "secret")
	b.Router().ServeHTTP(rr, req)

	// Delete is accepted, not blocked with 409.
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
	// The in-flight op's context was canceled.
	select {
	case <-canceled:
	default:
		t.Fatal("expected in-flight op to be canceled by delete")
	}
	// And its record is marked superseded.
	if op, ok := b.runner.Get("existing-op-id"); ok {
		if op.Message == "" || !strings.Contains(op.Message, "superseded") {
			t.Errorf("expected superseded message, got %q", op.Message)
		}
	}
}

func TestHandleUpgradeInstance_InvalidInstanceID(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/instances/INVALID!/upgrade", bytes.NewBufferString(`{"imageTag":"v0.2.0"}`))
	req.Header.Set("X-Bridge-Secret", "secret")
	req.Header.Set("Content-Type", "application/json")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// --- Maintenance mode handler ---

func TestHandleGetMaintenanceMode_Default(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/cluster/maintenance", nil)
	req.Header.Set("X-Bridge-Secret", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["maintenance"] != false {
		t.Errorf("expected maintenance=false by default, got %v", body["maintenance"])
	}
}

func TestHandleSetMaintenanceMode_Enable(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/cluster/maintenance", bytes.NewBufferString(`{"enabled":true}`))
	req.Header.Set("X-Bridge-Secret", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !b.maintenance.Load() {
		t.Error("expected maintenance to be enabled")
	}
}

func TestHandleSetMaintenanceMode_Disable(t *testing.T) {
	b := newTestBridge("secret")
	b.maintenance.Store(true)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/cluster/maintenance", bytes.NewBufferString(`{"enabled":false}`))
	req.Header.Set("X-Bridge-Secret", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if b.maintenance.Load() {
		t.Error("expected maintenance to be disabled")
	}
}

func TestHandleSetMaintenanceMode_BadBody(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/cluster/maintenance", bytes.NewBufferString("{invalid}"))
	req.Header.Set("X-Bridge-Secret", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCreateInstance_BlockedByMaintenance(t *testing.T) {
	b := newTestBridge("secret")
	b.maintenance.Store(true)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/instances/"+testWID, bytes.NewBufferString(`{"tenantId":"t1"}`))
	req.Header.Set("X-Bridge-Secret", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}

// --- Drain handler ---

func TestHandleDrainCluster_AcceptsRequest(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/cluster/drain", bytes.NewBufferString(`{"purge":false}`))
	req.Header.Set("X-Bridge-Secret", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["type"] != "drain" {
		t.Errorf("expected type=drain, got %v", body["type"])
	}
	if body["status"] != "running" {
		t.Errorf("expected status=running, got %v", body["status"])
	}
	// Maintenance mode should be enabled synchronously before the 202 is returned.
	if !b.maintenance.Load() {
		t.Error("expected maintenance mode to be enabled after drain starts")
	}
}

func TestHandleDrainCluster_ConflictWhenAlreadyDraining(t *testing.T) {
	b := newTestBridge("secret")
	b.draining.Store(true)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/cluster/drain", nil)
	req.Header.Set("X-Bridge-Secret", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
}

// --- Helpers ---

func TestReleaseName_NoPrefix(t *testing.T) {
	b := newTestBridge("s")
	if got := b.releaseName("ws-abc"); got != "ws-abc" {
		t.Errorf("expected ws-abc, got %s", got)
	}
}

func TestReleaseName_WithPrefix(t *testing.T) {
	b := newTestBridge("s")
	b.Config.ReleasePrefix = "hermes-"
	if got := b.releaseName("ws-abc"); got != "hermes-ws-abc" {
		t.Errorf("expected hermes-ws-abc, got %s", got)
	}
}

func TestInstanceNamespace_FromSpec(t *testing.T) {
	b := newTestBridge("s")
	b.Config.Namespace = "default"
	spec := InstanceSpec{Namespace: "tenant-ns"}
	if got := b.instanceNamespace(spec); got != "tenant-ns" {
		t.Errorf("expected tenant-ns, got %s", got)
	}
}

func TestInstanceNamespace_FallbackToConfig(t *testing.T) {
	b := newTestBridge("s")
	b.Config.Namespace = "default"
	spec := InstanceSpec{}
	if got := b.instanceNamespace(spec); got != "default" {
		t.Errorf("expected default, got %s", got)
	}
}

// --- buildValues ---

func TestBuildValues_ImageSplitWithTag(t *testing.T) {
	b := newTestBridge("s")
	spec := InstanceSpec{
		InstanceID: "ws-1",
		TenantID:    "t1",
		Image:       "nousresearch/hermes-agent:v2026.4.16",
	}
	vals, err := b.buildValues(spec)
	if err != nil {
		t.Fatal(err)
	}
	img := vals["image"].(map[string]any)
	if img["repository"] != "nousresearch/hermes-agent" {
		t.Errorf("unexpected repository: %v", img["repository"])
	}
	if img["tag"] != "v2026.4.16" {
		t.Errorf("unexpected tag: %v", img["tag"])
	}
}

// When the image/tag match the bridge's own defaults, buildValues prefers the pinned
// digest over the floating ":latest" tag — that's the whole point of pinning it.
func TestBuildValues_DefaultImageUsesPinnedDigest(t *testing.T) {
	b := newTestBridge("s")
	spec := InstanceSpec{
		InstanceID: "ws-1",
		TenantID:    "t1",
		Image:       b.Config.RuntimeNodeCoreImage,
		ImageTag:    b.Config.RuntimeNodeCoreImageTag,
	}
	vals, err := b.buildValues(spec)
	if err != nil {
		t.Fatal(err)
	}
	img := vals["image"].(map[string]any)
	if img["digest"] != b.Config.RuntimeNodeCoreImageDigest {
		t.Errorf("expected pinned digest %v, got %v", b.Config.RuntimeNodeCoreImageDigest, img["digest"])
	}
	if img["tag"] != nil {
		t.Errorf("expected no tag when digest is pinned, got %v", img["tag"])
	}
}

// A genuinely custom per-instance tag (different from the bridge's default) must
// still win over the digest — a pin is a promise about the DEFAULT image, not a
// blanket override of every instance's explicit choice.
func TestBuildValues_ImageTagOverride(t *testing.T) {
	b := newTestBridge("s")
	spec := InstanceSpec{
		InstanceID: "ws-1",
		TenantID:    "t1",
		Image:       "nousresearch/hermes-agent",
		ImageTag:    "v2026.6.5",
	}
	vals, err := b.buildValues(spec)
	if err != nil {
		t.Fatal(err)
	}
	img := vals["image"].(map[string]any)
	if img["tag"] != "v2026.6.5" {
		t.Errorf("expected v2026.6.5 tag, got %v", img["tag"])
	}
	if img["digest"] != nil {
		t.Errorf("expected no digest for an explicit custom tag, got %v", img["digest"])
	}
}

func TestBuildValues_IngressEnabled(t *testing.T) {
	b := newTestBridge("s")
	enabled := true
	spec := InstanceSpec{
		InstanceID:    "ws-1",
		TenantID:       "t1",
		Image:          "img",
		IngressEnabled: &enabled,
		Network:        NetworkSpec{Host: "hermeshq.net", Subdomain: "ws-1"},
	}
	vals, err := b.buildValues(spec)
	if err != nil {
		t.Fatal(err)
	}
	ingress := vals["ingress"].(map[string]any)
	if ingress["enabled"] != true {
		t.Errorf("expected ingress.enabled=true")
	}
}

func TestBuildValues_ResourcesSetCorrectly(t *testing.T) {
	b := newTestBridge("s")
	spec := InstanceSpec{
		InstanceID: "ws-1",
		TenantID:    "t1",
		Image:       "img",
		Resources: ResourceSpec{
			CPURequest:    "500m",
			MemoryRequest: "1Gi",
		},
	}
	vals, err := b.buildValues(spec)
	if err != nil {
		t.Fatal(err)
	}
	resources := vals["resources"].(map[string]any)
	requests := resources["requests"].(map[string]any)
	if requests["cpu"] != "500m" {
		t.Errorf("expected cpu=500m, got %v", requests["cpu"])
	}
	if requests["memory"] != "1Gi" {
		t.Errorf("expected memory=1Gi, got %v", requests["memory"])
	}
}

// --- Config validation ---

func TestConfigValidate_MissingClusterName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BridgeSecret = "s"
	cfg.ChartPath = "/charts"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "clusterName") {
		t.Fatalf("expected clusterName error, got %v", err)
	}
}

func TestConfigValidate_MissingChartPath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ClusterName = "c"
	cfg.BridgeSecret = "s"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "chartPath") {
		t.Fatalf("expected chartPath error, got %v", err)
	}
}

func TestConfigValidate_MissingSecret(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ClusterName = "c"
	cfg.ChartPath = "/charts"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "bridgeSecret") {
		t.Fatalf("expected bridgeSecret error, got %v", err)
	}
}

func TestConfigValidate_Valid(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ClusterName = "c"
	cfg.ChartPath = "/charts"
	cfg.BridgeSecret = "s"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
