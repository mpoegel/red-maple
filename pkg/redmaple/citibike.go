package redmaple

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	api "github.com/mpoegel/red-maple/pkg/api"
	citibike "github.com/mpoegel/red-maple/pkg/citibike"
)

func (s *Server) HandleCitibike(w http.ResponseWriter, r *http.Request) {
	stationStatus, err := s.citibike.GetStationStatus(r.Context())
	if err != nil {
		slog.Error("failed to get citibike station status", "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	data := api.CitibikePartial{}
	// TODO refactor names/station ID
	names := strings.Split(s.config.CitibikeStations, ",")
	// TODO error checking that there are two stations
	if first, err := findStationStatus(stationStatus, s.citibikeStations[0]); err != nil {
		slog.Error("failed to get find citibike station status", "err", err, "station", s.citibikeStations[0])
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	} else {
		data.First = *first
		data.First.Name = names[0]
	}
	if second, err := findStationStatus(stationStatus, s.citibikeStations[1]); err != nil {
		slog.Error("failed to get find citibike station status", "err", err, "station", s.citibikeStations[1])
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	} else {
		data.Second = *second
		data.Second.Name = names[1]
	}

	s.executeTemplate(w, "Citibike", data)
}

func (s *Server) HandleBikesFull(w http.ResponseWriter, r *http.Request) {
	s.executeTemplate(w, "BikesFull", struct{}{})
}

func loadCitibikeStations(ctx context.Context, client citibike.Client, names []string) ([]string, error) {
	stationInfo, err := client.GetStationInformation(ctx)
	if err != nil {
		return nil, err
	}
	res := []string{}
	for _, name := range names {
		found := false
		for _, si := range stationInfo.Data.Stations {
			if si.Name == name {
				res = append(res, si.StationID)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("citibike station not found: %s", name)
		}
	}
	return res, nil
}

func findStationStatus(stationStatus *citibike.StationStatusResponse, stationID string) (*api.CitibikeStation, error) {
	for _, station := range stationStatus.Data.Stations {
		if station.StationID == stationID {
			classics := getNumBikes(&station, classicBikeID)
			ebikes := getNumBikes(&station, eBikeID)
			return &api.CitibikeStation{
				Name:       stationID,
				TotalBikes: classics + ebikes,
				NumBikes:   classics,
				NumEbikes:  ebikes,
			}, nil
		}
	}
	return nil, errors.New("station status not found")
}

const (
	classicBikeID = "1"
	eBikeID       = "2"
)

func getNumBikes(stationStatus *citibike.StationStatus, bikeType string) int {
	for _, id := range stationStatus.VehicleTypesAvailable {
		if id.VehicleTypeID == bikeType {
			return id.Count
		}
	}
	return 0
}
