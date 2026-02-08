package subway

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	proto "google.golang.org/protobuf/proto"
)

//go:generate protoc --proto_path=../../vendored/mta --go_opt=paths=source_relative --go_out=. ../../vendored/mta/nyct-subway.proto ../../vendored/mta/gtfs-realtime.proto

type TrainLine string

const (
	LTrain       TrainLine = "L"
	GTrain       TrainLine = "G"
	UnknownTrain TrainLine = "n/a"
)

const (
	RootStationType string = "1"
)

var feedUrls = map[TrainLine]string{
	GTrain: "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-g",
	LTrain: "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-l",
}

type Client interface {
	GetFeed(ctx context.Context, line TrainLine) (*FeedMessage, error)
	GetTripsAtStop(ctx context.Context, stopID string) ([]*StopUpdate, []*Alert, error)
	GetTrains(ctx context.Context, line TrainLine) (trains []TrainUpdate, alerts []*Alert, err error)
	GetStopsOnLine(ctx context.Context, line TrainLine) (stops []SubwayStop, err error)
}

type ClientImpl struct {
	httpClient *http.Client
	stopMap    map[string]SubwayStop
}

var _ Client = (*ClientImpl)(nil)

func NewClient(dataDir string) (*ClientImpl, error) {
	stopMap := map[string]SubwayStop{}

	fp, err := os.Open(path.Join(dataDir, "mta/stops.txt"))
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	scanner := bufio.NewScanner(fp)
	// skip the header
	scanner.Scan()
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ",")
		if len(parts) != 6 {
			return nil, errors.New("malformed stops.txt")
		}

		lat, err1 := strconv.ParseFloat(parts[2], 64)
		lon, err2 := strconv.ParseFloat(parts[3], 64)
		if err1 != nil || err2 != nil {
			return nil, errors.Join(err1, err2)
		}
		stopMap[parts[0]] = SubwayStop{
			ID:                parts[0],
			Name:              parts[1],
			Latitude:          lat,
			Longitude:         lon,
			LocationType:      parts[4],
			ParentStation:     parts[5],
			AreTrainsStopping: 0,
		}
	}

	return &ClientImpl{
		httpClient: http.DefaultClient,
		stopMap:    stopMap,
	}, nil
}

func (c *ClientImpl) GetFeed(ctx context.Context, line TrainLine) (*FeedMessage, error) {
	slog.Debug("getting subway feed", "line", line)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedUrls[line], nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	feed := &FeedMessage{}
	if err := proto.Unmarshal(body, feed); err != nil {
		return nil, err
	}

	return feed, nil
}

func (c *ClientImpl) GetTripsAtStop(ctx context.Context, stopID string) ([]*StopUpdate, []*Alert, error) {
	feed, err := c.GetFeed(ctx, StopIdToLine(stopID))
	if err != nil {
		return nil, nil, err
	}

	res := []*StopUpdate{}
	alerts := []*Alert{}
	for _, entity := range feed.Entity {
		if entity.IsDeleted != nil && *entity.IsDeleted {
			continue
		}
		if entity.Alert != nil {
			alerts = append(alerts, entity.Alert)
			slog.Debug("subway alert", "alert", entity.Alert)
			continue
		}
		if entity.TripUpdate == nil {
			continue
		}
		found := false
		stopUpdate := &StopUpdate{
			Stop: c.stopMap[stopID],
		}
		for _, stopTimeUpdate := range entity.TripUpdate.StopTimeUpdate {
			if *stopTimeUpdate.StopId == stopID {
				found = true
				stopUpdate.Arrival = stopTimeUpdate.Arrival
				stopUpdate.Departure = stopTimeUpdate.Departure
			}
		}
		if found {
			lastIndex := len(entity.TripUpdate.StopTimeUpdate) - 1
			stopUpdate.Destination = c.stopMap[*entity.TripUpdate.StopTimeUpdate[lastIndex].StopId]
			res = append(res, stopUpdate)

		}
	}

	return res, alerts, nil
}

func StopIdToLine(stopID string) TrainLine {
	switch stopID[0] {
	case 'L':
		return LTrain
	case 'G':
		return GTrain
	default:
		return UnknownTrain
	}
}

func (c *ClientImpl) GetTrains(ctx context.Context, line TrainLine) (trains []TrainUpdate, alerts []*Alert, err error) {
	feed, err := c.GetFeed(ctx, line)
	if err != nil {
		return
	}

	for _, entity := range feed.Entity {
		if entity.IsDeleted != nil && *entity.IsDeleted {
			continue
		}
		if entity.Alert != nil {
			alerts = append(alerts, entity.Alert)
			slog.Debug("subway alert", "alert", entity.Alert)
			continue
		}
		if entity.Vehicle == nil {
			continue
		}
		trains = append(trains, TrainUpdate{
			NextStop: c.stopMap[entity.Vehicle.GetStopId()],
			IsAtStop: entity.Vehicle.GetCurrentStatus() == VehiclePosition_STOPPED_AT,
		})
	}
	return
}

func (c *ClientImpl) GetStopsOnLine(ctx context.Context, line TrainLine) (stops []SubwayStop, err error) {
	feed, err := c.GetFeed(ctx, line)
	if err != nil {
		return
	}

	trainsStopping := map[string]int{}
	for _, entity := range feed.Entity {
		if entity.IsDeleted != nil && *entity.IsDeleted {
			continue
		}
		if entity.TripUpdate == nil {
			continue
		}
		for _, stopTimeUpdate := range entity.TripUpdate.StopTimeUpdate {
			if strings.HasSuffix(*stopTimeUpdate.StopId, "N") {
				trainsStopping[*stopTimeUpdate.StopId] |= TrainsStoppingNorth
			} else if strings.HasSuffix(*stopTimeUpdate.StopId, "S") {
				trainsStopping[*stopTimeUpdate.StopId] |= TrainsStoppingSouth
			}
		}
	}

	for _, stop := range c.stopMap {
		if stop.ID[0] == line[0] {
			if !strings.HasSuffix(stop.ID, "N") && !strings.HasSuffix(stop.ID, "S") {
				stop.AreTrainsStopping = trainsStopping[stop.ID+"N"] | trainsStopping[stop.ID+"S"]
			} else {
				stop.AreTrainsStopping = trainsStopping[stop.ID]
			}
			stops = append(stops, stop)
		}
	}
	return
}
