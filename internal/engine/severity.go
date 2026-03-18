package engine

// Severity levels for verification findings.
const (
	SeverityCritical = "critical" // References non-existent files/functions
	SeverityMajor    = "major"    // Logical inconsistency, incorrect signatures
	SeverityMinor    = "minor"    // Approximately correct but imprecise
	SeverityInfo     = "info"     // Improvement suggestion
)

// IsBlocking returns true if this severity blocks convergence.
func IsBlocking(severity string) bool {
	return severity == SeverityCritical || severity == SeverityMajor
}
