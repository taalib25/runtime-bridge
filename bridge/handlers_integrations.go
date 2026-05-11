package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// handleGetWorkspaceIntegrations GET /v1/workspaces/{id}/integrations
func (b *Bridge) handleGetWorkspaceIntegrations(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validWorkspaceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format"))
		return
	}
	statuses, err := b.GetWorkspaceIntegrations(r.Context(), workspaceID)
	if err != nil {
		if isWorkspaceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"integrations": statuses})
}

// handleEnableIntegration POST /v1/workspaces/{id}/integrations/{platform}
func (b *Bridge) handleEnableIntegration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceID := vars["id"]
	platform := vars["platform"]
	if !validWorkspaceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format"))
		return
	}

	// Decode into a generic map first; we'll validate per-platform below.
	cfg := make(map[string]string)
	if err := decodeIntegrationRequest(r, platform, cfg); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := b.EnableIntegration(r.Context(), workspaceID, platform, cfg); err != nil {
		if isWorkspaceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := map[string]any{"platform": platform, "status": "active"}
	if platform == "whatsapp" {
		resp["status"] = "needs_pairing"
		resp["note"] = "exec into the workspace pod and run `hermes whatsapp` to scan the QR code"
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleDisableIntegration DELETE /v1/workspaces/{id}/integrations/{platform}
func (b *Bridge) handleDisableIntegration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceID := vars["id"]
	platform := vars["platform"]
	if !validWorkspaceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid workspaceId format"))
		return
	}
	if err := b.DisableIntegration(r.Context(), workspaceID, platform); err != nil {
		if isWorkspaceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"platform": platform, "status": "disabled"})
}

// decodeIntegrationRequest decodes a platform-specific JSON body into cfg.
// It validates required fields and maps struct fields to the flat cfg map.
func decodeIntegrationRequest(r *http.Request, platform string, cfg map[string]string) error {
	switch platform {
	case "telegram":
		var req TelegramIntegrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return fmt.Errorf("invalid request body: %w", err)
		}
		if req.BotToken == "" {
			return fmt.Errorf("botToken is required for telegram")
		}
		if req.WebhookURL != "" && req.WebhookSecret == "" {
			return fmt.Errorf("webhookSecret is required when webhookUrl is set")
		}
		setIfNotEmpty(cfg, "TELEGRAM_BOT_TOKEN", req.BotToken)
		setIfNotEmpty(cfg, "TELEGRAM_WEBHOOK_SECRET", req.WebhookSecret)
		setIfNotEmpty(cfg, "TELEGRAM_ALLOWED_USERS", req.AllowedUsers)
		setIfNotEmpty(cfg, "TELEGRAM_GROUP_ALLOWED_USERS", req.GroupAllowedUsers)
		setIfNotEmpty(cfg, "TELEGRAM_GROUP_ALLOWED_CHATS", req.GroupAllowedChats)
		setIfNotEmpty(cfg, "TELEGRAM_HOME_CHANNEL", req.HomeChannel)
		setIfNotEmpty(cfg, "TELEGRAM_WEBHOOK_URL", req.WebhookURL)
		setIfNotEmpty(cfg, "TELEGRAM_WEBHOOK_PORT", req.WebhookPort)

	case "discord":
		var req DiscordIntegrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return fmt.Errorf("invalid request body: %w", err)
		}
		if req.BotToken == "" {
			return fmt.Errorf("botToken is required for discord")
		}
		setIfNotEmpty(cfg, "DISCORD_BOT_TOKEN", req.BotToken)
		setIfNotEmpty(cfg, "DISCORD_ALLOWED_USERS", req.AllowedUsers)
		setIfNotEmpty(cfg, "DISCORD_ALLOWED_ROLES", req.AllowedRoles)
		setIfNotEmpty(cfg, "DISCORD_ALLOWED_CHANNELS", req.AllowedChannels)
		setIfNotEmpty(cfg, "DISCORD_HOME_CHANNEL", req.HomeChannel)
		setIfNotEmpty(cfg, "DISCORD_REQUIRE_MENTION", req.RequireMention)
		setIfNotEmpty(cfg, "DISCORD_FREE_RESPONSE_CHANNELS", req.FreeResponseChannels)

	case "slack":
		var req SlackIntegrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return fmt.Errorf("invalid request body: %w", err)
		}
		if req.BotToken == "" {
			return fmt.Errorf("botToken is required for slack")
		}
		if req.AppToken == "" {
			return fmt.Errorf("appToken is required for slack (Socket Mode)")
		}
		setIfNotEmpty(cfg, "SLACK_BOT_TOKEN", req.BotToken)
		setIfNotEmpty(cfg, "SLACK_APP_TOKEN", req.AppToken)
		setIfNotEmpty(cfg, "SLACK_ALLOWED_USERS", req.AllowedUsers)
		setIfNotEmpty(cfg, "SLACK_HOME_CHANNEL", req.HomeChannel)

	case "whatsapp":
		var req WhatsAppIntegrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return fmt.Errorf("invalid request body: %w", err)
		}
		cfg["WHATSAPP_ENABLED"] = "true"
		setIfNotEmpty(cfg, "WHATSAPP_MODE", req.Mode)
		setIfNotEmpty(cfg, "WHATSAPP_ALLOWED_USERS", req.AllowedUsers)
		if req.AllowAllUsers {
			cfg["WHATSAPP_ALLOW_ALL_USERS"] = "true"
		}

	case "signal":
		var req SignalIntegrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return fmt.Errorf("invalid request body: %w", err)
		}
		if req.HTTPURL == "" {
			return fmt.Errorf("httpUrl is required for signal")
		}
		if req.Account == "" {
			return fmt.Errorf("account is required for signal")
		}
		setIfNotEmpty(cfg, "SIGNAL_HTTP_URL", req.HTTPURL)
		setIfNotEmpty(cfg, "SIGNAL_ACCOUNT", req.Account)
		setIfNotEmpty(cfg, "SIGNAL_ALLOWED_USERS", req.AllowedUsers)
		setIfNotEmpty(cfg, "SIGNAL_GROUP_ALLOWED_USERS", req.GroupAllowedUsers)
		if req.AllowAllUsers {
			cfg["SIGNAL_ALLOW_ALL_USERS"] = "true"
		}

	default:
		return fmt.Errorf("unsupported platform %q; supported: telegram, discord, slack, whatsapp, signal", platform)
	}
	return nil
}

func setIfNotEmpty(m map[string]string, key, value string) {
	if value != "" {
		m[key] = value
	}
}
