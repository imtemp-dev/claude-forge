---
name: bts-audit
description: >
  Audit a document for completeness. Find missing scenarios, unconsidered
  edge cases, and hidden assumptions. Includes mermaid branch completeness
  analysis. Use after verify and cross-check.
user-invocable: true
allowed-tools: Read Grep Glob Agent WebSearch WebFetch mcp__context7__resolve-library-id mcp__context7__get-library-docs
argument-hint: "[file-path]"
context: fork
---

# Completeness Audit

Audit the specified document for missing items.

## Settings

Audit requires finding what's missing — it uses the main session model by default.
If `agents.auditor` is explicitly set in `.bts/config/settings.yaml`, use that model instead.

## Steps

1. Read the target document fully
2. Spawn Agent(auditor) with the following prompt:

   ```
   You are a completeness audit specialist. Read the document at $ARGUMENTS.

   Your goal: find everything the document fails to address that could cause
   problems at runtime, during deployment, or under adversarial conditions.

   **Content completeness:**
   Think about failure modes, boundary conditions, unstated assumptions,
   missing integrations, security gaps, and operational concerns. Do not
   limit yourself to a fixed checklist — reason about what this specific
   system needs and what the document leaves unanswered.

   **Flow completeness (mermaid diagrams):**
   If the document contains mermaid diagrams:
   - At EVERY decision node: are ALL branches specified? (yes/no/error/timeout)
   - At EVERY state: what happens on timeout? invalid input? resource exhaustion?
     concurrent access? If unspecified, flag as a completeness gap.
   - Are there states that can only be reached through a single path?
     (fragile — what if that path fails?)
   - For each error state: is the error message/response defined? Is cleanup specified?
   - Count: total decision nodes, branches specified, branches missing.

   **Evidence policy for framework/platform claims:**

   Before classifying a claim about framework or platform internals
   (animation timing, reconciler behavior, async runtime semantics,
   memory/lifecycle rules, OS-level UI dismissal windows, known failure
   modes, etc.) as CRITICAL or MAJOR, attempt evidence gathering in this
   order:

   1. Context7 MCP (preferred): mcp__context7__resolve-library-id then
      mcp__context7__get-library-docs with a topic from the claim.
   2. WebFetch on OFFICIAL domains only when Context7 misses:
      developer.apple.com, developer.android.com, react.dev, nodejs.org,
      docs.swift.org, kotlinlang.org, pytorch.org, tensorflow.org,
      learn.microsoft.com, docs.oracle.com, official GitHub RFCs/issues
      in the framework's own repo, WWDC / Google I/O official transcripts.
   3. WebSearch as last resort, always with site: filters on the same
      official domains. Never generic queries.

   NOT evidence: Medium, dev.to, personal blogs, StackOverflow (lead only),
   unofficial tutorials, unversioned docs.

   Reclassify by outcome:
   - Official source CONTRADICTS → CRITICAL, cite URL.
   - Official source CONFIRMS → REMOVE finding.
   - Official source SILENT, affects user code → keep as MAJOR (defensive).
   - Official source SILENT, purely framework-internal → downgrade to MINOR.
   - Only non-official sources found → downgrade to MINOR, note why.

   Citations:
   - Each evidence-resolved finding MUST include a `Source:` line with URLs.
   - Never invent citations. If a fetch fails, write "Evidence unavailable"
     and keep the conservative classification from the table above.
   - For every claim you attempted to evidence, include a line
     `Gathered: [Context7:<hit|miss> | WebFetch:<url>:<status> | WebSearch:<n>]`
     so downstream improve cycles can see what was tried.

   Budget: evidence-gather only CRITICAL/MAJOR candidates, cap at 5 findings
   per run to keep iteration time bounded. Minor findings need no evidence.

   **Minor sub-classification:**

   Every MINOR finding must be tagged as either [resolvable] or [deferred]:
   - MINOR [resolvable]: fixable in the spec itself — missing edge case
     documentation, unspecified minor branches, incomplete error messages,
     cross-reference gaps that do not block implementation.
   - MINOR [deferred]: the missing information can only be produced at
     implementation/runtime — device-specific behavior, actual measured
     thresholds, empirical limits, framework-version-specific quirks.

   Rule: if filling the gap requires executing the code (or observing it
   on a physical device) to resolve, it is [deferred], not an IMPROVE
   target. Every [deferred] minor MUST include a `Why-deferred:` line
   naming the specific runtime observation that would resolve it.

   CRITICAL and MAJOR are never [deferred] — if a gap would cause runtime
   failure AND is unknowable pre-implementation, it stays MAJOR and the
   spec must document the uncertainty as a defensive design decision.

   For each missing item, classify:
   - critical: Will cause runtime failure if not addressed
   - major: Important gap that should be filled before implementation
   - minor [resolvable]: Nice to have, fixable in the spec
   - minor [deferred]: Nice to have, only confirmable at implementation/runtime

   Output findings as a numbered list with severity tags.
   For each finding also include (when applicable):
     Source: <URL> | <URL>
     Gathered: <Context7|WebFetch|WebSearch summary>
     Why-deferred: <runtime observation that would resolve it>   (deferred only)

   Include: "Branch coverage: N/M decision branches specified (N%).
   Evidence-resolved: X (removed Y, downgraded Z). Framework-claim findings: W.
   Minors: R resolvable, D deferred."
   ```

3. Collect the auditor's findings
4. Report results with severity counts
