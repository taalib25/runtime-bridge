package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// handleListAgentTemplates GET /v1/agents
func (b *Bridge) handleListAgentTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := b.ListAgentTemplates(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": templates})
}

// handleCreateAgentTemplate POST /v1/agents
func (b *Bridge) handleCreateAgentTemplate(w http.ResponseWriter, r *http.Request) {
	var t AgentTemplate
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if t.Name == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("name is required"))
		return
	}
	created, err := b.CreateAgentTemplate(r.Context(), t)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// handleGetAgentTemplate GET /v1/agents/{agentId}
func (b *Bridge) handleGetAgentTemplate(w http.ResponseWriter, r *http.Request) {
	agentID := mux.Vars(r)["agentId"]
	tmpl, err := b.GetAgentTemplate(r.Context(), agentID)
	if err != nil {
		if isAgentNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, tmpl)
}

// handleUpdateAgentTemplate PUT /v1/agents/{agentId}
func (b *Bridge) handleUpdateAgentTemplate(w http.ResponseWriter, r *http.Request) {
	agentID := mux.Vars(r)["agentId"]
	var t AgentTemplate
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if t.Name == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("name is required"))
		return
	}
	updated, err := b.UpdateAgentTemplate(r.Context(), agentID, t)
	if err != nil {
		if isAgentNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// handleDeleteAgentTemplate DELETE /v1/agents/{agentId}
func (b *Bridge) handleDeleteAgentTemplate(w http.ResponseWriter, r *http.Request) {
	agentID := mux.Vars(r)["agentId"]
	if err := b.DeleteAgentTemplate(r.Context(), agentID); err != nil {
		if isAgentNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agentId": agentID, "status": "deleted"})
}

// handleApplyAgentTemplate POST /v1/workspaces/{id}/agent
func (b *Bridge) handleApplyAgentTemplate(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format"))
		return
	}
	var req ApplyAgentTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if req.AgentID == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("agentId is required"))
		return
	}
	if err := b.ApplyAgentTemplate(r.Context(), workspaceID, req.AgentID); err != nil {
		if isInstanceNotFound(err) || isAgentNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"instanceId": workspaceID,
		"agentId":     req.AgentID,
		"status":      "applied",
	})
}

// handleGetInstanceAgent GET /v1/workspaces/{id}/agent
func (b *Bridge) handleGetInstanceAgent(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format"))
		return
	}
	tmpl, err := b.GetInstanceAgent(r.Context(), workspaceID)
	if err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if tmpl == nil {
		writeJSON(w, http.StatusOK, map[string]any{"agent": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agent": tmpl})
}

func isAgentNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}
