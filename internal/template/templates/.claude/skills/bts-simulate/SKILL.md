---
name: bts-simulate
description: >
  Walk through scenarios to find gaps and incorrect assumptions.
  Document mode: test a spec document. Code mode: test implemented code
  against its spec. Both use scenario-based walkthrough.
user-invocable: true
allowed-tools: Read Write Agent Grep Glob
argument-hint: "[file-path] or code"
effort: max
context: fork
---

# Simulation

Run scenarios to find what's missing or wrong: $ARGUMENTS

## Settings

Simulation requires deep reasoning — it uses the main session model by default.
If `agents.simulator` is explicitly set in `.bts/config/settings.yaml`, use that model instead.

Adversarial validation uses `agents.simulator_validator` (default: sonnet) for the
defense round and `agents.simulator_rebuttal` (default: session model) for the
rebuttal round. Override in `.bts/config/settings.yaml`. Rebuttal uses the session
model because constructing concrete failure scenarios requires deeper reasoning.

## Mode Detection

Parse $ARGUMENTS:
- If first word is `code` → **Code Simulation** (see below)
- Otherwise → **Document Simulation** (spec walkthrough)

---

## Code Simulation

Simulate against implemented code to verify all paths are covered.

### Step 1: Identify Code Files and Spec

If tasks.json exists (implement recipe):
- Read tasks.json for implemented file list
- Read final.md for expected behavior and test scenarios

If no tasks.json (fix recipe):
- Read fix-spec.md "Changes" section for file paths
- Read fix-spec.md for expected behavior

### Step 2: Read Code

Read each implemented code file completely. Build a mental model of:
- All functions and their call graph
- All branches (if/else, switch, error returns)
- All error handling paths
- All external calls (DB, API, file I/O)

### Step 3: Design Scenarios

**Mermaid-guided scenario design**: If final.md/fix-spec.md contains mermaid
diagrams (state machines, flowcharts), read them first:
- Every edge in the state diagram should be covered by at least 1 scenario
- Every error/recovery path should have a dedicated scenario
- Flag uncovered edges as missing scenarios before proceeding

Design at least `simulate.min_scenarios` (default: 5) scenarios from the spec.
Cover the full risk surface — think about what could go wrong, what could be
misused, and what happens at boundaries. Typical concerns include normal flow,
failure modes, edge cases, security, and cross-component interactions, but
adapt the scenarios to what matters for this specific code.

### Step 4: Walk Through Code

For each scenario, trace the actual code path:
```
Scenario: [name]
Entry: [function/handler]
Step 1: [input] → code path: [function:line] → result: [X] ✓
Step 2: [action] → code path: [function:line] → **GAP: no handling for [Y]**
Step 3: [action] → code path: [function:line] → **ISSUE: spec says [A], code does [B]**
```

Additionally, for each scenario:
- Check if a test exists that exercises this code path
- If no test found → flag as **COVERAGE GAP**: "No test for scenario: [name]"
- Coverage gaps should be addressed by adding tests before re-running

Spawn Agent(simulator) for deeper analysis:
```
Read the code files [list] and spec at [final.md/fix-spec.md].
For each scenario [list], trace through the actual code paths.
At each step, check:
- Does the code handle this case?
- If handled, does it match the spec's expected behavior?
- If not handled, this is a GAP.
- Is there a test that covers this scenario? If not, flag COVERAGE GAP.
Report all GAPs, ISSUEs, and COVERAGE GAPs with severity and file:line references.
```

### Step 4.5: Flow Comparison (if spec has mermaid)

If the spec contains mermaid diagrams, generate a mermaid diagram of the
ACTUAL code flow and compare:
- Edge in spec but not in code → **GAP** (missing implementation)
- Edge in code but not in spec → **DEVIATION** (undocumented behavior)
- State in spec but unreachable in code → **GAP** (dead code or missing trigger)

**DEVIATIONs are excluded from adversarial validation** — they go to bts-sync
as undocumented behavior for spec reconciliation, not simulation gaps.
Only GAPs from flow comparison enter the adversarial step.

### Step 5: Assign Finding IDs

Before adversarial validation, assign stable IDs to all findings collected
from Step 4 and Step 4.5 (GAPs, ISSUEs, and COVERAGE GAPs — DEVIATIONs excluded):

- **GAP findings**: [GAP-001], [GAP-002], …
- **ISSUE findings**: [ISS-001], [ISS-002], …
- **COVERAGE GAP findings**: [COV-001], [COV-002], …

Compile the full finding list with IDs, severity, file:line, and description.
This is the raw input for adversarial validation.

### Step 5.5: Adversarial Validation

The simulation agents find problems (prosecution). Now the findings get a defense.

**Fallback**: If a validator or rebuttal agent fails (error, timeout), skip the
adversarial step and proceed to Step 6 with raw findings. Tag all findings as
`[UNVALIDATED]` in the report so the user knows they were not adversarially checked.

#### Round 1 — Defense (Validator)

Spawn **Agent(simulator-validator)** with a structured prompt:

```
Review the following simulation findings against the actual source material.
For each finding, read the referenced code and/or spec and determine if it
represents a real, practical gap or issue.

## Simulation Mode
Code

## Findings

1. [GAP-001] {title}
   Type: GAP | ISSUE | COVERAGE GAP
   Severity: {critical|major|minor}
   File: {file}:{line}
   Description: {what the simulator found}

2. [ISS-001] ...

## Files in scope (code + spec)
{list of code file paths and spec path}

## Test files in scope
{list of test file paths from test-results.json test_files field, or "none found"}
```

The validator reads the actual source material for each finding and returns:
- **CONFIRM**: Cannot defend. The finding is legitimate.
- **CHALLENGE**: Source-based evidence that this is not a practical problem.

#### Round 2 — Rebuttal (only if CHALLENGED items exist)

Collect all CHALLENGED findings. If none, skip to Step 6.

Spawn **Agent(simulator-rebuttal)** with a structured prompt:

```
The following simulation findings were challenged by a validator.
For each, determine whether the challenge is valid or the original finding stands.
You must read the actual source material to decide.

## Simulation Mode
Code

## Files in scope (code + spec)
{same list of code file paths and spec path as passed to the validator}

## Test files in scope
{same list of test file paths as passed to the validator, or "none found"}

## Challenged Findings

1. [GAP-001] {title}
   Type: GAP | ISSUE | COVERAGE GAP
   Original finding: {description from simulator}
   Validator's defense: {validator's CHALLENGE reasoning with source refs}
   Files to check: {relevant file paths cited by validator}

2. [ISS-001] ...
```

The rebuttal agent returns for each:
- **INSIST**: Concrete scenario (input → code path → failure) proving the gap is real.
- **CONCEDE**: Validator's defense is valid. Finding is not a practical gap.

#### Verdict (orchestrator — no agent)

| Simulator | Validator | Rebuttal | Result |
|-----------|-----------|----------|--------|
| Found     | CONFIRM   | —        | **AGREED**: Real gap |
| Found     | CHALLENGE | CONCEDE  | **DISMISSED**: Not practical |
| Found     | CHALLENGE | INSIST   | **DISPUTED**: Orchestrator adjudicates |

For **DISPUTED** items: the orchestrator designed the scenarios and is not a neutral
party, so DISPUTED findings are **INCLUDED by default**. Read both sides' evidence, then:
- Severity may be downgraded if the validator raised valid mitigating points
- Document both sides' arguments in the report for transparency
- EXCLUDED only if the validator's evidence conclusively proves the scenario is unreachable

### Step 6: Classify and Report

Count findings by verdict:
- **AGREED** and **DISPUTED/INCLUDED** findings enter the final report by severity
- **DISMISSED** findings are listed in a collapsed section

Severity classification:
- **critical**: Code path leads to crash, data loss, or security issue
- **major**: Important scenario not handled in code
- **minor**: Edge case missing but unlikely in practice

Save to `.bts/specs/recipes/{id}/simulations/NNN-code.md`

```markdown
# Simulation: Code — {recipe topic}

Generated: {ISO8601}
Recipe: {id}
Scenarios: N
Validation: adversarial (2-round debate)

## Summary
- GAPs: N (critical: N, major: N, minor: N)
- ISSUEs: N
- COVERAGE GAPs: N
- Dismissed: N (by adversarial validation)

## Critical
### [GAP-001] {title}
Scenario: {scenario name}
File: {file}:{line}
Consensus: AGREED | ADJUDICATED
{description}

## Major
...

## Minor
...

## Dismissed
<details>
<summary>N findings dismissed — click to expand</summary>

### [GAP-002] {title}
Original: {finding summary}
Defense: {validator's evidence}
Concession: {why rebuttal conceded}
</details>

## Adjudicated (disputed — orchestrator decided)
### [ISS-001] {title}
Prosecution: {rebuttal scenario}
Defense: {validator's evidence}
Verdict: INCLUDED (severity: {level}) | EXCLUDED — {orchestrator's reasoning}

## Flow Comparison (if mermaid present)
{mermaid diagram of actual code flow}

### DEVIATIONs (for bts-sync)
- [DEVIATION-001] {description}
```

Log:
```bash
bts recipe log {id} --action simulate --result "N scenarios, N gaps (N critical), N dismissed"
```

### After Code Simulation

The implement/fix flow should:
1. Fix the code to address GAPs and ISSUEs
2. Add tests for any COVERAGE GAPs found
3. Re-run tests: use Skill("bts-test") (mandatory after fixes)
4. Route DEVIATIONs to bts-sync for spec reconciliation
5. Do NOT re-run simulation (runs once per implementation)

---

## Document Simulation

Run scenarios against the spec to find what's missing or wrong.

### Protocol

1. Read the target document fully.

2. **Mermaid-guided scenario design**: If the document contains mermaid diagrams,
   read all state machines and flowcharts first. Use them to ensure every edge
   and every state transition is covered by at least one scenario. Flag uncovered
   edges before designing additional scenarios.

3. Design at least `simulate.min_scenarios` (default: 5) scenarios.
   Cover the full risk surface for this specific document — think about what could
   go wrong, what could be misused, what happens at boundaries, and what breaks
   under load. Adapt the scenario categories to what matters for this spec rather
   than following a fixed checklist.

4. For each scenario, walk through the spec step by step:
   ```
   Scenario: [name]
   Step 1: [action] → spec says [X] ✓ or → spec says nothing → GAP
   Step 2: [action] → spec says [Y] but [problem] → ISSUE
   ...
   ```

5. Spawn Agent(simulator) for deeper scenario analysis:
   ```
   Read the document at $ARGUMENTS and these scenarios: [list].
   For each scenario, trace through the document's described flow.
   At each step, check:
   - Is this step specified in the document?
   - If specified, is it correct and complete?
   - If not specified, this is a GAP.
   Report all GAPs and ISSUEs with severity.
   ```

6. Classify findings and assign stable IDs:
   - **GAP findings**: [GAP-001], [GAP-002], …
   - **ISSUE findings**: [ISS-001], [ISS-002], …

   Severity:
   - **critical**: Scenario leads to undefined behavior or crash
   - **major**: Important scenario not covered
   - **minor**: Edge case not mentioned but handleable

7. Adversarial Validation (Document Mode):

   **Fallback**: If a validator or rebuttal agent fails (error, timeout), skip
   adversarial and tag all findings as `[UNVALIDATED]` in the report.

   #### Round 1 — Defense (Validator)

   Spawn **Agent(simulator-validator)** with a structured prompt:

   ```
   Review the following simulation findings against the spec and external sources.
   For each finding, consult project-map.md, relevant layer specs in
   .bts/specs/layers/, and the codebase (if it exists). The spec document
   itself is NOT a sufficient defense source — you must find external authority.

   ## Simulation Mode
   Document

   ## Findings

   1. [GAP-001] {title}
      Type: GAP | ISSUE
      Severity: {critical|major|minor}
      Description: {what the simulator found}

   ## Spec document
   {path to spec document}

   ## External sources to consult
   - .bts/specs/project-map.md (if exists)
   - .bts/specs/layers/ (layer spec files, if any)
   - Codebase files (if implementation exists)
   ```

   The validator returns CONFIRM or CHALLENGE per finding.

   #### Round 2 — Rebuttal (only if CHALLENGED items exist)

   Collect all CHALLENGED findings. If none, skip to step 8.

   Spawn **Agent(simulator-rebuttal)** with a structured prompt:

   ```
   The following document simulation findings were challenged by a validator.
   For each, determine whether the challenge is valid or the original finding stands.

   ## Simulation Mode
   Document

   ## Spec document
   {same spec path as passed to the validator}

   ## External sources consulted by validator
   {same external source paths as passed to the validator}

   ## Challenged Findings

   1. [GAP-001] {title}
      Type: GAP | ISSUE
      Original finding: {description from simulator}
      Validator's defense: {CHALLENGE reasoning with source refs}
      Sources to check: {external source paths cited by validator}
   ```

   For Document mode, INSIST requires: realistic user/system action → spec gap →
   concrete bad outcome showing two reasonable implementors would make conflicting choices.

   #### Verdict (orchestrator — no agent)

   | Simulator | Validator | Rebuttal | Result |
   |-----------|-----------|----------|--------|
   | Found     | CONFIRM   | —        | **AGREED**: Real gap |
   | Found     | CHALLENGE | CONCEDE  | **DISMISSED**: Not practical |
   | Found     | CHALLENGE | INSIST   | **DISPUTED**: Orchestrator adjudicates |

   For **DISPUTED**: the orchestrator designed the scenarios and is not a neutral party,
   so DISPUTED findings are **INCLUDED by default**. Severity may be downgraded based
   on the validator's mitigating evidence. Document both sides' arguments transparently.

8. Save simulation results to `.bts/specs/{id}/simulations/NNN-[category].md`

   Report header should include: `Validation: adversarial (2-round debate)`

9. Log in changelog:
   ```bash
   bts recipe log {id} --action simulate --gaps N
   ```

### Output Format

```markdown
# Simulation: [document name]

Generated: {ISO8601}
Validation: adversarial (2-round debate)

## Scenario 1: [Happy Path - User Login]
- Step 1: User clicks login → spec: redirect to OAuth ✓
- Step 2: OAuth callback → spec: exchange code for token ✓
- Step 3: Token received → spec: create session → **GAP: session store not specified**
- Step 4: Redirect to dashboard → spec: redirect to / ✓
Result: 1 GAP found

## Scenario 2: [Error - Expired Auth Code]
- Step 1: Callback with expired code → spec: return 401 ✓
- Step 2: User experience → **GAP: what does the user see? Error page? Redirect?**
Result: 1 GAP found

...

## Summary
Total scenarios: 5
- GAPs: 4 (critical: 1, major: 2, minor: 1)
- ISSUEs: N
- Dismissed: N (by adversarial validation)

## Dismissed
<details>
<summary>N findings dismissed — click to expand</summary>

### [GAP-003] {title}
Original: {finding summary}
Defense: {validator's evidence}
Concession: {why rebuttal conceded}
</details>

## Adjudicated (disputed — orchestrator decided)
### [ISS-001] {title}
Prosecution: {rebuttal scenario}
Defense: {validator's evidence}
Verdict: INCLUDED (severity: {level}) | EXCLUDED — {orchestrator's reasoning}
```

### After Document Simulation

The recipe's adaptive loop should:
1. IMPROVE the spec to fill the gaps (AGREED and INCLUDED findings only)
2. Run /verify after improvement (mandatory)
3. Consider re-simulating after major changes
