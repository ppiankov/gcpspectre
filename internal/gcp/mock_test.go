package gcp

import "context"

// mockComputeAPI is a test double for ComputeAPI.
type mockComputeAPI struct {
	instances []ComputeInstance
	disks     []PersistentDisk
	addresses []StaticAddress
	snapshots []DiskSnapshot
	err       error
}

func (m *mockComputeAPI) ListInstances(_ context.Context, _ string) ([]ComputeInstance, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.instances, nil
}

func (m *mockComputeAPI) ListDisks(_ context.Context, _ string) ([]PersistentDisk, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.disks, nil
}

func (m *mockComputeAPI) ListAddresses(_ context.Context, _ string) ([]StaticAddress, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.addresses, nil
}

func (m *mockComputeAPI) ListSnapshots(_ context.Context, _ string) ([]DiskSnapshot, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.snapshots, nil
}

// mockMonitoringAPI is a test double for MonitoringAPI.
type mockMonitoringAPI struct {
	results map[string]float64
	err     error
}

func (m *mockMonitoringAPI) FetchMetricMean(_ context.Context, _, _, _ string, _ []string, _ int) (map[string]float64, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}
