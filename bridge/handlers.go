package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

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
		writeError(w, http.StatusBadRequest, err)
		return
	}

	op := b.submitOperation("create", workspaceID, func(ctx context.Context) error {
		_, err := b.CreateWorkspace(ctx, spec)
		return err
	})
	writeJSON(w, http.StatusAccepted, op)
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
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "cluster": b.ClusterName})
}

func (b *Bridge) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if err := b.CheckReadiness(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: "not ready", Details: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "cluster": b.ClusterName})
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
	if strings.TrimSpace(spec.TenantID) == "" {
		return WorkspaceSpec{}, fmt.Errorf("tenantId is required")
	}
	if strings.TrimSpace(spec.Image) == "" {
		return WorkspaceSpec{}, fmt.Errorf("image is required")
	}
	return b.normalizeWorkspaceSpec(spec), nil
}
