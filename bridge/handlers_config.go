package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// handleGetInstanceConfig GET /v1/instances/{id}/config
func (b *Bridge) handleGetInstanceConfig(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
		return
	}
	cfg, err := b.GetInstanceConfig(r.Context(), workspaceID)
	if err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// handleSetInstanceConfig PUT /v1/instances/{id}/config
// Accepts a partial HermesConfig — only non-nil sections are merged.
func (b *Bridge) handleSetInstanceConfig(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
		return
	}
	var incoming HermesConfig
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if err := b.SetInstanceConfig(r.Context(), workspaceID, incoming); err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "updated"})
}

// handleGetInstanceProviders GET /v1/instances/{id}/config/providers
func (b *Bridge) handleGetInstanceProviders(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
		return
	}
	providers, err := b.GetInstanceProviders(r.Context(), workspaceID)
	if err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": providers})
}

// handleSetInstanceProvider POST /v1/instances/{id}/config/providers
func (b *Bridge) handleSetInstanceProvider(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
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
	if err := b.SetInstanceProvider(r.Context(), workspaceID, req); err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"provider": req.Provider, "status": "updated"})
}

// handleUpdateInstanceProvider PUT /v1/instances/{id}/config/providers/{name}
func (b *Bridge) handleUpdateInstanceProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceID := vars["id"]
	providerName := vars["name"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
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
	if err := b.SetInstanceProvider(r.Context(), workspaceID, req); err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"provider": providerName, "status": "updated"})
}

// handleDeleteInstanceProvider DELETE /v1/instances/{id}/config/providers/{name}
func (b *Bridge) handleDeleteInstanceProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceID := vars["id"]
	providerName := vars["name"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
		return
	}
	if err := b.DeleteInstanceProvider(r.Context(), workspaceID, providerName); err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"provider": providerName, "status": "removed"})
}

// handleSetInstanceModel PUT /v1/instances/{id}/config/model
func (b *Bridge) handleSetInstanceModel(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
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
	if err := b.SetInstanceModel(r.Context(), workspaceID, req); err != nil {
		if isInstanceNotFound(err) {
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
