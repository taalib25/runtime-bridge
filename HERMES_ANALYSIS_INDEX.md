# Hermes Agent Helm Chart Analysis - Documentation Index

This directory contains comprehensive analysis of the **ultraworkers/hermes-agent-helm-chart** repository for deploying Hermes Agent to Kind clusters.

## 📚 Documentation Files

### 1. **HERMES_DEPLOYMENT_SUMMARY.md** (14 KB) ⭐ START HERE
**Complete end-to-end guide with all essential information**

- Executive summary
- How to get the chart
- Minimal values.yaml configurations
- Required secrets and how to provide them
- Default ports and services
- Verification procedures
- Complete Kind deployment example
- Key configuration options
- Deployment modes (Direct vs Operator-Ready)
- Integration points (ESO, Istio, Ingress, NetworkPolicy)
- Useful kubectl commands
- Important notes and constraints
- Quick troubleshooting checklist

**Best for**: Getting started, understanding the full picture

---

### 2. **HERMES_QUICK_REFERENCE.md** (3.8 KB) ⚡ QUICK START
**TL;DR version with essential commands and facts**

- 3-step deployment
- Verification commands
- Key facts table
- Minimal secrets required
- Common configurations (Gateway, API Server, Webhooks)
- Troubleshooting table
- Useful commands
- Important notes

**Best for**: Quick lookups, copy-paste commands, reference during deployment

---

### 3. **HERMES_DEPLOYMENT_GUIDE.md** (10 KB) 📖 DETAILED GUIDE
**In-depth deployment guide with step-by-step instructions**

- How to get the chart (clone/download)
- Minimal values.yaml for Kind
- With API Server enabled
- Required secrets (all 20+ options)
- How to provide secrets (3 methods)
- Default ports and services
- Port forwarding for local testing
- Verification procedures (6 steps)
- Test API server endpoints
- Check persistent storage
- Verify secrets mounted
- Common issues & troubleshooting
- Complete Kind deployment example (7 steps)
- Key configuration options
- Chart structure overview
- Reference links

**Best for**: Detailed understanding, troubleshooting, learning

---

### 4. **HERMES_CHART_ANALYSIS.md** (11 KB) 🔬 TECHNICAL DEEP DIVE
**Technical analysis of chart structure and design**

- Chart metadata
- Template structure (12 templates explained)
- Deployment.yaml validation rules
- Service auto-port derivation
- ConfigMap bootstrap logic
- Secret management
- PVC configuration
- Ingress, VirtualService, NetworkPolicy
- RBAC configuration
- External Secrets Operator integration
- Pod Disruption Budget
- Operator mode (HermesTenant CRDs)
- Default values analysis (all sections)
- Deployment modes explained
- Validation schema
- Test configurations
- Verification script
- Key design decisions
- Integration points
- Limitations & constraints

**Best for**: Understanding internals, customization, advanced usage

---

## 🎯 Quick Navigation

### I want to...

**Deploy Hermes to Kind right now**
→ Read: HERMES_QUICK_REFERENCE.md (3 min)
→ Then: HERMES_DEPLOYMENT_SUMMARY.md section 6 (5 min)

**Understand the full deployment process**
→ Read: HERMES_DEPLOYMENT_SUMMARY.md (15 min)

**Get detailed step-by-step instructions**
→ Read: HERMES_DEPLOYMENT_GUIDE.md (20 min)

**Understand how the chart works internally**
→ Read: HERMES_CHART_ANALYSIS.md (25 min)

**Troubleshoot a deployment issue**
→ Check: HERMES_QUICK_REFERENCE.md troubleshooting table
→ Then: HERMES_DEPLOYMENT_GUIDE.md common issues section
→ Then: HERMES_DEPLOYMENT_SUMMARY.md section 5

**Customize the deployment**
→ Read: HERMES_CHART_ANALYSIS.md default values section
→ Then: HERMES_DEPLOYMENT_GUIDE.md key configuration options

**Integrate with external systems**
→ Read: HERMES_DEPLOYMENT_SUMMARY.md section 9
→ Then: HERMES_CHART_ANALYSIS.md integration points section

---

## 📋 Key Information Summary

### Repository
- **URL**: https://github.com/ultraworkers/hermes-agent-helm-chart
- **Commit**: e3b685d4d0288668a37216435742cd0a659ebc6c
- **Chart Version**: 0.1.0
- **App Version**: 0.8.0
- **Min Kubernetes**: 1.25.0+

### Minimal Deployment
```bash
git clone --depth 1 https://github.com/ultraworkers/hermes-agent-helm-chart.git
cd hermes-agent-helm-chart
helm install hermes . --namespace hermes --create-namespace \
  --set secrets.OPENROUTER_API_KEY=sk-or-YOUR_KEY \
  --set apiServer.enabled=true \
  --set service.enabled=true
```

### Required Secrets
- **OPENROUTER_API_KEY** (mandatory): Get from https://openrouter.ai/keys
- **API_SERVER_KEY** (if apiServer.enabled=true)
- **TELEGRAM_BOT_TOKEN** (if telegramWebhook.enabled=true)

### Default Ports
- API Server: 8642 (disabled by default)
- Webhook: 8644 (disabled by default)
- Telegram Webhook: 8443 (disabled by default)

### Critical Constraints
- **replicaCount**: Must be 1 when persistence enabled
- **strategy.type**: Must be Recreate when persistence enabled
- **Reason**: Hermes stores mutable state in /opt/data

---

## 🔗 External References

- **Hermes Agent**: https://github.com/nousresearch/hermes-agent
- **OpenRouter API**: https://openrouter.ai
- **Helm Documentation**: https://helm.sh/docs/
- **Kubernetes Documentation**: https://kubernetes.io/docs/
- **Kind Documentation**: https://kind.sigs.k8s.io/

---

## 📝 Document Metadata

| Document | Size | Focus | Audience |
|----------|------|-------|----------|
| HERMES_DEPLOYMENT_SUMMARY.md | 14 KB | Complete guide | Everyone |
| HERMES_QUICK_REFERENCE.md | 3.8 KB | Quick lookup | Operators |
| HERMES_DEPLOYMENT_GUIDE.md | 10 KB | Detailed steps | Implementers |
| HERMES_CHART_ANALYSIS.md | 11 KB | Technical deep dive | Developers |

---

## ✅ Verification Checklist

Before deploying, ensure you have:

- [ ] Kind cluster created and running
- [ ] kubectl configured to access the cluster
- [ ] Helm 3.x installed
- [ ] OPENROUTER_API_KEY from https://openrouter.ai/keys
- [ ] Chart cloned from https://github.com/ultraworkers/hermes-agent-helm-chart
- [ ] values.yaml created with required secrets
- [ ] Sufficient disk space for 5Gi PVC (default)
- [ ] Sufficient CPU/memory (500m/1Gi minimum, 2/4Gi recommended)

---

## 🚀 Next Steps

1. **Start with**: HERMES_QUICK_REFERENCE.md (3 min read)
2. **Then read**: HERMES_DEPLOYMENT_SUMMARY.md (15 min read)
3. **Deploy**: Follow section 6 of HERMES_DEPLOYMENT_SUMMARY.md
4. **Verify**: Use commands from HERMES_QUICK_REFERENCE.md
5. **Troubleshoot**: Refer to troubleshooting sections as needed
6. **Deep dive**: Read HERMES_CHART_ANALYSIS.md for customization

---

## 📞 Support

If you encounter issues:

1. Check the troubleshooting sections in the documentation
2. Review the Common Issues & Fixes tables
3. Check pod logs: `kubectl logs -n hermes deployment/hermes -f`
4. Describe pod: `kubectl describe pod -n hermes <pod-name>`
5. Check chart repository: https://github.com/ultraworkers/hermes-agent-helm-chart/issues

---

**Last Updated**: April 18, 2026
**Chart Analyzed**: ultraworkers/hermes-agent-helm-chart @ e3b685d4d0288668a37216435742cd0a659ebc6c
