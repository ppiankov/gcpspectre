package gcp

import (
	"testing"
	"time"

	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/protobuf/proto"
)

func TestLastPathSegment(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"projects/my-proj/zones/us-central1-a/machineTypes/e2-medium", "e2-medium"},
		{"projects/my-proj/zones/us-central1-a/diskTypes/pd-ssd", "pd-ssd"},
		{"projects/my-proj/zones/us-central1-a", "us-central1-a"},
		{"projects/my-proj/regions/us-central1", "us-central1"},
		{"simple", "simple"},
		{"", ""},
	}
	for _, tt := range tests {
		got := lastPathSegment(tt.input)
		if got != tt.want {
			t.Errorf("lastPathSegment(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		input  string
		isZero bool
	}{
		{"2024-01-15T10:30:00Z", false},
		{"2024-01-15T10:30:00.123456789Z", false},
		{"2024-01-15T10:30:00-07:00", false},
		{"", true},
		{"not-a-timestamp", true},
	}
	for _, tt := range tests {
		got := parseTimestamp(tt.input)
		if got.IsZero() != tt.isZero {
			t.Errorf("parseTimestamp(%q).IsZero() = %v, want %v", tt.input, got.IsZero(), tt.isZero)
		}
	}

	// Verify exact time parsing
	got := parseTimestamp("2024-06-15T12:00:00Z")
	want := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("parseTimestamp(2024-06-15T12:00:00Z) = %v, want %v", got, want)
	}
}

func TestConvertInstance(t *testing.T) {
	inst := &computepb.Instance{
		Id:                 proto.Uint64(123),
		Name:               proto.String("my-vm"),
		Zone:               proto.String("projects/p/zones/us-central1-a"),
		MachineType:        proto.String("projects/p/zones/us-central1-a/machineTypes/e2-medium"),
		Status:             proto.String("RUNNING"),
		Labels:             map[string]string{"env": "dev"},
		LastStartTimestamp: proto.String("2024-06-15T12:00:00Z"),
		CreationTimestamp:  proto.String("2024-01-01T00:00:00Z"),
		Tags:               &computepb.Tags{Items: []string{"http-server", "https-server"}},
	}

	ci := convertInstance(inst, "test-proj")

	if ci.ID != 123 {
		t.Errorf("ID = %d, want 123", ci.ID)
	}
	if ci.Name != "my-vm" {
		t.Errorf("Name = %q, want %q", ci.Name, "my-vm")
	}
	if ci.Zone != "us-central1-a" {
		t.Errorf("Zone = %q, want %q", ci.Zone, "us-central1-a")
	}
	if ci.Project != "test-proj" {
		t.Errorf("Project = %q, want %q", ci.Project, "test-proj")
	}
	if ci.MachineType != "e2-medium" {
		t.Errorf("MachineType = %q, want %q", ci.MachineType, "e2-medium")
	}
	if ci.Status != "RUNNING" {
		t.Errorf("Status = %q, want %q", ci.Status, "RUNNING")
	}
	if ci.Labels["env"] != "dev" {
		t.Errorf("Labels[env] = %q, want %q", ci.Labels["env"], "dev")
	}
	if len(ci.Tags) != 2 || ci.Tags[0] != "http-server" {
		t.Errorf("Tags = %v, want [http-server https-server]", ci.Tags)
	}
	if ci.LastStarted.IsZero() {
		t.Error("LastStarted should not be zero")
	}
	if ci.CreateTime.IsZero() {
		t.Error("CreateTime should not be zero")
	}
}

func TestConvertInstanceNilTags(t *testing.T) {
	inst := &computepb.Instance{
		Id:   proto.Uint64(1),
		Name: proto.String("no-tags"),
	}
	ci := convertInstance(inst, "p")
	if ci.Tags != nil {
		t.Errorf("Tags = %v, want nil", ci.Tags)
	}
}

func TestConvertDisk(t *testing.T) {
	disk := &computepb.Disk{
		Id:                  proto.Uint64(456),
		Name:                proto.String("my-disk"),
		Zone:                proto.String("projects/p/zones/us-east1-b"),
		Type:                proto.String("projects/p/zones/us-east1-b/diskTypes/pd-ssd"),
		SizeGb:              proto.Int64(100),
		Status:              proto.String("READY"),
		Users:               []string{"projects/p/zones/us-east1-b/instances/vm1"},
		Labels:              map[string]string{"team": "data"},
		LastAttachTimestamp: proto.String("2024-03-01T00:00:00Z"),
		CreationTimestamp:   proto.String("2024-01-01T00:00:00Z"),
	}

	pd := convertDisk(disk, "test-proj")

	if pd.ID != 456 {
		t.Errorf("ID = %d, want 456", pd.ID)
	}
	if pd.Name != "my-disk" {
		t.Errorf("Name = %q", pd.Name)
	}
	if pd.Zone != "us-east1-b" {
		t.Errorf("Zone = %q", pd.Zone)
	}
	if pd.DiskType != "pd-ssd" {
		t.Errorf("DiskType = %q", pd.DiskType)
	}
	if pd.SizeGB != 100 {
		t.Errorf("SizeGB = %d", pd.SizeGB)
	}
	if len(pd.Users) != 1 {
		t.Errorf("Users = %v", pd.Users)
	}
	if pd.Labels["team"] != "data" {
		t.Errorf("Labels = %v", pd.Labels)
	}
}

func TestConvertAddress(t *testing.T) {
	addr := &computepb.Address{
		Id:                proto.Uint64(789),
		Name:              proto.String("my-ip"),
		Region:            proto.String("projects/p/regions/us-west1"),
		Address:           proto.String("34.100.0.1"),
		Status:            proto.String("RESERVED"),
		AddressType:       proto.String("EXTERNAL"),
		CreationTimestamp: proto.String("2024-02-01T00:00:00Z"),
	}

	sa := convertAddress(addr, "test-proj")

	if sa.ID != 789 {
		t.Errorf("ID = %d", sa.ID)
	}
	if sa.Name != "my-ip" {
		t.Errorf("Name = %q", sa.Name)
	}
	if sa.Region != "us-west1" {
		t.Errorf("Region = %q", sa.Region)
	}
	if sa.Address != "34.100.0.1" {
		t.Errorf("Address = %q", sa.Address)
	}
	if sa.Status != "RESERVED" {
		t.Errorf("Status = %q", sa.Status)
	}
	if sa.AddressType != "EXTERNAL" {
		t.Errorf("AddressType = %q", sa.AddressType)
	}
}

func TestConvertSnapshot(t *testing.T) {
	snap := &computepb.Snapshot{
		Id:                proto.Uint64(101),
		Name:              proto.String("snap-1"),
		SourceDisk:        proto.String("projects/p/zones/us-central1-a/disks/disk1"),
		DiskSizeGb:        proto.Int64(50),
		StorageBytes:      proto.Int64(1073741824),
		Status:            proto.String("READY"),
		Labels:            map[string]string{"backup": "daily"},
		CreationTimestamp: proto.String("2024-04-01T00:00:00Z"),
		StorageLocations:  []string{"us"},
	}

	ds := convertSnapshot(snap, "test-proj")

	if ds.ID != 101 {
		t.Errorf("ID = %d", ds.ID)
	}
	if ds.Name != "snap-1" {
		t.Errorf("Name = %q", ds.Name)
	}
	if ds.SourceDisk != "projects/p/zones/us-central1-a/disks/disk1" {
		t.Errorf("SourceDisk = %q", ds.SourceDisk)
	}
	if ds.DiskSizeGB != 50 {
		t.Errorf("DiskSizeGB = %d", ds.DiskSizeGB)
	}
	if ds.StorageBytes != 1073741824 {
		t.Errorf("StorageBytes = %d", ds.StorageBytes)
	}
	if len(ds.StorageLocations) != 1 || ds.StorageLocations[0] != "us" {
		t.Errorf("StorageLocations = %v", ds.StorageLocations)
	}
}

func TestConvertInstanceGroup(t *testing.T) {
	ig := &computepb.InstanceGroup{
		Id:   proto.Uint64(202),
		Name: proto.String("ig-1"),
		Zone: proto.String("projects/p/zones/europe-west1-b"),
		Size: proto.Int32(3),
	}

	info := convertInstanceGroup(ig, "test-proj")

	if info.ID != 202 {
		t.Errorf("ID = %d", info.ID)
	}
	if info.Name != "ig-1" {
		t.Errorf("Name = %q", info.Name)
	}
	if info.Zone != "europe-west1-b" {
		t.Errorf("Zone = %q", info.Zone)
	}
	if info.Size != 3 {
		t.Errorf("Size = %d", info.Size)
	}
}

func TestConvertFirewall(t *testing.T) {
	fw := &computepb.Firewall{
		Id:         proto.Uint64(303),
		Name:       proto.String("allow-http"),
		Network:    proto.String("projects/p/global/networks/default"),
		Direction:  proto.String("INGRESS"),
		Priority:   proto.Int32(1000),
		TargetTags: []string{"http-server"},
		Disabled:   proto.Bool(false),
	}

	rule := convertFirewall(fw, "test-proj")

	if rule.ID != 303 {
		t.Errorf("ID = %d", rule.ID)
	}
	if rule.Name != "allow-http" {
		t.Errorf("Name = %q", rule.Name)
	}
	if rule.Network != "default" {
		t.Errorf("Network = %q", rule.Network)
	}
	if rule.Direction != "INGRESS" {
		t.Errorf("Direction = %q", rule.Direction)
	}
	if rule.Priority != 1000 {
		t.Errorf("Priority = %d", rule.Priority)
	}
	if len(rule.TargetTags) != 1 || rule.TargetTags[0] != "http-server" {
		t.Errorf("TargetTags = %v", rule.TargetTags)
	}
	if rule.Disabled {
		t.Error("Disabled should be false")
	}
}
