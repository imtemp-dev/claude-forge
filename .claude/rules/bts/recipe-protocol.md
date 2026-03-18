---
paths:
  - ".bts/**"
---

# BTS Recipe Protocol

## Execution Rules

1. **Always check for resume**: Run `bts recipe status` before starting. If a recipe is active, resume from saved state.
2. **Save state after each step**: Use `bts recipe log` to record progress.
3. **Verify loop is mandatory**: Never skip Steps 3a-3d in any recipe.
4. **Skills, not direct calls**: Call /verify, /cross-check, /audit — not Agent() or Bash() directly.
5. **Completion marker**: Output `<bts>DONE</bts>` only when all verifications pass.

## State Files

All recipe artifacts go to `.bts/state/recipes/{id}/`:
- `recipe.json`: metadata (type, step, iteration)
- `01-research.md`: research results
- `02-draft.md`: spec draft
- `final.md`: verified final spec
- `verify-log.jsonl`: iteration history

## Decision Points

When the recipe encounters an uncertain technical choice:
1. Use /debate to evaluate alternatives
2. If debate reaches consensus → apply and re-verify
3. If debate deadlocks → [DECISION REQUIRED] → pause for user input
