package analyzer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockAnalyzer struct {
	AnalyzeFunc func(ctx context.Context, gcsURI, mimeType, mediaType string) (*MediaAnalysisResult, error)
}

func (m *MockAnalyzer) AnalyzeGCSMedia(ctx context.Context, gcsURI, mimeType, mediaType string) (*MediaAnalysisResult, error) {
	if m.AnalyzeFunc != nil {
		return m.AnalyzeFunc(ctx, gcsURI, mimeType, mediaType)
	}
	return nil, nil
}

func TestBuildImagePrompt(t *testing.T) {
	prompt := BuildImagePrompt()
	assert.Contains(t, prompt, "summary")
	assert.Contains(t, prompt, "tags")
	assert.Contains(t, prompt, "scenes")
	assert.Contains(t, prompt, "MUST be empty")
}

func TestBuildVideoPrompt(t *testing.T) {
	prompt := BuildVideoPrompt()
	assert.Contains(t, prompt, "summary")
	assert.Contains(t, prompt, "scenes")
	assert.Contains(t, prompt, "start_time_seconds")
	assert.Contains(t, prompt, "end_time_seconds")
}

func TestCleanJSONResponse(t *testing.T) {
	raw := "```json\n{\"summary\": \"test\"}\n```"
	cleaned := CleanJSONResponse(raw)
	assert.Equal(t, "{\"summary\": \"test\"}", cleaned)

	raw2 := "```\n{\"summary\": \"test2\"}\n```"
	cleaned2 := CleanJSONResponse(raw2)
	assert.Equal(t, "{\"summary\": \"test2\"}", cleaned2)
}

func TestImageAnalysis_JSONUnmarshalling(t *testing.T) {
	sampleImageJSON := `{
		"summary": "A golden retriever playing on a sunny beach at sunset.",
		"tags": ["dog", "golden retriever", "beach", "sunset", "ocean", "playful"],
		"detected_location": "Malibu Beach, California",
		"detected_season": "Summer",
		"visual_objects": ["dog", "ocean", "sand", "ball"],
		"scenes": []
	}`

	var result MediaAnalysisResult
	err := json.Unmarshal([]byte(sampleImageJSON), &result)
	require.NoError(t, err)

	assert.Equal(t, "A golden retriever playing on a sunny beach at sunset.", result.Summary)
	assert.Len(t, result.Tags, 6)
	assert.Equal(t, "Malibu Beach, California", result.DetectedLocation)
	assert.Equal(t, "Summer", result.DetectedSeason)
	assert.Empty(t, result.Scenes)
}

func TestVideoAnalysis_JSONUnmarshalling(t *testing.T) {
	sampleVideoJSON := `{
		"summary": "A skateboarder practicing kickflips at a skatepark during the afternoon.",
		"tags": ["skateboarding", "skatepark", "kickflip", "urban", "action"],
		"detected_location": "Venice Beach Skatepark",
		"detected_season": "Spring",
		"visual_objects": ["skateboard", "ramp", "skater"],
		"scenes": [
			{
				"start_time_seconds": 0.0,
				"end_time_seconds": 2.5,
				"description": "Skater prepares at top of the ramp.",
				"mood": "focused",
				"actions": ["aligning board", "looking forward"]
			},
			{
				"start_time_seconds": 2.5,
				"end_time_seconds": 6.0,
				"description": "Skater drops into bowl and performs a kickflip.",
				"speech_transcript": "Let's go!",
				"mood": "energetic",
				"actions": ["dropping in", "kickflip", "landing"]
			}
		]
	}`

	var result MediaAnalysisResult
	err := json.Unmarshal([]byte(sampleVideoJSON), &result)
	require.NoError(t, err)

	assert.Equal(t, "A skateboarder practicing kickflips at a skatepark during the afternoon.", result.Summary)
	assert.Len(t, result.Tags, 5)
	assert.Len(t, result.Scenes, 2)
	assert.Equal(t, 0.0, result.Scenes[0].StartTimeSeconds)
	assert.Equal(t, 2.5, result.Scenes[0].EndTimeSeconds)
	assert.Equal(t, "Let's go!", result.Scenes[1].SpeechTranscript)
}

func TestParseGCSURI(t *testing.T) {
	bucket, obj, err := parseGCSURI("gs://my-bucket/path/to/media.jpg")
	require.NoError(t, err)
	assert.Equal(t, "my-bucket", bucket)
	assert.Equal(t, "path/to/media.jpg", obj)

	_, _, err = parseGCSURI("http://example.com/file.jpg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid gcs uri scheme")

	_, _, err = parseGCSURI("gs://bucket-only")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid gcs uri format")
}

func TestNewAIStudioAnalyzer_Validation(t *testing.T) {
	_, err := NewAIStudioAnalyzer(context.Background(), "", "gemini-2.5-flash")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gemini api key cannot be empty")
}
