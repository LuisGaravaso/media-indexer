package search

import (
	"context"
	"fmt"
	"sort"

	"github.com/LuisGaravaso/media-indexer/internal/embeddings"
	"github.com/LuisGaravaso/media-indexer/internal/repository"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// SemanticSearchRequest holds parameters for querying natural language vector similarity.
type SemanticSearchRequest struct {
	Query         string   `json:"query" binding:"required"`
	MediaType     string   `json:"media_type,omitempty"`     // "image", "video", or "" for both
	Limit         int      `json:"limit,omitempty"`          // Default 20, max 100
	Threshold     *float64 `json:"threshold,omitempty"`      // Cosine similarity minimum threshold
	IncludeScenes *bool    `json:"include_scenes,omitempty"` // Default true
}

// SceneMatch contains timestamped scene breakdown with similarity score.
type SceneMatch struct {
	SceneIndex       int     `json:"scene_index"`
	StartTimeSeconds float64 `json:"start_time_seconds"`
	EndTimeSeconds   float64 `json:"end_time_seconds"`
	Description      string  `json:"description"`
	Similarity       float64 `json:"similarity"`
}

// SemanticSearchResult represents a ranked media item matching the natural language search.
type SemanticSearchResult struct {
	MediaID          uuid.UUID    `json:"media_id"`
	UserID           uuid.UUID    `json:"user_id"`
	Summary          string       `json:"summary"`
	Tags             []string     `json:"tags"`
	DetectedLocation string       `json:"detected_location,omitempty"`
	DetectedSeason   string       `json:"detected_season,omitempty"`
	Similarity       float64      `json:"similarity"`
	MatchingScenes   []SceneMatch `json:"matching_scenes,omitempty"`
}

// Service defines semantic search operations.
type Service interface {
	Search(ctx context.Context, userID uuid.UUID, req SemanticSearchRequest) ([]SemanticSearchResult, error)
}

// SemanticSearchService implements Service using Embedder and SemanticsRepository.
type SemanticSearchService struct {
	embedder embeddings.Embedder
	repo     repository.SemanticsRepository
}

// NewSemanticSearchService creates a new SemanticSearchService.
func NewSemanticSearchService(embedder embeddings.Embedder, repo repository.SemanticsRepository) (*SemanticSearchService, error) {
	if embedder == nil {
		return nil, fmt.Errorf("embedder cannot be nil")
	}
	if repo == nil {
		return nil, fmt.Errorf("repository cannot be nil")
	}

	return &SemanticSearchService{
		embedder: embedder,
		repo:     repo,
	}, nil
}

// Search performs embedding of the query and multi-tiered vector search across media and scene clips.
func (s *SemanticSearchService) Search(ctx context.Context, userID uuid.UUID, req SemanticSearchRequest) ([]SemanticSearchResult, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	threshold := 0.3
	if req.Threshold != nil {
		threshold = *req.Threshold
	}

	includeScenes := true
	if req.IncludeScenes != nil {
		includeScenes = *req.IncludeScenes
	}

	// 1. Embed query
	queryVectorSlice, err := s.embedder.EmbedText(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed embedding search query: %w", err)
	}
	queryVector := pgvector.NewVector(queryVectorSlice)

	// 2. Search media_semantics
	mediaMatches, err := s.repo.SearchSimilarMedia(ctx, userID, queryVector, limit, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed querying similar media: %w", err)
	}

	resultsMap := make(map[uuid.UUID]*SemanticSearchResult)
	for _, m := range mediaMatches {
		resultsMap[m.MediaID] = &SemanticSearchResult{
			MediaID:          m.MediaID,
			UserID:           m.UserID,
			Summary:          m.Summary,
			Tags:             m.Tags,
			DetectedLocation: m.DetectedLocation,
			DetectedSeason:   m.DetectedSeason,
			Similarity:       m.Similarity,
		}
	}

	// 3. Search media_scenes if enabled
	if includeScenes && req.MediaType != "image" {
		sceneMatches, err := s.repo.SearchSimilarScenes(ctx, userID, queryVector, limit*2, threshold)
		if err != nil {
			return nil, fmt.Errorf("failed querying similar scenes: %w", err)
		}

		for _, sm := range sceneMatches {
			res, exists := resultsMap[sm.MediaID]
			if !exists {
				// Fetch parent media summary if not already in result map
				parentSem, err := s.repo.GetSemanticsByMediaID(ctx, userID, sm.MediaID)
				if err == nil && parentSem != nil {
					res = &SemanticSearchResult{
						MediaID:          parentSem.MediaID,
						UserID:           parentSem.UserID,
						Summary:          parentSem.Summary,
						Tags:             parentSem.Tags,
						DetectedLocation: parentSem.DetectedLocation,
						DetectedSeason:   parentSem.DetectedSeason,
						Similarity:       sm.Similarity, // use highest scene similarity
					}
					resultsMap[sm.MediaID] = res
				}
			}

			if res != nil {
				if sm.Similarity > res.Similarity {
					res.Similarity = sm.Similarity
				}

				if sm.SceneIndex != nil && sm.StartTimeSeconds != nil && sm.EndTimeSeconds != nil && sm.SceneDescription != nil {
					res.MatchingScenes = append(res.MatchingScenes, SceneMatch{
						SceneIndex:       *sm.SceneIndex,
						StartTimeSeconds: *sm.StartTimeSeconds,
						EndTimeSeconds:   *sm.EndTimeSeconds,
						Description:      *sm.SceneDescription,
						Similarity:       sm.Similarity,
					})
				}
			}
		}
	}

	var finalResults []SemanticSearchResult
	for _, res := range resultsMap {
		finalResults = append(finalResults, *res)
	}

	// Rank by descending similarity
	sort.Slice(finalResults, func(i, j int) bool {
		return finalResults[i].Similarity > finalResults[j].Similarity
	})

	if len(finalResults) > limit {
		finalResults = finalResults[:limit]
	}

	return finalResults, nil
}
