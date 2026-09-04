package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/pubsub" //nolint:staticcheck // standard GCP pubsub v1 package
)

// Subscriber defines the contract for consuming asynchronous events.
type Subscriber interface {
	Start(ctx context.Context) error
	Stop() error
}

// NoopSubscriber is a subscriber that performs no operations (e.g. for testing/disabled state).
type NoopSubscriber struct{}

// NewNoopSubscriber returns a no-op subscriber instance.
func NewNoopSubscriber() *NoopSubscriber {
	return &NoopSubscriber{}
}

// Start immediately returns nil.
func (n *NoopSubscriber) Start(ctx context.Context) error {
	return nil
}

// Stop immediately returns nil.
func (n *NoopSubscriber) Stop() error {
	return nil
}

// GCPPubSubSubscriber manages consuming messages from GCP Pub/Sub and queueing them into a WorkerPool.
type GCPPubSubSubscriber struct {
	client       *pubsub.Client
	subscription *pubsub.Subscription
	pool         *WorkerPool
	cancel       context.CancelFunc
}

// NewGCPPubSubSubscriber creates a new GCPPubSubSubscriber.
func NewGCPPubSubSubscriber(ctx context.Context, projectID, subscriptionName string, pool *WorkerPool) (*GCPPubSubSubscriber, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}
	if subscriptionName == "" {
		return nil, fmt.Errorf("subscription name cannot be empty")
	}
	if pool == nil {
		return nil, fmt.Errorf("worker pool cannot be nil")
	}

	// Support full resource names: projects/<proj>/subscriptions/<sub> or just <sub>
	if strings.Contains(subscriptionName, "/") {
		parts := strings.Split(subscriptionName, "/")
		subscriptionName = parts[len(parts)-1]
	}

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	sub := client.Subscription(subscriptionName)
	// Configure default receive settings
	sub.ReceiveSettings.MaxOutstandingMessages = pool.concurrency * 2
	sub.ReceiveSettings.MaxExtension = 10 * time.Minute

	return &GCPPubSubSubscriber{
		client:       client,
		subscription: sub,
		pool:         pool,
	}, nil
}

// Start begins listening to the Pub/Sub subscription and delivering messages to the worker pool.
func (s *GCPPubSubSubscriber) Start(ctx context.Context) error {
	subCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	log.Printf("[PubSubSubscriber] Listening on subscription: %s", s.subscription.ID())

	err := s.subscription.Receive(subCtx, func(msgCtx context.Context, msg *pubsub.Message) {
		var event MediaConfirmedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("[PubSubSubscriber] Malformed message %s received: %v. Acknowledging to avoid poison pill.", msg.ID, err)
			msg.Ack()
			return
		}

		if event.Event != EventMediaConfirmed && event.Event != "" {
			log.Printf("[PubSubSubscriber] Ignoring unrecognized event type '%s' (message %s)", event.Event, msg.ID)
			msg.Ack()
			return
		}

		task := Task{
			Event: event,
			Ack: func() {
				msg.Ack()
			},
			Nack: func() {
				msg.Nack()
			},
		}

		if err := s.pool.Submit(msgCtx, task); err != nil {
			log.Printf("[PubSubSubscriber] Could not submit task for message %s: %v. Nacking message.", msg.ID, err)
			msg.Nack()
		}
	})

	if err != nil && err != context.Canceled {
		return fmt.Errorf("pubsub subscription receive error: %w", err)
	}

	return nil
}

// Stop cancels the receive context and closes the Pub/Sub client.
func (s *GCPPubSubSubscriber) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
