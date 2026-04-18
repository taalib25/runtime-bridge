# Hermes Agent Skills Documentation Index

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0

---

## Documentation Overview

This index provides a complete guide to Hermes Agent skills documentation. Choose the guide that matches your needs.

---

## 📚 Documentation Files

### 1. **SKILLS_COMPREHENSIVE_GUIDE.md** (1,089 lines)
**For**: Complete reference, skill catalog, all details  
**Contains**:
- Overview of skills system
- Complete catalog of 79 bundled skills (organized by category)
- Complete catalog of 22 optional skills
- SKILL.md format and structure
- Skill metadata and configuration options
- Skill installation sources (7 registries)
- Skill management commands
- Skills as slash commands
- Progressive disclosure pattern
- External skill directories
- Agent-managed skills
- Skills Hub and registries
- Security and trust levels

**Use When**: You need exhaustive reference material, complete skill listings, or detailed explanations

---

### 2. **SKILLS_QUICK_REFERENCE.md** (446 lines)
**For**: Quick lookup, common commands, at-a-glance reference  
**Contains**:
- Skills at a glance (101 total: 79 bundled + 22 optional)
- Bundled skills by category (quick list)
- Optional skills by category (quick list)
- Installation sources table
- Common commands (discovery, installation, management)
- Using skills (slash commands, conversation)
- SKILL.md template
- Skill metadata options
- Skill directory structure
- External skill directories
- Progressive disclosure
- Agent-managed skills
- Security and trust levels
- Troubleshooting

**Use When**: You need quick answers, command syntax, or a cheat sheet

---

### 3. **SKILLS_EXAMPLES_AND_BEST_PRACTICES.md** (879 lines)
**For**: Learning by example, best practices, skill development  
**Contains**:
- Complete skill examples (4 detailed examples)
  - Simple utility skill
  - Skill with environment variables
  - Skill with conditional activation
  - Complex multi-section skill
- Best practices (10 principles)
- Common patterns (4 patterns)
- Skill design decisions
- Testing skills
- Publishing skills
- Verification checklist

**Use When**: You're creating skills, learning best practices, or need examples

---

## 🎯 Quick Navigation

### I want to...

#### **Find a specific skill**
→ See **SKILLS_COMPREHENSIVE_GUIDE.md** → Bundled Skills Catalog or Optional Skills Catalog

#### **Get a quick command**
→ See **SKILLS_QUICK_REFERENCE.md** → Common Commands

#### **Install a skill**
→ See **SKILLS_QUICK_REFERENCE.md** → Installation Sources  
→ Or **SKILLS_COMPREHENSIVE_GUIDE.md** → Skill Installation Sources

#### **Create a new skill**
→ See **SKILLS_EXAMPLES_AND_BEST_PRACTICES.md** → Complete Skill Examples  
→ Then **SKILLS_COMPREHENSIVE_GUIDE.md** → SKILL.md Format & Structure

#### **Understand the skills system**
→ See **SKILLS_COMPREHENSIVE_GUIDE.md** → Overview

#### **Use a skill as a slash command**
→ See **SKILLS_QUICK_REFERENCE.md** → Using Skills  
→ Or **SKILLS_COMPREHENSIVE_GUIDE.md** → Skills as Slash Commands

#### **Configure external skill directories**
→ See **SKILLS_COMPREHENSIVE_GUIDE.md** → External Skill Directories  
→ Or **SKILLS_QUICK_REFERENCE.md** → External Skill Directories

#### **Understand progressive disclosure**
→ See **SKILLS_COMPREHENSIVE_GUIDE.md** → Progressive Disclosure Pattern  
→ Or **SKILLS_QUICK_REFERENCE.md** → Progressive Disclosure

#### **Learn about security and trust**
→ See **SKILLS_COMPREHENSIVE_GUIDE.md** → Security & Trust Levels  
→ Or **SKILLS_QUICK_REFERENCE.md** → Security & Trust

#### **Troubleshoot a skill issue**
→ See **SKILLS_QUICK_REFERENCE.md** → Troubleshooting

---

## 📊 Skills Summary

### Total Skills: 101

#### Bundled Skills: 79
- Apple (4) — macOS only
- Autonomous AI Agents (4)
- Data Science (1)
- Creative (4)
- DevOps (1)
- Dogfood (2)
- Email (1)
- Gaming (2)
- GitHub (6)
- Inference.sh (1)
- Leisure (1)
- MCP (2)
- Media (4)
- MLOps (1)
- MLOps/Cloud (2)
- MLOps/Evaluation (5)
- MLOps/Inference (8)
- MLOps/Models (6)
- MLOps/Research (1)
- MLOps/Training (14)
- MLOps/Vector Databases (4)
- Note-Taking (1)
- Productivity (6)
- Research (7)
- Red-Teaming (1)
- Smart Home (1)
- Social Media (1)
- Software Development (7)

#### Optional Skills: 22
- Autonomous AI Agents (1)
- Blockchain (2)
- Creative (2)
- DevOps (1)
- Email (1)
- Health (1)
- MCP (1)
- Migration (1)
- Productivity (1)
- Research (2)
- Security (3)

---

## 🔧 Key Concepts

### Skills System
- **Storage**: `~/.hermes/skills/` (primary, read-write)
- **Format**: Markdown with YAML frontmatter (`SKILL.md`)
- **Standard**: Compatible with agentskills.io
- **Slash Commands**: Every skill becomes `/skill-name`
- **Progressive Disclosure**: Load only what's needed

### Installation Sources
1. **Official** — Hermes repo optional-skills
2. **skills.sh** — Vercel's public directory
3. **Well-Known** — URL-based discovery
4. **GitHub** — Direct repo installs
5. **ClawHub** — Third-party marketplace
6. **LobeHub** — Agent catalog conversion
7. **Claude Marketplace** — Marketplace repos

### Skill Management
- **Browse**: `hermes skills browse`
- **Search**: `hermes skills search <query>`
- **Install**: `hermes skills install <id>`
- **List**: `hermes skills list`
- **Check**: `hermes skills check` (for updates)
- **Update**: `hermes skills update`
- **Uninstall**: `hermes skills uninstall <name>`

### Agent-Managed Skills
- Hermes creates skills from experience
- Actions: create, patch, edit, delete, write_file, remove_file
- Stored in `~/.hermes/skills/`
- Procedural memory for reuse

---

## 📖 SKILL.md Format

### Minimal Template
```markdown
---
name: my-skill
description: Brief description
version: 1.0.0
---

# Skill Title

## When to Use

When you need to [do something].

## Procedure

1. Step one
2. Step two

## Verification

Check that [result].
```

### Complete Template
```markdown
---
name: my-skill
description: Brief description
version: 1.0.0
author: Your Name
license: MIT
platforms: [macos, linux]
metadata:
  hermes:
    tags: [python, automation]
    category: devops
    related_skills: [other-skill]
    fallback_for_toolsets: [web]
    requires_toolsets: [terminal]
    config:
      - key: my.setting
        description: "What this controls"
        default: "value"
        prompt: "Prompt for setup"
    required_environment_variables:
      - name: MY_API_KEY
        prompt: "Enter your API key"
        help: "Get one at https://example.com"
        required_for: "API access"
---

# Skill Title

## When to Use

Trigger conditions.

## Procedure

1. Step one
2. Step two

## Pitfalls

- **Pitfall 1**: Description and fix

## Verification

Check that [result].

## References

- [Link 1](https://example.com)
```

---

## 🚀 Common Tasks

### Install a Skill
```bash
# From official optional skills
hermes skills install official/security/1password

# From skills.sh
hermes skills install skills-sh/vercel-labs/json-render/json-render-react --force

# From GitHub
hermes skills install openai/skills/k8s
```

### Use a Skill
```bash
# As slash command
/gif-search funny cats
/github-pr-workflow create a PR for auth refactor
/plan design a rollout

# In conversation
hermes chat --toolsets skills -q "Show me the axolotl skill"
```

### Create a Skill
1. Read **SKILLS_EXAMPLES_AND_BEST_PRACTICES.md** → Complete Skill Examples
2. Create `~/.hermes/skills/<category>/<skill-name>/SKILL.md`
3. Follow SKILL.md format
4. Test with: `hermes chat --toolsets skills -q "Show me my-skill"`
5. Verify all examples work

### Publish a Skill
```bash
# To GitHub
hermes skills publish skills/my-skill --to github --repo owner/repo

# To skills.sh
# Submit via skills.sh website

# To custom repository
# Create skill in your repo, users install with:
# hermes skills install <org>/<repo>/<path>
```

---

## 🔐 Security

### Trust Levels
- **builtin**: Ships with Hermes (always trusted)
- **official**: optional-skills/ in repo (builtin trust)
- **trusted**: openai/skills, anthropics/skills (permissive)
- **community**: Everything else (can override with --force)

### Security Scanning
All hub-installed skills scanned for:
- Data exfiltration
- Prompt injection
- Destructive commands
- Supply-chain signals

### Using --force
```bash
hermes skills install skills-sh/anthropics/skills/pdf --force
```
- Overrides non-dangerous findings
- Does NOT override dangerous verdicts
- Official skills treated as builtin trust

---

## 📝 Best Practices

1. **Clear Naming**: `github-pr-workflow` not `workflow`
2. **Comprehensive Descriptions**: Explain what skill does
3. **Include Pitfalls**: Document known failure modes
4. **Provide Verification**: Show how to verify it worked
5. **Use Tables**: For reference material
6. **Include Examples**: Real-world usage
7. **Document Env Vars**: If skill needs credentials
8. **Add Related Skills**: Link to related skills
9. **Use Proper Frontmatter**: Include all required fields
10. **Keep It Focused**: One skill, one domain

---

## 🆘 Troubleshooting

### Skill Not Appearing
- Check platform restrictions: `platforms: [macos]`
- Check conditional activation: `fallback_for_toolsets`, `requires_toolsets`

### Rate Limiting
- GitHub API: 60 requests/hour (unauthenticated)
- Solution: Set `GITHUB_TOKEN` in `~/.hermes/.env`

### Bundled Skill Conflicts
```bash
# Clear manifest, keep local copy
hermes skills reset google-workspace

# Full restore (delete local, re-copy bundled)
hermes skills reset google-workspace --restore
```

---

## 📚 Related Documentation

- [Hermes Agent Main Docs](https://hermes-agent.nousresearch.com/docs/)
- [Bundled Skills Catalog](https://hermes-agent.nousresearch.com/docs/reference/skills-catalog)
- [Optional Skills Catalog](https://hermes-agent.nousresearch.com/docs/reference/optional-skills-catalog)
- [Creating Skills Guide](https://hermes-agent.nousresearch.com/docs/developer-guide/creating-skills/)
- [agentskills.io Standard](https://agentskills.io/specification)

---

## 📞 Support

- **Issues**: https://github.com/NousResearch/hermes-agent/issues
- **Discussions**: https://github.com/NousResearch/hermes-agent/discussions
- **Discord**: https://discord.gg/NousResearch
- **Skills Hub**: https://agentskills.io

---

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0  
**Documentation Version**: 2.0

