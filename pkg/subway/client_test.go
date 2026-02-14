package subway_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	subway "github.com/mpoegel/red-maple/pkg/subway"
	proto "google.golang.org/protobuf/proto"
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

func TestStopIdToLine(t *testing.T) {
	tests := []struct {
		stopID string
		want   subway.TrainLine
	}{
		{"L03N", subway.LTrain},
		{"L03S", subway.LTrain},
		{"G01N", subway.GTrain},
		{"G01S", subway.GTrain},
		{"A03N", subway.UnknownTrain},
		{"unknown", subway.UnknownTrain},
	}

	for _, tt := range tests {
		t.Run(tt.stopID, func(t *testing.T) {
			got := subway.StopIdToLine(tt.stopID)
			if got != tt.want {
				t.Errorf("StopIdToLine(%q) = %v, want %v", tt.stopID, got, tt.want)
			}
		})
	}
}

func TestGetFeed_Success(t *testing.T) {
	feed := &subway.FeedMessage{
		Header: &subway.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
		},
		Entity: []*subway.FeedEntity{
			{
				Id: proto.String("trip1"),
				TripUpdate: &subway.TripUpdate{
					Trip: &subway.TripDescriptor{
						TripId: proto.String("trip1"),
					},
				},
			},
		},
	}
	feedBytes, _ := proto.Marshal(feed)

	mt := &mockTransport{
		responseBody: feedBytes,
		statusCode:   200,
		headers:      http.Header{},
	}
	client, err := subway.NewClientWithOptions(
		subway.WithHTTPClient(&http.Client{Transport: mt}),
		subway.WithFeedURLs(map[subway.TrainLine]string{
			subway.LTrain: "http://redmaple.tree/feed",
		}),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}

	result, err := client.GetFeed(t.Context(), subway.LTrain)
	if err != nil {
		t.Fatalf("GetFeed error: %v", err)
	}
	if len(result.Entity) != 1 {
		t.Errorf("expected 1 entity, got %d", len(result.Entity))
	}
	if *result.Entity[0].Id != "trip1" {
		t.Errorf("expected entity id 'trip1', got %s", *result.Entity[0].Id)
	}
}

func TestGetFeed_HTTPError(t *testing.T) {
	mt := &mockTransport{
		err:        io.EOF,
		statusCode: 500,
	}
	client, err := subway.NewClientWithOptions(
		subway.WithHTTPClient(&http.Client{Transport: mt}),
		subway.WithFeedURLs(map[subway.TrainLine]string{
			subway.LTrain: "http://redmaple.tree/feed",
		}),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}

	_, err = client.GetFeed(t.Context(), subway.LTrain)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetFeed_404Error(t *testing.T) {
	mt := &mockTransport{
		responseBody: []byte("Not Found"),
		statusCode:   404,
		headers:      http.Header{},
	}
	client, err := subway.NewClientWithOptions(
		subway.WithHTTPClient(&http.Client{Transport: mt}),
		subway.WithFeedURLs(map[subway.TrainLine]string{
			subway.LTrain: "http://redmaple.tree/feed",
		}),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}

	_, err = client.GetFeed(t.Context(), subway.LTrain)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetFeed_InvalidProtobuf(t *testing.T) {
	mt := &mockTransport{
		responseBody: []byte("not valid protobuf"),
		statusCode:   200,
		headers:      http.Header{},
	}
	client, err := subway.NewClientWithOptions(
		subway.WithHTTPClient(&http.Client{Transport: mt}),
		subway.WithFeedURLs(map[subway.TrainLine]string{
			subway.LTrain: "http://redmaple.tree/feed",
		}),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}

	_, err = client.GetFeed(t.Context(), subway.LTrain)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetTripsAtStop_Success(t *testing.T) {
	stopID := "L03N"
	feed := &subway.FeedMessage{
		Header: &subway.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
		},
		Entity: []*subway.FeedEntity{
			{
				Id: proto.String("trip1"),
				TripUpdate: &subway.TripUpdate{
					Trip: &subway.TripDescriptor{
						TripId: proto.String("trip1"),
					},
					StopTimeUpdate: []*subway.TripUpdate_StopTimeUpdate{
						{
							StopId: proto.String(stopID),
							Arrival: &subway.TripUpdate_StopTimeEvent{
								Time: proto.Int64(1234567890),
							},
						},
						{
							StopId: proto.String("L04N"),
						},
					},
				},
			},
		},
	}
	feedBytes, _ := proto.Marshal(feed)

	stopMap := map[string]subway.SubwayStop{
		stopID: {ID: stopID, Name: "Station 3"},
		"L04N": {ID: "L04N", Name: "Station 4"},
	}

	mt := &mockTransport{
		responseBody: feedBytes,
		statusCode:   200,
		headers:      http.Header{},
	}
	client, err := subway.NewClientWithOptions(
		subway.WithHTTPClient(&http.Client{Transport: mt}),
		subway.WithStopMap(stopMap),
		subway.WithFeedURLs(map[subway.TrainLine]string{
			subway.LTrain: "http://redmaple.tree/feed",
		}),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}

	trips, alerts, err := client.GetTripsAtStop(t.Context(), stopID)
	if err != nil {
		t.Fatalf("GetTripsAtStop error: %v", err)
	}
	if len(trips) != 1 {
		t.Errorf("expected 1 trip, got %d", len(trips))
	}
	if len(alerts) != 0 {
		t.Errorf("expected no alerts, got %v", alerts)
	}
	if trips[0].Stop.ID != stopID {
		t.Errorf("expected stop ID %s, got %s", stopID, trips[0].Stop.ID)
	}
}

func TestGetTripsAtStop_StopNotInMap(t *testing.T) {
	stopID := "L03N"
	feed := &subway.FeedMessage{
		Header: &subway.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
		},
		Entity: []*subway.FeedEntity{
			{
				Id: proto.String("trip1"),
				TripUpdate: &subway.TripUpdate{
					Trip: &subway.TripDescriptor{
						TripId: proto.String("trip1"),
					},
					StopTimeUpdate: []*subway.TripUpdate_StopTimeUpdate{
						{
							StopId: proto.String(stopID),
						},
					},
				},
			},
		},
	}
	feedBytes, _ := proto.Marshal(feed)

	stopMap := map[string]subway.SubwayStop{}

	mt := &mockTransport{
		responseBody: feedBytes,
		statusCode:   200,
		headers:      http.Header{},
	}
	client, err := subway.NewClientWithOptions(
		subway.WithHTTPClient(&http.Client{Transport: mt}),
		subway.WithStopMap(stopMap),
		subway.WithFeedURLs(map[subway.TrainLine]string{
			subway.LTrain: "http://redmaple.tree/feed",
		}),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}

	trips, _, err := client.GetTripsAtStop(t.Context(), stopID)
	if err != nil {
		t.Fatalf("GetTripsAtStop error: %v", err)
	}
	if len(trips) != 1 {
		t.Errorf("expected 1 trip (found in feed but stop not in map), got %d", len(trips))
	}
	if trips[0].Stop.ID != "" {
		t.Errorf("expected empty stop ID, got %s", trips[0].Stop.ID)
	}
}

func TestGetTripsAtStop_NoTrips(t *testing.T) {
	stopID := "L03N"
	feed := &subway.FeedMessage{
		Header: &subway.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
		},
		Entity: []*subway.FeedEntity{},
	}
	feedBytes, _ := proto.Marshal(feed)

	stopMap := map[string]subway.SubwayStop{
		stopID: {ID: stopID, Name: "Station 3"},
	}

	mt := &mockTransport{
		responseBody: feedBytes,
		statusCode:   200,
		headers:      http.Header{},
	}
	client, err := subway.NewClientWithOptions(
		subway.WithHTTPClient(&http.Client{Transport: mt}),
		subway.WithStopMap(stopMap),
		subway.WithFeedURLs(map[subway.TrainLine]string{
			subway.LTrain: "http://redmaple.tree/feed",
		}),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}

	trips, _, err := client.GetTripsAtStop(t.Context(), stopID)
	if err != nil {
		t.Fatalf("GetTripsAtStop error: %v", err)
	}
	if len(trips) != 0 {
		t.Errorf("expected 0 trips, got %d", len(trips))
	}
}

func TestGetTrains_Success(t *testing.T) {
	feed := &subway.FeedMessage{
		Header: &subway.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
		},
		Entity: []*subway.FeedEntity{
			{
				Id: proto.String("vehicle1"),
				Vehicle: &subway.VehiclePosition{
					Trip: &subway.TripDescriptor{
						TripId: proto.String("trip1"),
					},
					StopId:        proto.String("L03N"),
					CurrentStatus: subway.VehiclePosition_STOPPED_AT.Enum(),
				},
			},
			{
				Id: proto.String("vehicle2"),
				Vehicle: &subway.VehiclePosition{
					Trip: &subway.TripDescriptor{
						TripId: proto.String("trip2"),
					},
					StopId:        proto.String("L04N"),
					CurrentStatus: subway.VehiclePosition_IN_TRANSIT_TO.Enum(),
				},
			},
		},
	}
	feedBytes, _ := proto.Marshal(feed)

	stopMap := map[string]subway.SubwayStop{
		"L03N": {ID: "L03N", Name: "Station 3"},
		"L04N": {ID: "L04N", Name: "Station 4"},
	}

	mt := &mockTransport{
		responseBody: feedBytes,
		statusCode:   200,
		headers:      http.Header{},
	}
	client, err := subway.NewClientWithOptions(
		subway.WithHTTPClient(&http.Client{Transport: mt}),
		subway.WithStopMap(stopMap),
		subway.WithFeedURLs(map[subway.TrainLine]string{
			subway.LTrain: "http://redmaple.tree/feed",
		}),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}

	trains, alerts, err := client.GetTrains(t.Context(), subway.LTrain)
	if err != nil {
		t.Fatalf("GetTrains error: %v", err)
	}
	if len(trains) != 2 {
		t.Errorf("expected 2 trains, got %d", len(trains))
	}
	if alerts != nil {
		t.Errorf("expected no alerts, got %v", alerts)
	}
	if !trains[0].IsAtStop {
		t.Error("expected first train to be at stop")
	}
	if trains[1].IsAtStop {
		t.Error("expected second train to not be at stop")
	}
}

func TestGetTrains_Empty(t *testing.T) {
	feed := &subway.FeedMessage{
		Header: &subway.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
		},
		Entity: []*subway.FeedEntity{},
	}
	feedBytes, _ := proto.Marshal(feed)

	mt := &mockTransport{
		responseBody: feedBytes,
		statusCode:   200,
		headers:      http.Header{},
	}
	client, err := subway.NewClientWithOptions(
		subway.WithHTTPClient(&http.Client{Transport: mt}),
		subway.WithStopMap(map[string]subway.SubwayStop{}),
		subway.WithFeedURLs(map[subway.TrainLine]string{
			subway.LTrain: "http://redmaple.tree/feed",
		}),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}

	trains, _, err := client.GetTrains(t.Context(), subway.LTrain)
	if err != nil {
		t.Fatalf("GetTrains error: %v", err)
	}
	if len(trains) != 0 {
		t.Errorf("expected 0 trains, got %d", len(trains))
	}
}

func TestGetTrains_Alerts(t *testing.T) {
	feed := &subway.FeedMessage{
		Header: &subway.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
		},
		Entity: []*subway.FeedEntity{
			{
				Id: proto.String("alert1"),
				Alert: &subway.Alert{
					HeaderText: &subway.TranslatedString{
						Translation: []*subway.TranslatedString_Translation{
							{Text: proto.String("Delay on L line")},
						},
					},
				},
			},
		},
	}
	feedBytes, _ := proto.Marshal(feed)

	mt := &mockTransport{
		responseBody: feedBytes,
		statusCode:   200,
		headers:      http.Header{},
	}
	client, err := subway.NewClientWithOptions(
		subway.WithHTTPClient(&http.Client{Transport: mt}),
		subway.WithStopMap(map[string]subway.SubwayStop{}),
		subway.WithFeedURLs(map[subway.TrainLine]string{
			subway.LTrain: "http://redmaple.tree/feed",
		}),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}

	trains, alerts, err := client.GetTrains(t.Context(), subway.LTrain)
	if err != nil {
		t.Fatalf("GetTrains error: %v", err)
	}
	if len(trains) != 0 {
		t.Errorf("expected 0 trains, got %d", len(trains))
	}
	if len(alerts) != 1 {
		t.Errorf("expected 1 alert, got %d", len(alerts))
	}
}

func TestGetStopsOnLine_Success(t *testing.T) {
	feed := &subway.FeedMessage{
		Header: &subway.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
		},
		Entity: []*subway.FeedEntity{
			{
				Id: proto.String("trip1"),
				TripUpdate: &subway.TripUpdate{
					Trip: &subway.TripDescriptor{
						TripId: proto.String("trip1"),
					},
					StopTimeUpdate: []*subway.TripUpdate_StopTimeUpdate{
						{
							StopId: proto.String("L03N"),
						},
					},
				},
			},
		},
	}
	feedBytes, _ := proto.Marshal(feed)

	stopMap := map[string]subway.SubwayStop{
		"L03N": {ID: "L03N", Name: "Station 3"},
		"L04N": {ID: "L04N", Name: "Station 4"},
		"G01N": {ID: "G01N", Name: "G Station 1"},
	}

	mt := &mockTransport{
		responseBody: feedBytes,
		statusCode:   200,
		headers:      http.Header{},
	}
	client, err := subway.NewClientWithOptions(
		subway.WithHTTPClient(&http.Client{Transport: mt}),
		subway.WithStopMap(stopMap),
		subway.WithFeedURLs(map[subway.TrainLine]string{
			subway.LTrain: "http://redmaple.tree/feed",
		}),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}

	stops, err := client.GetStopsOnLine(t.Context(), subway.LTrain)
	if err != nil {
		t.Fatalf("GetStopsOnLine error: %v", err)
	}
	if len(stops) != 2 {
		t.Errorf("expected 2 L stops, got %d", len(stops))
	}
}

func TestNewClientWithOptions_StopMap(t *testing.T) {
	stopMap := map[string]subway.SubwayStop{
		"L03N": {ID: "L03N", Name: "Test Station"},
	}

	feed := &subway.FeedMessage{
		Header: &subway.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
		},
		Entity: []*subway.FeedEntity{
			{
				Id: proto.String("trip1"),
				TripUpdate: &subway.TripUpdate{
					Trip: &subway.TripDescriptor{
						TripId: proto.String("trip1"),
					},
					StopTimeUpdate: []*subway.TripUpdate_StopTimeUpdate{
						{
							StopId: proto.String("L03N"),
						},
					},
				},
			},
		},
	}
	feedBytes, _ := proto.Marshal(feed)

	mt := &mockTransport{
		responseBody: feedBytes,
		statusCode:   200,
		headers:      http.Header{},
	}

	client, err := subway.NewClientWithOptions(
		subway.WithStopMap(stopMap),
		subway.WithHTTPClient(&http.Client{Transport: mt}),
		subway.WithFeedURLs(map[subway.TrainLine]string{
			subway.LTrain: "http://redmaple.tree/feed",
		}),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}

	stops, err := client.GetStopsOnLine(t.Context(), subway.LTrain)
	if err != nil {
		t.Fatalf("GetStopsOnLine error: %v", err)
	}
	if len(stops) != 1 {
		t.Errorf("expected 1 stop, got %d", len(stops))
	}
}

func TestGetTripsAtStop_Alert(t *testing.T) {
	feed := &subway.FeedMessage{
		Header: &subway.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
		},
		Entity: []*subway.FeedEntity{
			{
				Id: proto.String("alert1"),
				Alert: &subway.Alert{
					HeaderText: &subway.TranslatedString{
						Translation: []*subway.TranslatedString_Translation{
							{Text: proto.String("Service change")},
						},
					},
				},
			},
		},
	}
	feedBytes, _ := proto.Marshal(feed)

	stopMap := map[string]subway.SubwayStop{
		"L03N": {ID: "L03N", Name: "Station 3"},
	}

	mt := &mockTransport{
		responseBody: feedBytes,
		statusCode:   200,
		headers:      http.Header{},
	}
	client, err := subway.NewClientWithOptions(
		subway.WithHTTPClient(&http.Client{Transport: mt}),
		subway.WithStopMap(stopMap),
		subway.WithFeedURLs(map[subway.TrainLine]string{
			subway.LTrain: "http://redmaple.tree/feed",
		}),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}

	_, alerts, err := client.GetTripsAtStop(t.Context(), "L03N")
	if err != nil {
		t.Fatalf("GetTripsAtStop error: %v", err)
	}
	if len(alerts) != 1 {
		t.Errorf("expected 1 alert, got %d", len(alerts))
	}
}

func TestGetTripsAtStop_DeletedEntity(t *testing.T) {
	deleted := true
	feed := &subway.FeedMessage{
		Header: &subway.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
		},
		Entity: []*subway.FeedEntity{
			{
				Id:        proto.String("trip1"),
				IsDeleted: &deleted,
				TripUpdate: &subway.TripUpdate{
					Trip: &subway.TripDescriptor{
						TripId: proto.String("trip1"),
					},
				},
			},
		},
	}
	feedBytes, _ := proto.Marshal(feed)

	stopMap := map[string]subway.SubwayStop{
		"L03N": {ID: "L03N", Name: "Station 3"},
	}

	mt := &mockTransport{
		responseBody: feedBytes,
		statusCode:   200,
		headers:      http.Header{},
	}
	client, err := subway.NewClientWithOptions(
		subway.WithHTTPClient(&http.Client{Transport: mt}),
		subway.WithStopMap(stopMap),
		subway.WithFeedURLs(map[subway.TrainLine]string{
			subway.LTrain: "http://redmaple.tree/feed",
		}),
	)
	if err != nil {
		t.Fatalf("NewClientWithOptions error: %v", err)
	}

	trips, _, err := client.GetTripsAtStop(t.Context(), "L03N")
	if err != nil {
		t.Fatalf("GetTripsAtStop error: %v", err)
	}
	if len(trips) != 0 {
		t.Errorf("expected 0 trips (deleted entity), got %d", len(trips))
	}
}
