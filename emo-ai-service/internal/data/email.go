package data

import (
	"context"
	"log"
	"time"

	"emo-ai-service/internal/biz"
)

type logEmailSender struct{}

func NewEmailSender() biz.EmailSender {
	return &logEmailSender{}
}

func (s *logEmailSender) SendVerificationCode(ctx context.Context, email, code string, ttl time.Duration) error {
	log.Printf("register email verification code: email=%s code=%s ttl=%s", email, code, ttl)
	return nil
}
