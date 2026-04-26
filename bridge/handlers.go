package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
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
				"workspaceId": workspaceID,
				"status":      "provisioning",
				"secrets":     map[string]string{"API_SERVER_KEY": rec.apiKey},
			})
			return
		}
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
		"workspaceId": workspaceID,
		"status":      "provisioning",
		"secrets":     map[string]string{"API_SERVER_KEY": apiKey},
	})
}

func (b *Bridge) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
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
	op := b.submitOperation("delete", workspaceID, func(ctx context.Context) error {
		return b.DeleteWorkspace(ctx, workspaceID)
	})
	writeJSON(w, http.StatusAccepted, op)
}

func (b *Bridge) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
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
