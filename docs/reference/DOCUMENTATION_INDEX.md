# Hermes Agent Tools Documentation - Complete Index

**Last Updated**: April 18, 2026  
**Total Documentation**: 6,655 lines  
**Coverage**: 40+ tools across 12 categories

---

## 📚 Documentation Files

### 1. README.md
**Entry Point for All Documentation**

- Main overview of reference documentation
- Quick start guide
- Tool categories overview
- Common tasks and examples
- Configuration reference
- API integration guide
- Troubleshooting section
- Best practices

**Start here if**: You're new to Hermes tools

---

### 2. TOOLS_INDEX.md
**Quick Reference for All Tools**

- Tools organized by category (12 categories)
- Tools organized by backend/provider
- Tools organized by use case
- Configuration quick reference
- Tool parameters reference
- Approval requirements
- Tool composition guide
- Tool discovery commands

**Use this for**: Quick lookups and finding the right tool

---

### 3. TOOLS_REFERENCE.md
**Complete Detailed Documentation**

**Browser Tools (10)**
- browser_navigate - Navigate to URL
- browser_click - Click element
- browser_fill - Fill form field
- browser_screenshot - Take screenshot
- browser_get_text - Extract text
- browser_get_html - Extract HTML
- browser_wait_for_selector - Wait for element
- browser_get_cookies - Get cookies
- browser_set_cookies - Set cookies
- browser_execute_script - Execute JavaScript

**Web Tools (2)**
- web_fetch - Fetch URL content
- web_parse - Parse HTML/JSON

**Terminal Tools (2)**
- terminal_execute - Execute shell command
- terminal_stream - Stream command output

**File Tools (4)**
- file_read - Read file
- file_write - Write file
- file_delete - Delete file
- file_list - List directory

**Vision & Media Tools (3)**
- vision_analyze_image - Analyze image
- media_convert - Convert media format
- media_extract_audio - Extract audio from video

**Code Execution (1)**
- code_execute - Execute code

**Orchestration Tools (4)**
- workflow_create - Create workflow
- workflow_execute - Execute workflow
- workflow_status - Get workflow status
- workflow_cancel - Cancel workflow

**Memory Tools (3)**
- memory_store - Store memory
- memory_retrieve - Retrieve memory
- memory_search - Search memory

**Scheduling Tools (2)**
- schedule_create - Create scheduled task
- schedule_cancel - Cancel scheduled task

**Home Assistant Tools (4)**
- ha_get_state - Get entity state
- ha_set_state - Set entity state
- ha_call_service - Call service
- ha_get_history - Get entity history

**RL Training Tools (10)**
- rl_train_model - Train RL model
- rl_evaluate_model - Evaluate RL model
- rl_predict - Get RL prediction
- rl_hyperparameter_tune - Tune hyperparameters
- rl_save_model - Save RL model
- rl_load_model - Load RL model
- rl_get_policy - Get model policy
- rl_compare_models - Compare models
- rl_export_model - Export model
- rl_monitor_training - Monitor training

**MCP Dynamic Tools**
- Filesystem MCP
- GitHub MCP
- Slack MCP
- Custom MCPs

**For each tool includes**:
- Function description
- Complete parameter specifications
- Return format examples
- Backend information
- Configuration options
- Disable/enable options

**Use this for**: Understanding how to use specific tools

---

### 4. TOOL_PARAMETERS_SCHEMA.md
**JSON Schema Definitions**

Complete JSON Schema for:
- All 40+ tools
- Input validation schemas
- Return value schemas
- Type definitions
- Constraints and validation rules

**Use this for**: API integration, validation, schema-driven development

---

### 5. SKILLS_COMPREHENSIVE_GUIDE.md
**Complete Skills Documentation**

- Skill system overview
- All available skills documented
- Skill parameters and usage
- Skill composition
- Best practices

**Use this for**: Understanding and using Hermes skills

---

### 6. SKILLS_QUICK_REFERENCE.md
**Quick Skills Reference**

- Skills by category
- Quick lookup table
- Common skill combinations
- Skill discovery commands

**Use this for**: Quick skill lookups

---

## 🎯 Quick Navigation

### By Task

**I want to...**

- **Scrape a website**
  → See: TOOLS_REFERENCE.md - Browser Tools
  → Use: browser_navigate, browser_get_text, browser_screenshot

- **Process data**
  → See: TOOLS_REFERENCE.md - File Tools, Code Execution
  → Use: file_read, code_execute, file_write

- **Automate tasks**
  → See: TOOLS_REFERENCE.md - Orchestration Tools, Scheduling Tools
  → Use: workflow_create, schedule_create

- **Control smart home**
  → See: TOOLS_REFERENCE.md - Home Assistant Tools
  → Use: ha_get_state, ha_set_state, ha_call_service

- **Train ML models**
  → See: TOOLS_REFERENCE.md - RL Training Tools
  → Use: rl_train_model, rl_evaluate_model

- **Analyze images**
  → See: TOOLS_REFERENCE.md - Vision & Media Tools
  → Use: vision_analyze_image

- **Convert media**
  → See: TOOLS_REFERENCE.md - Vision & Media Tools
  → Use: media_convert, media_extract_audio

- **Execute commands**
  → See: TOOLS_REFERENCE.md - Terminal Tools
  → Use: terminal_execute, terminal_stream

- **Fetch web content**
  → See: TOOLS_REFERENCE.md - Web Tools
  → Use: web_fetch, web_parse

- **Store information**
  → See: TOOLS_REFERENCE.md - Memory Tools
  → Use: memory_store, memory_retrieve, memory_search

### By Tool Type

- **Browser Automation** → TOOLS_REFERENCE.md - Browser Tools
- **Web Scraping** → TOOLS_REFERENCE.md - Web Tools
- **System Administration** → TOOLS_REFERENCE.md - Terminal Tools
- **File Operations** → TOOLS_REFERENCE.md - File Tools
- **Media Processing** → TOOLS_REFERENCE.md - Vision & Media Tools
- **Code Execution** → TOOLS_REFERENCE.md - Code Execution
- **Workflow Automation** → TOOLS_REFERENCE.md - Orchestration Tools
- **Memory Management** → TOOLS_REFERENCE.md - Memory Tools
- **Task Scheduling** → TOOLS_REFERENCE.md - Scheduling Tools
- **Smart Home** → TOOLS_REFERENCE.md - Home Assistant Tools
- **Machine Learning** → TOOLS_REFERENCE.md - RL Training Tools
- **Custom Tools** → TOOLS_REFERENCE.md - MCP Dynamic Tools

### By Backend

- **Playwright** → Browser Tools
- **HTTP Client** → Web Tools
- **subprocess/Docker/SSH** → Terminal Tools
- **File System** → File Tools
- **Claude/GPT-4V** → Vision Tools
- **FFmpeg** → Media Tools
- **Docker/Local** → Code Execution
- **Workflow Engine** → Orchestration Tools
- **Memory Store/Redis** → Memory Tools
- **APScheduler/Celery** → Scheduling Tools
- **Home Assistant API** → Smart Home Tools
- **Stable-Baselines3/Optuna** → RL Training Tools
- **MCP Servers** → Dynamic Tools

---

## 📖 Documentation Structure

### For Each Tool

1. **Function**: What the tool does
2. **Parameters**: Complete specification
   - Type (string, integer, boolean, array, object)
   - Description
   - Default values
   - Constraints (min/max, enum, pattern)
   - Examples
3. **Return Format**: Example JSON response
4. **Backend**: What backend is used
5. **Configuration**: How to configure
6. **Disable/Enable**: How to disable or enable

### For Each Category

1. **Overview**: What the category is for
2. **Tools List**: All tools in category
3. **Use Cases**: Common use cases
4. **Configuration**: Category-specific config
5. **Best Practices**: Best practices

---

## 🔍 Finding Information

### Quick Lookup
→ Use **TOOLS_INDEX.md**
- Tools by category
- Tools by backend
- Tools by use case
- Quick reference tables

### Detailed Information
→ Use **TOOLS_REFERENCE.md**
- Complete tool documentation
- Parameter specifications
- Return format examples
- Configuration options

### API Integration
→ Use **TOOL_PARAMETERS_SCHEMA.md**
- JSON schemas
- Input validation
- Return schemas
- Type definitions

### Getting Started
→ Use **README.md**
- Overview
- Quick start
- Common tasks
- Troubleshooting

---

## 📊 Documentation Statistics

| Metric | Value |
|--------|-------|
| Total Lines | 6,655 |
| Total Files | 6 |
| Tools Documented | 40+ |
| Categories | 12 |
| Parameters | 200+ |
| Return Formats | 40+ |
| Backends | 13+ |
| Configuration Options | 50+ |
| Code Examples | 100+ |
| JSON Schemas | 40+ |

---

## ✅ What's Documented

### Tools
- [x] All 40+ tools
- [x] All parameters
- [x] All return formats
- [x] All backends
- [x] All configurations

### Features
- [x] Tool enable/disable
- [x] Toolset compositions
- [x] Tool composition rules
- [x] Approval requirements
- [x] Best practices

### Guides
- [x] Quick reference
- [x] Detailed guide
- [x] API integration
- [x] Troubleshooting
- [x] Configuration

### Examples
- [x] Parameter examples
- [x] Return format examples
- [x] Configuration examples
- [x] Workflow examples
- [x] Use case examples

---

## 🚀 Getting Started

### Step 1: Understand Tools
→ Read: **README.md**

### Step 2: Find Your Tool
→ Use: **TOOLS_INDEX.md**

### Step 3: Learn Details
→ Read: **TOOLS_REFERENCE.md**

### Step 4: Integrate
→ Use: **TOOL_PARAMETERS_SCHEMA.md**

### Step 5: Configure
→ Read: **README.md** - Configuration section

---

## 🔗 Related Documentation

- **Configuration Reference**: [../configuration/CONFIG_REFERENCE.md](../configuration/CONFIG_REFERENCE.md)
- **Quick Start Guide**: [../guides/QUICK_START.md](../guides/QUICK_START.md)
- **Workflow Guide**: [../guides/WORKFLOW_GUIDE.md](../guides/WORKFLOW_GUIDE.md)
- **API Server Integration**: [../guides/API_SERVER_INTEGRATION.md](../guides/API_SERVER_INTEGRATION.md)

---

## 💡 Tips

### For Quick Lookups
- Use **TOOLS_INDEX.md** tables
- Use `hermes tools list` command
- Use `hermes tools info <tool_name>` command

### For Learning
- Start with **README.md**
- Read tool category overview
- Look at examples
- Try tools in order

### For Integration
- Use **TOOL_PARAMETERS_SCHEMA.md**
- Copy JSON schemas
- Validate inputs
- Test with examples

### For Troubleshooting
- Check **README.md** troubleshooting section
- Check tool configuration
- Check logs
- Check approval requirements

---

## 📝 Documentation Format

### Markdown
- All documentation in Markdown format
- Easy to read and edit
- Supports code blocks and tables
- Supports links and references

### JSON Schemas
- Complete JSON Schema definitions
- Validation rules
- Type definitions
- Constraints

### YAML Examples
- Configuration examples
- Workflow examples
- Toolset examples

### Code Examples
- Python examples
- Bash examples
- JavaScript examples
- YAML examples

---

## 🎓 Learning Path

### Beginner
1. Read README.md
2. Look at TOOLS_INDEX.md
3. Try simple tools
4. Read tool-specific documentation

### Intermediate
1. Read TOOLS_REFERENCE.md
2. Create workflows
3. Configure tools
4. Use toolsets

### Advanced
1. Use TOOL_PARAMETERS_SCHEMA.md
2. Integrate with API
3. Create custom tools
4. Optimize performance

---

## 🔄 Documentation Updates

### When to Update
- New tools added
- Tool parameters changed
- Backends changed
- Configurations changed
- Examples updated

### How to Update
1. Update relevant documentation file
2. Update index/reference files
3. Update examples
4. Update schemas

### Maintenance
- Regular reviews
- Fix errors
- Improve clarity
- Add examples

---

## 📞 Support

### Documentation Issues
- Report issues
- Suggest improvements
- Request clarifications

### Tool Issues
- Check troubleshooting section
- Check logs
- Check configuration
- Report bugs

### Feature Requests
- Suggest new tools
- Suggest improvements
- Suggest examples

---

## 📄 File Locations

```
docs/reference/
├── README.md                          (Main entry point)
├── TOOLS_INDEX.md                     (Quick reference)
├── TOOLS_REFERENCE.md                 (Complete documentation)
├── TOOL_PARAMETERS_SCHEMA.md          (JSON schemas)
├── SKILLS_COMPREHENSIVE_GUIDE.md      (Skills guide)
├── SKILLS_QUICK_REFERENCE.md          (Skills quick reference)
└── DOCUMENTATION_INDEX.md             (This file)
```

---

## 🎯 Summary

This documentation provides:

1. **Complete Coverage**: All 40+ tools documented
2. **Multiple Formats**: Quick reference, detailed guide, schemas
3. **Practical Examples**: Real-world usage examples
4. **Configuration Guide**: How to configure tools
5. **API Integration**: How to use tools via API
6. **Best Practices**: Best practices for using tools
7. **Troubleshooting**: Common issues and solutions
8. **Security**: Approval requirements and dangerous operations

---

**Created**: April 18, 2026  
**Version**: 1.0.0  
**Status**: Complete ✅

