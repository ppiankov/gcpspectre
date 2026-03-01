package gcp

import "log/slog"

// shouldExcludeLabels checks if a resource should be excluded based on label matching.
// An empty value in excludeLabels means "match any resource with this key" (key-only match).
func shouldExcludeLabels(resourceLabels, excludeLabels map[string]string) bool {
	if len(excludeLabels) == 0 {
		return false
	}
	for k, v := range excludeLabels {
		resVal, exists := resourceLabels[k]
		if v == "" {
			if exists {
				slog.Debug("Excluding resource by label key", "key", k)
				return true
			}
		} else {
			if resVal == v {
				slog.Debug("Excluding resource by label", "key", k, "value", v)
				return true
			}
		}
	}
	return false
}
