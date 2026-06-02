package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// k8sNamePattern matches valid Kubernetes resource names (RFC 1123 DNS label).
var k8sNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?$`)

func validInstanceID(id string) bool { return k8sNamePattern.MatchString(id) }

const invalidInstanceIDMsg = "invalid instanceId: must be a valid DNS label (lowercase alphanumeric and hyphens, e.g. tenant-abc123)"

func (b *Bridge) handleListInstances(w http.ResponseWriter, r *http.Request) {
	instances, err := b.ListInstances(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": instances})
}

func (b *Bridge) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	if b.maintenance.Load() {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("cluster is in maintenance mode — new instances are not accepted"))
		return
	}
	instanceID := mux.Vars(r)["id"]
	spec, err := b.decodeInstanceRequest(r, instanceID)
	if err != nil {
		b.Logger.Printf("[CreateInstance] validation failed for %s: %v", instanceID, err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateBackendMetadata(spec); err != nil {
		b.Logger.Printf("[CreateInstance] label contract violation for %s: %v", instanceID, err)
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}

	// Throttle: if a create is already in-flight for this instance, return the
	// existing key immediately without launching another Helm install.
	if v, ok := b.pendingCreates.Load(instanceID); ok {
		if rec := v.(pendingCreate); time.Now().Before(rec.until) {
			writeJSON(w, http.StatusAccepted, map[string]any{
				"instanceId":  instanceID,
				"clusterId":   b.ClusterName,
				"status":       "provisioning",
				"url":          instanceURL(spec),
				"dashboardUrl": dashboardURL(spec),
				"secrets":      map[string]string{"API_SERVER_KEY": rec.apiKey},
			})
			return
		}
	}

	if err := b.checkClusterCapacity(r.Context()); err != nil {
		b.Logger.Printf("[CreateInstance] Capacity check failed for %s: %v", instanceID, err)
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}

	// Ensure API_SERVER_KEY is set before the async op so we can return it now.
	// Backend must store this and pass it back on every future update.
	if spec.Secrets == nil {
		spec.Secrets = map[string]string{}
	}
	if spec.Secrets["API_SERVER_KEY"] == "" {
		spec.Secrets["API_SERVER_KEY"] = randomHex(32)
	}
	apiKey := spec.Secrets["API_SERVER_KEY"]

	b.pendingCreates.Store(instanceID, pendingCreate{
		apiKey: apiKey,
		until:  time.Now().Add(b.Config.OperationTimeout),
	})

	b.submitOperation("create", instanceID, func(ctx context.Context) error {
		_, err := b.CreateInstance(ctx, spec)
		return err
	})

	writeJSON(w, http.StatusAccepted, map[string]any{
		"instanceId":  instanceID,
		"clusterId":   b.ClusterName,
		"status":       "provisioning",
		"url":          instanceURL(spec),
		"dashboardUrl": dashboardURL(spec),
		"secrets":      map[string]string{"API_SERVER_KEY": apiKey},
	})
}

func (b *Bridge) handleGetInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf(invalidInstanceIDMsg))
		return
	}
	instance, err := b.getInstance(r.Context(), instanceID)
	if err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, instance)
}

func (b *Bridge) handleUpdateInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf(invalidInstanceIDMsg))
		return
	}

	// Sync existence check before submitting the async operation. Without this,
	// a PUT on a non-existent instance returns 202 and silently fails ~10min
	// later — QStash would mark the delivery as successful despite the failure.
	if _, err := b.lookupRelease(r.Context(), instanceID); err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	spec, err := b.decodeInstanceRequest(r, instanceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateBackendMetadata(spec); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}

	// Only overwrite the user's config.yaml if the caller explicitly sent a config block.
	// A PUT for resources/plan only (no config field) must NOT wipe user's runtime edits.
	spec.OverwriteConfig = !isEmptyHermesConfig(spec.HermesConfig)

	op := b.submitOperation("update", instanceID, func(ctx context.Context) error {
		_, err := b.UpdateInstance(ctx, spec)
		return err
	})
	writeJSON(w, http.StatusAccepted, op)
}

func (b *Bridge) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf(invalidInstanceIDMsg))
		return
	}
	// Purge is the default — delete removes everything (namespace, PVC, all data).
	// Pass ?purge=false to do a soft delete (Helm uninstall only, keeps PVC).
	purge := r.URL.Query().Get("purge") != "false"
	// Delete always wins: cancel any in-flight restart/upgrade/repair so it short-
	// circuits cleanly instead of rolling back into a release we're about to remove.
	b.supersedeInFlight(instanceID)
	op := b.submitOperation("delete", instanceID, func(ctx context.Context) error {
		return b.DeleteInstance(ctx, instanceID, purge)
	})
	writeJSON(w, http.StatusAccepted, op)
}

func (b *Bridge) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf(invalidInstanceIDMsg))
		return
	}
	status, err := b.GetInstanceStatus(r.Context(), instanceID)
	if err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (b *Bridge) handleHealth(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf(invalidInstanceIDMsg))
		return
	}
	status, err := b.GetInstanceStatus(r.Context(), instanceID)
	if err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	code := http.StatusOK
	if !status.Healthy {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]any{
		"instanceId": instanceID,
		"healthy":     status.Healthy,
		"phase":       status.Phase,
		"url":         status.URL,
		"statusCode":  status.HealthStatusCode,
		"message":     status.Message,
	})
}

func (b *Bridge) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "cluster": b.ClusterName, "version": version, "build": build})
}

func (b *Bridge) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if err := b.CheckReadiness(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: "not ready", Details: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "cluster": b.ClusterName, "version": version, "build": build})
}

func (b *Bridge) handleGetOperation(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	op, ok := b.runner.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("operation %q not found", id))
		return
	}
	writeJSON(w, http.StatusOK, op)
}

func (b *Bridge) handleListInstanceOperations(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf(invalidInstanceIDMsg))
		return
	}
	ops := b.runner.ListForInstance(instanceID)
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].StartedAt.After(ops[j].StartedAt)
	})
	// Apply limit — default 20, max 100, override via ?limit=N.
	limit := 20
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if len(ops) > limit {
		ops = ops[:limit]
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": ops})
}

func (b *Bridge) handleRestartInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf(invalidInstanceIDMsg))
		return
	}
	op, existing := b.submitInstanceOperation("restart", instanceID, func(ctx context.Context) (string, error) {
		return "", b.RestartInstance(ctx, instanceID)
	})
	if existing != nil {
		writeError(w, http.StatusConflict, fmt.Errorf("operation already in progress for instance %s (op: %s)", instanceID, existing.ID))
		return
	}
	writeJSON(w, http.StatusAccepted, op)
}

func (b *Bridge) handleRedeployInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf(invalidInstanceIDMsg))
		return
	}
	op, existing := b.submitInstanceOperation("redeploy", instanceID, func(ctx context.Context) (string, error) {
		return "", b.RedeployInstance(ctx, instanceID)
	})
	if existing != nil {
		writeError(w, http.StatusConflict, fmt.Errorf("operation already in progress for instance %s (op: %s)", instanceID, existing.ID))
		return
	}
	writeJSON(w, http.StatusAccepted, op)
}

func (b *Bridge) handleRollbackInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf(invalidInstanceIDMsg))
		return
	}
	var req struct {
		Version int `json:"version"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck — empty body or missing version defaults to 0 (previous release)
	}
	op, existing := b.submitInstanceOperation("rollback", instanceID, func(ctx context.Context) (string, error) {
		return "", b.RollbackInstance(ctx, instanceID, req.Version)
	})
	if existing != nil {
		writeError(w, http.StatusConflict, fmt.Errorf("operation already in progress for instance %s (op: %s)", instanceID, existing.ID))
		return
	}
	writeJSON(w, http.StatusAccepted, op)
}

func (b *Bridge) handleRepairInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf(invalidInstanceIDMsg))
		return
	}
	op, existing := b.submitInstanceOperation("repair", instanceID, func(ctx context.Context) (string, error) {
		_, err := b.RepairInstance(ctx, instanceID)
		return "", err
	})
	if existing != nil {
		writeError(w, http.StatusConflict, fmt.Errorf("operation already in progress for instance %s (op: %s)", instanceID, existing.ID))
		return
	}
	writeJSON(w, http.StatusAccepted, op)
}

func (b *Bridge) handleUpgradeInstance(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf(invalidInstanceIDMsg))
		return
	}
	var req struct {
		Image    string `json:"image"`
		ImageTag string `json:"imageTag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode request body: %w", err))
		return
	}
	if strings.TrimSpace(req.ImageTag) == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("imageTag is required"))
		return
	}
	op, existing := b.submitInstanceOperation("upgrade", instanceID, func(ctx context.Context) (string, error) {
		return b.UpgradeInstance(ctx, instanceID, req.Image, req.ImageTag)
	})
	if existing != nil {
		writeError(w, http.StatusConflict, fmt.Errorf("operation already in progress for instance %s (op: %s)", instanceID, existing.ID))
		return
	}
	writeJSON(w, http.StatusAccepted, op)
}

func (b *Bridge) handleGetEvents(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf(invalidInstanceIDMsg))
		return
	}
	events, err := b.GetInstanceEvents(r.Context(), instanceID)
	if err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"instanceId": instanceID, "items": events})
}

func (b *Bridge) handleRecreateTerminal(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf(invalidInstanceIDMsg))
		return
	}
	session, err := b.RecreateTerminalSession(r.Context(), instanceID)
	if err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (b *Bridge) decodeInstanceRequest(r *http.Request, instanceID string) (InstanceSpec, error) {
	var spec InstanceSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		return InstanceSpec{}, fmt.Errorf("decode request body: %w", err)
	}
	if spec.InstanceID != "" && spec.InstanceID != instanceID {
		return InstanceSpec{}, fmt.Errorf("instanceId in body must match path parameter")
	}
	spec.InstanceID = instanceID
	if !validInstanceID(instanceID) {
		return InstanceSpec{}, fmt.Errorf(invalidInstanceIDMsg)
	}
	if strings.TrimSpace(spec.TenantID) == "" {
		return InstanceSpec{}, fmt.Errorf("tenantId is required")
	}
	if spec.RuntimeMode != "" && spec.RuntimeMode != "runtime-node-core" {
		return InstanceSpec{}, fmt.Errorf("unsupported runtimeMode %q: only \"runtime-node-core\" is accepted", spec.RuntimeMode)
	}
	return b.normalizeInstanceSpec(spec), nil
}
