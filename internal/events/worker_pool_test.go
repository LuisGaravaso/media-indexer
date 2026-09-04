package events

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProcessor struct {
	mu           sync.Mutex
	processed    []MediaConfirmedEvent
	processFunc  func(ctx context.Context, event MediaConfirmedEvent) error
	callCount    int64
	activeWorker int64
	maxParallel  int64
}

func (m *mockProcessor) ProcessMediaConfirmed(ctx context.Context, event MediaConfirmedEvent) error {
	current := atomic.AddInt64(&m.activeWorker, 1)
	defer atomic.AddInt64(&m.activeWorker, -1)

	m.mu.Lock()
	if current > m.maxParallel {
		m.maxParallel = current
	}
	m.processed = append(m.processed, event)
	m.mu.Unlock()

	atomic.AddInt64(&m.callCount, 1)

	if m.processFunc != nil {
		return m.processFunc(ctx, event)
	}
	return nil
}

func TestWorkerPool_SuccessAndAck(t *testing.T) {
	proc := &mockProcessor{}
	pool, err := NewWorkerPool(3, 10, proc)
	require.NoError(t, err)
	pool.Start()

	var ackCount int64
	var nackCount int64

	event := MediaConfirmedEvent{
		Event:     EventMediaConfirmed,
		MediaID:   uuid.New(),
		UserID:    uuid.New(),
		MediaType: "video",
	}

	task := Task{
		Event: event,
		Ack: func() {
			atomic.AddInt64(&ackCount, 1)
		},
		Nack: func() {
			atomic.AddInt64(&nackCount, 1)
		},
	}

	err = pool.Submit(context.Background(), task)
	require.NoError(t, err)

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = pool.Stop(stopCtx)
	require.NoError(t, err)

	assert.Equal(t, int64(1), atomic.LoadInt64(&ackCount))
	assert.Equal(t, int64(0), atomic.LoadInt64(&nackCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&proc.callCount))
}

func TestWorkerPool_FailureAndNack(t *testing.T) {
	proc := &mockProcessor{
		processFunc: func(ctx context.Context, event MediaConfirmedEvent) error {
			return errors.New("gemini quota exceeded")
		},
	}
	pool, err := NewWorkerPool(2, 5, proc)
	require.NoError(t, err)
	pool.Start()

	var ackCount int64
	var nackCount int64

	event := MediaConfirmedEvent{
		Event:     EventMediaConfirmed,
		MediaID:   uuid.New(),
		UserID:    uuid.New(),
		MediaType: "image",
	}

	task := Task{
		Event: event,
		Ack: func() {
			atomic.AddInt64(&ackCount, 1)
		},
		Nack: func() {
			atomic.AddInt64(&nackCount, 1)
		},
	}

	err = pool.Submit(context.Background(), task)
	require.NoError(t, err)

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = pool.Stop(stopCtx)
	require.NoError(t, err)

	assert.Equal(t, int64(0), atomic.LoadInt64(&ackCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&nackCount))
}

func TestWorkerPool_ConcurrencyLimit(t *testing.T) {
	concurrency := 4
	totalTasks := 12

	proc := &mockProcessor{
		processFunc: func(ctx context.Context, event MediaConfirmedEvent) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		},
	}

	pool, err := NewWorkerPool(concurrency, 20, proc)
	require.NoError(t, err)
	pool.Start()

	var ackCount int64

	for i := 0; i < totalTasks; i++ {
		task := Task{
			Event: MediaConfirmedEvent{
				MediaID: uuid.New(),
			},
			Ack: func() {
				atomic.AddInt64(&ackCount, 1)
			},
		}
		err := pool.Submit(context.Background(), task)
		require.NoError(t, err)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = pool.Stop(stopCtx)
	require.NoError(t, err)

	assert.Equal(t, int64(totalTasks), atomic.LoadInt64(&ackCount))
	proc.mu.Lock()
	assert.LessOrEqual(t, proc.maxParallel, int64(concurrency))
	proc.mu.Unlock()
}
