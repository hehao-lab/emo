# Emo AI 前端接口对接文档

整理时间：2026-07-03

来源：

- HTTP 路由：`api/*/v1/*_http.pb.go`
- Proto 契约：`api/*/v1/*.proto`
- OpenAPI：`openapi.yaml`
- 鉴权中间件：`internal/server/http.go`、`internal/auth/auth.go`

## 1. 通用约定

### Base URL

本地默认 HTTP 端口：

```text
http://<后端机器IP>:8000
```

`configs/config.yaml` 中服务监听为 `0.0.0.0:8000`。

### 请求头

普通 JSON 接口：

```http
Content-Type: application/json
Accept: application/json
Authorization: Bearer <access_token>
```

登录时建议额外传设备信息，后端会写入登录日志：

```http
X-Device-ID: <设备ID>
X-Device-Name: <设备名称>
```

### 字段命名

自动生成的 HTTP 接口使用 `lowerCamelCase`：

- `user_id` -> `userId`
- `page_size` -> `pageSize`
- `refresh_token` -> `refreshToken`
- `avatar_url` -> `avatarUrl`

注意：`/api/v1/chat/stream` 是手写 SSE 接口，目前请求体更稳妥使用 `snake_case`，见“AIChat BFF”章节。

### int64 与时间

OpenAPI 中 `int64` 按 `string` 声明。前端建议把 ID、总数、时间戳都按字符串处理，避免 JS 精度问题。

时间戳字段通常是 Unix 秒级时间戳：

- `createdAt`
- `updatedAt`
- `expiresAt`
- `publishedAt`
- `startAt`
- `endAt`

日期字符串：

- `occurredOn`：`YYYY-MM-DD`
- `startDate` / `endDate`：`YYYY-MM-DD`
- `month`：建议 `YYYY-MM`

### 分页

分页参数统一：

```text
page=1
pageSize=20
```

后端默认：

- `page <= 0` 时按 `1`
- `pageSize <= 0` 时按 `20`
- `pageSize > 100` 时按 `100`

### 错误响应

普通 Kratos 接口错误大致为：

```json
{
  "code": 401,
  "reason": "UNAUTHORIZED",
  "message": "missing access token"
}
```

前端主要按 HTTP 状态码和 `reason` 处理。

## 2. 鉴权范围

公开接口，不需要 `Authorization`：

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/v1/users/register/email-code` | 发送注册邮箱验证码 |
| POST | `/v1/users/register` | 注册 |
| POST | `/v1/users/login` | 登录 |
| POST | `/v1/auth/refresh` | 刷新 token |
| POST | `/v1/auth/logout` | 退出登录，按 refreshToken 撤销 |
| GET | `/v1/system/about` | 关于我们 |
| GET | `/v1/system/configs/public` | 公开系统配置 |
| GET | `/v1/system/versions/latest` | 最新版本 |
| GET | `/v1/system/announcements` | 公告 |
| GET | `/api/v1/health` | AI 服务健康检查 |

其余接口都需要：

```http
Authorization: Bearer <access_token>
```

## 3. 用户与登录

### 接口列表

| 方法 | 路径 | 鉴权 | 请求 | 返回 | 说明 |
|---|---|---|---|---|---|
| POST | `/v1/users/register/email-code` | 否 | `SendRegisterEmailCodeRequest` | `SendRegisterEmailCodeResponse` | 发送注册邮箱验证码 |
| POST | `/v1/users/register` | 否 | `RegisterRequest` | `RegisterResponse` | 注册 |
| POST | `/v1/users/login` | 否 | `LoginRequest` | `LoginResponse` | 手机号密码登录 |
| GET | `/v1/users/info?userId=1` | 是 | query: `userId` | `GetUserInfoResponse` | 按用户 ID 获取用户信息 |

### 请求/响应字段

`SendRegisterEmailCodeRequest`

```json
{
  "email": "user@example.com"
}
```

`SendRegisterEmailCodeResponse`

```json
{
  "expiresIn": "300"
}
```

`RegisterRequest`

```json
{
  "username": "demo",
  "password": "123456",
  "phone": "13800000000",
  "email": "user@example.com",
  "verificationCode": "123456"
}
```

`RegisterResponse`

```json
{
  "userId": "1"
}
```

`LoginRequest`

```json
{
  "phone": "13800000000",
  "password": "123456"
}
```

`LoginResponse`

```json
{
  "accessToken": "jwt...",
  "refreshToken": "uuid.uuid",
  "expiresAt": "1751520000",
  "userId": "1",
  "username": "demo",
  "avatar": "",
  "roles": ["user"]
}
```

常见错误：

- `INVALID_ARGUMENT`
- `USERNAME_EXISTS`
- `PHONE_EXISTS`
- `EMAIL_EXISTS`
- `VERIFICATION_CODE_EXPIRED`
- `VERIFICATION_CODE_MISMATCH`
- `INVALID_CREDENTIALS`

## 4. 个人资料

### 接口列表

| 方法 | 路径 | 鉴权 | 请求 | 返回 | 说明 |
|---|---|---|---|---|---|
| GET | `/v1/users/me` | 是 | 无 | `UserProfile` | 当前登录用户资料 |
| PATCH | `/v1/users/me/profile` | 是 | `UpdateProfileRequest` | `UserProfile` | 修改资料 |
| PUT | `/v1/users/me/avatar` | 是 | `UpdateAvatarRequest` | `UserProfile` | 修改头像 |

### 字段

`UserProfile`

```json
{
  "userId": "1",
  "username": "demo",
  "phone": "13800000000",
  "email": "user@example.com",
  "nickname": "小明",
  "avatarUrl": "/uploads/avatar/a.png",
  "roles": ["user"],
  "gender": "male",
  "birthday": "2000-01-01",
  "bio": "个人简介",
  "location": "Shanghai",
  "occupation": "Engineer",
  "industry": "Internet",
  "language": "zh-CN",
  "timezone": "Asia/Shanghai",
  "createdAt": "1751520000",
  "updatedAt": "1751520000"
}
```

`UpdateProfileRequest`

```json
{
  "nickname": "小明",
  "email": "user@example.com",
  "gender": "male",
  "birthday": "2000-01-01",
  "bio": "个人简介",
  "location": "Shanghai",
  "occupation": "Engineer",
  "industry": "Internet",
  "language": "zh-CN",
  "timezone": "Asia/Shanghai"
}
```

`UpdateAvatarRequest`

```json
{
  "avatarUrl": "/uploads/avatar/a.png"
}
```

## 5. 安全与 Token

### 接口列表

| 方法 | 路径 | 鉴权 | 请求 | 返回 | 说明 |
|---|---|---|---|---|---|
| POST | `/v1/auth/refresh` | 否 | `RefreshTokenRequest` | `RefreshTokenResponse` | 刷新 token，旧 refreshToken 会轮换 |
| POST | `/v1/auth/logout` | 否 | `LogoutRequest` | 空 | 退出登录 |
| POST | `/v1/security/password/change` | 是 | `ChangePasswordRequest` | 空 | 修改密码并撤销全部 refreshToken |
| GET | `/v1/security/login-logs?page=1&pageSize=20` | 是 | query | `ListLoginLogsResponse` | 登录日志 |
| GET | `/v1/security/tokens` | 是 | 无 | `ListTokensResponse` | 登录设备/refreshToken 列表 |
| DELETE | `/v1/security/tokens/{token_id}` | 是 | path | 空 | 撤销单个设备 |
| DELETE | `/v1/security/tokens` | 是 | 无 | 空 | 撤销全部设备 |
| GET | `/v1/security/events?page=1&pageSize=20` | 是 | query | `ListSecurityEventsResponse` | 安全事件 |

### 请求/响应字段

`RefreshTokenRequest` / `LogoutRequest`

```json
{
  "refreshToken": "uuid.uuid"
}
```

`RefreshTokenResponse`

```json
{
  "accessToken": "jwt...",
  "refreshToken": "new.uuid",
  "expiresAt": "1751520000"
}
```

`ChangePasswordRequest`

```json
{
  "oldPassword": "old",
  "newPassword": "new"
}
```

`AuthToken`

```json
{
  "tokenId": "refresh-token-id",
  "deviceId": "device-id",
  "deviceName": "iPhone",
  "ip": "127.0.0.1",
  "userAgent": "Mozilla/5.0",
  "expiresAt": "1751520000",
  "revokedAt": "0",
  "createdAt": "1751520000"
}
```

## 6. 心情日记与标签

### 接口列表

| 方法 | 路径 | 鉴权 | 请求 | 返回 | 说明 |
|---|---|---|---|---|---|
| POST | `/v1/diaries` | 是 | `CreateDiaryRequest` | `MoodDiary` | 新增日记 |
| GET | `/v1/diaries?page=1&pageSize=20&tagId=1&mood=happy&startDate=2026-07-01&endDate=2026-07-31` | 是 | query | `ListDiariesResponse` | 日记列表 |
| GET | `/v1/diaries/{id}` | 是 | path | `MoodDiary` | 日记详情 |
| PATCH | `/v1/diaries/{id}` | 是 | `UpdateDiaryRequest` | `MoodDiary` | 修改日记 |
| DELETE | `/v1/diaries/{id}` | 是 | path | 空 | 删除日记 |
| GET | `/v1/mood-tags` | 是 | 无 | `ListTagsResponse` | 系统标签 + 当前用户自定义标签 |
| POST | `/v1/mood-tags` | 是 | `CreateTagRequest` | `MoodTag` | 创建自定义标签 |
| PATCH | `/v1/mood-tags/{id}` | 是 | `UpdateTagRequest` | `MoodTag` | 修改自定义标签 |
| DELETE | `/v1/mood-tags/{id}` | 是 | path | 空 | 删除自定义标签 |

### 请求/响应字段

`CreateDiaryRequest`

```json
{
  "title": "今天的心情",
  "content": "正文内容",
  "mood": "happy",
  "moodScore": 8,
  "weather": "sunny",
  "location": "Shanghai",
  "occurredOn": "2026-07-03",
  "visibility": "private",
  "tagIds": ["1", "2"],
  "attachmentUrls": ["/uploads/diary/a.png"]
}
```

`UpdateDiaryRequest` 请求体同上，可额外带 `id`；路径里的 `{id}` 仍是主 ID。

`MoodDiary`

```json
{
  "id": "1",
  "userId": "1",
  "title": "今天的心情",
  "content": "正文内容",
  "mood": "happy",
  "moodScore": 8,
  "weather": "sunny",
  "location": "Shanghai",
  "occurredOn": "2026-07-03",
  "visibility": "private",
  "tags": [],
  "attachmentUrls": [],
  "analysisId": "0",
  "createdAt": "1751520000",
  "updatedAt": "1751520000"
}
```

`CreateTagRequest`

```json
{
  "name": "开心",
  "color": "#22c55e",
  "icon": "smile",
  "sort": 10
}
```

`MoodTag`

```json
{
  "id": "1",
  "name": "开心",
  "color": "#22c55e",
  "icon": "smile",
  "sort": 10,
  "system": false,
  "createdAt": "1751520000",
  "updatedAt": "1751520000"
}
```

## 7. 本地聊天会话

这组接口使用后端本地 `chat_sessions` / `chat_messages` 表，适合普通咨询会话历史。

### 接口列表

| 方法 | 路径 | 鉴权 | 请求 | 返回 | 说明 |
|---|---|---|---|---|---|
| POST | `/v1/chat/sessions` | 是 | `CreateSessionRequest` | `ChatSession` | 创建会话 |
| GET | `/v1/chat/sessions?page=1&pageSize=20&status=active` | 是 | query | `ListSessionsResponse` | 会话列表 |
| GET | `/v1/chat/sessions/{id}` | 是 | path | `ChatSession` | 会话详情 |
| PATCH | `/v1/chat/sessions/{id}` | 是 | `UpdateSessionRequest` | `ChatSession` | 修改标题/状态 |
| DELETE | `/v1/chat/sessions/{id}` | 是 | path | 空 | 删除会话 |
| GET | `/v1/chat/sessions/{session_id}/messages?page=1&pageSize=20` | 是 | query | `ListMessagesResponse` | 消息列表 |
| POST | `/v1/chat/sessions/{session_id}/messages` | 是 | `SendMessageRequest` | `SendMessageResponse` | 发送消息并获取 AI 回复 |
| POST | `/v1/chat/messages/{message_id}/feedback` | 是 | `CreateFeedbackRequest` | `ChatFeedback` | 反馈 AI 回复 |
| POST | `/v1/chat/sessions/{session_id}/summarize` | 是 | path | `ChatContextSummary` | 生成会话摘要 |

### 字段

`CreateSessionRequest`

```json
{
  "title": "新的情感咨询",
  "scenario": "emotional_support"
}
```

`ChatSession`

```json
{
  "id": "1",
  "userId": "1",
  "title": "新的情感咨询",
  "scenario": "emotional_support",
  "status": "active",
  "summary": "",
  "messageCount": 0,
  "lastMessageAt": "0",
  "createdAt": "1751520000",
  "updatedAt": "1751520000"
}
```

`SendMessageRequest`

```json
{
  "content": "我今天有点焦虑",
  "contentType": "text"
}
```

`ChatMessage`

```json
{
  "id": "1",
  "sessionId": "1",
  "userId": "1",
  "role": "user",
  "content": "消息内容",
  "contentType": "text",
  "model": "",
  "promptTokens": 0,
  "completionTokens": 0,
  "totalTokens": 0,
  "latencyMs": 0,
  "emotionSnapshotJson": "{}",
  "safetyResultJson": "{}",
  "status": "success",
  "errorMessage": "",
  "createdAt": "1751520000"
}
```

`CreateFeedbackRequest`

```json
{
  "rating": 5,
  "feedbackType": "helpful",
  "content": "回复有帮助"
}
```

## 8. AIChat BFF / FastAPI 转发

这组接口由 Kratos 后端鉴权后转发到下游 FastAPI AI 服务。前端只需要传 JWT，不能也不需要传 `X-User-Id`。

### 接口列表

| 方法 | 路径 | 鉴权 | 请求 | 返回 | 说明 |
|---|---|---|---|---|---|
| GET | `/api/v1/health` | 否 | 无 | `HealthReply` | 下游 AI 服务健康检查 |
| POST | `/api/v1/conversations` | 是 | `CreateConversationRequest` | `Conversation` | 创建 AI 对话 |
| GET | `/api/v1/conversations` | 是 | 无 | `ConversationSet` | AI 对话列表 |
| GET | `/api/v1/conversations/{conversation_id}/messages` | 是 | path | `MessageSet` | AI 对话消息 |
| POST | `/api/v1/chat` | 是 | `ChatRequest` | `ChatReply` | 非流式聊天 |
| POST | `/api/v1/chat/stream` | 是 | `ChatRequest` | SSE | 流式聊天 |
| POST | `/api/v1/knowledge/documents` | 是 | `CreateKnowledgeDocumentRequest` | `KnowledgeDocument` | 创建知识库文档 |
| GET | `/api/v1/knowledge/documents` | 是 | 无 | `KnowledgeDocumentSet` | 知识库文档列表 |

### 普通非流式字段

普通生成接口使用 `lowerCamelCase`：

`ChatRequest`

```json
{
  "conversationId": "conversation-id",
  "message": "你好",
  "systemPrompt": "你是情绪陪伴助手"
}
```

`ChatReply`

```json
{
  "conversation": {
    "id": "conversation-id",
    "title": "会话标题",
    "createdAt": "2026-07-03T10:00:00Z",
    "updatedAt": "2026-07-03T10:00:00Z"
  },
  "userMessage": {
    "id": "msg-1",
    "conversationId": "conversation-id",
    "role": "user",
    "content": "你好",
    "sequence": 1,
    "modelName": "",
    "createdAt": "2026-07-03T10:00:00Z"
  },
  "assistantMessage": {
    "id": "msg-2",
    "conversationId": "conversation-id",
    "role": "assistant",
    "content": "你好，我在。",
    "sequence": 2,
    "modelName": "model",
    "createdAt": "2026-07-03T10:00:01Z"
  }
}
```

`CreateConversationRequest`

```json
{
  "title": "我的对话"
}
```

`CreateKnowledgeDocumentRequest`

```json
{
  "title": "知识标题",
  "content": "知识正文",
  "source": "manual"
}
```

### SSE 流式聊天

`POST /api/v1/chat/stream`

请求头：

```http
Authorization: Bearer <access_token>
Content-Type: application/json
Accept: text/event-stream
```

请求体建议使用 `snake_case`：

```json
{
  "conversation_id": "conversation-id",
  "message": "你好",
  "system_prompt": "你是情绪陪伴助手"
}
```

响应头：

```http
Content-Type: text/event-stream; charset=utf-8
Cache-Control: no-store
Connection: keep-alive
```

后端会原样透传下游 FastAPI SSE 帧。流开始前的错误是 JSON：

```json
{
  "code": 401,
  "reason": "UNAUTHORIZED",
  "message": "missing access token"
}
```

流开始后的中途错误：

```text
event: error
data: {"detail":"AI service stream interrupted"}
```

AIChat 校验：

- `message` 必填，最长 8000 字符
- `systemPrompt` / `system_prompt` 最长 4000 字符
- 创建会话 `title` 必填，最长 200 字符
- 知识库 `title` 必填最长 200 字符，`content` 必填最长 200000 字符，`source` 最长 200 字符

## 9. 情绪分析与报告

### 接口列表

| 方法 | 路径 | 鉴权 | 请求 | 返回 | 说明 |
|---|---|---|---|---|---|
| POST | `/v1/emotion/analyses` | 是 | `CreateAnalysisRequest` | `EmotionAnalysis` | 创建情绪分析 |
| GET | `/v1/emotion/analyses?page=1&pageSize=20&sourceType=diary&startDate=2026-07-01&endDate=2026-07-31` | 是 | query | `ListAnalysesResponse` | 分析历史 |
| GET | `/v1/emotion/analyses/{id}` | 是 | path | `EmotionAnalysis` | 分析详情 |
| GET | `/v1/emotion/reports/overview?range=7d` | 是 | query | `EmotionOverview` | 报告总览 |
| GET | `/v1/emotion/reports/trends?startDate=2026-07-01&endDate=2026-07-31` | 是 | query | `EmotionTrends` | 趋势 |
| GET | `/v1/emotion/reports/calendar?month=2026-07` | 是 | query | `EmotionCalendar` | 日历 |

### 字段

`CreateAnalysisRequest`

```json
{
  "sourceType": "diary",
  "sourceId": "1",
  "text": "需要分析的文本"
}
```

`EmotionAnalysis`

```json
{
  "id": "1",
  "userId": "1",
  "sourceType": "diary",
  "sourceId": "1",
  "primaryEmotion": "happy",
  "sentiment": "positive",
  "sentimentScore": 0.8,
  "stressScore": 20,
  "anxietyScore": 10,
  "depressionRiskScore": 5,
  "energyScore": 80,
  "confidence": 0.9,
  "summary": "分析摘要",
  "advice": "建议",
  "riskLevel": "low",
  "model": "local-v1",
  "dimensions": [
    {"dimension": "stress", "score": 0.2}
  ],
  "rawResultJson": "{}",
  "createdAt": "1751520000"
}
```

`range` 支持：

- `7d`，默认
- `30d`
- `90d`

## 10. 文件

### 接口列表

| 方法 | 路径 | 鉴权 | 请求 | 返回 | 说明 |
|---|---|---|---|---|---|
| POST | `/v1/files/upload-token` | 是 | `CreateUploadTokenRequest` | `UploadToken` | 生成上传凭证/本地上传地址 |
| POST | `/v1/files` | 是 | `CreateFileRequest` | `FileAsset` | 登记文件元数据 |
| GET | `/v1/files/{id}` | 是 | path | `FileAsset` | 文件详情 |
| DELETE | `/v1/files/{id}` | 是 | path | 空 | 删除文件记录 |

### 字段

`CreateUploadTokenRequest`

```json
{
  "bizType": "avatar",
  "filename": "a.png",
  "mimeType": "image/png",
  "sizeBytes": "102400"
}
```

`UploadToken`

```json
{
  "provider": "local",
  "uploadUrl": "/uploads/avatar/20260703/uuid-a.png",
  "objectKey": "avatar/20260703/uuid-a.png",
  "publicUrl": "/uploads/avatar/20260703/uuid-a.png",
  "expiresAt": "1751520000"
}
```

`CreateFileRequest`

```json
{
  "bizType": "avatar",
  "storageProvider": "local",
  "bucket": "",
  "objectKey": "avatar/20260703/uuid-a.png",
  "url": "/uploads/avatar/20260703/uuid-a.png",
  "mimeType": "image/png",
  "sizeBytes": "102400",
  "checksum": ""
}
```

`FileAsset`

```json
{
  "id": "1",
  "ownerUserId": "1",
  "bizType": "avatar",
  "storageProvider": "local",
  "bucket": "",
  "objectKey": "avatar/20260703/uuid-a.png",
  "url": "/uploads/avatar/20260703/uuid-a.png",
  "mimeType": "image/png",
  "sizeBytes": "102400",
  "checksum": "",
  "status": 1,
  "createdAt": "1751520000",
  "updatedAt": "1751520000"
}
```

## 11. 系统配置、版本、公告

这组是 uniapp 和管理平台最容易共用的数据读取接口。当前后端只暴露“前台读取”接口，未暴露后台管理写入接口。

### 接口列表

| 方法 | 路径 | 鉴权 | 请求 | 返回 | 说明 |
|---|---|---|---|---|---|
| GET | `/v1/system/about` | 否 | 无 | `AboutInfo` | 关于我们 |
| GET | `/v1/system/configs/public` | 否 | 无 | `ListPublicConfigsResponse` | 公开配置 |
| GET | `/v1/system/versions/latest?platform=android` | 否 | query: `platform` | `AppVersion` | 最新版本 |
| GET | `/v1/system/announcements?platform=android` | 否 | query: `platform` | `ListAnnouncementsResponse` | 当前有效公告 |

### 字段

`AboutInfo`

```json
{
  "appName": "Emo AI",
  "company": "Emo AI Team",
  "description": "应用介绍",
  "privacyUrl": "/privacy",
  "termsUrl": "/terms",
  "contactEmail": "support@example.com",
  "website": "https://example.com"
}
```

`PublicConfig`

```json
{
  "key": "home.banner",
  "valueJson": "{\"title\":\"xxx\"}",
  "description": "首页 banner 配置"
}
```

`AppVersion`

```json
{
  "id": "1",
  "platform": "android",
  "version": "1.0.0",
  "buildNo": 100,
  "forceUpdate": false,
  "downloadUrl": "https://example.com/app.apk",
  "changelog": "更新说明",
  "minSupportedVersion": "1.0.0",
  "publishedAt": "1751520000"
}
```

`Announcement`

```json
{
  "id": "1",
  "title": "公告标题",
  "content": "公告内容",
  "targetPlatform": "all",
  "startAt": "1751520000",
  "endAt": "1751606400"
}
```

`platform` 建议值：

- `ios`
- `android`
- `web`
- `all`，公告表中用于全平台

## 12. 管理端联动接口

你提到“管理平台改东西，uniapp 也会发生变化”。后端现在已经补充 `/v1/admin/*` 管理接口，管理端修改后，uniapp 继续通过这些公开读取接口拿最新数据：

- `system_configs` -> `/v1/system/configs/public`
- `app_versions` -> `/v1/system/versions/latest`
- `system_announcements` -> `/v1/system/announcements`
- `mood_tags` 中 `user_id = 0` 的系统标签 -> `/v1/mood-tags`

对应管理端接口见 [admin-frontend-api.md](D:/emo-ai/emo-ai-service/docs/admin-frontend-api.md)，核心联动接口：

| 方法 | 路径 | 说明 |
|---|---|---|
| GET/POST/PATCH/DELETE | `/v1/admin/system/configs` | 管理系统配置 |
| GET/POST/PATCH/DELETE | `/v1/admin/system/versions` | 管理 app 版本 |
| GET/POST/PATCH/DELETE | `/v1/admin/system/announcements` | 管理系统公告 |
| GET/POST/PATCH/DELETE | `/v1/admin/mood-tags` | 管理系统标签 |

## 13. 前端最小接入流程

1. 注册：`POST /v1/users/register/email-code` -> `POST /v1/users/register`
2. 登录：`POST /v1/users/login`，保存 `accessToken` 和 `refreshToken`
3. 私有接口统一加：`Authorization: Bearer <accessToken>`
4. accessToken 过期：`POST /v1/auth/refresh`，用新 token 覆盖旧 token
5. 首页/公共数据：调用 `/v1/system/configs/public`、`/v1/system/announcements`、`/v1/system/versions/latest`
6. 我的页面：`GET /v1/users/me`
7. 日记：`GET/POST /v1/diaries`，标签用 `GET /v1/mood-tags`
8. 聊天：普通历史聊天用 `/v1/chat/*`，FastAPI BFF 聊天用 `/api/v1/*`
