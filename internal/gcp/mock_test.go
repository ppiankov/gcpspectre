package gcp

import "context"

// mockComputeAPI is a test double for ComputeAPI.
type mockComputeAPI struct {
	instances       []ComputeInstance
	disks           []PersistentDisk
	addresses       []StaticAddress
	snapshots       []DiskSnapshot
	instanceGroups  []InstanceGroupInfo
	firewalls       []FirewallRule
	routers         []RouterInfo
	forwardingRules []ForwardingRuleInfo
	backendServices []BackendServiceInfo
	err             error
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

func (m *mockComputeAPI) ListInstanceGroups(_ context.Context, _ string) ([]InstanceGroupInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.instanceGroups, nil
}

func (m *mockComputeAPI) ListFirewalls(_ context.Context, _ string) ([]FirewallRule, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.firewalls, nil
}

func (m *mockComputeAPI) ListRouters(_ context.Context, _ string) ([]RouterInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.routers, nil
}

func (m *mockComputeAPI) ListForwardingRules(_ context.Context, _ string) ([]ForwardingRuleInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.forwardingRules, nil
}

func (m *mockComputeAPI) ListBackendServices(_ context.Context, _ string) ([]BackendServiceInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.backendServices, nil
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

// mockCloudSQLAPI is a test double for CloudSQLAPI.
type mockCloudSQLAPI struct {
	instances []CloudSQLInstance
	err       error
}

func (m *mockCloudSQLAPI) ListInstances(_ context.Context, _ string) ([]CloudSQLInstance, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.instances, nil
}

// mockCloudFunctionsAPI is a test double for CloudFunctionsAPI.
type mockCloudFunctionsAPI struct {
	functions []CloudFunction
	err       error
}

func (m *mockCloudFunctionsAPI) ListFunctions(_ context.Context, _ string) ([]CloudFunction, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.functions, nil
}

// mockPubSubAPI is a test double for PubSubAPI.
type mockPubSubAPI struct {
	topics        []PubSubTopic
	subscriptions []PubSubSubscription
	err           error
}

func (m *mockPubSubAPI) ListTopics(_ context.Context, _ string) ([]PubSubTopic, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.topics, nil
}

func (m *mockPubSubAPI) ListSubscriptions(_ context.Context, _ string) ([]PubSubSubscription, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.subscriptions, nil
}

// mockMonitoringAPIMulti is a test double that returns different results per metric type.
type mockMonitoringAPIMulti struct {
	results map[string]map[string]float64
	err     error
}

func (m *mockMonitoringAPIMulti) FetchMetricMean(_ context.Context, _, metricType, _ string, _ []string, _ int) (map[string]float64, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.results == nil {
		return nil, nil
	}
	return m.results[metricType], nil
}
