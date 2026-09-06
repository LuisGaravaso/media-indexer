package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/genai"
)

// AIStudioAnalyzer implements MediaAnalyzer using the Google AI Studio Gemini API key.
type AIStudioAnalyzer struct {
	client        *genai.Client
	storageClient *storage.Client
	modelName     string
}

// NewAIStudioAnalyzer initializes an AI Studio client using the official Google GenAI SDK.
func NewAIStudioAnalyzer(ctx context.Context, apiKey, modelName string) (*AIStudioAnalyzer, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api key cannot be empty")
	}
	if modelName == "" {
		modelName = "gemini-3.5-flash-lite"
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create google genai client: %w", err)
	}

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcs storage client: %w", err)
	}

	return &AIStudioAnalyzer{
		client:        client,
		storageClient: storageClient,
		modelName:     modelName,
	}, nil
}

// Close closes any underlying clients.
func (a *AIStudioAnalyzer) Close() error {
	if a.storageClient != nil {
		return a.storageClient.Close()
	}
	return nil
}

// parseGCSURI parses gs://bucket/object into bucket and object name.
func parseGCSURI(gcsURI string) (bucket string, object string, err error) {
	trimmed := strings.TrimPrefix(gcsURI, "gs://")
	if trimmed == gcsURI {
		return "", "", fmt.Errorf("invalid gcs uri scheme: %s", gcsURI)
	}
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid gcs uri format: %s", gcsURI)
	}
	return parts[0], parts[1], nil
}

// AnalyzeGCSMedia reads object bytes from GCS and performs multimodal analysis via Google AI Studio.
func (a *AIStudioAnalyzer) AnalyzeGCSMedia(ctx context.Context, gcsURI string, mimeType string, mediaType string) (*MediaAnalysisResult, error) {
	if gcsURI == "" {
		return nil, fmt.Errorf("gcsURI cannot be empty")
	}

	if mimeType == "" {
		if strings.ToLower(mediaType) == "image" {
			mimeType = "image/jpeg"
		} else if strings.ToLower(mediaType) == "video" {
			mimeType = "video/mp4"
		} else if strings.ToLower(mediaType) == "audio" {
			mimeType = "audio/mpeg"
		} else {
			mimeType = "application/octet-stream"
		}
	}

	bucket, object, err := parseGCSURI(gcsURI)
	if err != nil {
		return nil, err
	}

	rc, err := a.storageClient.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read object from gcs (%s): %w", gcsURI, err)
	}
	defer func() { _ = rc.Close() }()

	var prompt string
	var payloadBytes []byte
	if strings.ToLower(mediaType) == "image" || strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		prompt = BuildImagePrompt()
		mediaBytes, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("failed reading image bytes from gcs: %w", err)
		}
		payloadBytes = mediaBytes
	} else {
		prompt = BuildVideoPrompt()
		// Stream video reader directly through ffmpeg downscaling to 360p @ 1 FPS
		optimizedBytes, err := DownscaleVideoReader(ctx, rc, 360)
		if err != nil {
			return nil, fmt.Errorf("failed optimizing video stream: %w", err)
		}
		payloadBytes = optimizedBytes
		mimeType = "video/mp4"
	}

	filePart := genai.NewPartFromBytes(payloadBytes, mimeType)
	textPart := genai.NewPartFromText(prompt)
	content := genai.NewContentFromParts([]*genai.Part{filePart, textPart}, genai.RoleUser)

	temp := float32(0.2)
	cfg := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		Temperature:      &temp,
	}

	resp, err := a.client.Models.GenerateContent(ctx, a.modelName, []*genai.Content{content}, cfg)
	if err != nil {
		return nil, fmt.Errorf("gemini api generate content failed: %w", err)
	}

	textResponse := resp.Text()
	if textResponse == "" {
		return nil, fmt.Errorf("empty response received from gemini api")
	}

	rawJSON := CleanJSONResponse(textResponse)
	var result MediaAnalysisResult
	if err := json.Unmarshal([]byte(rawJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse gemini json output: %w (raw: %s)", err, rawJSON)
	}

	if strings.ToLower(mediaType) == "image" || strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		result.Scenes = nil
	}

	return &result, nil
}
