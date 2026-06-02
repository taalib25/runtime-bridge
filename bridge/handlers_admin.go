package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// adminContextTimeout wraps ctx with a 30s timeout for admin read operations.
func adminContextTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 30*time.Second)
}

// handleGetClusterSummary GET /v1/cluster/summary
func (b *Bridge) handleGetClusterSummary(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := adminContextTimeout(r.Context())
	defer cancel()

	summary, err := b.GetClusterSummary(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// handleGetClusterResources GET /v1/cluster/resources
func (b *Bridge) handleGetClusterResources(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := adminContextTimeout(r.Context())
	defer cancel()

	res, err := b.GetClusterResources(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleGetInstanceDiagnostics GET /v1/instances/{id}/diagnostics
func (b *Bridge) handleGetInstanceDiagnostics(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
		return
	}
	ctx, cancel := adminContextTimeout(r.Context())
	defer cancel()

	diag, err := b.GetInstanceDiagnostics(ctx, instanceID)
	if err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, diag)
}

// handleGetInstanceLogs GET /v1/instances/{id}/logs?tail=200&previous=false&container=
func (b *Bridge) handleGetInstanceLogs(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
		return
	}

	tail := int64(200)
	if v := r.URL.Query().Get("tail"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			tail = n
		}
	}
	previous := r.URL.Query().Get("previous") == "true"
	container := r.URL.Query().Get("container")

	ctx, cancel := adminContextTimeout(r.Context())
	defer cancel()

	resp, err := b.GetInstanceLogs(ctx, instanceID, container, tail, previous)
	if err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleGetMaintenanceMode GET /v1/cluster/maintenance
func (b *Bridge) handleGetMaintenanceMode(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, MaintenanceModeResponse{
		ClusterID:   b.ClusterName,
		Maintenance: b.maintenance.Load(),
		UpdatedAt:   time.Now().UTC(),
	})
}

// handleSetMaintenanceMode PUT /v1/cluster/maintenance
func (b *Bridge) handleSetMaintenanceMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode request body: %w", err))
		return
	}
	b.maintenance.Store(req.Enabled)
	b.Logger.Printf("[Maintenance] mode set to %v", req.Enabled)
	writeJSON(w, http.StatusOK, MaintenanceModeResponse{
		ClusterID:   b.ClusterName,
		Maintenance: req.Enabled,
		UpdatedAt:   time.Now().UTC(),
	})
}

// handleDrainCluster POST /v1/cluster/drain
func (b *Bridge) handleDrainCluster(w http.ResponseWriter, r *http.Request) {
	if !b.draining.CompareAndSwap(false, true) {
		writeError(w, http.StatusConflict, fmt.Errorf("drain already in progress"))
		return
	}

	var req struct {
		Purge bool `json:"purge"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck — empty body defaults to purge=false
	}

	// Enable maintenance mode synchronously so the backend sees it immediately
	// on the next cluster/summary poll, before any instance deletions begin.
	b.maintenance.Store(true)

	op := b.runner.Submit("drain", b.ClusterName, func(ctx context.Context) error {
		defer b.draining.Store(false)
		return b.DrainCluster(ctx, req.Purge)
	})
	writeJSON(w, http.StatusAccepted, op)
}

// handleGetInstanceResources GET /v1/instances/{id}/resources
func (b *Bridge) handleGetInstanceResources(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
		return
	}
	ctx, cancel := adminContextTimeout(r.Context())
	defer cancel()

	res, err := b.GetInstanceResources(ctx, instanceID)
	if err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
