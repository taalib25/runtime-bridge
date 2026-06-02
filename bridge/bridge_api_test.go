package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── Label & annotation contract (docs/label-contract.md) ────────────────────

func TestValidateBackendMetadata_AcceptsHermescloudPrefix(t *testing.T) {
	spec := InstanceSpec{
		CommonLabels: map[string]string{
			"hermescloud.dev/plan":        "pro",
			"hermescloud.dev/workspace-id": "wsp_987",
		},
		CommonAnnotations: map[string]string{
			"hermescloud.dev/display-name": "Ada's Workspace (long, spaces, & punctuation ok)",
		},
	}
	if err := validateBackendMetadata(spec); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidateBackendMetadata_EmptyIsValid(t *testing.T) {
	if err := validateBackendMetadata(InstanceSpec{}); err != nil {
		t.Fatalf("empty maps must be valid, got %v", err)
	}
}

func TestValidateBackendMetadata_RejectsReservedPrefixes(t *testing.T) {
	for _, key := range []string{
		"app.kubernetes.io/instance",
		"app.kubernetes.io/name",
		"helm.sh/chart",
		"meta.helm.sh/release-name",
		"kubernetes.io/foo",
		"k8s.io/foo",
		"hermeshq/managed-by",
	} {
		spec := InstanceSpec{CommonLabels: map[string]string{key: "x"}}
		if err := validateBackendMetadata(spec); err == nil {
			t.Errorf("expected reject for reserved key %q", key)
		}
	}
}

func TestValidateBackendMetadata_RejectsNonHermescloudKey(t *testing.T) {
	spec := InstanceSpec{CommonLabels: map[string]string{"example.com/foo": "x"}}
	if err := validateBackendMetadata(spec); err == nil {
		t.Fatal("expected reject for non-hermescloud.dev key")
	}
}

func TestValidateBackendMetadata_RejectsOversizeLabelValue(t *testing.T) {
	spec := InstanceSpec{CommonLabels: map[string]string{
		"hermescloud.dev/blob": strings.Repeat("a", 64), // >63 chars
	}}
	if err := validateBackendMetadata(spec); err == nil {
		t.Fatal("expected reject for label value >63 chars")
	}
}

func TestValidateBackendMetadata_AllowsOversizeAnnotationValue(t *testing.T) {
	spec := InstanceSpec{CommonAnnotations: map[string]string{
		"hermescloud.dev/notes": strings.Repeat("a", 500), // long is fine for annotations
	}}
	if err := validateBackendMetadata(spec); err != nil {
		t.Fatalf("annotations may hold long values, got %v", err)
	}
}

func TestBuildValues_CommonLabelsAndAnnotations(t *testing.T) {
	b := newTestBridge("s")
	spec := InstanceSpec{
		InstanceID:        "ws-aabbccddeeff0011",
		TenantID:          "t1",
		Image:             "img",
		CommonLabels:      map[string]string{"hermescloud.dev/plan": "pro"},
		CommonAnnotations: map[string]string{"hermescloud.dev/display-name": "Ada"},
	}
	vals, err := b.buildValues(spec)
	if err != nil {
		t.Fatal(err)
	}
	cl, ok := vals["commonLabels"].(map[string]string)
	if !ok || cl["hermescloud.dev/plan"] != "pro" {
		t.Errorf("commonLabels not plumbed into values: %v", vals["commonLabels"])
	}
	ca, ok := vals["commonAnnotations"].(map[string]string)
	if !ok || ca["hermescloud.dev/display-name"] != "Ada" {
		t.Errorf("commonAnnotations not plumbed into values: %v", vals["commonAnnotations"])
	}
	// The dead hermes.ai/plan podLabel must no longer be emitted.
	if _, exists := vals["podLabels"]; exists {
		t.Errorf("podLabels should no longer be set (hermes.ai/plan removed)")
	}
}

// Persistence round-trip: labels survive the values→release→spec cycle that
// repair/redeploy/upgrade rely on. Without this, internal ops silently strip them.
func TestInstanceSpecFromRelease_PreservesCommonMetadata(t *testing.T) {
	b := newTestBridge("s")
	spec := InstanceSpec{
		InstanceID:        "ws-aabbccddeeff0011",
		TenantID:          "t1",
		Image:             "img",
		CommonLabels:      map[string]string{"hermescloud.dev/plan": "pro"},
		CommonAnnotations: map[string]string{"hermescloud.dev/display-name": "Ada"},
	}
	vals, err := b.buildValues(spec)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate the round-trip through Helm storage: marshal to JSON and back, so
	// nested maps decode as map[string]any (exactly how a stored release deserializes).
	raw, _ := json.Marshal(vals)
	var stored map[string]any
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatal(err)
	}
	got, err := instanceSpecFromRelease("ws-aabbccddeeff0011", stored, "ws-aabbccddeeff0011", "test-cluster")
	if err != nil {
		t.Fatal(err)
	}
	if got.CommonLabels["hermescloud.dev/plan"] != "pro" {
		t.Errorf("commonLabels lost in round-trip: %v", got.CommonLabels)
	}
	if got.CommonAnnotations["hermescloud.dev/display-name"] != "Ada" {
		t.Errorf("commonAnnotations lost in round-trip: %v", got.CommonAnnotations)
	}
}

func TestDeriveSelectorLabels_IdentityOnly(t *testing.T) {
	sel := deriveSelectorLabels("ws-aabbccddeeff0011")
	if sel["app.kubernetes.io/name"] != chartName {
		t.Errorf("expected name=%s, got %q", chartName, sel["app.kubernetes.io/name"])
	}
	if sel["app.kubernetes.io/instance"] != "ws-aabbccddeeff0011" {
		t.Errorf("expected instance=releaseName, got %q", sel["app.kubernetes.io/instance"])
	}
	if len(sel) != 2 {
		t.Errorf("selectorLabels must be identity-only (2 keys), got %d: %v", len(sel), sel)
	}
}

func TestInstanceLabels_ImperativeResourceSet(t *testing.T) {
	b := newTestBridge("s")
	spec := InstanceSpec{
		InstanceID:   "ws-aabbccddeeff0011",
		CommonLabels: map[string]string{"hermescloud.dev/plan": "pro"},
	}
	l := b.instanceLabels(spec)
	if l["app.kubernetes.io/managed-by"] != "hermes-bridge" {
		t.Errorf("imperative resources must be managed-by=hermes-bridge, got %q", l["app.kubernetes.io/managed-by"])
	}
	if l["app.kubernetes.io/instance"] != "ws-aabbccddeeff0011" {
		t.Errorf("expected identity instance label, got %q", l["app.kubernetes.io/instance"])
	}
	if l["hermescloud.dev/plan"] != "pro" {
		t.Errorf("commonLabels must be merged, got %v", l)
	}
}

func TestHandleCreateInstance_RejectsBadLabelWith422(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	body := `{"tenantId":"t1","commonLabels":{"app.kubernetes.io/instance":"hijack"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/instances/ws-1234567890abcdef", bytes.NewBufferString(body))
	req.Header.Set("X-Bridge-Secret", "secret")
	req.Header.Set("Content-Type", "application/json")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for reserved-prefix label, got %d", rr.Code)
	}
}

// ─── Pure-function helpers ────────────────────────────────────────────────────

func TestBuildExtraSecretKeys_PassesAllKeys(t *testing.T) {
	input := map[string]string{
		"OPENAI_API_KEY":     "",
		"GOOGLE_API_KEY":     "",
		"TELEGRAM_BOT_TOKEN": "",
	}
	got := buildExtraSecretKeys(input)
	for k := range input {
		if got[k] != k {
			t.Errorf("expected extra[%q]=%q, got %q", k, k, got[k])
		}
	}
	if len(got) != len(input) {
		t.Errorf("expected %d keys, got %d", len(input), len(got))
	}
}

func TestBuildExtraSecretKeys_Empty(t *testing.T) {
	got := buildExtraSecretKeys(map[string]string{})
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestSecretKeysFromRelease_ExtractsKeys(t *testing.T) {
	config := map[string]any{
		"extraSecretKeys": map[string]any{
			"OPENAI_API_KEY":     "OPENAI_API_KEY",
			"TELEGRAM_BOT_TOKEN": "TELEGRAM_BOT_TOKEN",
		},
	}
	got := secretKeysFromRelease(config)
	if got["OPENAI_API_KEY"] != "OPENAI_API_KEY" {
		t.Errorf("expected OPENAI_API_KEY, got %q", got["OPENAI_API_KEY"])
	}
	if got["TELEGRAM_BOT_TOKEN"] != "TELEGRAM_BOT_TOKEN" {
		t.Errorf("expected TELEGRAM_BOT_TOKEN, got %q", got["TELEGRAM_BOT_TOKEN"])
	}
}

func TestSecretKeysFromRelease_MissingKey(t *testing.T) {
	got := secretKeysFromRelease(map[string]any{})
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestSecretKeysFromRelease_WrongType(t *testing.T) {
	config := map[string]any{"extraSecretKeys": "not-a-map"}
	got := secretKeysFromRelease(config)
	if len(got) != 0 {
		t.Errorf("expected empty result on wrong type, got %v", got)
	}
}

func TestEnvMapFromRelease_ExtractsEnv(t *testing.T) {
	config := map[string]any{
		"extraEnv": map[string]any{
			"WORKSPACE_ID":      "ws-abc",
			"WHATSAPP_ENABLED":  "true",
		},
	}
	got := envMapFromRelease(config)
	if got["WORKSPACE_ID"] != "ws-abc" {
		t.Errorf("expected ws-abc, got %q", got["WORKSPACE_ID"])
	}
	if got["WHATSAPP_ENABLED"] != "true" {
		t.Errorf("expected true, got %q", got["WHATSAPP_ENABLED"])
	}
}

func TestEnvMapFromRelease_Missing(t *testing.T) {
	got := envMapFromRelease(map[string]any{})
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestInstanceSecretName(t *testing.T) {
	b := newTestBridge("s")
	got := b.instanceSecretName("ws-1234567890abcdef")
	if !strings.HasSuffix(got, "-secrets") {
		t.Errorf("expected -secrets suffix, got %q", got)
	}
	if !strings.Contains(got, "ws-1234567890abcdef") {
		t.Errorf("expected instanceID in secret name, got %q", got)
	}
}

// ─── splitImageReference ──────────────────────────────────────────────────────

func TestSplitImageReference_WithTag(t *testing.T) {
	repo, tag := splitImageReference("ghcr.io/org/image:v1.2.3")
	if repo != "ghcr.io/org/image" {
		t.Errorf("unexpected repo: %q", repo)
	}
	if tag != "v1.2.3" {
		t.Errorf("unexpected tag: %q", tag)
	}
}

func TestSplitImageReference_NoTag(t *testing.T) {
	repo, tag := splitImageReference("ghcr.io/org/image")
	if repo != "ghcr.io/org/image" {
		t.Errorf("unexpected repo: %q", repo)
	}
	if tag != "" {
		t.Errorf("expected empty tag, got %q", tag)
	}
}

func TestSplitImageReference_Digest(t *testing.T) {
	img := "ghcr.io/org/image@sha256:abc123"
	repo, tag := splitImageReference(img)
	if repo != img {
		t.Errorf("expected full image as repo for digest, got %q", repo)
	}
	if tag != "" {
		t.Errorf("expected empty tag for digest, got %q", tag)
	}
}

func TestSplitImageReference_Empty(t *testing.T) {
	repo, tag := splitImageReference("")
	if repo != "" || tag != "" {
		t.Errorf("expected empty repo/tag, got %q/%q", repo, tag)
	}
}

// ─── buildValues — secrets structure ─────────────────────────────────────────

func TestBuildValues_SecretsExistingSecret(t *testing.T) {
	b := newTestBridge("s")
	spec := InstanceSpec{InstanceID: "ws-aabbccddeeff0011", TenantID: "t1", Image: "img"}
	vals, err := b.buildValues(spec)
	if err != nil {
		t.Fatal(err)
	}
	secrets, ok := vals["secrets"].(map[string]any)
	if !ok {
		t.Fatal("secrets is not a map")
	}
	existingSecret, _ := secrets["existingSecret"].(string)
	if !strings.Contains(existingSecret, "ws-aabbccddeeff0011") {
		t.Errorf("existingSecret should reference the instanceID, got %q", existingSecret)
	}
	if secrets["create"] != false {
		t.Errorf("secrets.create should be false")
	}
}

func TestBuildValues_ExtraSecretKeys(t *testing.T) {
	b := newTestBridge("s")
	spec := InstanceSpec{
		InstanceID: "ws-aabbccddeeff0011",
		TenantID:    "t1",
		Image:       "img",
		Secrets:     map[string]string{"GOOGLE_API_KEY": "", "DISCORD_BOT_TOKEN": ""},
	}
	vals, err := b.buildValues(spec)
	if err != nil {
		t.Fatal(err)
	}
	extra, ok := vals["extraSecretKeys"].(map[string]string)
	if !ok {
		t.Fatal("extraSecretKeys is not a map[string]string")
	}
	if extra["GOOGLE_API_KEY"] != "GOOGLE_API_KEY" {
		t.Errorf("expected GOOGLE_API_KEY in extraSecretKeys, got %v", extra)
	}
	if extra["DISCORD_BOT_TOKEN"] != "DISCORD_BOT_TOKEN" {
		t.Errorf("expected DISCORD_BOT_TOKEN in extraSecretKeys, got %v", extra)
	}
}

func TestBuildValues_ExtraEnv(t *testing.T) {
	b := newTestBridge("s")
	spec := InstanceSpec{
		InstanceID: "ws-aabbccddeeff0011",
		TenantID:    "t1",
		Image:       "img",
		EnvMap:      map[string]string{"MY_CUSTOM_VAR": "hello"},
	}
	vals, err := b.buildValues(spec)
	if err != nil {
		t.Fatal(err)
	}
	extraEnv, ok := vals["extraEnv"].(map[string]string)
	if !ok {
		t.Fatal("extraEnv is not a map[string]string")
	}
	if extraEnv["MY_CUSTOM_VAR"] != "hello" {
		t.Errorf("expected MY_CUSTOM_VAR in extraEnv, got %v", extraEnv)
	}
}

// ─── decodeIntegrationRequest ─────────────────────────────────────────────────

func integrationReq(t *testing.T, body string) *http.Request {
	t.Helper()
	return httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
}

func TestDecodeIntegration_Telegram_Valid(t *testing.T) {
	cfg := map[string]string{}
	err := decodeIntegrationRequest(integrationReq(t, `{"botToken":"tok123","allowedUsers":"u1,u2"}`), "telegram", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg["TELEGRAM_BOT_TOKEN"] != "tok123" {
		t.Errorf("expected token, got %q", cfg["TELEGRAM_BOT_TOKEN"])
	}
	if cfg["TELEGRAM_ALLOWED_USERS"] != "u1,u2" {
		t.Errorf("expected allowed users, got %q", cfg["TELEGRAM_ALLOWED_USERS"])
	}
}

func TestDecodeIntegration_Telegram_MissingToken(t *testing.T) {
	err := decodeIntegrationRequest(integrationReq(t, `{}`), "telegram", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "botToken") {
		t.Fatalf("expected botToken error, got %v", err)
	}
}

func TestDecodeIntegration_Telegram_WebhookRequiresSecret(t *testing.T) {
	err := decodeIntegrationRequest(
		integrationReq(t, `{"botToken":"tok","webhookUrl":"https://example.com"}`),
		"telegram", map[string]string{},
	)
	if err == nil || !strings.Contains(err.Error(), "webhookSecret") {
		t.Fatalf("expected webhookSecret error, got %v", err)
	}
}

func TestDecodeIntegration_Discord_Valid(t *testing.T) {
	cfg := map[string]string{}
	err := decodeIntegrationRequest(integrationReq(t, `{"botToken":"dbot","allowedUsers":"u1"}`), "discord", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg["DISCORD_BOT_TOKEN"] != "dbot" {
		t.Errorf("expected dbot, got %q", cfg["DISCORD_BOT_TOKEN"])
	}
}

func TestDecodeIntegration_Discord_MissingToken(t *testing.T) {
	err := decodeIntegrationRequest(integrationReq(t, `{}`), "discord", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "botToken") {
		t.Fatalf("expected botToken error, got %v", err)
	}
}

func TestDecodeIntegration_Slack_Valid(t *testing.T) {
	cfg := map[string]string{}
	err := decodeIntegrationRequest(integrationReq(t, `{"botToken":"xoxb","appToken":"xapp"}`), "slack", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg["SLACK_BOT_TOKEN"] != "xoxb" {
		t.Errorf("expected xoxb, got %q", cfg["SLACK_BOT_TOKEN"])
	}
	if cfg["SLACK_APP_TOKEN"] != "xapp" {
		t.Errorf("expected xapp, got %q", cfg["SLACK_APP_TOKEN"])
	}
}

func TestDecodeIntegration_Slack_MissingAppToken(t *testing.T) {
	err := decodeIntegrationRequest(integrationReq(t, `{"botToken":"xoxb"}`), "slack", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "appToken") {
		t.Fatalf("expected appToken error, got %v", err)
	}
}

func TestDecodeIntegration_WhatsApp(t *testing.T) {
	cfg := map[string]string{}
	err := decodeIntegrationRequest(integrationReq(t, `{"allowAllUsers":true}`), "whatsapp", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg["WHATSAPP_ENABLED"] != "true" {
		t.Errorf("expected WHATSAPP_ENABLED=true, got %q", cfg["WHATSAPP_ENABLED"])
	}
	if cfg["WHATSAPP_ALLOW_ALL_USERS"] != "true" {
		t.Errorf("expected WHATSAPP_ALLOW_ALL_USERS=true, got %q", cfg["WHATSAPP_ALLOW_ALL_USERS"])
	}
}

func TestDecodeIntegration_Signal_Valid(t *testing.T) {
	cfg := map[string]string{}
	err := decodeIntegrationRequest(integrationReq(t, `{"httpUrl":"http://signal:8080","account":"+1234"}`), "signal", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg["SIGNAL_HTTP_URL"] != "http://signal:8080" {
		t.Errorf("expected SIGNAL_HTTP_URL, got %q", cfg["SIGNAL_HTTP_URL"])
	}
}

func TestDecodeIntegration_Signal_MissingAccount(t *testing.T) {
	err := decodeIntegrationRequest(integrationReq(t, `{"httpUrl":"http://signal:8080"}`), "signal", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "account") {
		t.Fatalf("expected account error, got %v", err)
	}
}

func TestDecodeIntegration_DingTalk_Valid(t *testing.T) {
	cfg := map[string]string{}
	err := decodeIntegrationRequest(integrationReq(t, `{"clientId":"cid","clientSecret":"csec"}`), "dingtalk", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg["DINGTALK_CLIENT_ID"] != "cid" {
		t.Errorf("expected cid, got %q", cfg["DINGTALK_CLIENT_ID"])
	}
	if cfg["DINGTALK_CLIENT_SECRET"] != "csec" {
		t.Errorf("expected csec, got %q", cfg["DINGTALK_CLIENT_SECRET"])
	}
}

func TestDecodeIntegration_DingTalk_MissingFields(t *testing.T) {
	if err := decodeIntegrationRequest(integrationReq(t, `{"clientId":"cid"}`), "dingtalk", map[string]string{}); err == nil {
		t.Fatal("expected clientSecret error")
	}
	if err := decodeIntegrationRequest(integrationReq(t, `{}`), "dingtalk", map[string]string{}); err == nil {
		t.Fatal("expected clientId error")
	}
}

func TestDecodeIntegration_Feishu_Valid(t *testing.T) {
	cfg := map[string]string{}
	err := decodeIntegrationRequest(integrationReq(t, `{"appId":"aid","appSecret":"asec","encryptKey":"ek"}`), "feishu", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg["FEISHU_APP_ID"] != "aid" {
		t.Errorf("expected aid, got %q", cfg["FEISHU_APP_ID"])
	}
	if cfg["FEISHU_ENCRYPT_KEY"] != "ek" {
		t.Errorf("expected ek, got %q", cfg["FEISHU_ENCRYPT_KEY"])
	}
}

func TestDecodeIntegration_Feishu_OptionalEncryptKey(t *testing.T) {
	cfg := map[string]string{}
	err := decodeIntegrationRequest(integrationReq(t, `{"appId":"aid","appSecret":"asec"}`), "feishu", cfg)
	if err != nil {
		t.Fatalf("encryptKey should be optional, got error: %v", err)
	}
	if _, set := cfg["FEISHU_ENCRYPT_KEY"]; set {
		t.Error("FEISHU_ENCRYPT_KEY should not be set when not provided")
	}
}

func TestDecodeIntegration_WeCom_Valid(t *testing.T) {
	cfg := map[string]string{}
	err := decodeIntegrationRequest(integrationReq(t, `{"botId":"bid","secret":"wsec","websocketUrl":"wss://wecom"}`), "wecom", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg["WECOM_BOT_ID"] != "bid" {
		t.Errorf("expected bid, got %q", cfg["WECOM_BOT_ID"])
	}
	if cfg["WECOM_WEBSOCKET_URL"] != "wss://wecom" {
		t.Errorf("expected websocketUrl, got %q", cfg["WECOM_WEBSOCKET_URL"])
	}
}

func TestDecodeIntegration_WeCom_MissingSecret(t *testing.T) {
	err := decodeIntegrationRequest(integrationReq(t, `{"botId":"bid"}`), "wecom", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "secret") {
		t.Fatalf("expected secret error, got %v", err)
	}
}

func TestDecodeIntegration_BlueBubbles_Valid(t *testing.T) {
	cfg := map[string]string{}
	err := decodeIntegrationRequest(integrationReq(t, `{"serverUrl":"http://bb:1234","password":"pass"}`), "bluebubbles", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg["BLUEBUBBLES_SERVER_URL"] != "http://bb:1234" {
		t.Errorf("expected serverUrl, got %q", cfg["BLUEBUBBLES_SERVER_URL"])
	}
}

func TestDecodeIntegration_BlueBubbles_MissingPassword(t *testing.T) {
	err := decodeIntegrationRequest(integrationReq(t, `{"serverUrl":"http://bb:1234"}`), "bluebubbles", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "password") {
		t.Fatalf("expected password error, got %v", err)
	}
}

func TestDecodeIntegration_Unsupported(t *testing.T) {
	err := decodeIntegrationRequest(integrationReq(t, `{}`), "twitter", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported error, got %v", err)
	}
}

// ─── Provider handler validation (no k8s needed — fails before any k8s call) ──

func authedReq(method, path, body, secret string) *http.Request {
	var buf *strings.Reader
	if body != "" {
		buf = strings.NewReader(body)
	} else {
		buf = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("X-Bridge-Secret", secret)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestHandleSetInstanceProvider_BadBody(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := authedReq(http.MethodPost, "/v1/instances/ws-1234567890abcdef/config/providers", "{bad json}", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleSetInstanceProvider_MissingProvider(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := authedReq(http.MethodPost, "/v1/instances/ws-1234567890abcdef/config/providers", `{"apiKey":"sk-x"}`, "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetInstanceProvider_MissingAPIKey(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := authedReq(http.MethodPost, "/v1/instances/ws-1234567890abcdef/config/providers", `{"provider":"openai"}`, "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetInstanceModel_MissingFields(t *testing.T) {
	b := newTestBridge("secret")

	cases := []struct {
		body string
		want string
	}{
		{`{}`, "provider"},
		{`{"provider":"openai"}`, "model"},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		req := authedReq(http.MethodPut, "/v1/instances/ws-1234567890abcdef/config/model", tc.body, "secret")
		b.Router().ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("body=%s: expected 400, got %d: %s", tc.body, rr.Code, rr.Body.String())
		}
	}
}

// ─── Integration handler validation ──────────────────────────────────────────

func TestHandleEnableIntegration_BadBody(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := authedReq(http.MethodPost, "/v1/instances/ws-1234567890abcdef/integrations/telegram", "{bad}", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleEnableIntegration_UnsupportedPlatform(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := authedReq(http.MethodPost, "/v1/instances/ws-1234567890abcdef/integrations/twitter", `{}`, "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleEnableIntegration_MissingRequiredField(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	// Telegram with no botToken → should 400 before any k8s call
	req := authedReq(http.MethodPost, "/v1/instances/ws-1234567890abcdef/integrations/telegram", `{}`, "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ─── decodeIntegrationCfg unit tests ─────────────────────────────────────────

func TestDecodeIntegrationCfg_Signal_Valid(t *testing.T) {
	cfg, err := decodeIntegrationCfg("signal", json.RawMessage(`{"httpUrl":"http://signal:8080","account":"+1234567890"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg["SIGNAL_HTTP_URL"] != "http://signal:8080" {
		t.Errorf("expected SIGNAL_HTTP_URL, got %q", cfg["SIGNAL_HTTP_URL"])
	}
	if cfg["SIGNAL_ACCOUNT"] != "+1234567890" {
		t.Errorf("expected SIGNAL_ACCOUNT, got %q", cfg["SIGNAL_ACCOUNT"])
	}
	if _, hasToken := cfg["SIGNAL_BOT_TOKEN"]; hasToken {
		t.Error("SIGNAL_BOT_TOKEN must not be set — Signal has no bot token")
	}
}

func TestDecodeIntegrationCfg_Signal_MissingHTTPUrl(t *testing.T) {
	_, err := decodeIntegrationCfg("signal", json.RawMessage(`{"account":"+1"}`))
	if err == nil || !strings.Contains(err.Error(), "httpUrl") {
		t.Fatalf("expected httpUrl error, got %v", err)
	}
}

func TestDecodeIntegrationCfg_Signal_IgnoreStories(t *testing.T) {
	cfg, err := decodeIntegrationCfg("signal", json.RawMessage(`{"httpUrl":"http://s:8080","account":"+1","ignoreStories":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg["SIGNAL_IGNORE_STORIES"] != "true" {
		t.Errorf("expected SIGNAL_IGNORE_STORIES=true, got %q", cfg["SIGNAL_IGNORE_STORIES"])
	}
}

func TestDecodeIntegrationCfg_Telegram_HomeChannel(t *testing.T) {
	cfg, err := decodeIntegrationCfg("telegram", json.RawMessage(`{"botToken":"tok","homeChannel":"-1001234567890"}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg["TELEGRAM_HOME_CHANNEL"] != "-1001234567890" {
		t.Errorf("expected TELEGRAM_HOME_CHANNEL=-1001234567890, got %q", cfg["TELEGRAM_HOME_CHANNEL"])
	}
	if _, ok := cfg["TELEGRAM_HOME_CHANNEL_NAME"]; ok {
		t.Error("TELEGRAM_HOME_CHANNEL_NAME does not exist in the Hermes agent — should not be set")
	}
}

func TestDecodeIntegrationCfg_Discord_Fields(t *testing.T) {
	cfg, err := decodeIntegrationCfg("discord", json.RawMessage(`{"botToken":"tok","ignoredChannels":"spam","homeChannel":"C123"}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg["DISCORD_IGNORED_CHANNELS"] != "spam" {
		t.Errorf("expected DISCORD_IGNORED_CHANNELS=spam, got %q", cfg["DISCORD_IGNORED_CHANNELS"])
	}
	if cfg["DISCORD_HOME_CHANNEL"] != "C123" {
		t.Errorf("expected DISCORD_HOME_CHANNEL=C123, got %q", cfg["DISCORD_HOME_CHANNEL"])
	}
	if _, ok := cfg["DISCORD_HOME_CHANNEL_NAME"]; ok {
		t.Error("DISCORD_HOME_CHANNEL_NAME does not exist in the Hermes agent — should not be set")
	}
	if _, ok := cfg["DISCORD_ALLOWED_CHANNELS"]; ok {
		t.Error("DISCORD_ALLOWED_CHANNELS is not a valid env var — should not be set")
	}
}

func TestDecodeIntegrationCfg_UnknownPlatform(t *testing.T) {
	_, err := decodeIntegrationCfg("twitter", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for unknown platform")
	}
}

// ─── handleSetIntegrations handler tests ─────────────────────────────────────

func TestHandleSetIntegrations_BadBody(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := authedReq(http.MethodPut, "/v1/instances/ws-1234567890abcdef/integrations", "{bad}", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetIntegrations_UnknownPlatform(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := authedReq(http.MethodPut, "/v1/instances/ws-1234567890abcdef/integrations", `{"twitter":{}}`, "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetIntegrations_EmptyBody_Accepted(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := authedReq(http.MethodPut, "/v1/instances/ws-1234567890abcdef/integrations", `{}`, "secret")
	b.Router().ServeHTTP(rr, req)
	// Empty body is valid — it disables all. Returns 202 (async op).
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ─── applyGatewayAllowAll ────────────────────────────────────────────────────

func TestApplyGatewayAllowAll_NoUsers_SetsFlag(t *testing.T) {
	env := map[string]string{}
	applyGatewayAllowAll(env)
	if env["GATEWAY_ALLOW_ALL_USERS"] != "true" {
		t.Errorf("expected GATEWAY_ALLOW_ALL_USERS=true when no allowedUsers set, got %q", env["GATEWAY_ALLOW_ALL_USERS"])
	}
}

func TestApplyGatewayAllowAll_WithTelegramUsers_ClearsFlag(t *testing.T) {
	env := map[string]string{
		"GATEWAY_ALLOW_ALL_USERS": "true",
		"TELEGRAM_ALLOWED_USERS":  "123,456",
	}
	applyGatewayAllowAll(env)
	if _, ok := env["GATEWAY_ALLOW_ALL_USERS"]; ok {
		t.Error("expected GATEWAY_ALLOW_ALL_USERS cleared when allowedUsers present")
	}
}

func TestApplyGatewayAllowAll_WithDiscordUsers_ClearsFlag(t *testing.T) {
	env := map[string]string{"DISCORD_ALLOWED_USERS": "user1"}
	applyGatewayAllowAll(env)
	if _, ok := env["GATEWAY_ALLOW_ALL_USERS"]; ok {
		t.Error("expected GATEWAY_ALLOW_ALL_USERS cleared when discord allowedUsers present")
	}
}

// ─── Agent template handler validation ───────────────────────────────────────

func TestHandleCreateAgentTemplate_BadBody(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := authedReq(http.MethodPost, "/v1/agents", "{bad}", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCreateAgentTemplate_MissingName(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := authedReq(http.MethodPost, "/v1/agents", `{"description":"d"}`, "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleApplyAgentTemplate_MissingAgentID(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := authedReq(http.MethodPost, "/v1/instances/ws-1234567890abcdef/agent", `{}`, "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleApplyAgentTemplate_BadBody(t *testing.T) {
	b := newTestBridge("secret")
	rr := httptest.NewRecorder()
	req := authedReq(http.MethodPost, "/v1/instances/ws-1234567890abcdef/agent", "{bad}", "secret")
	b.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// ─── isAgentNotFound / isWorkspaceNotFound ────────────────────────────────────

func TestIsAgentNotFound(t *testing.T) {
	if !isAgentNotFound(bytes.ErrTooLarge) {
		// bytes.ErrTooLarge doesn't contain "not found" — make sure false
	}
	cases := []struct {
		msg  string
		want bool
	}{
		{"agent not found", true},
		{"template not found", true},
		{"connection refused", false},
		{"", false},
	}
	for _, tc := range cases {
		got := isAgentNotFound(errStr(tc.msg))
		if got != tc.want {
			t.Errorf("isAgentNotFound(%q) = %v, want %v", tc.msg, got, tc.want)
		}
	}
}

// errStr is a minimal error that wraps a string message.
type errStr string

func (e errStr) Error() string { return string(e) }
