package engine

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Settings is a partial mapping of .bts/config/settings.yaml — only the
// fields the Go engine reads. Fields not declared here are ignored when
// the YAML is parsed, so adding skill-only keys in settings.yaml does
// not require a code change.
type Settings struct {
	Implement ImplementSettings `yaml:"implement"`
	Simulate  SimulateSettings  `yaml:"simulate"`
	Verify    VerifySettings    `yaml:"verify"`
}

// ImplementSettings controls the implementation loop's retry limits and
// mid-run review cadence. These were previously hard-coded in three
// places (settings.yaml prose, bts-implement SKILL.md, protocol table);
// this struct is now the single source.
type ImplementSettings struct {
	MaxBuildRetries    int `yaml:"max_build_retries"`
	MaxTestIterations  int `yaml:"max_test_iterations"`
	MidrunReviewEvery  int `yaml:"midrun_review_every"`
}

// SimulateSettings captures the simulation checker thresholds.
type SimulateSettings struct {
	MinScenarios        int     `yaml:"min_scenarios"`
	CrossBoundaryRatio  float64 `yaml:"cross_boundary_ratio"`
}

// VerifySettings captures verify loop controls read by Go code. Fields
// not referenced from Go stay undeclared here and pass through the
// parser untouched.
type VerifySettings struct {
	MaxIterations int `yaml:"max_iterations"`
}

// DefaultSettings returns the built-in defaults matching the comments
// in the settings.yaml template. Callers on projects without an
// explicit settings.yaml receive this struct.
func DefaultSettings() *Settings {
	return &Settings{
		Implement: ImplementSettings{
			MaxBuildRetries:   5,
			MaxTestIterations: 5,
			MidrunReviewEvery: 5,
		},
		Simulate: SimulateSettings{
			MinScenarios:       5,
			CrossBoundaryRatio: DefaultCrossBoundaryRatio,
		},
		Verify: VerifySettings{
			MaxIterations: 3,
		},
	}
}

// LoadSettings reads .bts/config/settings.yaml under the given project
// root. Missing file → DefaultSettings (no error). Malformed YAML →
// error. Present file overrides fields individually; any field left
// zero in the YAML keeps the default.
func LoadSettings(root string) (*Settings, error) {
	path := filepath.Join(root, ".bts", "config", "settings.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultSettings(), nil
	}
	if err != nil {
		return nil, err
	}

	s := DefaultSettings()
	if err := yaml.Unmarshal(data, s); err != nil {
		return nil, err
	}
	// Restore defaults for zero-valued fields that the user left out.
	// yaml.Unmarshal of an absent key leaves the default intact, but
	// an explicit "max_build_retries: 0" would zero it — treat that as
	// an opt-in default instead of a silent "no retries".
	def := DefaultSettings()
	if s.Implement.MaxBuildRetries <= 0 {
		s.Implement.MaxBuildRetries = def.Implement.MaxBuildRetries
	}
	if s.Implement.MaxTestIterations <= 0 {
		s.Implement.MaxTestIterations = def.Implement.MaxTestIterations
	}
	if s.Implement.MidrunReviewEvery < 0 {
		s.Implement.MidrunReviewEvery = def.Implement.MidrunReviewEvery
	}
	if s.Simulate.MinScenarios <= 0 {
		s.Simulate.MinScenarios = def.Simulate.MinScenarios
	}
	if s.Simulate.CrossBoundaryRatio <= 0 {
		s.Simulate.CrossBoundaryRatio = def.Simulate.CrossBoundaryRatio
	}
	if s.Verify.MaxIterations <= 0 {
		s.Verify.MaxIterations = def.Verify.MaxIterations
	}
	return s, nil
}
