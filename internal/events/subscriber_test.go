package events

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoopSubscriber(t *testing.T) {
	sub := NewNoopSubscriber()
	require.NotNil(t, sub)

	err := sub.Start(context.Background())
	assert.NoError(t, err)

	err = sub.Stop()
	assert.NoError(t, err)
}

func TestNewGCPPubSubSubscriber_Validation(t *testing.T) {
	pool, err := NewWorkerPool(1, 1, &mockProcessor{})
	require.NoError(t, err)

	_, err = NewGCPPubSubSubscriber(context.Background(), "", "my-sub", pool)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project ID cannot be empty")

	_, err = NewGCPPubSubSubscriber(context.Background(), "my-proj", "", pool)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subscription name cannot be empty")

	_, err = NewGCPPubSubSubscriber(context.Background(), "my-proj", "my-sub", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worker pool cannot be nil")
}
