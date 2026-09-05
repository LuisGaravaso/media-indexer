package embeddings

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockEmbedder struct {
	EmbedTextFunc  func(ctx context.Context, text string) ([]float32, error)
	EmbedBatchFunc func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *MockEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if m.EmbedTextFunc != nil {
		return m.EmbedTextFunc(ctx, text)
	}
	vec := make([]float32, 768)
	for i := range vec {
		vec[i] = 0.01
	}
	return vec, nil
}

func (m *MockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if m.EmbedBatchFunc != nil {
		return m.EmbedBatchFunc(ctx, texts)
	}
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

func TestMockEmbedder_Dimensions(t *testing.T) {
	mock := &MockEmbedder{}
	vec, err := mock.EmbedText(context.Background(), "golden retriever on a sunny beach")
	require.NoError(t, err)
	assert.Len(t, vec, 768)

	batch, err := mock.EmbedBatch(context.Background(), []string{"first scene", "second scene"})
	require.NoError(t, err)
	assert.Len(t, batch, 2)
	assert.Len(t, batch[0], 768)
	assert.Len(t, batch[1], 768)
}

func TestNewVertexEmbedder_Validation(t *testing.T) {
	_, err := NewVertexEmbedder(context.Background(), "", "us-central1", "text-embedding-004")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project ID cannot be empty")
}

func TestNewAIStudioEmbedder_Validation(t *testing.T) {
	_, err := NewAIStudioEmbedder(context.Background(), "", "text-embedding-004")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gemini api key cannot be empty")
}
