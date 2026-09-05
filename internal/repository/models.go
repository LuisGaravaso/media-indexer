package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// MediaSemanticsRecord represents the media_semantics database row.
type MediaSemanticsRecord struct {
	MediaID          uuid.UUID       `json:"media_id"`
	UserID           uuid.UUID       `json:"user_id"`
	MediaType        string          `json:"media_type"`
	Summary          string          `json:"summary"`
	Tags             []string        `json:"tags"`
	DetectedLocation string          `json:"detected_location"`
	DetectedSeason   string          `json:"detected_season"`
	VisualObjects    []string        `json:"visual_objects"`
	Embedding        pgvector.Vector `json:"embedding"`
	IndexedAt        time.Time       `json:"indexed_at"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// MediaSceneRecord represents a single scene cut in media_scenes.
type MediaSceneRecord struct {
	ID               uuid.UUID       `json:"id"`
	MediaID          uuid.UUID       `json:"media_id"`
	UserID           uuid.UUID       `json:"user_id"`
	SceneIndex       int             `json:"scene_index"`
	StartTimeSeconds float64         `json:"start_time_seconds"`
	EndTimeSeconds   float64         `json:"end_time_seconds"`
	Description      string          `json:"description"`
	SpeechTranscript string          `json:"speech_transcript"`
	Mood             string          `json:"mood"`
	Actions          []string        `json:"actions"`
	Embedding        pgvector.Vector `json:"embedding"`
	CreatedAt        time.Time       `json:"created_at"`
}

// SearchResultItem represents semantic search matches returned across media items or scenes.
type SearchResultItem struct {
	MediaID          uuid.UUID `json:"media_id"`
	UserID           uuid.UUID `json:"user_id"`
	MediaType        string    `json:"media_type"`
	Summary          string    `json:"summary"`
	Tags             []string  `json:"tags"`
	DetectedLocation string    `json:"detected_location,omitempty"`
	DetectedSeason   string    `json:"detected_season,omitempty"`
	Similarity       float64   `json:"similarity"`
	SceneIndex       *int      `json:"scene_index,omitempty"`
	StartTimeSeconds *float64  `json:"start_time_seconds,omitempty"`
	EndTimeSeconds   *float64  `json:"end_time_seconds,omitempty"`
	SceneDescription *string   `json:"scene_description,omitempty"`
}
