package main

import (
	"context"
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
	workspaceID := mux.Vars(r)["id"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
		return
	}
	ctx, cancel := adminContextTimeout(r.Context())
	defer cancel()

	diag, err := b.GetInstanceDiagnostics(ctx, workspaceID)
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
	workspaceID := mux.Vars(r)["id"]
	if !validInstanceID(workspaceID) {
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

	resp, err := b.GetInstanceLogs(ctx, workspaceID, container, tail, previous)
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

// handleGetInstanceResources GET /v1/instances/{id}/resources
func (b *Bridge) handleGetInstanceResources(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
		return
	}
	ctx, cancel := adminContextTimeout(r.Context())
	defer cancel()

	res, err := b.GetInstanceResources(ctx, workspaceID)
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
