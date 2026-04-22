---
paths:
  - ".bts/**"
authoritative_for:
  - severity_classification
  - convergence_threshold
  - minor_subclassification
---

# BTS Verification Protocol

## Core Principle

Never verify your own output in the same context.

- **Internal consistency**: Checked by `bts verify` (deterministic) + Agent(verifier) (separate context)
- **Completeness**: Checked by Agent(auditor) (separate context)
- **Scenario coverage**: Checked by Agent(simulator) or /simulate (separate context)
- **Code references**: Checked by `bts verify` when code exists (deterministic, optional)

## Mandatory Verification Rule

**Every time a document is modified, /verify MUST run immediately after.**
This is non-negotiable. The recipe protocol enforces this.

## Severity Classification

- **critical**: Internal contradiction, undefined behavior in scenarios, impossible claims, execution path leading to undefined behavior. Never `[deferred]`.
- **major**: Missing error handling, incomplete data flow, unresolved design questions, important execution path not specified. Never `[deferred]`.
- **minor [resolvable]**: Fixable in the spec itself — metadata, typos, internal inconsistencies, cross-reference errors, unused declarations, outdated level/version headers, misused terminology, ambiguous wording, unspecified minor branches.
- **minor [deferred]**: Only resolvable at implementation/runtime — device-specific behavior, measured thresholds, framework-version-specific quirks, observable race windows. Every `[deferred]` minor MUST include a `Why-deferred:` line naming the specific runtime observation that would resolve it.
- **info**: Improvement suggestions, alternative approaches.

Rule: if filling the gap requires executing the code (or observing it on a physical device) to resolve, it is `[deferred]`, not an IMPROVE target. CRITICAL and MAJOR are never `[deferred]` — unknowable-pre-implementation gaps that would cause failure stay MAJOR; the spec must document the uncertainty as a defensive design decision.

## Convergence

- critical + major must reach 0 for Level 3
- `verify.max_iterations` consecutive IMPROVE→VERIFY cycles with no level change (default: 3) → report `[CONVERGENCE FAILED]`, ask human.
- Stagnation detector: if the SAME finding IDs persist across `verify.max_iterations` iterations, do not retry — report failed.
