package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
)

type Client interface {
	GetDeviceState(ctx context.Context, deviceID string) (*DeviceState, error)
	DeviceCache(deviceID string) *DeviceState
	GetProvider(deviceIDs ...string) api.ProviderFunc
}

type ClientImpl struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
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

	c.cache[deviceID] = data
	return data, nil
}

func (c *ClientImpl) DeviceCache(deviceID string) *DeviceState {
	return c.cache[deviceID]
}

func (c *ClientImpl) GetProvider(deviceIDs ...string) api.ProviderFunc {
	return func(ctx context.Context) (*api.DataPoint, error) {
		data := &api.DataPoint{
			Table: "home-assistant",
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
			data.Fields[state.Attributes.FriendlyName] = state.State
		}
		return data, nil
	}
}
