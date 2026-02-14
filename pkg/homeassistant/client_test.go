package homeassistant_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	homeassistant "github.com/mpoegel/red-maple/pkg/homeassistant"
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

func TestGetDeviceState_Success(t *testing.T) {
	state := homeassistant.DeviceState{
		EntityID: "sensor.outdoor_temp",
		State:    "72.5",
	}
	state.Attributes.FriendlyName = "Outdoor Temperature"
	state.Attributes.Unit = "°F"
	state.Attributes.StateClass = "measurement"
	state.LastChanged = time.Now()
	state.LastReported = time.Now()
	state.LastUpdated = time.Now()
	state.Context.ID = "context-1"

	jsonState, _ := json.Marshal(state)

	mt := &mockTransport{
		responseBody: jsonState,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := homeassistant.NewClient(
		"http://localhost:8123",
		"test-api-key",
		homeassistant.WithHTTPClient(&http.Client{Transport: mt}),
	)

	result, err := client.GetDeviceState(t.Context(), "sensor.outdoor_temp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EntityID != "sensor.outdoor_temp" {
		t.Errorf("expected entity_id 'sensor.outdoor_temp', got %s", result.EntityID)
	}
	if result.State != "72.5" {
		t.Errorf("expected state '72.5', got %s", result.State)
	}
	if result.Attributes.FriendlyName != "Outdoor Temperature" {
		t.Errorf("expected friendly_name 'Outdoor Temperature', got %s", result.Attributes.FriendlyName)
	}
	if result.Attributes.Unit != "°F" {
		t.Errorf("expected unit '°F', got %s", result.Attributes.Unit)
	}

	cached := client.DeviceCache("sensor.outdoor_temp")
	if cached == nil {
		t.Error("expected device to be cached")
	}
	if cached.EntityID != "sensor.outdoor_temp" {
		t.Errorf("expected cached entity_id 'sensor.outdoor_temp', got %s", cached.EntityID)
	}
}

func TestGetDeviceState_HTTPError(t *testing.T) {
	mt := &mockTransport{
		err:        io.EOF,
		statusCode: 500,
	}
	client := homeassistant.NewClient(
		"http://localhost:8123",
		"test-api-key",
		homeassistant.WithHTTPClient(&http.Client{Transport: mt}),
	)

	_, err := client.GetDeviceState(t.Context(), "sensor.outdoor_temp")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetDeviceState_InvalidJSON(t *testing.T) {
	mt := &mockTransport{
		responseBody: []byte("not json"),
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := homeassistant.NewClient(
		"http://localhost:8123",
		"test-api-key",
		homeassistant.WithHTTPClient(&http.Client{Transport: mt}),
	)

	_, err := client.GetDeviceState(t.Context(), "sensor.outdoor_temp")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetDeviceState_404NotFound(t *testing.T) {
	errorResp := map[string]any{
		"message": "Entity not found: sensor.nonexistent",
	}
	jsonResp, _ := json.Marshal(errorResp)

	mt := &mockTransport{
		responseBody: jsonResp,
		statusCode:   404,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := homeassistant.NewClient(
		"http://localhost:8123",
		"test-api-key",
		homeassistant.WithHTTPClient(&http.Client{Transport: mt}),
	)

	_, err := client.GetDeviceState(t.Context(), "sensor.nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeviceCache_Hit(t *testing.T) {
	preloadedCache := map[string]*homeassistant.DeviceState{
		"sensor.outdoor_temp": {
			EntityID: "sensor.outdoor_temp",
			State:    "72.5",
		},
	}
	client := homeassistant.NewClient(
		"http://localhost:8123",
		"test-api-key",
		homeassistant.WithCache(preloadedCache),
	)

	result := client.DeviceCache("sensor.outdoor_temp")
	if result == nil {
		t.Fatal("expected cached device, got nil")
	}
	if result.State != "72.5" {
		t.Errorf("expected state '72.5', got %s", result.State)
	}
}

func TestDeviceCache_Miss(t *testing.T) {
	preloadedCache := map[string]*homeassistant.DeviceState{
		"sensor.outdoor_temp": {
			EntityID: "sensor.outdoor_temp",
			State:    "72.5",
		},
	}
	client := homeassistant.NewClient(
		"http://localhost:8123",
		"test-api-key",
		homeassistant.WithCache(preloadedCache),
	)

	result := client.DeviceCache("sensor.indoor_temp")
	if result != nil {
		t.Errorf("expected nil for cache miss, got %v", result)
	}
}

func TestGetProvider_Success(t *testing.T) {
	states := []homeassistant.DeviceState{
		{
			EntityID: "sensor.outdoor_temp",
			State:    "72.5",
		},
		{
			EntityID: "sensor.outdoor_humidity",
			State:    "65",
		},
	}
	states[0].Attributes.FriendlyName = "Outdoor Temperature"
	states[1].Attributes.FriendlyName = "Outdoor Humidity"

	mt := &mockTransport{
		responseBody: mustJSON(states[0]),
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	mt2 := &mockTransport{
		responseBody: mustJSON(states[1]),
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}

	callCount := 0
	client := homeassistant.NewClient(
		"http://localhost:8123",
		"test-api-key",
		homeassistant.WithHTTPClient(&http.Client{
			Transport: &multiTransport{
				transports: []*mockTransport{mt, mt2},
				index:      &callCount,
			},
		}),
	)

	provider := client.GetProvider("sensor.outdoor_temp", "sensor.outdoor_humidity")
	data, err := provider(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Table != "home-assistant" {
		t.Errorf("expected table 'home-assistant', got %s", data.Table)
	}
	if data.Tags["location"] != "home" {
		t.Errorf("expected location tag 'home', got %s", data.Tags["location"])
	}
	if data.Fields["Outdoor Temperature"] != "72.5" {
		t.Errorf("expected 'Outdoor Temperature' = '72.5', got %v", data.Fields["Outdoor Temperature"])
	}
	if data.Fields["Outdoor Humidity"] != "65" {
		t.Errorf("expected 'Outdoor Humidity' = '65', got %v", data.Fields["Outdoor Humidity"])
	}
	if data.Stamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestGetProvider_OneDeviceFails(t *testing.T) {
	goodState := homeassistant.DeviceState{
		EntityID: "sensor.outdoor_temp",
		State:    "72.5",
	}
	goodState.Attributes.FriendlyName = "Outdoor Temperature"

	mt := &mockTransport{
		err:        io.EOF,
		statusCode: 500,
	}
	mt2 := &mockTransport{
		responseBody: mustJSON(goodState),
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}

	callCount := 0
	client := homeassistant.NewClient(
		"http://localhost:8123",
		"test-api-key",
		homeassistant.WithHTTPClient(&http.Client{
			Transport: &multiTransport{
				transports: []*mockTransport{mt, mt2},
				index:      &callCount,
			},
		}),
	)

	provider := client.GetProvider("sensor.bad_device", "sensor.outdoor_temp")
	data, err := provider(t.Context())
	if err != nil {
		t.Fatalf("expected no error (continues on failure), got %v", err)
	}
	if data.Fields["Outdoor Temperature"] != "72.5" {
		t.Errorf("expected 'Outdoor Temperature' = '72.5', got %v", data.Fields["Outdoor Temperature"])
	}
	if _, ok := data.Fields["sensor.bad_device"]; ok {
		t.Error("expected bad_device to not be in fields")
	}
}

func TestGetProvider_AllDevicesFail(t *testing.T) {
	mt := &mockTransport{
		err:        io.EOF,
		statusCode: 500,
	}

	callCount := 0
	client := homeassistant.NewClient(
		"http://localhost:8123",
		"test-api-key",
		homeassistant.WithHTTPClient(&http.Client{
			Transport: &multiTransport{
				transports: []*mockTransport{mt},
				index:      &callCount,
			},
		}),
	)

	provider := client.GetProvider("sensor.bad_device")
	data, err := provider(t.Context())
	if err != nil {
		t.Fatalf("expected no error (continues on failure), got %v", err)
	}
	if len(data.Fields) != 0 {
		t.Errorf("expected no fields, got %v", data.Fields)
	}
}

func TestGetProvider_SingleDevice(t *testing.T) {
	state := homeassistant.DeviceState{
		EntityID: "sensor.indoor_temp",
		State:    "70.0",
	}
	state.Attributes.FriendlyName = "Indoor Temperature"

	mt := &mockTransport{
		responseBody: mustJSON(state),
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}

	client := homeassistant.NewClient(
		"http://localhost:8123",
		"test-api-key",
		homeassistant.WithHTTPClient(&http.Client{Transport: mt}),
	)

	provider := client.GetProvider("sensor.indoor_temp")
	data, err := provider(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Fields["Indoor Temperature"] != "70.0" {
		t.Errorf("expected 'Indoor Temperature' = '70.0', got %v", data.Fields["Indoor Temperature"])
	}
}

func TestGetDeviceState_UpdatesCache(t *testing.T) {
	states := []homeassistant.DeviceState{
		{EntityID: "sensor.temp", State: "70"},
		{EntityID: "sensor.temp", State: "75"},
	}

	mt := &mockTransport{
		responseBody: mustJSON(states[0]),
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	mt2 := &mockTransport{
		responseBody: mustJSON(states[1]),
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}

	callCount := 0
	client := homeassistant.NewClient(
		"http://localhost:8123",
		"test-api-key",
		homeassistant.WithHTTPClient(&http.Client{
			Transport: &multiTransport{
				transports: []*mockTransport{mt, mt2},
				index:      &callCount,
			},
		}),
	)

	_, err := client.GetDeviceState(t.Context(), "sensor.temp")
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	cached1 := client.DeviceCache("sensor.temp")
	if cached1.State != "70" {
		t.Errorf("expected cached state '70', got %s", cached1.State)
	}

	_, err = client.GetDeviceState(t.Context(), "sensor.temp")
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	cached2 := client.DeviceCache("sensor.temp")
	if cached2.State != "75" {
		t.Errorf("expected updated cached state '75', got %s", cached2.State)
	}
}

type multiTransport struct {
	transports []*mockTransport
	index      *int
}

func (m *multiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if *m.index >= len(m.transports) {
		return nil, io.EOF
	}
	resp, err := m.transports[*m.index].RoundTrip(req)
	*m.index++
	return resp, err
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
