package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
)

type Client interface {
	GetDeviceState(ctx context.Context, deviceID string) (*DeviceState, error)
	DeviceCache(deviceID string) *DeviceState
	GetProvider(deviceIDs ...string) api.ProviderFunc
	GetDeviceHistory(ctx context.Context, importer api.Importer, deviceID string, duration time.Duration) ([]DeviceHistory, error)
}

type ClientImpl struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
	mu         sync.RWMutex
	cache      map[string]*DeviceState
}

var _ Client = (*ClientImpl)(nil)

type Option func(*ClientImpl)

func WithHTTPClient(client *http.Client) Option {
	return func(c *ClientImpl) {
		c.httpClient = client
	}
}

func WithCache(cache map[string]*DeviceState) Option {
	return func(c *ClientImpl) {
		c.cache = cache
	}
}

func NewClient(endpoint string, apiKey string, opts ...Option) *ClientImpl {
	c := &ClientImpl{
		httpClient: http.DefaultClient,
		endpoint:   endpoint,
		apiKey:     apiKey,
		cache:      map[string]*DeviceState{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *ClientImpl) GetDeviceState(ctx context.Context, deviceID string) (*DeviceState, error) {
	slog.Debug("getting device state", "deviceID", deviceID)

	uri := fmt.Sprintf("%s/api/states/%s", c.endpoint, deviceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Add("content-type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	data := &DeviceState{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(data); err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[deviceID] = data
	c.mu.Unlock()
	return data, nil
}

func (c *ClientImpl) DeviceCache(deviceID string) *DeviceState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache[deviceID]
}

func (c *ClientImpl) GetProvider(deviceIDs ...string) api.ProviderFunc {
	return func(ctx context.Context) (*api.DataPoint, error) {
		data := &api.DataPoint{
			Table: tableName,
			Tags: map[api.DataTag]string{
				api.LocationTag: "home",
			},
			Fields: map[string]any{},
			Stamp:  time.Now(),
		}
		for _, deviceID := range deviceIDs {
			state, err := c.GetDeviceState(ctx, deviceID)
			if err != nil {
				slog.Warn("failed to capture device state", "deviceID", deviceID, "err", err)
				continue
			}
			data.Fields[deviceID] = state.State
		}
		slog.Debug("exporting device devices", "data", data)
		return data, nil
	}
}

func (c *ClientImpl) GetDeviceHistory(ctx context.Context, importer api.Importer, deviceID string, duration time.Duration) ([]DeviceHistory, error) {
	rows, err := importer.QueryRange(ctx, tableName, duration)
	if err != nil {
		return nil, err
	}

	var results []DeviceHistory
	for _, row := range rows {
		slog.Debug("parsing device row", "row", row)

		var reading float64
		if readingVal, ok := row.Fields[deviceID]; ok {
			switch v := readingVal.(type) {
			case int64:
				reading = float64(v)
			case float64:
				reading = float64(v)
			case string:
				reading, err = strconv.ParseFloat(v, 64)
				if err != nil {
					slog.Warn("cannot parse device state", "err", err)
				}
			default:
				slog.Warn("unknown device state type", "type", fmt.Sprintf("%T", readingVal))
			}
		} else {
			slog.Warn("device not found", "id", deviceID)
			continue
		}

		results = append(results, DeviceHistory{
			Value: reading,
			Stamp: row.Stamp,
		})
	}

	return results, nil
}
