package data

import (
	"context"
	"time"

	"emo-ai-service/internal/biz"

	"gorm.io/gorm"
)

type ChatSessionModel struct {
	ID                     int64          `gorm:"primaryKey;autoIncrement;comment:聊天会话ID"`
	UserID                 int64          `gorm:"index:idx_user_last;not null;comment:用户ID"`
	Title                  string         `gorm:"type:varchar(128);default:'';comment:会话标题"`
	Scenario               string         `gorm:"type:varchar(64);default:'emotional_support';comment:咨询场景"`
	Status                 string         `gorm:"type:varchar(16);index;default:'active';comment:会话状态"`
	Summary                string         `gorm:"type:text;comment:会话摘要"`
	UpstreamConversationID string         `gorm:"type:varchar(128);index;default:'';comment:上游AI会话ID"`
	LastMessageAt          *time.Time     `gorm:"index:idx_user_last;comment:最后消息时间"`
	MessageCount           int32          `gorm:"default:0;comment:消息数量"`
	CreatedAt              time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt              time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt              gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

func (ChatSessionModel) TableName() string { return "chat_sessions" }

type ChatMessageModel struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement;comment:聊天消息ID"`
	SessionID           int64     `gorm:"index:idx_session_created;not null;comment:聊天会话ID"`
	UserID              int64     `gorm:"index;uniqueIndex:idx_chat_message_idempotency,priority:1;not null;comment:用户ID"`
	Role                string    `gorm:"type:varchar(16);uniqueIndex:idx_chat_message_idempotency,priority:3;not null;comment:消息角色 user assistant system tool"`
	Content             string    `gorm:"type:text;not null;comment:消息内容"`
	ContentType         string    `gorm:"type:varchar(32);default:'text';comment:消息内容类型"`
	Model               string    `gorm:"type:varchar(64);default:'';comment:AI模型名称"`
	PromptTokens        int32     `gorm:"default:0;comment:提示词token数"`
	CompletionTokens    int32     `gorm:"default:0;comment:回复token数"`
	TotalTokens         int32     `gorm:"default:0;comment:总token数"`
	LatencyMS           int32     `gorm:"default:0;comment:AI回复耗时毫秒"`
	EmotionSnapshotJSON string    `gorm:"type:json;comment:消息情绪快照JSON"`
	SafetyResultJSON    string    `gorm:"type:json;comment:安全检测结果JSON"`
	Status              string    `gorm:"type:varchar(16);default:'success';comment:消息状态"`
	ErrorMessage        string    `gorm:"type:varchar(512);default:'';comment:错误信息"`
	ClientRequestID     *string   `gorm:"type:varchar(128);index;comment:客户端请求ID"`
	IdempotencyKey      *string   `gorm:"type:varchar(128);uniqueIndex:idx_chat_message_idempotency,priority:2;comment:逻辑聊天轮次幂等键"`
	RequestPayloadHash  string    `gorm:"type:char(64);default:'';comment:幂等请求载荷哈希"`
	RequestID           string    `gorm:"type:varchar(128);index;default:'';comment:AI服务请求ID"`
	Provider            string    `gorm:"type:varchar(64);default:'';comment:模型提供商"`
	ProviderRequestID   string    `gorm:"type:varchar(128);index;default:'';comment:模型提供商请求ID"`
	ReferencesJSON      string    `gorm:"type:json;comment:结构化引用"`
	UsageJSON           string    `gorm:"type:json;comment:模型用量JSON"`
	CachedTokens        int32     `gorm:"default:0;comment:缓存token数"`
	CostMicros          int64     `gorm:"default:0;comment:费用微单位"`
	CreatedAt           time.Time `gorm:"autoCreateTime;index:idx_session_created;comment:创建时间"`
}

func (ChatMessageModel) TableName() string { return "chat_messages" }

type ChatContextSummaryModel struct {
	ID             int64     `gorm:"primaryKey;autoIncrement;comment:会话摘要ID"`
	SessionID      int64     `gorm:"index;not null;comment:聊天会话ID"`
	Summary        string    `gorm:"type:text;not null;comment:摘要内容"`
	MessageStartID int64     `gorm:"default:0;comment:摘要起始消息ID"`
	MessageEndID   int64     `gorm:"default:0;comment:摘要结束消息ID"`
	Model          string    `gorm:"type:varchar(64);default:'';comment:摘要模型名称"`
	CreatedAt      time.Time `gorm:"autoCreateTime;comment:创建时间"`
}

func (ChatContextSummaryModel) TableName() string { return "chat_context_summaries" }

type ChatFeedbackModel struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;comment:消息反馈ID"`
	UserID       int64     `gorm:"index;not null;comment:用户ID"`
	SessionID    int64     `gorm:"index;not null;comment:聊天会话ID"`
	MessageID    int64     `gorm:"index;not null;comment:被反馈的消息ID"`
	Rating       int32     `gorm:"default:0;comment:评分"`
	FeedbackType string    `gorm:"type:varchar(32);default:'';comment:反馈类型"`
	Content      string    `gorm:"type:varchar(512);default:'';comment:反馈内容"`
	CreatedAt    time.Time `gorm:"autoCreateTime;comment:创建时间"`
}

func (ChatFeedbackModel) TableName() string { return "chat_feedback" }

type chatRepoImpl struct {
	db *gorm.DB
}

func NewChatRepo(d *Data) biz.ChatRepo {
	return &chatRepoImpl{db: d.db}
}

func (r *chatRepoImpl) CreateSession(ctx context.Context, session *biz.ChatSession) (*biz.ChatSession, error) {
	model := &ChatSessionModel{
		UserID:                 session.UserID,
		Title:                  session.Title,
		Scenario:               session.Scenario,
		Status:                 session.Status,
		Summary:                session.Summary,
		UpstreamConversationID: session.UpstreamConversationID,
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toBizChatSession(model), nil
}

func (r *chatRepoImpl) ListSessions(ctx context.Context, userID int64, opt biz.ChatListOption) ([]*biz.ChatSession, int64, error) {
	p, size := normalizePage(opt.Page, opt.PageSize)
	q := r.db.WithContext(ctx).Model(&ChatSessionModel{}).Where("user_id = ?", userID)
	if opt.Status != "" {
		q = q.Where("status = ?", opt.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []ChatSessionModel
	if err := q.Order("COALESCE(last_message_at, created_at) desc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.ChatSession, 0, len(models))
	for i := range models {
		out = append(out, toBizChatSession(&models[i]))
	}
	return out, total, nil
}

func (r *chatRepoImpl) GetSession(ctx context.Context, userID, id int64) (*biz.ChatSession, error) {
	var model ChatSessionModel
	err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, id).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toBizChatSession(&model), nil
}

func (r *chatRepoImpl) UpdateSession(ctx context.Context, session *biz.ChatSession) (*biz.ChatSession, error) {
	var model ChatSessionModel
	err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", session.UserID, session.ID).First(&model).Error
	if err != nil {
		return nil, err
	}
	if session.Title != "" {
		model.Title = session.Title
	}
	if session.Status != "" {
		model.Status = session.Status
	}
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, err
	}
	return toBizChatSession(&model), nil
}

func (r *chatRepoImpl) DeleteSession(ctx context.Context, userID, id int64) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, id).Delete(&ChatSessionModel{}).Error
}

func (r *chatRepoImpl) CreateMessage(ctx context.Context, message *biz.ChatMessage) (*biz.ChatMessage, error) {
	model := &ChatMessageModel{
		SessionID:           message.SessionID,
		UserID:              message.UserID,
		Role:                message.Role,
		Content:             message.Content,
		ContentType:         message.ContentType,
		Model:               message.Model,
		PromptTokens:        message.PromptTokens,
		CompletionTokens:    message.CompletionTokens,
		TotalTokens:         message.TotalTokens,
		LatencyMS:           message.LatencyMS,
		EmotionSnapshotJSON: jsonObject(message.EmotionSnapshotJSON),
		SafetyResultJSON:    jsonObject(message.SafetyResultJSON),
		Status:              message.Status,
		ErrorMessage:        message.ErrorMessage,
		ClientRequestID:     message.ClientRequestID,
		IdempotencyKey:      message.IdempotencyKey,
		RequestPayloadHash:  message.RequestPayloadHash,
		RequestID:           message.RequestID,
		Provider:            message.Provider,
		ProviderRequestID:   message.ProviderRequestID,
		ReferencesJSON:      jsonArray(message.ReferencesJSON),
		UsageJSON:           jsonObject(message.UsageJSON),
		CachedTokens:        message.CachedTokens,
		CostMicros:          message.CostMicros,
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toBizChatMessage(model), nil
}

func (r *chatRepoImpl) UpdateMessage(ctx context.Context, message *biz.ChatMessage) (*biz.ChatMessage, error) {
	updates := map[string]any{
		"content": message.Content, "model": message.Model, "status": message.Status,
		"error_message": message.ErrorMessage, "prompt_tokens": message.PromptTokens,
		"completion_tokens": message.CompletionTokens, "total_tokens": message.TotalTokens,
		"latency_ms": message.LatencyMS, "references_json": jsonArray(message.ReferencesJSON),
		"usage_json": jsonObject(message.UsageJSON), "cached_tokens": message.CachedTokens,
		"cost_micros": message.CostMicros, "request_id": message.RequestID,
		"provider": message.Provider, "provider_request_id": message.ProviderRequestID,
	}
	if err := r.db.WithContext(ctx).Model(&ChatMessageModel{}).
		Where("id = ? AND user_id = ?", message.ID, message.UserID).Updates(updates).Error; err != nil {
		return nil, err
	}
	var model ChatMessageModel
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", message.ID, message.UserID).First(&model).Error; err != nil {
		return nil, err
	}
	return toBizChatMessage(&model), nil
}

func (r *chatRepoImpl) FindMessagesByIdempotencyKey(ctx context.Context, userID int64, idempotencyKey string) ([]*biz.ChatMessage, error) {
	if idempotencyKey == "" {
		return nil, nil
	}
	var models []ChatMessageModel
	if err := r.db.WithContext(ctx).Where("user_id = ? AND idempotency_key = ?", userID, idempotencyKey).
		Order("created_at asc, id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.ChatMessage, 0, len(models))
	for i := range models {
		out = append(out, toBizChatMessage(&models[i]))
	}
	return out, nil
}

func (r *chatRepoImpl) DailyUsage(ctx context.Context, userID int64, since time.Time) (*biz.ChatDailyUsage, error) {
	var total struct {
		TotalTokens int64
		CostMicros  int64
	}
	err := r.db.WithContext(ctx).Model(&ChatMessageModel{}).
		Select("COALESCE(SUM(total_tokens), 0) AS total_tokens, COALESCE(SUM(cost_micros), 0) AS cost_micros").
		Where("user_id = ? AND role = ? AND created_at >= ?", userID, "assistant", since).
		Scan(&total).Error
	if err != nil {
		return nil, err
	}
	return &biz.ChatDailyUsage{TotalTokens: total.TotalTokens, CostMicros: total.CostMicros}, nil
}

func (r *chatRepoImpl) ListMessages(ctx context.Context, userID, sessionID int64, page, pageSize int32) ([]*biz.ChatMessage, int64, error) {
	p, size := normalizePage(page, pageSize)
	var exists int64
	if err := r.db.WithContext(ctx).Model(&ChatSessionModel{}).Where("user_id = ? AND id = ?", userID, sessionID).Count(&exists).Error; err != nil {
		return nil, 0, err
	}
	if exists == 0 {
		return []*biz.ChatMessage{}, 0, nil
	}
	q := r.db.WithContext(ctx).Model(&ChatMessageModel{}).Where("session_id = ?", sessionID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []ChatMessageModel
	if err := q.Order("created_at asc, id asc").Offset((p - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.ChatMessage, 0, len(models))
	for i := range models {
		out = append(out, toBizChatMessage(&models[i]))
	}
	return out, total, nil
}

func (r *chatRepoImpl) RecentMessages(ctx context.Context, sessionID int64, limit int) ([]*biz.ChatMessage, error) {
	var models []ChatMessageModel
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at desc, id desc").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.ChatMessage, 0, len(models))
	for i := len(models) - 1; i >= 0; i-- {
		out = append(out, toBizChatMessage(&models[i]))
	}
	return out, nil
}

func (r *chatRepoImpl) TouchSession(ctx context.Context, sessionID int64, lastMessageAt time.Time, deltaCount int) error {
	return r.db.WithContext(ctx).Model(&ChatSessionModel{}).Where("id = ?", sessionID).
		Updates(map[string]any{"last_message_at": &lastMessageAt, "message_count": gorm.Expr("message_count + ?", deltaCount)}).Error
}

func (r *chatRepoImpl) BindSessionUpstream(ctx context.Context, sessionID int64, upstreamConversationID string) error {
	if upstreamConversationID == "" {
		return nil
	}

	return r.db.WithContext(ctx).Model(&ChatSessionModel{}).Where("id = ?", sessionID).
		Update("upstream_conversation_id", upstreamConversationID).Error
}

func (r *chatRepoImpl) CreateFeedback(ctx context.Context, feedback *biz.ChatFeedback) (*biz.ChatFeedback, error) {
	var message ChatMessageModel
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", feedback.MessageID, feedback.UserID).First(&message).Error; err != nil {
		return nil, err
	}
	model := &ChatFeedbackModel{
		UserID:       feedback.UserID,
		SessionID:    message.SessionID,
		MessageID:    feedback.MessageID,
		Rating:       feedback.Rating,
		FeedbackType: feedback.FeedbackType,
		Content:      feedback.Content,
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return &biz.ChatFeedback{ID: model.ID, UserID: model.UserID, SessionID: model.SessionID, MessageID: model.MessageID, Rating: model.Rating, FeedbackType: model.FeedbackType, Content: model.Content, CreatedAt: model.CreatedAt}, nil
}

func (r *chatRepoImpl) CreateSummary(ctx context.Context, summary *biz.ChatContextSummary) (*biz.ChatContextSummary, error) {
	model := &ChatContextSummaryModel{
		SessionID:      summary.SessionID,
		Summary:        summary.Summary,
		MessageStartID: summary.MessageStartID,
		MessageEndID:   summary.MessageEndID,
		Model:          summary.Model,
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	_ = r.db.WithContext(ctx).Model(&ChatSessionModel{}).Where("id = ?", summary.SessionID).Update("summary", summary.Summary).Error
	return &biz.ChatContextSummary{ID: model.ID, SessionID: model.SessionID, Summary: model.Summary, MessageStartID: model.MessageStartID, MessageEndID: model.MessageEndID, Model: model.Model, CreatedAt: model.CreatedAt}, nil
}

func toBizChatSession(model *ChatSessionModel) *biz.ChatSession {
	var last time.Time
	if model.LastMessageAt != nil {
		last = *model.LastMessageAt
	}
	return &biz.ChatSession{
		ID:                     model.ID,
		UserID:                 model.UserID,
		Title:                  model.Title,
		Scenario:               model.Scenario,
		Status:                 model.Status,
		Summary:                model.Summary,
		UpstreamConversationID: model.UpstreamConversationID,
		MessageCount:           model.MessageCount,
		LastMessageAt:          last,
		CreatedAt:              model.CreatedAt,
		UpdatedAt:              model.UpdatedAt,
	}
}

func toBizChatMessage(model *ChatMessageModel) *biz.ChatMessage {
	return &biz.ChatMessage{
		ID:                  model.ID,
		SessionID:           model.SessionID,
		UserID:              model.UserID,
		Role:                model.Role,
		Content:             model.Content,
		ContentType:         model.ContentType,
		Model:               model.Model,
		PromptTokens:        model.PromptTokens,
		CompletionTokens:    model.CompletionTokens,
		TotalTokens:         model.TotalTokens,
		LatencyMS:           model.LatencyMS,
		EmotionSnapshotJSON: model.EmotionSnapshotJSON,
		SafetyResultJSON:    model.SafetyResultJSON,
		Status:              model.Status,
		ErrorMessage:        model.ErrorMessage,
		ClientRequestID:     model.ClientRequestID,
		IdempotencyKey:      model.IdempotencyKey,
		RequestPayloadHash:  model.RequestPayloadHash,
		RequestID:           model.RequestID,
		Provider:            model.Provider,
		ProviderRequestID:   model.ProviderRequestID,
		ReferencesJSON:      model.ReferencesJSON,
		UsageJSON:           model.UsageJSON,
		CachedTokens:        model.CachedTokens,
		CostMicros:          model.CostMicros,
		CreatedAt:           model.CreatedAt,
	}
}

func jsonObject(value string) string {
	if value == "" {
		return "{}"
	}
	return value
}

func jsonArray(value string) string {
	if value == "" {
		return "[]"
	}
	return value
}
