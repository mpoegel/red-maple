package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
)

var _ api.DataExporter = (*Client)(nil)
var _ api.Importer = (*Client)(nil)

// Client is an S3-based time series data store that stores data in JSON Lines format.
// It implements both the api.DataExporter and api.Importer interfaces.
// Data is partitioned by table and hour: {bucket}/{table}/year/month/day/hour.jsonl
type Client struct {
	signer        *Signer
	endpoint      string
	scheme        string
	bucket        string
	region        string
	accessKey     string
	secretKey     string
	retentionDays int
	buffer        buffer
	mu            sync.Mutex
	httpClient    *http.Client
}

// buffer holds data points waiting to be flushed to S3.
type buffer struct {
	points   []*api.DataPoint
	flushed  time.Time
	interval time.Duration
}

// WithHTTPClient sets a custom HTTP client for making S3 requests.
// This allows configuration of timeouts, TLS settings, and proxy behavior.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// NewClient creates a new S3 client with the given configuration.
// It returns an error if credentials are not provided.
// Optional functional options can be provided to customize the client.
func NewClient(opts ...ClientOption) (*Client, error) {
	client := &Client{
		scheme:        "https",
		region:        "us-east-1",
		retentionDays: 30,
		httpClient:    http.DefaultClient,
	}
	client.buffer.interval = 1 * time.Minute

	for _, opt := range opts {
		opt(client)
	}

	if client.accessKey == "" || client.secretKey == "" {
		return nil, errors.New("s3 credentials are required")
	}

	if client.endpoint == "" {
		client.endpoint = fmt.Sprintf("s3.%s.amazonaws.com", client.region)
	}

	if client.bucket == "" {
		return nil, errors.New("s3 bucket is required")
	}

	client.signer = NewSigner(client.accessKey, client.secretKey, client.region)

	return client, nil
}

// Close flushes any buffered data to S3 and closes the client.
// It implements the io.Closer interface.
func (c *Client) Close() error {
	if err := c.flush(context.Background()); err != nil {
		return err
	}
	return nil
}

// Export adds data points to the internal buffer and flushes to S3 when
// the buffer reaches 100 points or after the flush interval (1 minute).
// It implements the api.DataExporter interface.
func (c *Client) Export(ctx context.Context, dataPoints []*api.DataPoint) error {
	if len(dataPoints) == 0 {
		return nil
	}

	c.mu.Lock()
	c.buffer.points = append(c.buffer.points, dataPoints...)

	shouldFlush := len(c.buffer.points) >= 100 ||
		time.Since(c.buffer.flushed) >= c.buffer.interval
	c.mu.Unlock()

	if shouldFlush {
		return c.flush(ctx)
	}

	return nil
}

// flush writes all buffered data points to S3, grouped by object key.
// This method is thread-safe and acquires the mutex before accessing the buffer.
func (c *Client) flush(ctx context.Context) error {
	c.mu.Lock()
	if len(c.buffer.points) == 0 {
		c.mu.Unlock()
		return nil
	}

	points := c.buffer.points
	c.buffer.points = nil
	c.buffer.flushed = time.Now()
	c.mu.Unlock()

	grouped := make(map[string][]*api.DataPoint)
	for _, p := range points {
		key := c.getObjectKey(p.Stamp, p.Table)
		grouped[key] = append(grouped[key], p)
	}

	for key, pts := range grouped {
		if err := c.appendToObject(ctx, key, pts); err != nil {
			slog.Warn("failed to append to s3 object", "key", key, "error", err)
			return err
		}
	}

	return nil
}

// getObjectKey generates the S3 object key for a given time and table.
// Format: {bucket}/{table}/{year}/{month}/{day}/{hour}.jsonl
func (c *Client) getObjectKey(t time.Time, table string) string {
	utc := t.UTC()
	return path.Join(
		c.bucket,
		table,
		fmt.Sprintf("%d", utc.Year()),
		fmt.Sprintf("%02d", utc.Month()),
		fmt.Sprintf("%02d", utc.Day()),
		fmt.Sprintf("%02d.jsonl", utc.Hour()),
	)
}

// appendToObject reads an existing object, appends new data points as JSON Lines,
// and writes the result back to S3.
func (c *Client) appendToObject(ctx context.Context, key string, points []*api.DataPoint) error {
	existingContent, err := c.getObject(ctx, key)
	if err != nil && !isNotFound(err) {
		return err
	}

	var content bytes.Buffer
	if len(existingContent) > 0 {
		content.Write(existingContent)
		if !bytes.HasSuffix(existingContent, []byte("\n")) {
			content.WriteString("\n")
		}
	}

	for _, p := range points {
		line, err := json.Marshal(p)
		if err != nil {
			return err
		}
		content.Write(line)
		content.WriteString("\n")
	}

	return c.putObject(ctx, key, content.Bytes())
}

// getObject retrieves an object's content from S3.
func (c *Client) getObject(ctx context.Context, key string) ([]byte, error) {
	req, err := c.newRequest(ctx, "GET", key, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &NotFoundError{Key: key}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("get s3 object failed", "status", resp.Status, "body", body)
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// putObject uploads data to an S3 object.
func (c *Client) putObject(ctx context.Context, key string, data []byte) error {
	req, err := c.newRequest(ctx, "PUT", key, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.doRequest(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// newRequest creates a new HTTP request for an S3 operation.
func (c *Client) newRequest(ctx context.Context, method, key string, body io.Reader) (*http.Request, error) {
	endpoint := c.endpoint
	u := &url.URL{
		Scheme: c.scheme,
		Host:   endpoint,
		Path:   "/" + key,
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Host = endpoint

	return req, nil
}

// doRequest signs and executes an HTTP request to S3.
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	now := time.Now()
	c.signer.SignRequest(req, now)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// QueryRange retrieves data points for a given table within a specified duration.
// The duration specifies how far back from now to query.
func (c *Client) QueryRange(ctx context.Context, table string, duration time.Duration) ([]*api.DataPoint, error) {
	now := time.Now().UTC()
	startTime := now.Add(-duration)

	keys := c.getObjectKeysForRange(table, startTime, now)

	var allResults []*api.DataPoint

	for _, key := range keys {
		data, err := c.getObject(ctx, key)
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, err
		}

		for line := range strings.SplitSeq(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			point, err := parseDataPoint(line)
			if err != nil {
				slog.Debug("failed to parse data point", "line", line, "error", err)
				continue
			}

			if !point.Stamp.Before(startTime) && point.Stamp.Before(now) {
				allResults = append(allResults, point)
			} else {
				slog.Debug("skipping", "point", point.Stamp)
			}
		}
	}

	return allResults, nil
}

// getObjectKeysForRange generates a list of object keys for each hour in a time range.
func (c *Client) getObjectKeysForRange(table string, start, end time.Time) []string {
	var keys []string

	current := start
	for current.Before(end) {
		keys = append(keys, c.getObjectKey(current, table))
		current = current.Add(1 * time.Hour)
	}

	return keys
}

type rawDataPoint struct {
	Table  string         `json:"Table"`
	Tags   map[string]any `json:"Tags"`
	Fields map[string]any `json:"Fields"`
	Stamp  any            `json:"Stamp"`
}

func parseDataPoint(line string) (*api.DataPoint, error) {
	var raw rawDataPoint
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	tags := make(map[api.DataTag]string)
	for k, v := range raw.Tags {
		if str, ok := v.(string); ok {
			tags[api.DataTag(k)] = str
		}
	}

	var stamp time.Time
	switch v := raw.Stamp.(type) {
	case string:
		stamp, _ = time.Parse(time.RFC3339, v)
	case float64:
		stamp = time.Unix(int64(v), 0).UTC()
	}

	return &api.DataPoint{
		Table:  raw.Table,
		Tags:   tags,
		Fields: raw.Fields,
		Stamp:  stamp,
	}, nil
}

// CleanupRetention deletes objects older than the configured retention period.
// It should be called periodically (e.g., daily) to enforce data retention policies.
func (c *Client) CleanupRetention(ctx context.Context) error {
	cutoff := time.Now().UTC().Add(-time.Duration(c.retentionDays) * 24 * time.Hour)
	var ContinuationToken *string
	for {
		objects, token, err := c.listObjects(ctx, ContinuationToken)
		if err != nil {
			return err
		}

		if len(objects) == 0 {
			break
		}

		var toDelete []string
		for _, obj := range objects {
			keyTime, err := time.Parse("2006/01/02/15.jsonl", strings.TrimPrefix(obj.Key, c.bucket+"/"))
			if err != nil {
				continue
			}

			if keyTime.Before(cutoff) {
				toDelete = append(toDelete, obj.Key)
			}
		}

		if len(toDelete) > 0 {
			if err := c.deleteObjects(ctx, toDelete); err != nil {
				return err
			}
		}

		if token == nil || *token == "" {
			break
		}
		ContinuationToken = token
	}

	return nil
}

// objectSummary represents the metadata of an S3 object.
type objectSummary struct {
	Key string
}

// listObjects lists objects in the bucket, supporting pagination via continuation token.
func (c *Client) listObjects(ctx context.Context, continuationToken *string) ([]objectSummary, *string, error) {
	u := &url.URL{
		Scheme: c.scheme,
		Host:   c.endpoint,
		Path:   "/" + c.bucket,
	}

	queryParams := u.Query()
	queryParams.Set("list-type", "2")
	if continuationToken != nil {
		queryParams.Set("continuation-token", *continuationToken)
	}
	u.RawQuery = queryParams.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Host = c.endpoint

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("list objects failed: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	objects, nextToken, err := parseListObjectsResponse(data)
	if err != nil {
		return nil, nil, err
	}

	return objects, nextToken, nil
}

// parseListObjectsResponse parses the XML response from S3 ListObjectsV2.
func parseListObjectsResponse(data []byte) ([]objectSummary, *string, error) {
	var result struct {
		Contents              []objectSummary `json:"Contents"`
		NextContinuationToken string          `json:"NextContinuationToken"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, nil, err
	}

	var token *string
	if result.NextContinuationToken != "" {
		token = &result.NextContinuationToken
	}

	return result.Contents, token, nil
}

// deleteObjects deletes multiple objects from S3 in a single request.
func (c *Client) deleteObjects(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	var deleteEntries []map[string]string
	for _, key := range keys {
		deleteEntries = append(deleteEntries, map[string]string{"Key": key})
	}

	deletePayload := map[string]any{
		"Objects": deleteEntries,
	}

	payloadBytes, err := json.Marshal(deletePayload)
	if err != nil {
		return err
	}

	u := &url.URL{
		Scheme: c.scheme,
		Host:   c.endpoint,
		Path:   "/" + c.bucket + "?delete",
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}

	req.Host = c.endpoint
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete objects failed: %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// NotFoundError is returned when an S3 object is not found.
type NotFoundError struct {
	Key string
}

// Error returns the error message for a NotFoundError.
func (e *NotFoundError) Error() string {
	return "not found: " + e.Key
}

// isNotFound checks if an error is a NotFoundError.
func isNotFound(err error) bool {
	var nfe *NotFoundError
	return errors.As(err, &nfe)
}
