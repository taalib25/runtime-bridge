package main

import (
	"os"
	"testing"

	"sigs.k8s.io/yaml"
)

// TestDumpBridgeEmittedValues is a Phase 5 helper: it writes the EXACT values
// map the bridge emits for a realistic create to BRIDGE_DUMP_PATH so we can
// `helm template` the fork against it. Skipped unless BRIDGE_DUMP_PATH is set.
func TestDumpBridgeEmittedValues(t *testing.T) {
	out := os.Getenv("BRIDGE_DUMP_PATH")
	if out == "" {
		t.Skip("set BRIDGE_DUMP_PATH to dump bridge-emitted values")
	}
	b := newTestBridge("s")

	spec := InstanceSpec{
		InstanceID:  "ws-aabbccddeeff0011",
		TenantID:    "t1",
		Image:       b.Config.RuntimeNodeCoreImage,
		ImageTag:    b.Config.RuntimeNodeCoreImageTag,
		CORSOrigins: "https://app.hermeshq.net,http://localhost:3002",
		Network:     NetworkSpec{Host: "wsp-aabbccddeeff0011.hermeshq.net"},
		CommonLabels: map[string]string{
			"hermescloud.dev/plan":     "pro",
			"hermescloud.dev/tenant":   "t1",
		},
		CommonAnnotations: map[string]string{
			"hermescloud.dev/display-name": "Ada",
		},
		Secrets: map[string]string{},
	}
	// Mirror handlers_instance.go create flow: secrets injected pre-buildValues.
	spec.Secrets["API_SERVER_KEY"] = randomHex(24)

	spec = b.normalizeInstanceSpec(spec)

	vals, err := b.buildValues(spec)
	if err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Marshal(vals)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote bridge-emitted values to %s (%d bytes)", out, len(data))
}
