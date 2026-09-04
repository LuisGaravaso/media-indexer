package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/LuisGaravaso/media-indexer/internal/analyzer"
	"github.com/LuisGaravaso/media-indexer/internal/events"
	"github.com/LuisGaravaso/media-indexer/internal/repository"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAnalyzer struct {
	analyzeFunc func(ctx context.Context, gcsURI, mimeType, mediaType string) (*analyzer.MediaAnalysisResult, error)
}

func (m *mockAnalyzer) AnalyzeGCSMedia(ctx context.Context, gcsURI, mimeType, mediaType string) (*analyzer.MediaAnalysisResult, error) {
	if m.analyzeFunc != nil {
		return m.analyzeFunc(ctx, gcsURI, mimeType, mediaType)
	}
	return nil, nil
}

type mockEmbedder struct{}

func (m *mockEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	vec := make([]float32, 768)
	for i := range vec {
		vec[i] = 0.01
	}
	return vec, nil
}

func (m *mockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i := range texts {
		vec := make([]float32, 768)
		for j := range vec {
			vec[j] = float32(i+1) * 0.01
		}
		results[i] = vec
	}
	return results, nil
}

type mockSemanticsRepository struct {
	Semantics map[uuid.UUID]*repository.MediaSemanticsRecord
	Scenes    map[uuid.UUID][]repository.MediaSceneRecord
}

func newMockSemanticsRepository() *mockSemanticsRepository {
	return &mockSemanticsRepository{
		Semantics: make(map[uuid.UUID]*repository.MediaSemanticsRecord),
		Scenes:    make(map[uuid.UUID][]repository.MediaSceneRecord),
	}
}

func (m *mockSemanticsRepository) SaveMediaIndex(ctx context.Context, semantics *repository.MediaSemanticsRecord, scenes []repository.MediaSceneRecord) error {
	m.Semantics[semantics.MediaID] = semantics
	m.Scenes[semantics.MediaID] = scenes
	return nil
}

func (m *mockSemanticsRepository) GetSemanticsByMediaID(ctx context.Context, userID, mediaID uuid.UUID) (*repository.MediaSemanticsRecord, error) {
	rec, ok := m.Semantics[mediaID]
	if !ok || rec.UserID != userID {
		return nil, nil
	}
	return rec, nil
}

func (m *mockSemanticsRepository) GetScenesByMediaID(ctx context.Context, userID, mediaID uuid.UUID) ([]repository.MediaSceneRecord, error) {
	scenes, ok := m.Scenes[mediaID]
	if !ok {
		return nil, nil
	}
	return scenes, nil
}

func (m *mockSemanticsRepository) SearchSimilarMedia(ctx context.Context, userID uuid.UUID, queryEmbedding pgvector.Vector, limit int, threshold float64) ([]repository.SearchResultItem, error) {
	return nil, nil
}

func (m *mockSemanticsRepository) SearchSimilarScenes(ctx context.Context, userID uuid.UUID, queryEmbedding pgvector.Vector, limit int, threshold float64) ([]repository.SearchResultItem, error) {
	return nil, nil
}

func TestMediaIndexingPipeline_ProcessImage(t *testing.T) {
	analyzerMock := &mockAnalyzer{
		analyzeFunc: func(ctx context.Context, gcsURI, mimeType, mediaType string) (*analyzer.MediaAnalysisResult, error) {
			return &analyzer.MediaAnalysisResult{
				Summary:          "A cute kitten playing with yarn",
				Tags:             []string{"cat", "kitten", "cute", "yarn"},
				DetectedLocation: "Living Room",
				DetectedSeason:   "Winter",
				VisualObjects:    []string{"cat", "yarn"},
				Scenes:           nil,
			}, nil
		},
	}

	embedderMock := &mockEmbedder{}
	repoMock := newMockSemanticsRepository()

	pipeline, err := NewMediaIndexingPipeline("reeler-media-sandbox", analyzerMock, embedderMock, repoMock)
	require.NoError(t, err)

	mediaID := uuid.New()
	userID := uuid.New()

	event := events.MediaConfirmedEvent{
		Event:       events.EventMediaConfirmed,
		MediaID:     mediaID,
		UserID:      userID,
		StoragePath: "user-123/image-456.jpg",
		FileName:    "image-456.jpg",
		MimeType:    "image/jpeg",
		MediaType:   "image",
		Timestamp:   time.Now(),
	}

	err = pipeline.ProcessMediaConfirmed(context.Background(), event)
	require.NoError(t, err)

	sem, err := repoMock.GetSemanticsByMediaID(context.Background(), userID, mediaID)
	require.NoError(t, err)
	require.NotNil(t, sem)
	assert.Equal(t, "A cute kitten playing with yarn", sem.Summary)
	assert.Len(t, sem.Tags, 4)

	scenes, err := repoMock.GetScenesByMediaID(context.Background(), userID, mediaID)
	require.NoError(t, err)
	assert.Empty(t, scenes)
}

func TestMediaIndexingPipeline_ProcessVideo(t *testing.T) {
	analyzerMock := &mockAnalyzer{
		analyzeFunc: func(ctx context.Context, gcsURI, mimeType, mediaType string) (*analyzer.MediaAnalysisResult, error) {
			return &analyzer.MediaAnalysisResult{
				Summary:          "Skateboarder performing trick at sunset",
				Tags:             []string{"skate", "sunset", "action"},
				DetectedLocation: "Venice Beach",
				DetectedSeason:   "Summer",
				Scenes: []analyzer.SceneMetadata{
					{
						StartTimeSeconds: 0.0,
						EndTimeSeconds:   3.0,
						Description:      "Skater accelerating down the ramp",
						Actions:          []string{"skating"},
					},
					{
						StartTimeSeconds: 3.0,
						EndTimeSeconds:   6.0,
						Description:      "Skater landing 360 flip",
						SpeechTranscript: "Yes!",
						Actions:          []string{"jumping", "landing"},
					},
				},
			}, nil
		},
	}

	embedderMock := &mockEmbedder{}
	repoMock := newMockSemanticsRepository()

	pipeline, err := NewMediaIndexingPipeline("reeler-media-sandbox", analyzerMock, embedderMock, repoMock)
	require.NoError(t, err)

	mediaID := uuid.New()
	userID := uuid.New()

	event := events.MediaConfirmedEvent{
		Event:       events.EventMediaConfirmed,
		MediaID:     mediaID,
		UserID:      userID,
		StoragePath: "user-123/clip.mp4",
		FileName:    "clip.mp4",
		MimeType:    "video/mp4",
		MediaType:   "video",
		Timestamp:   time.Now(),
	}

	err = pipeline.ProcessMediaConfirmed(context.Background(), event)
	require.NoError(t, err)

	sem, err := repoMock.GetSemanticsByMediaID(context.Background(), userID, mediaID)
	require.NoError(t, err)
	require.NotNil(t, sem)
	assert.Equal(t, "Skateboarder performing trick at sunset", sem.Summary)

	scenes, err := repoMock.GetScenesByMediaID(context.Background(), userID, mediaID)
	require.NoError(t, err)
	assert.Len(t, scenes, 2)
	assert.Equal(t, 0.0, scenes[0].StartTimeSeconds)
	assert.Equal(t, 3.0, scenes[0].EndTimeSeconds)
	assert.Equal(t, "Yes!", scenes[1].SpeechTranscript)
}
