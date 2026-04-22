package engine

import (
	"os"
	"regexp"
	"strings"
)

// architectDecisionBlockRe captures the architect-decision block.
// Requires both opening and closing tags on their own lines (the skill
// instructs authors to write them that way; any other shape is rejected
// so the block is unambiguously parseable later).
var architectDecisionBlockRe = regexp.MustCompile(
	`(?s)<!--\s*architect-decision\s*-->\s*\n(.*?)<!--\s*/architect-decision\s*-->`,
)

// architectSelectedRe captures the Selected: line inside the block.
var architectSelectedRe = regexp.MustCompile(`(?m)^\s*Selected:\s*(\S[^\r\n]*)`)

// CheckArchitectDecisionHeader enforces the wireframe.md architect-decision
// contract (Phase 5.3). Per bts-architect SKILL.md Step 4, wireframe.md
// must carry a `<!-- architect-decision -->` block with a `Selected:`
// line naming the chosen decomposition.
//
// Severity is major (not critical): legacy wireframes authored before
// Phase 5 will lack the block, and we'd rather make the signal visible
// during /verify than hard-block every historical recipe. The CLI
// precondition for phase=architect advances can promote this to a
// hard gate once migration is complete.
func CheckArchitectDecisionHeader(wireframePath string) []Issue {
	data, err := os.ReadFile(wireframePath)
	if err != nil {
		return nil
	}
	content := string(data)

	matches := architectDecisionBlockRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return []Issue{{
			Category: "architect_decision",
			Claim:    "missing_architect_decision_block",
			Severity: "major",
			Detail:   "wireframe.md has no <!-- architect-decision --> block. Run /bts-architect to propose and select a decomposition, then commit the block per bts-architect SKILL.md Step 4. Skip-architect recipes (tiny scope) still need a minimal block declaring Selected: single-path.",
		}}
	}
	if len(matches) > 1 {
		return []Issue{{
			Category: "architect_decision",
			Claim:    "duplicate_architect_decision_block",
			Severity: "major",
			Detail:   "wireframe.md has multiple <!-- architect-decision --> blocks; exactly one is required.",
		}}
	}

	body := matches[0][1]
	selected := architectSelectedRe.FindStringSubmatch(body)
	if len(selected) < 2 || strings.TrimSpace(selected[1]) == "" {
		return []Issue{{
			Category: "architect_decision",
			Claim:    "architect_decision_missing_selected",
			Severity: "major",
			Detail:   "architect-decision block is present but has no `Selected:` line naming the chosen alternative.",
		}}
	}
	return nil
}
