package data

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestListKnowledgeObjectsUsesUserPrefix(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		if got := r.URL.Query().Get("prefix"); got != "knowledge/42/" {
			t.Fatalf("prefix = %q, want %q", got, "knowledge/42/")
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
  <Name>emotion-knowledge</Name>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>knowledge/42/20260724/id-1/guide.pdf</Key>
    <LastModified>2026-07-24T08:00:00.000Z</LastModified>
    <Size>1234</Size>
  </Contents>
</ListBucketResult>`)
	}))
	defer server.Close()

	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	storage := &minioStorage{
		endpoint:       endpoint,
		publicEndpoint: endpoint,
		bucket:         "emotion-knowledge",
		accessKey:      "test-access",
		secretKey:      "test-secret",
		region:         "us-east-1",
		httpClient:     server.Client(),
	}

	objects, err := storage.listKnowledgeObjects(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 1 {
		t.Fatalf("len(objects) = %d, want 1", len(objects))
	}
	if objects[0].Name != "guide.pdf" {
		t.Fatalf("name = %q, want guide.pdf", objects[0].Name)
	}
	if objects[0].ObjectReference != "s3://emotion-knowledge/knowledge/42/20260724/id-1/guide.pdf" {
		t.Fatalf("object reference = %q", objects[0].ObjectReference)
	}
}
