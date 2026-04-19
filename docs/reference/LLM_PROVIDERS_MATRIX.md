# Hermes Agent - LLM Providers Matrix

**Last Updated:** April 2026

---

## Complete Provider Comparison Matrix

### Authentication & Setup

| Provider | Auth Type | Setup Complexity | Env Variable | Base URL | Notes |
|----------|-----------|------------------|--------------|----------|-------|
| openrouter | API Key | ⭐ Easy | `OPENROUTER_API_KEY` | `https://openrouter.ai/v1` | Recommended, 100+ models |
| anthropic | API Key | ⭐ Easy | `ANTHROPIC_API_KEY` | `https://api.anthropic.com` | Direct Claude access |
| gemini | API Key | ⭐ Easy | `GOOGLE_API_KEY` | `https://generativelanguage.googleapis.com` | Google AI Studio |
| copilot | GitHub Token | ⭐ Easy | `COPILOT_GITHUB_TOKEN` | `https://models.inference.ai.azure.com` | GitHub Models |
| nous | OAuth | ⭐⭐ Medium | — | Portal URL | `hermes login` required |
| nous-api | API Key | ⭐ Easy | `NOUS_API_KEY` | Portal API | Nous Portal API key |
| zai | API Key | ⭐ Easy | `GLM_API_KEY` | `https://open.bigmodel.cn` | ZhipuAI GLM |
| kimi-coding | API Key | ⭐ Easy | `KIMI_API_KEY` | `https://api.moonshot.cn` | Moonshot AI |
| kimi-coding-cn | API Key | ⭐ Easy | `KIMI_CN_API_KEY` | China endpoint | Moonshot China |
| minimax | API Key | ⭐ Easy | `MINIMAX_API_KEY` | `https://api.minimax.chat` | MiniMax global |
| minimax-cn | API Key | ⭐ Easy | `MINIMAX_CN_API_KEY` | China endpoint | MiniMax China |
| arcee | API Key | ⭐ Easy | `ARCEEAI_API_KEY` | `https://api.arcee.ai` | Arcee AI Trinity |
| deepseek | API Key | ⭐ Easy | `DEEPSEEK_API_KEY` | `https://api.deepseek.com` | DeepSeek API |
| huggingface | Token | ⭐ Easy | `HF_TOKEN` | HF Inference | Hugging Face |
| ollama-cloud | API Key | ⭐ Easy | `OLLAMA_API_KEY` | Cloud endpoint | Ollama Cloud |
| ai-gateway | API Key | ⭐ Easy | `AI_GATEWAY_API_KEY` | `https://api.vercel.ai` | Vercel AI Gateway |
| custom | API Key | ⭐⭐ Medium | `OPENAI_API_KEY` | Custom | Any OpenAI-compatible |
| ollama | Local | ⭐ Easy | — | `http://localhost:11434` | Local inference |
| lmstudio | Local | ⭐ Easy | — | `http://localhost:1234` | Local inference |
| vllm | Local | ⭐ Easy | — | `http://localhost:8000` | Local inference |
| llamacpp | Local | ⭐ Easy | — | `http://localhost:8080` | Local inference |

### Capabilities

| Provider | Vision | Reasoning | Tool Use | Streaming | Function Calling |
|----------|--------|-----------|----------|-----------|------------------|
| openrouter | ✓ | ✓ | ✓ | ✓ | ✓ |
| anthropic | ✓ | ✓ | ✓ | ✓ | ✓ |
| gemini | ✓ | ✓ | ✓ | ✓ | ✓ |
| copilot | ✓ | ✓ | ✓ | ✓ | ✓ |
| nous | ✓ | ✓ | ✓ | ✓ | ✓ |
| nous-api | ✓ | ✓ | ✓ | ✓ | ✓ |
| zai | ✓ | ✓ | ✓ | ✓ | ✓ |
| kimi-coding | ✓ | ✓ | ✓ | ✓ | ✓ |
| minimax | ✓ | ✓ | ✓ | ✓ | ✓ |
| arcee | ✓ | ✓ | ✓ | ✓ | ✓ |
| deepseek | ✓ | ✓ | ✓ | ✓ | ✓ |
| huggingface | ✓ | ✗ | ✓ | ✓ | ✗ |
| ollama-cloud | ✓ | ✗ | ✓ | ✓ | ✗ |
| ai-gateway | ✓ | ✓ | ✓ | ✓ | ✓ |
| custom | ✓ | ✓ | ✓ | ✓ | ✓ |
| ollama | ✓ | ✗ | ✓ | ✓ | ✗ |
| lmstudio | ✓ | ✗ | ✓ | ✓ | ✗ |
| vllm | ✓ | ✗ | ✓ | ✓ | ✗ |
| llamacpp | ✓ | ✗ | ✓ | ✓ | ✗ |

### Reasoning Effort Support

| Provider | none | minimal | low | medium | high | xhigh |
|----------|------|---------|-----|--------|------|-------|
| openrouter | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| anthropic | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| gemini | ✓ | ✓ | ✓ | ✓ | ✓ | ✗ |
| copilot | ✓ | ✓ | ✓ | ✓ | ✓ | ✗ |
| nous | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| deepseek | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| others | ✓ | ✗ | ✗ | ✗ | ✗ | ✗ |

### Performance Characteristics

| Provider | Latency | Throughput | Cost | Reliability |
|----------|---------|-----------|------|-------------|
| openrouter | Medium | High | Medium | ⭐⭐⭐⭐⭐ |
| anthropic | Medium | High | High | ⭐⭐⭐⭐⭐ |
| gemini | Low | Very High | Low | ⭐⭐⭐⭐⭐ |
| copilot | Low | High | Free/Low | ⭐⭐⭐⭐ |
| nous | Medium | High | Medium | ⭐⭐⭐⭐ |
| zai | Medium | High | Low | ⭐⭐⭐⭐ |
| kimi-coding | Medium | High | Low | ⭐⭐⭐⭐ |
| minimax | Medium | High | Low | ⭐⭐⭐⭐ |
| arcee | Medium | Medium | High | ⭐⭐⭐⭐ |
| deepseek | Low | High | Very Low | ⭐⭐⭐⭐ |
| huggingface | Medium | Medium | Low | ⭐⭐⭐ |
| ollama | Very Low | Medium | Free | ⭐⭐⭐ |
| lmstudio | Very Low | Medium | Free | ⭐⭐⭐ |
| vllm | Very Low | High | Free | ⭐⭐⭐ |
| llamacpp | Very Low | Low | Free | ⭐⭐⭐ |

### Model Availability

| Provider | Model Count | Latest Models | Open Source | Proprietary |
|----------|------------|---------------|-------------|------------|
| openrouter | 100+ | ✓ | ✓ | ✓ |
| anthropic | 4 | ✓ | ✗ | ✓ |
| gemini | 5+ | ✓ | ✗ | ✓ |
| copilot | 8+ | ✓ | ✓ | ✓ |
| nous | 5+ | ✓ | ✓ | ✓ |
| zai | 5+ | ✓ | ✗ | ✓ |
| kimi-coding | 3 | ✓ | ✗ | ✓ |
| minimax | 3+ | ✓ | ✗ | ✓ |
| arcee | 3+ | ✓ | ✗ | ✓ |
| deepseek | 3+ | ✓ | ✓ | ✓ |
| huggingface | 1000+ | ✓ | ✓ | ✓ |
| ollama | 100+ | ✓ | ✓ | ✗ |
| lmstudio | 1000+ | ✓ | ✓ | ✗ |
| vllm | 1000+ | ✓ | ✓ | ✗ |
| llamacpp | 1000+ | ✓ | ✓ | ✗ |

### Context Length Support

| Provider | Min | Max | Notes |
|----------|-----|-----|-------|
| openrouter | 1K | 200K+ | Varies by model |
| anthropic | 1K | 200K | Claude 3.5 Sonnet |
| gemini | 1K | 1M | Gemini 2.0 Flash |
| copilot | 1K | 128K | Varies by model |
| nous | 1K | 128K | Varies by model |
| zai | 1K | 128K | GLM series |
| kimi-coding | 1K | 128K | Moonshot series |
| minimax | 1K | 200K | MiniMax series |
| arcee | 1K | 128K | Trinity series |
| deepseek | 1K | 128K | DeepSeek series |
| huggingface | 1K | 32K | Varies by model |
| ollama | 1K | 32K | Varies by model |
| lmstudio | 1K | 32K | Varies by model |
| vllm | 1K | 32K | Varies by model |
| llamacpp | 1K | 32K | Varies by model |

### Pricing Comparison (Approximate)

| Provider | Input (1M tokens) | Output (1M tokens) | Notes |
|----------|-------------------|-------------------|-------|
| openrouter | $0.50-$15 | $1.50-$60 | Varies by model |
| anthropic | $3 | $15 | Claude 3.5 Sonnet |
| gemini | $0.075 | $0.30 | Flash model |
| copilot | Free-$20/mo | Included | GitHub Models |
| nous | $1-$5 | $3-$15 | Varies by model |
| zai | $0.10 | $0.30 | GLM series |
| kimi-coding | $0.30 | $1.00 | Moonshot series |
| minimax | $0.15 | $0.60 | MiniMax series |
| arcee | Custom | Custom | Enterprise pricing |
| deepseek | $0.14 | $0.28 | Very affordable |
| huggingface | $0.50-$5 | $1-$15 | Varies by model |
| ollama | Free | Free | Local only |
| lmstudio | Free | Free | Local only |
| vllm | Free | Free | Local only |
| llamacpp | Free | Free | Local only |

### Regional Availability

| Provider | Global | China | EU | Notes |
|----------|--------|-------|----|----|
| openrouter | ✓ | ✓ | ✓ | Worldwide |
| anthropic | ✓ | ✗ | ✓ | Limited China |
| gemini | ✓ | ✗ | ✓ | Limited China |
| copilot | ✓ | ✓ | ✓ | GitHub Models |
| nous | ✓ | ✓ | ✓ | Worldwide |
| zai | ✓ | ✓ | ✓ | China-optimized |
| kimi-coding | ✓ | ✓ | ✓ | China-optimized |
| minimax | ✓ | ✓ | ✓ | China-optimized |
| arcee | ✓ | ✗ | ✓ | Enterprise |
| deepseek | ✓ | ✓ | ✓ | China-optimized |
| huggingface | ✓ | ✓ | ✓ | Worldwide |
| ollama | Local | Local | Local | Self-hosted |
| lmstudio | Local | Local | Local | Self-hosted |
| vllm | Local | Local | Local | Self-hosted |
| llamacpp | Local | Local | Local | Self-hosted |

---

## Provider Selection Guide

### Best for Production
1. **OpenRouter** — 100+ models, price optimization, reliability
2. **Anthropic** — Direct Claude access, highest quality
3. **Google Gemini** — Fast, cheap, multimodal

### Best for Cost
1. **DeepSeek** — Extremely affordable
2. **Google Gemini Flash** — Fast and cheap
3. **Anthropic Claude Haiku** — Budget-friendly

### Best for Vision
1. **Google Gemini** — Excellent vision capabilities
2. **Anthropic Claude** — High-quality vision
3. **OpenRouter** — Access to multiple vision models

### Best for Reasoning
1. **Anthropic Claude** — Extended thinking
2. **DeepSeek** — Reasoning models
3. **OpenRouter** — Access to reasoning models

### Best for Local Development
1. **Ollama** — Easy setup, good models
2. **LM Studio** — User-friendly GUI
3. **vLLM** — High throughput

### Best for China
1. **ZhipuAI (z.ai)** — GLM models
2. **Kimi (Moonshot)** — Long context
3. **MiniMax** — Multimodal

### Best for Enterprise
1. **OpenRouter** — Flexible, scalable
2. **Anthropic** — Direct support
3. **Arcee** — Specialized models

### Best for Startups
1. **GitHub Copilot** — Free tier available
2. **Google Gemini** — Generous free tier
3. **DeepSeek** — Very affordable

---

## Quick Decision Tree

```
Do you want local inference?
├─ Yes → Ollama / LM Studio / vLLM
└─ No → Continue...

Do you need the best quality?
├─ Yes → Anthropic Claude / OpenRouter
└─ No → Continue...

Do you need to minimize cost?
├─ Yes → DeepSeek / Google Gemini Flash
└─ No → Continue...

Do you need 100+ models?
├─ Yes → OpenRouter
└─ No → Continue...

Do you need vision?
├─ Yes → Google Gemini / Anthropic Claude
└─ No → Continue...

Are you in China?
├─ Yes → ZhipuAI / Kimi / MiniMax
└─ No → OpenRouter / Anthropic / Gemini
```

---

**Last Updated:** April 2026  
**Hermes Version:** Latest (v2026+)
