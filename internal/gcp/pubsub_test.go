package gcp

import (
	"context"
	"testing"
)

func TestPubSubScanner_TopicIdle(t *testing.T) {
	pubsub := &mockPubSubAPI{
		topics: []PubSubTopic{
			{Name: "idle-topic", SubscriptionCount: 1},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{}, // no messages
	}

	s := NewPubSubScanner(pubsub, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 1 {
		t.Errorf("ResourcesScanned = %d, want 1", result.ResourcesScanned)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("Findings = %d, want 1", len(result.Findings))
	}
	if result.Findings[0].ID != FindingPubSubTopicIdle {
		t.Errorf("FindingID = %s, want %s", result.Findings[0].ID, FindingPubSubTopicIdle)
	}
}

func TestPubSubScanner_TopicNoSubscriptions(t *testing.T) {
	pubsub := &mockPubSubAPI{
		topics: []PubSubTopic{
			{Name: "orphan-topic", SubscriptionCount: 0},
		},
	}
	monitoring := &mockMonitoringAPI{}

	s := NewPubSubScanner(pubsub, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("Findings = %d, want 1", len(result.Findings))
	}
	if result.Findings[0].ID != FindingPubSubTopicNoSubs {
		t.Errorf("FindingID = %s, want %s", result.Findings[0].ID, FindingPubSubTopicNoSubs)
	}
}

func TestPubSubScanner_SubscriptionIdle(t *testing.T) {
	pubsub := &mockPubSubAPI{
		subscriptions: []PubSubSubscription{
			{Name: "idle-sub", Topic: "some-topic"},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{}, // no activity
	}

	s := NewPubSubScanner(pubsub, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	hasIdle := false
	for _, f := range result.Findings {
		if f.ID == FindingPubSubSubscriptionIdle {
			hasIdle = true
		}
	}
	if !hasIdle {
		t.Error("expected PUBSUB_SUBSCRIPTION_IDLE finding")
	}
}

func TestPubSubScanner_SubscriptionBacklog(t *testing.T) {
	pubsub := &mockPubSubAPI{
		subscriptions: []PubSubSubscription{
			{Name: "backlog-sub", Topic: "active-topic"},
		},
	}
	// Simulate: subscription has pull activity but massive backlog
	monitoring := &mockMonitoringAPIMulti{
		results: map[string]map[string]float64{
			"pubsub.googleapis.com/subscription/pull_message_operation_count": {"backlog-sub": 100},
			"pubsub.googleapis.com/subscription/push_request_count":           {"backlog-sub": 0},
			"pubsub.googleapis.com/subscription/num_undelivered_messages":     {"backlog-sub": 50000},
		},
	}

	s := NewPubSubScanner(pubsub, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	hasBacklog := false
	for _, f := range result.Findings {
		if f.ID == FindingPubSubSubscriptionBacklog {
			hasBacklog = true
		}
	}
	if !hasBacklog {
		t.Error("expected PUBSUB_SUBSCRIPTION_BACKLOG finding")
	}
}

func TestPubSubScanner_ActiveTopic(t *testing.T) {
	pubsub := &mockPubSubAPI{
		topics: []PubSubTopic{
			{Name: "active-topic", SubscriptionCount: 2},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{
			"active-topic": 5000, // active
		},
	}

	s := NewPubSubScanner(pubsub, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings for active topic, got %d", len(result.Findings))
	}
}

func TestPubSubScanner_ExcludeByLabel(t *testing.T) {
	pubsub := &mockPubSubAPI{
		topics: []PubSubTopic{
			{Name: "labeled-topic", SubscriptionCount: 0, Labels: map[string]string{"env": "dev"}},
		},
	}
	monitoring := &mockMonitoringAPI{}

	s := NewPubSubScanner(pubsub, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{
		IdleDays: 7,
		Exclude:  ExcludeConfig{Labels: map[string]string{"env": "dev"}},
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings after label exclude, got %d", len(result.Findings))
	}
}

func TestPubSubScanner_ExcludeByID(t *testing.T) {
	pubsub := &mockPubSubAPI{
		topics: []PubSubTopic{
			{Name: "excluded-topic", SubscriptionCount: 0},
		},
	}
	monitoring := &mockMonitoringAPI{}

	s := NewPubSubScanner(pubsub, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{
		IdleDays: 7,
		Exclude:  ExcludeConfig{ResourceIDs: map[string]bool{"excluded-topic": true}},
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings after ID exclude, got %d", len(result.Findings))
	}
}

func TestPubSubScanner_NilClient(t *testing.T) {
	s := NewPubSubScanner(nil, nil, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 0 {
		t.Errorf("ResourcesScanned = %d, want 0", result.ResourcesScanned)
	}
}

func TestPubSubScanner_Empty(t *testing.T) {
	pubsub := &mockPubSubAPI{}
	monitoring := &mockMonitoringAPI{}

	s := NewPubSubScanner(pubsub, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 0 {
		t.Errorf("ResourcesScanned = %d, want 0", result.ResourcesScanned)
	}
}
