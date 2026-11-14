package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// S3Config holds connection credentials for S3 / Cloudflare R2 / Neon Object Storage.
type S3Config struct {
	Endpoint        string // e.g. "https://<accountid>.r2.cloudflarestorage.com"
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	PublicBaseURL   string // CDN or public bucket URL
}

// S3Provider implements StorageProvider for S3-compatible object stores.
type S3Provider struct {
	cfg        S3Config
	httpClient *http.Client
}

// NewS3Provider creates a new S3 / R2 storage provider.
func NewS3Provider(cfg S3Config) (*S3Provider, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("S3 bucket name is required")
	}
	if cfg.Region == "" {
		cfg.Region = "auto"
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = fmt.Sprintf("https://s3.%s.amazonaws.com", cfg.Region)
	}
	endpoint, err := url.Parse(cfg.Endpoint)
	if err != nil || (endpoint.Scheme != "https" && endpoint.Scheme != "http") || endpoint.Host == "" || endpoint.User != nil {
		return nil, errors.New("S3 endpoint must be an absolute HTTP or HTTPS URL")
	}

	return &S3Provider{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (s *S3Provider) Name() string {
	return "s3"
}

// GenerateUploadURL creates an S3 Presigned PUT URL using AWS Signature Version 4.
func (s *S3Provider) GenerateUploadURL(ctx context.Context, key string, contentType string, sizeBytes int64, ttl time.Duration) (string, map[string]string, error) {
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	endpointURL, err := url.Parse(s.cfg.Endpoint)
	if err != nil {
		return "", nil, fmt.Errorf("invalid S3 endpoint URL: %w", err)
	}

	// Host path style: endpoint/bucket/key
	host := endpointURL.Host
	canonicalURI := fmt.Sprintf("/%s/%s", s.cfg.Bucket, strings.TrimPrefix(key, "/"))

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, s.cfg.Region)
	credential := fmt.Sprintf("%s/%s", s.cfg.AccessKeyID, credentialScope)

	expiresSeconds := int64(ttl.Seconds())
	if expiresSeconds <= 0 {
		expiresSeconds = 900 // 15 mins default
	}

	queryParams := url.Values{}
	queryParams.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	queryParams.Set("X-Amz-Credential", credential)
	queryParams.Set("X-Amz-Date", amzDate)
	queryParams.Set("X-Amz-Expires", fmt.Sprintf("%d", expiresSeconds))
	queryParams.Set("X-Amz-SignedHeaders", "content-type;host")

	canonicalQueryString := queryParams.Encode()
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\n", contentType, host)
	signedHeaders := "content-type;host"

	canonicalRequest := strings.Join([]string{
		"PUT",
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		"UNSIGNED-PAYLOAD",
	}, "\n")

	reqHash := sha256Hex([]byte(canonicalRequest))

	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		reqHash,
	}, "\n")

	signingKey := getSignatureKey(s.cfg.SecretAccessKey, dateStamp, s.cfg.Region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	queryParams.Set("X-Amz-Signature", signature)

	uploadURL := fmt.Sprintf("%s%s?%s", s.cfg.Endpoint, canonicalURI, queryParams.Encode())
	headers := map[string]string{
		"Content-Type": contentType,
	}

	return uploadURL, headers, nil
}

func (s *S3Provider) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error) {
	endpointURL, err := url.Parse(s.cfg.Endpoint)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	canonicalURI := fmt.Sprintf("/%s/%s", s.cfg.Bucket, strings.TrimPrefix(key, "/"))
	targetURL := fmt.Sprintf("%s%s", s.cfg.Endpoint, canonicalURI)

	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	payloadHash := sha256Hex(bodyBytes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, targetURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Host", endpointURL.Host)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		contentType, endpointURL.Host, payloadHash, amzDate)

	canonicalRequest := strings.Join([]string{
		http.MethodPut,
		canonicalURI,
		"",
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, s.cfg.Region)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := getSignatureKey(s.cfg.SecretAccessKey, dateStamp, s.cfg.Region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.cfg.AccessKeyID, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute S3 put request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("S3 PUT failed with status %d: %s", resp.StatusCode, string(body))
	}

	return s.PublicURL(key), nil
}

func (s *S3Provider) GetObject(ctx context.Context, key string) (io.ReadCloser, string, error) {
	req, err := s.newSignedRequest(ctx, http.MethodGet, key)
	if err != nil {
		return nil, "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to retrieve S3 object: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, "", errors.New("object not found")
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("S3 GET failed with status %d", resp.StatusCode)
	}

	return resp.Body, resp.Header.Get("Content-Type"), nil
}

func (s *S3Provider) DeleteObject(ctx context.Context, key string) error {
	req, err := s.newSignedRequest(ctx, http.MethodDelete, key)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete S3 object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("S3 DELETE returned unexpected status %d", resp.StatusCode)
	}

	return nil
}

// newSignedRequest signs empty-payload S3 requests used to read and remove private objects.
func (s *S3Provider) newSignedRequest(ctx context.Context, method, key string) (*http.Request, error) {
	endpointURL, err := url.Parse(s.cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid S3 endpoint URL: %w", err)
	}
	objectKey := strings.TrimLeft(key, "/")
	segments := strings.Split(objectKey, "/")
	encodedKey := make([]string, 0, len(segments))
	for _, segment := range segments {
		encodedKey = append(encodedKey, url.PathEscape(segment))
	}
	canonicalURI := "/" + url.PathEscape(s.cfg.Bucket) + "/" + strings.Join(encodedKey, "/")
	endpointURL.Path = strings.TrimRight(endpointURL.Path, "/") + canonicalURI
	endpointURL.RawPath = ""
	date := time.Now().UTC()
	dateStamp := date.Format("20060102")
	amzDate := date.Format("20060102T150405Z")
	payloadHash := sha256Hex(nil)
	req, err := http.NewRequestWithContext(ctx, method, endpointURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n", endpointURL.Host, payloadHash, amzDate)
	canonicalRequest := strings.Join([]string{method, canonicalURI, "", canonicalHeaders, signedHeaders, payloadHash}, "\n")
	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, s.cfg.Region)
	stringToSign := strings.Join([]string{"AWS4-HMAC-SHA256", amzDate, credentialScope, sha256Hex([]byte(canonicalRequest))}, "\n")
	signingKey := getSignatureKey(s.cfg.SecretAccessKey, dateStamp, s.cfg.Region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", s.cfg.AccessKeyID, credentialScope, signedHeaders, signature))
	return req, nil
}

func (s *S3Provider) PublicURL(key string) string {
	if s.cfg.PublicBaseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.cfg.PublicBaseURL, "/"), strings.TrimPrefix(key, "/"))
	}
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(s.cfg.Endpoint, "/"), s.cfg.Bucket, strings.TrimPrefix(key, "/"))
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key []byte, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func getSignatureKey(secretKey, dateStamp, regionName, serviceName string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(regionName))
	kService := hmacSHA256(kRegion, []byte(serviceName))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}
