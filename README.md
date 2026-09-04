# Media Indexer

Multimodal AI indexing and semantic search microservice for the **Reeler** ecosystem.

## Features
- **Event Consumer:** Asynchronously processes `media.confirmed` events from GCP Pub/Sub.
- **Multimodal AI Analysis:** Leverages Google Gemini 2.0 Flash to extract summaries, tags, seasons, locations, and timestamped scene breakdowns.
- **Vector Embeddings:** Generates 768-dimensional embeddings via Google Vertex AI `text-embedding-004`.
- **Relational & Vector Store:** Persists structured semantics and vector embeddings into Supabase PostgreSQL (`pgvector`) with strict Row-Level Security (RLS).
- **Semantic Search API:** Exposes `POST /api/v1/search/semantic` for natural language querying across video scenes and images.
