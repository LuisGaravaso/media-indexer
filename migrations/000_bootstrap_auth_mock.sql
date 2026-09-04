-- =============================================================================
-- Migration 000: Bootstrap Auth & Media Mock (Local Development & Standalone Postgres)
-- Guarded so it executes safely on both local PostgreSQL and Supabase.
-- =============================================================================

DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
    CREATE EXTENSION IF NOT EXISTS "pgcrypto";
EXCEPTION WHEN OTHERS THEN
    NULL;
END
$$;

-- Local mock auth schema for development when not inside managed Supabase
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'auth') THEN
        CREATE SCHEMA auth;
    END IF;
EXCEPTION WHEN OTHERS THEN
    NULL;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'auth' AND table_name = 'users') THEN
        CREATE TABLE auth.users (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            email TEXT UNIQUE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT now()
        );
    END IF;
EXCEPTION WHEN OTHERS THEN
    NULL;
END
$$;

-- Mock auth.uid() function returning session claim or null (only if auth.uid does not exist)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'uid' AND pronamespace = 'auth'::regnamespace) THEN
        CREATE OR REPLACE FUNCTION auth.uid()
        RETURNS UUID AS $func$
        BEGIN
            RETURN NULLIF(current_setting('request.jwt.claim.sub', true), '')::uuid;
        EXCEPTION WHEN OTHERS THEN
            RETURN NULL;
        END;
        $func$ LANGUAGE plpgsql STABLE;
    END IF;
EXCEPTION WHEN OTHERS THEN
    NULL;
END
$$;

-- Ensure authenticated role exists for local testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'authenticated') THEN
        CREATE ROLE authenticated;
    END IF;
EXCEPTION WHEN OTHERS THEN
    NULL;
END
$$;

-- Standalone mock media table for FK resolution in local/isolated test environments
CREATE TABLE IF NOT EXISTS public.media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    storage_path TEXT NOT NULL UNIQUE,
    file_name TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    media_type TEXT GENERATED ALWAYS AS (split_part(mime_type, '/', 1)) STORED,
    file_size_bytes BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ready',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
