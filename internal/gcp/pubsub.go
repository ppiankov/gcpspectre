package gcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ppiankov/gcpspectre/internal/pricing"
)

// PubSubScanner detects idle Pub/Sub topics and subscriptions.
type PubSubScanner struct {
	pubsub     PubSubAPI
	monitoring MonitoringAPI
	project    string
}

// NewPubSubScanner creates a scanner for Pub/Sub resources.
func NewPubSubScanner(pubsub PubSubAPI, monitoring MonitoringAPI, project string) *PubSubScanner {
	return &PubSubScanner{pubsub: pubsub, monitoring: monitoring, project: project}
}

// Type returns the resource type.
func (s *PubSubScanner) Type() ResourceType {
	return ResourcePubSub
}

// Scan examines Pub/Sub topics and subscriptions for idle resources and growing backlogs.
func (s *PubSubScanner) Scan(ctx context.Context, cfg ScanConfig) (*ScanResult, error) {
	if s.pubsub == nil {
		return &ScanResult{}, nil
	}

	topics, err := s.pubsub.ListTopics(ctx, s.project)
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}

	subscriptions, err := s.pubsub.ListSubscriptions(ctx, s.project)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}

	result := &ScanResult{ResourcesScanned: len(topics) + len(subscriptions)}

	s.scanTopics(ctx, cfg, topics, result)
	s.scanSubscriptions(ctx, cfg, subscriptions, result)

	return result, nil
}

func (s *PubSubScanner) scanTopics(ctx context.Context, cfg ScanConfig, topics []PubSubTopic, result *ScanResult) {
	// Check for topics with no subscriptions
	for _, t := range topics {
		if cfg.Exclude.ResourceIDs != nil && cfg.Exclude.ResourceIDs[t.Name] {
			continue
		}
		if shouldExcludeLabels(t.Labels, cfg.Exclude.Labels) {
			continue
		}
		if t.SubscriptionCount == 0 {
			result.Findings = append(result.Findings, Finding{
				ID:           FindingPubSubTopicNoSubs,
				Severity:     SeverityMedium,
				ResourceType: ResourcePubSub,
				ResourceID:   t.Name,
				ResourceName: t.Name,
				Project:      s.project,
				Message:      fmt.Sprintf("Topic %s has no subscriptions", t.Name),
				Metadata: map[string]any{
					"resource_kind": "topic",
				},
			})
			continue
		}
	}

	// Check for idle topics via monitoring
	if s.monitoring == nil {
		return
	}

	var topicNames []string
	topicMap := make(map[string]PubSubTopic)
	for _, t := range topics {
		if cfg.Exclude.ResourceIDs != nil && cfg.Exclude.ResourceIDs[t.Name] {
			continue
		}
		if shouldExcludeLabels(t.Labels, cfg.Exclude.Labels) {
			continue
		}
		if t.SubscriptionCount > 0 {
			topicNames = append(topicNames, t.Name)
			topicMap[t.Name] = t
		}
	}

	if len(topicNames) == 0 {
		return
	}

	sendMap, err := s.monitoring.FetchMetricMean(ctx, s.project,
		"pubsub.googleapis.com/topic/send_message_operation_count",
		"topic_id", topicNames, cfg.IdleDays)
	if err != nil {
		slog.Warn("Failed to fetch Pub/Sub topic metrics", "project", s.project, "error", err)
		return
	}

	for _, name := range topicNames {
		count, ok := sendMap[name]
		if !ok || count == 0 {
			result.Findings = append(result.Findings, Finding{
				ID:           FindingPubSubTopicIdle,
				Severity:     SeverityMedium,
				ResourceType: ResourcePubSub,
				ResourceID:   name,
				ResourceName: name,
				Project:      s.project,
				Message:      fmt.Sprintf("Topic %s has 0 messages published over %d days", name, cfg.IdleDays),
				Metadata: map[string]any{
					"resource_kind":      "topic",
					"subscription_count": topicMap[name].SubscriptionCount,
				},
			})
		}
	}
}

func (s *PubSubScanner) scanSubscriptions(ctx context.Context, cfg ScanConfig, subs []PubSubSubscription, result *ScanResult) {
	if s.monitoring == nil || len(subs) == 0 {
		return
	}

	var subNames []string
	subMap := make(map[string]PubSubSubscription)
	for _, sub := range subs {
		if cfg.Exclude.ResourceIDs != nil && cfg.Exclude.ResourceIDs[sub.Name] {
			continue
		}
		if shouldExcludeLabels(sub.Labels, cfg.Exclude.Labels) {
			continue
		}
		subNames = append(subNames, sub.Name)
		subMap[sub.Name] = sub
	}

	if len(subNames) == 0 {
		return
	}

	// Check for idle subscriptions (no pull or push activity)
	pullMap, err := s.monitoring.FetchMetricMean(ctx, s.project,
		"pubsub.googleapis.com/subscription/pull_message_operation_count",
		"subscription_id", subNames, cfg.IdleDays)
	if err != nil {
		slog.Warn("Failed to fetch Pub/Sub pull metrics", "project", s.project, "error", err)
		return
	}

	pushMap, err := s.monitoring.FetchMetricMean(ctx, s.project,
		"pubsub.googleapis.com/subscription/push_request_count",
		"subscription_id", subNames, cfg.IdleDays)
	if err != nil {
		slog.Warn("Failed to fetch Pub/Sub push metrics", "project", s.project, "error", err)
		return
	}

	// Check for backlog (undelivered messages)
	backlogMap, err := s.monitoring.FetchMetricMean(ctx, s.project,
		"pubsub.googleapis.com/subscription/num_undelivered_messages",
		"subscription_id", subNames, cfg.IdleDays)
	if err != nil {
		slog.Warn("Failed to fetch Pub/Sub backlog metrics", "project", s.project, "error", err)
	}

	for _, name := range subNames {
		pullCount := pullMap[name]
		pushCount := pushMap[name]
		backlog := float64(0)
		if backlogMap != nil {
			backlog = backlogMap[name]
		}

		if pullCount == 0 && pushCount == 0 {
			cost := pricing.MonthlyPubSubSubscriptionCost()
			result.Findings = append(result.Findings, Finding{
				ID:                    FindingPubSubSubscriptionIdle,
				Severity:              SeverityMedium,
				ResourceType:          ResourcePubSub,
				ResourceID:            name,
				ResourceName:          name,
				Project:               s.project,
				Message:               fmt.Sprintf("Subscription %s has 0 pull/push activity over %d days", name, cfg.IdleDays),
				EstimatedMonthlyWaste: cost,
				Metadata: map[string]any{
					"resource_kind": "subscription",
					"topic":         subMap[name].Topic,
					"backlog":       backlog,
				},
			})
			continue
		}

		// Active subscription but growing backlog — dead consumer
		if backlog > 10000 {
			result.Findings = append(result.Findings, Finding{
				ID:           FindingPubSubSubscriptionBacklog,
				Severity:     SeverityHigh,
				ResourceType: ResourcePubSub,
				ResourceID:   name,
				ResourceName: name,
				Project:      s.project,
				Message:      fmt.Sprintf("Subscription %s has %.0f undelivered messages (consumer may be dead)", name, backlog),
				Metadata: map[string]any{
					"resource_kind":        "subscription",
					"topic":                subMap[name].Topic,
					"undelivered_messages": backlog,
				},
			})
		}
	}
}
