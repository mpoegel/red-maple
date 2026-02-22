package nycdata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const (
	defaultDomain     = "data.cityofnewyork.us"
	defaultPageSize   = 1000
	bicycleCountsID   = "uczf-rk3c"
	cacheTTL          = 1 * time.Hour
	fsCacheTTL        = 24 * time.Hour
	defaultFSCacheDir = "/var/cache/red-maple/nycdata"
	dateFormat        = "2006-01-02"
)

type CounterID int

const (
	BrooklynBridgeCounterID     CounterID = 300020904
	ManhattanBridgeCounterID    CounterID = 100062893
	WilliamsburgBridgeCounterID CounterID = 100009427
	QueensboroBridgeCounterID   CounterID = 100009428
)

type Client interface {
	GetBicycleCounts(ctx context.Context, counterID CounterID, opts ...QueryOption) ([]BicycleCount, error)
}

type ClientImpl struct {
	httpClient *http.Client
	domain     string
	appToken   string
	fsCacheDir string

	cache   map[string]*bicycleCountCache
	cacheMu sync.RWMutex
}

type bicycleCountCache struct {
	data       []BicycleCount
	lastUpdate time.Time
}

var _ Client = (*ClientImpl)(nil)

type Option func(*ClientImpl)

func WithHTTPClient(client *http.Client) Option {
	return func(c *ClientImpl) {
		c.httpClient = client
	}
}

func WithAppToken(token string) Option {
	return func(c *ClientImpl) {
		c.appToken = token
	}
}

func WithFilesystemCache(dir string) Option {
	return func(c *ClientImpl) {
		c.fsCacheDir = dir
	}
}

func NewClient(opts ...Option) *ClientImpl {
	c := &ClientImpl{
		httpClient: http.DefaultClient,
		domain:     defaultDomain,
		fsCacheDir: defaultFSCacheDir,
		cache:      make(map[string]*bicycleCountCache),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type QueryRequest struct {
	Query string `json:"query"`
	Page  *Page  `json:"page,omitempty"`
}

type Page struct {
	PageNumber int `json:"pageNumber"`
	PageSize   int `json:"pageSize"`
}

type bicycleCount struct {
	CountID string `json:"countid"`
	ID      string `json:"id"`
	Date    string `json:"date"`
	Counts  string `json:"counts"`
	Status  string `json:"status"`
}

type BicycleCount struct {
	CountID int       `json:"countid"`
	ID      int       `json:"id"`
	Date    time.Time `json:"date"`
	Counts  int       `json:"counts"`
	Status  int       `json:"status"`
}

type QueryOption func(*queryOptions)

type queryOptions struct {
	startDate time.Time
	endDate   time.Time
	pageSize  int
}

func WithDateRange(start, end time.Time) QueryOption {
	return func(o *queryOptions) {
		o.startDate = start
		o.endDate = end
	}
}

func WithPageSize(size int) QueryOption {
	return func(o *queryOptions) {
		o.pageSize = size
	}
}

func (c *ClientImpl) GetBicycleCounts(ctx context.Context, counterID CounterID, opts ...QueryOption) ([]BicycleCount, error) {

	queryOpts := &queryOptions{
		pageSize: defaultPageSize,
	}
	for _, opt := range opts {
		opt(queryOpts)
	}
	slog.Debug("getting bicycle counts", "counterID", counterID, "from", queryOpts.startDate)

	cacheKey := c.makeCacheKey(counterID, queryOpts)

	c.cacheMu.RLock()
	if cached, ok := c.cache[cacheKey]; ok && time.Since(cached.lastUpdate) < cacheTTL {
		slog.Debug("using in-memory cached bicycle counts", "counterID", counterID)
		c.cacheMu.RUnlock()
		return cached.data, nil
	}
	c.cacheMu.RUnlock()

	if c.fsCacheDir != "" {
		if data, cachedAt, err := c.readFSCache(cacheKey); err == nil {
			age := time.Since(cachedAt)
			if age < fsCacheTTL {
				slog.Debug("using filesystem cached bicycle counts", "counterID", counterID, "age", age)
				c.cacheMu.Lock()
				c.cache[cacheKey] = &bicycleCountCache{
					data:       data,
					lastUpdate: time.Now(),
				}
				c.cacheMu.Unlock()
				return data, nil
			}

			slog.Debug("filesystem cache stale, returning stale data and async refreshing", "counterID", counterID, "age", age)
			c.cacheMu.Lock()
			c.cache[cacheKey] = &bicycleCountCache{
				data:       data,
				lastUpdate: cachedAt,
			}
			c.cacheMu.Unlock()

			go func() {
				if err := c.asyncRefreshCache(ctx, counterID, queryOpts); err != nil {
					slog.Error("async cache refresh failed", "counterID", counterID, "error", err)
				}
			}()

			return data, nil
		}
	}

	slog.Debug("fetching bicycle counts from API", "counterID", counterID)
	allCounts, err := c.fetchAllCounts(ctx, counterID, queryOpts)
	if err != nil {
		return nil, err
	}

	if c.fsCacheDir != "" {
		if err := c.writeFSCache(cacheKey, allCounts); err != nil {
			slog.Warn("failed to write filesystem cache", "error", err)
		}
	}

	c.cacheMu.Lock()
	c.cache[cacheKey] = &bicycleCountCache{
		data:       allCounts,
		lastUpdate: time.Now(),
	}
	c.cacheMu.Unlock()

	return allCounts, nil
}

func (c *ClientImpl) fetchAllCounts(ctx context.Context, counterID CounterID, queryOpts *queryOptions) ([]BicycleCount, error) {
	var allCounts []BicycleCount
	pageNum := 1

	for {
		counts, hasMore, err := c.fetchPage(ctx, counterID, pageNum, queryOpts)
		if err != nil {
			return nil, err
		}

		allCounts = append(allCounts, counts...)

		if !hasMore || len(counts) == 0 {
			break
		}

		pageNum++
	}

	return allCounts, nil
}

func (c *ClientImpl) asyncRefreshCache(ctx context.Context, counterID CounterID, queryOpts *queryOptions) error {
	slog.Debug("async refreshing bicycle counts cache", "counterID", counterID)

	cacheKey := c.makeCacheKey(counterID, queryOpts)

	allCounts, err := c.fetchAllCounts(ctx, counterID, queryOpts)
	if err != nil {
		return err
	}

	if err := c.writeFSCache(cacheKey, allCounts); err != nil {
		return err
	}

	c.cacheMu.Lock()
	c.cache[cacheKey] = &bicycleCountCache{
		data:       allCounts,
		lastUpdate: time.Now(),
	}
	c.cacheMu.Unlock()

	slog.Debug("async cache refresh completed", "counterID", counterID, "count", len(allCounts))
	return nil
}

func (c *ClientImpl) makeCacheKey(counterID CounterID, opts *queryOptions) string {
	startStr := opts.startDate.Format(dateFormat)
	endStr := opts.endDate.Format(dateFormat)
	return fmt.Sprintf("%d_%s_%s", counterID, startStr, endStr)
}

func (c *ClientImpl) fetchPage(ctx context.Context, counterID CounterID, pageNum int, opts *queryOptions) ([]BicycleCount, bool, error) {
	query := fmt.Sprintf("SELECT * WHERE id = %d", counterID)

	if !opts.startDate.IsZero() && !opts.endDate.IsZero() {
		query += fmt.Sprintf(" AND date >= '%s' AND date <= '%s'",
			opts.startDate.Format(dateFormat), opts.endDate.Format(dateFormat))
	}

	reqBody := QueryRequest{
		Query: query,
		Page: &Page{
			PageNumber: pageNum,
			PageSize:   opts.pageSize,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, false, err
	}

	uri := fmt.Sprintf("https://%s/api/v3/views/%s/query.json", c.domain, bicycleCountsID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.appToken != "" {
		req.Header.Set("X-App-Token", c.appToken)
	}

	slog.Debug("prepared nycdata request", "req", *req, "body", body)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("HTTP error: %s: %s", resp.Status, body)
	}

	var data []bicycleCount
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&data); err != nil {
		return nil, false, err
	}

	hasMore := len(data) == opts.pageSize
	return parseBicycleCounts(data), hasMore, nil
}

func parseBicycleCounts(counts []bicycleCount) []BicycleCount {
	res := make([]BicycleCount, len(counts))
	for i, bc := range counts {
		res[i].CountID, _ = strconv.Atoi(bc.CountID)
		res[i].ID, _ = strconv.Atoi(bc.ID)
		res[i].Counts, _ = strconv.Atoi(bc.Counts)
		res[i].Date, _ = time.Parse(time.RFC3339, bc.Date)
		res[i].Status, _ = strconv.Atoi(bc.Status)
	}
	return res
}

type fsCacheData struct {
	Data     []BicycleCount `json:"data"`
	CachedAt time.Time      `json:"cached_at"`
}

func (c *ClientImpl) fsCacheFilePath(key string) string {
	return filepath.Join(c.fsCacheDir, fmt.Sprintf("%s.json", key))
}

func (c *ClientImpl) readFSCache(key string) ([]BicycleCount, time.Time, error) {
	path := c.fsCacheFilePath(key)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, time.Time{}, err
	}

	var cache fsCacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, time.Time{}, err
	}

	return cache.Data, cache.CachedAt, nil
}

func (c *ClientImpl) writeFSCache(key string, data []BicycleCount) error {
	path := c.fsCacheFilePath(key)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	cache := fsCacheData{
		Data:     data,
		CachedAt: time.Now(),
	}

	body, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(path, body, 0644)
}
