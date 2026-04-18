# Hermes Agent - Complete Tools Reference

**Last Updated**: April 18, 2026

Comprehensive documentation of every tool available in Hermes Agent, including parameters, schemas, return formats, backends, and configuration options.

---

## Table of Contents

1. [Browser Tools](#browser-tools) (10 tools)
2. [Web Tools](#web-tools) (2 tools)
3. [Terminal Tools](#terminal-tools) (2 tools)
4. [File Tools](#file-tools) (4 tools)
5. [Vision & Media Tools](#vision--media-tools) (3 tools)
6. [Code Execution Tools](#code-execution-tools) (1 tool)
7. [Orchestration Tools](#orchestration-tools) (4 tools)
8. [Memory Tools](#memory-tools) (3 tools)
9. [Scheduling Tools](#scheduling-tools) (2 tools)
10. [Home Assistant Tools](#home-assistant-tools) (4 tools)
11. [RL Training Tools](#rl-training-tools) (10 tools)
12. [MCP Dynamic Tools](#mcp-dynamic-tools)
13. [Tool Configuration](#tool-configuration)
14. [Toolset Compositions](#toolset-compositions)

---

## Browser Tools

### 1. browser_navigate

Navigate to a URL in the browser.

**Function**: Open and load a webpage

**Parameters**:
```json
{
  "url": {
    "type": "string",
    "description": "Full URL to navigate to (must include protocol: http:// or https://)",
    "required": true,
    "example": "https://example.com"
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
    "description": "CSS selector or XPath to wait for before returning",
    "required": false,
    "example": ".content-loaded"
  },
  "headers": {
    "type": "object",
    "description": "Custom HTTP headers to send with request",
    "required": false,
    "example": {"User-Agent": "Custom Agent"}
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "url": "https://example.com",
  "status_code": 200,
  "title": "Example Domain",
  "content_length": 1256,
  "load_time_ms": 1234,
  "redirects": ["https://www.example.com"],
  "error": null
}
```

**Backend**: Playwright (Chromium)

**Configuration**:
```yaml
browser:
  backend: playwright
  headless: true
  timeout: 30
  viewport:
    width: 1920
    height: 1080
```

**Disable**: Add to `disabled_tools` in config.yaml

---

### 2. browser_click

Click an element on the page.

**Function**: Interact with clickable elements (buttons, links, etc.)

**Parameters**:
```json
{
  "selector": {
    "type": "string",
    "description": "CSS selector or XPath of element to click",
    "required": true,
    "example": "button.submit"
  },
  "button": {
    "type": "string",
    "enum": ["left", "right", "middle"],
    "description": "Mouse button to use",
    "default": "left"
  },
  "click_count": {
    "type": "integer",
    "description": "Number of times to click",
    "default": 1,
    "minimum": 1
  },
  "delay_ms": {
    "type": "integer",
    "description": "Delay between clicks in milliseconds",
    "default": 0
  },
  "timeout": {
    "type": "integer",
    "description": "Timeout in seconds",
    "default": 10
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "element_found": true,
  "element_text": "Submit",
  "element_visible": true,
  "element_enabled": true,
  "click_position": {"x": 100, "y": 50},
  "error": null
}
```

**Backend**: Playwright

---

### 3. browser_fill

Fill form fields with text.

**Function**: Input text into form fields

**Parameters**:
```json
{
  "selector": {
    "type": "string",
    "description": "CSS selector or XPath of input field",
    "required": true,
    "example": "input#email"
  },
  "text": {
    "type": "string",
    "description": "Text to fill into the field",
    "required": true
  },
  "clear": {
    "type": "boolean",
    "description": "Clear field before filling",
    "default": true
  },
  "delay_ms": {
    "type": "integer",
    "description": "Delay between keystrokes in milliseconds",
    "default": 0
  },
  "timeout": {
    "type": "integer",
    "description": "Timeout in seconds",
    "default": 10
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "field_found": true,
  "field_type": "text",
  "text_filled": "user@example.com",
  "field_value": "user@example.com",
  "error": null
}
```

**Backend**: Playwright

---

### 4. browser_screenshot

Take a screenshot of the current page.

**Function**: Capture visual state of webpage

**Parameters**:
```json
{
  "path": {
    "type": "string",
    "description": "File path to save screenshot (PNG format)",
    "required": true,
    "example": "/tmp/screenshot.png"
  },
  "full_page": {
    "type": "boolean",
    "description": "Capture full page or just viewport",
    "default": false
  },
  "selector": {
    "type": "string",
    "description": "CSS selector to capture specific element",
    "required": false
  },
  "quality": {
    "type": "integer",
    "description": "JPEG quality (1-100), PNG uses lossless",
    "default": 90,
    "minimum": 1,
    "maximum": 100
  },
  "omit_background": {
    "type": "boolean",
    "description": "Omit background color",
    "default": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "path": "/tmp/screenshot.png",
  "file_size_bytes": 45678,
  "width": 1920,
  "height": 1080,
  "format": "png",
  "error": null
}
```

**Backend**: Playwright

---

### 5. browser_get_text

Extract text content from page or element.

**Function**: Read text from webpage

**Parameters**:
```json
{
  "selector": {
    "type": "string",
    "description": "CSS selector or XPath (empty = entire page)",
    "required": false
  },
  "include_hidden": {
    "type": "boolean",
    "description": "Include hidden elements",
    "default": false
  },
  "trim": {
    "type": "boolean",
    "description": "Trim whitespace",
    "default": true
  },
  "timeout": {
    "type": "integer",
    "description": "Timeout in seconds",
    "default": 10
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "text": "Welcome to Example Domain\n\nThis domain is for use in examples...",
  "element_count": 1,
  "character_count": 1234,
  "error": null
}
```

**Backend**: Playwright

---

### 6. browser_get_html

Extract HTML content from page or element.

**Function**: Get raw HTML markup

**Parameters**:
```json
{
  "selector": {
    "type": "string",
    "description": "CSS selector or XPath (empty = entire page)",
    "required": false
  },
  "outer_html": {
    "type": "boolean",
    "description": "Include element's own tags",
    "default": true
  },
  "pretty": {
    "type": "boolean",
    "description": "Format HTML with indentation",
    "default": true
  },
  "timeout": {
    "type": "integer",
    "description": "Timeout in seconds",
    "default": 10
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "html": "<!DOCTYPE html>\n<html>\n<head>...",
  "element_count": 1,
  "byte_size": 5678,
  "error": null
}
```

**Backend**: Playwright

---

### 7. browser_wait_for_selector

Wait for an element to appear on page.

**Function**: Synchronize with dynamic content

**Parameters**:
```json
{
  "selector": {
    "type": "string",
    "description": "CSS selector or XPath to wait for",
    "required": true,
    "example": ".loading-complete"
  },
  "timeout": {
    "type": "integer",
    "description": "Maximum wait time in seconds",
    "default": 30,
    "minimum": 1,
    "maximum": 300
  },
  "state": {
    "type": "string",
    "enum": ["attached", "detached", "visible", "hidden"],
    "description": "State to wait for",
    "default": "visible"
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "selector": ".loading-complete",
  "state": "visible",
  "wait_time_ms": 1234,
  "element_found": true,
  "error": null
}
```

**Backend**: Playwright

---

### 8. browser_get_cookies

Retrieve cookies from current page.

**Function**: Access browser cookies

**Parameters**:
```json
{
  "url": {
    "type": "string",
    "description": "URL to get cookies for (optional, uses current if not specified)",
    "required": false
  },
  "name": {
    "type": "string",
    "description": "Specific cookie name to retrieve",
    "required": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "cookies": [
    {
      "name": "session_id",
      "value": "abc123def456",
      "domain": ".example.com",
      "path": "/",
      "expires": 1735689600,
      "http_only": true,
      "secure": true,
      "same_site": "Strict"
    }
  ],
  "count": 1,
  "error": null
}
```

**Backend**: Playwright

---

### 9. browser_set_cookies

Set cookies in browser.

**Function**: Manage browser cookies

**Parameters**:
```json
{
  "cookies": {
    "type": "array",
    "description": "Array of cookie objects to set",
    "required": true,
    "items": {
      "type": "object",
      "properties": {
        "name": {"type": "string"},
        "value": {"type": "string"},
        "domain": {"type": "string"},
        "path": {"type": "string", "default": "/"},
        "expires": {"type": "integer", "description": "Unix timestamp"},
        "http_only": {"type": "boolean"},
        "secure": {"type": "boolean"},
        "same_site": {"type": "string", "enum": ["Strict", "Lax", "None"]}
      }
    }
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "cookies_set": 1,
  "cookies": [
    {
      "name": "session_id",
      "value": "abc123def456",
      "domain": ".example.com"
    }
  ],
  "error": null
}
```

**Backend**: Playwright

---

### 10. browser_execute_script

Execute JavaScript in browser context.

**Function**: Run custom JavaScript code

**Parameters**:
```json
{
  "script": {
    "type": "string",
    "description": "JavaScript code to execute",
    "required": true,
    "example": "return document.title;"
  },
  "args": {
    "type": "array",
    "description": "Arguments to pass to script function",
    "required": false,
    "example": ["arg1", "arg2"]
  },
  "timeout": {
    "type": "integer",
    "description": "Timeout in seconds",
    "default": 30
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "result": "Example Domain",
  "type": "string",
  "error": null
}
```

**Backend**: Playwright

---

## Web Tools

### 1. web_fetch

Fetch content from a URL.

**Function**: Download and parse web content

**Parameters**:
```json
{
  "url": {
    "type": "string",
    "description": "URL to fetch",
    "required": true
  },
  "method": {
    "type": "string",
    "enum": ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"],
    "description": "HTTP method",
    "default": "GET"
  },
  "headers": {
    "type": "object",
    "description": "Custom HTTP headers",
    "required": false
  },
  "body": {
    "type": "string",
    "description": "Request body (for POST/PUT/PATCH)",
    "required": false
  },
  "timeout": {
    "type": "integer",
    "description": "Timeout in seconds",
    "default": 30
  },
  "follow_redirects": {
    "type": "boolean",
    "description": "Follow HTTP redirects",
    "default": true
  },
  "verify_ssl": {
    "type": "boolean",
    "description": "Verify SSL certificates",
    "default": true
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "url": "https://example.com",
  "status_code": 200,
  "headers": {
    "content-type": "text/html; charset=utf-8",
    "content-length": "1234"
  },
  "body": "<!DOCTYPE html>...",
  "body_size_bytes": 1234,
  "response_time_ms": 456,
  "error": null
}
```

**Backend**: HTTP client (requests/httpx)

**Configuration**:
```yaml
web:
  timeout: 30
  verify_ssl: true
  follow_redirects: true
  max_redirects: 5
  user_agent: "Hermes-Agent/1.0"
```

---

### 2. web_parse

Parse and extract structured data from HTML/JSON.

**Function**: Extract data from web content

**Parameters**:
```json
{
  "content": {
    "type": "string",
    "description": "HTML or JSON content to parse",
    "required": true
  },
  "format": {
    "type": "string",
    "enum": ["html", "json", "xml", "csv"],
    "description": "Content format",
    "required": true
  },
  "selectors": {
    "type": "object",
    "description": "CSS selectors or JSON paths to extract",
    "required": false,
    "example": {"title": "h1", "description": ".desc"}
  },
  "extract_all": {
    "type": "boolean",
    "description": "Extract all matching elements",
    "default": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "format": "html",
  "data": {
    "title": "Example Domain",
    "description": "This domain is for use in examples..."
  },
  "elements_found": 2,
  "error": null
}
```

**Backend**: BeautifulSoup, lxml, json

---

## Terminal Tools

### 1. terminal_execute

Execute shell commands.

**Function**: Run system commands

**Parameters**:
```json
{
  "command": {
    "type": "string",
    "description": "Shell command to execute",
    "required": true,
    "example": "ls -la /tmp"
  },
  "cwd": {
    "type": "string",
    "description": "Working directory",
    "required": false,
    "default": "~"
  },
  "timeout": {
    "type": "integer",
    "description": "Timeout in seconds",
    "default": 30,
    "maximum": 300
  },
  "shell": {
    "type": "string",
    "enum": ["bash", "sh", "zsh", "fish"],
    "description": "Shell to use",
    "default": "bash"
  },
  "env": {
    "type": "object",
    "description": "Environment variables",
    "required": false
  },
  "capture_output": {
    "type": "boolean",
    "description": "Capture stdout/stderr",
    "default": true
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "command": "ls -la /tmp",
  "exit_code": 0,
  "stdout": "total 48\ndrwxrwxrwt 10 root root...",
  "stderr": "",
  "execution_time_ms": 123,
  "error": null
}
```

**Backend**: subprocess, Docker, SSH, Modal, Daytona, Singularity

**Configuration**:
```yaml
terminal:
  backend: local  # or: docker, ssh, modal, daytona, singularity
  timeout: 30
  max_concurrent: 5
  
  # Backend-specific configs
  docker:
    image: ubuntu:22.04
    container_name: hermes-terminal
    volumes:
      - /tmp:/tmp
  
  ssh:
    host: localhost
    port: 22
    username: user
    key_path: ~/.ssh/id_rsa
  
  modal:
    token_path: ~/.modal/token
    workspace: default
  
  daytona:
    api_url: http://localhost:3000
    api_key: your-api-key
  
  singularity:
    image_path: /path/to/image.sif
    bind_paths:
      - /tmp:/tmp
```

---

### 2. terminal_stream

Stream command output in real-time.

**Function**: Execute command with streaming output

**Parameters**:
```json
{
  "command": {
    "type": "string",
    "description": "Shell command to execute",
    "required": true
  },
  "cwd": {
    "type": "string",
    "description": "Working directory",
    "required": false
  },
  "timeout": {
    "type": "integer",
    "description": "Timeout in seconds",
    "default": 30
  },
  "chunk_size": {
    "type": "integer",
    "description": "Output chunk size in bytes",
    "default": 1024
  }
}
```

**Return Format** (streaming):
```json
{
  "success": true,
  "command": "ls -la /tmp",
  "chunks": [
    {"type": "stdout", "data": "total 48\n", "timestamp": 1234567890},
    {"type": "stdout", "data": "drwxrwxrwt 10 root root...\n", "timestamp": 1234567891}
  ],
  "exit_code": 0,
  "total_output_bytes": 1234,
  "error": null
}
```

**Backend**: subprocess with streaming

---

## File Tools

### 1. file_read

Read file contents.

**Function**: Read text or binary files

**Parameters**:
```json
{
  "path": {
    "type": "string",
    "description": "File path to read",
    "required": true
  },
  "encoding": {
    "type": "string",
    "enum": ["utf-8", "ascii", "latin-1", "binary"],
    "description": "Text encoding",
    "default": "utf-8"
  },
  "offset": {
    "type": "integer",
    "description": "Byte offset to start reading",
    "default": 0
  },
  "limit": {
    "type": "integer",
    "description": "Maximum bytes to read",
    "required": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "path": "/path/to/file.txt",
  "content": "File contents here...",
  "size_bytes": 1234,
  "encoding": "utf-8",
  "lines": 42,
  "error": null
}
```

**Backend**: File system

---

### 2. file_write

Write content to file.

**Function**: Create or overwrite files

**Parameters**:
```json
{
  "path": {
    "type": "string",
    "description": "File path to write",
    "required": true
  },
  "content": {
    "type": "string",
    "description": "Content to write",
    "required": true
  },
  "mode": {
    "type": "string",
    "enum": ["write", "append"],
    "description": "Write mode",
    "default": "write"
  },
  "encoding": {
    "type": "string",
    "enum": ["utf-8", "ascii", "latin-1"],
    "description": "Text encoding",
    "default": "utf-8"
  },
  "create_dirs": {
    "type": "boolean",
    "description": "Create parent directories if needed",
    "default": true
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "path": "/path/to/file.txt",
  "bytes_written": 1234,
  "mode": "write",
  "created": true,
  "error": null
}
```

**Backend**: File system

---

### 3. file_delete

Delete files or directories.

**Function**: Remove files/folders

**Parameters**:
```json
{
  "path": {
    "type": "string",
    "description": "File or directory path to delete",
    "required": true
  },
  "recursive": {
    "type": "boolean",
    "description": "Recursively delete directories",
    "default": false
  },
  "force": {
    "type": "boolean",
    "description": "Force delete without confirmation",
    "default": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "path": "/path/to/file.txt",
  "deleted": true,
  "was_directory": false,
  "error": null
}
```

**Backend**: File system

**Requires Approval**: Yes (dangerous operation)

---

### 4. file_list

List directory contents.

**Function**: Browse directories

**Parameters**:
```json
{
  "path": {
    "type": "string",
    "description": "Directory path to list",
    "required": true
  },
  "recursive": {
    "type": "boolean",
    "description": "List recursively",
    "default": false
  },
  "pattern": {
    "type": "string",
    "description": "Glob pattern to filter files",
    "required": false,
    "example": "*.txt"
  },
  "include_hidden": {
    "type": "boolean",
    "description": "Include hidden files",
    "default": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "path": "/path/to/dir",
  "entries": [
    {
      "name": "file.txt",
      "type": "file",
      "size_bytes": 1234,
      "modified": 1234567890,
      "permissions": "644"
    },
    {
      "name": "subdir",
      "type": "directory",
      "size_bytes": 4096,
      "modified": 1234567890,
      "permissions": "755"
    }
  ],
  "total_entries": 2,
  "error": null
}
```

**Backend**: File system

---

## Vision & Media Tools

### 1. vision_analyze_image

Analyze images using vision AI.

**Function**: Extract information from images

**Parameters**:
```json
{
  "image_path": {
    "type": "string",
    "description": "Path to image file or URL",
    "required": true
  },
  "query": {
    "type": "string",
    "description": "What to analyze in the image",
    "required": true,
    "example": "What objects are in this image?"
  },
  "detail_level": {
    "type": "string",
    "enum": ["low", "medium", "high"],
    "description": "Analysis detail level",
    "default": "medium"
  },
  "format": {
    "type": "string",
    "enum": ["text", "json", "structured"],
    "description": "Response format",
    "default": "text"
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "image_path": "/path/to/image.jpg",
  "query": "What objects are in this image?",
  "analysis": "The image contains a cat, a dog, and a ball...",
  "objects_detected": ["cat", "dog", "ball"],
  "confidence_scores": {"cat": 0.95, "dog": 0.92, "ball": 0.88},
  "error": null
}
```

**Backend**: Claude Vision API, GPT-4V, or local vision model

**Configuration**:
```yaml
vision:
  backend: claude  # or: gpt4v, local
  model: claude-3-5-sonnet
  max_image_size_mb: 20
  supported_formats:
    - jpg
    - jpeg
    - png
    - gif
    - webp
```

---

### 2. media_convert

Convert media files between formats.

**Function**: Transcode audio/video

**Parameters**:
```json
{
  "input_path": {
    "type": "string",
    "description": "Input media file path",
    "required": true
  },
  "output_path": {
    "type": "string",
    "description": "Output file path",
    "required": true
  },
  "format": {
    "type": "string",
    "description": "Output format (auto-detected from extension if not specified)",
    "required": false,
    "example": "mp4"
  },
  "codec": {
    "type": "string",
    "description": "Video/audio codec",
    "required": false,
    "example": "h264"
  },
  "quality": {
    "type": "integer",
    "description": "Quality level (1-100)",
    "default": 85,
    "minimum": 1,
    "maximum": 100
  },
  "bitrate": {
    "type": "string",
    "description": "Bitrate (e.g., '128k', '5M')",
    "required": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "input_path": "/path/to/input.mov",
  "output_path": "/path/to/output.mp4",
  "format": "mp4",
  "codec": "h264",
  "duration_seconds": 120,
  "input_size_mb": 500,
  "output_size_mb": 150,
  "conversion_time_ms": 45000,
  "error": null
}
```

**Backend**: FFmpeg

**Configuration**:
```yaml
media:
  backend: ffmpeg
  ffmpeg_path: /usr/bin/ffmpeg
  max_file_size_mb: 1000
  timeout: 300
```

---

### 3. media_extract_audio

Extract audio from video files.

**Function**: Convert video to audio

**Parameters**:
```json
{
  "input_path": {
    "type": "string",
    "description": "Video file path",
    "required": true
  },
  "output_path": {
    "type": "string",
    "description": "Output audio file path",
    "required": true
  },
  "format": {
    "type": "string",
    "enum": ["mp3", "wav", "aac", "flac", "ogg"],
    "description": "Audio format",
    "default": "mp3"
  },
  "bitrate": {
    "type": "string",
    "description": "Audio bitrate",
    "default": "128k"
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "input_path": "/path/to/video.mp4",
  "output_path": "/path/to/audio.mp3",
  "format": "mp3",
  "duration_seconds": 120,
  "bitrate": "128k",
  "output_size_mb": 15,
  "error": null
}
```

**Backend**: FFmpeg

---

## Code Execution Tools

### 1. code_execute

Execute code in various languages.

**Function**: Run Python, JavaScript, Go, Rust, etc.

**Parameters**:
```json
{
  "code": {
    "type": "string",
    "description": "Code to execute",
    "required": true
  },
  "language": {
    "type": "string",
    "enum": ["python", "javascript", "go", "rust", "java", "csharp", "ruby", "php"],
    "description": "Programming language",
    "required": true
  },
  "timeout": {
    "type": "integer",
    "description": "Execution timeout in seconds",
    "default": 30,
    "maximum": 300
  },
  "dependencies": {
    "type": "array",
    "description": "Required packages/imports",
    "required": false,
    "example": ["numpy", "pandas"]
  },
  "input": {
    "type": "string",
    "description": "Standard input for the program",
    "required": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "language": "python",
  "exit_code": 0,
  "stdout": "Hello, World!\n",
  "stderr": "",
  "execution_time_ms": 234,
  "memory_used_mb": 45,
  "error": null
}
```

**Backend**: Docker containers, local interpreters

**Configuration**:
```yaml
code_execution:
  backend: docker  # or: local
  timeout: 30
  max_memory_mb: 512
  supported_languages:
    - python
    - javascript
    - go
    - rust
    - java
    - csharp
    - ruby
    - php
  docker:
    image: hermes-code-executor:latest
```

---

## Orchestration Tools

### 1. workflow_create

Create and manage workflows.

**Function**: Define multi-step automation

**Parameters**:
```json
{
  "name": {
    "type": "string",
    "description": "Workflow name",
    "required": true
  },
  "steps": {
    "type": "array",
    "description": "Workflow steps",
    "required": true,
    "items": {
      "type": "object",
      "properties": {
        "name": {"type": "string"},
        "tool": {"type": "string"},
        "params": {"type": "object"},
        "on_success": {"type": "string"},
        "on_failure": {"type": "string"}
      }
    }
  },
  "triggers": {
    "type": "array",
    "description": "Workflow triggers",
    "required": false
  },
  "schedule": {
    "type": "string",
    "description": "Cron schedule",
    "required": false,
    "example": "0 9 * * MON"
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "workflow_id": "wf_abc123def456",
  "name": "Daily Report",
  "steps": 3,
  "created": 1234567890,
  "status": "active",
  "error": null
}
```

**Backend**: Workflow engine

---

### 2. workflow_execute

Execute a workflow.

**Function**: Run workflow steps

**Parameters**:
```json
{
  "workflow_id": {
    "type": "string",
    "description": "Workflow ID to execute",
    "required": true
  },
  "variables": {
    "type": "object",
    "description": "Workflow variables",
    "required": false
  },
  "async": {
    "type": "boolean",
    "description": "Execute asynchronously",
    "default": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "workflow_id": "wf_abc123def456",
  "execution_id": "exec_xyz789",
  "status": "running",
  "steps_completed": 0,
  "total_steps": 3,
  "error": null
}
```

**Backend**: Workflow engine

---

### 3. workflow_status

Get workflow execution status.

**Function**: Monitor workflow progress

**Parameters**:
```json
{
  "execution_id": {
    "type": "string",
    "description": "Execution ID to check",
    "required": true
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "execution_id": "exec_xyz789",
  "workflow_id": "wf_abc123def456",
  "status": "completed",
  "steps_completed": 3,
  "total_steps": 3,
  "results": [
    {"step": "fetch_data", "status": "success", "output": {...}},
    {"step": "process_data", "status": "success", "output": {...}},
    {"step": "send_report", "status": "success", "output": {...}}
  ],
  "execution_time_ms": 5000,
  "error": null
}
```

**Backend**: Workflow engine

---

### 4. workflow_cancel

Cancel a running workflow.

**Function**: Stop workflow execution

**Parameters**:
```json
{
  "execution_id": {
    "type": "string",
    "description": "Execution ID to cancel",
    "required": true
  },
  "force": {
    "type": "boolean",
    "description": "Force cancel without cleanup",
    "default": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "execution_id": "exec_xyz789",
  "status": "cancelled",
  "steps_completed": 1,
  "total_steps": 3,
  "error": null
}
```

**Backend**: Workflow engine

---

## Memory Tools

### 1. memory_store

Store information in memory.

**Function**: Save facts and context

**Parameters**:
```json
{
  "key": {
    "type": "string",
    "description": "Memory key",
    "required": true
  },
  "value": {
    "type": "string",
    "description": "Memory value",
    "required": true
  },
  "type": {
    "type": "string",
    "enum": ["short_term", "long_term", "episodic"],
    "description": "Memory type",
    "default": "short_term"
  },
  "ttl": {
    "type": "integer",
    "description": "Time-to-live in seconds",
    "required": false
  },
  "tags": {
    "type": "array",
    "description": "Memory tags for organization",
    "required": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "key": "user_preference_theme",
  "value": "dark",
  "type": "long_term",
  "stored": true,
  "memory_id": "mem_abc123",
  "error": null
}
```

**Backend**: In-memory store, Redis, or persistent database

**Configuration**:
```yaml
memory:
  enabled: true
  type: hybrid  # short_term, long_term, or hybrid
  backend: memory  # or: redis, database
  max_entries: 1000
  compression:
    enabled: true
    threshold: 500
    ratio: 0.5
  
  redis:
    host: localhost
    port: 6379
    db: 0
```

---

### 2. memory_retrieve

Retrieve information from memory.

**Function**: Recall stored facts

**Parameters**:
```json
{
  "key": {
    "type": "string",
    "description": "Memory key to retrieve",
    "required": true
  },
  "type": {
    "type": "string",
    "enum": ["short_term", "long_term", "episodic", "all"],
    "description": "Memory type to search",
    "default": "all"
  },
  "fuzzy": {
    "type": "boolean",
    "description": "Use fuzzy matching",
    "default": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "key": "user_preference_theme",
  "value": "dark",
  "type": "long_term",
  "retrieved": true,
  "created": 1234567890,
  "accessed": 1234567999,
  "error": null
}
```

**Backend**: In-memory store, Redis, or persistent database

---

### 3. memory_search

Search memory by tags or content.

**Function**: Find related memories

**Parameters**:
```json
{
  "query": {
    "type": "string",
    "description": "Search query",
    "required": true
  },
  "tags": {
    "type": "array",
    "description": "Filter by tags",
    "required": false
  },
  "type": {
    "type": "string",
    "enum": ["short_term", "long_term", "episodic", "all"],
    "description": "Memory type to search",
    "default": "all"
  },
  "limit": {
    "type": "integer",
    "description": "Maximum results",
    "default": 10
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "query": "user preferences",
  "results": [
    {
      "key": "user_preference_theme",
      "value": "dark",
      "type": "long_term",
      "relevance_score": 0.95
    },
    {
      "key": "user_preference_language",
      "value": "en",
      "type": "long_term",
      "relevance_score": 0.87
    }
  ],
  "total_results": 2,
  "error": null
}
```

**Backend**: In-memory store, Redis, or persistent database

---

## Scheduling Tools

### 1. schedule_create

Create a scheduled task.

**Function**: Schedule recurring or one-time tasks

**Parameters**:
```json
{
  "name": {
    "type": "string",
    "description": "Task name",
    "required": true
  },
  "tool": {
    "type": "string",
    "description": "Tool to execute",
    "required": true
  },
  "params": {
    "type": "object",
    "description": "Tool parameters",
    "required": true
  },
  "schedule": {
    "type": "string",
    "description": "Cron expression or interval",
    "required": true,
    "example": "0 9 * * MON"
  },
  "timezone": {
    "type": "string",
    "description": "Timezone for schedule",
    "default": "UTC"
  },
  "max_retries": {
    "type": "integer",
    "description": "Maximum retry attempts",
    "default": 3
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "task_id": "task_abc123def456",
  "name": "Daily Report",
  "schedule": "0 9 * * MON",
  "next_run": 1234567890,
  "created": 1234567890,
  "status": "active",
  "error": null
}
```

**Backend**: APScheduler, Celery, or Kubernetes CronJob

**Configuration**:
```yaml
scheduling:
  backend: apscheduler  # or: celery, k8s
  timezone: UTC
  max_concurrent_tasks: 10
```

---

### 2. schedule_cancel

Cancel a scheduled task.

**Function**: Remove scheduled task

**Parameters**:
```json
{
  "task_id": {
    "type": "string",
    "description": "Task ID to cancel",
    "required": true
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "task_id": "task_abc123def456",
  "cancelled": true,
  "was_running": false,
  "error": null
}
```

**Backend**: APScheduler, Celery, or Kubernetes CronJob

---

## Home Assistant Tools

### 1. ha_get_state

Get Home Assistant entity state.

**Function**: Query smart home device status

**Parameters**:
```json
{
  "entity_id": {
    "type": "string",
    "description": "Home Assistant entity ID",
    "required": true,
    "example": "light.living_room"
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "entity_id": "light.living_room",
  "state": "on",
  "attributes": {
    "brightness": 255,
    "color_temp": 2700,
    "friendly_name": "Living Room Light"
  },
  "last_changed": 1234567890,
  "error": null
}
```

**Backend**: Home Assistant API

**Configuration**:
```yaml
home_assistant:
  enabled: false
  url: "http://localhost:8123"
  token: "YOUR_HA_TOKEN"
  verify_ssl: true
```

---

### 2. ha_set_state

Set Home Assistant entity state.

**Function**: Control smart home devices

**Parameters**:
```json
{
  "entity_id": {
    "type": "string",
    "description": "Home Assistant entity ID",
    "required": true
  },
  "state": {
    "type": "string",
    "description": "New state",
    "required": true,
    "example": "on"
  },
  "attributes": {
    "type": "object",
    "description": "State attributes",
    "required": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "entity_id": "light.living_room",
  "state": "on",
  "attributes": {
    "brightness": 200
  },
  "error": null
}
```

**Backend**: Home Assistant API

---

### 3. ha_call_service

Call Home Assistant service.

**Function**: Execute Home Assistant automation

**Parameters**:
```json
{
  "domain": {
    "type": "string",
    "description": "Service domain",
    "required": true,
    "example": "light"
  },
  "service": {
    "type": "string",
    "description": "Service name",
    "required": true,
    "example": "turn_on"
  },
  "data": {
    "type": "object",
    "description": "Service data",
    "required": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "domain": "light",
  "service": "turn_on",
  "data": {
    "entity_id": "light.living_room",
    "brightness": 200
  },
  "error": null
}
```

**Backend**: Home Assistant API

---

### 4. ha_get_history

Get Home Assistant entity history.

**Function**: Retrieve historical data

**Parameters**:
```json
{
  "entity_id": {
    "type": "string",
    "description": "Home Assistant entity ID",
    "required": true
  },
  "hours": {
    "type": "integer",
    "description": "Hours of history to retrieve",
    "default": 24
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "entity_id": "sensor.temperature",
  "history": [
    {
      "state": "22.5",
      "timestamp": 1234567890,
      "attributes": {"unit_of_measurement": "°C"}
    }
  ],
  "total_entries": 100,
  "error": null
}
```

**Backend**: Home Assistant API

---

## RL Training Tools

### 1. rl_train_model

Train a reinforcement learning model.

**Function**: Train RL agent

**Parameters**:
```json
{
  "environment": {
    "type": "string",
    "description": "Environment name or path",
    "required": true,
    "example": "CartPole-v1"
  },
  "algorithm": {
    "type": "string",
    "enum": ["DQN", "PPO", "A3C", "DDPG", "TD3", "SAC"],
    "description": "RL algorithm",
    "required": true
  },
  "episodes": {
    "type": "integer",
    "description": "Number of training episodes",
    "default": 1000
  },
  "learning_rate": {
    "type": "number",
    "description": "Learning rate",
    "default": 0.001
  },
  "batch_size": {
    "type": "integer",
    "description": "Batch size",
    "default": 32
  },
  "gamma": {
    "type": "number",
    "description": "Discount factor",
    "default": 0.99
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "training_id": "train_abc123",
  "environment": "CartPole-v1",
  "algorithm": "PPO",
  "episodes": 1000,
  "status": "completed",
  "total_reward": 195.5,
  "training_time_seconds": 3600,
  "model_path": "/models/cartpole_ppo.pkl",
  "error": null
}
```

**Backend**: Stable-Baselines3, Ray RLlib

**Configuration**:
```yaml
rl_training:
  backend: stable_baselines3  # or: ray_rllib
  max_training_time_hours: 24
  gpu_enabled: true
  models_dir: ~/.hermes/rl_models
```

---

### 2. rl_evaluate_model

Evaluate a trained RL model.

**Function**: Test RL agent performance

**Parameters**:
```json
{
  "model_path": {
    "type": "string",
    "description": "Path to trained model",
    "required": true
  },
  "environment": {
    "type": "string",
    "description": "Environment name",
    "required": true
  },
  "episodes": {
    "type": "integer",
    "description": "Number of evaluation episodes",
    "default": 100
  },
  "render": {
    "type": "boolean",
    "description": "Render environment",
    "default": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "model_path": "/models/cartpole_ppo.pkl",
  "environment": "CartPole-v1",
  "episodes": 100,
  "mean_reward": 195.5,
  "std_reward": 5.2,
  "min_reward": 180,
  "max_reward": 200,
  "evaluation_time_seconds": 120,
  "error": null
}
```

**Backend**: Stable-Baselines3, Ray RLlib

---

### 3. rl_predict

Make predictions with trained RL model.

**Function**: Get RL agent action

**Parameters**:
```json
{
  "model_path": {
    "type": "string",
    "description": "Path to trained model",
    "required": true
  },
  "observation": {
    "type": "array",
    "description": "Environment observation",
    "required": true
  },
  "deterministic": {
    "type": "boolean",
    "description": "Use deterministic policy",
    "default": true
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "model_path": "/models/cartpole_ppo.pkl",
  "action": 1,
  "action_probability": 0.95,
  "value_estimate": 195.5,
  "error": null
}
```

**Backend**: Stable-Baselines3, Ray RLlib

---

### 4. rl_hyperparameter_tune

Tune RL model hyperparameters.

**Function**: Optimize RL training parameters

**Parameters**:
```json
{
  "environment": {
    "type": "string",
    "description": "Environment name",
    "required": true
  },
  "algorithm": {
    "type": "string",
    "enum": ["DQN", "PPO", "A3C", "DDPG", "TD3", "SAC"],
    "description": "RL algorithm",
    "required": true
  },
  "param_space": {
    "type": "object",
    "description": "Hyperparameter search space",
    "required": true
  },
  "n_trials": {
    "type": "integer",
    "description": "Number of trials",
    "default": 100
  },
  "n_jobs": {
    "type": "integer",
    "description": "Parallel jobs",
    "default": 1
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "tuning_id": "tune_xyz789",
  "environment": "CartPole-v1",
  "algorithm": "PPO",
  "n_trials": 100,
  "best_params": {
    "learning_rate": 0.0005,
    "batch_size": 64,
    "gamma": 0.995
  },
  "best_reward": 198.5,
  "tuning_time_seconds": 7200,
  "error": null
}
```

**Backend**: Optuna with Stable-Baselines3

---

### 5. rl_save_model

Save trained RL model.

**Function**: Persist RL model

**Parameters**:
```json
{
  "model": {
    "type": "object",
    "description": "Trained model object",
    "required": true
  },
  "path": {
    "type": "string",
    "description": "Save path",
    "required": true
  },
  "include_replay_buffer": {
    "type": "boolean",
    "description": "Include replay buffer",
    "default": false
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "path": "/models/cartpole_ppo.pkl",
  "file_size_mb": 45,
  "saved": true,
  "error": null
}
```

**Backend**: Pickle, PyTorch, TensorFlow

---

### 6. rl_load_model

Load trained RL model.

**Function**: Restore RL model

**Parameters**:
```json
{
  "path": {
    "type": "string",
    "description": "Model file path",
    "required": true
  },
  "algorithm": {
    "type": "string",
    "enum": ["DQN", "PPO", "A3C", "DDPG", "TD3", "SAC"],
    "description": "RL algorithm",
    "required": true
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "path": "/models/cartpole_ppo.pkl",
  "algorithm": "PPO",
  "loaded": true,
  "model_info": {
    "policy": "MlpPolicy",
    "learning_rate": 0.0003
  },
  "error": null
}
```

**Backend**: Pickle, PyTorch, TensorFlow

---

### 7. rl_get_policy

Get RL model policy.

**Function**: Extract policy from model

**Parameters**:
```json
{
  "model_path": {
    "type": "string",
    "description": "Path to trained model",
    "required": true
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "model_path": "/models/cartpole_ppo.pkl",
  "policy_type": "MlpPolicy",
  "policy_architecture": {
    "layers": [64, 64],
    "activation": "relu"
  },
  "error": null
}
```

**Backend**: Stable-Baselines3

---

### 8. rl_compare_models

Compare multiple RL models.

**Function**: Benchmark RL agents

**Parameters**:
```json
{
  "model_paths": {
    "type": "array",
    "description": "Paths to models to compare",
    "required": true
  },
  "environment": {
    "type": "string",
    "description": "Environment name",
    "required": true
  },
  "episodes": {
    "type": "integer",
    "description": "Episodes per model",
    "default": 100
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "models": [
    {
      "path": "/models/cartpole_ppo.pkl",
      "mean_reward": 195.5,
      "std_reward": 5.2
    },
    {
      "path": "/models/cartpole_dqn.pkl",
      "mean_reward": 185.3,
      "std_reward": 8.1
    }
  ],
  "best_model": "/models/cartpole_ppo.pkl",
  "error": null
}
```

**Backend**: Stable-Baselines3

---

### 9. rl_export_model

Export RL model to different formats.

**Function**: Convert RL model format

**Parameters**:
```json
{
  "model_path": {
    "type": "string",
    "description": "Path to trained model",
    "required": true
  },
  "format": {
    "type": "string",
    "enum": ["onnx", "tensorflow", "pytorch", "tflite"],
    "description": "Export format",
    "required": true
  },
  "output_path": {
    "type": "string",
    "description": "Output file path",
    "required": true
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "model_path": "/models/cartpole_ppo.pkl",
  "format": "onnx",
  "output_path": "/models/cartpole_ppo.onnx",
  "file_size_mb": 42,
  "error": null
}
```

**Backend**: ONNX, TensorFlow, PyTorch

---

### 10. rl_monitor_training

Monitor RL training progress.

**Function**: Track training metrics

**Parameters**:
```json
{
  "training_id": {
    "type": "string",
    "description": "Training ID to monitor",
    "required": true
  }
}
```

**Return Format**:
```json
{
  "success": true,
  "training_id": "train_abc123",
  "status": "running",
  "current_episode": 500,
  "total_episodes": 1000,
  "progress_percent": 50,
  "mean_reward": 150.5,
  "best_reward": 195.5,
  "elapsed_time_seconds": 1800,
  "estimated_time_remaining_seconds": 1800,
  "error": null
}
```

**Backend**: Training framework

---

## MCP Dynamic Tools

### Overview

MCP (Model Context Protocol) allows dynamic tool registration from external servers.

**Configuration**:
```yaml
mcp:
  enabled: true
  servers:
    - name: "filesystem"
      url: "stdio:///usr/local/bin/mcp-server-filesystem"
      auto_start: true
    
    - name: "github"
      url: "stdio:///usr/local/bin/mcp-server-github"
      auto_start: true
      config:
        token: "${GITHUB_TOKEN}"
    
    - name: "slack"
      url: "stdio:///usr/local/bin/mcp-server-slack"
      auto_start: true
      config:
        token: "${SLACK_BOT_TOKEN}"
```

### Dynamic Tool Discovery

Tools are automatically discovered from MCP servers:

```bash
hermes tools list --mcp
```

### Example MCP Tools

**Filesystem MCP**:
- `fs_read_file` - Read file contents
- `fs_write_file` - Write file contents
- `fs_list_directory` - List directory contents
- `fs_create_directory` - Create directory
- `fs_delete_file` - Delete file

**GitHub MCP**:
- `github_search_repos` - Search repositories
- `github_get_issue` - Get issue details
- `github_create_issue` - Create issue
- `github_list_pull_requests` - List PRs
- `github_create_pull_request` - Create PR

**Slack MCP**:
- `slack_send_message` - Send message
- `slack_get_channel_history` - Get channel history
- `slack_list_channels` - List channels
- `slack_create_channel` - Create channel

---

## Tool Configuration

### Global Tool Settings

```yaml
tools:
  # Enable/disable tool categories
  enabled_categories:
    - web
    - code
    - system
    - data
    - communication
    - productivity
    - ai
    - custom
  
  # Specific tools to disable
  disabled_tools:
    - system_reboot
    - network_change
    - file_delete
  
  # Tool timeout (seconds)
  timeout: 30
  
  # Max concurrent tools
  max_concurrent: 5
  
  # Tool retry policy
  retry:
    max_attempts: 3
    backoff_factor: 2
    backoff_max: 60
  
  # Tool rate limiting
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
```

### Tool Categories

| Category | Tools | Purpose |
|----------|-------|---------|
| `web` | web_fetch, web_parse | Web scraping and data extraction |
| `code` | code_execute, code_analyze | Code execution and analysis |
| `system` | terminal_execute, terminal_stream | System commands |
| `data` | file_read, file_write, file_delete, file_list | File operations |
| `communication` | gateways (Telegram, Discord, etc.) | Messaging platforms |
| `productivity` | workflow_*, schedule_* | Automation and scheduling |
| `ai` | vision_analyze_image, code_execute | AI-powered tools |
| `custom` | User-defined tools | Custom integrations |

### Disabling Tools

**Method 1: Configuration File**
```yaml
tools:
  disabled_tools:
    - system_reboot
    - network_change
    - file_delete
```

**Method 2: Environment Variable**
```bash
HERMES_TOOLS_DISABLED_TOOLS="system_reboot,network_change,file_delete"
```

**Method 3: Runtime**
```bash
hermes tools disable system_reboot
hermes tools disable network_change
```

### Tool Approval Requirements

Dangerous operations require approval:

```yaml
approvals:
  enabled: true
  mode: manual  # or: auto, hybrid
  
  dangerous_operations:
    - file_delete
    - system_reboot
    - network_change
    - database_drop
  
  timeout: 300  # seconds
  
  channels:
    - telegram
    - email
```

---

## Toolset Compositions

### Predefined Toolsets

**Web Scraper Toolset**
```yaml
toolsets:
  web_scraper:
    tools:
      - browser_navigate
      - browser_get_text
      - browser_get_html
      - web_fetch
      - web_parse
    description: "Web scraping and analysis"
```

**Code Analyzer Toolset**
```yaml
toolsets:
  code_analyzer:
    tools:
      - code_execute
      - terminal_execute
      - file_read
      - file_list
    description: "Code analysis and execution"
```

**Data Pipeline Toolset**
```yaml
toolsets:
  data_pipeline:
    tools:
      - file_read
      - file_write
      - code_execute
      - workflow_create
      - workflow_execute
    description: "Data processing and transformation"
```

**Automation Toolset**
```yaml
toolsets:
  automation:
    tools:
      - workflow_create
      - workflow_execute
      - schedule_create
      - schedule_cancel
      - terminal_execute
    description: "Task automation and scheduling"
```

### Custom Toolsets

Define custom toolsets:

```yaml
toolsets:
  my_custom_toolset:
    tools:
      - tool_1
      - tool_2
      - tool_3
    description: "My custom toolset"
    enabled: true
```

Use custom toolsets:

```bash
hermes run --toolset my_custom_toolset "task description"
```

---

## Tool Composition Rules

### Sequential Execution

Tools execute sequentially by default:

```yaml
workflow:
  steps:
    - name: "fetch_data"
      tool: web_fetch
      params:
        url: "https://example.com"
    
    - name: "parse_data"
      tool: web_parse
      params:
        content: "{{ steps.fetch_data.body }}"
        format: "html"
```

### Parallel Execution

Enable parallel execution:

```yaml
workflow:
  steps:
    - name: "parallel_tasks"
      parallel: true
      tasks:
        - tool: web_fetch
          params:
            url: "https://example1.com"
        
        - tool: web_fetch
          params:
            url: "https://example2.com"
```

### Conditional Execution

Execute tools conditionally:

```yaml
workflow:
  steps:
    - name: "fetch_data"
      tool: web_fetch
      params:
        url: "https://example.com"
    
    - name: "process_if_success"
      tool: code_execute
      condition: "{{ steps.fetch_data.success }}"
      params:
        code: "print('Processing data')"
```

### Error Handling

Handle tool errors:

```yaml
workflow:
  steps:
    - name: "fetch_data"
      tool: web_fetch
      params:
        url: "https://example.com"
      on_failure: "retry"  # or: skip, fail
      retry:
        max_attempts: 3
        backoff_factor: 2
```

---

## Tool Monitoring & Metrics

### Tool Execution Metrics

```bash
hermes metrics tools
```

Returns:
- Total executions
- Success rate
- Average execution time
- Error rate
- Most used tools

### Tool Logs

```bash
hermes logs tools --level DEBUG
```

### Tool Performance

```bash
hermes tools performance --sort time
```

---

## Best Practices

1. **Use Toolsets**: Group related tools for better organization
2. **Set Timeouts**: Always configure appropriate timeouts
3. **Enable Approvals**: Require approval for dangerous operations
4. **Monitor Usage**: Track tool execution metrics
5. **Error Handling**: Implement proper error handling in workflows
6. **Rate Limiting**: Enable rate limiting for external APIs
7. **Caching**: Cache results when possible to reduce API calls
8. **Logging**: Enable detailed logging for debugging

---

## Troubleshooting

### Tool Not Found

```bash
hermes tools list
hermes tools info <tool_name>
```

### Tool Timeout

Increase timeout in configuration:

```yaml
tools:
  timeout: 60  # Increase from default 30
```

### Tool Disabled

Check if tool is disabled:

```bash
hermes tools list --disabled
```

Re-enable tool:

```bash
hermes tools enable <tool_name>
```

### Tool Execution Failed

Check logs:

```bash
hermes logs tools --level DEBUG
```

---

## References

- **Configuration Reference**: [CONFIG_REFERENCE.md](../configuration/CONFIG_REFERENCE.md)
- **API Server Integration**: [API_SERVER_INTEGRATION.md](../guides/API_SERVER_INTEGRATION.md)
- **Workflow Guide**: [WORKFLOW_GUIDE.md](../guides/WORKFLOW_GUIDE.md)
- **MCP Documentation**: https://modelcontextprotocol.io/

---

**Last Updated**: April 18, 2026
**Version**: 1.0.0
