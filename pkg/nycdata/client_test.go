package nycdata_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	nycdata "github.com/mpoegel/red-maple/pkg/nycdata"
)

type mockTransport struct {
	responses  map[int][]byte
	err        error
	callCount  int
	statusCode int
	headers    http.Header
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}

	var body map[string]any
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		return nil, err
	}

	pageNum := 1
	if pageObj, ok := body["page"].(map[string]any); ok {
		if pn, ok := pageObj["pageNumber"].(float64); ok {
			pageNum = int(pn)
		}
	}

	respBody := m.responses[pageNum]
	if respBody == nil {
		respBody = []byte("[]")
	}

	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     m.headers,
	}, nil
}

func TestGetBicycleCounts_MultiplePages(t *testing.T) {
	page1Bytes := []byte(`[{"countid":"1","id":"300020904","date":"2025-01-01T00:00:00","counts":"100","status":"0"},{"countid":"2","id":"300020904","date":"2025-01-01T00:15:00","counts":"150","status":"0"}]`)
	page2Bytes := []byte(`[{"countid":"3","id":"300020904","date":"2025-01-01T00:30:00","counts":"200","status":"0"},{"countid":"4","id":"300020904","date":"2025-01-01T00:45:00","counts":"175","status":"0"}]`)
	page3Bytes := []byte(`[]`)

	mt := &mockTransport{
		responses: map[int][]byte{
			1: page1Bytes,
			2: page2Bytes,
			3: page3Bytes,
		},
		statusCode: 200,
		headers:    http.Header{"Content-Type": []string{"application/json"}},
	}

	client := nycdata.NewClient(
		nycdata.WithHTTPClient(&http.Client{Transport: mt}),
	)

	counts, err := client.GetBicycleCounts(
		t.Context(),
		nycdata.CounterID(300020904),
		nycdata.WithPageSize(2),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(counts) != 4 {
		t.Errorf("expected 4 counts, got %d", len(counts))
	}

	if mt.callCount != 3 {
		t.Errorf("expected 3 HTTP calls (2 data pages + 1 empty page), got %d", mt.callCount)
	}

	if counts[0].Counts != 100 {
		t.Errorf("expected first count 100, got %d", counts[0].Counts)
	}
	if counts[2].Counts != 200 {
		t.Errorf("expected third count 200, got %d", counts[2].Counts)
	}
}

func TestGetBicycleCounts_SinglePage(t *testing.T) {
	page1Bytes := []byte(`[{"countid":"1","id":"300020904","date":"2025-01-01T00:00:00","counts":"100","status":"0"}]`)

	mt := &mockTransport{
		responses: map[int][]byte{
			1: page1Bytes,
		},
		statusCode: 200,
		headers:    http.Header{"Content-Type": []string{"application/json"}},
	}

	client := nycdata.NewClient(
		nycdata.WithHTTPClient(&http.Client{Transport: mt}),
	)

	counts, err := client.GetBicycleCounts(t.Context(), nycdata.CounterID(300020904))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(counts) != 1 {
		t.Errorf("expected 1 count, got %d", len(counts))
	}

	if mt.callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", mt.callCount)
	}
}

func TestGetBicycleCounts_Caching(t *testing.T) {
	page1Bytes := []byte(`[{"countid":"1","id":"300020904","date":"2025-01-01T00:00:00","counts":"100","status":"0"}]`)

	mt := &mockTransport{
		responses: map[int][]byte{
			1: page1Bytes,
		},
		statusCode: 200,
		headers:    http.Header{"Content-Type": []string{"application/json"}},
	}

	client := nycdata.NewClient(
		nycdata.WithHTTPClient(&http.Client{Transport: mt}),
	)

	_, err := client.GetBicycleCounts(t.Context(), nycdata.CounterID(300020904))
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if mt.callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", mt.callCount)
	}

	_, err = client.GetBicycleCounts(t.Context(), nycdata.CounterID(300020904))
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if mt.callCount != 1 {
		t.Errorf("expected 1 total call (cached), got %d", mt.callCount)
	}
}

func TestGetBicycleCounts_HTTPError(t *testing.T) {
	mt := &mockTransport{
		err:        io.EOF,
		statusCode: 500,
	}
	client := nycdata.NewClient(
		nycdata.WithHTTPClient(&http.Client{Transport: mt}),
	)

	_, err := client.GetBicycleCounts(t.Context(), nycdata.CounterID(300020904))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetBicycleCounts_InvalidJSON(t *testing.T) {
	mt := &mockTransport{
		responses: map[int][]byte{
			1: []byte("not json"),
		},
		statusCode: 200,
		headers:    http.Header{"Content-Type": []string{"application/json"}},
	}
	client := nycdata.NewClient(
		nycdata.WithHTTPClient(&http.Client{Transport: mt}),
	)

	_, err := client.GetBicycleCounts(t.Context(), nycdata.CounterID(300020904))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetBicycleCounts_WithDateRange(t *testing.T) {
	page1Bytes := []byte(`[{"countid":"1","id":"300020904","date":"2025-01-01T00:00:00","counts":"100","status":"0"}]`)

	mt := &mockTransport{
		responses: map[int][]byte{
			1: page1Bytes,
		},
		statusCode: 200,
		headers:    http.Header{"Content-Type": []string{"application/json"}},
	}

	client := nycdata.NewClient(
		nycdata.WithHTTPClient(&http.Client{Transport: mt}),
	)

	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	_, err := client.GetBicycleCounts(
		t.Context(),
		nycdata.CounterID(300020904),
		nycdata.WithDateRange(startDate, endDate),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mt.callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", mt.callCount)
	}
}
