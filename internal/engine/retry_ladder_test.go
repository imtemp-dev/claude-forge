package engine

import (
	"testing"
)

func TestClassifyBuildError_Syntactic(t *testing.T) {
	cases := []string{
		"error TS2345: Argument of type 'number' is not assignable to parameter of type 'string'",
		"SyntaxError: unexpected token",
		"cannot find module '@foo/bar'",
		"ModuleNotFoundError: No module named 'foo'",
		"expected 'func'",
	}
	for _, c := range cases {
		if got := ClassifyBuildError(c, "go"); got != ErrorSyntactic {
			t.Errorf("ClassifyBuildError(%q) = %q, want syntactic", c, got)
		}
	}
}

func TestClassifyBuildError_Semantic(t *testing.T) {
	cases := []string{
		"assertion failed: expected 3 got 2",
		"FAIL TestAdd",
		"panic: runtime error: index out of range",
		"test_auth_refresh failed",
	}
	for _, c := range cases {
		if got := ClassifyBuildError(c, ""); got != ErrorSemantic {
			t.Errorf("ClassifyBuildError(%q) = %q, want semantic", c, got)
		}
	}
}

func TestClassifyBuildError_Empty(t *testing.T) {
	if got := ClassifyBuildError("", ""); got != ErrorUnknown {
		t.Errorf("empty error → want unknown, got %q", got)
	}
}

func TestNextRetryDecision_Tier1Retries(t *testing.T) {
	cfg := DefaultLadder()
	d := NextRetryDecision(1, 0, ErrorSyntactic, cfg)
	if d.Action != ActionRetryInplace || d.NextTier != 1 {
		t.Errorf("tier 1 syntactic should retry in place, got %+v", d)
	}
}

// Tier 1 exhaustion → tier 2 strategy switch.
func TestNextRetryDecision_Tier1Exhausted(t *testing.T) {
	cfg := DefaultLadder()
	d := NextRetryDecision(1, cfg.SyntacticMax, ErrorSyntactic, cfg)
	if d.Action != ActionStrategySwitch || d.NextTier != 2 {
		t.Errorf("tier 1 exhausted → tier 2 switch, got %+v", d)
	}
}

// Non-syntactic error at tier 1 jumps straight to semantic tier.
func TestNextRetryDecision_Tier1NonSyntactic(t *testing.T) {
	cfg := DefaultLadder()
	d := NextRetryDecision(1, 0, ErrorSemantic, cfg)
	if d.Action != ActionStrategySwitch || d.NextTier != 2 {
		t.Errorf("semantic at tier 1 → tier 2, got %+v", d)
	}
}

// Tier 2 budget exhaustion moves to spec escalation.
func TestNextRetryDecision_Tier2ToSpec(t *testing.T) {
	cfg := DefaultLadder()
	d := NextRetryDecision(2, cfg.SemanticMax, ErrorSemantic, cfg)
	if d.Action != ActionSpecEscalate || d.NextTier != 3 {
		t.Errorf("tier 2 exhausted → spec escalate, got %+v", d)
	}
}

// Disabling spec_escalate skips tier 3.
func TestNextRetryDecision_SkipsDisabledSpec(t *testing.T) {
	cfg := DefaultLadder()
	cfg.SpecEscalate = false
	d := NextRetryDecision(2, cfg.SemanticMax, ErrorSemantic, cfg)
	if d.Action != ActionDomainEscalate || d.NextTier != 4 {
		t.Errorf("disabled spec → jump to domain, got %+v", d)
	}
}

// Tier 5 (architect) exhaustion → block.
func TestNextRetryDecision_Tier5ToBlock(t *testing.T) {
	cfg := DefaultLadder()
	d := NextRetryDecision(5, 1, ErrorSemantic, cfg)
	if d.Action != ActionBlock || d.NextTier != 6 {
		t.Errorf("tier 5 → block, got %+v", d)
	}
}

// Disabling every escalation → tier 2 exhaustion drops to block.
func TestNextRetryDecision_AllEscalationsDisabled(t *testing.T) {
	cfg := LadderConfig{
		SyntacticMax:      3,
		SemanticMax:       2,
		SpecEscalate:      false,
		DomainEscalate:    false,
		ArchitectEscalate: false,
	}
	d := NextRetryDecision(2, cfg.SemanticMax, ErrorSemantic, cfg)
	if d.Action != ActionBlock {
		t.Errorf("no escalations enabled → block directly, got %+v", d)
	}
}

func TestShortErrorSignature_Truncates(t *testing.T) {
	long := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
	if got := ShortErrorSignature(long); len(got) != 60 {
		t.Errorf("expected 60 chars, got %d", len(got))
	}
	if got := ShortErrorSignature("short"); got != "short" {
		t.Errorf("short string must pass through, got %q", got)
	}
}
