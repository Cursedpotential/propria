# Chat Parser Project - Session State

**Last Updated:** 2025-12-21 00:48 UTC

## Current Status: READY FOR GEMINI MCP TEST

### What Was Completed

1. ✅ **Workflow diagrams created** → `C:\Users\matts\chat_parser_workflows.md`
   - Complete data flow architecture
   - Tool assignment decision trees (Python vs LLM)
   - Entity, event, artifact, document extraction workflows
   - Error handling, deduplication, incremental processing flows
   - Performance targets and optimization strategies

2. ✅ **Existing parser research** → Found open-source parsers:
   - **ChatGPT**: `chat-export-structurer`, `quantified_chatgpt`, `chatgpt-export-to-text`
   - **Claude**: `claude-conversation-extractor`, `claude-export`, `claude-to-markdown`
   - **Gemini**: Official CLI `/export` command, `gemini-pro-chat`

3. ✅ **Fixed Gemini MCP configuration**:
   - Old (broken): `@raydius/gemini-for-claude-mcp`
   - New (working): `@maxanatsko/gemini-mcp-tool` v2.3.0
   - File updated: `C:\Users\matts\.claude.json` line 122-131
   - **Requires restart** to load new MCP server

### Next Steps (After Restart)

1. **Test Gemini MCP** → Verify `@maxanatsko/gemini-mcp-tool` is working
2. **Scrape GMU page** → Use Gemini to extract tools from https://infoguides.gmu.edu/geohumanities/begin
3. **Opus workflow review** → Send `chat_parser_workflows.md` to Opus for:
   - Missing workflows
   - Process gaps
   - Best practices validation
   - Tool assignment optimization
4. **Package for Gemini** → After Opus approval, send complete spec to Gemini for coding

### Key Files

- `C:\Users\matts\chat_parser_workflows.md` - Complete workflow diagrams (NEW)
- `C:\Users\matts\chat_parser_plan_v2.md` - Opus-approved strategic plan
- `C:\Users\matts\chat_history_processor_plan.md` - Original plan
- `C:\Users\matts\chatgpt_parser.py` - Gemini's Sprint 1 delivery (untested)
- `C:\Users\matts\compare_nltk_vs_agent.py` - Token reduction proof

### Open-Source Parsers to Consider

**Don't reinvent the wheel** - evaluate these first:

1. **chat-export-structurer** (1ch1n)
   - Supports ChatGPT, Claude, Grok
   - SQLite output
   - Streaming parser for multi-GB files
   - Local-first, no API keys
   - **URL**: https://github.com/1ch1n/chat-export-structurer

2. **claude-conversation-extractor** (ZeroSumQuant)
   - Extracts Claude Code conversations from `~/.claude/projects`
   - CLI tool: `pipx install claude-conversation-extractor`
   - **URL**: https://github.com/ZeroSumQuant/claude-conversation-extractor

3. **quantified_chatgpt** (markwk)
   - CSV/spreadsheet export
   - Data viz and analysis
   - **URL**: https://github.com/markwk/quantified_chatgpt

### Architecture Decisions (Locked In)

- **Python/spaCy** for entity extraction (free, local)
- **Regex** for code detection, patterns (free, local)
- **NLTK VADER** for sentiment analysis (free, local)
- **Gemini API** for context-aware extraction (10-20% of data, batched)
- **Streaming I/O** for memory efficiency (<500MB peak)
- **SHA256 hashing** for deduplication
- **JSONL output** for incremental processing

### Cost Target

- <$1 per 10K messages (Gemini 2.0 Flash pricing)
- 90% of work done locally (Python/spaCy/NLTK)
- Only 10-20% sent to LLM (flagged conversations)

### User Preferences (Token Conscious)

- "stop fucking wasting my usage"
- "don't guess shit, use context7 or google search"
- "no inventing wheels, there are parsers online, open source"
- "use all your dev and A2A tools"

### Gemini MCP Tools (After Restart)

Should have access to:
- `mcp__gemini__*` tools from `@maxanatsko/gemini-mcp-tool`
- Can use Gemini 2.0 Flash Experimental (massive context window)
- API key configured: <redacted:GEMINI_API_KEY>
