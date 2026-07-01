# 情感 AI 聊天助手后端项目介绍与开发步骤

## 1. 项目介绍

本项目是一个基于 Go + Kratos 的“情感 AI 聊天助手后端系统”。前端已有页面包括：

- 我的页面：用户资料、头像、个人信息修改。
- 心情日记：心情记录、日记增删改查、标签体系。
- 历史咨询：AI 聊天会话、聊天消息、咨询历史。
- 情感分析报告：情绪分析结果、趋势统计、日历视图。
- 隐私与安全：密码修改、登录日志、token 管理、设备下线。
- 关于我们/系统配置：应用信息、版本管理、公告、公开配置。

后端采用 Kratos 的分层结构，核心目标是：

- 接口清晰：所有 REST API 先由 proto 定义。
- 分层清楚：service、biz、data 各做各的事。
- 可扩展：AI 服务、情绪分析、文件存储都通过接口隔离。
- 可上线：JWT 鉴权、refresh token 存库、登录日志、安全事件、完整数据库表结构。

## 2. 技术栈

- 语言：Go
- 框架：Kratos
- API 定义：Protocol Buffers
- HTTP 路由：Kratos `protoc-gen-go-http`
- 数据库：MySQL
- ORM：GORM
- 依赖注入：Wire
- 鉴权：JWT access token + refresh token
- 文档：OpenAPI 自动生成

## 3. 当前后端模块

| 模块 | API 包 | service 文件 | biz 文件 | data 文件 |
|---|---|---|---|---|
| 用户登录注册 | `api/user/v1` | `internal/service/user.go` | `internal/biz/user.go` | `internal/data/user.go` |
| 用户资料 | `api/profile/v1` | `internal/service/profile.go` | `internal/biz/user.go` | `internal/data/user.go` |
| 心情日记 | `api/diary/v1` | `internal/service/diary.go` | `internal/biz/diary.go` | `internal/data/diary.go` |
| AI 聊天 | `api/chat/v1` | `internal/service/chat.go` | `internal/biz/chat.go` | `internal/data/chat.go` |
| 情绪分析 | `api/emotion/v1` | `internal/service/emotion.go` | `internal/biz/emotion.go` | `internal/data/emotion.go` |
| 隐私安全 | `api/security/v1` | `internal/service/security.go` | `internal/biz/security.go` | `internal/data/security.go` |
| 系统配置 | `api/system/v1` | `internal/service/system.go` | `internal/biz/system.go` | `internal/data/system.go` |
| 文件资源 | `api/file/v1` | `internal/service/file.go` | `internal/biz/file.go` | `internal/data/file.go` |
| JWT 鉴权 | - | `internal/server/http.go` | - | `internal/auth/auth.go` |

## 4. 项目分层说明

项目必须遵守下面的调用方向：

```text
前端
  |
  v
api proto DTO
  |
  v
internal/service  负责：参数校验、DTO 和 DO 转换、获取当前登录用户
  |
  v
internal/biz      负责：业务实体、业务规则、usecase、repo 接口
  |
  v
internal/data     负责：GORM 表模型、数据库查询、外部服务适配
  |
  v
MySQL / AI Provider / Object Storage
```

### service 层做什么

位置：`internal/service/*.go`

职责：

- 接收 proto 生成的 request。
- 从 JWT context 获取当前用户 ID。
- 做接口级参数校验。
- 把 DTO 转成 biz 层 DO。
- 调用 usecase。
- 把 DO 转成 proto response。

不要做：

- 不直接操作数据库。
- 不写复杂业务规则。
- 不导入 `internal/data`。

### biz 层做什么

位置：`internal/biz/*.go`

职责：

- 定义业务实体，例如 `MoodDiary`、`ChatSession`、`EmotionAnalysis`。
- 定义 usecase，例如 `DiaryUsecase`、`ChatUsecase`。
- 定义 repo 接口，例如 `DiaryRepo`、`ChatRepo`。
- 编排业务流程，例如发送消息时“保存用户消息 -> 调 AI -> 保存 AI 回复”。

不要做：

- 不写 GORM tag。
- 不直接依赖 MySQL、Redis、HTTP SDK。
- 不返回 proto DTO。

### data 层做什么

位置：`internal/data/*.go`

职责：

- 定义数据库 PO/GORM Model。
- 实现 biz 层 repo 接口。
- 做 DO 和 PO 转换。
- 封装数据库查询。
- 封装 AI provider、文件存储 provider。

不要做：

- 不导入 `internal/service`。
- 不接收 proto DTO。
- 不把 GORM Model 暴露给上层。

### server 层做什么

位置：`internal/server/*.go`

职责：

- 注册 HTTP/gRPC 服务。
- 配置 middleware。
- 配置 JWT 鉴权放行规则。

## 5. 数据库说明

开发环境不用手动建表，只需要先创建数据库：

```sql
CREATE DATABASE emo_ai DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

然后启动服务时，`internal/data/data.go` 会通过 GORM `AutoMigrate` 自动建表。

生产环境不要依赖服务自动改表，建议使用：

```text
database/schema.sql
```

把这份 SQL 放进正式 migration 流程。

## 6. 本地启动步骤

### 第一步：准备 MySQL 数据库

确认 MySQL 已启动，然后执行：

```sql
CREATE DATABASE emo_ai DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

检查配置文件：

```text
configs/config.yaml
```

重点看数据库连接：

```yaml
data:
  database:
    source: root:root123456@tcp(127.0.0.1:3306)/emo_ai?charset=utf8mb4&parseTime=True&loc=Local
```

### 第二步：确认 JWT 配置

配置在：

```text
configs/config.yaml
```

```yaml
auth:
  jwt_secret: please-change-this-secret-in-production
  access_token_ttl: 2h
  refresh_token_ttl: 720h
  issuer: emo-ai-service
```

开发环境可以先用默认值。上线前必须替换 `jwt_secret`。

### 第三步：安装生成工具

```bash
make init
```

如果工具已经安装过，可以跳过。

### 第四步：生成代码

```bash
make all
```

这个命令会生成：

- proto Go 代码
- HTTP handler 代码
- OpenAPI 文档
- Wire 依赖注入代码
- config proto 代码

### 第五步：运行测试

```bash
go test ./...
```

### 第六步：启动服务

```bash
go run ./cmd/emo-ai-service -conf ./configs
```

默认端口：

- HTTP：`8000`
- gRPC：`9000`

## 7. 前端调用顺序

### 第一步：注册

```http
POST /v1/users/register
```

### 第二步：登录

```http
POST /v1/users/login
```

登录成功后会返回：

- `access_token`
- `refresh_token`
- `expires_at`
- 用户基础信息

### 第三步：调用需要登录的接口

请求头带上：

```http
Authorization: Bearer <access_token>
```

例如：

```http
GET /v1/users/me
GET /v1/diaries
GET /v1/chat/sessions
GET /v1/emotion/reports/overview
```

### 第四步：access token 过期后刷新

```http
POST /v1/auth/refresh
```

刷新成功后，前端保存新的 access token 和 refresh token。

## 8. 新增一个功能的开发步骤

下面是以后开发任何新功能时的标准流程。一定按这个顺序来。

### 第一步：先定义 API，也就是改 proto

位置：

```text
api/<module>/v1/<module>.proto
```

例如要给心情日记增加“收藏日记”功能：

```proto
rpc FavoriteDiary(FavoriteDiaryRequest) returns (MoodDiary) {
  option (google.api.http) = {
    post: "/v1/diaries/{id}/favorite"
    body: "*"
  };
}
```

这一步要想清楚：

- 前端请求路径是什么。
- 请求参数有哪些。
- 返回什么数据。
- 是否需要登录。

### 第二步：生成 API 代码

```bash
make api
```

这一步会生成：

- `*.pb.go`
- `*_http.pb.go`
- `*_grpc.pb.go`
- `openapi.yaml`

不要手改这些生成文件。

### 第三步：写 service 层

位置：

```text
internal/service/<module>.go
```

service 层要做：

1. 实现 proto 生成的接口方法。
2. 从 context 里取当前用户 ID。
3. 校验请求参数。
4. 把 request DTO 转成 biz DO。
5. 调用 usecase。
6. 把 biz DO 转回 response DTO。

示例：

```go
// FavoriteDiary 实现收藏日记接口：把当前用户指定日记标记为收藏。
func (s *DiaryService) FavoriteDiary(ctx context.Context, req *v1.FavoriteDiaryRequest) (*v1.MoodDiary, error) {
    userID, err := currentUserID(ctx)
    if err != nil {
        return nil, err
    }
    diary, err := s.uc.FavoriteDiary(ctx, userID, req.GetId())
    if err != nil {
        return nil, err
    }
    return toDiaryDTO(diary), nil
}
```

### 第四步：写 biz 层

位置：

```text
internal/biz/<module>.go
```

biz 层要做：

1. 如果需要，给 DO 增加字段。
2. 在 repo interface 增加方法。
3. 在 usecase 增加业务方法。
4. 写业务规则。

示例：

```go
type DiaryRepo interface {
    FavoriteDiary(ctx context.Context, userID, diaryID int64) (*MoodDiary, error)
}

func (uc *DiaryUsecase) FavoriteDiary(ctx context.Context, userID, diaryID int64) (*MoodDiary, error) {
    return uc.repo.FavoriteDiary(ctx, userID, diaryID)
}
```

这一层不能出现 GORM、SQL、proto。

### 第五步：写 data 层

位置：

```text
internal/data/<module>.go
```

data 层要做：

1. 如果需要新字段，修改 GORM Model。
2. 如果需要新表，新增 Model。
3. 实现 biz repo interface。
4. 写数据库查询。
5. 做 PO 和 DO 转换。

示例：

```go
func (r *diaryRepoImpl) FavoriteDiary(ctx context.Context, userID, diaryID int64) (*biz.MoodDiary, error) {
    if err := r.db.WithContext(ctx).
        Model(&MoodDiaryModel{}).
        Where("user_id = ? AND id = ?", userID, diaryID).
        Update("favorite", true).Error; err != nil {
        return nil, err
    }
    return r.GetDiary(ctx, userID, diaryID)
}
```

### 第六步：如果新增了 service/repo/usecase，要注册 Wire

常见位置：

```text
internal/biz/wire.go
internal/data/data.go
internal/service/wire.go
cmd/emo-ai-service/wire.go
```

如果只是给已有模块加方法，通常不需要改 Wire。

如果新增了一个完整模块，例如 `notification`，就要注册：

- `NewNotificationRepo`
- `NewNotificationUsecase`
- `NewNotificationService`

### 第七步：如果新增了 HTTP 服务，要注册 server

位置：

```text
internal/server/http.go
```

新增：

```go
notificationv1.RegisterNotificationServiceHTTPServer(srv, notificationSvc)
```

如果这个接口不需要登录，还要加入 `publicOperations`。

### 第八步：如果改了配置，要改 conf.proto

位置：

```text
internal/conf/conf.proto
configs/config.yaml
```

改完后生成配置代码：

```bash
make config
```

### 第九步：如果改了 Wire，重新生成

```bash
go generate ./cmd/emo-ai-service
```

或者执行：

```bash
make all
```

### 第十步：格式化和测试

```bash
gofmt -w internal api cmd
go test ./...
```

测试通过后，再给前端联调。

## 9. 新增数据库表的开发步骤

### 第一步：先想清楚这个表属于哪个 data 文件

例如：

- 日记相关：`internal/data/diary.go`
- 聊天相关：`internal/data/chat.go`
- 情绪分析相关：`internal/data/emotion.go`
- 安全相关：`internal/data/security.go`

### 第二步：新增 GORM Model

```go
type ExampleModel struct {
    ID        int64          `gorm:"primaryKey;autoIncrement"`
    UserID    int64          `gorm:"index;not null"`
    CreatedAt time.Time      `gorm:"autoCreateTime"`
    UpdatedAt time.Time      `gorm:"autoUpdateTime"`
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (ExampleModel) TableName() string {
    return "examples"
}
```

### 第三步：注册 AutoMigrate

位置：

```text
internal/data/data.go
```

加入：

```go
&ExampleModel{},
```

### 第四步：同步生产 SQL

位置：

```text
database/schema.sql
```

加上对应 `CREATE TABLE`。

## 10. AI 聊天开发步骤

当前 AI 聊天流程：

```text
前端发送消息
  -> service.ChatService.SendMessage
  -> biz.ChatUsecase.SendMessage
  -> data 保存 user message
  -> data.AIClient 生成 AI 回复
  -> data 保存 assistant message
  -> 返回双方消息
```

如果要接真实大模型，优先改：

```text
internal/data/ai_client.go
```

实现：

```go
type AIClient interface {
    Reply(ctx context.Context, req AIReplyRequest) (*AIReply, error)
    Summarize(ctx context.Context, messages []*ChatMessage) (*AIReply, error)
}
```

不要在 service 层直接调大模型。

## 11. 情绪分析开发步骤

当前情绪分析流程：

```text
前端提交文本
  -> service.EmotionService.CreateAnalysis
  -> biz.EmotionUsecase.CreateAnalysis
  -> data.EmotionAnalyzer 分析文本
  -> data 保存 emotion_analyses
  -> data 保存 emotion_dimension_scores
  -> 返回分析结果
```

如果要接真实情绪分析模型，优先改：

```text
internal/data/emotion.go
```

实现：

```go
type EmotionAnalyzer interface {
    Analyze(ctx context.Context, userID int64, sourceType string, sourceID int64, text string) (*EmotionAnalysis, error)
}
```

## 12. 开发时最容易犯错的地方

### 不要在 service 层查数据库

错误：

```go
s.db.Where(...)
```

正确：

```go
s.uc.SomeAction(...)
```

### 不要在 biz 层写 GORM Model

错误：

```go
type Diary struct {
    ID int64 `gorm:"primaryKey"`
}
```

正确：

```go
type Diary struct {
    ID int64
}
```

### 不要手改生成文件

这些文件不要手改：

- `*.pb.go`
- `*_http.pb.go`
- `*_grpc.pb.go`
- `wire_gen.go`
- `openapi.yaml`

应该改源文件：

- `*.proto`
- `wire.go`
- `conf.proto`

然后重新生成。

## 13. 每次开发完成后的检查清单

开发完一个接口后，按顺序检查：

1. proto 是否定义清楚。
2. 是否执行过 `make api`。
3. service 是否只做 DTO 转换和参数校验。
4. biz 是否包含业务规则。
5. data 是否实现了 repo interface。
6. 新表是否加入 `AutoMigrate`。
7. 新表是否同步到 `database/schema.sql`。
8. 新 service 是否注册到 `internal/server/http.go`。
9. Wire 是否重新生成。
10. 是否通过 `go test ./...`。

## 14. 推荐开发顺序

如果你接下来继续完善项目，推荐顺序是：

1. 先跑通登录注册和 `/v1/users/me`。
2. 再联调心情日记 CRUD。
3. 再联调心情标签。
4. 再联调聊天会话和消息列表。
5. 再接真实 AI 回复。
6. 再接真实情绪分析模型。
7. 再完善报告统计口径。
8. 最后补管理后台、配置管理、运营公告。

这个顺序最稳，因为后面的聊天和报告都依赖前面的用户、日记、情绪数据。
