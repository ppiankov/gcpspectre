package commands

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/ppiankov/gcpspectre/internal/report"
)

// ExitCodeError signals a non-zero exit code without being a runtime error.
type ExitCodeError struct {
	Code int
}

func (e ExitCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
}

// enhanceError wraps an error with context and suggestions for common GCP issues.
func enhanceError(action string, err error) error {
	msg := err.Error()

	var hint string
	switch {
	case strings.Contains(msg, "could not find default credentials"):
		hint = "Configure GCP credentials: run 'gcloud auth application-default login' or set GOOGLE_APPLICATION_CREDENTIALS"
	case strings.Contains(msg, "oauth2: cannot fetch token"):
		hint = "GCP credentials expired. Run 'gcloud auth application-default login' to refresh"
	case strings.Contains(msg, "403") || strings.Contains(msg, "Forbidden"):
		hint = "Insufficient permissions. Ensure your account has Compute Viewer and Monitoring Viewer roles"
	case strings.Contains(msg, "429") || strings.Contains(msg, "RESOURCE_EXHAUSTED"):
		hint = "GCP API rate limit hit. Retry with fewer projects or increase timeout"
	case strings.Contains(msg, "404") || strings.Contains(msg, "notFound"):
		hint = "Resource or API not found. Verify the project ID and that required APIs are enabled"
	}

	if hint != "" {
		return fmt.Errorf("%s: %w\n  hint: %s", action, err, hint)
	}
	return fmt.Errorf("%s: %w", action, err)
}

// parseExcludeLabels converts a slice of "key=value" or "key" strings to a map.
// Key-only entries (no "=") become key→"" which triggers key-only matching.
func parseExcludeLabels(labels []string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	result := make(map[string]string, len(labels))
	for _, l := range labels {
		k, v, _ := strings.Cut(l, "=")
		result[k] = v
	}
	return result
}

// parseResourceIDs converts a slice of resource ID strings to a lookup map.
func parseResourceIDs(ids []string) map[string]bool {
	if len(ids) == 0 {
		return nil
	}
	result := make(map[string]bool, len(ids))
	for _, id := range ids {
		result[id] = true
	}
	return result
}

// mergeExcludeLabels combines config and CLI label exclusions into one map.
func mergeExcludeLabels(configLabels, flagLabels []string) map[string]string {
	combined := append(configLabels, flagLabels...)
	return parseExcludeLabels(combined)
}

// convertToScanErrors parses scanner error strings into structured ScanError values.
// Scanner errors follow the format "project/scanner_type: message".
func convertToScanErrors(errors []string) []report.ScanError {
	if len(errors) == 0 {
		return nil
	}
	result := make([]report.ScanError, 0, len(errors))
	for _, e := range errors {
		se := report.ScanError{Recoverable: true}
		scanner, msg, found := strings.Cut(e, ": ")
		if found {
			se.Message = msg
			// scanner may be "project/resource_type" or just "project"
			project, resType, hasSep := strings.Cut(scanner, "/")
			if hasSep {
				se.Scanner = project
				se.ResourceType = resType
			} else {
				se.Scanner = project
			}
		} else {
			se.Scanner = "unknown"
			se.Message = e
		}
		result = append(result, se)
	}
	return result
}

// computeTargetHash generates a SHA256 hash for the target URI.
func computeTargetHash(projectList []string) string {
	input := fmt.Sprintf("projects:%s", strings.Join(projectList, ","))
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("sha256:%x", h)
}
