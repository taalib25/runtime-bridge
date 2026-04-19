# Hermes Agent - Troubleshooting Guide

**Comprehensive troubleshooting guide for common Hermes Agent issues.**

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0+

---

## 🔍 Quick Troubleshooting Index

### By Symptom
- [Agent not responding](#agent-not-responding)
- [Platform connection issues](#platform-connection-issues)
- [API server errors](#api-server-errors)
- [Configuration problems](#configuration-problems)
- [Performance issues](#performance-issues)
- [Memory/resource issues](#memory-and-resource-issues)

### By Platform
- [Telegram issues](#telegram-troubleshooting)
- [Discord issues](#discord-troubleshooting)
- [Slack issues](#slack-troubleshooting)
- [WhatsApp issues](#whatsapp-troubleshooting)
- [Email issues](#email-troubleshooting)

---

## Agent Not Responding

### Symptom: Agent doesn't reply to messages

**Diagnosis Steps:**
1. Check if agent is running: `hermes status`
2. Check logs: `hermes logs --tail 50`
3. Verify gateway is active: `hermes gateway list`
4. Check if user is in allowlist: `hermes allowlist show`

**Common Causes & Solutions:**

| Cause | Solution |
|-------|----------|
| Agent process crashed | Restart: `hermes restart` |
| Gateway not connected | Check gateway config: `hermes gateway status <name>` |
| User not in allowlist | Add user: `hermes allowlist add <platform> <user_id>` |
| API key invalid | Verify in `.env`: `grep HERMES_API_KEY ~/.hermes/.env` |
| Model not responding | Check model status: `hermes model status` |
| Rate limited | Wait 60 seconds, check rate limit config |

**Debug Commands:**
```bash
# Enable debug logging
export HERMES_LOG_LEVEL=debug
hermes run

# Check gateway connectivity
hermes gateway test <gateway-name>

# Verify API key
hermes auth verify

# Check model availability
hermes model list
```

---

## Platform Connection Issues

### Telegram Not Connecting

**Symptoms:**
- "Failed to connect to Telegram API"
- "Invalid bot token"
- "Webhook failed"

**Solutions:**

1. **Verify bot token:**
   ```bash
   curl -s https://api.telegram.org/bot<YOUR_TOKEN>/getMe
   ```
   Should return bot info, not error.

2. **Check firewall/network:**
   ```bash
   # Test connectivity to Telegram
   curl -I https://api.telegram.org
   ```

3. **Verify webhook URL (if using webhooks):**
   ```bash
   # Check webhook status
   curl -s https://api.telegram.org/bot<YOUR_TOKEN>/getWebhookInfo
   ```

4. **Restart Telegram gateway:**
   ```bash
   hermes gateway restart telegram
   ```

**Configuration Check:**
```yaml
# config.yaml
gateways:
  - name: telegram
    platform: telegram
    enabled: true
    token: ${TELEGRAM_BOT_TOKEN}  # Must be set in .env
```

### Discord Not Connecting

**Symptoms:**
- "Invalid token"
- "Failed to connect to Discord"
- "Permission denied"

**Solutions:**

1. **Verify bot token:**
   ```bash
   # Token should be in .env
   grep DISCORD_BOT_TOKEN ~/.hermes/.env
   ```

2. **Check bot permissions:**
   - Go to Discord Developer Portal
   - Select your application
   - Check "Bot" permissions include:
     - Send Messages
     - Read Messages/View Channels
     - Read Message History
     - Mention @everyone, @here, and @[Role]

3. **Verify bot is in server:**
   ```bash
   # Check bot status
   hermes gateway status discord
   ```

4. **Check intents:**
   ```yaml
   # config.yaml
   gateways:
     - name: discord
       platform: discord
       intents:
         - message_content
         - guild_messages
         - direct_messages
   ```

### Slack Not Connecting

**Symptoms:**
- "Invalid app token"
- "Socket mode connection failed"
- "Permission denied"

**Solutions:**

1. **Verify tokens:**
   ```bash
   grep SLACK_BOT_TOKEN ~/.hermes/.env
   grep SLACK_APP_TOKEN ~/.hermes/.env
   ```

2. **Check Socket Mode enabled:**
   - Go to Slack App settings
   - Enable "Socket Mode"
   - Verify app token starts with `xapp-`

3. **Verify scopes:**
   Required scopes:
   - `chat:write`
   - `chat:write.public`
   - `files:read`
   - `files:write`
   - `app_mentions:read`

4. **Restart Slack gateway:**
   ```bash
   hermes gateway restart slack
   ```

---

## API Server Errors

### Symptom: "Connection refused" on API server

**Solutions:**

1. **Check if API server is running:**
   ```bash
   curl http://localhost:8080/health
   ```

2. **Verify port is not in use:**
   ```bash
   lsof -i :8080
   ```

3. **Check API server logs:**
   ```bash
   hermes logs api --tail 50
   ```

4. **Restart API server:**
   ```bash
   hermes api restart
   ```

### Symptom: "Invalid API key"

**Solutions:**

1. **Verify API key in request:**
   ```bash
   curl -H "Authorization: Bearer YOUR_API_KEY" \
     http://localhost:8080/v1/models
   ```

2. **Check API key in config:**
   ```bash
   grep HERMES_API_KEY ~/.hermes/.env
   ```

3. **Regenerate API key:**
   ```bash
   hermes auth generate-key
   ```

### Symptom: Open WebUI can't connect

**Solutions:**

1. **Verify API server is accessible:**
   ```bash
   curl http://localhost:8080/health
   ```

2. **Check CORS configuration:**
   ```yaml
   # config.yaml
   api_server:
     cors:
       allowed_origins:
         - "http://localhost:3000"
         - "http://localhost:8000"
   ```

3. **Verify Open WebUI URL:**
   - Check browser console for CORS errors
   - Verify API server URL in Open WebUI settings

---

## Configuration Problems

### Symptom: "Invalid configuration"

**Solutions:**

1. **Validate YAML syntax:**
   ```bash
   hermes config validate
   ```

2. **Check for required fields:**
   ```bash
   # config.yaml must have:
   # - model
   # - gateways (at least one)
   # - api_server (optional but recommended)
   ```

3. **Verify environment variables:**
   ```bash
   hermes config show-env
   ```

### Symptom: "Environment variable not found"

**Solutions:**

1. **Check .env file:**
   ```bash
   cat ~/.hermes/.env
   ```

2. **Verify variable name:**
   ```bash
   # Common variables:
   HERMES_API_KEY
   TELEGRAM_BOT_TOKEN
   DISCORD_BOT_TOKEN
   SLACK_BOT_TOKEN
   OPENAI_API_KEY
   ```

3. **Reload environment:**
   ```bash
   source ~/.hermes/.env
   hermes restart
   ```

### Symptom: "Invalid allowlist configuration"

**Solutions:**

1. **Check allowlist format:**
   ```yaml
   # config.yaml
   gateways:
     - name: telegram
       allowlist:
         users:
           - "123456789"  # Telegram user ID
           - "987654321"
   ```

2. **Verify user IDs:**
   ```bash
   # Get your Telegram ID
   hermes telegram get-id
   ```

3. **Test allowlist:**
   ```bash
   hermes allowlist test telegram 123456789
   ```

---

## Performance Issues

### Symptom: Slow response times

**Diagnosis:**

1. **Check system resources:**
   ```bash
   hermes system info
   ```

2. **Monitor API latency:**
   ```bash
   hermes metrics latency
   ```

3. **Check model performance:**
   ```bash
   hermes model benchmark
   ```

**Solutions:**

| Issue | Solution |
|-------|----------|
| High CPU usage | Reduce concurrent requests, check model size |
| High memory usage | Reduce batch size, enable memory optimization |
| Slow API responses | Check network latency, verify model endpoint |
| Slow message processing | Check gateway queue, verify platform API |

### Symptom: Timeouts

**Solutions:**

1. **Increase timeout values:**
   ```yaml
   # config.yaml
   api_server:
     timeout: 60  # seconds
   gateways:
     - name: telegram
       timeout: 30
   ```

2. **Check network connectivity:**
   ```bash
   ping api.openai.com
   ping api.telegram.org
   ```

3. **Verify model endpoint:**
   ```bash
   curl -I https://api.openai.com/v1/models
   ```

---

## Memory and Resource Issues

### Symptom: "Out of memory"

**Solutions:**

1. **Check memory usage:**
   ```bash
   hermes system memory
   ```

2. **Reduce model size:**
   ```yaml
   # config.yaml
   model:
     name: "gpt-3.5-turbo"  # Smaller model
     max_tokens: 1000  # Reduce max tokens
   ```

3. **Enable memory optimization:**
   ```yaml
   # config.yaml
   optimization:
     memory: true
     cache_size: 100  # MB
   ```

4. **Clear cache:**
   ```bash
   hermes cache clear
   ```

### Symptom: "Disk space full"

**Solutions:**

1. **Check disk usage:**
   ```bash
   du -sh ~/.hermes/
   ```

2. **Clear logs:**
   ```bash
   hermes logs clear --older-than 7d
   ```

3. **Clear cache:**
   ```bash
   hermes cache clear
   ```

---

## Telegram Troubleshooting

### Issue: Webhook not working

**Solutions:**
1. Verify webhook URL is publicly accessible
2. Check SSL certificate is valid
3. Verify firewall allows incoming connections on webhook port

### Issue: Commands not recognized

**Solutions:**
1. Register commands with Telegram: `hermes telegram register-commands`
2. Verify command format: `/command`
3. Check command permissions in allowlist

---

## Discord Troubleshooting

### Issue: Slash commands not appearing

**Solutions:**
1. Sync commands: `hermes discord sync-commands`
2. Check bot permissions in server
3. Verify command scopes in Discord Developer Portal

### Issue: Embeds not rendering

**Solutions:**
1. Check embed format is valid
2. Verify bot has permission to send embeds
3. Check message length (max 2000 characters)

---

## Slack Troubleshooting

### Issue: App not responding in channels

**Solutions:**
1. Verify app is installed in workspace
2. Check app is in channel: `/invite @hermes`
3. Verify message format (Slack uses different formatting)

### Issue: File uploads failing

**Solutions:**
1. Check file size limits
2. Verify bot has `files:write` scope
3. Check file type is allowed

---

## Getting Help

### Debug Information

Collect this information when reporting issues:

```bash
# System info
hermes system info

# Configuration (without secrets)
hermes config show --no-secrets

# Recent logs
hermes logs --tail 100 > logs.txt

# Gateway status
hermes gateway list

# Model status
hermes model status
```

### Support Resources

- **GitHub Issues**: https://github.com/NousResearch/hermes-agent/issues
- **Discord Community**: https://discord.gg/NousResearch
- **Documentation**: https://hermes-agent.nousresearch.com/docs/
- **Email Support**: support@nousresearch.com

---

## Common Error Messages

| Error | Meaning | Solution |
|-------|---------|----------|
| `ECONNREFUSED` | Connection refused | Check if service is running |
| `ETIMEDOUT` | Connection timeout | Check network, increase timeout |
| `ENOTFOUND` | DNS resolution failed | Check hostname, verify network |
| `EACCES` | Permission denied | Check file permissions, run with correct user |
| `ENOMEM` | Out of memory | Free up memory, reduce model size |
| `ENOSPC` | No space left | Clear cache/logs, free up disk space |

---

## Best Practices for Troubleshooting

1. **Check logs first**: `hermes logs --tail 50`
2. **Verify configuration**: `hermes config validate`
3. **Test connectivity**: `hermes gateway test <name>`
4. **Check system resources**: `hermes system info`
5. **Restart service**: `hermes restart`
6. **Enable debug logging**: `export HERMES_LOG_LEVEL=debug`
7. **Collect diagnostics**: `hermes diagnostics collect`
8. **Check documentation**: https://hermes-agent.nousresearch.com/docs/

---

## Still Need Help?

If you can't find a solution:

1. Collect debug information (see above)
2. Check GitHub issues for similar problems
3. Post in Discord community
4. Open a GitHub issue with:
   - Error message
   - Steps to reproduce
   - System information
   - Configuration (without secrets)
   - Recent logs
