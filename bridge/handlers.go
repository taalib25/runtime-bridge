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

var wsIDPattern = regexp.MustCompile(`^ws-[0-9a-f]{16}$`)

func validWorkspaceID(id string) bool { return wsIDPattern.MatchString(id) }

func (b *Bridge) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	workspaces, err := b.ListWorkspaces(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": workspaces})
}

func (b *Bridge) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	spec, err := b.decodeWorkspaceRequest(r, workspaceID)
	if err != nil {
		b.Logger.Printf("[CreateWorkspace] validation failed for %s: %v", workspaceID, err)
		writeError(w, http.StatusBadRequest, err)
		return
	}

	// Throttle: if a create is already in-flight for this workspace, return the
	// existing key immediately without launching another Helm install.
	if v, ok := b.pendingCreates.Load(workspaceID); ok {
		if rec := v.(pendingCreate); time.Now().Before(rec.until) {
			writeJSON(w, http.StatusAccepted, map[string]any{
				"workspaceId":  workspaceID,
				"status":       "provisioning",
				"url":          workspaceURL(spec),
				"dashboardUrl": dashboardURL(spec),
				"secrets":      map[string]string{"API_SERVER_KEY": rec.apiKey},
			})
			return
		}
	}

	if err := b.checkClusterCapacity(r.Context()); err != nil {
		b.Logger.Printf("[CreateWorkspace] Capacity check failed for %s: %v", workspaceID, err)
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

	b.pendingCreates.Store(workspaceID, pendingCreate{
		apiKey: apiKey,
		until:  time.Now().Add(b.Config.OperationTimeout),
	})

	b.submitOperation("create", workspaceID, func(ctx context.Context) error {
		_, err := b.CreateWorkspace(ctx, spec)
		return err
	})

	writeJSON(w, http.StatusAccepted, map[string]any{
		"workspaceId":  workspaceID,
		"status":       "provisioning",
		"url":          workspaceURL(spec),
		"dashboardUrl": dashboardURL(spec),
		"secrets":      map[string]string{"API_SERVER_KEY": apiKey},
	})
}

func (b *Bridge) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validWorkspaceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format: must match ws-[0-9a-f]{16}"))
		return
	}
	workspace, err := b.getWorkspace(r.Context(), workspaceID)
	if err != nil {
		if isWorkspaceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, workspace)
}

func (b *Bridge) handleUpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validWorkspaceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format: must match ws-[0-9a-f]{16}"))
		return
	}

	// Sync existence check before submitting the async operation. Without this,
	// a PUT on a non-existent workspace returns 202 and silently fails ~10min
	// later — QStash would mark the delivery as successful despite the failure.
	if _, err := b.lookupRelease(r.Context(), workspaceID); err != nil {
		if isWorkspaceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	spec, err := b.decodeWorkspaceRequest(r, workspaceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	// Only overwrite the user's config.yaml if the caller explicitly sent a config block.
	// A PUT for resources/plan only (no config field) must NOT wipe user's runtime edits.
	spec.OverwriteConfig = !isEmptyHermesConfig(spec.HermesConfig)

	op := b.submitOperation("update", workspaceID, func(ctx context.Context) error {
		_, err := b.UpdateWorkspace(ctx, spec)
		return err
	})
	writeJSON(w, http.StatusAccepted, op)
}

func (b *Bridge) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validWorkspaceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format: must match ws-[0-9a-f]{16}"))
		return
	}
	op := b.submitOperation("delete", workspaceID, func(ctx context.Context) error {
		return b.DeleteWorkspace(ctx, workspaceID)
	})
	writeJSON(w, http.StatusAccepted, op)
}

func (b *Bridge) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validWorkspaceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format: must match ws-[0-9a-f]{16}"))
		return
	}
	status, err := b.GetWorkspaceStatus(r.Context(), workspaceID)
	if err != nil {
		if isWorkspaceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (b *Bridge) handleHealth(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validWorkspaceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format: must match ws-[0-9a-f]{16}"))
		return
	}
	status, err := b.GetWorkspaceStatus(r.Context(), workspaceID)
	if err != nil {
		if isWorkspaceNotFound(err) {
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
		"workspaceId": workspaceID,
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
	b.mu.RLock()
	op, ok := b.operations[id]
	b.mu.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("operation %q not found", id))
		return
	}
	writeJSON(w, http.StatusOK, op)
}

func (b *Bridge) handleListWorkspaceOperations(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validWorkspaceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format: must match ws-[0-9a-f]{16}"))
		return
	}
	b.mu.RLock()
	var ops []*Operation
	for _, op := range b.operations {
		if op.WorkspaceID == workspaceID {
			ops = append(ops, op)
		}
	}
	b.mu.RUnlock()
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

func (b *Bridge) decodeWorkspaceRequest(r *http.Request, workspaceID string) (WorkspaceSpec, error) {
	var spec WorkspaceSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		return WorkspaceSpec{}, fmt.Errorf("decode request body: %w", err)
	}
	if spec.WorkspaceID != "" && spec.WorkspaceID != workspaceID {
		return WorkspaceSpec{}, fmt.Errorf("workspaceId in body must match path parameter")
	}
	spec.WorkspaceID = workspaceID
	if !validWorkspaceID(workspaceID) {
		return WorkspaceSpec{}, fmt.Errorf("workspaceId must match ws-[0-9a-f]{16}")
	}
	if strings.TrimSpace(spec.TenantID) == "" {
		return WorkspaceSpec{}, fmt.Errorf("tenantId is required")
	}
	if spec.Plan != "" {
		switch spec.Plan {
		case "free", "pro", "enterprise":
		default:
			return WorkspaceSpec{}, fmt.Errorf("plan must be one of: free, pro, enterprise")
		}
	}
	return b.normalizeWorkspaceSpec(spec), nil
}
