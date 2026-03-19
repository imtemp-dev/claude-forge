# bts — Bulletproof Technical Specification

Make your implementation spec so detailed that AI generates working code on the first try.

## The Problem

```
Rough plan → AI codes → bugs → fix → bugs → fix → ... (repeat N times)
```

Most time is spent debugging AI-generated code. The root cause: the spec was vague, so the AI guessed.

## The Solution

```
Spec → verify → fix → verify → ... → bulletproof spec → AI codes → done
```

Iterate on the **document**, not the code. Documents are free to change — no builds, no tests, no side effects. When the spec is bulletproof, AI generates code with minimal iteration.

## Full Lifecycle

```
/recipe blueprint "feature"
  → Scoping → Research → Draft → Verify Loop → Simulate → Debate → Finalize
  → /implement → Build Loop → /test → /sync → Complete

/recipe fix "bug description"
  → Diagnose → Fix Spec → Simulate → Expert Review → Verify → Implement → Test → Complete
```

bts covers **Planning → Build → Verify** as a single automated pipeline.

## Install

```bash
# One-line install (macOS / Linux)
curl -fsSL https://raw.githubusercontent.com/jlim/bts/main/install.sh | bash

# Or build from source (Go 1.22+)
git clone https://github.com/jlim/bts.git
cd bts
make install    # installs to ~/.local/bin/bts
```

PATH에 `~/.local/bin`이 없으면 `.zshrc` 또는 `.bashrc`에 추가:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

업데이트:
```bash
git pull && make install
```

버전 확인:
```bash
bts --version
# bts v0.1.0 (commit: abc1234, date: 2026-03-20T10:00:00Z)
```

## Quick Start

```bash
# Initialize in your project
bts init .

# Start Claude Code
claude

# Create a bulletproof spec
/recipe blueprint "add OAuth2 authentication"

# Fix a bug (lightweight)
/recipe fix "login bcrypt hash comparison fails"

# Or use individual skills
/bts-verify docs/spec.md
/bts-simulate docs/spec.md
/bts-debate "Redis vs Memcached for session store"
```

## Recipes

| Recipe | Purpose | Output |
|--------|---------|--------|
| `/recipe analyze` | Understand existing system | Level 1 analysis doc |
| `/recipe design` | Design a feature | Level 2 design doc |
| `/recipe blueprint` | Full implementation spec | Level 3 spec → code → tests |
| `/recipe fix` | Bug diagnosis and fix | Fix spec → code → tests |

### Blueprint Flow

```
Scoping (user alignment)
  → Research (codebase + Context7 + web)
  → Draft + Self-Check
  → Verify Loop (max 3 cycles)
  → Simulate (scenarios, early after first critical=0)
  → Debate + Adjudicate (if uncertain decisions)
  → Finalize (Level 3 spec)
  → Implement (task decomposition + build loop)
  → Test (generate + run + fix loop)
  → Sync (spec ↔ code comparison)
  → Complete
```

### Fix Flow

```
Diagnose (root cause analysis)
  → Fix Spec (document-first change description)
  → Simulate (impact analysis)
  → Expert Review (1-round debate)
  → Verify Loop
  → Implement (direct code fix)
  → Test (existing + regression)
  → Complete
```

## Skills (17)

| Category | Skills |
|----------|--------|
| **Recipes** | blueprint, design, analyze, fix |
| **Verification** | verify, cross-check, audit, assess, sync-check |
| **Analysis** | research, simulate, debate, adjudicate |
| **Implementation** | implement, test, sync, status |

## Architecture

```
Go binary (bts)                    Claude Code
├── bts init        deploy →       .claude/skills/     (17 skills)
├── bts validate    schema →       .claude/agents/     (3 agents)
├── bts hook        lifecycle      .claude/hooks/      (6 hooks)
├── bts recipe      state mgmt    .claude/rules/      (6 rules)
├── bts statusline  display       .claude/commands/    (1 dispatcher)
└── bts debate      state mgmt    .mcp.json           (Context7)
```

### Hooks

| Hook | Purpose |
|------|---------|
| session-start | Context injection (source-aware: resume/compact/startup) |
| pre-compact | Work state snapshot before context compaction |
| session-end | Work state persistence for cross-session resume |
| stop | Completion gates (DONE / IMPLEMENT DONE / FIX DONE) |
| subagent-start/stop | 🟡 indicator on statusline during agent execution |

### Statusline

```
bts v0.1.0 │ JWT auth │ 🟡 verify │ ctx 45%
bts v0.1.0 │ JWT auth │ implement 3/5 │ ctx 60%
bts v0.1.0 │ bcrypt fix │ test │ ctx 30%
```

## Key Principles

- **Document first**: Iterate on the spec, not the code
- **Never verify your own output**: Verification uses separate agent contexts
- **Context as glue**: Skills provide situational awareness, not rigid rules
- **Deviation = follow-up**: Spec-code differences are reports, not gates
- **Crash resilient**: Work state persists via tasks.json + work-state.json

## CLI

```
bts init [dir]              Initialize project
bts validate [recipe-id]    Check JSON schema compliance
bts recipe status           Show active recipe
bts recipe list             All recipes
bts recipe log <id>         Record action/phase/iteration
bts recipe cancel           Cancel active recipe
bts debate list             All debates
bts statusline              Render status for Claude Code (internal)
bts hook <event>            Handle lifecycle events (internal)
```

## Document Levels

| Level | Name | Contains | AI Code Accuracy |
|-------|------|----------|-----------------|
| 1 | Understanding | System structure, files, dependencies | Not possible |
| 2 | Design | Components, data flow, tech choices | ~60-70% |
| 3 | Implementation-ready | File paths, signatures, types, edge cases, scaffolding | **Very high** |

## License

MIT
