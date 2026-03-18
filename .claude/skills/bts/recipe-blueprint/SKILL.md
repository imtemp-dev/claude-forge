---
name: bts-recipe-blueprint
description: >
  Create a Level 3 implementation spec — detailed enough for AI to generate
  code with high accuracy. Includes verification loop until all checks pass.
user-invocable: true
allowed-tools: Read Write Edit Grep Glob Bash Agent
argument-hint: "\"feature description\""
---

# Recipe: Blueprint (Level 3 Implementation Spec)

Create a bulletproof implementation spec for: $ARGUMENTS

## Resume Protocol

Before starting, check if this recipe has been started before:
```bash
bts recipe status
```
If a recipe is in progress, read its state file and resume from the saved step/iteration.

## Step 1: Research

Use Skill("bts-research") to investigate the codebase:
- What files exist related to this feature?
- What functions, types, and patterns are already in place?
- What dependencies are involved?
- What constraints exist?

Save results to `.bts/state/{id}/01-research.md`.

Log progress:
```bash
bts recipe log {id} --step research --status complete
```

## Step 2: Draft Level 3 Spec

Using research results, write a detailed implementation spec that includes:

For EACH file to be created or modified:
- **Exact file path** (e.g., `src/auth/oauth.ts`)
- **Action**: create / modify
- **Function signatures**: name, parameters with types, return type
- **Connection points**: which existing function calls this, or this calls
- **Error handling**: what errors can occur, how each is handled
- **Edge cases**: empty input, null, concurrent access, etc.
- **Scaffolding**: code skeleton showing structure

For the overall feature:
- **Data flow**: input → processing → output chain
- **State changes**: what state is modified and how
- **Test scenarios**: happy path + error paths with expected results

Save to `.bts/state/{id}/02-draft.md`.

```bash
bts recipe log {id} --step draft --status complete
```

## Step 3: Verify Loop

Repeat the following (max 3 iterations):

### 3a. Factual Cross-Check
Use Skill("bts-cross-check") on the draft:
- Are all file paths real?
- Do referenced functions exist?
- Are line numbers correct?
- Do import paths resolve?

### 3b. Logical Verification
Use Skill("bts-verify") on the draft:
- Any contradictions?
- Any unsupported claims?
- Any causal errors?

### 3c. Completeness Audit
Use Skill("bts-audit") on the draft:
- Missing error cases?
- Missing edge cases?
- Hidden assumptions?

### 3d. Evaluate Results

Collect all findings. Log iteration:
```bash
bts recipe log {id} --iteration N --critical X --major Y --minor Z
```

**If critical > 0 OR major > 0:**
- Fix all critical and major issues in the draft
- Go back to 3a (next iteration)

**If critical = 0 AND major = 0:**
- Proceed to Step 4

**If max iterations reached with remaining issues:**
- Report [CONVERGENCE FAILED] to the user
- List remaining issues
- Ask for guidance

## Step 4: Decision (if needed)

If the spec contains uncertain technical choices:
- Use Skill("bts-debate") to evaluate alternatives
- After debate conclusion, update the spec
- Go back to Step 3 for re-verification (iteration counter resets, log preserved)

If no decisions are needed, skip to Step 5.

## Step 5: Finalize

1. Copy the verified draft to `.bts/state/{id}/final.md`
2. Output `<bts>DONE</bts>` to signal completion

The Stop hook will verify that the last iteration in verify-log has
critical=0 and major=0 before allowing completion.

---

## Output Quality Target

The final spec should be detailed enough that giving it to Claude Opus
produces working code with minimal iteration. Every file path, function
signature, type, and connection point should be verified against actual code.
