package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/vertexai/genai"
)

// GeminiAnalyzer implements MediaAnalyzer using Vertex AI Gemini models.
type GeminiAnalyzer struct {
	client    *genai.Client
	modelName string
}

// NewGeminiAnalyzer initializes a Vertex AI Gemini client for multimodal analysis.
func NewGeminiAnalyzer(ctx context.Context, projectID, location, modelName string) (*GeminiAnalyzer, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}
	if location == "" {
		location = "us-central1"
	}
	if modelName == "" {
		modelName = "gemini-2.0-flash"
	}

	client, err := genai.NewClient(ctx, projectID, location)
	if err != nil {
		return nil, fmt.Errorf("failed to create vertex genai client: %w", err)
	}

	return &GeminiAnalyzer{
		client:    client,
		modelName: modelName,
	}, nil
}

// Close closes the underlying Vertex AI client.
func (g *GeminiAnalyzer) Close() error {
	if g.client != nil {
		return g.client.Close()
	}
	return nil
}

// BuildImagePrompt returns the prompt specifically designed for single-image semantic extraction.
func BuildImagePrompt() string {
	return `You are an expert visual semantic analyzer. Analyze the provided image and return a strictly valid JSON object matching this schema:
{
  "summary": "Concise 1-3 sentence summary of what is seen in the image, the subjects, context, and aesthetic mood.",
  "tags": ["tag1", "tag2", "tag3", "tag4", "tag5"],
  "detected_location": "Geographic or contextual place if identifiable (e.g. 'Beach, Hawaii', 'Office', 'Paris, France', or 'Unknown')",
  "detected_season": "Season if recognizable ('Spring', 'Summer', 'Autumn', 'Winter', 'Year-round', or 'Unknown')",
  "visual_objects": ["person", "car", "dog", "sunglasses"],
  "scenes": []
}

Rules:
- Output ONLY the JSON object. Do not include markdown code block formatting or backticks.
- For images, the "scenes" array MUST be empty.
- Provide descriptive, searchable tags (at least 5-10 tags).`
}

// BuildVideoPrompt returns the prompt for multimodal video narrative & scene cut extraction.
func BuildVideoPrompt() string {
	return `You are an expert video and cinematography semantic analyzer. Analyze the entire provided video and return a strictly valid JSON object matching this schema:
{
  "summary": "Concise 2-4 sentence narrative summary of the entire video storyline, key subjects, setting, and overall mood.",
  "tags": ["tag1", "tag2", "tag3", "tag4", "tag5"],
  "detected_location": "Geographic or contextual place (e.g. 'Skatepark, Venice Beach', 'Kitchen', or 'Unknown')",
  "detected_season": "Season ('Spring', 'Summer', 'Autumn', 'Winter', 'Year-round', or 'Unknown')",
  "visual_objects": ["skateboard", "ramp", "people", "camera"],
  "scenes": [
    {
      "start_time_seconds": 0.0,
      "end_time_seconds": 3.5,
      "description": "Close-up shot of skater tying shoelaces and preparing skateboard.",
      "speech_transcript": "Spoken dialogue in this segment if present",
      "mood": "anticipation",
      "actions": ["tying laces", "stepping on board"]
    }
  ]
}

Rules:
- Output ONLY the JSON object. Do not include markdown code block formatting or backticks.
- Break down video into temporal scene segments covering the full duration.
- Each scene must have accurate start/end timestamps, visual description, mood, and actions.
- Provide descriptive, searchable tags covering global theme, mood, objects, and setting.`
}

// CleanJSONResponse strips markdown fences and surrounding whitespace from LLM text output.
func CleanJSONResponse(raw string) string {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	return strings.TrimSpace(cleaned)
}

// AnalyzeGCSMedia performs multimodal analysis on a media file stored in GCS.
func (g *GeminiAnalyzer) AnalyzeGCSMedia(ctx context.Context, gcsURI string, mimeType string, mediaType string) (*MediaAnalysisResult, error) {
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

	model := g.client.GenerativeModel(g.modelName)
	model.ResponseMIMEType = "application/json"
	model.SetTemperature(0.2)

	var prompt string
	if strings.ToLower(mediaType) == "image" || strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		prompt = BuildImagePrompt()
	} else {
		prompt = BuildVideoPrompt()
	}

	filePart := genai.FileData{
		MIMEType: mimeType,
		FileURI:  gcsURI,
	}

	resp, err := model.GenerateContent(ctx, filePart, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gemini generate content failed: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response received from gemini")
	}

	var sb strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			sb.WriteString(string(textPart))
		}
	}

	rawJSON := CleanJSONResponse(sb.String())
	var result MediaAnalysisResult
	if err := json.Unmarshal([]byte(rawJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse gemini json output: %w (raw: %s)", err, rawJSON)
	}

	if strings.ToLower(mediaType) == "image" || strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		result.Scenes = nil
	}

	return &result, nil
}
