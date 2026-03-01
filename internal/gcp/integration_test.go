//go:build integration

package gcp

import (
	"context"
	"testing"
	"time"
)

func TestFullPipeline(t *testing.T) {
	now := time.Now().UTC()

	compute := &mockComputeAPI{
		instances: []ComputeInstance{
			{ID: 1, Name: "idle-vm", Zone: "us-central1-a", MachineType: "e2-medium", Status: "RUNNING"},
			{ID: 2, Name: "stopped-vm", Zone: "us-central1-a", MachineType: "e2-medium", Status: "STOPPED", LastStarted: now.AddDate(0, 0, -60)},
			{ID: 3, Name: "busy-vm", Zone: "us-central1-a", MachineType: "e2-medium", Status: "RUNNING"},
		},
		disks: []PersistentDisk{
			{ID: 10, Name: "orphan-disk", Zone: "us-central1-a", DiskType: "pd-ssd", SizeGB: 100, Status: "READY", CreateTime: now.AddDate(0, 0, -30)},
			{ID: 11, Name: "attached-disk", Zone: "us-central1-a", DiskType: "pd-standard", SizeGB: 50, Status: "READY", Users: []string{"instances/busy-vm"}},
		},
		addresses: []StaticAddress{
			{ID: 20, Name: "unused-ip", Region: "us-central1", Status: "RESERVED", AddressType: "EXTERNAL"},
			{ID: 21, Name: "in-use-ip", Region: "us-central1", Status: "IN_USE", Users: []string{"instances/busy-vm"}},
		},
		snapshots: []DiskSnapshot{
			{ID: 30, Name: "old-snap", DiskSizeGB: 50, Status: "READY", CreateTime: now.AddDate(0, 0, -120), StorageLocations: []string{"us"}},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{
			"1": 0.02, // 2% CPU — idle
			"3": 0.60, // 60% CPU — busy
		},
	}
	cloudSQL := &mockCloudSQLAPI{
		instances: []CloudSQLInstance{
			{Name: "idle-db", Region: "us-central1", Tier: "db-f1-micro", State: "RUNNABLE", DatabaseVersion: "MYSQL_8_0"},
		},
	}

	scanner := NewMultiProjectScanner(compute, monitoring, cloudSQL, []string{"test-project"}, 1, ScanConfig{IdleDays: 7, StaleDays: 90})
	result, err := scanner.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("ScanAll: %v", err)
	}

	// 3 instances + 2 disks + 2 addresses + 1 snapshot + 1 CloudSQL = 9 resources
	if result.ResourcesScanned < 7 {
		t.Errorf("ResourcesScanned = %d, expected >= 7", result.ResourcesScanned)
	}

	// Expected findings: idle-vm (idle), stopped-vm (stopped), orphan-disk (detached),
	// unused-ip (reserved), old-snap (stale) = at least 5
	if len(result.Findings) < 4 {
		t.Errorf("Findings = %d, expected >= 4", len(result.Findings))
	}

	// Verify specific finding types are present
	findingTypes := make(map[FindingID]bool)
	for _, f := range result.Findings {
		findingTypes[f.ID] = true
	}

	mustHave := []FindingID{FindingIdleInstance, FindingStoppedInstance, FindingDetachedDisk, FindingUnusedAddress, FindingStaleSnapshot}
	for _, id := range mustHave {
		if !findingTypes[id] {
			t.Errorf("missing expected finding type: %s", id)
		}
	}

	// Verify findings reference correct projects
	for _, f := range result.Findings {
		if f.Project != "test-project" {
			t.Errorf("finding %s has project %q, want test-project", f.ID, f.Project)
		}
	}

	// Verify no errors in scan
	if len(result.Errors) > 0 {
		t.Errorf("unexpected scan errors: %v", result.Errors)
	}
}
