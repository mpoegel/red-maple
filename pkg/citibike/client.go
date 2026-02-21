package citibike

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
)

const (
	baseURL               = "https://gbfs.lyft.com/gbfs/2.3/bkn/en/"
	vehicleTypeEndpoint   = "vehicle_types.json"
	stationInfoEndpoint   = "station_information.json"
	stationStatusEndpoint = "station_status.json"
	tableName             = "citibike"
)

type Client interface {
	GetVehicleTypes(ctx context.Context) (*VehicleTypesResponse, error)
	GetStationInformation(ctx context.Context) (*StationInformationResponse, error)
	GetStationStatus(ctx context.Context) (*StationStatusResponse, error)
	GetStationID(ctx context.Context, name string) (string, error)
	GetNumBikesAtStation(ctx context.Context, name string) (numClassics, numEbikes int, err error)
	GetProvider(stationName string) api.ProviderFunc
	GetHistoricalBikeCounts24Hours(ctx context.Context, importer api.Importer, stationName string) ([]HistoricalBikeCount, error)
	GetHistoricalBikeCounts7Days(ctx context.Context, importer api.Importer, stationName string) ([]HistoricalBikeCount, error)
	GetHistoricalBikeCounts30Days(ctx context.Context, importer api.Importer, stationName string) ([]HistoricalBikeCount, error)
}

type ClientImpl struct {
	httpClient *http.Client
	baseURL    string

	lastVehicleTypesResp       *VehicleTypesResponse
	lastVehicleTypesUpdatedAt  time.Time
	lastStationInfoResp        *StationInformationResponse
	lastStationInfoUpdatedAt   time.Time
	lastStationStatusResp      *StationStatusResponse
	lastStationStatusUpdatedAt time.Time

	mu           sync.RWMutex
	stationCache map[string]StationInfo
}

var _ Client = (*ClientImpl)(nil)

type Option func(*ClientImpl)

func WithHTTPClient(client *http.Client) Option {
	return func(c *ClientImpl) {
		c.httpClient = client
	}
}

func WithBaseURL(url string) Option {
	return func(c *ClientImpl) {
		c.baseURL = url
	}
}

func WithStationCache(cache map[string]StationInfo) Option {
	return func(c *ClientImpl) {
		c.stationCache = cache
	}
}

func NewClient(opts ...Option) *ClientImpl {
	c := &ClientImpl{
		httpClient:   http.DefaultClient,
		baseURL:      baseURL,
		stationCache: map[string]StationInfo{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *ClientImpl) GetVehicleTypes(ctx context.Context) (*VehicleTypesResponse, error) {
	now := time.Now()
	if c.lastVehicleTypesResp != nil && c.lastVehicleTypesUpdatedAt.Add(time.Duration(c.lastVehicleTypesResp.TimeToLive)*time.Second).After(now) {
		return c.lastVehicleTypesResp, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+vehicleTypeEndpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	res := &VehicleTypesResponse{}
	decoder := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if err = decoder.Decode(res); err != nil {
		return nil, err
	}

	c.lastVehicleTypesResp = res
	c.lastVehicleTypesUpdatedAt = now
	return res, nil
}

func (c *ClientImpl) GetStationInformation(ctx context.Context) (*StationInformationResponse, error) {
	now := time.Now()
	if c.lastStationInfoResp != nil && c.lastStationInfoUpdatedAt.Add(time.Duration(c.lastStationInfoResp.TimeToLive)*time.Second).After(now) {
		return c.lastStationInfoResp, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+stationInfoEndpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	res := &StationInformationResponse{}
	decoder := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if err = decoder.Decode(res); err != nil {
		return nil, err
	}

	c.lastStationInfoResp = res
	c.lastStationInfoUpdatedAt = now

	return res, nil
}

func (c *ClientImpl) GetStationStatus(ctx context.Context) (*StationStatusResponse, error) {
	now := time.Now()
	if c.lastStationStatusResp != nil && c.lastStationStatusUpdatedAt.Add(time.Duration(c.lastStationStatusResp.TimeToLive)*time.Second).After(now) {
		return c.lastStationStatusResp, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+stationStatusEndpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	res := &StationStatusResponse{}
	decoder := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if err = decoder.Decode(res); err != nil {
		return nil, err
	}

	c.lastStationStatusResp = res
	c.lastStationStatusUpdatedAt = now

	return res, nil
}

func (c *ClientImpl) GetStationID(ctx context.Context, name string) (string, error) {
	c.mu.RLock()
	if station, ok := c.stationCache[name]; ok {
		c.mu.RUnlock()
		return station.StationID, nil
	}
	c.mu.RUnlock()

	stationInfo, err := c.GetStationInformation(ctx)
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	for _, si := range stationInfo.Data.Stations {
		c.stationCache[si.Name] = si
	}

	station, ok := c.stationCache[name]
	c.mu.Unlock()
	if !ok {
		return "", errors.New("station not found")
	}
	return station.StationID, nil
}

func (c *ClientImpl) GetNumBikesAtStation(ctx context.Context, name string) (numClassics, numEbikes int, err error) {
	stations, err := c.GetStationStatus(ctx)
	if err != nil {
		return
	}

	id, err := c.GetStationID(ctx, name)
	if err != nil {
		return
	}

	for _, station := range stations.Data.Stations {
		if station.StationID == id {
			numClassics = countBikes(&station, classicBikeID)
			numEbikes = countBikes(&station, eBikeID)
			slog.Debug("counted bikes", "station", name, "classics", numClassics, "ebikes", numEbikes)
			return
		}
	}
	err = errors.New("station status not found")
	return
}

const (
	classicBikeID = "1"
	eBikeID       = "2"
)

type HistoricalBikeCount struct {
	Classics int
	Ebikes   int
	Stamp    time.Time
}

func countBikes(stationStatus *StationStatus, bikeType string) int {
	for _, id := range stationStatus.VehicleTypesAvailable {
		if id.VehicleTypeID == bikeType {
			return id.Count
		}
	}
	return 0
}

func (c *ClientImpl) GetProvider(stationName string) api.ProviderFunc {
	return func(ctx context.Context) (*api.DataPoint, error) {
		numClassics, numEbikes, err := c.GetNumBikesAtStation(ctx, stationName)
		if err != nil {
			return nil, err
		}

		data := &api.DataPoint{
			Table: tableName,
			Tags: map[api.DataTag]string{
				api.LocationTag: stationName,
			},
			Fields: map[string]any{
				"classics": numClassics,
				"ebikes":   numEbikes,
			},
			Stamp: time.Now(),
		}
		return data, nil
	}
}

func (c *ClientImpl) GetHistoricalBikeCounts24Hours(ctx context.Context, importer api.Importer, stationName string) ([]HistoricalBikeCount, error) {
	return c.queryBikeCounts(ctx, importer, stationName, func(i api.Importer, ctx context.Context) ([]*api.DataPoint, error) {
		return i.QueryRange(ctx, tableName, 24*time.Hour)
	})
}

func (c *ClientImpl) GetHistoricalBikeCounts7Days(ctx context.Context, importer api.Importer, stationName string) ([]HistoricalBikeCount, error) {
	return c.queryBikeCounts(ctx, importer, stationName, func(i api.Importer, ctx context.Context) ([]*api.DataPoint, error) {
		return i.QueryRange(ctx, tableName, 7*24*time.Hour)
	})
}

func (c *ClientImpl) GetHistoricalBikeCounts30Days(ctx context.Context, importer api.Importer, stationName string) ([]HistoricalBikeCount, error) {
	return c.queryBikeCounts(ctx, importer, stationName, func(i api.Importer, ctx context.Context) ([]*api.DataPoint, error) {
		return i.QueryRange(ctx, tableName, 30*24*time.Hour)
	})
}

type queryFunc func(importer api.Importer, ctx context.Context) ([]*api.DataPoint, error)

func (c *ClientImpl) queryBikeCounts(ctx context.Context, importer api.Importer, stationName string, queryFn queryFunc) ([]HistoricalBikeCount, error) {
	rows, err := queryFn(importer, ctx)
	if err != nil {
		return nil, err
	}

	var results []HistoricalBikeCount
	for _, row := range rows {
		slog.Debug("parsing citibike row", "row", row)
		location, ok := row.Tags["location"]
		if !ok || location != stationName {
			continue
		}

		var classics, ebikes int
		if classicsVal, ok := row.Fields["classics"]; ok {
			switch v := classicsVal.(type) {
			case int64:
				classics = int(v)
			case float64:
				classics = int(v)
			default:
				slog.Warn("unknown classics type", "type", fmt.Sprintf("%T", classicsVal))
			}
		}
		if ebikesVal, ok := row.Fields["ebikes"]; ok {
			switch v := ebikesVal.(type) {
			case int64:
				ebikes = int(v)
			case float64:
				ebikes = int(v)
			default:
				slog.Warn("unknown ebikes type", "type", fmt.Sprintf("%T", ebikesVal))
			}
		}

		results = append(results, HistoricalBikeCount{
			Classics: classics,
			Ebikes:   ebikes,
			Stamp:    row.Stamp,
		})
	}

	return results, nil
}
