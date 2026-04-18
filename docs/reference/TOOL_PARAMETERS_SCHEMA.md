# Hermes Agent - Tool Parameters Schema Reference

Complete JSON Schema definitions for all tool parameters.

---

## Browser Tools Schemas

### browser_navigate

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "url": {
      "type": "string",
      "description": "Full URL to navigate to",
      "pattern": "^https?://"
    },
    "timeout": {
      "type": "integer",
      "description": "Navigation timeout in seconds",
      "default": 30,
      "minimum": 1,
      "maximum": 300
    },
    "wait_for": {
      "type": "string",
      "description": "CSS selector or XPath to wait for"
    },
    "headers": {
      "type": "object",
      "description": "Custom HTTP headers",
      "additionalProperties": {"type": "string"}
    }
  },
  "required": ["url"],
  "additionalProperties": false
}
```

### browser_click

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "selector": {
      "type": "string",
      "description": "CSS selector or XPath"
    },
    "button": {
      "type": "string",
      "enum": ["left", "right", "middle"],
      "default": "left"
    },
    "click_count": {
      "type": "integer",
      "default": 1,
      "minimum": 1
    },
    "delay_ms": {
      "type": "integer",
      "default": 0,
      "minimum": 0
    },
    "timeout": {
      "type": "integer",
      "default": 10,
      "minimum": 1,
      "maximum": 300
    }
  },
  "required": ["selector"],
  "additionalProperties": false
}
```

### browser_fill

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "selector": {
      "type": "string",
      "description": "CSS selector or XPath of input field"
    },
    "text": {
      "type": "string",
      "description": "Text to fill"
    },
    "clear": {
      "type": "boolean",
      "default": true
    },
    "delay_ms": {
      "type": "integer",
      "default": 0,
      "minimum": 0
    },
    "timeout": {
      "type": "integer",
      "default": 10,
      "minimum": 1,
      "maximum": 300
    }
  },
  "required": ["selector", "text"],
  "additionalProperties": false
}
```

### browser_screenshot

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "File path to save screenshot"
    },
    "full_page": {
      "type": "boolean",
      "default": false
    },
    "selector": {
      "type": "string",
      "description": "CSS selector for specific element"
    },
    "quality": {
      "type": "integer",
      "default": 90,
      "minimum": 1,
      "maximum": 100
    },
    "omit_background": {
      "type": "boolean",
      "default": false
    }
  },
  "required": ["path"],
  "additionalProperties": false
}
```

### browser_get_text

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "selector": {
      "type": "string",
      "description": "CSS selector (empty = entire page)"
    },
    "include_hidden": {
      "type": "boolean",
      "default": false
    },
    "trim": {
      "type": "boolean",
      "default": true
    },
    "timeout": {
      "type": "integer",
      "default": 10,
      "minimum": 1,
      "maximum": 300
    }
  },
  "additionalProperties": false
}
```

### browser_get_html

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "selector": {
      "type": "string",
      "description": "CSS selector (empty = entire page)"
    },
    "outer_html": {
      "type": "boolean",
      "default": true
    },
    "pretty": {
      "type": "boolean",
      "default": true
    },
    "timeout": {
      "type": "integer",
      "default": 10,
      "minimum": 1,
      "maximum": 300
    }
  },
  "additionalProperties": false
}
```

### browser_wait_for_selector

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "selector": {
      "type": "string",
      "description": "CSS selector or XPath"
    },
    "timeout": {
      "type": "integer",
      "default": 30,
      "minimum": 1,
      "maximum": 300
    },
    "state": {
      "type": "string",
      "enum": ["attached", "detached", "visible", "hidden"],
      "default": "visible"
    }
  },
  "required": ["selector"],
  "additionalProperties": false
}
```

### browser_get_cookies

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "url": {
      "type": "string",
      "description": "URL to get cookies for"
    },
    "name": {
      "type": "string",
      "description": "Specific cookie name"
    }
  },
  "additionalProperties": false
}
```

### browser_set_cookies

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "cookies": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "value": {"type": "string"},
          "domain": {"type": "string"},
          "path": {"type": "string", "default": "/"},
          "expires": {"type": "integer"},
          "http_only": {"type": "boolean"},
          "secure": {"type": "boolean"},
          "same_site": {
            "type": "string",
            "enum": ["Strict", "Lax", "None"]
          }
        },
        "required": ["name", "value"]
      }
    }
  },
  "required": ["cookies"],
  "additionalProperties": false
}
```

### browser_execute_script

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "script": {
      "type": "string",
      "description": "JavaScript code to execute"
    },
    "args": {
      "type": "array",
      "description": "Arguments to pass to script"
    },
    "timeout": {
      "type": "integer",
      "default": 30,
      "minimum": 1,
      "maximum": 300
    }
  },
  "required": ["script"],
  "additionalProperties": false
}
```

---

## Web Tools Schemas

### web_fetch

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "url": {
      "type": "string",
      "pattern": "^https?://"
    },
    "method": {
      "type": "string",
      "enum": ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"],
      "default": "GET"
    },
    "headers": {
      "type": "object",
      "additionalProperties": {"type": "string"}
    },
    "body": {
      "type": "string"
    },
    "timeout": {
      "type": "integer",
      "default": 30,
      "minimum": 1,
      "maximum": 300
    },
    "follow_redirects": {
      "type": "boolean",
      "default": true
    },
    "verify_ssl": {
      "type": "boolean",
      "default": true
    }
  },
  "required": ["url"],
  "additionalProperties": false
}
```

### web_parse

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "content": {
      "type": "string",
      "description": "HTML or JSON content"
    },
    "format": {
      "type": "string",
      "enum": ["html", "json", "xml", "csv"]
    },
    "selectors": {
      "type": "object",
      "description": "CSS selectors or JSON paths",
      "additionalProperties": {"type": "string"}
    },
    "extract_all": {
      "type": "boolean",
      "default": false
    }
  },
  "required": ["content", "format"],
  "additionalProperties": false
}
```

---

## Terminal Tools Schemas

### terminal_execute

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "command": {
      "type": "string",
      "description": "Shell command to execute"
    },
    "cwd": {
      "type": "string",
      "description": "Working directory"
    },
    "timeout": {
      "type": "integer",
      "default": 30,
      "minimum": 1,
      "maximum": 300
    },
    "shell": {
      "type": "string",
      "enum": ["bash", "sh", "zsh", "fish"],
      "default": "bash"
    },
    "env": {
      "type": "object",
      "description": "Environment variables",
      "additionalProperties": {"type": "string"}
    },
    "capture_output": {
      "type": "boolean",
      "default": true
    }
  },
  "required": ["command"],
  "additionalProperties": false
}
```

### terminal_stream

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "command": {
      "type": "string"
    },
    "cwd": {
      "type": "string"
    },
    "timeout": {
      "type": "integer",
      "default": 30,
      "minimum": 1,
      "maximum": 300
    },
    "chunk_size": {
      "type": "integer",
      "default": 1024,
      "minimum": 256,
      "maximum": 65536
    }
  },
  "required": ["command"],
  "additionalProperties": false
}
```

---

## File Tools Schemas

### file_read

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "File path to read"
    },
    "encoding": {
      "type": "string",
      "enum": ["utf-8", "ascii", "latin-1", "binary"],
      "default": "utf-8"
    },
    "offset": {
      "type": "integer",
      "default": 0,
      "minimum": 0
    },
    "limit": {
      "type": "integer",
      "minimum": 1
    }
  },
  "required": ["path"],
  "additionalProperties": false
}
```

### file_write

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "path": {
      "type": "string"
    },
    "content": {
      "type": "string"
    },
    "mode": {
      "type": "string",
      "enum": ["write", "append"],
      "default": "write"
    },
    "encoding": {
      "type": "string",
      "enum": ["utf-8", "ascii", "latin-1"],
      "default": "utf-8"
    },
    "create_dirs": {
      "type": "boolean",
      "default": true
    }
  },
  "required": ["path", "content"],
  "additionalProperties": false
}
```

### file_delete

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "path": {
      "type": "string"
    },
    "recursive": {
      "type": "boolean",
      "default": false
    },
    "force": {
      "type": "boolean",
      "default": false
    }
  },
  "required": ["path"],
  "additionalProperties": false
}
```

### file_list

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "path": {
      "type": "string"
    },
    "recursive": {
      "type": "boolean",
      "default": false
    },
    "pattern": {
      "type": "string",
      "description": "Glob pattern"
    },
    "include_hidden": {
      "type": "boolean",
      "default": false
    }
  },
  "required": ["path"],
  "additionalProperties": false
}
```

---

## Vision & Media Tools Schemas

### vision_analyze_image

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "image_path": {
      "type": "string",
      "description": "Path or URL to image"
    },
    "query": {
      "type": "string",
      "description": "Analysis query"
    },
    "detail_level": {
      "type": "string",
      "enum": ["low", "medium", "high"],
      "default": "medium"
    },
    "format": {
      "type": "string",
      "enum": ["text", "json", "structured"],
      "default": "text"
    }
  },
  "required": ["image_path", "query"],
  "additionalProperties": false
}
```

### media_convert

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "input_path": {
      "type": "string"
    },
    "output_path": {
      "type": "string"
    },
    "format": {
      "type": "string"
    },
    "codec": {
      "type": "string"
    },
    "quality": {
      "type": "integer",
      "default": 85,
      "minimum": 1,
      "maximum": 100
    },
    "bitrate": {
      "type": "string"
    }
  },
  "required": ["input_path", "output_path"],
  "additionalProperties": false
}
```

### media_extract_audio

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "input_path": {
      "type": "string"
    },
    "output_path": {
      "type": "string"
    },
    "format": {
      "type": "string",
      "enum": ["mp3", "wav", "aac", "flac", "ogg"],
      "default": "mp3"
    },
    "bitrate": {
      "type": "string",
      "default": "128k"
    }
  },
  "required": ["input_path", "output_path"],
  "additionalProperties": false
}
```

---

## Code Execution Schema

### code_execute

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "code": {
      "type": "string"
    },
    "language": {
      "type": "string",
      "enum": ["python", "javascript", "go", "rust", "java", "csharp", "ruby", "php"]
    },
    "timeout": {
      "type": "integer",
      "default": 30,
      "minimum": 1,
      "maximum": 300
    },
    "dependencies": {
      "type": "array",
      "items": {"type": "string"}
    },
    "input": {
      "type": "string"
    }
  },
  "required": ["code", "language"],
  "additionalProperties": false
}
```

---

## Orchestration Tools Schemas

### workflow_create

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "name": {
      "type": "string"
    },
    "steps": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "tool": {"type": "string"},
          "params": {"type": "object"},
          "on_success": {"type": "string"},
          "on_failure": {"type": "string"}
        },
        "required": ["name", "tool", "params"]
      }
    },
    "triggers": {
      "type": "array"
    },
    "schedule": {
      "type": "string",
      "description": "Cron expression"
    }
  },
  "required": ["name", "steps"],
  "additionalProperties": false
}
```

### workflow_execute

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "workflow_id": {
      "type": "string"
    },
    "variables": {
      "type": "object"
    },
    "async": {
      "type": "boolean",
      "default": false
    }
  },
  "required": ["workflow_id"],
  "additionalProperties": false
}
```

### workflow_status

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "execution_id": {
      "type": "string"
    }
  },
  "required": ["execution_id"],
  "additionalProperties": false
}
```

### workflow_cancel

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "execution_id": {
      "type": "string"
    },
    "force": {
      "type": "boolean",
      "default": false
    }
  },
  "required": ["execution_id"],
  "additionalProperties": false
}
```

---

## Memory Tools Schemas

### memory_store

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "key": {
      "type": "string"
    },
    "value": {
      "type": "string"
    },
    "type": {
      "type": "string",
      "enum": ["short_term", "long_term", "episodic"],
      "default": "short_term"
    },
    "ttl": {
      "type": "integer",
      "minimum": 1
    },
    "tags": {
      "type": "array",
      "items": {"type": "string"}
    }
  },
  "required": ["key", "value"],
  "additionalProperties": false
}
```

### memory_retrieve

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "key": {
      "type": "string"
    },
    "type": {
      "type": "string",
      "enum": ["short_term", "long_term", "episodic", "all"],
      "default": "all"
    },
    "fuzzy": {
      "type": "boolean",
      "default": false
    }
  },
  "required": ["key"],
  "additionalProperties": false
}
```

### memory_search

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "query": {
      "type": "string"
    },
    "tags": {
      "type": "array",
      "items": {"type": "string"}
    },
    "type": {
      "type": "string",
      "enum": ["short_term", "long_term", "episodic", "all"],
      "default": "all"
    },
    "limit": {
      "type": "integer",
      "default": 10,
      "minimum": 1
    }
  },
  "required": ["query"],
  "additionalProperties": false
}
```

---

## Scheduling Tools Schemas

### schedule_create

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "name": {
      "type": "string"
    },
    "tool": {
      "type": "string"
    },
    "params": {
      "type": "object"
    },
    "schedule": {
      "type": "string",
      "description": "Cron expression or interval"
    },
    "timezone": {
      "type": "string",
      "default": "UTC"
    },
    "max_retries": {
      "type": "integer",
      "default": 3,
      "minimum": 0
    }
  },
  "required": ["name", "tool", "params", "schedule"],
  "additionalProperties": false
}
```

### schedule_cancel

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "task_id": {
      "type": "string"
    }
  },
  "required": ["task_id"],
  "additionalProperties": false
}
```

---

## Home Assistant Tools Schemas

### ha_get_state

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "entity_id": {
      "type": "string",
      "pattern": "^[a-z_]+\\.[a-z_]+$"
    }
  },
  "required": ["entity_id"],
  "additionalProperties": false
}
```

### ha_set_state

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "entity_id": {
      "type": "string"
    },
    "state": {
      "type": "string"
    },
    "attributes": {
      "type": "object"
    }
  },
  "required": ["entity_id", "state"],
  "additionalProperties": false
}
```

### ha_call_service

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "domain": {
      "type": "string"
    },
    "service": {
      "type": "string"
    },
    "data": {
      "type": "object"
    }
  },
  "required": ["domain", "service"],
  "additionalProperties": false
}
```

### ha_get_history

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "entity_id": {
      "type": "string"
    },
    "hours": {
      "type": "integer",
      "default": 24,
      "minimum": 1
    }
  },
  "required": ["entity_id"],
  "additionalProperties": false
}
```

---

## RL Training Tools Schemas

### rl_train_model

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "environment": {
      "type": "string"
    },
    "algorithm": {
      "type": "string",
      "enum": ["DQN", "PPO", "A3C", "DDPG", "TD3", "SAC"]
    },
    "episodes": {
      "type": "integer",
      "default": 1000,
      "minimum": 1
    },
    "learning_rate": {
      "type": "number",
      "default": 0.001,
      "minimum": 0.00001,
      "maximum": 0.1
    },
    "batch_size": {
      "type": "integer",
      "default": 32,
      "minimum": 1
    },
    "gamma": {
      "type": "number",
      "default": 0.99,
      "minimum": 0,
      "maximum": 1
    }
  },
  "required": ["environment", "algorithm"],
  "additionalProperties": false
}
```

### rl_evaluate_model

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "model_path": {
      "type": "string"
    },
    "environment": {
      "type": "string"
    },
    "episodes": {
      "type": "integer",
      "default": 100,
      "minimum": 1
    },
    "render": {
      "type": "boolean",
      "default": false
    }
  },
  "required": ["model_path", "environment"],
  "additionalProperties": false
}
```

### rl_predict

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "model_path": {
      "type": "string"
    },
    "observation": {
      "type": "array"
    },
    "deterministic": {
      "type": "boolean",
      "default": true
    }
  },
  "required": ["model_path", "observation"],
  "additionalProperties": false
}
```

### rl_hyperparameter_tune

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "environment": {
      "type": "string"
    },
    "algorithm": {
      "type": "string",
      "enum": ["DQN", "PPO", "A3C", "DDPG", "TD3", "SAC"]
    },
    "param_space": {
      "type": "object"
    },
    "n_trials": {
      "type": "integer",
      "default": 100,
      "minimum": 1
    },
    "n_jobs": {
      "type": "integer",
      "default": 1,
      "minimum": 1
    }
  },
  "required": ["environment", "algorithm", "param_space"],
  "additionalProperties": false
}
```

### rl_save_model

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "model": {
      "type": "object"
    },
    "path": {
      "type": "string"
    },
    "include_replay_buffer": {
      "type": "boolean",
      "default": false
    }
  },
  "required": ["model", "path"],
  "additionalProperties": false
}
```

### rl_load_model

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "path": {
      "type": "string"
    },
    "algorithm": {
      "type": "string",
      "enum": ["DQN", "PPO", "A3C", "DDPG", "TD3", "SAC"]
    }
  },
  "required": ["path", "algorithm"],
  "additionalProperties": false
}
```

### rl_get_policy

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "model_path": {
      "type": "string"
    }
  },
  "required": ["model_path"],
  "additionalProperties": false
}
```

### rl_compare_models

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "model_paths": {
      "type": "array",
      "items": {"type": "string"}
    },
    "environment": {
      "type": "string"
    },
    "episodes": {
      "type": "integer",
      "default": 100,
      "minimum": 1
    }
  },
  "required": ["model_paths", "environment"],
  "additionalProperties": false
}
```

### rl_export_model

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "model_path": {
      "type": "string"
    },
    "format": {
      "type": "string",
      "enum": ["onnx", "tensorflow", "pytorch", "tflite"]
    },
    "output_path": {
      "type": "string"
    }
  },
  "required": ["model_path", "format", "output_path"],
  "additionalProperties": false
}
```

### rl_monitor_training

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "training_id": {
      "type": "string"
    }
  },
  "required": ["training_id"],
  "additionalProperties": false
}
```

---

## Common Return Schema

All tools return this schema:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "success": {
      "type": "boolean",
      "description": "Whether tool execution succeeded"
    },
    "data": {
      "type": "object",
      "description": "Tool-specific return data"
    },
    "error": {
      "type": ["string", "null"],
      "description": "Error message if failed"
    },
    "execution_time_ms": {
      "type": "integer",
      "description": "Execution time in milliseconds"
    }
  },
  "required": ["success"],
  "additionalProperties": true
}
```

---

**Last Updated**: April 18, 2026
