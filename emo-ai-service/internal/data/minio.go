package data

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	minioDefaultBucket          = "emotion-avatars"
	minioDefaultKnowledgeBucket = "emotion-knowledge"
	minioService                = "s3"
)

// minioStorage is a small S3-compatible client used only for user avatars.
// Keeping it in data means credentials and HTTP signing never cross into biz.
type minioStorage struct {
	endpoint       *url.URL
	publicEndpoint *url.URL
	bucket         string
	accessKey      string
	secretKey      string
	region         string
	httpClient     *http.Client
	publicRead     bool

	mu          sync.Mutex
	initialized bool
}

type minioKnowledgeObject struct {
	ObjectReference string
	ObjectKey       string
	Name            string
	SizeBytes       int64
	LastModified    time.Time
}

type minioListBucketResult struct {
	IsTruncated           bool   `xml:"IsTruncated"`
	NextContinuationToken string `xml:"NextContinuationToken"`
	Contents              []struct {
		Key          string    `xml:"Key"`
		LastModified time.Time `xml:"LastModified"`
		Size         int64     `xml:"Size"`
	} `xml:"Contents"`
}

func newMinioStorage() *minioStorage {
	return newMinioStorageForBucket("EMO_MINIO_BUCKET", minioDefaultBucket, true)
}

func newKnowledgeMinioStorage() *minioStorage {
	return newMinioStorageForBucket("EMO_MINIO_KNOWLEDGE_BUCKET", minioDefaultKnowledgeBucket, false)
}

func newMinioStorageForBucket(bucketEnv, defaultBucket string, publicRead bool) *minioStorage {
	endpoint := parseMinioEndpoint(os.Getenv("EMO_MINIO_ENDPOINT"), os.Getenv("EMO_MINIO_USE_SSL"))
	publicEndpoint := parseMinioEndpoint(os.Getenv("EMO_MINIO_PUBLIC_ENDPOINT"), os.Getenv("EMO_MINIO_USE_SSL"))
	if publicEndpoint == nil {
		publicEndpoint = endpoint
	}
	bucket := strings.TrimSpace(os.Getenv(bucketEnv))
	if bucket == "" {
		bucket = defaultBucket
	}
	region := strings.TrimSpace(os.Getenv("EMO_MINIO_REGION"))
	if region == "" {
		region = "us-east-1"
	}
	return &minioStorage{
		endpoint:       endpoint,
		publicEndpoint: publicEndpoint,
		bucket:         bucket,
		accessKey:      strings.TrimSpace(os.Getenv("EMO_MINIO_ACCESS_KEY")),
		secretKey:      strings.TrimSpace(os.Getenv("EMO_MINIO_SECRET_KEY")),
		region:         region,
		httpClient:     &http.Client{Timeout: 20 * time.Second},
		publicRead:     publicRead,
	}
}

func (s *minioStorage) configured() bool {
	return s != nil && s.endpoint != nil && s.publicEndpoint != nil && s.bucket != "" && s.accessKey != "" && s.secretKey != ""
}

func (s *minioStorage) uploadAvatar(ctx context.Context, objectKey, mimeType string, content []byte) (string, error) {
	if err := s.uploadObject(ctx, objectKey, mimeType, content); err != nil {
		return "", err
	}
	return s.objectURL(s.publicEndpoint, objectKey), nil
}

func (s *minioStorage) uploadKnowledge(ctx context.Context, objectKey, mimeType string, content []byte) (string, error) {
	if err := s.uploadObject(ctx, objectKey, mimeType, content); err != nil {
		return "", err
	}
	return "s3://" + s.bucket + "/" + strings.TrimLeft(objectKey, "/"), nil
}

func (s *minioStorage) listKnowledgeObjects(ctx context.Context, userID int64) ([]minioKnowledgeObject, error) {
	if !s.configured() {
		return nil, fmt.Errorf("minio is not configured")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("invalid knowledge owner")
	}
	if err := s.ensureBucket(ctx); err != nil {
		return nil, err
	}

	prefix := fmt.Sprintf("knowledge/%d/", userID)
	objects := make([]minioKnowledgeObject, 0)
	continuationToken := ""
	for {
		query := url.Values{
			"list-type": {"2"},
			"max-keys":  {"1000"},
			"prefix":    {prefix},
		}
		if continuationToken != "" {
			query.Set("continuation-token", continuationToken)
		}

		resp, err := s.doSigned(ctx, http.MethodGet, s.bucketURL(s.endpoint), query, nil, nil)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			err = minioResponseError("list knowledge objects", resp)
			resp.Body.Close()
			return nil, err
		}

		var result minioListBucketResult
		err = xml.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decode knowledge object list: %w", err)
		}
		for _, item := range result.Contents {
			if item.Key == "" || strings.HasSuffix(item.Key, "/") {
				continue
			}
			objects = append(objects, minioKnowledgeObject{
				ObjectReference: "s3://" + s.bucket + "/" + strings.TrimLeft(item.Key, "/"),
				ObjectKey:       item.Key,
				Name:            path.Base(item.Key),
				SizeBytes:       item.Size,
				LastModified:    item.LastModified,
			})
		}
		if !result.IsTruncated || result.NextContinuationToken == "" {
			break
		}
		continuationToken = result.NextContinuationToken
	}

	sort.SliceStable(objects, func(i, j int) bool {
		return objects[i].LastModified.After(objects[j].LastModified)
	})
	return objects, nil
}

func (s *minioStorage) uploadObject(ctx context.Context, objectKey, mimeType string, content []byte) error {
	if !s.configured() {
		return fmt.Errorf("minio is not configured")
	}
	if err := s.ensureBucket(ctx); err != nil {
		return err
	}

	objectURL := s.objectURL(s.endpoint, objectKey)
	headers := make(http.Header)
	headers.Set("Content-Type", mimeType)
	resp, err := s.doSigned(ctx, http.MethodPut, objectURL, nil, content, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return minioResponseError("upload object", resp)
	}
	return nil
}

func (s *minioStorage) ensureBucket(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.initialized {
		return nil
	}

	bucketURL := s.bucketURL(s.endpoint)
	resp, err := s.doSigned(ctx, http.MethodHead, bucketURL, nil, nil, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		resp, err = s.doSigned(ctx, http.MethodPut, bucketURL, nil, nil, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return minioResponseError("create object bucket", resp)
		}
	} else if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("check object bucket: unexpected status %s", resp.Status)
	}
	if !s.publicRead {
		s.initialized = true
		return nil
	}

	policy, err := json.Marshal(map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{{
			"Effect":    "Allow",
			"Principal": "*",
			"Action":    []string{"s3:GetObject"},
			"Resource":  []string{fmt.Sprintf("arn:aws:s3:::%s/*", s.bucket)},
		}},
	})
	if err != nil {
		return err
	}
	policyURL := bucketURL + "?policy="
	resp, err = s.doSigned(ctx, http.MethodPut, policyURL, nil, policy, http.Header{"Content-Type": []string{"application/json"}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return minioResponseError("set avatar bucket policy", resp)
	}
	s.initialized = true
	return nil
}

func (s *minioStorage) doSigned(ctx context.Context, method, rawURL string, query url.Values, payload []byte, headers http.Header) (*http.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}
	if headers == nil {
		headers = make(http.Header)
	}
	payloadHash := sha256Hex(payload)
	now := time.Now().UTC()
	headers.Set("X-Amz-Content-Sha256", payloadHash)
	headers.Set("X-Amz-Date", now.Format("20060102T150405Z"))

	canonicalHeaders := "host:" + u.Host + "\n" +
		"x-amz-content-sha256:" + payloadHash + "\n" +
		"x-amz-date:" + headers.Get("X-Amz-Date") + "\n"
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalRequest := strings.Join([]string{
		method,
		canonicalPath(u),
		canonicalQuery(u.Query()),
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")
	date := now.Format("20060102")
	credentialScope := strings.Join([]string{date, s.region, minioService, "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		headers.Get("X-Amz-Date"),
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	signature := hex.EncodeToString(hmacSHA256(signingKey(s.secretKey, date, s.region, minioService), stringToSign))
	headers.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", s.accessKey, credentialScope, signedHeaders, signature))

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		req.Body = nil
	}
	req.Header = headers
	return s.httpClient.Do(req)
}

func (s *minioStorage) bucketURL(base *url.URL) string {
	u := *base
	u.Path = joinURLPath(u.Path, s.bucket)
	u.RawQuery = ""
	return u.String()
}

func (s *minioStorage) objectURL(base *url.URL, objectKey string) string {
	u := *base
	u.Path = joinURLPath(u.Path, s.bucket, objectKey)
	u.RawQuery = ""
	return u.String()
}

func (s *minioStorage) ownsAvatarURL(userID int64, avatarURL string) bool {
	if !s.configured() || userID <= 0 {
		return false
	}
	actual, err := url.Parse(strings.TrimSpace(avatarURL))
	if err != nil {
		return false
	}
	expected, err := url.Parse(s.objectURL(s.publicEndpoint, fmt.Sprintf("avatars/%d/", userID)))
	if err != nil {
		return false
	}
	if actual.Scheme != expected.Scheme || actual.Host != expected.Host {
		return false
	}
	ownedPrefix := strings.TrimRight(expected.EscapedPath(), "/") + "/"
	return strings.HasPrefix(actual.EscapedPath(), ownedPrefix) && actual.RawQuery == "" && actual.Fragment == ""
}

func parseMinioEndpoint(value, useSSL string) *url.URL {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if !strings.Contains(value, "://") {
		scheme := "http"
		if strings.EqualFold(strings.TrimSpace(useSSL), "true") {
			scheme = "https"
		}
		value = scheme + "://" + value
	}
	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	return u
}

func joinURLPath(parts ...string) string {
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(part, "/")
		if part != "" {
			segments = append(segments, part)
		}
	}
	return "/" + strings.Join(segments, "/")
}

func canonicalPath(u *url.URL) string {
	path := u.EscapedPath()
	if path == "" {
		return "/"
	}
	return path
}

func canonicalQuery(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		values := append([]string(nil), values[key]...)
		sort.Strings(values)
		for _, value := range values {
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(value))
		}
	}
	return strings.Join(parts, "&")
}

func signingKey(secret, date, region, service string) []byte {
	dateKey := hmacSHA256([]byte("AWS4"+secret), date)
	regionKey := hmacSHA256(dateKey, region)
	serviceKey := hmacSHA256(regionKey, service)
	return hmacSHA256(serviceKey, "aws4_request")
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = io.WriteString(mac, value)
	return mac.Sum(nil)
}

func sha256Hex(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func minioResponseError(action string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = resp.Status
	}
	return fmt.Errorf("%s: %s", action, message)
}
