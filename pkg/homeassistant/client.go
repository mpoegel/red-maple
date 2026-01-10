package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type Client interface {
	GetDeviceState(ctx context.Context, deviceID string) (*DeviceState, error)
	DeviceCache(deviceID string) *DeviceState
}

type ClientImpl struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
	cache      map[string]*DeviceState
}

var _ Client = (*ClientImpl)(nil)

func NewClient(endpoint string, apiKey string) *ClientImpl {
	return &ClientImpl{
		httpClient: http.DefaultClient,
		endpoint:   endpoint,
		apiKey:     apiKey,
		cache:      map[string]*DeviceState{},
	}
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
