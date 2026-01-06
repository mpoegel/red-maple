package citibike

import (
	"context"
	"encoding/json"
	"net/http"
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
}

type ClientImpl struct {
	httpClient *http.Client
}

var _ Client = (*ClientImpl)(nil)

func NewClient() *ClientImpl {
	return &ClientImpl{
		httpClient: http.DefaultClient,
	}
}

func (c *ClientImpl) GetVehicleTypes(ctx context.Context) (*VehicleTypesResponse, error) {
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
	err = decoder.Decode(res)
	return res, err
}

func (c *ClientImpl) GetStationInformation(ctx context.Context) (*StationInformationResponse, error) {
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
	err = decoder.Decode(res)
	return res, err
}

func (c *ClientImpl) GetStationStatus(ctx context.Context) (*StationStatusResponse, error) {
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
	err = decoder.Decode(res)
	return res, err
}
