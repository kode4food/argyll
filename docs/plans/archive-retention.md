# Flow Archival & Retention

Goal: prevent unbounded Valkey growth while preserving full event history and keeping the UI/API transparent, with the simplest viable flow.

## Objectives
- Keep flow event streams as the source of truth (no compaction/pruning in-place).
- Move terminal flows (completed/failed) older than a retention window to cold storage (S3-compatible).
- Let the API/GUI fetch archived flows seamlessly via a fallback.
- Be idempotent and safe: never delete Valkey data unless archive upload is verified.

## Scope / Non-Goals
- Scope: archive terminal flows; read fallback.
- Non-goals: in-place log compaction, partial event deletion, speculative flow statuses, manifests, restore tooling (unless later needed).

## Architecture
1) **Retention window**: configurable (e.g., `FLOW_RETENTION_DAYS`). Eligible = terminal + `last_updated` older than window.
2) **Archiver job** (background or cron):
   - Scan eligible flows (batch + cursor).
   - Stream their event log and final snapshot.
   - Write one gzipped JSONL file to S3; verify checksum/size.
   - Record index entry (`archived:flows[flow_id] = s3://...`).
   - Delete only flow-specific keys from the workflow store (event log, projections, active-flow index). Leave engine-wide indexes intact.
3) **Archive format** (simple):
   - Object key: `flows/{flow_id}.jsonl.gz` (or add date prefix later if needed).
   - Header line: metadata (flow_id, version, checksum).
   - Event lines: ordered event stream.
   - Trailer line: compact snapshot (plan, final FlowState).
4) **Lookup index**:
   - Redis hash `archived:flows` mapping `flow_id -> s3://bucket/key`.
5) **API fallback**:
   - `GET /engine/flow/{flowId}`: check live store; if missing, consult `archived:flows`; if present, fetch from S3, hydrate to `FlowState`, return. Optional `archived: true` flag for UI badging.
6) **Observability & safety**:
   - Metrics: archived count/bytes/failures; log per flow.
   - Never delete Redis data unless upload/verify succeeded.

## Data Removal (post-archive)
- Delete per-flow workflow-store keys only:
  - Event log for the flow.
  - Materialized projections/state for the flow.
  - Active/terminal flow index entries referencing the flow.
- Keep engine/global indexes and non-workflow stores untouched.

## API / Contract Changes
- Optional: add `archived: true` (or `source: "archived"`) in flow responses so the UI can badge archived flows; otherwise fully transparent.

## Configuration
- `FLOW_RETENTION_DAYS` (or ms): age threshold for archiving terminal flows.
- `ARCHIVE_BUCKET`, `ARCHIVE_PREFIX`: S3 destination.
- `ARCHIVE_CONCURRENCY`, `ARCHIVE_BATCH_SIZE`: throughput tuning.

## CLI / Ops Hooks (optional)
- `archive sweep`: run archiver once (cron-friendly).
- `archive flow <flow_id>`: manual archive.

## Risks & Mitigations
- **Late events after terminal**: enforce “terminal + idle > retention” before archiving.
- **Partial uploads**: verify checksum/size before deletion; uploads idempotent by key.
- **S3 latency**: acceptable; can add short cache later if needed.
- **Index drift**: write `archived:flows` only after verified upload; keep updates idempotent.

## Open Questions
- Exact Redis key names to delete per flow (align with event-store schema).
- Is an `archived` flag in responses desired by the UI?
