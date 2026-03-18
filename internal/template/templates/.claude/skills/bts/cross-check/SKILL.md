---
name: bts-cross-check
description: >
  Cross-check a document's factual claims against actual source code.
  Use when you need to verify file paths, function names, types, and line counts.
user-invocable: true
allowed-tools: Read Grep Glob Bash Agent
argument-hint: "[file-path]"
---

# Factual Cross-Check

Verify the specified document's factual claims against source code.

## Steps

1. Run deterministic fact-checking via the bts binary:
   ```bash
   bts verify $ARGUMENTS
   ```
   This checks: file existence, symbol names, line counts, import paths.

2. Spawn Agent(cross-checker) for semantic fact-checking:

   ```
   You are a fact-checking specialist. Read the document at $ARGUMENTS and the
   bts verify results. For each claim in the document that references code:

   - Verify the claim matches reality by reading the actual source file
   - Check function signatures match (parameter names, types, return types)
   - Check described behavior matches actual implementation
   - Check dependency versions if mentioned

   For each mismatch, classify:
   - critical: References something that does not exist
   - major: Exists but described incorrectly
   - minor: Approximately correct but imprecise (e.g., line count ±10%)

   Output findings as a numbered list with severity tags.
   ```

3. Merge binary results + agent findings
4. Report combined results with severity counts
