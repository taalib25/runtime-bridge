package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// handleGetWorkspaceProviders GET /v1/workspaces/{id}/config/providers
func (b *Bridge) handleGetWorkspaceProviders(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validWorkspaceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format"))
		return
	}
	providers, err := b.GetWorkspaceProviders(r.Context(), workspaceID)
	if err != nil {
		if isWorkspaceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": providers})
}

// handleSetWorkspaceProvider POST /v1/workspaces/{id}/config/providers
func (b *Bridge) handleSetWorkspaceProvider(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validWorkspaceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format"))
		return
	}
	var req ProviderConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if req.Provider == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("provider is required"))
		return
	}
	if req.APIKey == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("apiKey is required"))
		return
	}
	if err := b.SetWorkspaceProvider(r.Context(), workspaceID, req); err != nil {
		if isWorkspaceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"provider": req.Provider, "status": "updated"})
}

// handleUpdateWorkspaceProvider PUT /v1/workspaces/{id}/config/providers/{name}
func (b *Bridge) handleUpdateWorkspaceProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceID := vars["id"]
	providerName := vars["name"]
	if !validWorkspaceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format"))
		return
	}
	var req ProviderConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}
	req.Provider = providerName
	if req.APIKey == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("apiKey is required"))
		return
	}
	if err := b.SetWorkspaceProvider(r.Context(), workspaceID, req); err != nil {
		if isWorkspaceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"provider": providerName, "status": "updated"})
}

// handleDeleteWorkspaceProvider DELETE /v1/workspaces/{id}/config/providers/{name}
func (b *Bridge) handleDeleteWorkspaceProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceID := vars["id"]
	providerName := vars["name"]
	if !validWorkspaceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format"))
		return
	}
	if err := b.DeleteWorkspaceProvider(r.Context(), workspaceID, providerName); err != nil {
		if isWorkspaceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"provider": providerName, "status": "removed"})
}

// handleSetWorkspaceModel PUT /v1/workspaces/{id}/config/model
func (b *Bridge) handleSetWorkspaceModel(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validWorkspaceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format"))
		return
	}
	var req SetModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if req.Provider == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("provider is required"))
		return
	}
	if req.Model == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("model is required"))
		return
	}
	if err := b.SetWorkspaceModel(r.Context(), workspaceID, req); err != nil {
		if isWorkspaceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"provider": req.Provider,
		"model":    req.Model,
		"status":   "updated",
	})
}
