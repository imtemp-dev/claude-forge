# BTS Flow Metrics

This document defines the seven indicators tracked by
`scripts/bts-monitor.ts` against a baseline from `scripts/bts-baseline.ts`.
Each indicator targets a specific failure mode the Sprint 1–3 changes
were designed to catch.

The baseline of record (2026-04-22) is
`data/baselines/snap-voca-2026-04-22.json`. Weekly runs land in
`data/monitoring/{target}-{YYYY-MM-DD}.json`.

Run:

```bash
BTS_BIN=$(which bts) npx tsx scripts/bts-monitor.ts \
  --target /Users/jlim/Workspace/snap-voca-doc-scanner \
  --baseline data/baselines/snap-voca-2026-04-22.json \
  --out data/monitoring/snap-voca-$(date +%Y-%m-%d).json
```

---

## 1. Iteration-to-converge

**What**: mean number of verify iterations to reach
`(critical=0, major=0, minor_resolvable=0)` for the first time.

**Signal it catches**: the "patch-of-patches" loop. Each unnecessary
iteration is time spent circling instead of advancing, usually because
the spec is incomplete, the wireframe is wrong, or the decomposition is
confused.

**Baseline (snap-voca, 17 recipes)**: 3.93.

**Targets by phase**:

| After | Target |
|-------|--------|
| Phase 2 (schema contracts) | −10% → ≤ 3.54 |
| Phase 4 (domain model) | −20% → ≤ 3.14 |
| Phase 6 (cross-boundary sim) | −30% → ≤ 2.75 |

---

## 2. Re-decomposition count

**What**: mean number of `architect` action entries in a recipe's
changelog. Also tracked: count of recipes where this exceeds 1.

**Signal it catches**: the decomposition was wrong on the first attempt
and had to be redone mid-recipe. One architect invocation per recipe is
healthy (alternatives debated, selected). Two or more means the initial
debate missed something the implementation surfaced.

**Baseline**: 0 (legacy recipes predate /bts-architect).

**Target after Phase 5**: mean ≤ 1.0; zero recipes with > 1.

---

## 3. Invariant mono-owner rate

**What**: share of recipes whose `domain.md §2 Invariants` table has
zero `invariant_multiple_owners` critical from `bts verify`.

**Signal it catches**: "the truth is stored in two places at once" — the
Duolingo failure mode directly. Each violation indicates a duplicated
source-of-truth the implementation will paper over with coupling.

**Baseline**: 100% (no recipes have domain.md yet, so the denominator is
zero and the ratio defaults to 1.0 — this will fall as recipes start
adopting domain.md; a drop to 100% from new recipes is the target state).

**Target after Phase 4**: 100% for recipes with domain.md. Track the
*denominator growth* (number of recipes carrying domain.md) as a
separate uptake metric.

---

## 4. Cross-boundary simulation coverage

**What**: mean fraction of non-legacy tagged simulation scenarios that
carry `[cross-boundary: ...]` or `[illegal-cell: ...]`.

**Signal it catches**: simulations that only exercise per-module paths
and miss the interaction bugs that show up under concurrent state
changes across modules.

**Baseline**: 0.0% (all existing scenarios tagged `[single-axis: legacy]`
by `bts migrate simulations`).

**Target after Phase 6**: ≥ 30% per simulation file. Recipes below the
threshold are reported as `recipes_below_cross_boundary_threshold`.

---

## 5. Spec-code structural match

**What**: total count of `unauthorized_coupling` and
`unimplemented_dependency` findings across all recipes. Measured by
`bts graph --import` against `wireframe.md` component diagrams.

**Signal it catches**: code that silently imports modules the wireframe
did not authorize, or planned dependencies that were never wired up
during implementation. Both are signs the implementation has drifted
from the selected decomposition.

**Baseline**: 0 (measurement not yet wired through `bts-review` agent
output; placeholder zero).

**Target after Phase 6.3**: 0 unauthorized couplings per recipe.
Follow-up work (beyond this sprint): parse `review.md` for these
findings so the monitor can count them automatically instead of relying
on the reviewer's prose summary.

---

## 6. Refactor signal frequency

**What**: total signals returned by `bts refactor-signal` across all
recipes, and count of recipes where the CLI reports at least one
signal.

**Signal it catches**: the recipe's history itself shows a
patch-of-patches pattern — `test_fix_cascade` (one test → 3+ module
edits) or `cross_module_churn` (one module edited 4+ times, or 3+
modules edited 3+ times each).

**Baseline**: 0 signals across 0 recipes (none of the 17 snap-voca
recipes trigger the detectors — clean baseline).

**Target**: no increase over baseline. If this metric rises for new
recipes, the decomposition gates (Phase 4/5) are not catching the
problem in time.

---

## 7. Convergence failure rate

**What**: number of `verify-log.jsonl` entries with `status=failed`
divided by total recipes.

**Signal it catches**: recipes where the verify loop hits
`verify.max_iterations` (default 3) without ever reaching
`critical=0, major=0`. These are the recipes the user has to intervene
on manually.

**Baseline**: 0.000 (no failed status in current verify-logs).

**Target**: 0. A non-zero rate going forward is a direct signal that
Phase 4/5 gates are insufficient — the decomposition was allowed to
reach implementation despite structural issues.

---

## Reading the weekly report

`data/monitoring/{target}-{date}.json` contains:

- `indicators` — the 7 numbers above plus derived counts
- `per_recipe` — per-recipe breakdown so outliers can be traced
- `delta` — difference vs. the supplied baseline (currently tracks
  `mean_iteration_to_converge` and `convergence_failure_rate`; more
  fields as baselines grow to carry them)

When an indicator regresses (wrong direction) by more than 5%, treat it
as a real signal and investigate the responsible recipe before the
pattern spreads.

## Uptake metrics (not in the indicator set)

Distinct from the 7 indicators, track the *adoption rate* of the new
gates so we know whether a steady indicator means "healthy" or "not
yet exercised":

- Number of recipes with `domain.md`
- Number of recipes with a `<!-- architect-decision -->` block
- Number of scenarios tagged `[cross-boundary: ...]` or
  `[illegal-cell: ...]` (non-legacy)

These live in `per_recipe.*` of the monitoring report; derived counts
can be added to `indicators` when adoption becomes a tracked goal.
