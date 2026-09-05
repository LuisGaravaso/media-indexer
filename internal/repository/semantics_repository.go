package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// SemanticsRepository defines data access methods for media semantics and scene cuts.
type SemanticsRepository interface {
	SaveMediaIndex(ctx context.Context, semantics *MediaSemanticsRecord, scenes []MediaSceneRecord) error
	GetSemanticsByMediaID(ctx context.Context, userID, mediaID uuid.UUID) (*MediaSemanticsRecord, error)
	GetScenesByMediaID(ctx context.Context, userID, mediaID uuid.UUID) ([]MediaSceneRecord, error)
	SearchSimilarMedia(ctx context.Context, userID uuid.UUID, queryEmbedding pgvector.Vector, limit int, threshold float64) ([]SearchResultItem, error)
	SearchSimilarScenes(ctx context.Context, userID uuid.UUID, queryEmbedding pgvector.Vector, limit int, threshold float64) ([]SearchResultItem, error)
}

// PostgresSemanticsRepository implements SemanticsRepository using pgxpool.
type PostgresSemanticsRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresSemanticsRepository creates an instance of PostgresSemanticsRepository.
func NewPostgresSemanticsRepository(pool *pgxpool.Pool) *PostgresSemanticsRepository {
	return &PostgresSemanticsRepository{
		pool: pool,
	}
}

// SaveMediaIndex saves media_semantics and media_scenes in a single atomic transaction.
func (r *PostgresSemanticsRepository) SaveMediaIndex(ctx context.Context, semantics *MediaSemanticsRecord, scenes []MediaSceneRecord) error {
	if semantics == nil {
		return fmt.Errorf("semantics record cannot be nil")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	upsertSemanticsSQL := `
		INSERT INTO media_semantics (
			media_id, user_id, media_type, summary, tags, detected_location, detected_season,
			visual_objects, embedding, indexed_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		ON CONFLICT (media_id) DO UPDATE SET
			media_type = EXCLUDED.media_type,
			summary = EXCLUDED.summary,
			tags = EXCLUDED.tags,
			detected_location = EXCLUDED.detected_location,
			detected_season = EXCLUDED.detected_season,
			visual_objects = EXCLUDED.visual_objects,
			embedding = EXCLUDED.embedding,
			indexed_at = NOW(),
			updated_at = NOW()
	`

	_, err = tx.Exec(ctx, upsertSemanticsSQL,
		semantics.MediaID,
		semantics.UserID,
		semantics.MediaType,
		semantics.Summary,
		semantics.Tags,
		semantics.DetectedLocation,
		semantics.DetectedSeason,
		semantics.VisualObjects,
		semantics.Embedding,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert media_semantics: %w", err)
	}

	// Clean existing scenes before inserting new scene breakdown
	deleteScenesSQL := `DELETE FROM media_scenes WHERE media_id = $1`
	_, err = tx.Exec(ctx, deleteScenesSQL, semantics.MediaID)
	if err != nil {
		return fmt.Errorf("failed to clear existing media_scenes: %w", err)
	}

	if len(scenes) > 0 {
		insertSceneSQL := `
			INSERT INTO media_scenes (
				media_id, user_id, scene_index, start_time_seconds, end_time_seconds,
				description, speech_transcript, mood, actions, embedding
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`

		batch := &pgx.Batch{}
		for _, s := range scenes {
			batch.Queue(insertSceneSQL,
				semantics.MediaID,
				semantics.UserID,
				s.SceneIndex,
				s.StartTimeSeconds,
				s.EndTimeSeconds,
				s.Description,
				s.SpeechTranscript,
				s.Mood,
				s.Actions,
				s.Embedding,
			)
		}

		br := tx.SendBatch(ctx, batch)
		for i := 0; i < len(scenes); i++ {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return fmt.Errorf("failed to insert scene %d: %w", i, err)
			}
		}
		if err := br.Close(); err != nil {
			return fmt.Errorf("failed to close scenes batch: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetSemanticsByMediaID fetches the semantic summary record for a specific media.
func (r *PostgresSemanticsRepository) GetSemanticsByMediaID(ctx context.Context, userID, mediaID uuid.UUID) (*MediaSemanticsRecord, error) {
	query := `
		SELECT media_id, user_id, media_type, summary, tags, detected_location, detected_season,
		       visual_objects, embedding, indexed_at, created_at, updated_at
		FROM media_semantics
		WHERE user_id = $1 AND media_id = $2
	`

	var rec MediaSemanticsRecord
	err := r.pool.QueryRow(ctx, query, userID, mediaID).Scan(
		&rec.MediaID,
		&rec.UserID,
		&rec.MediaType,
		&rec.Summary,
		&rec.Tags,
		&rec.DetectedLocation,
		&rec.DetectedSeason,
		&rec.VisualObjects,
		&rec.Embedding,
		&rec.IndexedAt,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query media_semantics: %w", err)
	}

	return &rec, nil
}

// GetScenesByMediaID fetches all scene cuts for a video media item in timeline order.
func (r *PostgresSemanticsRepository) GetScenesByMediaID(ctx context.Context, userID, mediaID uuid.UUID) ([]MediaSceneRecord, error) {
	query := `
		SELECT id, media_id, user_id, scene_index, start_time_seconds, end_time_seconds,
		       description, speech_transcript, mood, actions, embedding, created_at
		FROM media_scenes
		WHERE user_id = $1 AND media_id = $2
		ORDER BY scene_index ASC
	`

	rows, err := r.pool.Query(ctx, query, userID, mediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to query media_scenes: %w", err)
	}
	defer rows.Close()

	var scenes []MediaSceneRecord
	for rows.Next() {
		var s MediaSceneRecord
		err := rows.Scan(
			&s.ID,
			&s.MediaID,
			&s.UserID,
			&s.SceneIndex,
			&s.StartTimeSeconds,
			&s.EndTimeSeconds,
			&s.Description,
			&s.SpeechTranscript,
			&s.Mood,
			&s.Actions,
			&s.Embedding,
			&s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scene row: %w", err)
		}
		scenes = append(scenes, s)
	}

	return scenes, nil
}

// SearchSimilarMedia performs cosine similarity search on media_semantics embeddings.
func (r *PostgresSemanticsRepository) SearchSimilarMedia(ctx context.Context, userID uuid.UUID, queryEmbedding pgvector.Vector, limit int, threshold float64) ([]SearchResultItem, error) {
	if limit <= 0 {
		limit = 20
	}

	query := `
		SELECT media_id, user_id, media_type, summary, tags, detected_location, detected_season,
		       1 - (embedding <=> $2) AS similarity
		FROM media_semantics
		WHERE user_id = $1 AND (1 - (embedding <=> $2)) >= $3
		ORDER BY embedding <=> $2 ASC
		LIMIT $4
	`

	rows, err := r.pool.Query(ctx, query, userID, queryEmbedding, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar media: %w", err)
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var item SearchResultItem
		err := rows.Scan(
			&item.MediaID,
			&item.UserID,
			&item.MediaType,
			&item.Summary,
			&item.Tags,
			&item.DetectedLocation,
			&item.DetectedSeason,
			&item.Similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan media search row: %w", err)
		}
		results = append(results, item)
	}

	return results, nil
}

// SearchSimilarScenes performs cosine similarity search on granular video scene cut embeddings.
func (r *PostgresSemanticsRepository) SearchSimilarScenes(ctx context.Context, userID uuid.UUID, queryEmbedding pgvector.Vector, limit int, threshold float64) ([]SearchResultItem, error) {
	if limit <= 0 {
		limit = 20
	}

	query := `
		SELECT media_id, user_id, scene_index, start_time_seconds, end_time_seconds,
		       description, 1 - (embedding <=> $2) AS similarity
		FROM media_scenes
		WHERE user_id = $1 AND (1 - (embedding <=> $2)) >= $3
		ORDER BY embedding <=> $2 ASC
		LIMIT $4
	`

	rows, err := r.pool.Query(ctx, query, userID, queryEmbedding, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar scenes: %w", err)
	}
	defer rows.Close()

	var results []SearchResultItem
	for rows.Next() {
		var item SearchResultItem
		var sceneIdx int
		var start, end float64
		var desc string

		err := rows.Scan(
			&item.MediaID,
			&item.UserID,
			&sceneIdx,
			&start,
			&end,
			&desc,
			&item.Similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scene search row: %w", err)
		}

		item.SceneIndex = &sceneIdx
		item.StartTimeSeconds = &start
		item.EndTimeSeconds = &end
		item.SceneDescription = &desc

		results = append(results, item)
	}

	return results, nil
}
