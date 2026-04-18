# Hermes Agent Skills — Examples & Best Practices

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0

---

## Table of Contents

1. [Complete Skill Examples](#complete-skill-examples)
2. [Best Practices](#best-practices)
3. [Common Patterns](#common-patterns)
4. [Skill Design Decisions](#skill-design-decisions)
5. [Testing Skills](#testing-skills)
6. [Publishing Skills](#publishing-skills)

---

## Complete Skill Examples

### Example 1: Simple Utility Skill

**Use Case**: Quick reference for a common task

```markdown
---
name: docker-quick-reference
description: Quick reference for common Docker commands and patterns
version: 1.0.0
author: Hermes Team
license: MIT
metadata:
  hermes:
    tags: [docker, devops, containers]
    category: devops
---

# Docker Quick Reference

## When to Use

When you need to quickly look up Docker commands, patterns, or best practices.

## Quick Reference

### Container Lifecycle

| Command | Purpose |
|---------|---------|
| `docker run -d --name myapp image:tag` | Run container in background |
| `docker ps` | List running containers |
| `docker logs myapp` | View container logs |
| `docker stop myapp` | Stop container |
| `docker rm myapp` | Remove container |

### Image Management

| Command | Purpose |
|---------|---------|
| `docker build -t myapp:1.0 .` | Build image |
| `docker tag myapp:1.0 registry/myapp:1.0` | Tag image |
| `docker push registry/myapp:1.0` | Push to registry |
| `docker pull registry/myapp:1.0` | Pull from registry |

### Debugging

| Command | Purpose |
|---------|---------|
| `docker exec -it myapp bash` | Shell into container |
| `docker inspect myapp` | View container details |
| `docker stats` | View resource usage |

## Pitfalls

- **Using `:latest` tag**: Always use specific version tags
- **Running as root**: Create non-root user in Dockerfile
- **Large images**: Use multi-stage builds to reduce size
- **Secrets in images**: Use Docker secrets or env vars at runtime

## Verification

Verify Docker is working:

```bash
docker run hello-world
```

## References

- [Docker Documentation](https://docs.docker.com/)
- [Docker Best Practices](https://docs.docker.com/develop/dev-best-practices/)
```

### Example 2: Skill with Environment Variables

**Use Case**: Skill requiring API credentials

```markdown
---
name: github-automation
description: Automate GitHub workflows using the gh CLI
version: 1.0.0
author: Hermes Team
license: MIT
metadata:
  hermes:
    tags: [github, automation, devops]
    category: github
    required_environment_variables:
      - name: GITHUB_TOKEN
        prompt: "GitHub personal access token"
        help: "Create at https://github.com/settings/tokens"
        required_for: "GitHub API access"
      - name: GITHUB_USER
        prompt: "Your GitHub username"
        help: "Used for authentication"
        required_for: "API requests"
---

# GitHub Automation

## When to Use

When you need to automate GitHub workflows — creating issues, managing PRs, running workflows.

## Procedure

1. Ensure `GITHUB_TOKEN` and `GITHUB_USER` are set in `~/.hermes/.env`
2. Verify authentication: `gh auth status`
3. Run GitHub commands via the agent

## Examples

### Create an Issue

```bash
gh issue create --title "Bug: Login fails" --body "Users cannot log in with SSO"
```

### List Pull Requests

```bash
gh pr list --state open --limit 10
```

### Run Workflow

```bash
gh workflow run deploy.yml --ref main
```

## Pitfalls

- **Token expiration**: Refresh token if commands fail with auth errors
- **Insufficient permissions**: Token needs `repo`, `workflow` scopes
- **Rate limiting**: GitHub API has rate limits; use `gh auth refresh` to increase

## Verification

```bash
gh auth status
gh repo view
```

## References

- [GitHub CLI Documentation](https://cli.github.com/)
- [Creating Personal Access Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
```

### Example 3: Skill with Conditional Activation

**Use Case**: Fallback skill when premium tool unavailable

```markdown
---
name: free-web-search
description: Free web search via DuckDuckGo (fallback when premium search unavailable)
version: 1.0.0
author: Hermes Team
license: MIT
metadata:
  hermes:
    tags: [search, research, web]
    category: research
    fallback_for_toolsets: [web]
    fallback_for_tools: [web_search]
---

# Free Web Search

## When to Use

When you need to search the web but don't have access to premium search tools (Firecrawl, etc.).

This skill automatically appears only when the web toolset is unavailable.

## Procedure

1. Use the `ddgs` CLI or Python library to search
2. Parse results
3. Extract relevant information

## Examples

### Text Search

```bash
ddgs --text "machine learning papers 2024" --max-results 10
```

### News Search

```bash
ddgs --news "AI safety" --max-results 5
```

### Image Search

```bash
ddgs --images "neural networks" --max-results 20
```

## Pitfalls

- **Rate limiting**: DuckDuckGo may rate-limit aggressive searches
- **Result quality**: Free search may have fewer results than premium tools
- **No API key needed**: But respect rate limits

## Verification

```bash
ddgs --text "test query" --max-results 1
```

## References

- [DuckDuckGo Search](https://duckduckgo.com/)
- [ddgs CLI](https://github.com/raynold/ddgs)
```

### Example 4: Complex Skill with Multiple Sections

**Use Case**: Comprehensive workflow skill

```markdown
---
name: kubernetes-deployment
description: Deploy applications to Kubernetes with health checks and monitoring
version: 2.0.0
author: Hermes Team
license: MIT
metadata:
  hermes:
    tags: [kubernetes, devops, deployment, infrastructure]
    category: devops
    related_skills: [docker-management, monitoring-setup]
    requires_toolsets: [terminal]
    config:
      - key: k8s.namespace
        description: Default Kubernetes namespace
        default: "default"
        prompt: "Kubernetes namespace for deployments"
      - key: k8s.registry
        description: Container registry URL
        default: "docker.io"
        prompt: "Container registry (e.g., docker.io, gcr.io)"
---

# Kubernetes Deployment

## When to Use

When deploying containerized applications to Kubernetes clusters with:
- Health checks and readiness probes
- Resource limits and requests
- Monitoring and logging
- Rolling updates and rollbacks

## Quick Reference

### Deployment Manifest Template

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: myapp
        image: registry/myapp:1.0
        ports:
        - containerPort: 8080
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
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

## Procedure

### 1. Prepare Application

- Containerize application (see docker-management skill)
- Push image to registry
- Ensure health check endpoints exist (`/health`, `/ready`)

### 2. Create Deployment Manifest

- Use template above
- Update image, replicas, resources
- Configure health checks for your app

### 3. Deploy to Cluster

```bash
kubectl apply -f deployment.yaml
kubectl rollout status deployment/myapp
```

### 4. Verify Deployment

```bash
kubectl get pods -l app=myapp
kubectl logs -l app=myapp
kubectl describe deployment myapp
```

### 5. Monitor and Update

```bash
# View metrics
kubectl top pods -l app=myapp

# Update image
kubectl set image deployment/myapp myapp=registry/myapp:2.0

# Check rollout status
kubectl rollout status deployment/myapp

# Rollback if needed
kubectl rollout undo deployment/myapp
```

## Advanced Usage

### Blue-Green Deployment

```bash
# Deploy new version (green)
kubectl apply -f deployment-v2.yaml

# Switch traffic
kubectl patch service myapp -p '{"spec":{"selector":{"version":"v2"}}}'

# Remove old version (blue)
kubectl delete deployment myapp-v1
```

### Canary Deployment

Use Flagger or Istio for automated canary deployments:

```bash
kubectl apply -f canary.yaml
```

### Horizontal Pod Autoscaling

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: myapp-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: myapp
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## Pitfalls

- **No health checks**: Always implement `/health` and `/ready` endpoints
- **Resource limits too low**: Monitor actual usage and adjust
- **Immediate restarts**: Set `initialDelaySeconds` high enough for app startup
- **No PodDisruptionBudget**: Add PDB to prevent disruptions during node maintenance
- **Hardcoded configuration**: Use ConfigMaps and Secrets
- **No logging**: Ensure logs go to stdout/stderr for `kubectl logs`

## Verification

```bash
# Check deployment status
kubectl get deployment myapp
kubectl describe deployment myapp

# Check pod status
kubectl get pods -l app=myapp
kubectl logs -l app=myapp

# Check service
kubectl get service myapp
kubectl describe service myapp

# Test endpoint
kubectl port-forward svc/myapp 8080:8080
curl http://localhost:8080/health
```

## References

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Deployment Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
- [Health Checks](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [Resource Management](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
```

---

## Best Practices

### 1. Clear Naming

**Good**:
- `github-pr-workflow` — Clear what it does
- `kubernetes-deployment` — Specific domain
- `fine-tuning-with-trl` — Tool + action

**Bad**:
- `workflow` — Too generic
- `ml-stuff` — Vague
- `helper` — Unclear purpose

### 2. Comprehensive Descriptions

**Good**:
```yaml
description: Deploy applications to Kubernetes with health checks, resource limits, and monitoring
```

**Bad**:
```yaml
description: Kubernetes deployment
```

### 3. Include Pitfalls Section

Every skill should document known failure modes:

```markdown
## Pitfalls

- **Pitfall 1**: Description and how to fix
- **Pitfall 2**: Description and how to fix
```

### 4. Provide Verification Steps

Always include how to verify the skill worked:

```markdown
## Verification

- Check that [result]
- Verify [condition]
- Run [command] to confirm
```

### 5. Use Tables for Reference Material

```markdown
| Command | Purpose |
|---------|---------|
| `command1` | What it does |
| `command2` | What it does |
```

### 6. Include Examples

Provide real-world examples:

```markdown
## Examples

### Example 1: Basic Usage

```bash
command --flag value
```

### Example 2: Advanced Usage

```bash
command --flag value --advanced-flag
```
```

### 7. Document Environment Variables

If skill needs env vars:

```yaml
required_environment_variables:
  - name: API_KEY
    prompt: "Enter your API key"
    help: "Get one at https://example.com"
    required_for: "API access"
```

### 8. Add Related Skills

Link to related skills:

```yaml
related_skills: [docker-management, monitoring-setup]
```

### 9. Use Proper Frontmatter

Always include required fields:

```yaml
name: skill-name
description: Brief description
version: 1.0.0
```

### 10. Keep It Focused

Each skill should do one thing well:

**Good**: `github-pr-workflow` (focused on PR workflow)  
**Bad**: `github-everything` (too broad)

---

## Common Patterns

### Pattern 1: CLI Tool Wrapper

Skill that wraps a CLI tool:

```markdown
---
name: tool-wrapper
description: Wrapper around [Tool] CLI for [purpose]
---

## When to Use

When you need to [use the tool].

## Prerequisites

- Tool installed: `which tool-name`
- Configuration: `tool-name config`

## Procedure

1. Verify tool is installed
2. Run tool with appropriate flags
3. Parse output

## Examples

```bash
tool-name --flag value
```
```

### Pattern 2: API Integration

Skill that integrates with an API:

```markdown
---
name: api-integration
description: Integration with [Service] API
metadata:
  hermes:
    required_environment_variables:
      - name: API_KEY
        prompt: "API key"
        help: "Get at https://example.com"
---

## When to Use

When you need to interact with [Service].

## Authentication

Set `API_KEY` in `~/.hermes/.env`.

## Procedure

1. Authenticate with API
2. Make request
3. Parse response

## Examples

```bash
curl -H "Authorization: Bearer $API_KEY" https://api.example.com/endpoint
```
```

### Pattern 3: Multi-Step Workflow

Skill that orchestrates multiple steps:

```markdown
---
name: multi-step-workflow
description: Complete workflow for [task]
---

## When to Use

When you need to [accomplish task].

## Procedure

### Step 1: Preparation

- Check prerequisites
- Validate inputs

### Step 2: Execution

- Run main task
- Handle errors

### Step 3: Verification

- Verify results
- Cleanup

## Pitfalls

- **Step 1 failure**: How to recover
- **Step 2 failure**: How to recover
- **Step 3 failure**: How to recover
```

### Pattern 4: Fallback Skill

Skill that provides alternative when premium tool unavailable:

```markdown
---
name: fallback-skill
description: Free alternative to [Premium Tool]
metadata:
  hermes:
    fallback_for_toolsets: [premium]
---

## When to Use

When you don't have access to [Premium Tool].

This skill automatically appears only when [Premium Tool] is unavailable.

## Limitations

- [Limitation 1]
- [Limitation 2]

## Procedure

[Steps using free alternative]
```

---

## Skill Design Decisions

### Should It Be a Skill or a Tool?

**Make it a Skill when**:
- Capability can be expressed as instructions + shell commands
- Uses existing tools
- Provides workflow/procedure
- Examples: arXiv search, git workflows, Docker management

**Make it a Tool when**:
- Requires custom code/logic
- Needs to be called frequently
- Requires state management
- Examples: web search, code execution, terminal

### Skill Scope

**Good Scope**:
- Single domain (GitHub, Kubernetes, Docker)
- Clear use case
- Focused workflow
- 1-3 main procedures

**Bad Scope**:
- Multiple unrelated domains
- Too broad ("everything")
- Too narrow ("one command")
- Unclear purpose

### Skill Complexity

**Simple Skills** (< 100 lines):
- Quick reference
- Single command wrapper
- Basic workflow

**Medium Skills** (100-500 lines):
- Multi-step workflow
- Multiple examples
- Advanced usage section

**Complex Skills** (> 500 lines):
- Comprehensive guide
- Multiple workflows
- Extensive examples
- Consider breaking into multiple skills

---

## Testing Skills

### Manual Testing

```bash
# Load skill in CLI
hermes chat --toolsets skills -q "Show me the my-skill skill"

# Use skill via slash command
/my-skill test argument

# Test in conversation
hermes chat -q "Use my-skill to do something"
```

### Verification Checklist

- [ ] Skill loads without errors
- [ ] Description is clear and accurate
- [ ] All examples work
- [ ] Pitfalls are documented
- [ ] Verification steps work
- [ ] Environment variables (if any) are set
- [ ] Related skills are linked
- [ ] Frontmatter is valid YAML
- [ ] Markdown is properly formatted
- [ ] No broken links

### Testing Conditional Activation

```bash
# Test fallback skill
# 1. Disable required toolset
hermes config set skills.config.my-setting false

# 2. Verify skill appears
hermes chat --toolsets skills -q "List skills"

# 3. Re-enable toolset
hermes config set skills.config.my-setting true

# 4. Verify skill disappears
hermes chat --toolsets skills -q "List skills"
```

---

## Publishing Skills

### To Official Optional Skills

1. Create skill in `optional-skills/<category>/<skill-name>/SKILL.md`
2. Test thoroughly
3. Submit PR to [hermes-agent](https://github.com/NousResearch/hermes-agent)
4. Wait for review and merge

### To skills.sh

1. Create skill in your GitHub repo
2. Ensure proper structure and frontmatter
3. Submit to [skills.sh](https://skills.sh/)
4. Wait for approval

### To Custom Repository

1. Create skill in your repo
2. Ensure proper structure
3. Users install with: `hermes skills install <org>/<repo>/<path>`

### To Well-Known Endpoint

1. Create `/.well-known/skills/index.json` on your domain
2. List skills with metadata
3. Users discover with: `hermes skills search https://yourdomain.com --source well-known`

### Publishing Checklist

- [ ] Skill is well-tested
- [ ] Documentation is complete
- [ ] Examples are accurate
- [ ] Pitfalls are documented
- [ ] Verification steps work
- [ ] Frontmatter is valid
- [ ] No secrets in skill content
- [ ] License is specified
- [ ] Version is incremented
- [ ] README explains what skill does

---

## Summary

### Key Principles

1. **Clear Purpose**: One skill, one domain
2. **Complete Documentation**: Include examples, pitfalls, verification
3. **User-Friendly**: Easy to understand and use
4. **Well-Tested**: Verify all examples work
5. **Maintainable**: Easy to update and improve

### Quick Checklist

- [ ] Skill has clear name and description
- [ ] Frontmatter is complete and valid
- [ ] "When to Use" section explains purpose
- [ ] Procedure is step-by-step
- [ ] Examples are real and tested
- [ ] Pitfalls are documented
- [ ] Verification steps work
- [ ] Related skills are linked
- [ ] No broken links or typos

---

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0

