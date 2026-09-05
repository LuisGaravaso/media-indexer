-- =============================================================================
-- Migration 002: Create Media Scenes Table & RLS Policies
-- =============================================================================

CREATE TABLE IF NOT EXISTS public.media_scenes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    media_id UUID NOT NULL REFERENCES public.media(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    scene_index INT NOT NULL DEFAULT 0,
    start_time_seconds NUMERIC(8,2) NOT NULL,
    end_time_seconds NUMERIC(8,2) NOT NULL,
    description TEXT NOT NULL,
    speech_transcript TEXT,
    mood TEXT,
    actions TEXT[] NOT NULL DEFAULT '{}',
    embedding vector(768), -- Vertex AI text-embedding-004
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_scene_timestamps CHECK (end_time_seconds > start_time_seconds)
);

-- Timeline ordering index on media_id
CREATE INDEX IF NOT EXISTS idx_media_scenes_media_timeline ON public.media_scenes (media_id, start_time_seconds ASC);

-- Multi-tenant lookup index
CREATE INDEX IF NOT EXISTS idx_media_scenes_user_id ON public.media_scenes (user_id);

-- HNSW Vector Index for granular scene similarity search (<=> operator)
CREATE INDEX IF NOT EXISTS idx_media_scenes_embedding_hnsw 
    ON public.media_scenes 
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- Row Level Security
ALTER TABLE public.media_scenes ENABLE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE tablename = 'media_scenes' AND policyname = 'Users can view own media scenes'
    ) THEN
        CREATE POLICY "Users can view own media scenes"
            ON public.media_scenes FOR SELECT
            TO authenticated
            USING (auth.uid() = user_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE tablename = 'media_scenes' AND policyname = 'Users can insert own media scenes'
    ) THEN
        CREATE POLICY "Users can insert own media scenes"
            ON public.media_scenes FOR INSERT
            TO authenticated
            WITH CHECK (auth.uid() = user_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE tablename = 'media_scenes' AND policyname = 'Users can update own media scenes'
    ) THEN
        CREATE POLICY "Users can update own media scenes"
            ON public.media_scenes FOR UPDATE
            TO authenticated
            USING (auth.uid() = user_id);
    END IF;
END
$$;
