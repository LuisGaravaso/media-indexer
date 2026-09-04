-- =============================================================================
-- Migration 001: Enable pgvector and Create Media Semantics Table & RLS Policies
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS public.media_semantics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    media_id UUID NOT NULL REFERENCES public.media(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    media_type TEXT NOT NULL, -- 'video' | 'image' | 'audio'
    summary TEXT NOT NULL,
    tags TEXT[] NOT NULL DEFAULT '{}',
    detected_season TEXT,
    detected_location TEXT,
    embedding vector(768), -- Vertex AI text-embedding-004
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_media_semantics_media_id UNIQUE (media_id)
);

-- Fast user & media_type lookup index
CREATE INDEX IF NOT EXISTS idx_media_semantics_user_type ON public.media_semantics (user_id, media_type);

-- HNSW Vector Index for fast cosine similarity search (<=> operator)
CREATE INDEX IF NOT EXISTS idx_media_semantics_embedding_hnsw 
    ON public.media_semantics 
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- GIN index on tags for fast keyword/tag filtering
CREATE INDEX IF NOT EXISTS idx_media_semantics_tags_gin ON public.media_semantics USING GIN (tags);

-- Row Level Security
ALTER TABLE public.media_semantics ENABLE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE tablename = 'media_semantics' AND policyname = 'Users can view own media semantics'
    ) THEN
        CREATE POLICY "Users can view own media semantics"
            ON public.media_semantics FOR SELECT
            TO authenticated
            USING (auth.uid() = user_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE tablename = 'media_semantics' AND policyname = 'Users can insert own media semantics'
    ) THEN
        CREATE POLICY "Users can insert own media semantics"
            ON public.media_semantics FOR INSERT
            TO authenticated
            WITH CHECK (auth.uid() = user_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE tablename = 'media_semantics' AND policyname = 'Users can update own media semantics'
    ) THEN
        CREATE POLICY "Users can update own media semantics"
            ON public.media_semantics FOR UPDATE
            TO authenticated
            USING (auth.uid() = user_id);
    END IF;
END
$$;
