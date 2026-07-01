package data

import (
	"context"
	"strings"
	"time"

	"emo-ai-service/internal/biz"
)

type localAIClient struct{}

func NewAIClient(*Data) biz.AIClient {
	return &localAIClient{}
}

func (c *localAIClient) Reply(ctx context.Context, req biz.AIReplyRequest) (*biz.AIReply, error) {
	start := time.Now()
	content := strings.TrimSpace(req.Content)
	reply := "我在认真听你说。"
	if content != "" {
		reply = "我理解你提到的感受。我们可以先把这件事拆小一点：刚才最影响你的念头是什么？"
	}
	return &biz.AIReply{
		Content:          reply,
		Model:            "local-support-v1",
		PromptTokens:     int32(len(req.History) * 20),
		CompletionTokens: int32(len([]rune(reply))),
		LatencyMS:        int32(time.Since(start).Milliseconds()),
		SafetyResultJSON: `{"risk":"low"}`,
	}, nil
}

func (c *localAIClient) Summarize(ctx context.Context, messages []*biz.ChatMessage) (*biz.AIReply, error) {
	summary := "本次咨询主要围绕情绪表达、压力来源和下一步行动展开。"
	if len(messages) == 0 {
		summary = "暂无可总结的聊天内容。"
	}
	return &biz.AIReply{
		Content:          summary,
		Model:            "local-summary-v1",
		CompletionTokens: int32(len([]rune(summary))),
		SafetyResultJSON: `{"risk":"low"}`,
	}, nil
}
