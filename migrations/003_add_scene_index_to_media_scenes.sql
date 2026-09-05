-- =============================================================================
-- Migration 003: Add scene_index to media_scenes table if missing
-- =============================================================================

ALTER TABLE public.media_scenes ADD COLUMN IF NOT EXISTS scene_index INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_media_scenes_scene_index ON public.media_scenes (media_id, scene_index ASC);
