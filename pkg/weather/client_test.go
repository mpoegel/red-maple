package weather_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	weather "github.com/mpoegel/red-maple/pkg/weather"
)

type mockTransport struct {
	responseBody []byte
	err          error
	callCount    int
	statusCode   int
	headers      http.Header
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	body := m.responseBody
	m.responseBody = bytes.Clone(body)
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     m.headers,
	}, nil
}

func TestGetWeather_Success(t *testing.T) {
	resp := weather.WeatherData{
		Latitude:  40.7128,
		Longitude: -74.0060,
		Timezone:  "America/New_York",
		Current: weather.Current{
			Temperature: 72.5,
			Humidity:    65,
		},
	}
	jsonResp, _ := json.Marshal(resp)

	mt := &mockTransport{
		responseBody: jsonResp,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := weather.NewClient(
		40.7128,
		-74.0060,
		"test-api-key",
		weather.WithHTTPClient(&http.Client{Transport: mt}),
		weather.WithBaseURL("http://redmaple.tree/"),
	)

	result, err := client.GetWeather(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Current.Temperature != 72.5 {
		t.Errorf("expected temperature 72.5, got %f", result.Current.Temperature)
	}
	if result.Current.Humidity != 65 {
		t.Errorf("expected humidity 65, got %d", result.Current.Humidity)
	}
}

func TestGetWeather_HTTPError(t *testing.T) {
	mt := &mockTransport{
		err:        io.EOF,
		statusCode: 500,
	}
	client := weather.NewClient(
		40.7128,
		-74.0060,
		"test-api-key",
		weather.WithHTTPClient(&http.Client{Transport: mt}),
		weather.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetWeather(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetWeather_InvalidJSON(t *testing.T) {
	mt := &mockTransport{
		responseBody: []byte("not json"),
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := weather.NewClient(
		40.7128,
		-74.0060,
		"test-api-key",
		weather.WithHTTPClient(&http.Client{Transport: mt}),
		weather.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetWeather(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetWeather_404Error(t *testing.T) {
	mt := &mockTransport{
		responseBody: []byte("Not Found"),
		statusCode:   404,
		headers:      http.Header{},
	}
	client := weather.NewClient(
		40.7128,
		-74.0060,
		"test-api-key",
		weather.WithHTTPClient(&http.Client{Transport: mt}),
		weather.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetWeather(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetWeather_Caching(t *testing.T) {
	resp := weather.WeatherData{
		Current: weather.Current{
			Temperature: 72.5,
		},
	}
	jsonResp, _ := json.Marshal(resp)

	mt := &mockTransport{
		responseBody: jsonResp,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := weather.NewClient(
		40.7128,
		-74.0060,
		"test-api-key",
		weather.WithHTTPClient(&http.Client{Transport: mt}),
		weather.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetWeather(t.Context())
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if mt.callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", mt.callCount)
	}

	_, err = client.GetWeather(t.Context())
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if mt.callCount != 1 {
		t.Errorf("expected 1 total call (cached), got %d", mt.callCount)
	}
}

func TestGetPollution_Success(t *testing.T) {
	resp := map[string]any{
		"coord": map[string]float64{
			"lat": 40.7128,
			"lon": -74.0060,
		},
		"list": []map[string]any{
			{
				"dt": 1234567890,
				"main": map[string]int{
					"aqi": 2,
				},
			},
		},
	}
	jsonResp, _ := json.Marshal(resp)

	mt := &mockTransport{
		responseBody: jsonResp,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := weather.NewClient(
		40.7128,
		-74.0060,
		"test-api-key",
		weather.WithHTTPClient(&http.Client{Transport: mt}),
		weather.WithBaseURL("http://redmaple.tree/"),
	)

	result, err := client.GetPollution(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 data point, got %d", len(result.Data))
	}
	if result.Data[0].Main.AQI != 2 {
		t.Errorf("expected AQI 2, got %d", result.Data[0].Main.AQI)
	}
}

func TestGetPollution_HTTPError(t *testing.T) {
	mt := &mockTransport{
		err:        io.EOF,
		statusCode: 500,
	}
	client := weather.NewClient(
		40.7128,
		-74.0060,
		"test-api-key",
		weather.WithHTTPClient(&http.Client{Transport: mt}),
		weather.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetPollution(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetPollution_InvalidJSON(t *testing.T) {
	mt := &mockTransport{
		responseBody: []byte("not json"),
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := weather.NewClient(
		40.7128,
		-74.0060,
		"test-api-key",
		weather.WithHTTPClient(&http.Client{Transport: mt}),
		weather.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetPollution(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetPollution_404Error(t *testing.T) {
	mt := &mockTransport{
		responseBody: []byte("Not Found"),
		statusCode:   404,
		headers:      http.Header{},
	}
	client := weather.NewClient(
		40.7128,
		-74.0060,
		"test-api-key",
		weather.WithHTTPClient(&http.Client{Transport: mt}),
		weather.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetPollution(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetPollution_Caching(t *testing.T) {
	resp := map[string]any{
		"list": []map[string]any{
			{
				"dt":   1234567890,
				"main": map[string]int{"aqi": 2},
			},
		},
	}
	jsonResp, _ := json.Marshal(resp)

	mt := &mockTransport{
		responseBody: jsonResp,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := weather.NewClient(
		40.7128,
		-74.0060,
		"test-api-key",
		weather.WithHTTPClient(&http.Client{Transport: mt}),
		weather.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetPollution(t.Context())
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if mt.callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", mt.callCount)
	}

	_, err = client.GetPollution(t.Context())
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if mt.callCount != 1 {
		t.Errorf("expected 1 total call (cached), got %d", mt.callCount)
	}
}
