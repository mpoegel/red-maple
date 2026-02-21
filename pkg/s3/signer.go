package s3

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	algorithm = "AWS4-HMAC-SHA256"
)

// Signer handles AWS Signature Version 4 signing for S3 API requests.
// It implements the AWS Signature Version 4 algorithm using only Go standard library.
type Signer struct {
	accessKey string
	secretKey string
	region    string
	service   string
}

// NewSigner creates a new Signer with the given AWS credentials and region.
func NewSigner(accessKey, secretKey, region string) *Signer {
	return &Signer{
		accessKey: accessKey,
		secretKey: secretKey,
		region:    region,
		service:   "s3",
	}
}

// SignRequest signs an HTTP request using AWS Signature Version 4.
// It adds the required Authorization, X-Amz-Date, X-Amz-Content-Sha256, and Host headers.
func (s *Signer) SignRequest(req *http.Request, now time.Time) {
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("Host", req.Host)
	req.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")

	canonicalURI := s.getCanonicalURI(req.URL.Path)

	canonicalQueryString := ""
	if req.URL.RawQuery != "" {
		canonicalQueryString = s.getCanonicalQueryString(req.URL)
	}

	canonicalHeaders := s.getCanonicalHeaders(req)
	signedHeaders := s.getSignedHeaders(req)

	hashedPayload := "UNSIGNED-PAYLOAD"

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedPayload,
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, s.region, s.service)

	stringToSign := strings.Join([]string{
		algorithm,
		amzDate,
		credentialScope,
		hashHex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := s.getSignatureKey(dateStamp)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, s.accessKey, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

// getCanonicalURI returns the canonical URI path for S3.
// For virtual-hosted style, this should be just the key (without bucket).
// The path is expected to be URL-encoded by Go's http package.
func (s *Signer) getCanonicalURI(path string) string {
	if path == "" {
		return "/"
	}
	canonicalPath := path
	if len(canonicalPath) > 0 && canonicalPath[0] == '/' {
		canonicalPath = canonicalPath[1:]
	}
	if canonicalPath == "" {
		return "/"
	}
	return "/" + canonicalPath
}

// getCanonicalQueryString encodes and sorts query string parameters.
// AWS Signature V4 requires specific encoding for query parameters.
func (s *Signer) getCanonicalQueryString(u *url.URL) string {
	queryParams := u.Query()
	encodedParams := make([]string, 0, len(queryParams))

	for k, v := range queryParams {
		key := percentEncode(k)
		for _, val := range v {
			encodedParams = append(encodedParams, key+"="+percentEncode(val))
		}
	}

	sort.Strings(encodedParams)
	return strings.Join(encodedParams, "&")
}

// percentEncode encodes a string per AWS Signature V4 requirements.
// This is different from url.QueryEscape - it encodes more characters.
func percentEncode(s string) string {
	var result strings.Builder
	for _, r := range s {
		if isUnreserved(r) {
			result.WriteRune(r)
		} else {
			result.WriteString(fmt.Sprintf("%%%02X", r))
		}
	}
	return result.String()
}

// getCanonicalHeaders builds the canonical headers string.
func (s *Signer) getCanonicalHeaders(req *http.Request) string {
	headers := make([]string, 0, len(req.Header))

	for k := range req.Header {
		headers = append(headers, strings.ToLower(k))
	}

	sort.Strings(headers)

	var canonicalHeaders strings.Builder
	for _, h := range headers {
		canonicalHeaders.WriteString(h)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(strings.TrimSpace(req.Header.Get(h)))
		canonicalHeaders.WriteString("\n")
	}

	return canonicalHeaders.String()
}

// getSignedHeaders returns the sorted, semicolon-separated list of signed header names.
func (s *Signer) getSignedHeaders(req *http.Request) string {
	headers := make([]string, 0, len(req.Header))

	for k := range req.Header {
		headers = append(headers, strings.ToLower(k))
	}

	sort.Strings(headers)
	return strings.Join(headers, ";")
}

// getSignatureKey derives the signing key from the secret key.
func (s *Signer) getSignatureKey(dateStamp string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+s.secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(s.region))
	kService := hmacSHA256(kRegion, []byte(s.service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

// hmacSHA256 computes HMAC-SHA256.
func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// hashHex computes the hexadecimal SHA-256 hash of data.
func hashHex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// isUnreserved returns true if a character is an unreserved character per RFC 3986.
func isUnreserved(r rune) bool {
	return (r >= 'A' && r <= 'Z') ||
		(r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_' || r == '.' || r == '~'
}
