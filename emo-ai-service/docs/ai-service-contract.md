# AI Orchestration and Model API Contract

The project uses a third-party model API rather than a privately deployed
model. The Kratos service remains the public entry point, while the existing
FastAPI service is the project's own AI orchestration layer. The model provider
is called only from that orchestration layer with a server-side API key.

```text
Browser -> Kratos BFF -> project FastAPI/RAG service -> third-party model API
                                      |              -> embedding API
                                      +--------------> vector database
```

The third-party model API must never receive the application's user JWT,
roles, phone number, or internal user ID.

## Identity and authorization

- The BFF authenticates the user with its access JWT and derives `user_id` and
  `roles` only from verified claims.
- Between the Kratos BFF and the project's FastAPI service, the current BFF
  sends `X-Internal-User-Assertion` in the form
  `<user_id>.<unix_seconds>.<base64url(hmac_sha256)>`, signed with
  `EMO_AI_SERVICE_SHARED_SECRET`. FastAPI validates the HMAC with a constant-time
  comparison and rejects assertions older than 60 seconds. This assertion is
  never forwarded to the model provider.
- FastAPI calls the provider with a server-side provider API key. Provider API
  keys must not be stored in the frontend, returned by an API, or written to
  application logs.
- The model provider does not perform application authorization. The BFF and
  FastAPI enforce user ownership before retrieval or generation.
- `system_prompt` is an administrator-only capability. Ordinary chat requests
  use a server-side prompt template selected by scenario, policy version, and
  user entitlement.

## Stream contract

FastAPI exposes `POST /api/v1/chat/stream` to the BFF. It receives the verified
internal subject, selected conversation, message, and an `Idempotency-Key`,
then translates the provider's streaming format into these UTF-8 SSE frames:

```text
event: delta
data: {"content":"partial text"}

event: done
data: {
  "conversation_id":"upstream-conversation-id",
  "content":"final answer",
  "model_name":"model-id",
  "usage":{"prompt_tokens":123,"completion_tokens":45,"total_tokens":168,"cost_micros":3200},
  "references":[
    {
      "key":"K1",
      "document_id":"doc_123",
      "title":"Relationship communication guide",
      "source":"uploaded-file.pdf",
      "chunk_id":"chunk_9",
      "snippet":"The cited original passage...",
      "score":0.91
    }
  ]
}

event: error
data: {"code":"MODEL_UNAVAILABLE","detail":"safe user-facing message","retryable":true}
```

`references` are built by the project's RAG layer from the retrieval results;
they are not expected from the third-party model API. FastAPI maps the chunks
actually supplied to the model to stable keys such as `K1`, instructs the model
to use those keys, and returns the authoritative structured mapping. It must
omit raw document text the current user cannot read.

When a client disconnects, the BFF cancels FastAPI's request. FastAPI should
cancel the provider HTTP stream where supported. Providers may still bill
tokens already generated, so cancellation must be recorded with the actual
usage returned by the provider rather than assumed to be free.

## Knowledge base contract

The project FastAPI/RAG service needs tenant-scoped asynchronous document APIs.
These are application capabilities and are not model-provider endpoints:

- `POST /api/v1/knowledge/documents`: upload metadata and an object-storage
  reference; returns `{id,status:"queued"}` immediately.
- `GET /api/v1/knowledge/documents?page&page_size&status&query`: returns items,
  `next_cursor`, total where inexpensive, parse progress, error code/detail,
  chunk count, timestamps, and current index version.
- `GET /api/v1/knowledge/documents/{id}`: returns metadata plus authorized
  original-text preview/chunk lookup data.
- `PATCH /api/v1/knowledge/documents/{id}`: title/source/metadata update.
- `DELETE /api/v1/knowledge/documents/{id}`: soft delete followed by vector and
  object cleanup. It must be idempotent.
- `POST /api/v1/knowledge/documents/{id}:reindex`: creates a new ingestion job
  and returns its job ID.
- `GET /api/v1/knowledge/jobs/{id}`: returns queued/parsing/chunking/embedding/
  indexing/ready/failed status, progress 0-100, and a safe failure reason.

Vectors must be partitioned by tenant/user and document ID, have index version
metadata, and support hard deletion. The embedding model name and dimension
must be retained with each index version so reindexing is reversible.

## Usage, quotas, and operations

For every finished or cancelled turn FastAPI must return the provider model ID,
provider request ID, token usage, cached-token usage where available, latency,
and retriever metrics. The BFF computes cost from a versioned local pricing
table unless the provider returns an authoritative billed amount. These values
support per-user daily token/cost limits and usage reporting.

FastAPI should expose `/health/live`, `/health/ready`, and Prometheus metrics.
Provider readiness should be checked with a cheap, rate-limited probe rather
than a billable chat request on every health check. Required metric
dimensions include outcome, model, provider, endpoint, tenant plan, and status
class; do not use user ID as a metric label. Propagate W3C `traceparent` across
the BFF, retriever, vector store, and model provider. Alert on readiness loss,
queue age, indexing failures, elevated stream errors, quota rejects, and cost
anomalies. Backup both metadata and vector collections, then perform scheduled
restore-and-query verification against a non-production environment.

## What the third-party model API must provide

- Server-side API-key authentication and documented timeout/rate-limit errors.
- Streaming text deltas, a final finish reason, provider request ID, and token
  usage. If streamed usage is optional, it must be explicitly enabled.
- A stable model identifier and published context-window/output limits.
- An embeddings endpoint if the project does not choose a separate embedding
  provider. The embedding model ID and vector dimension must be stable and
  stored with each index version.
- Preferably structured output or tool calling for emotion/safety analysis;
  the application must still validate every returned structure.
- Provider data-retention and training controls suitable for the sensitivity
  of emotional-support conversations.

The provider does not need to implement users, permissions, `[K1]` mappings,
document CRUD, vector storage, ingestion progress, application quotas, CORS,
or backups. Those remain project responsibilities.

## BFF deployment settings

The BFF exposes structured references and usage on `chat.v1.ChatMessage` and
also retains their JSON forms for existing stored history. Streaming requests
forward `Idempotency-Key` and `traceparent`, and terminal stream events persist
model, provider request, usage, cost, and turn-status fields.

Configure these environment variables before deployment:

```text
EMO_AI_SERVICE_SHARED_SECRET=<same value configured on AI Service>
EMO_AI_DAILY_TOKEN_LIMIT=<per-user UTC-day token limit; 0 disables>
EMO_AI_DAILY_COST_MICROS_LIMIT=<per-user UTC-day cost limit; 0 disables>
EMO_MINIO_KNOWLEDGE_BUCKET=emotion-knowledge
```

The knowledge bucket is private. `POST /v1/files/knowledge` returns an opaque
`s3://bucket/key` reference, which is then sent to
`POST /api/v1/knowledge/documents`. The AI Service object-storage adapter must
be configured with access to the same private bucket.
