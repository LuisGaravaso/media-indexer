package search

import (
	"context"
	"testing"
	"time"

	"github.com/LuisGaravaso/media-indexer/internal/repository"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSearchEmbedder struct{}

func (m *mockSearchEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	vec := make([]float32, 768)
	for i := range vec {
		vec[i] = 0.05
	}
	return vec, nil
}

func (m *mockSearchEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i := range texts {
		vec := make([]float32, 768)
		for j := range vec {
			vec[j] = float32(i+1) * 0.05
		}
		results[i] = vec
	}
	return results, nil
}

type mockSearchRepo struct {
	mediaItems []repository.SearchResultItem
	sceneItems []repository.SearchResultItem
}

func (m *mockSearchRepo) SaveMediaIndex(ctx context.Context, semantics *repository.MediaSemanticsRecord, scenes []repository.MediaSceneRecord) error {
	return nil
}

func (m *mockSearchRepo) GetSemanticsByMediaID(ctx context.Context, userID, mediaID uuid.UUID) (*repository.MediaSemanticsRecord, error) {
	return &repository.MediaSemanticsRecord{
		MediaID:          mediaID,
		UserID:           userID,
		Summary:          "A day at the sunny beach",
		Tags:             []string{"beach", "sun", "ocean"},
		DetectedLocation: "Miami",
		DetectedSeason:   "Summer",
		Embedding:        pgvector.NewVector(make([]float32, 768)),
		IndexedAt:        time.Now(),
	}, nil
}

func (m *mockSearchRepo) GetScenesByMediaID(ctx context.Context, userID, mediaID uuid.UUID) ([]repository.MediaSceneRecord, error) {
	return nil, nil
}

func (m *mockSearchRepo) SearchSimilarMedia(ctx context.Context, userID uuid.UUID, queryEmbedding pgvector.Vector, limit int, threshold float64) ([]repository.SearchResultItem, error) {
	return m.mediaItems, nil
}

func (m *mockSearchRepo) SearchSimilarScenes(ctx context.Context, userID uuid.UUID, queryEmbedding pgvector.Vector, limit int, threshold float64) ([]repository.SearchResultItem, error) {
	return m.sceneItems, nil
}

func TestSemanticSearchService_Search(t *testing.T) {
	userID := uuid.New()
	mediaID1 := uuid.New()
	mediaID2 := uuid.New()

	sceneIdx := 1
	start := 5.0
	end := 10.0
	desc := "Surfer catching barrel wave"

	mockRepo := &mockSearchRepo{
		mediaItems: []repository.SearchResultItem{
			{
				MediaID:          mediaID1,
				UserID:           userID,
				Summary:          "Surfer on huge waves",
				Tags:             []string{"surfing", "ocean"},
				DetectedLocation: "Hawaii",
				Similarity:       0.88,
			},
		},
		sceneItems: []repository.SearchResultItem{
			{
				MediaID:          mediaID2,
				UserID:           userID,
				SceneIndex:       &sceneIdx,
				StartTimeSeconds: &start,
				EndTimeSeconds:   &end,
				SceneDescription: &desc,
				Similarity:       0.94,
			},
		},
	}

	svc, err := NewSemanticSearchService(&mockSearchEmbedder{}, mockRepo)
	require.NoError(t, err)

	results, err := svc.Search(context.Background(), userID, SemanticSearchRequest{
		Query: "surfing action in Hawaii",
	})
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, mediaID2, results[0].MediaID) // Higher similarity ranked first
	assert.Equal(t, 0.94, results[0].Similarity)   // Raw cosine similarity rounded
	assert.Equal(t, 0.94, results[0].RawSimilarity)
	assert.Len(t, results[0].MatchingScenes, 1)
	assert.Equal(t, "Surfer catching barrel wave", results[0].MatchingScenes[0].Description)
}
