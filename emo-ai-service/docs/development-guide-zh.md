# 情感 AI 后端项目说明和开发步骤（简单版）

这份文档是给刚接手项目的人看的。不要先纠结所有代码，先按步骤走。

## 一、这个项目是做什么的

这是一个“情感 AI 聊天助手”的后端项目。

前端已经有这些页面：

1. 我的页面
2. 心情日记
3. 历史咨询
4. 情感分析报告
5. 隐私与安全
6. 关于我们

后端要负责：

1. 用户注册、登录、修改资料、上传头像
2. 保存心情日记
3. 保存 AI 聊天记录
4. 保存情绪分析结果
5. 查询情绪趋势和报告
6. 修改密码、查看登录日志、管理登录设备
7. 返回关于我们、版本信息、系统公告

## 二、你先记住项目分成 4 层

开发时只要记住这 4 层就行：

```text
api      定义接口
service  接收前端请求
biz      写业务逻辑
data     操作数据库
```

调用顺序是固定的：

```text
前端
  -> api
  -> service
  -> biz
  -> data
  -> 数据库
```

不要反着写。

## 三、每一层是干什么的

### 1. api 层

位置：

```text
api/
```

这里写接口定义。

比如：

```text
api/diary/v1/diary.proto
```

这个文件定义“心情日记”有哪些接口。

例如：

```text
POST /v1/diaries
GET  /v1/diaries
GET  /v1/diaries/{id}
```

简单理解：

api 层就是告诉前端：

1. 请求地址是什么
2. 请求参数是什么
3. 返回数据是什么

### 2. service 层

位置：

```text
internal/service/
```

这里接收前端请求。

service 层主要做：

1. 判断用户有没有登录
2. 检查参数有没有传
3. 把前端参数转换成业务对象
4. 调用 biz 层
5. 把结果返回给前端

例如：

```text
internal/service/diary.go
```

这里就是心情日记接口的入口。

### 3. biz 层

位置：

```text
internal/biz/
```

这里写业务逻辑。

biz 层主要做：

1. 定义业务对象
2. 写业务规则
3. 定义数据库接口
4. 调用 data 层

例如：

```text
internal/biz/diary.go
```

这里写心情日记的业务逻辑。

### 4. data 层

位置：

```text
internal/data/
```

这里操作数据库。

data 层主要做：

1. 定义数据库表结构
2. 写查询数据库的代码
3. 新增、修改、删除、查询数据

例如：

```text
internal/data/diary.go
```

这里真正操作 `mood_diaries` 表。

## 四、第一次运行项目怎么做

### 第一步：先建数据库

你只需要建库，不用一张张建表。

执行：

```sql
CREATE DATABASE emo_ai DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

表会在项目启动时自动创建。

表结构也放在这里：

```text
database/schema.sql
```

### 第二步：检查数据库配置

打开：

```text
configs/config.yaml
```

看这一段：

```yaml
data:
  database:
    source: root:root123456@tcp(127.0.0.1:3306)/emo_ai?charset=utf8mb4&parseTime=True&loc=Local
```

如果你的 MySQL 用户名、密码不是这个，就改这里。

### 第三步：生成代码

执行：

```bash
make all
```

这个命令会根据 proto 自动生成接口代码。

### 第四步：运行测试

执行：

```bash
go test ./...
```

看到没有报错，就说明代码能编译。

### 第五步：启动后端

执行：

```bash
go run ./cmd/emo-ai-service -conf ./configs
```

默认 HTTP 地址：

```text
http://localhost:8000
```

## 五、前端怎么调用

### 第一步：注册

```http
POST /v1/users/register
```

### 第二步：登录

```http
POST /v1/users/login
```

登录成功后会返回：

```text
access_token
refresh_token
expires_at
```

### 第三步：调用其他接口

其他需要登录的接口都要带请求头：

```http
Authorization: Bearer access_token
```

例如：

```http
GET /v1/users/me
GET /v1/diaries
GET /v1/chat/sessions
GET /v1/emotion/reports/overview
```

## 六、开发一个新接口应该怎么做

以后你要加新功能，就按这个顺序来。

不要一上来就写数据库。

正确顺序是：

```text
第一步：改 api
第二步：改 service
第三步：改 biz
第四步：改 data
第五步：注册接口
第六步：生成代码
第七步：测试
```

下面举例。

假设你要加一个功能：

```text
收藏心情日记
```

### 第一步：改 api 层

先打开：

```text
api/diary/v1/diary.proto
```

加一个接口：

```proto
rpc FavoriteDiary(FavoriteDiaryRequest) returns (MoodDiary) {
  option (google.api.http) = {
    post: "/v1/diaries/{id}/favorite"
    body: "*"
  };
}
```

这一步是在告诉前端：

```text
收藏日记的接口地址是 POST /v1/diaries/{id}/favorite
```

### 第二步：生成接口代码

执行：

```bash
make api
```

生成之后，Go 代码里才会出现这个新接口。

### 第三步：改 service 层

打开：

```text
internal/service/diary.go
```

写接口实现：

```go
// FavoriteDiary 实现收藏日记接口：把当前用户的指定日记标记为收藏。
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

service 层只做这些事：

1. 拿当前登录用户 ID
2. 拿请求参数
3. 调用 biz 层
4. 返回结果

### 第四步：改 biz 层

打开：

```text
internal/biz/diary.go
```

在 `DiaryRepo` 里加方法：

```go
FavoriteDiary(ctx context.Context, userID, diaryID int64) (*MoodDiary, error)
```

然后在 `DiaryUsecase` 里写：

```go
func (uc *DiaryUsecase) FavoriteDiary(ctx context.Context, userID, diaryID int64) (*MoodDiary, error) {
    return uc.repo.FavoriteDiary(ctx, userID, diaryID)
}
```

biz 层负责业务逻辑。

如果以后规则复杂，比如：

```text
已删除的日记不能收藏
```

这个规则就写在 biz 层。

### 第五步：改 data 层

打开：

```text
internal/data/diary.go
```

先看数据库表模型：

```go
type MoodDiaryModel struct {
    ID     int64
    UserID int64
}
```

如果需要加字段，比如：

```go
Favorite bool `gorm:"default:false"`
```

就加在这里。

然后实现数据库操作：

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

data 层才可以写数据库代码。

### 第六步：如果加了新表，要注册自动建表

打开：

```text
internal/data/data.go
```

找到：

```go
db.AutoMigrate(...)
```

把新表模型加进去。

### 第七步：如果加了新模块，要注册 service

如果只是给已有模块加接口，一般不用改。

如果你新增了一个完整模块，比如：

```text
通知 notification
```

那就要改：

```text
internal/service/wire.go
internal/biz/wire.go
internal/data/data.go
internal/server/http.go
cmd/emo-ai-service/wire.go
```

### 第八步：重新生成 Wire

如果改了依赖注入，执行：

```bash
go generate ./cmd/emo-ai-service
```

### 第九步：跑测试

最后执行：

```bash
go test ./...
```

## 七、现在有哪些主要接口

### 用户

```text
POST /v1/users/register
POST /v1/users/login
GET  /v1/users/me
PATCH /v1/users/me/profile
PUT /v1/users/me/avatar
```

### 心情日记

```text
POST /v1/diaries
GET /v1/diaries
GET /v1/diaries/{id}
PATCH /v1/diaries/{id}
DELETE /v1/diaries/{id}
```

### 心情标签

```text
GET /v1/mood-tags
POST /v1/mood-tags
PATCH /v1/mood-tags/{id}
DELETE /v1/mood-tags/{id}
```

### AI 聊天

```text
POST /v1/chat/sessions
GET /v1/chat/sessions
GET /v1/chat/sessions/{id}
GET /v1/chat/sessions/{session_id}/messages
POST /v1/chat/sessions/{session_id}/messages
```

### 情绪分析报告

```text
POST /v1/emotion/analyses
GET /v1/emotion/analyses
GET /v1/emotion/reports/overview
GET /v1/emotion/reports/trends
GET /v1/emotion/reports/calendar
```

### 隐私与安全

```text
POST /v1/security/password/change
GET /v1/security/login-logs
GET /v1/security/tokens
DELETE /v1/security/tokens/{token_id}
DELETE /v1/security/tokens
```

### 系统配置

```text
GET /v1/system/about
GET /v1/system/configs/public
GET /v1/system/versions/latest
GET /v1/system/announcements
```

## 八、AI 聊天是怎么走的

现在流程是：

```text
前端发送消息
  -> service/chat.go
  -> biz/chat.go
  -> data/chat.go 保存用户消息
  -> data/ai_client.go 生成 AI 回复
  -> data/chat.go 保存 AI 回复
  -> 返回给前端
```

以后接真实 AI，不要改 service。

应该改：

```text
internal/data/ai_client.go
```

## 九、情绪分析是怎么走的

现在流程是：

```text
前端提交文本
  -> service/emotion.go
  -> biz/emotion.go
  -> data/emotion.go 分析文本
  -> 保存 emotion_analyses
  -> 保存 emotion_dimension_scores
  -> 返回分析结果
```

以后接真实情绪分析模型，不要改 service。

应该改：

```text
internal/data/emotion.go
```

## 十、开发时不要这样写

### 1. 不要在 service 层查数据库

错误：

```go
s.db.Where(...)
```

正确：

```go
s.uc.CreateDiary(...)
```

### 2. 不要在 biz 层写数据库标签

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

### 3. 不要手动修改生成文件

不要改：

```text
*.pb.go
*_http.pb.go
*_grpc.pb.go
wire_gen.go
openapi.yaml
```

应该改：

```text
*.proto
wire.go
conf.proto
```

然后重新生成。

## 十一、你接下来应该按这个顺序做

如果你现在要继续开发，建议顺序是：

1. 先启动项目。
2. 先调通注册和登录。
3. 再调通 `/v1/users/me`。
4. 再调通心情日记新增、列表、详情、修改、删除。
5. 再调通标签接口。
6. 再调通聊天会话和发送消息。
7. 再接真实 AI。
8. 再接真实情绪分析。
9. 最后做系统公告、版本管理、文件上传。

不要一开始就接 AI。

先把用户、日记、聊天记录这些基础数据跑通。
