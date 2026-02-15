package subway

import (
	"bufio"
	"context"
	"errors"
	"fmt"
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

const (
	RootStationType string = "1"
)

var feedUrls = map[TrainLine]string{
	GTrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-g",
	LTrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-l",
	ATrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-ace",
	CTrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-ace",
	ETrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-ace",
	BTrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-bdfm",
	DTrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-bdfm",
	FTrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-bdfm",
	MTrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-bdfm",
	JTrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-jz",
	ZTrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-jz",
	NTrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-nqrw",
	QTrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-nqrw",
	RTrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-nqrw",
	WTrain:     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-nqrw",
	OneTrain:   "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs",
	TwoTrain:   "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs",
	ThreeTrain: "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs",
	FourTrain:  "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs",
	FiveTrain:  "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs",
	SixTrain:   "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs",
	SevenTrain: "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs",
	// TODO S train has 3 different feeds for the different segments
	STrain: "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs",
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
	feedURLs   map[TrainLine]string
}

var _ Client = (*ClientImpl)(nil)

type Option func(*ClientImpl)

func WithHTTPClient(client *http.Client) Option {
	return func(c *ClientImpl) {
		c.httpClient = client
	}
}

func WithStopMap(stopMap map[string]SubwayStop) Option {
	return func(c *ClientImpl) {
		c.stopMap = stopMap
	}
}

func WithFeedURLs(urls map[TrainLine]string) Option {
	return func(c *ClientImpl) {
		c.feedURLs = urls
	}
}

func NewClient(dataDir string, opts ...Option) (*ClientImpl, error) {
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

	c, _ := NewClientWithOptions(opts...)
	c.stopMap = stopMap
	return c, nil
}

func NewClientWithOptions(opts ...Option) (*ClientImpl, error) {
	c := &ClientImpl{
		httpClient: http.DefaultClient,
		feedURLs:   feedUrls,
		stopMap:    map[string]SubwayStop{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

func (c *ClientImpl) GetFeed(ctx context.Context, line TrainLine) (*FeedMessage, error) {
	slog.Debug("getting subway feed", "line", line)
	url := c.feedURLs[line]
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

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
