package server

import (
	"net/http"

	"github.com/LuisGaravaso/media-indexer/internal/middleware"
	"github.com/LuisGaravaso/media-indexer/internal/search"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SearchHandler handles natural language semantic search queries.
func SearchHandler(svc search.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, exists := middleware.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID in context"})
			return
		}

		var req search.SemanticSearchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}

		results, err := svc.Search(c.Request.Context(), userID, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed: " + err.Error()})
			return
		}

		if results == nil {
			results = []search.SemanticSearchResult{}
		}

		c.JSON(http.StatusOK, gin.H{
			"results": results,
			"count":   len(results),
		})
	}
}
