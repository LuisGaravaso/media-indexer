package events

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EventType defines structured event action names.
type EventType string

const (
	// EventMediaConfirmed is published when an upload is confirmed and ready for indexing.
	EventMediaConfirmed EventType = "media.confirmed"
)

// MediaConfirmedEvent payload published when an upload is confirmed and ready.
type MediaConfirmedEvent struct {
	Event         EventType `json:"event"`
	MediaID       uuid.UUID `json:"media_id"`
	UserID        uuid.UUID `json:"user_id"`
	StoragePath   string    `json:"storage_path"`
	FileName      string    `json:"file_name"`
	MimeType      string    `json:"mime_type"`
	MediaType     string    `json:"media_type"`
	FileSizeBytes int64     `json:"file_size_bytes"`
	Timestamp     time.Time `json:"timestamp"`
}

// EventProcessor defines the business logic contract to process an incoming MediaConfirmedEvent.
type EventProcessor interface {
	ProcessMediaConfirmed(ctx context.Context, event MediaConfirmedEvent) error
}
