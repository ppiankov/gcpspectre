package commands

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

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

// computeTargetHash generates a SHA256 hash for the target URI.
func computeTargetHash(projectList []string) string {
	input := fmt.Sprintf("projects:%s", strings.Join(projectList, ","))
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("sha256:%x", h)
}
