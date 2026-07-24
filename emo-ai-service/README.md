# Emo AI Service

情感 AI 对话服务，基于 [Kratos](https://go-kratos.dev/) 构建的 BFF（Backend For Frontend），提供 HTTP/gRPC 双协议、JWT 鉴权、AI 聊天流式转发、情绪分析、日记、文件管理等接口。

## 依赖服务

启动前请确保以下服务已运行并可访问：

| 服务 | 用途 | 必需 |
|------|------|------|
| **MySQL** (8.0+) | 主数据库，存储用户、聊天、日记等全部业务数据 | ✅ 是 |
| **Redis** (6.0+) | Token 缓存、会话管理 | ✅ 是 |
| **MinIO** (或兼容 S3 的对象存储) | 头像上传、知识库文件存储 | 否（不配则不启用文件上传） |
| **FastAPI AI Service** | 下游 AI 模型服务，提供聊天、知识库索引等能力 | 否（不配则聊天功能不可用） |

## 环境变量

全部环境变量及其默认值和用途。**粗体** 为必须配置的变量。

### 基础运行环境

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `EMO_ENV` | (空) | 设为 `production` 时会强制校验密钥，不满足则 **panic** |
| `EMO_SKIP_AUTOMIGRATE` | (空) | 设为 `true` 跳过启动时的数据库自动迁移 |

### 数据库

| 变量 | 默认值 (来自 config.yaml) | 说明 |
|------|---------------------------|------|
| **`EMO_DATABASE_SOURCE`** | `emo_app:root123456789@tcp(121.43.33.200:3306)/emo_ai?charset=utf8mb4&parseTime=True&loc=Local` | MySQL DSN 连接串 |

> ⚠️ **重要**：默认 DSN 指向了一台远程服务器。本地开发时**必须**通过环境变量覆盖为自己的数据库地址。

### Redis

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `EMO_REDIS_ADDR` | `127.0.0.1:6379` | Redis 地址 |
| `EMO_REDIS_PASSWORD` | (空) | Redis 密码 |

### 对象存储 (MinIO / S3)

文件上传功能依赖以下变量，全都不配时上传接口会返回错误而不是 panic：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `EMO_MINIO_ENDPOINT` | (空) | MinIO 服务地址，如 `127.0.0.1:9000` |
| `EMO_MINIO_PUBLIC_ENDPOINT` | 取 `EMO_MINIO_ENDPOINT` | 对外可访问的 MinIO 地址（浏览器端使用） |
| `EMO_MINIO_USE_SSL` | (空) | 设为 `true` 启用 HTTPS |
| `EMO_MINIO_ACCESS_KEY` | (空) | Access Key |
| `EMO_MINIO_SECRET_KEY` | (空) | Secret Key |
| `EMO_MINIO_REGION` | `us-east-1` | Region |
| `EMO_MINIO_BUCKET` | `emotion-avatars` | 头像存储桶 |
| `EMO_MINIO_KNOWLEDGE_BUCKET` | `emotion-knowledge` | 知识库存储桶 |

### AI 服务

| 变量 | 默认值 (来自 config.yaml) | 说明 |
|------|---------------------------|------|
| `EMO_AI_SERVICE_BASE_URL` | `http://192.168.31.155:8000` | 下游 FastAPI AI 服务地址 |
| `EMO_AI_SERVICE_SHARED_SECRET` | (空) | 内部 HMAC 签名的共享密钥，**production 下必填** |
| `EMO_AI_DAILY_TOKEN_LIMIT` | (空) | 每日 token 用量上限（不设则不限制） |
| `EMO_AI_DAILY_COST_MICROS_LIMIT` | (空) | 每日费用上限（微美元，不设则不限制） |

### 鉴权

| 变量 | 默认值 (来自 config.yaml) | 说明 |
|------|---------------------------|------|
| **`EMO_JWT_SECRET`** | `please-change-this-secret-in-production` | JWT 签名密钥，**production 下必须更换** |

### CORS

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `EMO_CORS_ALLOWED_ORIGINS` | (空) | 允许的跨域来源，英文逗号分隔，如 `http://localhost:5173,http://localhost:3000` |

---

## 准备数据库

启动前先在 MySQL 中创建数据库：

```sql
CREATE DATABASE emo_ai DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

表结构会在服务首次启动时自动迁移（除非设置了 `EMO_SKIP_AUTOMIGRATE=true`）。

---

## 本地运行

### 1. 确保 MySQL 和 Redis 已启动

```powershell
# 检查 MySQL
mysql -u root -p -e "SELECT 1"

# 检查 Redis
redis-cli ping
```

### 2. 设置环境变量

**PowerShell：**

```powershell
# 必配 — 覆盖为你本地的数据库连接
$env:EMO_DATABASE_SOURCE = "root:你的密码@tcp(127.0.0.1:3306)/emo_ai?charset=utf8mb4&parseTime=True&loc=Local"

# 可选 — 如果你本地有 AI 服务
$env:EMO_AI_SERVICE_BASE_URL = "http://127.0.0.1:8000"

# 可选 — 如果你本地有 MinIO（开发阶段可以不配，不影响启动）
# $env:EMO_MINIO_ENDPOINT = "127.0.0.1:9000"
# $env:EMO_MINIO_ACCESS_KEY = "minioadmin"
# $env:EMO_MINIO_SECRET_KEY = "minioadmin"

# 可选 — CORS（前端开发时需要）
$env:EMO_CORS_ALLOWED_ORIGINS = "http://localhost:5173"
```

**Linux / macOS (bash/zsh)：**

```bash
export EMO_DATABASE_SOURCE="root:你的密码@tcp(127.0.0.1:3306)/emo_ai?charset=utf8mb4&parseTime=True&loc=Local"
export EMO_CORS_ALLOWED_ORIGINS="http://localhost:5173"
```

### 3. 启动服务

```bash
go run ./cmd/emo-ai-service -conf ./configs
```

### 4. 验证

```bash
# HTTP 端口
curl http://localhost:8000/v1/system/about

# gRPC 端口
grpcurl -plaintext localhost:9000 list
```

默认端口（在 `configs/config.yaml` 中配置）：

| 协议 | 地址 |
|------|------|
| HTTP | `0.0.0.0:8000` |
| gRPC | `0.0.0.0:9000` |

---

## 常见启动 panic 及排查

| panic 信息 | 原因 | 解决 |
|-----------|------|------|
| `panic: dial tcp ... connect: ...` | MySQL 不可达 | 检查 `EMO_DATABASE_SOURCE` 地址是否正确，MySQL 是否已启动 |
| `panic: redis: ...` | Redis 不可达 | 检查 Redis 是否启动，`EMO_REDIS_ADDR` 和 `EMO_REDIS_PASSWORD` 是否正确 |
| `panic: EMO_JWT_SECRET must be configured in production` | `EMO_ENV=production` 但 JWT 密钥是默认值 | 设置 `EMO_JWT_SECRET` 为安全随机字符串 |
| `panic: EMO_AI_SERVICE_SHARED_SECRET must be configured in production` | `EMO_ENV=production` 但 AI 共享密钥未配置 | 设置 `EMO_AI_SERVICE_SHARED_SECRET` |
| `panic: ... no such file ... configs` | 运行目录不对 | 用 `-conf` 参数指定配置文件路径，或 `cd` 到项目根目录再运行 |

---

## 项目结构

```text
api/                    Protobuf 接口定义和生成的 Go 代码
cmd/                   应用入口
configs/               本地配置文件
internal/
  server/              HTTP / gRPC 服务端构建与路由注册
  service/             接口层（transport handler）
  biz/                 业务逻辑层（usecase / domain）
  data/                数据层（MySQL、Redis、MinIO、AI 客户端）
  auth/                JWT 鉴权、中间件
  conf/                配置结构体（protobuf 生成）
openapi.yaml           生成的 OpenAPI 文档
```

## 开发命令

```bash
make init      # 安装代码生成工具
make api       # 重新生成 API 绑定和 OpenAPI
make config    # 重新生成配置 protobuf
make all       # 全部生成 + Wire + module 整理
make build     # 编译
go test ./...  # 运行测试
```

## Docker

```bash
docker build -t emo-ai-service .
docker run --rm -p 8000:8000 -p 9000:9000 \
  -e EMO_DATABASE_SOURCE="root:password@tcp(host.docker.internal:3306)/emo_ai?charset=utf8mb4&parseTime=True&loc=Local" \
  -e EMO_REDIS_ADDR="host.docker.internal:6379" \
  -e EMO_JWT_SECRET="your-secret-here" \
  -v $(pwd)/configs:/data/conf \
  emo-ai-service
```
