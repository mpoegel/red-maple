package citibike_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	citibike "github.com/mpoegel/red-maple/pkg/citibike"
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

func TestGetVehicleTypes_Success(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"vehicle_types": []map[string]any{
				{"vehicle_type_id": "1", "propulsion_type": "human", "form_factor": "bike"},
				{"vehicle_type_id": "2", "propulsion_type": "electric", "form_factor": "bike"},
			},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	jsonResp, _ := json.Marshal(resp)

	mt := &mockTransport{
		responseBody: jsonResp,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	result, err := client.GetVehicleTypes(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data.VehicleTypes) != 2 {
		t.Errorf("expected 2 vehicle types, got %d", len(result.Data.VehicleTypes))
	}
	if result.Data.VehicleTypes[0].VehicleTypeID != "1" {
		t.Errorf("expected first vehicle type ID to be '1', got %s", result.Data.VehicleTypes[0].VehicleTypeID)
	}
}

func TestGetVehicleTypes_HTTPError(t *testing.T) {
	mt := &mockTransport{
		err:        io.EOF,
		statusCode: 500,
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetVehicleTypes(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetVehicleTypes_InvalidJSON(t *testing.T) {
	mt := &mockTransport{
		responseBody: []byte("not json"),
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetVehicleTypes(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetStationInformation_Success(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{
				{
					"station_id":  "station-1",
					"name":        "Test Station",
					"short_name":  "TS1",
					"lat":         40.7128,
					"lon":         -74.0060,
					"capacity":    20,
					"region_id":   "1",
					"rental_uris": map[string]string{"android": "app://1", "ios": "app://2"},
				},
			},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	jsonResp, _ := json.Marshal(resp)

	mt := &mockTransport{
		responseBody: jsonResp,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	result, err := client.GetStationInformation(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data.Stations) != 1 {
		t.Errorf("expected 1 station, got %d", len(result.Data.Stations))
	}
	if result.Data.Stations[0].Name != "Test Station" {
		t.Errorf("expected station name 'Test Station', got %s", result.Data.Stations[0].Name)
	}
}

func TestGetStationInformation_HTTPError(t *testing.T) {
	mt := &mockTransport{
		err:        io.EOF,
		statusCode: 500,
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetStationInformation(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetStationStatus_Success(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{
				{
					"station_id":           "station-1",
					"num_bikes_available":  5,
					"num_ebikes_available": 3,
					"num_docks_available":  12,
					"is_installed":         1,
					"is_renting":           1,
					"is_returning":         1,
					"last_reported":        1234567890,
				},
			},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	jsonResp, _ := json.Marshal(resp)

	mt := &mockTransport{
		responseBody: jsonResp,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	result, err := client.GetStationStatus(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data.Stations) != 1 {
		t.Errorf("expected 1 station, got %d", len(result.Data.Stations))
	}
}

func TestGetStationStatus_HTTPError(t *testing.T) {
	mt := &mockTransport{
		err:        io.EOF,
		statusCode: 500,
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetStationStatus(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetStationID_CacheHit(t *testing.T) {
	preloadedCache := map[string]citibike.StationInfo{
		"Test Station": {StationID: "station-123"},
	}
	client := citibike.NewClient(
		citibike.WithStationCache(preloadedCache),
	)

	id, err := client.GetStationID(t.Context(), "Test Station")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "station-123" {
		t.Errorf("expected 'station-123', got %s", id)
	}
}

func TestGetStationID_CacheMiss(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{
				{
					"station_id": "station-456",
					"name":       "New Station",
					"short_name": "NS1",
					"lat":        40.0,
					"lon":        -74.0,
					"capacity":   15,
				},
			},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	jsonResp, _ := json.Marshal(resp)

	mt := &mockTransport{
		responseBody: jsonResp,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	id, err := client.GetStationID(t.Context(), "New Station")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "station-456" {
		t.Errorf("expected 'station-456', got %s", id)
	}
	if mt.callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", mt.callCount)
	}
}

func TestGetStationID_NotFound(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	jsonResp, _ := json.Marshal(resp)

	mt := &mockTransport{
		responseBody: jsonResp,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetStationID(t.Context(), "Nonexistent Station")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetNumBikesAtStation_Success(t *testing.T) {
	infoResp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{
				{"station_id": "station-1", "name": "Test Station"},
			},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	infoJson, _ := json.Marshal(infoResp)

	statusResp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{
				{
					"station_id": "station-1",
					"vehicle_types_available": []map[string]any{
						{"vehicle_type_id": "1", "count": 5},
						{"vehicle_type_id": "2", "count": 3},
					},
				},
			},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	statusJson, _ := json.Marshal(statusResp)

	callCount := 0
	mt := &mockTransport{
		responseBody: statusJson,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	mt2 := &mockTransport{
		responseBody: infoJson,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}

	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{
			Transport: &multiTransport{
				transports: []*mockTransport{mt, mt2},
				index:      &callCount,
			},
		}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	classics, ebikes, err := client.GetNumBikesAtStation(t.Context(), "Test Station")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if classics != 5 {
		t.Errorf("expected 5 classics, got %d", classics)
	}
	if ebikes != 3 {
		t.Errorf("expected 3 ebikes, got %d", ebikes)
	}
}

func TestGetNumBikesAtStation_StationNotFound(t *testing.T) {
	infoResp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{
				{"station_id": "station-1", "name": "Test Station"},
			},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	infoJson, _ := json.Marshal(infoResp)

	statusResp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	statusJson, _ := json.Marshal(statusResp)

	callCount := 0
	mt := &mockTransport{
		responseBody: infoJson,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	mt2 := &mockTransport{
		responseBody: statusJson,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}

	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{
			Transport: &multiTransport{
				transports: []*mockTransport{mt, mt2},
				index:      &callCount,
			},
		}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	_, _, err := client.GetNumBikesAtStation(t.Context(), "Test Station")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetNumBikesAtStation_StationIDError(t *testing.T) {
	infoResp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	infoJson, _ := json.Marshal(infoResp)

	mt := &mockTransport{
		responseBody: infoJson,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	_, _, err := client.GetNumBikesAtStation(t.Context(), "Nonexistent Station")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetProvider_Success(t *testing.T) {
	infoResp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{
				{"station_id": "station-1", "name": "Test Station"},
			},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	infoJson, _ := json.Marshal(infoResp)

	statusResp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{
				{
					"station_id": "station-1",
					"vehicle_types_available": []map[string]any{
						{"vehicle_type_id": "1", "count": 7},
						{"vehicle_type_id": "2", "count": 2},
					},
				},
			},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	statusJson, _ := json.Marshal(statusResp)

	callCount := 0
	mt := &mockTransport{
		responseBody: statusJson,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	mt2 := &mockTransport{
		responseBody: infoJson,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}

	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{
			Transport: &multiTransport{
				transports: []*mockTransport{mt, mt2},
				index:      &callCount,
			},
		}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	provider := client.GetProvider("Test Station")
	data, err := provider(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Table != "citibike" {
		t.Errorf("expected table 'citibike', got %s", data.Table)
	}
	if data.Tags["location"] != "Test Station" {
		t.Errorf("expected location tag 'Test Station', got %s", data.Tags["location"])
	}
	if data.Fields["classics"] != 7 {
		t.Errorf("expected classics=7, got %v", data.Fields["classics"])
	}
	if data.Fields["ebikes"] != 2 {
		t.Errorf("expected ebikes=2, got %v", data.Fields["ebikes"])
	}
	if data.Stamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestGetProvider_Error(t *testing.T) {
	infoResp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	infoJson, _ := json.Marshal(infoResp)

	mt := &mockTransport{
		responseBody: infoJson,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	provider := client.GetProvider("Nonexistent Station")
	_, err := provider(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCaching_Enabled(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"vehicle_types": []map[string]any{
				{"vehicle_type_id": "1", "propulsion_type": "human"},
			},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	jsonResp, _ := json.Marshal(resp)

	mt := &mockTransport{
		responseBody: jsonResp,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetVehicleTypes(t.Context())
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if mt.callCount != 1 {
		t.Errorf("expected 1 call after first request, got %d", mt.callCount)
	}

	_, err = client.GetVehicleTypes(t.Context())
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if mt.callCount != 1 {
		t.Errorf("expected 1 total call (cached), got %d", mt.callCount)
	}
}

func TestCaching_Disabled(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"vehicle_types": []map[string]any{
				{"vehicle_type_id": "1", "propulsion_type": "human"},
			},
		},
		"last_updated": 1234567890,
		"ttl":          0,
		"version":      "2.3",
	}
	jsonResp, _ := json.Marshal(resp)

	mt := &mockTransport{
		responseBody: jsonResp,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetVehicleTypes(t.Context())
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if mt.callCount != 1 {
		t.Errorf("expected 1 call after first request, got %d", mt.callCount)
	}

	_, err = client.GetVehicleTypes(t.Context())
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if mt.callCount != 2 {
		t.Errorf("expected 2 total calls (no cache), got %d", mt.callCount)
	}
}

func TestGetStationInformation_Caching(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{
				{"station_id": "station-1", "name": "Test Station"},
			},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	jsonResp, _ := json.Marshal(resp)

	mt := &mockTransport{
		responseBody: jsonResp,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetStationInformation(t.Context())
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	_, err = client.GetStationInformation(t.Context())
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if mt.callCount != 1 {
		t.Errorf("expected 1 total call (cached), got %d", mt.callCount)
	}
}

func TestGetStationStatus_Caching(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{
				{"station_id": "station-1"},
			},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	jsonResp, _ := json.Marshal(resp)

	mt := &mockTransport{
		responseBody: jsonResp,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	_, err := client.GetStationStatus(t.Context())
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	_, err = client.GetStationStatus(t.Context())
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if mt.callCount != 1 {
		t.Errorf("expected 1 total call (cached), got %d", mt.callCount)
	}
}

func TestGetStationID_CachePopulates(t *testing.T) {
	infoResp := map[string]any{
		"data": map[string]any{
			"stations": []map[string]any{
				{"station_id": "station-1", "name": "Station A"},
				{"station_id": "station-2", "name": "Station B"},
			},
		},
		"last_updated": 1234567890,
		"ttl":          60,
		"version":      "2.3",
	}
	infoJson, _ := json.Marshal(infoResp)

	mt := &mockTransport{
		responseBody: infoJson,
		statusCode:   200,
		headers:      http.Header{"Content-Type": []string{"application/json"}},
	}
	client := citibike.NewClient(
		citibike.WithHTTPClient(&http.Client{Transport: mt}),
		citibike.WithBaseURL("http://redmaple.tree/"),
	)

	id1, err := client.GetStationID(t.Context(), "Station A")
	if err != nil {
		t.Fatalf("error getting Station A: %v", err)
	}
	if id1 != "station-1" {
		t.Errorf("expected station-1, got %s", id1)
	}

	id2, err := client.GetStationID(t.Context(), "Station B")
	if err != nil {
		t.Fatalf("error getting Station B: %v", err)
	}
	if id2 != "station-2" {
		t.Errorf("expected station-2, got %s", id2)
	}

	if mt.callCount != 1 {
		t.Errorf("expected 1 HTTP call (cache populated), got %d", mt.callCount)
	}
}
