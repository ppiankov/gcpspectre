package commands

import (
	"errors"
	"strings"
	"testing"
)

func TestEnhanceErrorCredentials(t *testing.T) {
	err := enhanceError("scan", errors.New("could not find default credentials"))
	if !strings.Contains(err.Error(), "gcloud auth") {
		t.Error("expected gcloud auth hint for missing credentials")
	}
}

func TestEnhanceErrorForbidden(t *testing.T) {
	err := enhanceError("scan", errors.New("403 Forbidden"))
	if !strings.Contains(err.Error(), "Compute Viewer") {
		t.Error("expected permissions hint for 403")
	}
}

func TestEnhanceErrorRateLimit(t *testing.T) {
	err := enhanceError("scan", errors.New("429 RESOURCE_EXHAUSTED"))
	if !strings.Contains(err.Error(), "rate limit") {
		t.Error("expected rate limit hint for 429")
	}
}

func TestEnhanceErrorNotFound(t *testing.T) {
	err := enhanceError("scan", errors.New("404 notFound"))
	if !strings.Contains(err.Error(), "APIs are enabled") {
		t.Error("expected API hint for 404")
	}
}

func TestEnhanceErrorOAuth(t *testing.T) {
	err := enhanceError("scan", errors.New("oauth2: cannot fetch token"))
	if !strings.Contains(err.Error(), "expired") {
		t.Error("expected token refresh hint")
	}
}

func TestEnhanceErrorGeneric(t *testing.T) {
	err := enhanceError("scan", errors.New("something went wrong"))
	if strings.Contains(err.Error(), "hint:") {
		t.Error("generic errors should not have hints")
	}
	if !strings.Contains(err.Error(), "scan:") {
		t.Error("expected action prefix")
	}
}

func TestParseExcludeLabels(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    map[string]string
		wantNil bool
	}{
		{"nil input", nil, nil, true},
		{"empty input", []string{}, nil, true},
		{"key=value", []string{"env=prod"}, map[string]string{"env": "prod"}, false},
		{"key-only", []string{"do-not-scan"}, map[string]string{"do-not-scan": ""}, false},
		{"mixed", []string{"env=prod", "do-not-scan"}, map[string]string{"env": "prod", "do-not-scan": ""}, false},
		{"value with equals", []string{"note=a=b"}, map[string]string{"note": "a=b"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseExcludeLabels(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("got %v, want nil", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("len = %d, want %d", len(got), len(tt.want))
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("got[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestParseResourceIDs(t *testing.T) {
	if got := parseResourceIDs(nil); got != nil {
		t.Errorf("nil input: got %v, want nil", got)
	}
	if got := parseResourceIDs([]string{}); got != nil {
		t.Errorf("empty input: got %v, want nil", got)
	}

	got := parseResourceIDs([]string{"123", "456"})
	if !got["123"] || !got["456"] {
		t.Errorf("expected both IDs present, got %v", got)
	}
	if got["789"] {
		t.Error("unexpected ID present")
	}
}

func TestMergeExcludeLabels(t *testing.T) {
	// nil + nil
	if got := mergeExcludeLabels(nil, nil); got != nil {
		t.Errorf("nil+nil: got %v, want nil", got)
	}

	// config only
	got := mergeExcludeLabels([]string{"env=prod"}, nil)
	if got["env"] != "prod" {
		t.Errorf("config only: got %v", got)
	}

	// flag only
	got = mergeExcludeLabels(nil, []string{"team=infra"})
	if got["team"] != "infra" {
		t.Errorf("flag only: got %v", got)
	}

	// union
	got = mergeExcludeLabels([]string{"env=prod"}, []string{"team=infra", "do-not-scan"})
	if len(got) != 3 {
		t.Errorf("union: len = %d, want 3", len(got))
	}
	if got["do-not-scan"] != "" {
		t.Errorf("key-only: got %q, want empty", got["do-not-scan"])
	}

	// flag overrides config for same key
	got = mergeExcludeLabels([]string{"env=prod"}, []string{"env=staging"})
	if got["env"] != "staging" {
		t.Errorf("override: got %q, want staging", got["env"])
	}
}

func TestConvertToScanErrors(t *testing.T) {
	if got := convertToScanErrors(nil); got != nil {
		t.Error("nil input should return nil")
	}

	errors := convertToScanErrors([]string{
		"my-project/compute_instance: timeout",
		"proj: general error",
		"bare message",
	})
	if len(errors) != 3 {
		t.Fatalf("len = %d, want 3", len(errors))
	}

	e0 := errors[0]
	if e0.Scanner != "my-project" || e0.ResourceType != "compute_instance" || e0.Message != "timeout" {
		t.Errorf("e0 = %+v", e0)
	}
	if !e0.Recoverable {
		t.Error("expected recoverable=true")
	}

	e1 := errors[1]
	if e1.Scanner != "proj" || e1.ResourceType != "" || e1.Message != "general error" {
		t.Errorf("e1 = %+v", e1)
	}

	e2 := errors[2]
	if e2.Scanner != "unknown" || e2.Message != "bare message" {
		t.Errorf("e2 = %+v", e2)
	}
}

func TestComputeTargetHash(t *testing.T) {
	hash1 := computeTargetHash([]string{"project-a", "project-b"})
	hash2 := computeTargetHash([]string{"project-a", "project-b"})
	hash3 := computeTargetHash([]string{"project-c"})

	if !strings.HasPrefix(hash1, "sha256:") {
		t.Error("expected sha256: prefix")
	}
	if hash1 != hash2 {
		t.Error("same input should produce same hash")
	}
	if hash1 == hash3 {
		t.Error("different input should produce different hash")
	}
}
