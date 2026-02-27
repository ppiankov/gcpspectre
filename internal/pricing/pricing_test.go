package pricing

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

func TestMonthlyInstanceCost(t *testing.T) {
	cost := MonthlyInstanceCost("e2-medium", "us-central1-a")
	if cost == 0 {
		t.Error("expected non-zero cost for e2-medium in us-central1-a")
	}
	expected := 0.0335 * hoursPerMonth
	if !almostEqual(cost, expected) {
		t.Errorf("MonthlyInstanceCost = %f, want ~%f", cost, expected)
	}
}

func TestMonthlyInstanceCostUnknown(t *testing.T) {
	cost := MonthlyInstanceCost("unknown-type", "us-central1-a")
	if cost != 0 {
		t.Errorf("expected 0 for unknown machine type, got %f", cost)
	}
}

func TestMonthlyInstanceCostFallbackRegion(t *testing.T) {
	cost := MonthlyInstanceCost("e2-medium", "unknown-region-a")
	expected := 0.0335 * hoursPerMonth
	if !almostEqual(cost, expected) {
		t.Errorf("expected fallback to us-central1, got %f (want ~%f)", cost, expected)
	}
}

func TestMonthlyDiskCost(t *testing.T) {
	cost := MonthlyDiskCost("pd-ssd", 100, "us-central1-a")
	expected := 0.17 * 100
	if cost != expected {
		t.Errorf("MonthlyDiskCost = %f, want %f", cost, expected)
	}
}

func TestMonthlyDiskCostUnknown(t *testing.T) {
	cost := MonthlyDiskCost("pd-unknown", 100, "us-central1-a")
	if cost != 0 {
		t.Errorf("expected 0 for unknown disk type, got %f", cost)
	}
}

func TestMonthlyAddressCost(t *testing.T) {
	cost := MonthlyAddressCost("us-central1")
	if cost != 7.30 {
		t.Errorf("MonthlyAddressCost = %f, want 7.30", cost)
	}
}

func TestMonthlyAddressCostFallback(t *testing.T) {
	cost := MonthlyAddressCost("unknown-region")
	if cost != 7.30 {
		t.Errorf("expected fallback to us-central1, got %f", cost)
	}
}

func TestMonthlySnapshotCost(t *testing.T) {
	cost := MonthlySnapshotCost(200, "us-central1")
	expected := 0.026 * 200
	if cost != expected {
		t.Errorf("MonthlySnapshotCost = %f, want %f", cost, expected)
	}
}

func TestMonthlyCloudSQLCost(t *testing.T) {
	cost := MonthlyCloudSQLCost("db-f1-micro", "us-central1")
	expected := 0.0150 * hoursPerMonth
	if cost != expected {
		t.Errorf("MonthlyCloudSQLCost = %f, want %f", cost, expected)
	}
}

func TestMonthlyCloudSQLCostUnknown(t *testing.T) {
	cost := MonthlyCloudSQLCost("db-unknown", "us-central1")
	if cost != 0 {
		t.Errorf("expected 0 for unknown tier, got %f", cost)
	}
}

func TestRegionFromZone(t *testing.T) {
	tests := []struct {
		zone, want string
	}{
		{"us-central1-a", "us-central1"},
		{"europe-west1-b", "europe-west1"},
		{"asia-east1-c", "asia-east1"},
		{"us-central1", "us-central1"},
	}
	for _, tt := range tests {
		got := regionFromZone(tt.zone)
		if got != tt.want {
			t.Errorf("regionFromZone(%q) = %q, want %q", tt.zone, got, tt.want)
		}
	}
}
