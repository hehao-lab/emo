# Emo AI 管理后台前端接口文档

整理时间：2026-07-03

## 重要说明

当前后端已经补充 `/v1/admin/*` 管理后台接口，管理端接口统一要求：

- 请求头带 `Authorization: Bearer <accessToken>`
- 登录用户 JWT 的 `roles` 必须包含 `admin`
- 无 token 返回 `401`
- 非管理员返回 `403`

## 通用约定

Base URL：

```text
http://<后端机器IP>:8000
```

请求头：

```http
Content-Type: application/json
Accept: application/json
Authorization: Bearer <accessToken>
```

字段命名使用 `lowerCamelCase`，例如：

- `userId`
- `pageSize`
- `refreshToken`
- `createdAt`

`int64` 类型建议前端按字符串处理，避免 JS 精度问题。

分页参数：

```text
page=1&pageSize=20
```

统一错误格式大致为：

```json
{
  "code": 401,
  "reason": "UNAUTHORIZED",
  "message": "missing access token"
}
```

## 1. 管理后台登录接口

### 1.1 登录

```http
POST /v1/users/login
```

说明：当前登录接口会返回 `roles`。管理后台账号需要 `roles` 里有 `admin`，否则访问 `/v1/admin/*` 会返回 `403`。

请求：

```json
{
  "phone": "13800000000",
  "password": "123456"
}
```

响应：

```json
{
  "accessToken": "jwt...",
  "refreshToken": "uuid.uuid",
  "expiresAt": "1751520000",
  "userId": "1",
  "username": "admin",
  "avatar": "",
  "roles": ["admin"]
}
```

### 1.2 刷新 Token

```http
POST /v1/auth/refresh
```

请求：

```json
{
  "refreshToken": "uuid.uuid"
}
```

响应：

```json
{
  "accessToken": "jwt...",
  "refreshToken": "new.uuid",
  "expiresAt": "1751520000"
}
```

### 1.3 退出登录

```http
POST /v1/auth/logout
```

请求：

```json
{
  "refreshToken": "uuid.uuid"
}
```

响应：空对象或空响应。

### 1.4 当前登录用户信息

```http
GET /v1/users/me
```

鉴权：需要 `Authorization`

响应：

```json
{
  "userId": "1",
  "username": "admin",
  "phone": "13800000000",
  "email": "admin@example.com",
  "nickname": "管理员",
  "avatarUrl": "",
  "roles": ["admin"],
  "createdAt": "1751520000",
  "updatedAt": "1751520000"
}
```

## 2. 管理后台核心接口

下面接口已经由后端实现，路径前缀统一为 `/v1/admin`。

## 3. 管理后台概览

### 3.1 统计概览

```http
GET /v1/admin/dashboard/overview
```

响应：

```json
{
  "userCount": "1024",
  "todayNewUsers": "12",
  "diaryCount": "5300",
  "todayDiaries": "98",
  "chatSessionCount": "1200",
  "todayChatMessages": "450",
  "emotionAnalysisCount": "3200",
  "highRiskAnalysisCount": "8"
}
```

### 3.2 最近趋势

```http
GET /v1/admin/dashboard/trends?startDate=2026-07-01&endDate=2026-07-31
```

响应：

```json
{
  "points": [
    {
      "date": "2026-07-03",
      "newUsers": "10",
      "diaries": "88",
      "chatMessages": "320",
      "emotionAnalyses": "75"
    }
  ]
}
```

## 4. 用户管理

### 4.1 用户列表

```http
GET /v1/admin/users?page=1&pageSize=20&keyword=张三&status=1&role=user
```

查询参数：

| 参数 | 类型 | 说明 |
|---|---|---|
| page | number | 页码 |
| pageSize | number | 每页数量 |
| keyword | string | 用户名/手机号/邮箱搜索 |
| status | number | 1 正常，2 冻结，3 注销 |
| role | string | 角色筛选，例如 `user`、`admin` |

响应：

```json
{
  "users": [
    {
      "userId": "1",
      "username": "demo",
      "phone": "13800000000",
      "email": "demo@example.com",
      "avatar": "",
      "roles": ["user"],
      "status": 1,
      "lastLoginAt": "1751520000",
      "createdAt": "1751520000",
      "updatedAt": "1751520000"
    }
  ],
  "total": "1"
}
```

### 4.2 用户详情

```http
GET /v1/admin/users/{user_id}
```

响应：

```json
{
  "userId": "1",
  "username": "demo",
  "phone": "13800000000",
  "email": "demo@example.com",
  "avatar": "",
  "roles": ["user"],
  "status": 1,
  "profile": {
    "nickname": "昵称",
    "gender": "male",
    "birthday": "2000-01-01",
    "bio": "简介",
    "location": "Shanghai",
    "occupation": "Engineer",
    "industry": "Internet",
    "language": "zh-CN",
    "timezone": "Asia/Shanghai"
  },
  "createdAt": "1751520000",
  "updatedAt": "1751520000"
}
```

### 4.3 修改用户状态

```http
PATCH /v1/admin/users/{user_id}/status
```

请求：

```json
{
  "status": 2,
  "reason": "违规冻结"
}
```

响应：

```json
{
  "userId": "1",
  "status": 2
}
```

### 4.4 修改用户角色

```http
PATCH /v1/admin/users/{user_id}/roles
```

请求：

```json
{
  "roles": ["user", "admin"]
}
```

响应：

```json
{
  "userId": "1",
  "roles": ["user", "admin"]
}
```

## 5. 系统配置管理

对应数据表：`system_configs`

### 5.1 配置列表

```http
GET /v1/admin/system/configs?page=1&pageSize=20&keyword=home&isPublic=true
```

响应：

```json
{
  "configs": [
    {
      "id": "1",
      "key": "home.banner",
      "valueJson": "{\"title\":\"首页标题\"}",
      "description": "首页 banner 配置",
      "isPublic": true,
      "createdAt": "1751520000",
      "updatedAt": "1751520000"
    }
  ],
  "total": "1"
}
```

### 5.2 新增配置

```http
POST /v1/admin/system/configs
```

请求：

```json
{
  "key": "home.banner",
  "valueJson": "{\"title\":\"首页标题\"}",
  "description": "首页 banner 配置",
  "isPublic": true
}
```

响应：返回配置详情。

### 5.3 修改配置

```http
PATCH /v1/admin/system/configs/{id}
```

请求：

```json
{
  "valueJson": "{\"title\":\"新的首页标题\"}",
  "description": "首页 banner 配置",
  "isPublic": true
}
```

响应：返回配置详情。

### 5.4 删除配置

```http
DELETE /v1/admin/system/configs/{id}
```

响应：空。

## 6. 公告管理

对应数据表：`system_announcements`

### 6.1 公告列表

```http
GET /v1/admin/system/announcements?page=1&pageSize=20&platform=android&status=1
```

响应：

```json
{
  "announcements": [
    {
      "id": "1",
      "title": "维护公告",
      "content": "今晚系统维护",
      "targetPlatform": "all",
      "startAt": "1751520000",
      "endAt": "1751606400",
      "status": 1,
      "createdAt": "1751520000",
      "updatedAt": "1751520000"
    }
  ],
  "total": "1"
}
```

### 6.2 新增公告

```http
POST /v1/admin/system/announcements
```

请求：

```json
{
  "title": "维护公告",
  "content": "今晚系统维护",
  "targetPlatform": "all",
  "startAt": "1751520000",
  "endAt": "1751606400",
  "status": 1
}
```

响应：返回公告详情。

### 6.3 修改公告

```http
PATCH /v1/admin/system/announcements/{id}
```

请求：

```json
{
  "title": "维护公告",
  "content": "维护时间调整",
  "targetPlatform": "android",
  "startAt": "1751520000",
  "endAt": "1751606400",
  "status": 1
}
```

响应：返回公告详情。

### 6.4 删除公告

```http
DELETE /v1/admin/system/announcements/{id}
```

响应：空。

## 7. App 版本管理

对应数据表：`app_versions`

### 7.1 版本列表

```http
GET /v1/admin/system/versions?page=1&pageSize=20&platform=android
```

响应：

```json
{
  "versions": [
    {
      "id": "1",
      "platform": "android",
      "version": "1.0.0",
      "buildNo": 100,
      "forceUpdate": false,
      "downloadUrl": "https://example.com/app.apk",
      "changelog": "更新说明",
      "minSupportedVersion": "1.0.0",
      "publishedAt": "1751520000",
      "createdAt": "1751520000",
      "updatedAt": "1751520000"
    }
  ],
  "total": "1"
}
```

### 7.2 新增版本

```http
POST /v1/admin/system/versions
```

请求：

```json
{
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

响应：返回版本详情。

### 7.3 修改版本

```http
PATCH /v1/admin/system/versions/{id}
```

请求：同新增版本，可只传修改字段。

响应：返回版本详情。

### 7.4 删除版本

```http
DELETE /v1/admin/system/versions/{id}
```

响应：空。

## 8. 系统心情标签管理

对应数据表：`mood_tags`

说明：用户自定义标签 `userId > 0`，系统标签 `userId = 0`。

### 8.1 系统标签列表

```http
GET /v1/admin/mood-tags?page=1&pageSize=20&keyword=开心
```

响应：

```json
{
  "tags": [
    {
      "id": "1",
      "name": "开心",
      "color": "#22c55e",
      "icon": "smile",
      "sort": 10,
      "system": true,
      "createdAt": "1751520000",
      "updatedAt": "1751520000"
    }
  ],
  "total": "1"
}
```

### 8.2 新增系统标签

```http
POST /v1/admin/mood-tags
```

请求：

```json
{
  "name": "开心",
  "color": "#22c55e",
  "icon": "smile",
  "sort": 10
}
```

响应：返回标签详情。

### 8.3 修改系统标签

```http
PATCH /v1/admin/mood-tags/{id}
```

请求：

```json
{
  "name": "开心",
  "color": "#22c55e",
  "icon": "smile",
  "sort": 10
}
```

响应：返回标签详情。

### 8.4 删除系统标签

```http
DELETE /v1/admin/mood-tags/{id}
```

响应：空。

## 9. 日记内容管理

当前用户端日记接口只允许看当前登录用户自己的日记，管理后台需要全量查询接口。

### 9.1 日记列表

```http
GET /v1/admin/diaries?page=1&pageSize=20&keyword=焦虑&userId=1&mood=happy&startDate=2026-07-01&endDate=2026-07-31
```

响应：

```json
{
  "diaries": [
    {
      "id": "1",
      "userId": "1",
      "username": "demo",
      "title": "今天的心情",
      "content": "正文",
      "mood": "happy",
      "moodScore": 8,
      "occurredOn": "2026-07-03",
      "visibility": "private",
      "createdAt": "1751520000",
      "updatedAt": "1751520000"
    }
  ],
  "total": "1"
}
```

### 9.2 日记详情

```http
GET /v1/admin/diaries/{id}
```

响应：返回日记详情。

### 9.3 删除日记

```http
DELETE /v1/admin/diaries/{id}
```

响应：空。

## 10. 情绪分析管理

### 10.1 分析列表

```http
GET /v1/admin/emotion/analyses?page=1&pageSize=20&userId=1&riskLevel=high&sourceType=diary
```

响应：

```json
{
  "analyses": [
    {
      "id": "1",
      "userId": "1",
      "username": "demo",
      "sourceType": "diary",
      "sourceId": "1",
      "primaryEmotion": "anxiety",
      "sentiment": "negative",
      "sentimentScore": -0.6,
      "stressScore": 80,
      "anxietyScore": 75,
      "depressionRiskScore": 40,
      "energyScore": 20,
      "confidence": 0.9,
      "summary": "分析摘要",
      "advice": "建议",
      "riskLevel": "high",
      "model": "local-v1",
      "createdAt": "1751520000"
    }
  ],
  "total": "1"
}
```

### 10.2 分析详情

```http
GET /v1/admin/emotion/analyses/{id}
```

响应：返回分析详情。

## 11. 聊天内容管理

### 11.1 会话列表

```http
GET /v1/admin/chat/sessions?page=1&pageSize=20&userId=1&status=active
```

响应：

```json
{
  "sessions": [
    {
      "id": "1",
      "userId": "1",
      "username": "demo",
      "title": "咨询会话",
      "scenario": "emotional_support",
      "status": "active",
      "messageCount": 12,
      "lastMessageAt": "1751520000",
      "createdAt": "1751520000",
      "updatedAt": "1751520000"
    }
  ],
  "total": "1"
}
```

### 11.2 会话消息

```http
GET /v1/admin/chat/sessions/{session_id}/messages?page=1&pageSize=50
```

响应：

```json
{
  "messages": [
    {
      "id": "1",
      "sessionId": "1",
      "userId": "1",
      "role": "user",
      "content": "消息内容",
      "contentType": "text",
      "status": "success",
      "createdAt": "1751520000"
    }
  ],
  "total": "1"
}
```

## 12. 文件管理

### 12.1 文件列表

```http
GET /v1/admin/files?page=1&pageSize=20&bizType=avatar&ownerUserId=1
```

响应：

```json
{
  "files": [
    {
      "id": "1",
      "ownerUserId": "1",
      "bizType": "avatar",
      "storageProvider": "local",
      "objectKey": "avatar/20260703/a.png",
      "url": "/uploads/avatar/20260703/a.png",
      "mimeType": "image/png",
      "sizeBytes": "102400",
      "status": 1,
      "createdAt": "1751520000",
      "updatedAt": "1751520000"
    }
  ],
  "total": "1"
}
```

### 12.2 文件详情

```http
GET /v1/admin/files/{id}
```

响应：返回文件详情。

### 12.3 删除文件记录

```http
DELETE /v1/admin/files/{id}
```

响应：空。

## 13. 前端联调优先级

建议管理后台页面按这个顺序对接：

1. 登录、刷新 token、当前用户信息
2. 用户管理
3. 系统配置管理
4. 公告管理
5. 版本管理
6. 系统标签管理
7. 日记、情绪分析、聊天内容查看
8. 文件管理
