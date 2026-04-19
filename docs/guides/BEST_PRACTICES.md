# Hermes Agent - Best Practices Guide

**Production-ready best practices for deploying and operating Hermes Agent.**

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0+

---

## 🎯 Core Principles

### 1. Security First
- Always use environment variables for secrets
- Implement user allowlists for all platforms
- Enable webhook signature verification
- Rotate API keys regularly
- Use HTTPS for all external connections

### 2. Reliability
- Monitor agent health continuously
- Implement graceful error handling
- Use rate limiting to prevent abuse
- Set up automated backups
- Plan for failover scenarios

### 3. Performance
- Optimize model selection for latency
- Implement caching where appropriate
- Monitor resource usage
- Scale horizontally when needed
- Use streaming for long responses

### 4. Observability
- Structured logging for all operations
- Prometheus metrics for monitoring
- Distributed tracing for debugging
- Health check endpoints
- Alert on critical issues

---

## 🔐 Security Best Practices

### API Key Management

**DO:**
- Store API keys in `.env` file (not in config.yaml)
- Rotate keys every 90 days
- Use separate keys for different environments
- Restrict key permissions to minimum needed
- Monitor key usage for anomalies

**DON'T:**
- Commit `.env` to version control
- Share API keys via email or chat
- Use same key for dev and production
- Log API keys in debug output
- Hardcode keys in source code

```bash
# Rotate API key
hermes auth rotate-key

# Verify key permissions
hermes auth verify --key <key>

# List active keys
hermes auth list-keys
```

### User Access Control

**Implement allowlists:**
```yaml
# config.yaml
gateways:
  - name: telegram
    allowlist:
      users:
        - "123456789"  # Only specific users
        - "987654321"
      channels: []  # Empty = all channels allowed
      servers: []   # Empty = all servers allowed
```

**Use DM pairing for sensitive operations:**
```yaml
# config.yaml
security:
  dm_pairing:
    enabled: true
    require_verification: true
    timeout: 300  # 5 minutes
```

**Implement rate limiting:**
```yaml
# config.yaml
rate_limiting:
  global:
    requests_per_minute: 60
  per_user:
    requests_per_minute: 10
  per_gateway:
    requests_per_minute: 100
```

### Webhook Security

**Always verify signatures:**
```yaml
# config.yaml
webhooks:
  - url: "https://example.com/webhook"
    verify_signature: true
    secret: ${WEBHOOK_SECRET}
```

**Use HTTPS only:**
```bash
# Verify SSL certificate
curl -I https://your-webhook-endpoint.com

# Test webhook
hermes webhook test <webhook-name>
```

---

## 🚀 Deployment Best Practices

### Environment Setup

**Development:**
```bash
# Use local model for testing
export HERMES_MODEL=gpt-3.5-turbo
export HERMES_LOG_LEVEL=debug
export HERMES_API_KEY=dev-key-only
```

**Staging:**
```bash
# Use production model, but staging API key
export HERMES_MODEL=gpt-4
export HERMES_LOG_LEVEL=info
export HERMES_API_KEY=staging-key
```

**Production:**
```bash
# Use production settings
export HERMES_MODEL=gpt-4
export HERMES_LOG_LEVEL=warn
export HERMES_API_KEY=prod-key
export HERMES_ENABLE_METRICS=true
export HERMES_ENABLE_TRACING=true
```

### Kubernetes Deployment

**Use resource limits:**
```yaml
# deployment.yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "250m"
  limits:
    memory: "2Gi"
    cpu: "1000m"
```

**Implement health checks:**
```yaml
# deployment.yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
```

**Use persistent volumes for state:**
```yaml
# deployment.yaml
volumeMounts:
  - name: config
    mountPath: /etc/hermes
  - name: cache
    mountPath: /var/cache/hermes
```

### Docker Deployment

**Use multi-stage builds:**
```dockerfile
# Dockerfile
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN go build -o hermes .

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/hermes /usr/local/bin/
ENTRYPOINT ["hermes"]
```

**Run as non-root:**
```dockerfile
# Dockerfile
RUN addgroup -g 1000 hermes && \
    adduser -D -u 1000 -G hermes hermes
USER hermes
```

---

## 📊 Monitoring & Observability

### Prometheus Metrics

**Key metrics to monitor:**
```yaml
# Scrape config
scrape_configs:
  - job_name: 'hermes'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

**Important metrics:**
- `hermes_requests_total` - Total requests by platform
- `hermes_request_duration_seconds` - Request latency
- `hermes_errors_total` - Error count by type
- `hermes_gateway_status` - Gateway connection status
- `hermes_model_tokens_used` - Token usage
- `hermes_cache_hits_total` - Cache hit rate

### Logging

**Use structured logging:**
```json
{
  "timestamp": "2026-04-18T10:30:00Z",
  "level": "info",
  "message": "Message processed",
  "user_id": "123456789",
  "platform": "telegram",
  "duration_ms": 1234,
  "tokens_used": 150
}
```

**Log levels:**
- `debug` - Development only
- `info` - Normal operations
- `warn` - Potential issues
- `error` - Errors that need attention
- `fatal` - Critical errors

### Alerting

**Set up alerts for:**
```yaml
# prometheus-rules.yaml
groups:
  - name: hermes
    rules:
      - alert: HermesDown
        expr: up{job="hermes"} == 0
        for: 5m
        
      - alert: HighErrorRate
        expr: rate(hermes_errors_total[5m]) > 0.05
        
      - alert: HighLatency
        expr: histogram_quantile(0.95, hermes_request_duration_seconds) > 5
        
      - alert: GatewayDown
        expr: hermes_gateway_status == 0
```

---

## 🔄 Operational Best Practices

### Backup & Recovery

**Backup configuration:**
```bash
# Daily backup
0 2 * * * tar -czf /backup/hermes-$(date +%Y%m%d).tar.gz ~/.hermes/

# Verify backup
tar -tzf /backup/hermes-20260418.tar.gz
```

**Restore from backup:**
```bash
# Stop agent
hermes stop

# Restore backup
tar -xzf /backup/hermes-20260418.tar.gz -C ~/

# Restart agent
hermes start
```

### Update Strategy

**Test updates in staging first:**
```bash
# Staging
hermes update --version v0.11.0 --dry-run
hermes update --version v0.11.0

# Verify
hermes version
hermes test

# Production
hermes update --version v0.11.0
```

**Rollback procedure:**
```bash
# If update fails
hermes rollback

# Verify rollback
hermes version
hermes test
```

### Maintenance Windows

**Schedule maintenance:**
```bash
# Announce maintenance
hermes announce "Maintenance window: 2:00-2:30 AM UTC"

# Graceful shutdown
hermes shutdown --grace-period 60

# Perform maintenance
# ...

# Restart
hermes start

# Verify
hermes health
```

---

## 🎯 Performance Optimization

### Model Selection

**Choose appropriate model:**
| Use Case | Model | Latency | Cost |
|----------|-------|---------|------|
| Quick responses | gpt-3.5-turbo | <1s | Low |
| Complex reasoning | gpt-4 | 2-5s | High |
| Real-time chat | gpt-3.5-turbo | <1s | Low |
| Analysis | gpt-4 | 2-5s | High |

### Caching Strategy

**Cache frequently used responses:**
```yaml
# config.yaml
cache:
  enabled: true
  ttl: 3600  # 1 hour
  max_size: 1000  # MB
  strategy: "lru"  # Least Recently Used
```

### Batch Processing

**Process messages in batches:**
```yaml
# config.yaml
batch:
  enabled: true
  size: 10
  timeout: 5  # seconds
```

---

## 🧪 Testing Best Practices

### Unit Tests

```bash
# Run tests
hermes test

# Run specific test
hermes test --filter TestGateway

# Run with coverage
hermes test --coverage
```

### Integration Tests

```bash
# Test with real platforms (staging)
hermes test --integration --env staging

# Test specific platform
hermes test --integration --platform telegram
```

### Load Testing

```bash
# Simulate load
hermes load-test --requests 1000 --concurrency 10

# Monitor during load test
hermes metrics --live
```

---

## 📋 Checklist for Production Deployment

- [ ] All secrets in `.env` (not in config.yaml)
- [ ] API keys rotated and verified
- [ ] User allowlists configured
- [ ] Rate limiting enabled
- [ ] Webhook signature verification enabled
- [ ] HTTPS enabled for all external connections
- [ ] Monitoring and alerting configured
- [ ] Backup strategy implemented
- [ ] Disaster recovery plan documented
- [ ] Load testing completed
- [ ] Security audit completed
- [ ] Documentation updated
- [ ] Team trained on operations
- [ ] Runbooks created for common issues
- [ ] On-call rotation established

---

## 🚨 Incident Response

### Incident Severity Levels

| Level | Response Time | Impact |
|-------|---------------|--------|
| Critical | 15 minutes | Service down, data loss risk |
| High | 1 hour | Degraded service, errors |
| Medium | 4 hours | Minor issues, workarounds available |
| Low | 24 hours | Cosmetic issues, no impact |

### Incident Response Steps

1. **Detect**: Monitor alerts, user reports
2. **Assess**: Determine severity, scope
3. **Respond**: Implement immediate fix or workaround
4. **Communicate**: Update status page, notify users
5. **Resolve**: Implement permanent fix
6. **Review**: Post-mortem, prevent recurrence

### Common Incidents

**Agent not responding:**
1. Check if process is running: `hermes status`
2. Check logs: `hermes logs --tail 100`
3. Restart: `hermes restart`
4. If still down, check system resources

**High error rate:**
1. Check error logs: `hermes logs --level error`
2. Check rate limiting: `hermes metrics rate-limit`
3. Check model status: `hermes model status`
4. Implement circuit breaker if needed

**Performance degradation:**
1. Check system resources: `hermes system info`
2. Check request latency: `hermes metrics latency`
3. Check cache hit rate: `hermes metrics cache`
4. Scale horizontally if needed

---

## 📚 Additional Resources

- **Kubernetes Best Practices**: https://kubernetes.io/docs/concepts/configuration/overview/
- **Docker Best Practices**: https://docs.docker.com/develop/dev-best-practices/
- **Prometheus Best Practices**: https://prometheus.io/docs/practices/
- **Security Best Practices**: https://owasp.org/www-project-top-ten/
- **Hermes Documentation**: https://hermes-agent.nousresearch.com/docs/

