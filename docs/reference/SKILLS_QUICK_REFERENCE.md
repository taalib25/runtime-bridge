# Hermes Agent Skills — Quick Reference

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0

---

## Skills at a Glance

### Total Skills: 101
- **Bundled**: 79 (always available)
- **Optional**: 22 (install as needed)

---

## Bundled Skills by Category (79 Total)

### Apple (4) — macOS only
`apple-notes` • `apple-reminders` • `findmy` • `imessage`

### Autonomous AI Agents (4)
`claude-code` • `codex` • `hermes-agent-spawning` • `opencode`

### Data Science (1)
`jupyter-live-kernel`

### Creative (4)
`ascii-art` • `ascii-video` • `excalidraw` • `p5js`

### DevOps (1)
`webhook-subscriptions`

### Dogfood (2)
`dogfood` • `hermes-agent-setup`

### Email (1)
`himalaya`

### Gaming (2)
`minecraft-modpack-server` • `pokemon-player`

### GitHub (6)
`codebase-inspection` • `github-auth` • `github-code-review` • `github-issues` • `github-pr-workflow` • `github-repo-management`

### Inference.sh (1)
`inference-sh-cli`

### Leisure (1)
`find-nearby`

### MCP (2)
`mcporter` • `native-mcp`

### Media (4)
`gif-search` • `heartmula` • `songsee` • `youtube-content`

### MLOps (1)
`huggingface-hub`

### MLOps/Cloud (2)
`lambda-labs-gpu-cloud` • `modal-serverless-gpu`

### MLOps/Evaluation (5)
`evaluating-llms-harness` • `huggingface-tokenizers` • `nemo-curator` • `sparse-autoencoder-training` • `weights-and-biases`

### MLOps/Inference (8)
`gguf-quantization` • `guidance` • `instructor` • `llama-cpp` • `obliteratus` • `outlines` • `serving-llms-vllm` • `tensorrt-llm`

### MLOps/Models (6)
`audiocraft-audio-generation` • `clip` • `llava` • `segment-anything-model` • `stable-diffusion-image-generation` • `whisper`

### MLOps/Research (1)
`dspy`

### MLOps/Training (14)
`axolotl` • `distributed-llm-pretraining-torchtitan` • `fine-tuning-with-trl` • `grpo-rl-training` • `hermes-atropos-environments` • `huggingface-accelerate` • `optimizing-attention-flash` • `peft-fine-tuning` • `pytorch-fsdp` • `pytorch-lightning` • `simpo-training` • `slime-rl-training` • `unsloth`

### MLOps/Vector Databases (4)
`chroma` • `faiss` • `pinecone` • `qdrant-vector-search`

### Note-Taking (1)
`obsidian`

### Productivity (6)
`google-workspace` • `linear` • `nano-pdf` • `notion` • `ocr-and-documents` • `powerpoint`

### Research (7)
`arxiv` • `blogwatcher` • `llm-wiki` • `domain-intel` • `duckduckgo-search` • `ml-paper-writing` • `polymarket`

### Red-Teaming (1)
`godmode`

### Smart Home (1)
`openhue`

### Social Media (1)
`xitter`

### Software Development (7)
`code-review` • `plan` • `requesting-code-review` • `subagent-driven-development` • `systematic-debugging` • `test-driven-development` • `writing-plans`

---

## Optional Skills by Category (22 Total)

### Autonomous AI Agents (1)
`blackbox`

### Blockchain (2)
`base` • `solana`

### Creative (2)
`blender-mcp` • `meme-generation`

### DevOps (1)
`docker-management`

### Email (1)
`agentmail`

### Health (1)
`neuroskill-bci`

### MCP (1)
`fastmcp`

### Migration (1)
`openclaw-migration`

### Productivity (1)
`telephony`

### Research (2)
`bioinformatics` • `qmd`

### Security (3)
`1password` • `oss-forensics` • `sherlock`

---

## Installation Sources

| Source | Command | Example |
|--------|---------|---------|
| **Official** | `hermes skills install official/<cat>/<skill>` | `hermes skills install official/security/1password` |
| **skills.sh** | `hermes skills install skills-sh/<path>` | `hermes skills install skills-sh/vercel-labs/json-render/json-render-react` |
| **Well-Known** | `hermes skills install well-known:<url>` | `hermes skills install well-known:https://mintlify.com/docs/.well-known/skills/mintlify` |
| **GitHub** | `hermes skills install <org>/<repo>/<path>` | `hermes skills install openai/skills/k8s` |
| **ClawHub** | Search via `hermes skills search --source clawhub` | — |
| **LobeHub** | Search via `hermes skills search --source lobehub` | — |

---

## Common Commands

### Discovery
```bash
hermes skills browse                              # Browse all
hermes skills browse --source official            # Official only
hermes skills search kubernetes                   # Search all sources
hermes skills search react --source skills-sh     # Search specific source
hermes skills inspect openai/skills/k8s           # Preview
```

### Installation
```bash
hermes skills install official/security/1password
hermes skills install openai/skills/k8s
hermes skills install skills-sh/vercel-labs/json-render/json-render-react --force
```

### Management
```bash
hermes skills list --source hub                   # List installed hub skills
hermes skills check                               # Check for updates
hermes skills update                              # Update all
hermes skills update react                        # Update specific
hermes skills audit                               # Re-scan for security
hermes skills uninstall k8s                       # Remove
```

### Bundled Skills
```bash
hermes skills reset google-workspace              # Clear manifest, keep local
hermes skills reset google-workspace --restore    # Full restore to bundled
```

### In Chat
```bash
/skills browse
/skills search react --source skills-sh
/skills install openai/skills/k8s --force
/skills check
/skills update
/skills reset google-workspace
```

---

## Using Skills

### As Slash Commands
```bash
/gif-search funny cats
/axolotl help me fine-tune Llama 3
/github-pr-workflow create a PR for auth refactor
/plan design a rollout for auth migration
/excalidraw
```

### In Conversation
```bash
hermes chat --toolsets skills -q "What skills do you have?"
hermes chat --toolsets skills -q "Show me the axolotl skill"
```

---

## SKILL.md Template

```markdown
---
name: my-skill
description: Brief description
version: 1.0.0
platforms: [macos, linux]     # Optional
metadata:
  hermes:
    tags: [python, automation]
    category: devops
    required_environment_variables:
      - name: MY_API_KEY
        prompt: "Enter your API key"
        help: "Get one at https://example.com"
---

# Skill Title

## When to Use

When you need to [do something].

## Procedure

1. Step one
2. Step two

## Pitfalls

- **Pitfall 1**: Description and fix

## Verification

Check that [result].
```

---

## Skill Metadata Options

### Frontmatter Fields

```yaml
name: my-skill                    # Required: unique identifier
description: Brief description    # Required: shown in search
version: 1.0.0                   # Required: semantic version
author: Your Name                # Optional
license: MIT                     # Optional
platforms: [macos, linux]        # Optional: restrict to OS

metadata:
  hermes:
    tags: [python, automation]   # Optional: categorization
    category: devops             # Optional: category
    related_skills: [other]      # Optional: related skills
    
    # Conditional activation
    fallback_for_toolsets: [web]      # Show when unavailable
    requires_toolsets: [terminal]     # Show when available
    fallback_for_tools: [web_search]  # Show when unavailable
    requires_tools: [terminal]        # Show when available
    
    # Configuration
    config:
      - key: my.setting
        description: "What this controls"
        default: "value"
        prompt: "Prompt for setup"
    
    # Environment variables
    required_environment_variables:
      - name: MY_API_KEY
        prompt: "Enter your API key"
        help: "Get one at https://example.com"
        required_for: "API access"
```

---

## Skill Directory Structure

```
~/.hermes/skills/                  # Primary (read-write)
├── category/
│   ├── skill-name/
│   │   ├── SKILL.md               # Main instructions (required)
│   │   ├── references/            # Additional docs
│   │   ├── templates/             # Output formats
│   │   ├── scripts/               # Helper scripts
│   │   └── assets/                # Supplementary files
│   └── another-skill/
│       └── SKILL.md
├── .hub/                          # Skills Hub state
│   ├── lock.json
│   ├── quarantine/
│   └── audit.log
└── .bundled_manifest              # Tracks bundled skills
```

---

## External Skill Directories

Configure in `~/.hermes/config.yaml`:

```yaml
skills:
  external_dirs:
    - ~/.agents/skills
    - /home/shared/team-skills
    - ${SKILLS_REPO}/skills
```

---

## Progressive Disclosure

Skills load in three levels:

```
Level 0: skills_list()           → Names + descriptions (~3k tokens)
Level 1: skill_view(name)        → Full content + metadata
Level 2: skill_view(name, path)  → Specific reference file
```

---

## Agent-Managed Skills

Hermes creates skills via `skill_manage` tool:

| Action | Use For | Params |
|--------|---------|--------|
| `create` | New skill | `name`, `content`, optional `category` |
| `patch` | Targeted fixes | `name`, `old_string`, `new_string` |
| `edit` | Major rewrites | `name`, `content` |
| `delete` | Remove skill | `name` |
| `write_file` | Add files | `name`, `file_path`, `file_content` |
| `remove_file` | Remove files | `name`, `file_path` |

---

## Security & Trust

### Trust Levels

| Level | Source | Policy |
|-------|--------|--------|
| `builtin` | Ships with Hermes | Always trusted |
| `official` | `optional-skills/` | Builtin trust |
| `trusted` | `openai/skills`, `anthropics/skills` | Permissive |
| `community` | Everything else | `--force` to override |

### Using `--force`

```bash
hermes skills install skills-sh/anthropics/skills/pdf --force
```

- Overrides non-dangerous findings
- Does NOT override `dangerous` verdicts
- Official skills treated as builtin trust

---

## Troubleshooting

### Rate Limiting

GitHub API: 60 requests/hour (unauthenticated)

**Solution**: Set `GITHUB_TOKEN` in `~/.hermes/.env`

```bash
GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
```

### Bundled Skill Conflicts

If you edit a bundled skill and want to restore:

```bash
# Clear manifest, keep local copy
hermes skills reset google-workspace

# Full restore (delete local, re-copy bundled)
hermes skills reset google-workspace --restore

# Non-interactive
hermes skills reset google-workspace --restore --yes
```

### Skill Not Appearing

Check platform restrictions:

```yaml
platforms: [macos]  # Only shows on macOS
```

Check conditional activation:

```yaml
fallback_for_toolsets: [web]  # Only shows when web unavailable
requires_toolsets: [terminal]  # Only shows when terminal available
```

---

## Key Concepts

- **Bundled**: 79 skills, always available
- **Optional**: 22 skills, install as needed
- **Slash Commands**: `/skill-name` for any skill
- **Progressive Disclosure**: Load only what's needed
- **Agent-Created**: Hermes creates skills from experience
- **External Dirs**: Point to shared directories
- **7 Registries**: Multiple discovery sources
- **Security Scanning**: All hub skills scanned
- **Trust Levels**: Builtin, official, trusted, community

---

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0

