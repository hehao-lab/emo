package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	v1 "emo-ai-service/api/aichat/v1"
	"emo-ai-service/internal/biz"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

// StreamChatHTTP forwards POST SSE bytes from FastAPI to the frontend.
//
// It is registered as a raw HTTP handler so headers, flush timing, and client
// disconnects can be controlled without unary RPC response encoding.
func (s *AIChatService) StreamChatHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	userID, err := s.userIDFromHTTPRequest(r)
	if err != nil {
		writeJSONError(w, err)
		return
	}

	// Validation/auth errors are returned as JSON before the SSE response starts.
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var req v1.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, kerrors.BadRequest("INVALID_JSON", "invalid request body"))
		return
	}
	if req.SystemPrompt != nil && !s.requestHasRole(r, "admin") {
		writeJSONError(w, kerrors.Forbidden("SYSTEM_PROMPT_FORBIDDEN", "system prompt is restricted to administrators"))
		return
	}

	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	clientRequestID := strings.TrimSpace(req.GetClientRequestId())
	if idempotencyKey == "" {
		idempotencyKey = clientRequestID
	}
	if idempotencyKey == "" {
		idempotencyKey = uuid.NewString()
	}
	if clientRequestID == "" {
		clientRequestID = idempotencyKey
	}

	stream, err := s.uc.StreamChat(r.Context(), &biz.AIChatRequest{
		UserID:         userID,
		UpstreamUserID: upstreamUserID(userID),
		ConversationID: req.ConversationId,
		Message:        req.GetMessage(),
		SystemPrompt:   req.SystemPrompt,
		ClientRequestID: clientRequestID,
		IdempotencyKey: idempotencyKey,
		Traceparent:    r.Header.Get("traceparent"),
	})
	if err != nil {
		writeJSONError(w, err)
		return
	}
	defer stream.Body.Close()

	// A streaming response must support Flush so deltas are visible immediately.
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, kerrors.InternalServer("SSE_UNSUPPORTED", "streaming unsupported"))
		return
	}

	header := w.Header()
	header.Set("Content-Type", "text/event-stream; charset=utf-8")
	header.Set("Cache-Control", "no-store")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	if stream.Traceparent != "" {
		header.Set("traceparent", stream.Traceparent)
	}
	if stream.IdempotencyReplayed {
		header.Set("Idempotency-Replayed", "true")
	}

	// After headers are written, later failures must be sent as SSE events.
	status := stream.StatusCode
	if status == 0 {
		status = http.StatusCreated
	}
	w.WriteHeader(status)
	flusher.Flush()

	// Copy raw SSE frames as-is; FastAPI already formats event/data blocks.
	buf := make([]byte, 32*1024)
	for {
		n, readErr := stream.Body.Read(buf)
		if n > 0 {
			if _, err := w.Write(buf[:n]); err != nil {
				return
			}
			flusher.Flush()
		}
		if readErr == io.EOF {
			return
		}
		if readErr != nil {
			writeSSEError(w, flusher, "AI_SERVICE_STREAM_INTERRUPTED", "AI service stream interrupted", true)
			return
		}
	}
}

func (s *AIChatService) requestHasRole(r *http.Request, wanted string) bool {
	if s.tokenManager == nil {
		return false
	}
	claims, err := s.tokenManager.Parse(bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		return false
	}
	for _, role := range claims.Roles {
		if strings.EqualFold(role, wanted) {
			return true
		}
	}
	return false
}

// userIDFromHTTPRequest authenticates raw HTTP handlers with the same JWT token.
func (s *AIChatService) userIDFromHTTPRequest(r *http.Request) (int64, error) {
	if s.tokenManager == nil {
		return 0, kerrors.Unauthorized("UNAUTHORIZED", "login required")
	}
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		return 0, kerrors.Unauthorized("UNAUTHORIZED", "missing access token")
	}
	claims, err := s.tokenManager.Parse(token)
	if err != nil {
		return 0, kerrors.Unauthorized("UNAUTHORIZED", "invalid access token")
	}
	if claims.UserID <= 0 {
		return 0, kerrors.Unauthorized("UNAUTHORIZED", "invalid access token")
	}
	return claims.UserID, nil
}

// bearerToken extracts the OAuth2 Bearer token from Authorization.
func bearerToken(value string) string {
	parts := strings.Fields(value)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

// writeJSONError returns a stable JSON error shape before an SSE stream starts.
func writeJSONError(w http.ResponseWriter, err error) {
	se := kerrors.FromError(err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(int(se.Code))
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    se.Code,
		"reason":  se.Reason,
		"message": se.Message,
	})
}

// writeSSEError reports mid-stream failures in the documented SSE event format.
func writeSSEError(w http.ResponseWriter, flusher http.Flusher, code, detail string, retryable bool) {
	payload, _ := json.Marshal(map[string]any{"code": code, "detail": detail, "retryable": retryable})
	_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", payload)
	flusher.Flush()
}
