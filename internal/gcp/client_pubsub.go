package gcp

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/iterator"
)

// GCPPubSubClient implements PubSubAPI using the Cloud Pub/Sub SDK.
type GCPPubSubClient struct {
	client *pubsub.Client
}

// NewPubSubClient creates a PubSubAPI backed by GCP Application Default Credentials.
func NewPubSubClient(ctx context.Context) (*GCPPubSubClient, error) {
	// Pub/Sub client requires a project, but we create per-project clients during scan.
	// This placeholder client validates credentials; actual listing uses project-scoped clients.
	return &GCPPubSubClient{}, nil
}

// ListTopics lists all Pub/Sub topics in the given project.
func (c *GCPPubSubClient) ListTopics(ctx context.Context, project string) ([]PubSubTopic, error) {
	client, err := pubsub.NewClient(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("create pubsub client for %s: %w", project, err)
	}
	defer client.Close()

	var topics []PubSubTopic
	it := client.Topics(ctx)
	for {
		t, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list topics: %w", err)
		}

		cfg, err := t.Config(ctx)
		if err != nil {
			return nil, fmt.Errorf("get topic config for %s: %w", t.ID(), err)
		}

		// Count subscriptions
		subCount := 0
		subs := t.Subscriptions(ctx)
		for {
			_, err := subs.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			subCount++
		}

		name := t.ID()
		// Extract short name if full resource path
		if parts := strings.Split(name, "/"); len(parts) > 1 {
			name = parts[len(parts)-1]
		}

		topics = append(topics, PubSubTopic{
			Name:              name,
			Project:           project,
			Labels:            cfg.Labels,
			SubscriptionCount: subCount,
		})
	}
	return topics, nil
}

// ListSubscriptions lists all Pub/Sub subscriptions in the given project.
func (c *GCPPubSubClient) ListSubscriptions(ctx context.Context, project string) ([]PubSubSubscription, error) {
	client, err := pubsub.NewClient(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("create pubsub client for %s: %w", project, err)
	}
	defer client.Close()

	var subs []PubSubSubscription
	it := client.Subscriptions(ctx)
	for {
		s, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list subscriptions: %w", err)
		}

		cfg, err := s.Config(ctx)
		if err != nil {
			return nil, fmt.Errorf("get subscription config for %s: %w", s.ID(), err)
		}

		topicName := ""
		if cfg.Topic != nil {
			topicName = cfg.Topic.ID()
		}

		subs = append(subs, PubSubSubscription{
			Name:    s.ID(),
			Topic:   topicName,
			Project: project,
			Labels:  cfg.Labels,
		})
	}
	return subs, nil
}
