---
name: bts-review
description: >
  Multi-perspective code review with quality, security, and architecture agents.
  Basic review (all perspectives) by default, or focused review with category.
  Includes practical assessment of findings.
user-invocable: true
allowed-tools: Read Grep Glob Bash Agent AskUserQuestion
argument-hint: "[security|performance|patterns] [file-path]"
effort: high
---

# Code Review

Review code for: $ARGUMENTS

## Settings

Quality and architecture review require deep reasoning — they use the main session model by default.
Security review uses sonnet (pattern-based). Override any agent model via `.bts/config/settings.yaml`:
`agents.reviewer_quality`, `agents.reviewer_security`, `agents.reviewer_arch`.

## Step 1: Determine Review Mode

Parse $ARGUMENTS:
- If first word is `security`, `performance`, or `patterns` → **focused mode**
  Remaining words = file scope (or all if empty)
- Otherwise → **full mode** (all perspectives), arguments = file scope

## Step 2: Identify Scope and Context

**File scope:**
If inside a recipe (tasks.json exists):
- Read tasks.json for the list of implemented files
- If test-results.json exists, also include files from `test_files` in scope.
  Test code quality matters: correct assertions, realistic mocks, test isolation.
- If file scope given → filter to matching files
- If no scope → review all files from tasks.json + test files

If inside a recipe but no tasks.json (fix recipe):
- Read fix-spec.md "Changes" section for file paths
- If no fix-spec.md → fall back to git diff

If standalone (no recipe):
- If file scope given → review those files/directories
- If no scope → try `git diff --name-only HEAD~1` to detect recently changed files.
  If changes found, propose reviewing those files.
  If no git or no changes, ask user which files to review.

**Architecture context:**
- Read `.bts/specs/project-map.md` for layer structure
- Read `.bts/specs/layers/{name}.md` for the relevant layer's patterns
- If inside a recipe, read final.md for design intent
- Pass this context to the architecture agent

## Step 3: Multi-Perspective Review

### Full Mode (no category — default)

Spawn all 3 agents **in a single message with multiple Agent tool calls** to ensure
true parallel execution. Do NOT spawn them sequentially.
The default perspectives are quality, security, and architecture, but adapt
if the code warrants different emphasis (e.g., performance-critical code may
need a performance perspective instead of or in addition to security):

1. **Agent(reviewer-quality)**: Code quality — correctness, error handling,
   resource management, maintainability.
   If the spec (final.md) contains mermaid flow diagrams, compare the code's
   actual control flow against the spec's expected flow. Flag deviations.

2. **Agent(reviewer-security)**: Security — input validation, authentication,
   data protection, common vulnerability patterns

3. **Agent(reviewer-arch)**: Architecture — alignment with project structure
   (project-map.md, layers), naming conventions, pattern consistency.
   Include project-map.md and layers content in the agent prompt.

Each agent produces a numbered list of findings with severity tags.

### Focused Mode

| Category | Agent(s) | Focus |
|----------|----------|-------|
| `security` | reviewer-security only | Deep security analysis |
| `performance` | reviewer-quality | Performance focus: N+1 queries, memory, blocking I/O, algorithm complexity |
| `patterns` | reviewer-arch | Pattern focus: conventions, structure, consistency |

For focused mode, give the single agent a deeper, more thorough prompt
for its specific domain rather than a broad scan.

## Step 4: Synthesize Raw Findings

After collecting findings from all agents:

1. **Deduplicate**: Same issue found by multiple agents → merge, note all perspectives
2. **Assign unified IDs**: [CRT-001], [MAJ-001], [MIN-001], [INF-001]
3. **Compile findings list**: Each finding with ID, severity, file:line, description, fix suggestion

This is the raw input for adversarial validation. Do NOT assess practicality here.

## Step 4.5: Adversarial Validation

The review agents find problems (prosecution). Now the code gets a defense.
Configure the validator model via `agents.reviewer_validator` (default: sonnet)
and the rebuttal model via `agents.reviewer_rebuttal` (default: session model)
in settings.yaml. Rebuttal uses the session model because constructing concrete
failure scenarios requires deeper reasoning than pattern-based validation.

**Fallback**: If a validator or rebuttal agent fails (error, timeout), skip the
adversarial step and proceed to Step 5 with raw findings. Tag all findings as
`[UNVALIDATED]` in the report so the user knows they were not adversarially checked.

### Round 1 — Defense (Validator)

Spawn **Agent(reviewer-validator)** with a structured prompt:

```
Review the following findings against the actual source code.
For each finding, read the referenced code and determine if it is
a real, practical problem.

## Findings

1. [CRT-001] {title}
   File: {file}:{line}
   Severity: {critical|major|minor|info}
   Found by: {agent}
   Description: {what the reviewer found}
   Fix suggestion: {suggested fix}

2. [MAJ-001] ...

## Files in scope
{list of file paths the agent should read}
```

The validator reads the actual code for each finding and returns:
- **CONFIRM**: Cannot defend the code. Finding is legitimate.
- **CHALLENGE**: Code-based evidence that this is not a practical problem.

### Round 2 — Rebuttal (only if CHALLENGED items exist)

Collect all CHALLENGED findings. If none, skip to Step 5.

Spawn **Agent(reviewer-rebuttal)** with a structured prompt:

```
The following review findings were challenged by a code validator.
For each, evaluate whether the challenge is valid or the original
finding stands. You must read the actual code to decide.

## Challenged Findings

1. [MAJ-001] {title}
   Original finding: {description from reviewer}
   Validator's defense: {validator's CHALLENGE reasoning with code refs}
   Files to check: {relevant file paths}

2. [MAJ-002] ...
```

The rebuttal agent returns for each:
- **INSIST**: Concrete scenario proving the issue is real, rebutting the defense.
- **CONCEDE**: Validator's defense is valid. Finding is not practical.

### Verdict (orchestrator — no agent)

Classify each finding into consensus categories:

| Review | Validator | Rebuttal | Result |
|--------|-----------|----------|--------|
| Found  | CONFIRM   | —        | **AGREED**: Real issue |
| Found  | CHALLENGE | CONCEDE  | **DISMISSED**: Not practical |
| Found  | CHALLENGE | INSIST   | **DISPUTED**: Orchestrator adjudicates |

For **DISPUTED** items: read the code yourself, evaluate both sides' evidence,
and make a final call. Include both arguments in the report for transparency.

## Step 5: Generate Report

Save to `.bts/specs/recipes/{id}/review.md` if inside a recipe.
Otherwise output directly to user.

```markdown
# Code Review: {scope}

Generated: {ISO8601}
Recipe: {id} (if applicable)
Mode: {full|security|performance|patterns}
Perspectives: {quality + security + architecture | single agent}
Validation: adversarial (2-round debate)

## Summary
- Critical: N
- Major: N
- Minor: N
- Dismissed: N (by adversarial validation)

## Critical
1. [CRT-001] **{title}** in `{file}:{line}`
   Found by: {agent name}
   Consensus: AGREED | ADJUDICATED
   {code context}
   → {fix suggestion}

## Major
...

## Minor
...

## Dismissed (validator defended successfully)
<details>
<summary>N findings dismissed — click to expand</summary>

1. [MAJ-002] **{title}** in `{file}:{line}`
   Original: {finding summary}
   Defense: {validator's evidence}
   Concession: {why rebuttal agent conceded}
</details>

## Adjudicated (disputed — orchestrator decided)
1. [MAJ-003] **{title}** in `{file}:{line}`
   Prosecution: {rebuttal agent's scenario}
   Defense: {validator's evidence}
   Verdict: {INCLUDED|EXCLUDED} — {orchestrator's reasoning}
```

Log if inside recipe:
```bash
bts recipe log {id} --action review --output review.md --result "N critical, N major, N dismissed"
```

Review is a **mandatory step** in implement and fix flows.
Critical/major consensus items should be fixed before proceeding.
