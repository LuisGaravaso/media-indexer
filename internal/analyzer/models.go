package analyzer

import "context"

// SceneMetadata represents temporal scene cuts within a video.
type SceneMetadata struct {
	StartTimeSeconds float64  `json:"start_time_seconds"`
	EndTimeSeconds   float64  `json:"end_time_seconds"`
	Description      string   `json:"description"`
	SpeechTranscript string   `json:"speech_transcript,omitempty"`
	Mood             string   `json:"mood,omitempty"`
	Actions          []string `json:"actions,omitempty"`
}

// MediaAnalysisResult contains global multimodal semantic metadata and scene breakdowns.
type MediaAnalysisResult struct {
	Summary          string          `json:"summary"`
	Tags             []string        `json:"tags"`
	DetectedLocation string          `json:"detected_location,omitempty"`
	DetectedSeason   string          `json:"detected_season,omitempty"`
	VisualObjects    []string        `json:"visual_objects,omitempty"`
	Scenes           []SceneMetadata `json:"scenes,omitempty"`
}

// MediaAnalyzer defines the contract for analyzing images and videos using multimodal LLMs.
type MediaAnalyzer interface {
	AnalyzeGCSMedia(ctx context.Context, gcsURI string, mimeType string, mediaType string) (*MediaAnalysisResult, error)
}
