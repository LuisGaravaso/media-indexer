package embeddings

import (
	"context"
	"fmt"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	aiplatformpb "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
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

// EmbedText generates a vector embedding for a single string using Vertex AI Predict API.
func (v *VertexEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	instance, err := structpb.NewStruct(map[string]interface{}{
		"content": text,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create predict instance: %w", err)
	}

	req := &aiplatformpb.PredictRequest{
		Endpoint:  v.modelPath,
		Instances: []*structpb.Value{structpb.NewStructValue(instance)},
	}

	res, err := v.client.Predict(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vertex predict embeddings failed: %w", err)
	}

	if res == nil || len(res.Predictions) == 0 {
		return nil, fmt.Errorf("empty predictions returned from vertex ai")
	}

	predStruct := res.Predictions[0].GetStructValue()
	if predStruct == nil {
		return nil, fmt.Errorf("invalid prediction structure returned from vertex ai")
	}

	embeddingsField := predStruct.Fields["embeddings"]
	if embeddingsField == nil || embeddingsField.GetStructValue() == nil {
		return nil, fmt.Errorf("embeddings field missing in vertex response")
	}

	valuesList := embeddingsField.GetStructValue().Fields["values"]
	if valuesList == nil || valuesList.GetListValue() == nil {
		return nil, fmt.Errorf("embedding values list missing in vertex response")
	}

	rawValues := valuesList.GetListValue().GetValues()
	if len(rawValues) == 0 {
		return nil, fmt.Errorf("empty embedding values returned from vertex ai")
	}

	embedding := make([]float32, len(rawValues))
	for i, v := range rawValues {
		embedding[i] = float32(v.GetNumberValue())
	}

	return embedding, nil
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
