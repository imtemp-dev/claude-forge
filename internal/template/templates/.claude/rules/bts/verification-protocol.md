---
paths:
  - ".bts/**"
---

# BTS Verification Protocol

## Core Principle

Never verify your own output in the same context. Verification uses:
1. **Deterministic checks** (bts binary): file existence, symbol names, line counts
2. **Independent agents** (separate context): logical and completeness review

## Severity Classification

- **critical**: References non-existent files, functions, or types. Must be 0 for completion.
- **major**: Logical inconsistency, missing error handling, incorrect signatures. Must be 0 for completion.
- **minor**: Imprecise wording, approximate numbers. Allowed in final document as annotations.
- **info**: Style suggestions. Ignored for convergence.

## Convergence Rules

- Maximum 3 verification iterations per recipe step
- critical + major must reach 0 for convergence
- If 2 consecutive iterations have identical errors → strategy change or human input
- After max iterations with remaining issues → [CONVERGENCE FAILED] → ask user
