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
		operations:  make(map[string]*Operation),
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
	if spec.Image != "ghcr.io/taalib25/runtime-node-core" {
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
	_, err := b.decodeInstanceRequest(req, "ws-1")
	if err == nil || !strings.Contains(err.Error(), "instanceId") {
		t.Fatalf("expected workspaceId format error, got %v", err)
	}
}

func TestDecodeInstanceRequest_InvalidPlan(t *testing.T) {
	b := newTestBridge("s")
	payload := `{"tenantId":"t1","plan":"premium"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	_, err := b.decodeInstanceRequest(req, testWID)
	if err == nil || !strings.Contains(err.Error(), "plan") {
		t.Fatalf("expected plan error, got %v", err)
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
	if got := b.workspaceNamespace(spec); got != "tenant-ns" {
		t.Errorf("expected tenant-ns, got %s", got)
	}
}

func TestInstanceNamespace_FallbackToConfig(t *testing.T) {
	b := newTestBridge("s")
	b.Config.Namespace = "default"
	spec := InstanceSpec{}
	if got := b.workspaceNamespace(spec); got != "default" {
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

func TestBuildValues_ImageTagOverride(t *testing.T) {
	b := newTestBridge("s")
	spec := InstanceSpec{
		InstanceID: "ws-1",
		TenantID:    "t1",
		Image:       "nousresearch/hermes-agent",
		ImageTag:    "latest",
	}
	vals, err := b.buildValues(spec)
	if err != nil {
		t.Fatal(err)
	}
	img := vals["image"].(map[string]any)
	if img["tag"] != "latest" {
		t.Errorf("expected latest tag, got %v", img["tag"])
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
