package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/iterator"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GCPComputeClient implements ComputeAPI using the GCP Compute Engine SDK.
type GCPComputeClient struct {
	instances      *compute.InstancesClient
	disks          *compute.DisksClient
	addresses      *compute.AddressesClient
	snapshots      *compute.SnapshotsClient
	instanceGroups *compute.InstanceGroupsClient
	firewalls      *compute.FirewallsClient
}

// NewComputeClient creates a ComputeAPI backed by GCP Application Default Credentials.
func NewComputeClient(ctx context.Context) (*GCPComputeClient, error) {
	instances, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create instances client: %w", err)
	}
	disks, err := compute.NewDisksRESTClient(ctx)
	if err != nil {
		_ = instances.Close()
		return nil, fmt.Errorf("create disks client: %w", err)
	}
	addresses, err := compute.NewAddressesRESTClient(ctx)
	if err != nil {
		_ = instances.Close()
		_ = disks.Close()
		return nil, fmt.Errorf("create addresses client: %w", err)
	}
	snapshots, err := compute.NewSnapshotsRESTClient(ctx)
	if err != nil {
		_ = instances.Close()
		_ = disks.Close()
		_ = addresses.Close()
		return nil, fmt.Errorf("create snapshots client: %w", err)
	}
	instanceGroups, err := compute.NewInstanceGroupsRESTClient(ctx)
	if err != nil {
		_ = instances.Close()
		_ = disks.Close()
		_ = addresses.Close()
		_ = snapshots.Close()
		return nil, fmt.Errorf("create instance groups client: %w", err)
	}
	firewalls, err := compute.NewFirewallsRESTClient(ctx)
	if err != nil {
		_ = instances.Close()
		_ = disks.Close()
		_ = addresses.Close()
		_ = snapshots.Close()
		_ = instanceGroups.Close()
		return nil, fmt.Errorf("create firewalls client: %w", err)
	}
	return &GCPComputeClient{
		instances:      instances,
		disks:          disks,
		addresses:      addresses,
		snapshots:      snapshots,
		instanceGroups: instanceGroups,
		firewalls:      firewalls,
	}, nil
}

// Close releases all underlying client connections.
func (c *GCPComputeClient) Close() {
	_ = c.instances.Close()
	_ = c.disks.Close()
	_ = c.addresses.Close()
	_ = c.snapshots.Close()
	_ = c.instanceGroups.Close()
	_ = c.firewalls.Close()
}

func (c *GCPComputeClient) ListInstances(ctx context.Context, project string) ([]ComputeInstance, error) {
	var result []ComputeInstance
	it := c.instances.AggregatedList(ctx, &computepb.AggregatedListInstancesRequest{Project: project})
	for {
		pair, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list instances: %w", err)
		}
		for _, inst := range pair.Value.GetInstances() {
			result = append(result, convertInstance(inst, project))
		}
	}
	return result, nil
}

func (c *GCPComputeClient) ListDisks(ctx context.Context, project string) ([]PersistentDisk, error) {
	var result []PersistentDisk
	it := c.disks.AggregatedList(ctx, &computepb.AggregatedListDisksRequest{Project: project})
	for {
		pair, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list disks: %w", err)
		}
		for _, disk := range pair.Value.GetDisks() {
			result = append(result, convertDisk(disk, project))
		}
	}
	return result, nil
}

func (c *GCPComputeClient) ListAddresses(ctx context.Context, project string) ([]StaticAddress, error) {
	var result []StaticAddress
	it := c.addresses.AggregatedList(ctx, &computepb.AggregatedListAddressesRequest{Project: project})
	for {
		pair, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list addresses: %w", err)
		}
		for _, addr := range pair.Value.GetAddresses() {
			result = append(result, convertAddress(addr, project))
		}
	}
	return result, nil
}

func (c *GCPComputeClient) ListSnapshots(ctx context.Context, project string) ([]DiskSnapshot, error) {
	var result []DiskSnapshot
	it := c.snapshots.List(ctx, &computepb.ListSnapshotsRequest{Project: project})
	for {
		snap, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list snapshots: %w", err)
		}
		result = append(result, convertSnapshot(snap, project))
	}
	return result, nil
}

func (c *GCPComputeClient) ListInstanceGroups(ctx context.Context, project string) ([]InstanceGroupInfo, error) {
	var result []InstanceGroupInfo
	it := c.instanceGroups.AggregatedList(ctx, &computepb.AggregatedListInstanceGroupsRequest{Project: project})
	for {
		pair, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list instance groups: %w", err)
		}
		for _, ig := range pair.Value.GetInstanceGroups() {
			result = append(result, convertInstanceGroup(ig, project))
		}
	}
	return result, nil
}

func (c *GCPComputeClient) ListFirewalls(ctx context.Context, project string) ([]FirewallRule, error) {
	var result []FirewallRule
	it := c.firewalls.List(ctx, &computepb.ListFirewallsRequest{Project: project})
	for {
		fw, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list firewalls: %w", err)
		}
		result = append(result, convertFirewall(fw, project))
	}
	return result, nil
}

// GCPMonitoringClient implements MonitoringAPI using Cloud Monitoring.
type GCPMonitoringClient struct {
	client *monitoring.MetricClient
}

// NewMonitoringClient creates a MonitoringAPI backed by GCP Application Default Credentials.
func NewMonitoringClient(ctx context.Context) (*GCPMonitoringClient, error) {
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create monitoring client: %w", err)
	}
	return &GCPMonitoringClient{client: client}, nil
}

// Close releases the underlying client connection.
func (c *GCPMonitoringClient) Close() {
	_ = c.client.Close()
}

func (c *GCPMonitoringClient) FetchMetricMean(ctx context.Context, project, metricType, resourceLabel string, resourceIDs []string, lookbackDays int) (map[string]float64, error) {
	if len(resourceIDs) == 0 {
		return nil, nil
	}

	now := time.Now().UTC()
	startTime := now.Add(-time.Duration(lookbackDays) * 24 * time.Hour)

	filter := fmt.Sprintf(`metric.type = "%s"`, metricType)

	it := c.client.ListTimeSeries(ctx, &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + project,
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(startTime),
			EndTime:   timestamppb.New(now),
		},
		View: monitoringpb.ListTimeSeriesRequest_FULL,
	})

	wantIDs := make(map[string]bool, len(resourceIDs))
	for _, id := range resourceIDs {
		wantIDs[id] = true
	}

	results := make(map[string]float64)
	for {
		ts, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list time series: %w", err)
		}

		resID := ts.GetResource().GetLabels()[resourceLabel]
		if !wantIDs[resID] {
			continue
		}

		var sum float64
		count := 0
		for _, point := range ts.GetPoints() {
			sum += point.GetValue().GetDoubleValue()
			count++
		}
		if count > 0 {
			results[resID] = sum / float64(count)
		}
	}

	return results, nil
}

// GCPCloudSQLClient implements CloudSQLAPI using the Cloud SQL Admin API.
type GCPCloudSQLClient struct {
	service *sqladmin.Service
}

// NewCloudSQLClient creates a CloudSQLAPI backed by GCP Application Default Credentials.
func NewCloudSQLClient(ctx context.Context) (*GCPCloudSQLClient, error) {
	svc, err := sqladmin.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("create Cloud SQL client: %w", err)
	}
	return &GCPCloudSQLClient{service: svc}, nil
}

func (c *GCPCloudSQLClient) ListInstances(ctx context.Context, project string) ([]CloudSQLInstance, error) {
	resp, err := c.service.Instances.List(project).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("list Cloud SQL instances: %w", err)
	}

	var result []CloudSQLInstance
	for _, inst := range resp.Items {
		tier := ""
		if inst.Settings != nil {
			tier = inst.Settings.Tier
		}
		result = append(result, CloudSQLInstance{
			Name:            inst.Name,
			Project:         inst.Project,
			Region:          inst.Region,
			Tier:            tier,
			State:           inst.State,
			DatabaseVersion: inst.DatabaseVersion,
		})
	}
	return result, nil
}

// Conversion helpers

func convertInstance(inst *computepb.Instance, project string) ComputeInstance {
	ci := ComputeInstance{
		ID:          inst.GetId(),
		Name:        inst.GetName(),
		Zone:        lastPathSegment(inst.GetZone()),
		Project:     project,
		MachineType: lastPathSegment(inst.GetMachineType()),
		Status:      inst.GetStatus(),
		Labels:      inst.GetLabels(),
		LastStarted: parseTimestamp(inst.GetLastStartTimestamp()),
		CreateTime:  parseTimestamp(inst.GetCreationTimestamp()),
	}
	if inst.GetTags() != nil {
		ci.Tags = inst.GetTags().GetItems()
	}
	return ci
}

func convertDisk(disk *computepb.Disk, project string) PersistentDisk {
	return PersistentDisk{
		ID:         disk.GetId(),
		Name:       disk.GetName(),
		Zone:       lastPathSegment(disk.GetZone()),
		Project:    project,
		DiskType:   lastPathSegment(disk.GetType()),
		SizeGB:     disk.GetSizeGb(),
		Status:     disk.GetStatus(),
		Users:      disk.GetUsers(),
		Labels:     disk.GetLabels(),
		LastAttach: parseTimestamp(disk.GetLastAttachTimestamp()),
		CreateTime: parseTimestamp(disk.GetCreationTimestamp()),
	}
}

func convertAddress(addr *computepb.Address, project string) StaticAddress {
	return StaticAddress{
		ID:          addr.GetId(),
		Name:        addr.GetName(),
		Region:      lastPathSegment(addr.GetRegion()),
		Project:     project,
		Address:     addr.GetAddress(),
		Status:      addr.GetStatus(),
		Users:       addr.GetUsers(),
		AddressType: addr.GetAddressType(),
		CreateTime:  parseTimestamp(addr.GetCreationTimestamp()),
	}
}

func convertSnapshot(snap *computepb.Snapshot, project string) DiskSnapshot {
	return DiskSnapshot{
		ID:               snap.GetId(),
		Name:             snap.GetName(),
		Project:          project,
		SourceDisk:       snap.GetSourceDisk(),
		DiskSizeGB:       snap.GetDiskSizeGb(),
		StorageBytes:     snap.GetStorageBytes(),
		Status:           snap.GetStatus(),
		Labels:           snap.GetLabels(),
		CreateTime:       parseTimestamp(snap.GetCreationTimestamp()),
		StorageLocations: snap.GetStorageLocations(),
	}
}

func convertInstanceGroup(ig *computepb.InstanceGroup, project string) InstanceGroupInfo {
	return InstanceGroupInfo{
		ID:      ig.GetId(),
		Name:    ig.GetName(),
		Zone:    lastPathSegment(ig.GetZone()),
		Project: project,
		Size:    int(ig.GetSize()),
	}
}

func convertFirewall(fw *computepb.Firewall, project string) FirewallRule {
	return FirewallRule{
		ID:         fw.GetId(),
		Name:       fw.GetName(),
		Project:    project,
		Network:    lastPathSegment(fw.GetNetwork()),
		Direction:  fw.GetDirection(),
		Priority:   int64(fw.GetPriority()),
		TargetTags: fw.GetTargetTags(),
		Disabled:   fw.GetDisabled(),
	}
}

func lastPathSegment(url string) string {
	if url == "" {
		return ""
	}
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, s)
	}
	return t
}
