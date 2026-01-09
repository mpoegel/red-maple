package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const (
	url          = "https://api.openweathermap.org/data/3.0/onecall"
	defaultUnits = "imperial"
	weatherTTL   = 5 * time.Minute
)

type Client interface {
	GetWeather(ctx context.Context) (*WeatherData, error)
}

type ClientImpl struct {
	httpClient *http.Client
	lat        float64
	lon        float64
	apiKey     string

	lastData   *WeatherData
	lastUpdate time.Time
}

var _ Client = (*ClientImpl)(nil)

func NewClient(lat, lon float64, apiKey string) *ClientImpl {
	return &ClientImpl{
		httpClient: http.DefaultClient,
		lat:        lat,
		lon:        lon,
		apiKey:     apiKey,
	}
}

func (c *ClientImpl) GetWeather(ctx context.Context) (*WeatherData, error) {
	slog.Debug("getting weather")
	if c.lastData != nil && time.Since(c.lastUpdate) < weatherTTL {
		slog.Debug("using cached weather data")
		return c.lastData, nil
	}

	uri := fmt.Sprintf("%s?lat=%f&lon=%f&appid=%s&units=%s", url, c.lat, c.lon, c.apiKey, defaultUnits)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data := &WeatherData{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(data); err != nil {
		return nil, err
	}

	c.lastData = data
	c.lastUpdate = time.Now()
	return data, nil
}
