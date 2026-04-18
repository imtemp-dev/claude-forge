---
name: bts-verify
description: >
  Verify a document for logical errors, contradictions, and unsupported claims.
  Includes mermaid flow path enumeration to find unspecified execution paths.
user-invocable: true
allowed-tools: Read Grep Glob Agent WebSearch WebFetch mcp__context7__resolve-library-id mcp__context7__get-library-docs
argument-hint: "[file-path]"
context: fork
effort: max
---

# Logical Verification

Verify the specified document for logical correctness.

## Settings

Verification is the core quality gate — it uses the main session model by default.
If `agents.verifier` is explicitly set in `.bts/config/settings.yaml`, use that model instead.

## Steps

1. Read the target document fully

2. Spawn Agent(verifier) with the following prompt:

   ```
   You are a logical verification specialist. Read the document at $ARGUMENTS and check for:

   **Text-level verification:**
   - Contradictions: Does the document make conflicting claims?
   - Unsupported conclusions: Are conclusions drawn from insufficient evidence?
   - Causal errors: Are cause-effect relationships correctly established?
   - Missing premises: Are there hidden assumptions not stated?
   - Circular reasoning: Does any argument reference itself?

   **Flow-level verification (mermaid diagrams):**
   If the document contains mermaid diagrams (stateDiagram, flowchart, etc.):
   - Enumerate ALL possible paths from start to end in each diagram
   - For EACH path: is the behavior fully specified in the document text?
   - Flag paths where behavior is unspecified as GAPs
   - Check for dead-end states (states with no exit transition)
   - Check for orphan states (states with no entry transition)
   - Check that every error/failure state has a recovery or terminal path
   - Check for missing transitions: at each state, what happens on
     timeout? invalid input? resource exhaustion? concurrent access?

   **Evidence policy for framework/platform claims:**

   Before classifying a claim about framework or platform internals
   (animation timing, reconciler behavior, async runtime semantics,
   memory/lifecycle rules, OS-level UI dismissal windows, etc.) as
   CRITICAL or MAJOR, attempt evidence gathering in this order:

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
   - MINOR [resolvable]: fixable in the spec itself — metadata, typos,
     internal inconsistencies, cross-reference errors, unused declarations,
     outdated level/version headers, misused terminology.
   - MINOR [deferred]: can only be verified at implementation or runtime —
     actual rendering outcomes, measured timing, real allocator behavior,
     observable race windows on a specific device, framework-version-specific
     edge cases that require running the code to confirm/deny.

   Rule: if a finding requires executing the code (or observing it on a
   physical device) to resolve, it is [deferred], not an IMPROVE target.
   Every [deferred] minor MUST include a `Why-deferred:` line naming the
   specific runtime observation that would confirm or deny the finding.

   CRITICAL and MAJOR are never [deferred] — if something is unknowable
   pre-implementation AND would cause failure, it stays MAJOR and the
   spec must document the uncertainty as a defensive design decision.

   **Report format:**
   For each issue found, classify severity:
   - critical: Factually impossible, self-contradicting, or execution path leads to undefined behavior
   - major: Logically flawed, or important execution path not specified
   - minor [resolvable]: Ambiguous or imprecise but fixable in the spec
   - minor [deferred]: Only resolvable at implementation/runtime

   Output your findings as a numbered list with severity tags.
   For each finding also include (when applicable):
     Source: <URL> | <URL>
     Gathered: <Context7|WebFetch|WebSearch summary>
     Why-deferred: <runtime observation that would resolve it>   (deferred only)

   Summary line:
     Text issues: N. Flow path issues: N. Total paths analyzed: N.
     Evidence-resolved: X (removed Y, downgraded Z). Framework-claim findings: W.
     Minors: R resolvable, D deferred.
   ```

3. Collect the verifier's findings
4. Report results to the user with severity counts
