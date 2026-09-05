package pipeline

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/LuisGaravaso/media-indexer/internal/analyzer"
	"github.com/LuisGaravaso/media-indexer/internal/embeddings"
	"github.com/LuisGaravaso/media-indexer/internal/events"
	"github.com/LuisGaravaso/media-indexer/internal/repository"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// MediaIndexingPipeline coordinates multimodal analysis, vector embedding generation, and database storage.
type MediaIndexingPipeline struct {
	bucketName string
	analyzer   analyzer.MediaAnalyzer
	embedder   embeddings.Embedder
	repo       repository.SemanticsRepository
}

// NewMediaIndexingPipeline creates a new MediaIndexingPipeline.
func NewMediaIndexingPipeline(bucketName string, analyzer analyzer.MediaAnalyzer, embedder embeddings.Embedder, repo repository.SemanticsRepository) (*MediaIndexingPipeline, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucketName cannot be empty")
	}
	if analyzer == nil {
		return nil, fmt.Errorf("analyzer cannot be nil")
	}
	if embedder == nil {
		return nil, fmt.Errorf("embedder cannot be nil")
	}
	if repo == nil {
		return nil, fmt.Errorf("repository cannot be nil")
	}

	return &MediaIndexingPipeline{
		bucketName: bucketName,
		analyzer:   analyzer,
		embedder:   embedder,
		repo:       repo,
	}, nil
}

// ProcessMediaConfirmed fulfills the events.EventProcessor interface.
func (p *MediaIndexingPipeline) ProcessMediaConfirmed(ctx context.Context, event events.MediaConfirmedEvent) error {
	log.Printf("[Pipeline] Starting indexing for media %s (user: %s, file: %s, type: %s)",
		event.MediaID, event.UserID, event.FileName, event.MediaType)

	gcsURI := fmt.Sprintf("gs://%s/%s", p.bucketName, event.StoragePath)

	// Step 1: Multimodal Gemini analysis
	analysis, err := p.analyzer.AnalyzeGCSMedia(ctx, gcsURI, event.MimeType, event.MediaType)
	if err != nil {
		return fmt.Errorf("analysis failed for media %s: %w", event.MediaID, err)
	}

	// Step 2: Global semantic embedding (summary + location + season + objects + tags)
	var textParts []string
	if analysis.Summary != "" {
		textParts = append(textParts, analysis.Summary)
	}
	if analysis.DetectedLocation != "" && !strings.EqualFold(analysis.DetectedLocation, "unknown") {
		textParts = append(textParts, "Location: "+analysis.DetectedLocation)
	}
	if analysis.DetectedSeason != "" && !strings.EqualFold(analysis.DetectedSeason, "unknown") {
		textParts = append(textParts, "Season/Climate: "+analysis.DetectedSeason)
	}
	if len(analysis.VisualObjects) > 0 {
		textParts = append(textParts, "Objects: "+strings.Join(analysis.VisualObjects, ", "))
	}
	if len(analysis.Tags) > 0 {
		textParts = append(textParts, "Tags: "+strings.Join(analysis.Tags, ", "))
	}
	combinedGlobalText := strings.Join(textParts, ". ")

	globalEmbeddingSlice, err := p.embedder.EmbedText(ctx, combinedGlobalText)
	if err != nil {
		return fmt.Errorf("failed generating global embedding for media %s: %w", event.MediaID, err)
	}

	semanticsRecord := &repository.MediaSemanticsRecord{
		MediaID:          event.MediaID,
		UserID:           event.UserID,
		MediaType:        event.MediaType,
		Summary:          analysis.Summary,
		Tags:             analysis.Tags,
		DetectedLocation: analysis.DetectedLocation,
		DetectedSeason:   analysis.DetectedSeason,
		VisualObjects:    analysis.VisualObjects,
		Embedding:        pgvector.NewVector(globalEmbeddingSlice),
	}

	// Step 3: Embed scenes for video files (skip for image files)
	var sceneRecords []repository.MediaSceneRecord
	if (event.MediaType == "video" || strings.HasPrefix(event.MimeType, "video/")) && len(analysis.Scenes) > 0 {
		sceneTexts := make([]string, len(analysis.Scenes))
		for i, scene := range analysis.Scenes {
			text := scene.Description
			if scene.SpeechTranscript != "" {
				text += " Transcript: " + scene.SpeechTranscript
			}
			if len(scene.Actions) > 0 {
				text += " Actions: " + strings.Join(scene.Actions, ", ")
			}
			sceneTexts[i] = text
		}

		sceneEmbeddings, err := p.embedder.EmbedBatch(ctx, sceneTexts)
		if err != nil {
			return fmt.Errorf("failed generating scene embeddings for media %s: %w", event.MediaID, err)
		}

		for i, scene := range analysis.Scenes {
			sceneRecords = append(sceneRecords, repository.MediaSceneRecord{
				ID:               uuid.New(),
				MediaID:          event.MediaID,
				UserID:           event.UserID,
				SceneIndex:       i,
				StartTimeSeconds: scene.StartTimeSeconds,
				EndTimeSeconds:   scene.EndTimeSeconds,
				Description:      scene.Description,
				SpeechTranscript: scene.SpeechTranscript,
				Mood:             scene.Mood,
				Actions:          scene.Actions,
				Embedding:        pgvector.NewVector(sceneEmbeddings[i]),
			})
		}
	}

	// Step 4: Persist in PostgreSQL (pgvector)
	if err := p.repo.SaveMediaIndex(ctx, semanticsRecord, sceneRecords); err != nil {
		return fmt.Errorf("failed persisting media index for media %s: %w", event.MediaID, err)
	}

	log.Printf("[Pipeline] Successfully indexed media %s (%d scenes recorded)", event.MediaID, len(sceneRecords))
	return nil
}
