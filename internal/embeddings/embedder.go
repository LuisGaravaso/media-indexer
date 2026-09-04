package embeddings

import (
	"context"
	"fmt"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	aiplatformpb "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/api/option"
)

// Embedder defines the contract for generating text vector embeddings.
type Embedder interface {
	EmbedText(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// VertexEmbedder generates embeddings using Vertex AI PredictionClient and text-embedding models.
type VertexEmbedder struct {
	client    *aiplatform.PredictionClient
	modelPath string
	projectID string
	location  string
	modelName string
}

// NewVertexEmbedder creates a new VertexEmbedder.
func NewVertexEmbedder(ctx context.Context, projectID, location, modelName string) (*VertexEmbedder, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}
	if location == "" {
		location = "us-central1"
	}
	if modelName == "" {
		modelName = "text-embedding-004"
	}

	apiEndpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", location)
	client, err := aiplatform.NewPredictionClient(ctx, option.WithEndpoint(apiEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create vertex ai prediction client: %w", err)
	}

	modelPath := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", projectID, location, modelName)

	return &VertexEmbedder{
		client:    client,
		modelPath: modelPath,
		projectID: projectID,
		location:  location,
		modelName: modelName,
	}, nil
}

// Close closes the underlying Vertex client.
func (v *VertexEmbedder) Close() error {
	if v.client != nil {
		return v.client.Close()
	}
	return nil
}

// EmbedText generates a vector embedding for a single string.
func (v *VertexEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	model := v.modelPath
	req := &aiplatformpb.EmbedContentRequest{
		Model: &model,
		Content: &aiplatformpb.Content{
			Role: "user",
			Parts: []*aiplatformpb.Part{
				{
					Data: &aiplatformpb.Part_Text{
						Text: text,
					},
				},
			},
		},
	}

	res, err := v.client.EmbedContent(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vertex embed content failed: %w", err)
	}

	if res == nil || res.Embedding == nil || len(res.Embedding.Values) == 0 {
		return nil, fmt.Errorf("empty embedding returned from vertex ai")
	}

	return res.Embedding.Values, nil
}

// EmbedBatch generates embeddings for a slice of strings sequentially or concurrently.
func (v *VertexEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	results := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := v.EmbedText(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed embedding item at index %d: %w", i, err)
		}
		results[i] = emb
	}

	return results, nil
}
