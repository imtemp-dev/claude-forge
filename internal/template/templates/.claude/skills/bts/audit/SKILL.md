---
name: bts-audit
description: >
  Audit a document for completeness. Find missing scenarios, unconsidered
  edge cases, and hidden assumptions. Use after verify and cross-check.
user-invocable: true
allowed-tools: Read Grep Glob Agent
argument-hint: "[file-path]"
---

# Completeness Audit

Audit the specified document for missing items.

## Steps

1. Read the target document fully
2. Spawn Agent(auditor) with the following prompt:

   ```
   You are a completeness audit specialist. Read the document at $ARGUMENTS and check:

   - Missing error cases: What happens when things fail? Network errors? Invalid input?
   - Missing edge cases: Empty lists? Null values? Concurrent access? Large data?
   - Hidden assumptions: What does the document assume without stating?
   - Missing integration points: Are all connections to other systems specified?
   - Missing security considerations: Auth? Validation? Rate limiting?
   - Missing rollback/recovery: What if deployment fails? How to undo?

   For each missing item, classify:
   - critical: Will cause runtime failure if not addressed
   - major: Important gap that should be filled before implementation
   - minor: Nice to have but not blocking

   Output findings as a numbered list with severity tags.
   ```

3. Collect the auditor's findings
4. Report results with severity counts
