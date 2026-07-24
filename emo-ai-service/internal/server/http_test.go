package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	aichatv1 "emo-ai-service/api/aichat/v1"
)

func TestContractResponseEncoderStatusCodes(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		value  any
		want   int
	}{
		{name: "conversation created", method: http.MethodPost, path: "/api/v1/conversations", value: &aichatv1.Conversation{}, want: http.StatusCreated},
		{name: "chat created", method: http.MethodPost, path: "/api/v1/chat", value: &aichatv1.ChatReply{}, want: http.StatusCreated},
		{name: "knowledge queued", method: http.MethodPost, path: "/api/v1/knowledge/documents", value: &aichatv1.CreateKnowledgeDocumentReply{}, want: http.StatusAccepted},
		{name: "reindex queued", method: http.MethodPost, path: "/api/v1/knowledge/documents/doc-1:reindex", value: &aichatv1.ReindexKnowledgeDocumentReply{}, want: http.StatusAccepted},
		{name: "knowledge deleted", method: http.MethodDelete, path: "/api/v1/knowledge/documents/doc-1", value: nil, want: http.StatusNoContent},
		{name: "ordinary response unchanged", method: http.MethodGet, path: "/api/v1/knowledge/documents", value: &aichatv1.KnowledgeDocumentSet{}, want: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, nil)
			if err := contractResponseEncoder(recorder, request, tt.value); err != nil {
				t.Fatalf("encode response: %v", err)
			}
			if recorder.Code != tt.want {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.want)
			}
			if tt.want == http.StatusNoContent && recorder.Body.Len() != 0 {
				t.Fatalf("204 response body length = %d, want 0", recorder.Body.Len())
			}
		})
	}
}
