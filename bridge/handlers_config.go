package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	body, status, err := b.GetInstanceProviders(r.Context(), workspaceID)
	if err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(body) //nolint:errcheck
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

// handleGetInstanceProfiles GET /v1/instances/{id}/profiles
// Proxies to the hermes API server running inside the workspace pod and returns
// the verbatim response: {profiles: [...], active: "name"}.
func (b *Bridge) handleGetInstanceProfiles(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
		return
	}
	body, status, err := b.getInstanceProfiles(r.Context(), workspaceID)
	if err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(body) //nolint:errcheck
}

// proxyHermesAPI fetches API_SERVER_KEY from the workspace k8s Secret, then
// calls the given path on the hermes API server via the internal Kubernetes
// service URL (bypasses Traefik/ForwardAuth). Returns the verbatim body and status.
func (b *Bridge) proxyHermesAPI(ctx context.Context, workspaceID, path string) ([]byte, int, error) {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return nil, 0, err
	}
	ns := rel.Namespace

	secretName := b.workspaceSecretName(workspaceID)
	secret, err := b.KubeClient.CoreV1().Secrets(ns).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("get workspace secret: %w", err)
	}
	apiKey := string(secret.Data["API_SERVER_KEY"])
	if apiKey == "" {
		return nil, 0, fmt.Errorf("API_SERVER_KEY not set for instance %s", workspaceID)
	}

	svc := b.releaseName(workspaceID)
	url := fmt.Sprintf("http://%s.%s.svc.cluster.local:8787%s", svc, ns, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := b.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("call hermes API %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read response: %w", err)
	}
	return body, resp.StatusCode, nil
}

// getInstanceProfiles proxies /api/profiles from the hermes API server.
func (b *Bridge) getInstanceProfiles(ctx context.Context, workspaceID string) ([]byte, int, error) {
	return b.proxyHermesAPI(ctx, workspaceID, "/api/profiles")
}
