# Hermes Agent - API Server & Open WebUI Integration Guide

Complete reference for setting up the OpenAI-compatible API server and integrating with Open WebUI.

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0+

---

## Table of Contents

1. [API Server Overview](#api-server-overview)
2. [API Server Configuration](#api-server-configuration)
3. [OpenAI-Compatible Endpoints](#openai-compatible-endpoints)
4. [Authentication](#authentication)
5. [CORS Configuration](#cors-configuration)
6. [Rate Limiting](#rate-limiting)
7. [Open WebUI Integration](#open-webui-integration)
8. [Client Libraries](#client-libraries)
9. [Streaming Responses](#streaming-responses)
10. [Error Handling](#error-handling)
11. [Monitoring & Debugging](#monitoring--debugging)

---

## API Server Overview

### What is the API Server?

The API server provides an OpenAI-compatible REST API that allows external applications to interact with Hermes Agent.

### Key Features

- ✓ OpenAI-compatible endpoints
- ✓ Streaming responses
- ✓ Authentication via API key
- ✓ CORS support
- ✓ Rate limiting
- ✓ Health checks
- ✓ Prometheus metrics

### Architecture

```
Client (Open WebUI, SDK, etc.)
    ↓
API Server (Port 8642)
    ↓
Authentication & Rate Limiting
    ↓
Hermes Agent
    ↓
Response → Streaming → Client
```

---

## API Server Configuration

### Basic Configuration

**config.yaml**:
```yaml
api_server:
  enabled: true
  host: 0.0.0.0
  port: 8642
  key: "${API_SERVER_KEY}"
  model_name: "hermes-agent"
  openai_compatible: true
```

**Environment Variables**:
```bash
API_SERVER_ENABLED=true
API_SERVER_HOST=0.0.0.0
API_SERVER_PORT=8642
API_SERVER_KEY=your-api-key-here
API_SERVER_MODEL_NAME=hermes-agent
```

### Advanced Configuration

**config.yaml**:
```yaml
api_server:
  enabled: true
  host: 0.0.0.0
  port: 8642
  key: "${API_SERVER_KEY}"
  model_name: "hermes-agent"
  openai_compatible: true
  
  # SSL/TLS
  ssl:
    enabled: false
    cert_path: /path/to/cert.pem
    key_path: /path/to/key.pem
  
  # CORS
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:3000"
      - "http://localhost:8000"
      - "https://app.example.com"
    allowed_methods:
      - GET
      - POST
      - OPTIONS
    allowed_headers:
      - Content-Type
      - Authorization
    max_age: 3600
  
  # Rate limiting
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
  
  # Timeouts
  timeout:
    request: 30
    response: 300
  
  # Logging
  logging:
    enabled: true
    level: INFO
    log_requests: true
    log_responses: false
```

**Environment Variables**:
```bash
API_SERVER_ENABLED=true
API_SERVER_HOST=0.0.0.0
API_SERVER_PORT=8642
API_SERVER_KEY=your-api-key
API_SERVER_MODEL_NAME=hermes-agent
API_SERVER_OPENAI_COMPATIBLE=true
API_SERVER_SSL_ENABLED=false
API_SERVER_CORS_ENABLED=true
API_SERVER_CORS_ORIGINS=http://localhost:3000,http://localhost:8000
API_SERVER_RATE_LIMIT_ENABLED=true
API_SERVER_RATE_LIMIT_RPM=60
API_SERVER_TIMEOUT_REQUEST=30
API_SERVER_TIMEOUT_RESPONSE=300
```

### Kubernetes Deployment

**values.yaml**:
```yaml
apiServer:
  enabled: true
  host: 0.0.0.0
  port: 8642
  
service:
  enabled: true
  type: ClusterIP
  port: 8642

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: hermes-api.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: hermes-api-tls
      hosts:
        - hermes-api.example.com

secrets:
  API_SERVER_KEY: your-api-key
```

---

## OpenAI-Compatible Endpoints

### Chat Completion

**Endpoint**: `POST /v1/chat/completions`

**Request**:
```bash
curl -X POST http://localhost:8642/v1/chat/completions \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "hermes-agent",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "What is 2+2?"}
    ],
    "temperature": 0.7,
    "max_tokens": 1000,
    "stream": false
  }'
```

**Response**:
```json
{
  "id": "chatcmpl-123456",
  "object": "chat.completion",
  "created": 1713436200,
  "model": "hermes-agent",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "2+2 equals 4."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 5,
    "total_tokens": 25
  }
}
```

### Streaming Chat Completion

**Request**:
```bash
curl -X POST http://localhost:8642/v1/chat/completions \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "hermes-agent",
    "messages": [
      {"role": "user", "content": "Tell me a story"}
    ],
    "stream": true
  }'
```

**Response** (Server-Sent Events):
```
data: {"choices":[{"delta":{"content":"Once"},"index":0}]}
data: {"choices":[{"delta":{"content":" upon"},"index":0}]}
data: {"choices":[{"delta":{"content":" a"},"index":0}]}
data: {"choices":[{"delta":{"content":" time"},"index":0}]}
data: [DONE]
```

### Models List

**Endpoint**: `GET /v1/models`

**Request**:
```bash
curl http://localhost:8642/v1/models \
  -H "Authorization: Bearer your-api-key"
```

**Response**:
```json
{
  "object": "list",
  "data": [
    {
      "id": "hermes-agent",
      "object": "model",
      "owned_by": "hermes",
      "permission": []
    }
  ]
}
```

### Health Check

**Endpoint**: `GET /health`

**Request**:
```bash
curl http://localhost:8642/health
```

**Response**:
```json
{
  "status": "healthy",
  "version": "0.10.0",
  "uptime": 3600,
  "timestamp": "2026-04-18T10:30:00Z"
}
```

### Gateway Health

**Endpoint**: `GET /health/gateways`

**Request**:
```bash
curl http://localhost:8642/health/gateways \
  -H "Authorization: Bearer your-api-key"
```

**Response**:
```json
{
  "gateways": {
    "telegram": {
      "status": "healthy",
      "connected": true,
      "last_message": "2026-04-18T10:25:00Z"
    },
    "discord": {
      "status": "healthy",
      "connected": true,
      "last_message": "2026-04-18T10:28:00Z"
    },
    "slack": {
      "status": "unhealthy",
      "connected": false,
      "error": "Invalid token"
    }
  }
}
```

---

## Authentication

### API Key Authentication

All requests must include the API key in the `Authorization` header:

```bash
Authorization: Bearer your-api-key
```

### Setting API Key

**Environment Variable**:
```bash
API_SERVER_KEY=your-api-key-here
```

**config.yaml**:
```yaml
api_server:
  key: "${API_SERVER_KEY}"
```

### API Key Rotation

**Rotate API key**:
```bash
# Generate new key
hermes admin api-key rotate

# Old key remains valid for 24 hours
# Update clients to use new key
```

### API Key Management

**List API keys**:
```bash
hermes admin api-keys list
```

**Revoke API key**:
```bash
hermes admin api-keys revoke <key-id>
```

**Create new API key**:
```bash
hermes admin api-keys create --name "Open WebUI"
```

---

## CORS Configuration

### Enable CORS

**config.yaml**:
```yaml
api_server:
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:3000"
      - "http://localhost:8000"
      - "https://app.example.com"
    allowed_methods:
      - GET
      - POST
      - OPTIONS
    allowed_headers:
      - Content-Type
      - Authorization
    expose_headers:
      - X-RateLimit-Limit
      - X-RateLimit-Remaining
      - X-RateLimit-Reset
    max_age: 3600
    allow_credentials: true
```

**Environment Variables**:
```bash
API_SERVER_CORS_ENABLED=true
API_SERVER_CORS_ORIGINS=http://localhost:3000,http://localhost:8000,https://app.example.com
API_SERVER_CORS_METHODS=GET,POST,OPTIONS
API_SERVER_CORS_HEADERS=Content-Type,Authorization
API_SERVER_CORS_MAX_AGE=3600
```

### CORS Preflight

Browsers automatically send preflight requests:

```bash
OPTIONS /v1/chat/completions HTTP/1.1
Origin: http://localhost:3000
Access-Control-Request-Method: POST
Access-Control-Request-Headers: Content-Type
```

Response:
```
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
Access-Control-Max-Age: 3600
```

---

## Rate Limiting

### Configuration

**config.yaml**:
```yaml
api_server:
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
    
    # Per-user limits
    per_user:
      enabled: true
      requests_per_minute: 30
    
    # Per-IP limits
    per_ip:
      enabled: true
      requests_per_minute: 100
```

**Environment Variables**:
```bash
API_SERVER_RATE_LIMIT_ENABLED=true
API_SERVER_RATE_LIMIT_RPM=60
API_SERVER_RATE_LIMIT_BURST_SIZE=10
API_SERVER_RATE_LIMIT_PER_USER_ENABLED=true
API_SERVER_RATE_LIMIT_PER_USER_RPM=30
API_SERVER_RATE_LIMIT_PER_IP_ENABLED=true
API_SERVER_RATE_LIMIT_PER_IP_RPM=100
```

### Rate Limit Headers

All responses include rate limit information:

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1713436200
```

### Rate Limit Exceeded

When rate limit is exceeded:

```json
{
  "error": {
    "message": "Rate limit exceeded",
    "type": "rate_limit_error",
    "param": null,
    "code": "rate_limit_exceeded"
  }
}
```

HTTP Status: `429 Too Many Requests`

---

## Open WebUI Integration

### Installation

**Docker**:
```bash
docker run -d -p 3000:8080 \
  -e OPENAI_API_BASE_URL=http://hermes:8642/v1 \
  -e OPENAI_API_KEY=your-api-key \
  ghcr.io/open-webui/open-webui:latest
```

**Docker Compose**:
```yaml
version: '3.8'

services:
  hermes:
    image: nousresearch/hermes-agent:0.10.0
    ports:
      - "8642:8642"
    environment:
      API_SERVER_ENABLED: "true"
      API_SERVER_KEY: "your-api-key"
      OPENROUTER_API_KEY: "sk-or-..."
    volumes:
      - hermes-data:/opt/data

  open-webui:
    image: ghcr.io/open-webui/open-webui:latest
    ports:
      - "3000:8080"
    environment:
      OPENAI_API_BASE_URL: "http://hermes:8642/v1"
      OPENAI_API_KEY: "your-api-key"
    depends_on:
      - hermes

volumes:
  hermes-data:
```

### Configuration in Open WebUI

1. **Access Open WebUI**: http://localhost:3000
2. **Go to Settings**: Click gear icon
3. **Select Models**: Click "Models"
4. **Add Model**:
   - Model name: `hermes-agent`
   - API Base URL: `http://localhost:8642/v1`
   - API Key: `your-api-key`
5. **Save and use**

### Kubernetes Deployment

**values.yaml**:
```yaml
hermes:
  apiServer:
    enabled: true
    port: 8642
  
  service:
    enabled: true
    type: ClusterIP

open-webui:
  enabled: true
  image: ghcr.io/open-webui/open-webui:latest
  port: 3000
  
  env:
    OPENAI_API_BASE_URL: "http://hermes:8642/v1"
    OPENAI_API_KEY: "your-api-key"
  
  ingress:
    enabled: true
    host: webui.example.com
```

---

## Client Libraries

### Python (OpenAI SDK)

```python
from openai import OpenAI

client = OpenAI(
    api_key="your-api-key",
    base_url="http://localhost:8642/v1"
)

# Non-streaming
response = client.chat.completions.create(
    model="hermes-agent",
    messages=[
        {"role": "user", "content": "What is 2+2?"}
    ]
)
print(response.choices[0].message.content)

# Streaming
stream = client.chat.completions.create(
    model="hermes-agent",
    messages=[
        {"role": "user", "content": "Tell me a story"}
    ],
    stream=True
)
for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
```

### JavaScript (OpenAI SDK)

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "your-api-key",
  baseURL: "http://localhost:8642/v1",
  dangerouslyAllowBrowser: true
});

// Non-streaming
const response = await client.chat.completions.create({
  model: "hermes-agent",
  messages: [
    { role: "user", content: "What is 2+2?" }
  ]
});
console.log(response.choices[0].message.content);

// Streaming
const stream = await client.chat.completions.create({
  model: "hermes-agent",
  messages: [
    { role: "user", content: "Tell me a story" }
  ],
  stream: true
});

for await (const chunk of stream) {
  if (chunk.choices[0].delta.content) {
    process.stdout.write(chunk.choices[0].delta.content);
  }
}
```

### cURL

```bash
# Non-streaming
curl -X POST http://localhost:8642/v1/chat/completions \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "hermes-agent",
    "messages": [
      {"role": "user", "content": "What is 2+2?"}
    ]
  }'

# Streaming
curl -X POST http://localhost:8642/v1/chat/completions \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "hermes-agent",
    "messages": [
      {"role": "user", "content": "Tell me a story"}
    ],
    "stream": true
  }'
```

---

## Streaming Responses

### Server-Sent Events (SSE)

Streaming responses use Server-Sent Events format:

```
data: {"choices":[{"delta":{"content":"Hello"},"index":0}]}
data: {"choices":[{"delta":{"content":" world"},"index":0}]}
data: [DONE]
```

### Streaming Configuration

**config.yaml**:
```yaml
streaming:
  enabled: true
  chunk_size: 1000
  chunk_delay: 100
  
  api_server:
    enabled: true
    chunk_size: 1000
    chunk_delay: 100
```

### Streaming Example (Python)

```python
stream = client.chat.completions.create(
    model="hermes-agent",
    messages=[{"role": "user", "content": "Tell me a story"}],
    stream=True
)

for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
```

---

## Error Handling

### Error Response Format

```json
{
  "error": {
    "message": "Error description",
    "type": "error_type",
    "param": null,
    "code": "error_code"
  }
}
```

### Common Errors

| Code | Status | Description |
|------|--------|-------------|
| `invalid_request_error` | 400 | Invalid request |
| `authentication_error` | 401 | Invalid API key |
| `permission_error` | 403 | Insufficient permissions |
| `not_found_error` | 404 | Resource not found |
| `rate_limit_error` | 429 | Rate limit exceeded |
| `server_error` | 500 | Server error |

### Error Handling (Python)

```python
from openai import OpenAI, APIError, RateLimitError, AuthenticationError

client = OpenAI(api_key="your-api-key", base_url="http://localhost:8642/v1")

try:
    response = client.chat.completions.create(
        model="hermes-agent",
        messages=[{"role": "user", "content": "Hello"}]
    )
except AuthenticationError:
    print("Invalid API key")
except RateLimitError:
    print("Rate limit exceeded, retry after 60 seconds")
except APIError as e:
    print(f"API error: {e}")
```

---

## Monitoring & Debugging

### Health Checks

```bash
# API server health
curl http://localhost:8642/health

# Gateway health
curl http://localhost:8642/health/gateways \
  -H "Authorization: Bearer your-api-key"

# Detailed status
curl http://localhost:8642/status \
  -H "Authorization: Bearer your-api-key"
```

### Metrics

**Prometheus endpoint**: `GET /metrics`

```bash
curl http://localhost:8642/metrics
```

**Metrics**:
- `api_requests_total` - Total requests
- `api_request_duration_seconds` - Request duration
- `api_errors_total` - Total errors
- `api_rate_limit_hits_total` - Rate limit hits

### Logging

**config.yaml**:
```yaml
api_server:
  logging:
    enabled: true
    level: INFO
    log_requests: true
    log_responses: false
    log_file: ~/.hermes/logs/api-server.log
```

**View logs**:
```bash
tail -f ~/.hermes/logs/api-server.log
```

### Debugging

**Enable debug logging**:
```bash
API_SERVER_LOG_LEVEL=DEBUG
```

**Test API endpoint**:
```bash
curl -v -X POST http://localhost:8642/v1/chat/completions \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model":"hermes-agent","messages":[{"role":"user","content":"test"}]}'
```

---

## Best Practices

1. **Use HTTPS in production** - Enable SSL/TLS
2. **Rotate API keys** - Regularly update keys
3. **Enable rate limiting** - Prevent abuse
4. **Monitor metrics** - Track usage and errors
5. **Use streaming** - Better UX for long responses
6. **Handle errors** - Implement proper error handling
7. **Test thoroughly** - Before production deployment
8. **Document setup** - For team collaboration
9. **Use environment variables** - For secrets
10. **Enable logging** - For debugging

---

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0+
