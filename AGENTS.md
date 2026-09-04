# AGENTS.md

Context and operational guidelines for AI agents working on **Media Indexer** (part of the **Reeler** ecosystem).

## 1. Stack & Architecture
- **Tech Stack:** Go 1.26, Gin, pgx/v5, Supabase PostgreSQL (pgvector, RLS), Google GenAI / Vertex AI SDK (`gemini-2.0-flash`, `text-embedding-004`), GCP Pub/Sub (`cloud.google.com/go/pubsub`), Testify, GCP Cloud Run.
- **Environments:** Dual GCP Projects (`reeler-sandbox` & `reeler-prod`).
- **CI/CD & Auth:** GitHub Actions, Docker Buildx, Artifact Registry, Keyless Auth via Workload Identity Federation (WIF).
- **Security & Scale:** Private IAM invocation only (no `allUsers`), `--min-instances=0`, `--max-instances=2`, `--concurrency=80`.
- **Health Checks:** Use `/health` or `/ping` for application probes (avoid `/healthz` on Cloud Run due to GFE interception).
- **Commands:** Always use `Makefile` targets (`make test`, `make vet`, `make lint`, `make docker-build`, `make db-up`, `make db-down`, `make proxy`, `make proxy-sandbox`, `make proxy-prod`).

## 2. Multimodal Indexing & Image/Video Design
- **Pub/Sub Trigger:** Consumes `media.confirmed` events from topic `projects/reeler-<env>/topics/media-events`.
- **Table Distinction:**
  - `public.media_semantics`: 1-to-1 with `media_id`. Stores summary, tags, detected season/location, and 768-dim embedding (`text-embedding-004`).
  - `public.media_scenes`: 1-to-many with `media_id`. Stores timestamped clip breakdown (`start_time_seconds`, `end_time_seconds`, `description`, `speech_transcript`, 768-dim `embedding`).
- **Image vs Video Flexibility:**
  - **Images (`image/*`):** Generates `media_semantics` metadata and embedding; **skips** `media_scenes` generation completely. Lightweight and fast for carousel/still-image pipelines.
  - **Videos (`video/*`):** Generates both `media_semantics` (global summary) and `media_scenes` (sub-clip breakdowns with exact cut timestamps).
- **Semantic Search:** `POST /api/v1/search/semantic` embeds queries with `text-embedding-004`, queries `pgvector` cosine distance (`<=>`), and returns ranked media + matching scene intervals. Supports filtering by `media_type`.

## 3. Issue-Driven Workflow
All work is tracked via **GitHub Issues** on `LuisGaravaso/media-indexer` linked to GitHub Project #3 (@LuisGaravaso's Reeler):
1. **/ticket-enhance `<id>`:** Audit and update ticket description and acceptance criteria against current codebase state before coding.
2. **/ticket-start `<id>`:** Branch off updated `main`, implement step-by-step, verify with `make`, and create PR.

## 4. Working Rules
- **Branch Isolation:** Never touch `main` directly. Always pull latest `main` and create `feat/issue-<number>-<slug>`.
- **Atomic Commits:** Keep commits focused and self-contained; link issues (`closes #<id>`).
- **CI Gate Requirement:** Never merge any PR if CI checks or integration tests fail. All GitHub Actions checks must be green before merging.
- **Testing Standard:** Every feature/endpoint/pipeline must have Testify tests (`assert`, `require`, `mock`) passing `make test` with race detection.
- **Code Quality:** Must pass `make vet` and `make lint` cleanly.
- **Database & Security:** Strict multi-tenant isolation adhering to Supabase Row-Level Security (RLS) (`auth.uid() = user_id`).
- **PgBouncer Standard:** Use `pgx.QueryExecModeDescribeExec` on all pooler connections (port 6543) and marshal JSONB parameters explicitly.
- **Deployments:** Triggered manually via GitHub Actions `deploy.yml` workflow (`sandbox` for testing from any branch/PR, `production` strictly from `main`).
