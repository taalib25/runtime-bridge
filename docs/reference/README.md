# Hermes Agent - Reference Documentation

Complete reference documentation for Hermes Agent tools, configurations, and APIs.

---

## Documentation Files

### 1. **TOOLS_INDEX.md** - Quick Reference
- All 40+ tools organized by category
- Tools by backend/provider
- Tools by use case
- Configuration quick reference
- Tool discovery commands

**Use this for**: Quick lookups, finding the right tool, understanding tool categories

### 2. **TOOLS_REFERENCE.md** - Complete Documentation
- Detailed documentation for every tool
- Full parameter specifications
- Return format examples
- Backend information
- Configuration options
- Tool disable/enable options
- Toolset compositions
- Best practices

**Use this for**: Understanding how to use specific tools, detailed parameter info, examples

### 3. **TOOL_PARAMETERS_SCHEMA.md** - JSON Schemas
- Complete JSON Schema definitions for all tool parameters
- Input validation schemas
- Return value schemas
- Type definitions

**Use this for**: API integration, validation, schema-driven development

---

## Quick Start

### Find a Tool

```bash
# List all tools
hermes tools list

# Get tool info
hermes tools info browser_navigate

# List tools by category
hermes tools list --category web
```

### Use a Tool

```bash
# Execute a tool directly
hermes run "Navigate to https://example.com and take a screenshot"

# Use specific tool
hermes tools execute browser_navigate --url "https://example.com"
```

### Configure Tools

```yaml
# ~/.hermes/config.yaml
tools:
  enabled_categories:
    - web
    - code
    - system
  
  disabled_tools:
    - file_delete
    - system_reboot
  
  timeout: 30
  max_concurrent: 5
```

---

## Tool Categories

### Browser Tools (10)
Automate web browser interactions using Playwright.

**Tools**: navigate, click, fill, screenshot, get_text, get_html, wait_for_selector, get_cookies, set_cookies, execute_script

**Use for**: Web automation, testing, scraping, form filling

**See**: [TOOLS_REFERENCE.md - Browser Tools](./TOOLS_REFERENCE.md#browser-tools)

### Web Tools (2)
Fetch and parse web content.

**Tools**: web_fetch, web_parse

**Use for**: API calls, web scraping, data extraction

**See**: [TOOLS_REFERENCE.md - Web Tools](./TOOLS_REFERENCE.md#web-tools)

### Terminal Tools (2)
Execute shell commands locally or remotely.

**Tools**: terminal_execute, terminal_stream

**Use for**: System administration, running scripts, automation

**See**: [TOOLS_REFERENCE.md - Terminal Tools](./TOOLS_REFERENCE.md#terminal-tools)

### File Tools (4)
Read, write, and manage files.

**Tools**: file_read, file_write, file_delete, file_list

**Use for**: File operations, data processing, configuration management

**See**: [TOOLS_REFERENCE.md - File Tools](./TOOLS_REFERENCE.md#file-tools)

### Vision & Media Tools (3)
Analyze images and convert media formats.

**Tools**: vision_analyze_image, media_convert, media_extract_audio

**Use for**: Image analysis, media processing, video/audio conversion

**See**: [TOOLS_REFERENCE.md - Vision & Media Tools](./TOOLS_REFERENCE.md#vision--media-tools)

### Code Execution (1)
Execute code in multiple languages.

**Tools**: code_execute

**Use for**: Running Python, JavaScript, Go, Rust, Java, etc.

**See**: [TOOLS_REFERENCE.md - Code Execution](./TOOLS_REFERENCE.md#code-execution-tools)

### Orchestration Tools (4)
Create and manage workflows.

**Tools**: workflow_create, workflow_execute, workflow_status, workflow_cancel

**Use for**: Multi-step automation, complex workflows, task orchestration

**See**: [TOOLS_REFERENCE.md - Orchestration Tools](./TOOLS_REFERENCE.md#orchestration-tools)

### Memory Tools (3)
Store and retrieve information.

**Tools**: memory_store, memory_retrieve, memory_search

**Use for**: Context management, fact storage, knowledge base

**See**: [TOOLS_REFERENCE.md - Memory Tools](./TOOLS_REFERENCE.md#memory-tools)

### Scheduling Tools (2)
Schedule recurring or one-time tasks.

**Tools**: schedule_create, schedule_cancel

**Use for**: Cron jobs, recurring tasks, automation scheduling

**See**: [TOOLS_REFERENCE.md - Scheduling Tools](./TOOLS_REFERENCE.md#scheduling-tools)

### Home Assistant Tools (4)
Control smart home devices.

**Tools**: ha_get_state, ha_set_state, ha_call_service, ha_get_history

**Use for**: Smart home automation, device control, home monitoring

**See**: [TOOLS_REFERENCE.md - Home Assistant Tools](./TOOLS_REFERENCE.md#home-assistant-tools)

### RL Training Tools (10)
Train and manage reinforcement learning models.

**Tools**: rl_train_model, rl_evaluate_model, rl_predict, rl_hyperparameter_tune, rl_save_model, rl_load_model, rl_get_policy, rl_compare_models, rl_export_model, rl_monitor_training

**Use for**: ML model training, RL agent development, model optimization

**See**: [TOOLS_REFERENCE.md - RL Training Tools](./TOOLS_REFERENCE.md#rl-training-tools)

### MCP Dynamic Tools
Dynamically loaded from Model Context Protocol servers.

**Use for**: Extending Hermes with custom tools, integrations

**See**: [TOOLS_REFERENCE.md - MCP Dynamic Tools](./TOOLS_REFERENCE.md#mcp-dynamic-tools)

---

## Common Tasks

### Web Scraping

```bash
# Using browser tools
hermes run "
1. Navigate to https://example.com
2. Wait for .content to load
3. Extract all text
4. Take a screenshot
"

# Using web tools
hermes run "
1. Fetch https://example.com
2. Parse HTML to extract titles
3. Save results to file
"
```

### Data Processing

```bash
# Read, process, write
hermes run "
1. Read data.csv
2. Execute Python to process data
3. Write results.csv
"
```

### Automation

```bash
# Create workflow
hermes workflow create daily_report \
  --step "fetch_data" \
  --step "process_data" \
  --step "send_report" \
  --schedule "0 9 * * MON"
```

### Smart Home Control

```bash
# Get device state
hermes tools execute ha_get_state --entity_id "light.living_room"

# Turn on light
hermes tools execute ha_set_state \
  --entity_id "light.living_room" \
  --state "on"
```

---

## Configuration Reference

### Tool Settings

```yaml
tools:
  # Enable/disable categories
  enabled_categories:
    - web
    - code
    - system
    - data
    - communication
    - productivity
    - ai
    - custom
  
  # Disable specific tools
  disabled_tools:
    - system_reboot
    - network_change
    - file_delete
  
  # Global timeout
  timeout: 30
  
  # Max concurrent executions
  max_concurrent: 5
  
  # Retry policy
  retry:
    max_attempts: 3
    backoff_factor: 2
```

### Backend Configuration

```yaml
# Terminal backend
terminal:
  backend: local  # or: docker, ssh, modal, daytona
  timeout: 30

# Memory backend
memory:
  backend: memory  # or: redis, database
  type: hybrid

# RL training backend
rl_training:
  backend: stable_baselines3
  gpu_enabled: true
```

### Approval Requirements

```yaml
approvals:
  enabled: true
  dangerous_operations:
    - file_delete
    - system_reboot
    - network_change
  timeout: 300
  channels:
    - telegram
    - email
```

---

## API Integration

### Using Tools via API

```bash
# Get tool info
curl http://localhost:8642/api/tools/browser_navigate

# Execute tool
curl -X POST http://localhost:8642/api/tools/execute \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "browser_navigate",
    "params": {
      "url": "https://example.com"
    }
  }'

# List all tools
curl http://localhost:8642/api/tools
```

### Tool Schemas

All tools follow this schema:

**Input**:
```json
{
  "tool": "tool_name",
  "params": {
    "param1": "value1",
    "param2": "value2"
  }
}
```

**Output**:
```json
{
  "success": true,
  "data": {...},
  "error": null,
  "execution_time_ms": 1234
}
```

---

## Troubleshooting

### Tool Not Found

```bash
# List available tools
hermes tools list

# Check if tool is disabled
hermes tools list --disabled

# Enable tool
hermes tools enable tool_name
```

### Tool Timeout

Increase timeout in config:

```yaml
tools:
  timeout: 60  # Increase from default 30
```

### Tool Execution Failed

Check logs:

```bash
hermes logs tools --level DEBUG
```

### Permission Denied

Check approvals:

```bash
hermes approvals list
hermes approvals approve <approval_id>
```

---

## Best Practices

1. **Use Toolsets**: Group related tools for better organization
2. **Set Timeouts**: Configure appropriate timeouts for your use case
3. **Enable Approvals**: Require approval for dangerous operations
4. **Monitor Usage**: Track tool execution metrics
5. **Error Handling**: Implement proper error handling in workflows
6. **Rate Limiting**: Enable rate limiting for external APIs
7. **Caching**: Cache results when possible
8. **Logging**: Enable detailed logging for debugging

---

## Related Documentation

- **Configuration Reference**: [../configuration/CONFIG_REFERENCE.md](../configuration/CONFIG_REFERENCE.md)
- **Quick Start Guide**: [../guides/QUICK_START.md](../guides/QUICK_START.md)
- **Workflow Guide**: [../guides/WORKFLOW_GUIDE.md](../guides/WORKFLOW_GUIDE.md)
- **API Server Integration**: [../guides/API_SERVER_INTEGRATION.md](../guides/API_SERVER_INTEGRATION.md)

---

## Tool Statistics

| Metric | Value |
|--------|-------|
| Total Tools | 40+ |
| Tool Categories | 12 |
| Browser Tools | 10 |
| RL Training Tools | 10 |
| Orchestration Tools | 4 |
| Home Assistant Tools | 4 |
| File Tools | 4 |
| Memory Tools | 3 |
| Vision & Media Tools | 3 |
| Terminal Tools | 2 |
| Web Tools | 2 |
| Scheduling Tools | 2 |
| Code Execution Tools | 1 |
| MCP Dynamic Tools | Unlimited |

---

## Support

- **Documentation**: https://hermes-agent.nousresearch.com/docs/
- **GitHub**: https://github.com/NousResearch/hermes-agent
- **Discord**: https://discord.gg/NousResearch
- **Issues**: https://github.com/NousResearch/hermes-agent/issues

---

**Last Updated**: April 18, 2026
**Version**: 1.0.0
