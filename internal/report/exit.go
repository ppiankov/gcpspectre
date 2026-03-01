package report

import gcptype "github.com/ppiankov/gcpspectre/internal/gcp"

const (
	ExitOK                = 0
	ExitThresholdExceeded = 1
)

// ComputeExitCode determines the process exit code based on findings and thresholds.
// Returns ExitOK if failOn is empty (backward-compatible default).
func ComputeExitCode(findings []gcptype.Finding, failOn string, threshold int) int {
	if failOn == "" {
		return ExitOK
	}
	minSeverity := parseSeverity(failOn)
	count := 0
	for _, f := range findings {
		if severityRank(f.Severity) >= severityRank(minSeverity) {
			count++
		}
	}
	if count >= threshold {
		return ExitThresholdExceeded
	}
	return ExitOK
}

func severityRank(s gcptype.Severity) int {
	switch s {
	case gcptype.SeverityLow:
		return 1
	case gcptype.SeverityMedium:
		return 2
	case gcptype.SeverityHigh:
		return 3
	default:
		return 0
	}
}

func parseSeverity(s string) gcptype.Severity {
	switch s {
	case "high":
		return gcptype.SeverityHigh
	case "medium":
		return gcptype.SeverityMedium
	case "low":
		return gcptype.SeverityLow
	default:
		return gcptype.SeverityLow
	}
}
