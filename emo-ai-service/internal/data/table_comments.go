package data

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

var tableComments = map[string]string{
	"users":                    "用户账号表",
	"user_profiles":            "用户资料表",
	"auth_refresh_tokens":      "刷新令牌表",
	"login_logs":               "登录日志表",
	"security_events":          "安全事件表",
	"mood_diaries":             "心情日记表",
	"mood_tags":                "心情标签表",
	"mood_diary_tags":          "心情日记标签关系表",
	"mood_diary_attachments":   "心情日记附件表",
	"chat_sessions":            "聊天会话表",
	"chat_messages":            "聊天消息表",
	"chat_context_summaries":   "聊天上下文摘要表",
	"chat_feedback":            "聊天反馈表",
	"emotion_analyses":         "情绪分析表",
	"emotion_dimension_scores": "情绪维度分数表",
	"emotion_daily_stats":      "每日情绪统计表",
	"system_configs":           "系统配置表",
	"app_versions":             "应用版本表",
	"system_announcements":     "系统公告表",
	"file_assets":              "文件资源表",
}

func applyTableComments(db *gorm.DB) error {
	for tableName, comment := range tableComments {
		sql := fmt.Sprintf("ALTER TABLE %s COMMENT = %s", quoteIdent(tableName), quoteLiteral(comment))
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}
	return nil
}

func quoteIdent(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func quoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
