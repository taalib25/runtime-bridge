# Hermes Agent Skills — Comprehensive Reference Guide

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0  
**Documentation Version**: 2.0

---

## Table of Contents

1. [Overview](#overview)
2. [Bundled Skills Catalog (79 Skills)](#bundled-skills-catalog)
3. [Optional Skills Catalog (22 Skills)](#optional-skills-catalog)
4. [SKILL.md Format & Structure](#skillmd-format--structure)
5. [Skill Metadata & Configuration](#skill-metadata--configuration)
6. [Skill Installation Sources](#skill-installation-sources)
7. [Skill Management Commands](#skill-management-commands)
8. [Skills as Slash Commands](#skills-as-slash-commands)
9. [Progressive Disclosure Pattern](#progressive-disclosure-pattern)
10. [External Skill Directories](#external-skill-directories)
11. [Agent-Managed Skills](#agent-managed-skills)
12. [Skills Hub & Registries](#skills-hub--registries)
13. [Security & Trust Levels](#security--trust-levels)

---

## Overview

Skills are **on-demand knowledge documents** that Hermes Agent loads when needed. They follow a **progressive disclosure** pattern to minimize token usage and are compatible with the [agentskills.io](https://agentskills.io/specification) open standard.

### Key Facts

- **Total Skills**: 101 (79 bundled + 22 optional)
- **Storage**: `~/.hermes/skills/` (primary, read-write)
- **External Support**: Can scan additional directories via `config.yaml`
- **Format**: Markdown with YAML frontmatter (`SKILL.md`)
- **Slash Commands**: Every skill becomes a `/skill-name` command
- **Agent-Created**: Hermes can create and modify skills autonomously
- **Hub Integration**: 7 integrated registries (official, skills.sh, well-known, GitHub, ClawHub, LobeHub, Claude marketplace)

---

## Bundled Skills Catalog

Hermes ships with **79 bundled skills** copied into `~/.hermes/skills/` on install. These are broadly useful to most users.

### Apple (4 skills)
**Category**: macOS-specific automation  
**Platform**: macOS only

| Skill | Description | Path |
|-------|-------------|------|
| `apple-notes` | Manage Apple Notes via memo CLI (create, view, search, edit) | `apple/apple-notes` |
| `apple-reminders` | Manage Apple Reminders via remindctl CLI (list, add, complete, delete) | `apple/apple-reminders` |
| `findmy` | Track Apple devices and AirTags via FindMy.app using AppleScript | `apple/findmy` |
| `imessage` | Send and receive iMessages/SMS via imsg CLI on macOS | `apple/imessage` |

### Autonomous AI Agents (4 skills)
**Category**: Multi-agent orchestration and delegation  
**Purpose**: Spawn and coordinate autonomous AI agents

| Skill | Description | Path |
|-------|-------------|------|
| `claude-code` | Delegate coding tasks to Claude Code (Anthropic's CLI agent). Use for building features, refactoring, PR reviews. Requires claude CLI. | `autonomous-ai-agents/claude-code` |
| `codex` | Delegate coding tasks to OpenAI Codex CLI agent. Use for building features, refactoring, PR reviews, batch issue fixing. Requires codex CLI and git repo. | `autonomous-ai-agents/codex` |
| `hermes-agent-spawning` | Spawn additional Hermes Agent instances as autonomous subprocesses for independent long-running tasks. Supports non-interactive (-q) and interactive PTY modes. | `autonomous-ai-agents/hermes-agent` |
| `opencode` | Delegate coding tasks to OpenCode CLI agent for feature implementation, refactoring, PR review, long-running sessions. Requires opencode CLI. | `autonomous-ai-agents/opencode` |

### Data Science (1 skill)
**Category**: Interactive exploration and analysis

| Skill | Description | Path |
|-------|-------------|------|
| `jupyter-live-kernel` | Use live Jupyter kernel for stateful, iterative Python execution via hamelnb. Load when task involves exploration, iteration, or inspecting intermediate results. | `data-science/jupyter-live-kernel` |

### Creative (4 skills)
**Category**: Visual content generation

| Skill | Description | Path |
|-------|-------------|------|
| `ascii-art` | Generate ASCII art using pyfiglet (571 fonts), cowsay, boxes, toilet, image-to-ascii, remote APIs. No API keys required. | `creative/ascii-art` |
| `ascii-video` | Production pipeline for ASCII art video — any format. Converts video/audio/images/generative input into colored ASCII character video (MP4, GIF, image sequence). | `creative/ascii-video` |
| `excalidraw` | Create hand-drawn style diagrams using Excalidraw JSON format. Generate .excalidraw files for architecture diagrams, flowcharts, sequence diagrams. | `creative/excalidraw` |
| `p5js` | Production pipeline for interactive and generative visual art using p5.js. Create sketches, render to images/video via headless browser, serve live previews. | `creative/p5js` |

### DevOps (1 skill)
**Category**: Infrastructure automation

| Skill | Description | Path |
|-------|-------------|------|
| `webhook-subscriptions` | Create and manage webhook subscriptions for event-driven agent activation. External services (GitHub, Stripe, CI/CD, IoT) POST events to trigger agent runs. | `devops/webhook-subscriptions` |

### Dogfood (2 skills)
**Category**: QA and setup

| Skill | Description | Path |
|-------|-------------|------|
| `dogfood` | Systematic exploratory QA testing of web applications — find bugs, capture evidence, generate structured reports. | `dogfood/dogfood` |
| `hermes-agent-setup` | Help users configure Hermes Agent — CLI usage, setup wizard, model/provider selection, tools, skills, voice/STT/TTS, gateway, troubleshooting. | `dogfood/hermes-agent-setup` |

### Email (1 skill)
**Category**: Email management

| Skill | Description | Path |
|-------|-------------|------|
| `himalaya` | CLI to manage emails via IMAP/SMTP. List, read, write, reply, forward, search, organize emails. Supports multiple accounts and MML composition. | `email/himalaya` |

### Gaming (2 skills)
**Category**: Game server and automation

| Skill | Description | Path |
|-------|-------------|------|
| `minecraft-modpack-server` | Set up modded Minecraft server from CurseForge/Modrinth server pack zip. Covers NeoForge/Forge install, Java version, JVM tuning, firewall, LAN config, backups. | `gaming/minecraft-modpack-server` |
| `pokemon-player` | Play Pokemon games autonomously via headless emulation. Starts game server, reads structured game state from RAM, makes strategic decisions, sends button inputs. | `gaming/pokemon-player` |

### GitHub (6 skills)
**Category**: Repository and workflow management

| Skill | Description | Path |
|-------|-------------|------|
| `codebase-inspection` | Inspect and analyze codebases using pygount for LOC counting, language breakdown, code-vs-comment ratios. Use when checking lines of code, repo size, language composition. | `github/codebase-inspection` |
| `github-auth` | Set up GitHub authentication using git or gh CLI. Covers HTTPS tokens, SSH keys, credential helpers with automatic detection. | `github/github-auth` |
| `github-code-review` | Review code changes by analyzing git diffs, leaving inline comments on PRs, performing thorough pre-push review. Works with gh CLI or git + GitHub REST API. | `github/github-code-review` |
| `github-issues` | Create, manage, triage, close GitHub issues. Search existing issues, add labels, assign people, link to PRs. Works with gh CLI or git + GitHub REST API. | `github/github-issues` |
| `github-pr-workflow` | Full pull request lifecycle — create branches, commit changes, open PRs, monitor CI status, auto-fix failures, merge. Works with gh CLI or git + GitHub REST API. | `github/github-pr-workflow` |
| `github-repo-management` | Clone, create, fork, configure, manage GitHub repositories. Manage remotes, secrets, releases, workflows. Works with gh CLI or git + GitHub REST API. | `github/github-repo-management` |

### Inference.sh (1 skill)
**Category**: Cloud AI app execution

| Skill | Description | Path |
|-------|-------------|------|
| `inference-sh-cli` | Run 150+ AI apps via inference.sh CLI (infsh) — image generation, video creation, LLMs, search, 3D, social automation. | `inference-sh/cli` |

### Leisure (1 skill)
**Category**: Location and discovery

| Skill | Description | Path |
|-------|-------------|------|
| `find-nearby` | Find nearby places (restaurants, cafes, bars, pharmacies, etc.) using OpenStreetMap. Works with coordinates, addresses, cities, zip codes, Telegram location pins. No API keys. | `leisure/find-nearby` |

### MCP (2 skills)
**Category**: Model Context Protocol integration

| Skill | Description | Path |
|-------|-------------|------|
| `mcporter` | Use mcporter CLI to list, configure, auth, call MCP servers/tools directly (HTTP or stdio). Ad-hoc servers, config edits, CLI/type generation. | `mcp/mcporter` |
| `native-mcp` | Built-in MCP (Model Context Protocol) client connecting to external MCP servers, discovering tools, registering as native Hermes tools. Supports stdio and HTTP with auto-reconnection. | `mcp/native-mcp` |

### Media (4 skills)
**Category**: Content and audio

| Skill | Description | Path |
|-------|-------------|------|
| `gif-search` | Search and download GIFs from Tenor using curl. No dependencies beyond curl and jq. Useful for reaction GIFs, visual content, sending GIFs in chat. | `media/gif-search` |
| `heartmula` | Set up and run HeartMuLa, open-source music generation model family (Suno-like). Generates full songs from lyrics + tags with multilingual support. | `media/heartmula` |
| `songsee` | Generate spectrograms and audio feature visualizations (mel, chroma, MFCC, tempogram, etc.) from audio files. Useful for audio analysis, music production debugging. | `media/songsee` |
| `youtube-content` | Fetch YouTube video transcripts and transform into structured content (chapters, summaries, threads, blog posts). | `media/youtube-content` |

### MLOps (1 skill)
**Category**: General ML operations

| Skill | Description | Path |
|-------|-------------|------|
| `huggingface-hub` | Hugging Face Hub CLI (hf) — search, download, upload models and datasets, manage repos, deploy inference endpoints. | `mlops/huggingface-hub` |

### MLOps/Cloud (2 skills)
**Category**: GPU cloud providers

| Skill | Description | Path |
|-------|-------------|------|
| `lambda-labs-gpu-cloud` | Reserved and on-demand GPU cloud instances for ML training and inference. Use for dedicated GPU instances with simple SSH access, persistent filesystems, multi-node clusters. | `mlops/cloud/lambda-labs` |
| `modal-serverless-gpu` | Serverless GPU cloud platform for running ML workloads. Use for on-demand GPU access without infrastructure management, deploying ML models as APIs, batch jobs. | `mlops/cloud/modal` |

### MLOps/Evaluation (5 skills)
**Category**: Model evaluation and data curation

| Skill | Description | Path |
|-------|-------------|------|
| `evaluating-llms-harness` | Evaluates LLMs across 60+ academic benchmarks (MMLU, HumanEval, GSM8K, TruthfulQA, HellaSwag). Use when benchmarking model quality, comparing models, reporting academic results. | `mlops/evaluation/lm-evaluation-harness` |
| `huggingface-tokenizers` | Fast tokenizers optimized for research and production. Rust-based, tokenizes 1GB in <20 seconds. Supports BPE, WordPiece, Unigram. Train custom vocabularies, track alignments. | `mlops/evaluation/huggingface-tokenizers` |
| `nemo-curator` | GPU-accelerated data curation for LLM training. Supports text/image/video/audio. Features fuzzy deduplication (16× faster), quality filtering (30+ heuristics), semantic deduplication, PII redaction. | `mlops/evaluation/nemo-curator` |
| `sparse-autoencoder-training` | Guidance for training and analyzing Sparse Autoencoders (SAEs) using SAELens to decompose neural network activations into interpretable features. | `mlops/evaluation/saelens` |
| `weights-and-biases` | Track ML experiments with automatic logging, visualize training in real-time, optimize hyperparameters with sweeps, manage model registry with W&B. | `mlops/evaluation/weights-and-biases` |

### MLOps/Inference (8 skills)
**Category**: Model serving and optimization

| Skill | Description | Path |
|-------|-------------|------|
| `gguf-quantization` | GGUF format and llama.cpp quantization for efficient CPU/GPU inference. Use when deploying models on consumer hardware, Apple Silicon, or when needing flexible quantization (2-8 bit). | `mlops/inference/gguf` |
| `guidance` | Control LLM output with regex and grammars, guarantee valid JSON/XML/code generation, enforce structured formats, build multi-step workflows with Guidance. | `mlops/inference/guidance` |
| `instructor` | Extract structured data from LLM responses with Pydantic validation, retry failed extractions, parse complex JSON with type safety, stream partial results. | `mlops/inference/instructor` |
| `llama-cpp` | Runs LLM inference on CPU, Apple Silicon, consumer GPUs without NVIDIA hardware. Use for edge deployment, M1/M2/M3 Macs, AMD/Intel GPUs. Supports GGUF quantization (1.5-8 bit). | `mlops/inference/llama-cpp` |
| `obliteratus` | Remove refusal behaviors from open-weight LLMs using mechanistic interpretability techniques (diff-in-means, SVD, whitened SVD, LEACE, SAE decomposition). 9 CLI methods, 28 analysis modules. | `mlops/inference/obliteratus` |
| `outlines` | Guarantee valid JSON/XML/code structure during generation, use Pydantic models for type-safe outputs, support local models (Transformers, vLLM), maximize inference speed. | `mlops/inference/outlines` |
| `serving-llms-vllm` | Serves LLMs with high throughput using vLLM's PagedAttention and continuous batching. Use when deploying production LLM APIs, optimizing inference latency/throughput. | `mlops/inference/vllm` |
| `tensorrt-llm` | Optimizes LLM inference with NVIDIA TensorRT for maximum throughput and lowest latency. Use for production deployment on NVIDIA GPUs (A100/H100), 10-100x faster inference. | `mlops/inference/tensorrt-llm` |

### MLOps/Models (6 skills)
**Category**: Specific model architectures

| Skill | Description | Path |
|-------|-------------|------|
| `audiocraft-audio-generation` | PyTorch library for audio generation including text-to-music (MusicGen) and text-to-sound (AudioGen). Use when generating music from text, creating sound effects, melody-conditioned generation. | `mlops/models/audiocraft` |
| `clip` | OpenAI's model connecting vision and language. Enables zero-shot image classification, image-text matching, cross-modal retrieval. Trained on 400M image-text pairs. | `mlops/models/clip` |
| `llava` | Large Language and Vision Assistant. Enables visual instruction tuning and image-based conversations. Combines CLIP vision encoder with Vicuna/LLaMA language models. | `mlops/models/llava` |
| `segment-anything-model` | Foundation model for image segmentation with zero-shot transfer. Use when segmenting any object in images using points, boxes, masks as prompts. | `mlops/models/segment-anything` |
| `stable-diffusion-image-generation` | State-of-the-art text-to-image generation with Stable Diffusion models via HuggingFace Diffusers. Use when generating images from text, image-to-image translation, inpainting. | `mlops/models/stable-diffusion` |
| `whisper` | OpenAI's general-purpose speech recognition model. Supports 99 languages, transcription, translation to English, language identification. Six model sizes (39M-1550M params). | `mlops/models/whisper` |

### MLOps/Research (1 skill)
**Category**: ML research frameworks

| Skill | Description | Path |
|-------|-------------|------|
| `dspy` | Build complex AI systems with declarative programming, optimize prompts automatically, create modular RAG systems and agents with DSPy - Stanford NLP's framework. | `mlops/research/dspy` |

### MLOps/Training (14 skills)
**Category**: Fine-tuning and distributed training

| Skill | Description | Path |
|-------|-------------|------|
| `axolotl` | Expert guidance for fine-tuning LLMs with Axolotl - YAML configs, 100+ models, LoRA/QLoRA, DPO/KTO/ORPO/GRPO, multimodal support. | `mlops/training/axolotl` |
| `distributed-llm-pretraining-torchtitan` | PyTorch-native distributed LLM pretraining using torchtitan with 4D parallelism (FSDP2, TP, PP, CP). Use when pretraining Llama 3.1, DeepSeek V3, custom models at scale. | `mlops/training/torchtitan` |
| `fine-tuning-with-trl` | Fine-tune LLMs using reinforcement learning with TRL - SFT for instruction tuning, DPO for preference alignment, PPO/GRPO for reward optimization. | `mlops/training/trl-fine-tuning` |
| `grpo-rl-training` | Expert guidance for GRPO/RL fine-tuning with TRL for reasoning and task-specific model training. | `mlops/training/grpo-rl-training` |
| `hermes-atropos-environments` | Build, test, debug Hermes Agent RL environments for Atropos training. Covers HermesAgentBaseEnv interface, reward functions, agent loop integration, wandb logging. | `mlops/training/hermes-atropos-environments` |
| `huggingface-accelerate` | Simplest distributed training API. 4 lines to add distributed support to any PyTorch script. Unified API for DeepSpeed/FSDP/Megatron/DDP. | `mlops/training/accelerate` |
| `optimizing-attention-flash` | Optimizes transformer attention with Flash Attention for 2-4x speedup and 10-20x memory reduction. Use when training/running transformers with long sequences. | `mlops/training/flash-attention` |
| `peft-fine-tuning` | Parameter-efficient fine-tuning for LLMs using LoRA, QLoRA, and 25+ methods. Use when fine-tuning large models (7B-70B) with limited GPU memory. | `mlops/training/peft` |
| `pytorch-fsdp` | Expert guidance for Fully Sharded Data Parallel training with PyTorch FSDP - parameter sharding, mixed precision, CPU offloading, FSDP2. | `mlops/training/pytorch-fsdp` |
| `pytorch-lightning` | High-level PyTorch framework with Trainer class, automatic distributed training (DDP/FSDP/DeepSpeed), callbacks system. Scales from laptop to supercomputer. | `mlops/training/pytorch-lightning` |
| `simpo-training` | Simple Preference Optimization for LLM alignment. Reference-free alternative to DPO with better performance (+6.4 points on AlpacaEval 2.0). | `mlops/training/simpo` |
| `slime-rl-training` | Guidance for LLM post-training with RL using slime, a Megatron+SGLang framework. Use when training GLM models, implementing custom data generation workflows. | `mlops/training/slime` |
| `unsloth` | Expert guidance for fast fine-tuning with Unsloth - 2-5x faster training, 50-80% less memory, LoRA/QLoRA optimization. | `mlops/training/unsloth` |

### MLOps/Vector Databases (4 skills)
**Category**: Vector similarity search

| Skill | Description | Path |
|-------|-------------|------|
| `chroma` | Open-source embedding database for AI applications. Store embeddings and metadata, perform vector and full-text search, filter by metadata. Simple 4-function API. | `mlops/vector-databases/chroma` |
| `faiss` | Facebook's library for efficient similarity search and clustering of dense vectors. Supports billions of vectors, GPU acceleration, various index types. | `mlops/vector-databases/faiss` |
| `pinecone` | Managed vector database for production AI applications. Fully managed, auto-scaling, hybrid search (dense + sparse), metadata filtering, namespaces. | `mlops/vector-databases/pinecone` |
| `qdrant-vector-search` | High-performance vector similarity search engine for RAG and semantic search. Use when building production RAG systems requiring fast nearest neighbor search. | `mlops/vector-databases/qdrant` |

### Note-Taking (1 skill)
**Category**: Knowledge management

| Skill | Description | Path |
|-------|-------------|------|
| `obsidian` | Read, search, create notes in the Obsidian vault. | `note-taking/obsidian` |

### Productivity (5 skills)
**Category**: Document and workflow management

| Skill | Description | Path |
|-------|-------------|------|
| `google-workspace` | Gmail, Calendar, Drive, Contacts, Sheets, Docs integration via Python. Uses OAuth2 with automatic token refresh. No external binaries needed. | `productivity/google-workspace` |
| `linear` | Manage Linear issues, projects, teams via GraphQL API. Create, update, search, organize issues. Uses API key authentication. | `productivity/linear` |
| `nano-pdf` | Edit PDFs with natural-language instructions using nano-pdf CLI. Modify text, fix typos, update titles, make content changes to specific pages. | `productivity/nano-pdf` |
| `notion` | Notion API for creating and managing pages, databases, blocks via curl. Search, create, update, query Notion workspaces directly from terminal. | `productivity/notion` |
| `ocr-and-documents` | Extract text from PDFs and scanned documents. Use web_extract for remote URLs, pymupdf for local text-based PDFs, marker-pdf for OCR/scanned docs. | `productivity/ocr-and-documents` |
| `powerpoint` | Use any time a .pptx file is involved — creating slide decks, pitch decks, presentations; reading, parsing, extracting text from .pptx files. | `productivity/powerpoint` |

### Research (7 skills)
**Category**: Academic and domain research

| Skill | Description | Path |
|-------|-------------|------|
| `arxiv` | Search and retrieve academic papers from arXiv using free REST API. No API key needed. Search by keyword, author, category, or ID. | `research/arxiv` |
| `blogwatcher` | Monitor blogs and RSS/Atom feeds for updates using blogwatcher CLI. Add blogs, scan for new articles, track what you've read. | `research/blogwatcher` |
| `llm-wiki` | Karpathy's LLM Wiki — build and maintain persistent, interlinked markdown knowledge base. Ingest sources, query compiled knowledge, lint for consistency. | `research/llm-wiki` |
| `domain-intel` | Passive domain reconnaissance using Python stdlib. Subdomain discovery, SSL certificate inspection, WHOIS lookups, DNS records, domain availability checks. | `research/domain-intel` |
| `duckduckgo-search` | Free web search via DuckDuckGo — text, news, images, videos. No API key needed. Prefer ddgs CLI when installed. | `research/duckduckgo-search` |
| `ml-paper-writing` | Write publication-ready ML/AI papers for NeurIPS, ICML, ICLR, ACL, AAAI, COLM. Includes LaTeX templates, reviewer guidelines, citation verification. | `research/ml-paper-writing` |
| `polymarket` | Query Polymarket prediction market data — search markets, get prices, orderbooks, price history. Read-only via public REST APIs. | `research/polymarket` |

### Red-Teaming (1 skill)
**Category**: LLM safety research

| Skill | Description | Path |
|-------|-------------|------|
| `godmode` | Jailbreak API-served LLMs using G0DM0D3 techniques — Parseltongue input obfuscation (33 techniques), GODMODE CLASSIC system prompt templates, ULTRAPLINIAN multi-model racing. | `red-teaming/godmode` |

### Smart Home (1 skill)
**Category**: Home automation

| Skill | Description | Path |
|-------|-------------|------|
| `openhue` | Control Philips Hue lights, rooms, scenes via OpenHue CLI. Turn lights on/off, adjust brightness, color, color temperature, activate scenes. | `smart-home/openhue` |

### Social Media (1 skill)
**Category**: Social platform interaction

| Skill | Description | Path |
|-------|-------------|------|
| `xitter` | Interact with X/Twitter via x-cli terminal client using official X API credentials. | `social-media/xitter` |

### Software Development (7 skills)
**Category**: Development workflows

| Skill | Description | Path |
|-------|-------------|------|
| `code-review` | Guidelines for performing thorough code reviews with security and quality focus. | `software-development/code-review` |
| `plan` | Plan mode for Hermes — inspect context, write markdown plan into `.hermes/plans/`, do not execute work. | `software-development/plan` |
| `requesting-code-review` | Use when completing tasks, implementing major features, or before merging. Validates work meets requirements through systematic review. | `software-development/requesting-code-review` |
| `subagent-driven-development` | Use when executing implementation plans with independent tasks. Dispatches fresh delegate_task per task with two-stage review. | `software-development/subagent-driven-development` |
| `systematic-debugging` | Use when encountering any bug, test failure, unexpected behavior. 4-phase root cause investigation — NO fixes without understanding problem first. | `software-development/systematic-debugging` |
| `test-driven-development` | Use when implementing any feature or bugfix, before writing implementation code. Enforces RED-GREEN-REFACTOR cycle with test-first approach. | `software-development/test-driven-development` |
| `writing-plans` | Use when you have a spec or requirements for multi-step task. Creates comprehensive implementation plans with bite-sized tasks, exact file paths, code examples. | `software-development/writing-plans` |

---

## Optional Skills Catalog

Hermes ships with **22 optional skills** in `optional-skills/` that are **not active by default**. They cover heavier or niche use cases.

**Installation**: `hermes skills install official/<category>/<skill>`

### Autonomous AI Agents (1 skill)

| Skill | Description | Path |
|-------|-------------|------|
| `blackbox` | Delegate coding tasks to Blackbox AI CLI agent. Multi-model agent with built-in judge that runs tasks through multiple LLMs and picks best result. Requires blackbox CLI and API key. | `autonomous-ai-agents/blackbox` |

### Blockchain (2 skills)

| Skill | Description | Path |
|-------|-------------|------|
| `base` | Query Base (Ethereum L2) blockchain data with USD pricing — wallet balances, token info, transaction details, gas analysis, contract inspection, whale detection. Uses Base RPC + CoinGecko. | `blockchain/base` |
| `solana` | Query Solana blockchain data with USD pricing — wallet balances, token portfolios with values, transaction details, NFTs, whale detection, live network stats. Uses Solana RPC + CoinGecko. | `blockchain/solana` |

### Creative (2 skills)

| Skill | Description | Path |
|-------|-------------|------|
| `blender-mcp` | Control Blender directly from Hermes via socket connection to blender-mcp addon. Create 3D objects, materials, animations, run arbitrary Blender Python (bpy) code. | `creative/blender-mcp` |
| `meme-generation` | Generate real meme images by picking a template and overlaying text with Pillow. Produces actual .png meme files. | `creative/meme-generation` |

### DevOps (1 skill)

| Skill | Description | Path |
|-------|-------------|------|
| `docker-management` | Manage Docker containers, images, volumes, networks, Compose stacks — lifecycle ops, debugging, cleanup, Dockerfile optimization. | `devops/docker-management` |

### Email (1 skill)

| Skill | Description | Path |
|-------|-------------|------|
| `agentmail` | Give the agent its own dedicated email inbox via AgentMail. Send, receive, manage email autonomously using agent-owned email addresses (e.g. hermes-agent@agentmail.to). | `email/agentmail` |

### Health (1 skill)

| Skill | Description | Path |
|-------|-------------|------|
| `neuroskill-bci` | Connect to running NeuroSkill instance and incorporate user's real-time cognitive and emotional state (focus, relaxation, mood, cognitive load, drowsiness, heart rate, HRV, sleep staging, 40+ EXG scores). Requires BCI wearable (Muse 2/S or OpenBCI). | `health/neuroskill-bci` |

### MCP (1 skill)

| Skill | Description | Path |
|-------|-------------|------|
| `fastmcp` | Build, test, inspect, install, deploy MCP servers with FastMCP in Python. Use when creating new MCP server, wrapping API/database as MCP tools, exposing resources/prompts. | `mcp/fastmcp` |

### Migration (1 skill)

| Skill | Description | Path |
|-------|-------------|------|
| `openclaw-migration` | Migrate user's OpenClaw customization footprint into Hermes Agent. Imports Hermes-compatible memories, SOUL.md, command allowlists, user skills, workspace assets from ~/.openclaw. | `migration/openclaw-migration` |

### Productivity (1 skill)

| Skill | Description | Path |
|-------|-------------|------|
| `telephony` | Give Hermes phone capabilities — provision and persist Twilio number, send/receive SMS/MMS, make direct calls, place AI-driven outbound calls through Bland.ai or Vapi. | `productivity/telephony` |

### Research (2 skills)

| Skill | Description | Path |
|-------|-------------|------|
| `bioinformatics` | Gateway to 400+ bioinformatics skills from bioSkills and ClawBio. Covers genomics, transcriptomics, single-cell, variant calling, pharmacogenomics, metagenomics, structural biology. | `research/bioinformatics` |
| `qmd` | Search personal knowledge bases, notes, docs, meeting transcripts locally using qmd — hybrid retrieval engine with BM25, vector search, LLM reranking. Supports CLI and MCP integration. | `research/qmd` |

### Security (3 skills)

| Skill | Description | Path |
|-------|-------------|------|
| `1password` | Set up and use 1Password CLI (op). Use when installing CLI, enabling desktop app integration, signing in, reading/injecting secrets for commands. | `security/1password` |
| `oss-forensics` | Supply chain investigation, evidence recovery, forensic analysis for GitHub repositories. Covers deleted commit recovery, force-push detection, IOC extraction, multi-source evidence collection. | `security/oss-forensics` |
| `sherlock` | OSINT username search across 400+ social networks. Hunt down social media accounts by username. | `security/sherlock` |

---

## SKILL.md Format & Structure

All skills follow the **agentskills.io** open standard with YAML frontmatter and Markdown content.

### Complete Template

```markdown
---
name: my-skill
description: Brief description of what this skill does (shown in search results)
version: 1.0.0
author: Your Name
license: MIT
platforms: [macos, linux]     # Optional — restrict to specific OS platforms
                              # Valid: macos, linux, windows
                              # Omit to load on all platforms (default)
metadata:
  hermes:
    tags: [python, automation, devops]
    category: devops
    related_skills: [other-skill-name]
    fallback_for_toolsets: [web]      # Optional — conditional activation
    requires_toolsets: [terminal]     # Optional — conditional activation
    fallback_for_tools: [web_search]  # Optional — conditional activation
    requires_tools: [terminal]        # Optional — conditional activation
    config:                           # Optional — config.yaml settings
      - key: my.setting
        description: "What this controls"
        default: "value"
        prompt: "Prompt for setup"
    required_environment_variables:   # Optional — env vars the skill needs
      - name: MY_API_KEY
        prompt: "Enter your API key"
        help: "Get one at https://example.com"
        required_for: "API access"
---

# Skill Title

Brief introduction explaining what this skill does and when to use it.

## When to Use

Trigger conditions — when should the agent load this skill?

- Specific problem types
- Use cases
- Prerequisites

## Quick Reference

Table of common commands or patterns.

| Command | Purpose |
|---------|---------|
| `command` | What it does |

## Procedure

Step-by-step instructions for the main workflow.

1. First step
2. Second step
3. Third step

### Substep Details

Additional details for complex steps.

## Pitfalls

Known failure modes and how to fix them.

- **Pitfall 1**: Description and fix
- **Pitfall 2**: Description and fix

## Verification

How to confirm the skill worked correctly.

- Check that [result]
- Verify [condition]

## Advanced Usage

Optional section for power users.

## References

- [Link 1](https://example.com)
- [Link 2](https://example.com)
```

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

---

## Skill Metadata & Configuration

### Frontmatter Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✓ | Unique skill identifier (lowercase, hyphens) |
| `description` | string | ✓ | Brief description (shown in search results) |
| `version` | string | ✓ | Semantic version (e.g., 1.0.0) |
| `author` | string | | Author name |
| `license` | string | | License (MIT, Apache-2.0, etc.) |
| `platforms` | array | | Restrict to OS: `[macos]`, `[linux]`, `[windows]`, `[macos, linux]` |
| `metadata.hermes.tags` | array | | Tags for categorization and search |
| `metadata.hermes.category` | string | | Category (devops, mlops, research, etc.) |
| `metadata.hermes.related_skills` | array | | Related skill names |
| `metadata.hermes.fallback_for_toolsets` | array | | Show ONLY when these toolsets unavailable |
| `metadata.hermes.requires_toolsets` | array | | Show ONLY when these toolsets available |
| `metadata.hermes.fallback_for_tools` | array | | Show ONLY when these tools unavailable |
| `metadata.hermes.requires_tools` | array | | Show ONLY when these tools available |
| `metadata.hermes.config` | array | | Config settings (key, description, default, prompt) |
| `metadata.hermes.required_environment_variables` | array | | Required env vars (name, prompt, help, required_for) |

### Platform-Specific Skills

Restrict skills to specific operating systems:

```yaml
platforms: [macos]            # macOS only
platforms: [linux]            # Linux only
platforms: [windows]          # Windows only
platforms: [macos, linux]     # macOS and Linux
# Omit to load on all platforms (default)
```

**Examples**:
- `apple-notes`, `apple-reminders`, `findmy`, `imessage` — macOS only
- Most other skills — all platforms

### Conditional Activation (Fallback Skills)

Skills can show/hide based on available tools:

```yaml
metadata:
  hermes:
    fallback_for_toolsets: [web]      # Show ONLY when web toolset unavailable
    requires_toolsets: [terminal]     # Show ONLY when terminal toolset available
    fallback_for_tools: [web_search]  # Show ONLY when web_search tool unavailable
    requires_tools: [terminal]        # Show ONLY when terminal tool available
```

**Example**: `duckduckgo-search` uses `fallback_for_toolsets: [web]`
- When `FIRECRAWL_API_KEY` is set → web toolset available → DuckDuckGo skill hidden
- When API key missing → web toolset unavailable → DuckDuckGo skill shown as fallback

### Config Settings

Skills can declare non-secret config stored in `config.yaml`:

```yaml
metadata:
  hermes:
    config:
      - key: myplugin.path
        description: Path to plugin data directory
        default: "~/myplugin-data"
        prompt: Plugin data directory path
```

Settings stored under `skills.config` in `config.yaml`. When skill loads, resolved config values injected into context.

### Required Environment Variables

Skills can declare required env vars without disappearing from discovery:

```yaml
metadata:
  hermes:
    required_environment_variables:
      - name: TENOR_API_KEY
        prompt: Tenor API key
        help: Get a key from https://developers.google.com/tenor
        required_for: full functionality
```

When missing value encountered, Hermes asks for it securely only when skill actually loaded in local CLI. Messaging surfaces never ask for secrets in chat.

Once set, declared env vars automatically passed through to `execute_code` and `terminal` sandboxes.

---

## Skill Installation Sources

### 1. Official Optional Skills (`official`)

Maintained in Hermes repository, install with builtin trust.

```bash
hermes skills install official/security/1password
hermes skills install official/blockchain/solana
hermes skills install official/research/bioinformatics
```

**Catalog**: [Official Optional Skills Catalog](/docs/reference/optional-skills-catalog)

### 2. skills.sh (`skills-sh`)

Vercel's public skills directory. Searchable, inspectable, installable.

```bash
hermes skills search react --source skills-sh
hermes skills inspect skills-sh/vercel-labs/json-render/json-render-react
hermes skills install skills-sh/vercel-labs/json-render/json-render-react --force
```

**Directory**: [skills.sh](https://skills.sh/)

### 3. Well-Known Skill Endpoints (`well-known`)

URL-based discovery from sites publishing `/.well-known/skills/index.json`.

```bash
hermes skills search https://mintlify.com/docs --source well-known
hermes skills inspect well-known:https://mintlify.com/docs/.well-known/skills/mintlify
hermes skills install well-known:https://mintlify.com/docs/.well-known/skills/mintlify
```

**Example**: [Mintlify docs skills](https://mintlify.com/docs/.well-known/skills/index.json)

### 4. Direct GitHub (`github`)

Install directly from GitHub repositories and custom taps.

```bash
hermes skills install openai/skills/k8s
hermes skills install anthropics/skills/pdf
hermes skills tap add myorg/skills-repo
```

**Default Taps**:
- `openai/skills`
- `anthropics/skills`
- `VoltAgent/awesome-agent-skills`
- `garrytan/gstack`

### 5. ClawHub (`clawhub`)

Third-party skills marketplace.

```bash
hermes skills search something --source clawhub
```

**Site**: [clawhub.ai](https://clawhub.ai/)

### 6. LobeHub (`lobehub`)

Search and convert agent entries from LobeHub's public catalog.

```bash
hermes skills search something --source lobehub
```

**Site**: [LobeHub](https://lobehub.com/)  
**Agents Index**: [chat-agents.lobehub.com](https://chat-agents.lobehub.com/)

### 7. Claude Marketplace (`claude-marketplace`)

Marketplace repos publishing Claude-compatible manifests.

**Known Sources**:
- `anthropics/skills`
- `aiskillstore/marketplace`

---

## Skill Management Commands

### Browse & Search

```bash
# Browse all hub skills (official first)
hermes skills browse

# Browse only official optional skills
hermes skills browse --source official

# Search all sources
hermes skills search kubernetes

# Search specific source
hermes skills search react --source skills-sh
hermes skills search https://mintlify.com/docs --source well-known

# Preview before installing
hermes skills inspect openai/skills/k8s
hermes skills inspect skills-sh/vercel-labs/json-render/json-render-react
```

### Install & Manage

```bash
# Install from official optional skills
hermes skills install official/security/1password

# Install from skills.sh
hermes skills install skills-sh/vercel-labs/json-render/json-render-react --force

# Install from well-known endpoint
hermes skills install well-known:https://mintlify.com/docs/.well-known/skills/mintlify

# Install from GitHub
hermes skills install openai/skills/k8s

# List installed hub skills
hermes skills list --source hub

# Check for upstream updates
hermes skills check

# Update hub skills with upstream changes
hermes skills update

# Update specific skill
hermes skills update react

# Re-scan all hub skills for security
hermes skills audit

# Uninstall a hub skill
hermes skills uninstall k8s
```

### Bundled Skill Management

```bash
# Reset bundled skill (clear manifest, preserve local copy)
hermes skills reset google-workspace

# Full restore (delete local copy, re-copy bundled version)
hermes skills reset google-workspace --restore

# Non-interactive (skip confirmation)
hermes skills reset google-workspace --restore --yes
```

### Publishing & Sharing

```bash
# Publish skill to GitHub
hermes skills publish skills/my-skill --to github --repo owner/repo

# Add custom GitHub source
hermes skills tap add myorg/skills-repo

# Export skill configuration
hermes skills snapshot export setup.json
```

### Slash Commands (In Chat)

All commands work with `/skills`:

```bash
/skills browse
/skills search react --source skills-sh
/skills search https://mintlify.com/docs --source well-known
/skills inspect skills-sh/vercel-labs/json-render/json-render-react
/skills install openai/skills/skill-creator --force
/skills check
/skills update
/skills reset google-workspace
/skills list
```

---

## Skills as Slash Commands

Every installed skill automatically becomes a slash command:

### Usage

```bash
# In CLI or any messaging platform:
/gif-search funny cats
/axolotl help me fine-tune Llama 3 on my dataset
/github-pr-workflow create a PR for the auth refactor
/plan design a rollout for migrating our auth provider
/excalidraw
```

### How It Works

1. Skill name becomes command: `gif-search` → `/gif-search`
2. Arguments passed to skill: `/gif-search funny cats` → skill receives "funny cats"
3. Skill content loaded on demand
4. Agent executes skill instructions

### Custom Behavior

Some skills have custom behavior:

- **`/plan`**: Writes markdown plan to `.hermes/plans/` instead of executing
- **`/skills`**: Manages skill installation, search, updates
- **`/tools`**: Manages tool enable/disable
- **`/cron`**: Manages scheduled tasks

### Prefix Matching

Commands support prefix matching:

```bash
/h          → /help
/mod        → /model
/gif        → /gif-search
```

When prefix is ambiguous, Hermes shows options.

---

## Progressive Disclosure Pattern

Skills use token-efficient loading:

### Three Levels

```
Level 0: skills_list()           → [{name, description, category}, ...]   (~3k tokens)
Level 1: skill_view(name)        → Full content + metadata                 (varies)
Level 2: skill_view(name, path)  → Specific reference file                 (varies)
```

### How It Works

1. **Session Start**: Load only names and descriptions (~3k tokens for 40+ skills)
2. **On Demand**: Load full skill content when agent actually needs it
3. **Precision**: Load specific reference files for targeted information

### Benefits

- **Token Efficiency**: Avoid loading 40 skills at once
- **Fast Discovery**: Quick skill search and listing
- **Scalability**: Support 100+ skills without token bloat
- **Flexibility**: Agent loads only what it needs

---

## External Skill Directories

Point Hermes at additional skill directories outside `~/.hermes/skills/`.

### Configuration

Add `external_dirs` under `skills` section in `~/.hermes/config.yaml`:

```yaml
skills:
  external_dirs:
    - ~/.agents/skills
    - /home/shared/team-skills
    - ${SKILLS_REPO}/skills
```

Paths support `~` expansion and `${VAR}` environment variable substitution.

### How It Works

- **Read-Only**: External dirs scanned for discovery only
- **Local Precedence**: Local version wins if same skill exists in both
- **Full Integration**: External skills appear in system prompt, `skills_list`, `skill_view`, slash commands
- **Silent Skip**: Non-existent paths ignored without errors

### Example

```
~/.hermes/skills/               # Local (primary, read-write)
├── devops/deploy-k8s/
│   └── SKILL.md
└── mlops/axolotl/
    └── SKILL.md

~/.agents/skills/               # External (read-only, shared)
├── my-custom-workflow/
│   └── SKILL.md
└── team-conventions/
    └── SKILL.md
```

All four skills appear in skill index. Creating `my-custom-workflow` locally shadows external version.

---

## Agent-Managed Skills

Hermes can create, update, and delete its own skills via the `skill_manage` tool. This is the agent's **procedural memory**.

### When Agent Creates Skills

- After completing complex task (5+ tool calls) successfully
- When it hit errors/dead ends and found working path
- When user corrected its approach
- When it discovered non-trivial workflow

### Actions

| Action | Use For | Key Params |
|--------|---------|-----------|
| `create` | New skill from scratch | `name`, `content` (full SKILL.md), optional `category` |
| `patch` | Targeted fixes (preferred) | `name`, `old_string`, `new_string` |
| `edit` | Major structural rewrites | `name`, `content` (full SKILL.md replacement) |
| `delete` | Remove skill entirely | `name` |
| `write_file` | Add/update supporting files | `name`, `file_path`, `file_content` |
| `remove_file` | Remove supporting file | `name`, `file_path` |

### Best Practices

- **Use `patch` for updates**: More token-efficient than `edit`
- **Include verification**: Always add verification steps
- **Document pitfalls**: Include known failure modes
- **Test before saving**: Verify skill works before creating

### Example: Agent Creates Skill

```
User: "I need a workflow to deploy to Kubernetes with health checks"

Agent:
1. Researches Kubernetes deployment patterns
2. Tests deployment workflow
3. Discovers working approach
4. Creates skill via skill_manage:
   - name: k8s-deploy-with-health-checks
   - content: Full SKILL.md with procedure, pitfalls, verification
   - category: devops

Result: Skill saved to ~/.hermes/skills/devops/k8s-deploy-with-health-checks/SKILL.md
```

---

## Skills Hub & Registries

### Integrated Hubs

Hermes integrates with 7 skills ecosystems:

1. **Official** (`official`) — Hermes repo optional-skills
2. **skills.sh** (`skills-sh`) — Vercel's public directory
3. **Well-Known** (`well-known`) — URL-based discovery
4. **GitHub** (`github`) — Direct repo installs
5. **ClawHub** (`clawhub`) — Third-party marketplace
6. **LobeHub** (`lobehub`) — Agent catalog conversion
7. **Claude Marketplace** (`claude-marketplace`) — Marketplace repos

### Common Workflows

```bash
# Discover skills
hermes skills browse                    # All sources
hermes skills browse --source official  # Official only
hermes skills search kubernetes         # All sources
hermes skills search react --source skills-sh

# Preview before installing
hermes skills inspect openai/skills/k8s

# Install with security scanning
hermes skills install openai/skills/k8s
hermes skills install official/security/1password
hermes skills install skills-sh/vercel-labs/json-render/json-render-react --force

# Manage installed skills
hermes skills list --source hub
hermes skills check                     # Check for updates
hermes skills update                    # Update all
hermes skills audit                     # Re-scan for security
hermes skills uninstall k8s             # Remove skill
```

### Rate Limiting

Skills hub uses GitHub API (60 requests/hour unauthenticated).

**Solution**: Set `GITHUB_TOKEN` in `.env` to increase to 5,000 requests/hour.

```bash
# In ~/.hermes/.env
GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
```

---

## Security & Trust Levels

### Security Scanning

All hub-installed skills go through security scanner checking for:

- Data exfiltration
- Prompt injection
- Destructive commands
- Supply-chain signals
- Other threats

### Trust Levels

| Level | Source | Policy |
|-------|--------|--------|
| `builtin` | Ships with Hermes | Always trusted |
| `official` | `optional-skills/` in repo | Builtin trust, no third-party warning |
| `trusted` | `openai/skills`, `anthropics/skills` | More permissive policy |
| `community` | Everything else | Non-dangerous findings can be overridden with `--force` |

### Using `--force`

Override non-dangerous policy blocks:

```bash
hermes skills install skills-sh/anthropics/skills/pdf --force
```

**Important**:
- `--force` can override caution/warn-style findings
- `--force` does NOT override `dangerous` verdicts
- Official optional skills (`official/...`) treated as builtin trust

### Upstream Metadata

`hermes skills inspect` surfaces upstream metadata:

- Repo URL
- skills.sh detail page URL
- Install command
- Weekly installs
- Upstream security audit statuses
- Well-known index/endpoint URLs

---

## Summary

### Quick Reference

| Task | Command |
|------|---------|
| Browse all skills | `hermes skills browse` |
| Search skills | `hermes skills search <query>` |
| Install skill | `hermes skills install <id>` |
| List installed | `hermes skills list` |
| Check updates | `hermes skills check` |
| Update skills | `hermes skills update` |
| Uninstall skill | `hermes skills uninstall <name>` |
| Use skill | `/skill-name [args]` |
| Create skill | Agent uses `skill_manage` tool |
| Reset bundled | `hermes skills reset <name>` |

### Key Concepts

- **79 Bundled Skills**: Broadly useful, always available
- **22 Optional Skills**: Niche/heavy, install as needed
- **Progressive Disclosure**: Load only what's needed
- **Slash Commands**: Every skill becomes `/skill-name`
- **Agent-Managed**: Hermes creates skills from experience
- **7 Registries**: Multiple sources for discovery
- **Security Scanning**: All hub skills scanned
- **External Dirs**: Point to shared skill directories

---

**Last Updated**: April 18, 2026  
**Hermes Version**: v0.10.0  
**Documentation Version**: 2.0

