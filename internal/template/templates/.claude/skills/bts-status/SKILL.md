---
name: bts-status
description: >
  Generate or update project-status.md — a comprehensive view of all recipes,
  their implementation state, deviations, and next steps.
user-invocable: true
allowed-tools: Read Write Edit Grep Glob Bash
argument-hint: "[recipe-id or 'all']"
---

# Project Status: Generate/Update

Update project status for: $ARGUMENTS

If argument is a recipe ID, update status for that recipe only.
If argument is "all" or empty, scan all recipes.

## Step 1: Scan Recipes

Read `.bts/state/recipes/` directory:
```bash
ls .bts/state/recipes/
```

For each recipe directory, read:
- `recipe.json` → type, topic, phase
- `tasks.json` → implementation progress (if exists)
- `test-results.json` → test status (if exists)
- `deviation.md` → sync status (if exists)
- `final.md` → spec exists? (if exists)

## Step 2: Determine Recipe States

For each recipe, determine its state:

| State | Criteria |
|-------|----------|
| `drafting` | recipe.json exists, no final.md |
| `spec` | final.md exists, no tasks.json |
| `implementing` | tasks.json exists, some tasks pending |
| `implemented` | tasks.json exists, all tasks done |
| `tested` | test-results.json exists, status=pass |
| `synced` | deviation.md exists |
| `complete` | tested + synced |

## Step 3: Generate project-status.md

Write to `.bts/state/project-status.md`:

```markdown
# Project Status

Updated: {ISO8601}

## Features

| Recipe | Type | Topic | State | Tests | Deviations |
|--------|------|-------|-------|-------|------------|
| r-xxx | blueprint | Auth | complete | 15/15 pass | 0 |
| r-yyy | design | API | spec | — | — |

## Architecture

### Implemented Files
List all files created/modified across all recipes with tasks.json:

```
src/
  auth/
    types.ts (r-xxx)
    oauth.ts (r-xxx)
    session.ts (r-xxx)
  api/
    routes.ts (r-yyy)
```

## Deviations

Aggregate all deviation.md findings:

| Recipe | Item | Type | Status |
|--------|------|------|--------|
| r-xxx | getUserById | signature | resolved |

## Next Steps

Based on current state, recommend what to do next:
- Recipes in `spec` state → "Run /implement {id}"
- Recipes in `implementing` state → "Resume /implement {id}"
- Recipes in `implemented` state → "Run /test {id}"
- Recipes with failing tests → "Fix failures in ..."
- Complete recipes with deviations → "Follow-up: review deviation.md for improvements"
```

**Note**: `project-status.md` is a global derived document at `.bts/state/` level.
It is NOT tracked in per-recipe manifests because it spans multiple recipes.

## Step 4: Log

If a specific recipe ID was given:
```bash
bts recipe log {id} --action status --result "state: {determined-state}"
```

Validate:
```bash
bts validate
```

Output the status summary to the user directly.
