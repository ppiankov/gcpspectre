package gcp

import "testing"

func TestShouldExcludeLabels(t *testing.T) {
	tests := []struct {
		name     string
		resource map[string]string
		exclude  map[string]string
		want     bool
	}{
		{"no exclude labels", map[string]string{"env": "prod"}, nil, false},
		{"matching label", map[string]string{"env": "prod"}, map[string]string{"env": "prod"}, true},
		{"non-matching label", map[string]string{"env": "prod"}, map[string]string{"env": "dev"}, false},
		{"missing label key", map[string]string{"team": "infra"}, map[string]string{"env": "prod"}, false},
		{"nil resource labels", nil, map[string]string{"env": "prod"}, false},
		{"key-only match", map[string]string{"do-not-scan": "true"}, map[string]string{"do-not-scan": ""}, true},
		{"key-only no match", map[string]string{"team": "infra"}, map[string]string{"do-not-scan": ""}, false},
		{"key-only empty value", map[string]string{"do-not-scan": ""}, map[string]string{"do-not-scan": ""}, true},
		{"mixed key-only and value", map[string]string{"env": "prod", "team": "infra"}, map[string]string{"env": "dev", "team": ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldExcludeLabels(tt.resource, tt.exclude)
			if got != tt.want {
				t.Errorf("shouldExcludeLabels = %v, want %v", got, tt.want)
			}
		})
	}
}
