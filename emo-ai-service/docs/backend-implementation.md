# Emo AI Backend Implementation

This backend now covers the product pages: profile, mood diary, chat history,
emotion reports, privacy/security, system/about, and file metadata.

## Database

Development mode uses GORM `AutoMigrate` in `internal/data/data.go`.
You only need to create the database itself:

```sql
CREATE DATABASE emo_ai DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

Tables are created automatically when the service starts. For production,
prefer applying `database/schema.sql` through a migration tool and disabling
ad-hoc online schema changes in the deployment process.

## API Modules

Authentication:

- `POST /v1/users/register`
- `POST /v1/users/login`
- `POST /v1/auth/refresh`
- `POST /v1/auth/logout`

Profile:

- `GET /v1/users/me`
- `PATCH /v1/users/me/profile`
- `PUT /v1/users/me/avatar`

Mood diary:

- `POST /v1/diaries`
- `GET /v1/diaries`
- `GET /v1/diaries/{id}`
- `PATCH /v1/diaries/{id}`
- `DELETE /v1/diaries/{id}`
- `GET /v1/mood-tags`
- `POST /v1/mood-tags`
- `PATCH /v1/mood-tags/{id}`
- `DELETE /v1/mood-tags/{id}`

Chat:

- `POST /v1/chat/sessions`
- `GET /v1/chat/sessions`
- `GET /v1/chat/sessions/{id}`
- `PATCH /v1/chat/sessions/{id}`
- `DELETE /v1/chat/sessions/{id}`
- `GET /v1/chat/sessions/{session_id}/messages`
- `POST /v1/chat/sessions/{session_id}/messages`
- `POST /v1/chat/messages/{message_id}/feedback`
- `POST /v1/chat/sessions/{session_id}/summarize`

Emotion reports:

- `POST /v1/emotion/analyses`
- `GET /v1/emotion/analyses`
- `GET /v1/emotion/analyses/{id}`
- `GET /v1/emotion/reports/overview`
- `GET /v1/emotion/reports/trends`
- `GET /v1/emotion/reports/calendar`

Security:

- `POST /v1/security/password/change`
- `GET /v1/security/login-logs`
- `GET /v1/security/tokens`
- `DELETE /v1/security/tokens/{token_id}`
- `DELETE /v1/security/tokens`
- `GET /v1/security/events`

System and files:

- `GET /v1/system/about`
- `GET /v1/system/configs/public`
- `GET /v1/system/versions/latest`
- `GET /v1/system/announcements`
- `POST /v1/files/upload-token`
- `POST /v1/files`
- `GET /v1/files/{id}`
- `DELETE /v1/files/{id}`

## JWT

`configs/config.yaml` contains:

```yaml
auth:
  jwt_secret: please-change-this-secret-in-production
  access_token_ttl: 2h
  refresh_token_ttl: 720h
  issuer: emo-ai-service
```

Use `Authorization: Bearer <access_token>` for all private APIs. Public APIs
are login, register, refresh, logout, and public system config endpoints.
Refresh tokens are stored as SHA-256 hashes in `auth_refresh_tokens` and are
rotated on refresh.

## AI Extension Point

The current implementation uses local placeholder providers:

- `internal/data/ai_client.go`
- `internal/data/emotion.go`

Replace `NewAIClient` and `NewEmotionAnalyzer` implementations to call a real
model provider. The upper layers depend on `biz.AIClient` and
`biz.EmotionAnalyzer`, so the REST API and business flow stay stable.
