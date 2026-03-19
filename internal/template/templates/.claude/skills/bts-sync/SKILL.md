---
name: bts-sync
description: >
  Synchronize final.md spec with actual implementation. Finds deviations between
  spec and code, updates final.md to reflect reality, and generates deviation report.
user-invocable: true
allowed-tools: Read Write Edit Grep Glob Bash Agent
argument-hint: "[recipe-id]"
---

# Sync: final.md ↔ Code

Synchronize spec and implementation for recipe: $ARGUMENTS

## Prerequisites

1. Verify test-results.json exists:
   ```bash
   ls .bts/state/recipes/{id}/test-results.json
   ```
   If not found → "Run /test first. Sync requires passing tests."

2. Check test status: should be `"pass"`.
   If `"fail"` → "Tests are failing. Fix tests before syncing."

## Step 1: Extract Spec Definitions (grouped by file)

Read `.bts/state/recipes/{id}/final.md` and group all definitions by file:

```
{
  "src/config.py": {
    functions: [name, params, return_type, error_handling],
    types: [name, fields],
    exports: [public API items]
  },
  "src/agent.py": { ... },
  ...
}
```

This creates one group per file, containing ALL items for that file.

## Step 2: File-by-File Comparison

For EACH file group (not each individual item):

1. Read the spec's definitions for this file (all functions, types, exports)
2. Read the actual code file
3. Compare ALL items in ONE pass:
   - List every function/type in spec → check it exists in code with correct signature
   - List every function/type in code → check if spec mentions it
   - This is a full comparison — nothing is skipped

This reduces comparison rounds from N-items to N-files while checking
every single item within each file.

Also scan for code files NOT mentioned in the spec that appear related
(same directory, similar naming).

## Step 3: Classify Results

### Cosmetic vs Functional

Automatically classify as **cosmetic** (non-blocking, one-line note each):
- Import ordering differences
- Docstring length/format differences
- Whitespace or formatting differences
- Lint suppression comments (# noqa, // nolint, //nolint)
- Language-conventional naming (spec: `max_retries`, JS code: `maxRetries`)

These are recorded under "Spec Additions" but do NOT require detailed analysis.

**Functional** differences require full analysis:
- Missing functions/methods
- Different parameter types or counts
- Different return types
- Different error handling behavior
- Missing error cases

For each item, classify as one of:

### Match
Spec and code agree. No action needed.

### Not Implemented
Spec defines it but code doesn't have it.
- Record in deviation report
- Severity: major (if core functionality) or minor (if optional)

### Spec Addition Needed (non-blocking)
Code has it but spec doesn't mention it.
- This happens when implementation required additional helpers, utilities, or types
- Record in deviation report for traceability
- Add to final.md — once added, this is resolved
- These do NOT block completion (sync already updated the spec)

### Deviation
Both exist but differ (different signature, different behavior).
- Record exact difference
- Determine which is correct (usually the code, since it passed tests)
- Update final.md to match code if code is correct
- Flag for user review if unclear

## Step 4: Preserve and Update final.md

1. **Preserve the original**: Copy `final.md` → `final.pre-sync.md`
2. **Update final.md** to reflect actual implementation:
   - Fix incorrect file paths
   - Update function signatures to match code
   - Add missing types/functions that were created during implementation
   - Mark removed items as deprecated or remove them
3. **Do not change the spec's intent or requirements** — only update implementation
   details to match reality.

## Step 5: Generate Deviation Report

Create `.bts/state/recipes/{id}/deviation.md`:

```markdown
# Deviation Report: {topic}

Generated: {ISO8601}
Recipe: {id}

## Summary
- Matches: N
- Not Implemented: N
- Spec Additions Needed: N
- Deviations: N

## Not Implemented
| Item | File | Reason |
|------|------|--------|
| ... | ... | ... |

## Spec Additions
| Item | File | Description |
|------|------|-------------|
| ... | ... | ... |

## Deviations
| Item | Spec Says | Code Has | Resolution |
|------|-----------|----------|------------|
| ... | ... | ... | ... |
```

## Step 6: Log and Validate

1. Log sync action (includes manifest registration and pre-sync backup):
   ```bash
   bts recipe log {id} --action sync --output deviation.md --based-on final.md --result "N matches, N deviations"
   ```
   Also manually register `final.pre-sync.md` in manifest.json as type `"draft"`.

2. Validate schema:
   ```bash
   bts validate
   ```

3. If final.md was modified (any Spec Addition or Deviation resolved by updating spec),
   run /verify on the updated final.md to ensure no contradictions were introduced.
   This satisfies the "every modification → /verify" rule.

Output `<bts>SYNC DONE</bts>` when complete.
