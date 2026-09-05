package embeddings

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// AIStudioEmbedder generates text vector embeddings using Google AI Studio API key.
type AIStudioEmbedder struct {
	client    *genai.Client
	modelName string
}

// NewAIStudioEmbedder creates a new AIStudioEmbedder using the GenAI SDK.
func NewAIStudioEmbedder(ctx context.Context, apiKey, modelName string) (*AIStudioEmbedder, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api key cannot be empty")
	}
	if modelName == "" {
		modelName = "text-embedding-004"
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create google genai client for embeddings: %w", err)
	}

	return &AIStudioEmbedder{
		client:    client,
		modelName: modelName,
	}, nil
}

// EmbedText generates a vector embedding for a single text using Google AI Studio.
func (a *AIStudioEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	content := genai.NewContentFromText(text, genai.RoleUser)
	resp, err := a.client.Models.EmbedContent(ctx, a.modelName, []*genai.Content{content}, nil)
	if err != nil {
		return nil, fmt.Errorf("gemini api embed content failed: %w", err)
	}

	if resp == nil || len(resp.Embeddings) == 0 || len(resp.Embeddings[0].Values) == 0 {
		return nil, fmt.Errorf("empty embedding values returned from gemini api")
	}

	return resp.Embeddings[0].Values, nil
}

// EmbedBatch generates embeddings for a slice of strings sequentially or concurrently.
func (a *AIStudioEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	results := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := a.EmbedText(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed embedding item at index %d: %w", i, err)
		}
		results[i] = emb
	}

	return results, nil
}
