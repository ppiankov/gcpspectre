package report

import "fmt"

// ScanError represents a structured error from a scanner.
type ScanError struct {
	Scanner      string `json:"scanner"`
	Message      string `json:"message"`
	ResourceType string `json:"resource_type,omitempty"`
	Recoverable  bool   `json:"recoverable"`
}

// String returns a human-readable representation.
func (e ScanError) String() string {
	if e.ResourceType != "" {
		return fmt.Sprintf("%s (%s): %s", e.Scanner, e.ResourceType, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Scanner, e.Message)
}
