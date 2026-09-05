package database_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationFiles_ExistAndValid(t *testing.T) {
	// Find migrations directory relative to repo root
	rootPath := filepath.Join("..", "..")
	migrationsDir := filepath.Join(rootPath, "migrations")

	entries, err := os.ReadDir(migrationsDir)
	require.NoError(t, err, "migrations directory should exist")

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}

	expectedFiles := []string{
		"000_bootstrap_auth_mock.sql",
		"001_create_media_semantics.sql",
		"002_create_media_scenes.sql",
		"003_add_scene_index_to_media_scenes.sql",
	}

	assert.Equal(t, expectedFiles, sqlFiles, "should have exactly the expected migration files in sequence")

	// Verify content of each migration
	for _, f := range expectedFiles {
		content, err := os.ReadFile(filepath.Join(migrationsDir, f))
		require.NoError(t, err, "migration file should be readable: %s", f)
		assert.NotEmpty(t, strings.TrimSpace(string(content)), "migration file should not be empty: %s", f)
	}

	// Specific assertion for Media Semantics migration
	semanticsSQL, _ := os.ReadFile(filepath.Join(migrationsDir, "001_create_media_semantics.sql"))
	assert.Contains(t, string(semanticsSQL), "CREATE EXTENSION IF NOT EXISTS vector")
	assert.Contains(t, string(semanticsSQL), "CREATE TABLE IF NOT EXISTS public.media_semantics")
	assert.Contains(t, string(semanticsSQL), "embedding vector(768)")
	assert.Contains(t, string(semanticsSQL), "ENABLE ROW LEVEL SECURITY")
	assert.Contains(t, string(semanticsSQL), "idx_media_semantics_embedding_hnsw")
	assert.Contains(t, string(semanticsSQL), "idx_media_semantics_user_type")

	// Specific assertion for Media Scenes migration
	scenesSQL, _ := os.ReadFile(filepath.Join(migrationsDir, "002_create_media_scenes.sql"))
	assert.Contains(t, string(scenesSQL), "CREATE TABLE IF NOT EXISTS public.media_scenes")
	assert.Contains(t, string(scenesSQL), "embedding vector(768)")
	assert.Contains(t, string(scenesSQL), "ENABLE ROW LEVEL SECURITY")
	assert.Contains(t, string(scenesSQL), "idx_media_scenes_embedding_hnsw")
	assert.Contains(t, string(scenesSQL), "idx_media_scenes_media_timeline")
}
