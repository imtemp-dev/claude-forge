package engine

// HardGate is a deterministic enforcement point for a mandatory rule.
// Each entry pairs a stable ID used in rule frontmatter (e.g., "{gate:
// hard}" inline tags) with the source location that actually enforces it.
// `bts validate --gates` cross-checks that every declared hard gate maps
// to a real code path.
type HardGate struct {
	ID          string // stable identifier cited in rule docs
	Rule        string // rule source (file:rule-number)
	Enforcement string // code location that enforces it
	Summary     string // one-line explanation
}

// HardGates is the authoritative registry of machine-enforced rules.
// Adding a "{gate: hard}" tag in a rule document without registering it
// here will fail `bts validate --gates` once that subcommand ships.
var HardGates = []HardGate{
	{
		ID:          "verify_after_modification",
		Rule:        "bts-recipe-protocol.md §Mandatory Rules rule 3",
		Enforcement: "internal/hook/stop.go:handleSpecDone",
		Summary:     "Block <bts>DONE</bts> unless verify-log last entry has critical=0 major=0 minor_resolvable=0",
	},
	{
		ID:          "log_every_action",
		Rule:        "bts-recipe-protocol.md §Mandatory Rules rule 4",
		Enforcement: "internal/cli/recipe.go:recipeLogCmd",
		Summary:     "Validator flags changelog gaps; log subcommand is the only way to append canonical entries",
	},
	{
		ID:          "simulate_at_least_once",
		Rule:        "bts-recipe-protocol.md §Mandatory Rules rule 5",
		Enforcement: "internal/cli/recipe.go:checkPhasePreConditions(review)",
		Summary:     "Phase transition to review warns if no simulation exists",
	},
	{
		ID:          "adjudicate_every_debate",
		Rule:        "bts-recipe-protocol.md §Mandatory Rules rule 7",
		Enforcement: "internal/engine/validator.go:validateDebateMetaJSON",
		Summary:     "Debate meta.json must carry a decided boolean and, when decided, a string conclusion",
	},
	{
		ID:          "sync_check_before_final",
		Rule:        "bts-recipe-protocol.md §Mandatory Rules rule 8",
		Enforcement: "internal/hook/stop.go:handleSpecDone",
		Summary:     "Sync-check records land in verify-log; gate reads latest entry",
	},
	{
		ID:          "status_at_finalization",
		Rule:        "bts-recipe-protocol.md §Mandatory Rules rule 9",
		Enforcement: "internal/hook/stop.go:handleSpecDone",
		Summary:     "Completion sets phase=finalize which triggers the status flow",
	},
	{
		ID:          "spec_before_code",
		Rule:        "bts-implement-protocol.md §Execution Rules rule 1",
		Enforcement: "internal/cli/recipe.go:checkPhasePreConditions(implement)",
		Summary:     "Phase transition to implement warns if final.md missing; stop hook blocks IMPLEMENT DONE without tasks.json",
	},
	{
		ID:          "build_verification",
		Rule:        "bts-implement-protocol.md §Execution Rules rule 2",
		Enforcement: "bts-implement/SKILL.md Step 3 + stop hook via test-results.json",
		Summary:     "Implementation skill runs build per task; stop hook requires test-results.status=pass",
	},
	{
		ID:          "test_after_implement",
		Rule:        "bts-implement-protocol.md §Execution Rules rule 4",
		Enforcement: "internal/hook/stop.go:handleImplementDone",
		Summary:     "IMPLEMENT DONE blocked unless test-results.json exists with status=pass",
	},
	{
		ID:          "sync_after_test",
		Rule:        "bts-implement-protocol.md §Execution Rules rule 5",
		Enforcement: "internal/hook/stop.go:handleImplementDone",
		Summary:     "IMPLEMENT DONE blocked unless deviation.md exists",
	},
	{
		ID:          "review_before_done",
		Rule:        "bts-implement-protocol.md (implied by handleImplementDone)",
		Enforcement: "internal/hook/stop.go:handleImplementDone",
		Summary:     "IMPLEMENT DONE blocked unless review.md exists",
	},
	{
		ID:          "phase_transitions_logged",
		Rule:        "bts-implement-protocol.md §Execution Rules rule 7",
		Enforcement: "internal/cli/recipe.go:recipeLogCmd phase flag",
		Summary:     "Phase updates flow through the log command; metrics append captures every change",
	},
	{
		ID:          "tracked_status_writes",
		Rule:        "bts-implement-protocol.md §Execution Rules rule 6",
		Enforcement: "internal/engine/validator.go:validateTasksJSON",
		Summary:     "tasks.json schema validated; status enum forces the state machine",
	},
}

// InvariantGates lists domain-level checks enforced via `bts verify`.
// They surface as critical findings but operate through the standard
// verification pipeline rather than a dedicated hook.
var InvariantGates = []HardGate{
	{
		ID:          "domain_before_wireframe",
		Rule:        "bts-recipe-protocol.md §Mandatory Rules rule 10",
		Enforcement: "internal/cli/recipe.go:checkPhasePreConditions(wireframe)",
		Summary:     "domain.md must exist for blueprint/design recipes before phase=wireframe (strict)",
	},
	{
		ID:          "invariant_single_owner",
		Rule:        "bts-domain-model/SKILL.md §Quality Gate",
		Enforcement: "internal/engine/domain_checker.go:CheckInvariantOwnership",
		Summary:     "Every invariant in domain.md §2 must have exactly one owner; duplicates raise critical",
	},
}
