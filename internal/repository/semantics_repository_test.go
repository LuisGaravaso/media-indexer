package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
)

type MockSemanticsRepository struct {
	Semantics map[uuid.UUID]*MediaSemanticsRecord
	Scenes    map[uuid.UUID][]MediaSceneRecord
}

func NewMockSemanticsRepository() *MockSemanticsRepository {
	return &MockSemanticsRepository{
		Semantics: make(map[uuid.UUID]*MediaSemanticsRecord),
		Scenes:    make(map[uuid.UUID][]MediaSceneRecord),
	}
}

func (m *MockSemanticsRepository) SaveMediaIndex(ctx context.Context, semantics *MediaSemanticsRecord, scenes []MediaSceneRecord) error {
	m.Semantics[semantics.MediaID] = semantics
	m.Scenes[semantics.MediaID] = scenes
	return nil
}

func (m *MockSemanticsRepository) GetSemanticsByMediaID(ctx context.Context, userID, mediaID uuid.UUID) (*MediaSemanticsRecord, error) {
	rec, ok := m.Semantics[mediaID]
	if !ok || rec.UserID != userID {
		return nil, nil
	}
	return rec, nil
}

func (m *MockSemanticsRepository) GetScenesByMediaID(ctx context.Context, userID, mediaID uuid.UUID) ([]MediaSceneRecord, error) {
	scenes, ok := m.Scenes[mediaID]
	if !ok {
		return nil, nil
	}
	return scenes, nil
}

func (m *MockSemanticsRepository) SearchSimilarMedia(ctx context.Context, userID uuid.UUID, queryEmbedding pgvector.Vector, limit int, threshold float64) ([]SearchResultItem, error) {
	var results []SearchResultItem
	for _, rec := range m.Semantics {
		if rec.UserID == userID {
			results = append(results, SearchResultItem{
				MediaID:          rec.MediaID,
				UserID:           rec.UserID,
				Summary:          rec.Summary,
				Tags:             rec.Tags,
				DetectedLocation: rec.DetectedLocation,
				DetectedSeason:   rec.DetectedSeason,
				Similarity:       0.95,
			})
		}
	}
	return results, nil
}

func (m *MockSemanticsRepository) SearchSimilarScenes(ctx context.Context, userID uuid.UUID, queryEmbedding pgvector.Vector, limit int, threshold float64) ([]SearchResultItem, error) {
	var results []SearchResultItem
	for mediaID, scenesList := range m.Scenes {
		for _, s := range scenesList {
			if s.UserID == userID {
				idx := s.SceneIndex
				start := s.StartTimeSeconds
				end := s.EndTimeSeconds
				desc := s.Description

				results = append(results, SearchResultItem{
					MediaID:          mediaID,
					UserID:           s.UserID,
					SceneIndex:       &idx,
					StartTimeSeconds: &start,
					EndTimeSeconds:   &end,
					SceneDescription: &desc,
					Similarity:       0.92,
				})
			}
		}
	}
	return results, nil
}

func TestMockRepository_SaveAndGet(t *testing.T) {
	repo := NewMockSemanticsRepository()
	mediaID := uuid.New()
	userID := uuid.New()

	vec := pgvector.NewVector(make([]float32, 768))

	semantics := &MediaSemanticsRecord{
		MediaID:          mediaID,
		UserID:           userID,
		Summary:          "Surfer catching a big wave",
		Tags:             []string{"surf", "ocean", "wave"},
		DetectedLocation: "Hawaii",
		DetectedSeason:   "Winter",
		Embedding:        vec,
		IndexedAt:        time.Now(),
	}

	scenes := []MediaSceneRecord{
		{
			ID:               uuid.New(),
			MediaID:          mediaID,
			UserID:           userID,
			SceneIndex:       0,
			StartTimeSeconds: 0.0,
			EndTimeSeconds:   4.5,
			Description:      "Surfer paddling into the wave",
			Embedding:        vec,
		},
	}

	err := repo.SaveMediaIndex(context.Background(), semantics, scenes)
	assert.NoError(t, err)

	fetchedSem, err := repo.GetSemanticsByMediaID(context.Background(), userID, mediaID)
	assert.NoError(t, err)
	assert.NotNil(t, fetchedSem)
	assert.Equal(t, "Surfer catching a big wave", fetchedSem.Summary)

	fetchedScenes, err := repo.GetScenesByMediaID(context.Background(), userID, mediaID)
	assert.NoError(t, err)
	assert.Len(t, fetchedScenes, 1)
	assert.Equal(t, "Surfer paddling into the wave", fetchedScenes[0].Description)
}
