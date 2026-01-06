package citibike

import (
	"context"
	"time"
)

type CachedClient struct {
	client                     Client
	lastVehicleTypesResp       *VehicleTypesResponse
	lastVehicleTypesUpdatedAt  time.Time
	lastStationInfoResp        *StationInformationResponse
	lastStationInfoUpdatedAt   time.Time
	lastStationStatusResp      *StationStatusResponse
	lastStationStatusUpdatedAt time.Time
}

var _ Client = (*CachedClient)(nil)

func NewCachedClient() *CachedClient {
	return &CachedClient{
		client: NewClient(),
	}
}

func (c *CachedClient) GetVehicleTypes(ctx context.Context) (*VehicleTypesResponse, error) {
	now := time.Now()
	if c.lastVehicleTypesResp == nil || c.lastVehicleTypesUpdatedAt.Add(time.Duration(c.lastVehicleTypesResp.TimeToLive)*time.Second).Before(now) {
		resp, err := c.client.GetVehicleTypes(ctx)
		if err != nil {
			return nil, err
		}
		c.lastVehicleTypesResp = resp
		c.lastVehicleTypesUpdatedAt = now
	}
	return c.lastVehicleTypesResp, nil
}

func (c *CachedClient) GetStationInformation(ctx context.Context) (*StationInformationResponse, error) {
	now := time.Now()
	if c.lastStationInfoResp == nil || c.lastStationInfoUpdatedAt.Add(time.Duration(c.lastStationInfoResp.TimeToLive)*time.Second).Before(now) {
		resp, err := c.client.GetStationInformation(ctx)
		if err != nil {
			return nil, err
		}
		c.lastStationInfoResp = resp
		c.lastStationInfoUpdatedAt = now
	}
	return c.lastStationInfoResp, nil
}

func (c *CachedClient) GetStationStatus(ctx context.Context) (*StationStatusResponse, error) {
	now := time.Now()
	if c.lastStationStatusResp == nil || c.lastStationStatusUpdatedAt.Add(time.Duration(c.lastStationStatusResp.TimeToLive)*time.Second).Before(now) {
		resp, err := c.client.GetStationStatus(ctx)
		if err != nil {
			return nil, err
		}
		c.lastStationStatusResp = resp
		c.lastStationStatusUpdatedAt = now
	}
	return c.lastStationStatusResp, nil
}
