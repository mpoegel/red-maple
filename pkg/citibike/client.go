package citibike

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	api "github.com/mpoegel/red-maple/pkg/api"
)

const (
	baseURL               = "https://gbfs.lyft.com/gbfs/2.3/bkn/en/"
	vehicleTypeEndpoint   = "vehicle_types.json"
	stationInfoEndpoint   = "station_information.json"
	stationStatusEndpoint = "station_status.json"
)

type Client interface {
	GetVehicleTypes(ctx context.Context) (*VehicleTypesResponse, error)
	GetStationInformation(ctx context.Context) (*StationInformationResponse, error)
	GetStationStatus(ctx context.Context) (*StationStatusResponse, error)
	GetStationID(ctx context.Context, name string) (string, error)
	GetNumBikesAtStation(ctx context.Context, name string) (numClassics, numEbikes int, err error)
	GetProvider(stationName string) api.ProviderFunc
}

type ClientImpl struct {
	httpClient *http.Client

	lastVehicleTypesResp       *VehicleTypesResponse
	lastVehicleTypesUpdatedAt  time.Time
	lastStationInfoResp        *StationInformationResponse
	lastStationInfoUpdatedAt   time.Time
	lastStationStatusResp      *StationStatusResponse
	lastStationStatusUpdatedAt time.Time

	stationCache map[string]StationInfo
}

var _ Client = (*ClientImpl)(nil)

func NewClient() *ClientImpl {
	return &ClientImpl{
		httpClient:   http.DefaultClient,
		stationCache: map[string]StationInfo{},
	}
}

func (c *ClientImpl) GetVehicleTypes(ctx context.Context) (*VehicleTypesResponse, error) {
	now := time.Now()
	if c.lastVehicleTypesResp != nil && c.lastVehicleTypesUpdatedAt.Add(time.Duration(c.lastVehicleTypesResp.TimeToLive)*time.Second).After(now) {
		return c.lastVehicleTypesResp, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+vehicleTypeEndpoint, nil)
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+stationInfoEndpoint, nil)
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+stationStatusEndpoint, nil)
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
	if station, ok := c.stationCache[name]; ok {
		return station.StationID, nil
	}

	stationInfo, err := c.GetStationInformation(ctx)
	if err != nil {
		return "", err
	}

	for _, si := range stationInfo.Data.Stations {
		c.stationCache[si.Name] = si
	}

	station, ok := c.stationCache[name]
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
			Table: "citibike",
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
