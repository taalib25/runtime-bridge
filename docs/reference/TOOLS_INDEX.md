# Hermes Agent - Tools Index

**Quick reference for all 40+ tools in Hermes Agent**

---

## Tools by Category

### Browser Tools (10)
| Tool | Function | Backend |
|------|----------|---------|
| `browser_navigate` | Navigate to URL | Playwright |
| `browser_click` | Click element | Playwright |
| `browser_fill` | Fill form field | Playwright |
| `browser_screenshot` | Take screenshot | Playwright |
| `browser_get_text` | Extract text | Playwright |
| `browser_get_html` | Extract HTML | Playwright |
| `browser_wait_for_selector` | Wait for element | Playwright |
| `browser_get_cookies` | Get cookies | Playwright |
| `browser_set_cookies` | Set cookies | Playwright |
| `browser_execute_script` | Execute JavaScript | Playwright |

### Web Tools (2)
| Tool | Function | Backend |
|------|----------|---------|
| `web_fetch` | Fetch URL content | HTTP client |
| `web_parse` | Parse HTML/JSON | BeautifulSoup/lxml |

### Terminal Tools (2)
| Tool | Function | Backend |
|------|----------|---------|
| `terminal_execute` | Execute shell command | subprocess/Docker/SSH |
| `terminal_stream` | Stream command output | subprocess |

### File Tools (4)
| Tool | Function | Backend |
|------|----------|---------|
| `file_read` | Read file | File system |
| `file_write` | Write file | File system |
| `file_delete` | Delete file | File system |
| `file_list` | List directory | File system |

### Vision & Media Tools (3)
| Tool | Function | Backend |
|------|----------|---------|
| `vision_analyze_image` | Analyze image | Claude/GPT-4V |
| `media_convert` | Convert media format | FFmpeg |
| `media_extract_audio` | Extract audio from video | FFmpeg |

### Code Execution Tools (1)
| Tool | Function | Backend |
|------|----------|---------|
| `code_execute` | Execute code | Docker/local |

### Orchestration Tools (4)
| Tool | Function | Backend |
|------|----------|---------|
| `workflow_create` | Create workflow | Workflow engine |
| `workflow_execute` | Execute workflow | Workflow engine |
| `workflow_status` | Get workflow status | Workflow engine |
| `workflow_cancel` | Cancel workflow | Workflow engine |

### Memory Tools (3)
| Tool | Function | Backend |
|------|----------|---------|
| `memory_store` | Store memory | Memory store/Redis |
| `memory_retrieve` | Retrieve memory | Memory store/Redis |
| `memory_search` | Search memory | Memory store/Redis |

### Scheduling Tools (2)
| Tool | Function | Backend |
|------|----------|---------|
| `schedule_create` | Create scheduled task | APScheduler/Celery |
| `schedule_cancel` | Cancel scheduled task | APScheduler/Celery |

### Home Assistant Tools (4)
| Tool | Function | Backend |
|------|----------|---------|
| `ha_get_state` | Get entity state | Home Assistant API |
| `ha_set_state` | Set entity state | Home Assistant API |
| `ha_call_service` | Call service | Home Assistant API |
| `ha_get_history` | Get entity history | Home Assistant API |

### RL Training Tools (10)
| Tool | Function | Backend |
|------|----------|---------|
| `rl_train_model` | Train RL model | Stable-Baselines3 |
| `rl_evaluate_model` | Evaluate RL model | Stable-Baselines3 |
| `rl_predict` | Get RL prediction | Stable-Baselines3 |
| `rl_hyperparameter_tune` | Tune hyperparameters | Optuna |
| `rl_save_model` | Save RL model | Pickle/PyTorch |
| `rl_load_model` | Load RL model | Pickle/PyTorch |
| `rl_get_policy` | Get model policy | Stable-Baselines3 |
| `rl_compare_models` | Compare models | Stable-Baselines3 |
| `rl_export_model` | Export model | ONNX/TensorFlow |
| `rl_monitor_training` | Monitor training | Training framework |

### MCP Dynamic Tools
Dynamically loaded from MCP servers:
- Filesystem MCP (fs_*)
- GitHub MCP (github_*)
- Slack MCP (slack_*)
- Custom MCPs

---

## Tools by Backend

### Playwright (10)
All browser tools use Playwright for browser automation

### HTTP Client (1)
- `web_fetch`

### BeautifulSoup/lxml (1)
- `web_parse`

### subprocess/Docker/SSH (2)
- `terminal_execute`
- `terminal_stream`

### File System (4)
- `file_read`
- `file_write`
- `file_delete`
- `file_list`

### Claude/GPT-4V (1)
- `vision_analyze_image`

### FFmpeg (2)
- `media_convert`
- `media_extract_audio`

### Docker/Local (1)
- `code_execute`

### Workflow Engine (4)
- `workflow_create`
- `workflow_execute`
- `workflow_status`
- `workflow_cancel`

### Memory Store/Redis (3)
- `memory_store`
- `memory_retrieve`
- `memory_search`

### APScheduler/Celery (2)
- `schedule_create`
- `schedule_cancel`

### Home Assistant API (4)
- `ha_get_state`
- `ha_set_state`
- `ha_call_service`
- `ha_get_history`

### Stable-Baselines3/Optuna (10)
- `rl_train_model`
- `rl_evaluate_model`
- `rl_predict`
- `rl_hyperparameter_tune`
- `rl_save_model`
- `rl_load_model`
- `rl_get_policy`
- `rl_compare_models`
- `rl_export_model`
- `rl_monitor_training`

---

## Tools by Use Case

### Web Automation
- `browser_navigate`
- `browser_click`
- `browser_fill`
- `browser_screenshot`
- `browser_get_text`
- `browser_get_html`
- `browser_wait_for_selector`
- `web_fetch`
- `web_parse`

### Data Processing
- `file_read`
- `file_write`
- `file_list`
- `code_execute`
- `web_parse`

### System Administration
- `terminal_execute`
- `terminal_stream`
- `file_delete`
- `schedule_create`

### Automation & Workflows
- `workflow_create`
- `workflow_execute`
- `workflow_status`
- `workflow_cancel`
- `schedule_create`
- `schedule_cancel`

### Smart Home
- `ha_get_state`
- `ha_set_state`
- `ha_call_service`
- `ha_get_history`

### Machine Learning
- `rl_train_model`
- `rl_evaluate_model`
- `rl_predict`
- `rl_hyperparameter_tune`
- `rl_save_model`
- `rl_load_model`
- `rl_get_policy`
- `rl_compare_models`
- `rl_export_model`
- `rl_monitor_training`

### Media Processing
- `vision_analyze_image`
- `media_convert`
- `media_extract_audio`

### Memory & Context
- `memory_store`
- `memory_retrieve`
- `memory_search`

---

## Configuration Quick Reference

### Enable/Disable Tools

```yaml
tools:
  enabled_categories:
    - web
    - code
    - system
    - data
  
  disabled_tools:
    - system_reboot
    - file_delete
```

### Tool Timeouts

```yaml
tools:
  timeout: 30  # seconds
  max_concurrent: 5
```

### Dangerous Operations

```yaml
approvals:
  enabled: true
  dangerous_operations:
    - file_delete
    - system_reboot
    - network_change
```

### Terminal Backend

```yaml
terminal:
  backend: local  # or: docker, ssh, modal, daytona
  timeout: 30
```

### Memory Configuration

```yaml
memory:
  enabled: true
  type: hybrid  # short_term, long_term, or hybrid
  backend: memory  # or: redis, database
```

### Home Assistant

```yaml
home_assistant:
  enabled: false
  url: "http://localhost:8123"
  token: "YOUR_HA_TOKEN"
```

### RL Training

```yaml
rl_training:
  backend: stable_baselines3
  gpu_enabled: true
  max_training_time_hours: 24
```

---

## Tool Parameters Reference

### Common Parameters

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `timeout` | integer | Timeout in seconds | 30 |
| `retry` | boolean | Retry on failure | true |
| `async` | boolean | Execute asynchronously | false |
| `tags` | array | Tool tags | [] |

### Return Format

All tools return:
```json
{
  "success": true/false,
  "data": {...},
  "error": null/error_message,
  "execution_time_ms": 1234
}
```

---

## Tool Approval Requirements

These tools require approval:
- `file_delete`
- `system_reboot`
- `network_change`
- `database_drop`

Configure approval:
```yaml
approvals:
  enabled: true
  mode: manual  # or: auto, hybrid
  timeout: 300  # seconds
  channels:
    - telegram
    - email
```

---

## Tool Composition

### Toolsets

Predefined toolsets:
- `web_scraper` - Web scraping tools
- `code_analyzer` - Code analysis tools
- `data_pipeline` - Data processing tools
- `automation` - Automation tools

Use toolset:
```bash
hermes run --toolset web_scraper "task"
```

### Workflows

Compose tools in workflows:
```yaml
workflow:
  steps:
    - tool: web_fetch
      params: {...}
    - tool: web_parse
      params: {...}
    - tool: code_execute
      params: {...}
```

---

## Tool Discovery

List all tools:
```bash
hermes tools list
```

Get tool info:
```bash
hermes tools info <tool_name>
```

List disabled tools:
```bash
hermes tools list --disabled
```

List MCP tools:
```bash
hermes tools list --mcp
```

---

## Tool Monitoring

View tool metrics:
```bash
hermes metrics tools
```

View tool logs:
```bash
hermes logs tools --level DEBUG
```

View tool performance:
```bash
hermes tools performance --sort time
```

---

## References

- **Full Documentation**: [TOOLS_REFERENCE.md](./TOOLS_REFERENCE.md)
- **Configuration**: [CONFIG_REFERENCE.md](../configuration/CONFIG_REFERENCE.md)
- **Workflows**: [WORKFLOW_GUIDE.md](../guides/WORKFLOW_GUIDE.md)

---

**Last Updated**: April 18, 2026
**Total Tools**: 40+
**Categories**: 12
