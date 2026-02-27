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
