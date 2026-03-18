---
name: bts-research
description: >
  Systematically research code, documentation, or external sources.
  Produces a structured research document. Use at the start of any recipe.
user-invocable: true
allowed-tools: Read Grep Glob Agent WebSearch WebFetch
argument-hint: "\"topic or question\""
---

# Systematic Research

Research the given topic and produce a structured document.

## Steps

1. Spawn Agent(Explore) to investigate the codebase:
   ```
   Thoroughly explore the codebase related to: $ARGUMENTS

   Find:
   - Relevant files and their roles
   - Key functions, types, and interfaces
   - Dependencies and import relationships
   - Existing patterns and conventions
   - Configuration and environment requirements
   ```

2. If external research is needed, use WebSearch/WebFetch for:
   - Official documentation
   - API references
   - Known issues or limitations

3. Synthesize findings into a structured document:
   ```markdown
   # Research: [topic]

   ## Current State
   - What exists now

   ## Key Components
   - Files, functions, types involved

   ## Dependencies
   - What depends on what

   ## Constraints
   - Limitations discovered

   ## Patterns
   - Conventions to follow
   ```

4. Save to `.bts/state/{recipe-id}/01-research.md` if inside a recipe
