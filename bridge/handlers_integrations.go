package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// handleGetInstanceIntegrations GET /v1/instances/{id}/integrations
func (b *Bridge) handleGetInstanceIntegrations(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
		return
	}
	statuses, err := b.GetInstanceIntegrations(r.Context(), workspaceID)
	if err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"integrations": statuses})
}

// handleEnableIntegration POST /v1/instances/{id}/integrations/{platform}
func (b *Bridge) handleEnableIntegration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceID := vars["id"]
	platform := vars["platform"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
		return
	}

	// Decode into a generic map first; we'll validate per-platform below.
	cfg := make(map[string]string)
	if err := decodeIntegrationRequest(r, platform, cfg); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := b.EnableIntegration(r.Context(), workspaceID, platform, cfg); err != nil {
		if isInstanceNotFound(err) {
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

// handleDisableIntegration DELETE /v1/instances/{id}/integrations/{platform}
func (b *Bridge) handleDisableIntegration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workspaceID := vars["id"]
	platform := vars["platform"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
		return
	}
	if err := b.DisableIntegration(r.Context(), workspaceID, platform); err != nil {
		if isInstanceNotFound(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"platform": platform, "status": "disabled"})
}

// decodeIntegrationRequest is a thin wrapper around decodeIntegrationCfg for
// HTTP handlers — it reads the body once and delegates platform-specific logic.
func decodeIntegrationRequest(r *http.Request, platform string, cfg map[string]string) error {
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	result, err := decodeIntegrationCfg(platform, raw)
	if err != nil {
		return err
	}
	for k, v := range result {
		cfg[k] = v
	}
	return nil
}

// decodeIntegrationCfg decodes platform-specific JSON into a flat cfg map of
// env-var-name → value pairs (both secret and plain). Used by the per-platform
// handler and the bulk PUT handler.
func decodeIntegrationCfg(platform string, data json.RawMessage) (map[string]string, error) {
	cfg := make(map[string]string)
	switch platform {
	case "telegram":
		var req TelegramIntegrationRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}
		if req.BotToken == "" {
			return nil, fmt.Errorf("botToken is required for telegram")
		}
		if req.WebhookURL != "" && req.WebhookSecret == "" {
			return nil, fmt.Errorf("webhookSecret is required when webhookUrl is set")
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
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}
		if req.BotToken == "" {
			return nil, fmt.Errorf("botToken is required for discord")
		}
		setIfNotEmpty(cfg, "DISCORD_BOT_TOKEN", req.BotToken)
		setIfNotEmpty(cfg, "DISCORD_ALLOWED_USERS", req.AllowedUsers)
		setIfNotEmpty(cfg, "DISCORD_ALLOWED_ROLES", req.AllowedRoles)
		setIfNotEmpty(cfg, "DISCORD_HOME_CHANNEL", req.HomeChannel)
		setIfNotEmpty(cfg, "DISCORD_REQUIRE_MENTION", req.RequireMention)
		setIfNotEmpty(cfg, "DISCORD_FREE_RESPONSE_CHANNELS", req.FreeResponseChannels)
		setIfNotEmpty(cfg, "DISCORD_IGNORED_CHANNELS", req.IgnoredChannels)

	case "slack":
		var req SlackIntegrationRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}
		if req.BotToken == "" {
			return nil, fmt.Errorf("botToken is required for slack")
		}
		if req.AppToken == "" {
			return nil, fmt.Errorf("appToken is required for slack (Socket Mode)")
		}
		setIfNotEmpty(cfg, "SLACK_BOT_TOKEN", req.BotToken)
		setIfNotEmpty(cfg, "SLACK_APP_TOKEN", req.AppToken)
		setIfNotEmpty(cfg, "SLACK_ALLOWED_USERS", req.AllowedUsers)
		setIfNotEmpty(cfg, "SLACK_ALLOWED_CHANNELS", req.AllowedChannels)
		setIfNotEmpty(cfg, "SLACK_HOME_CHANNEL", req.HomeChannel)
		setIfNotEmpty(cfg, "SLACK_HOME_CHANNEL_NAME", req.HomeChannelName)

	case "whatsapp":
		var req WhatsAppIntegrationRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}
		cfg["WHATSAPP_ENABLED"] = "true"
		setIfNotEmpty(cfg, "WHATSAPP_MODE", req.Mode)
		setIfNotEmpty(cfg, "WHATSAPP_ALLOWED_USERS", req.AllowedUsers)
		if req.AllowAllUsers {
			cfg["WHATSAPP_ALLOW_ALL_USERS"] = "true"
		}

	case "signal":
		var req SignalIntegrationRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}
		if req.HTTPURL == "" {
			return nil, fmt.Errorf("httpUrl is required for signal")
		}
		if req.Account == "" {
			return nil, fmt.Errorf("account is required for signal")
		}
		// All Signal config is plain env — no secret tokens needed.
		setIfNotEmpty(cfg, "SIGNAL_HTTP_URL", req.HTTPURL)
		setIfNotEmpty(cfg, "SIGNAL_ACCOUNT", req.Account)
		setIfNotEmpty(cfg, "SIGNAL_ALLOWED_USERS", req.AllowedUsers)
		setIfNotEmpty(cfg, "SIGNAL_GROUP_ALLOWED_USERS", req.GroupAllowedUsers)
		setIfNotEmpty(cfg, "SIGNAL_HOME_CHANNEL", req.HomeChannel)
		if req.AllowAllUsers {
			cfg["SIGNAL_ALLOW_ALL_USERS"] = "true"
		}
		if req.IgnoreStories {
			cfg["SIGNAL_IGNORE_STORIES"] = "true"
		}

	case "dingtalk":
		var req DingTalkIntegrationRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}
		if req.ClientID == "" {
			return nil, fmt.Errorf("clientId is required for dingtalk")
		}
		if req.ClientSecret == "" {
			return nil, fmt.Errorf("clientSecret is required for dingtalk")
		}
		setIfNotEmpty(cfg, "DINGTALK_CLIENT_ID", req.ClientID)
		setIfNotEmpty(cfg, "DINGTALK_CLIENT_SECRET", req.ClientSecret)
		setIfNotEmpty(cfg, "DINGTALK_ALLOWED_USERS", req.AllowedUsers)

	case "feishu":
		var req FeishuIntegrationRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}
		if req.AppID == "" {
			return nil, fmt.Errorf("appId is required for feishu")
		}
		if req.AppSecret == "" {
			return nil, fmt.Errorf("appSecret is required for feishu")
		}
		setIfNotEmpty(cfg, "FEISHU_APP_ID", req.AppID)
		setIfNotEmpty(cfg, "FEISHU_APP_SECRET", req.AppSecret)
		setIfNotEmpty(cfg, "FEISHU_ENCRYPT_KEY", req.EncryptKey)
		setIfNotEmpty(cfg, "FEISHU_VERIFICATION_TOKEN", req.VerificationToken)
		setIfNotEmpty(cfg, "FEISHU_DOMAIN", req.Domain)
		setIfNotEmpty(cfg, "FEISHU_CONNECTION_MODE", req.ConnectionMode)
		setIfNotEmpty(cfg, "FEISHU_ALLOWED_USERS", req.AllowedUsers)
		setIfNotEmpty(cfg, "FEISHU_HOME_CHANNEL", req.HomeChannel)

	case "wecom":
		var req WeComIntegrationRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}
		if req.BotID == "" {
			return nil, fmt.Errorf("botId is required for wecom")
		}
		if req.Secret == "" {
			return nil, fmt.Errorf("secret is required for wecom")
		}
		setIfNotEmpty(cfg, "WECOM_BOT_ID", req.BotID)
		setIfNotEmpty(cfg, "WECOM_SECRET", req.Secret)
		setIfNotEmpty(cfg, "WECOM_WEBSOCKET_URL", req.WebsocketURL)
		setIfNotEmpty(cfg, "WECOM_ALLOWED_USERS", req.AllowedUsers)
		setIfNotEmpty(cfg, "WECOM_HOME_CHANNEL", req.HomeChannel)

	case "bluebubbles":
		var req BlueBubblesIntegrationRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}
		if req.ServerURL == "" {
			return nil, fmt.Errorf("serverUrl is required for bluebubbles")
		}
		if req.Password == "" {
			return nil, fmt.Errorf("password is required for bluebubbles")
		}
		setIfNotEmpty(cfg, "BLUEBUBBLES_SERVER_URL", req.ServerURL)
		setIfNotEmpty(cfg, "BLUEBUBBLES_PASSWORD", req.Password)
		setIfNotEmpty(cfg, "BLUEBUBBLES_WEBHOOK_HOST", req.WebhookHost)
		setIfNotEmpty(cfg, "BLUEBUBBLES_WEBHOOK_PORT", req.WebhookPort)
		setIfNotEmpty(cfg, "BLUEBUBBLES_ALLOWED_USERS", req.AllowedUsers)
		if req.AllowAllUsers {
			cfg["BLUEBUBBLES_ALLOW_ALL_USERS"] = "true"
		}

	case "email":
		var req EmailIntegrationRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}
		if req.Address == "" {
			return nil, fmt.Errorf("address is required for email")
		}
		if req.Password == "" {
			return nil, fmt.Errorf("password is required for email")
		}
		if req.IMAPHost == "" {
			return nil, fmt.Errorf("imapHost is required for email")
		}
		if req.SMTPHost == "" {
			return nil, fmt.Errorf("smtpHost is required for email")
		}
		setIfNotEmpty(cfg, "EMAIL_ADDRESS", req.Address)
		setIfNotEmpty(cfg, "EMAIL_PASSWORD", req.Password)
		setIfNotEmpty(cfg, "EMAIL_IMAP_HOST", req.IMAPHost)
		setIfNotEmpty(cfg, "EMAIL_SMTP_HOST", req.SMTPHost)
		setIfNotEmpty(cfg, "EMAIL_IMAP_PORT", req.IMAPPort)
		setIfNotEmpty(cfg, "EMAIL_SMTP_PORT", req.SMTPPort)
		setIfNotEmpty(cfg, "EMAIL_ALLOWED_USERS", req.AllowedUsers)
		setIfNotEmpty(cfg, "EMAIL_HOME_ADDRESS", req.HomeAddress)
		setIfNotEmpty(cfg, "EMAIL_POLL_INTERVAL", req.PollInterval)
		if req.AllowAllUsers {
			cfg["EMAIL_ALLOW_ALL_USERS"] = "true"
		}

	default:
		return nil, fmt.Errorf("unsupported platform %q; supported: telegram, discord, slack, whatsapp, signal, email, dingtalk, feishu, wecom, bluebubbles", platform)
	}
	return cfg, nil
}

// handleSetIntegrations PUT /v1/instances/{id}/integrations
// Converges the instance to exactly the set of platforms in the request body.
// Platforms absent from the body are disabled; an empty body disables all.
func (b *Bridge) handleSetIntegrations(w http.ResponseWriter, r *http.Request) {
	workspaceID := mux.Vars(r)["id"]
	if !validInstanceID(workspaceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId format"))
		return
	}

	var desired IntegrationsPutRequest
	if err := json.NewDecoder(r.Body).Decode(&desired); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	// Validate all platform names before submitting the async op.
	for platform := range desired {
		if _, ok := platformSecretKeys[platform]; !ok {
			writeError(w, http.StatusBadRequest, fmt.Errorf(
				"unsupported platform %q; supported: telegram, discord, slack, whatsapp, signal, email, dingtalk, feishu, wecom, bluebubbles", platform,
			))
			return
		}
	}

	op := b.submitOperation("set-integrations", workspaceID, func(ctx context.Context) error {
		return b.SetIntegrations(ctx, workspaceID, desired)
	})
	writeJSON(w, http.StatusAccepted, op)
}

func setIfNotEmpty(m map[string]string, key, value string) {
	if value != "" {
		m[key] = value
	}
}
