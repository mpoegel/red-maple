package redmaple

import (
	"errors"
	"log/slog"
	"net/http"

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
	data := api.CitibikePartial{
		Stations: []api.CitibikeStation{},
	}
	for i := 0; i < max(len(s.config.CitibikeStations), 2); i++ {
		id, err := s.citibike.GetStationID(r.Context(), s.config.CitibikeStations[i])
		if err != nil {
			slog.Error("failed to find citibike station", "err", err, "station", s.config.CitibikeStations[i])
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if status, err := findStationStatus(stationStatus, id); err != nil {
			slog.Error("failed to get find citibike station status", "err", err, "station", id)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		} else {
			data.Stations = append(data.Stations, *status)
			data.Stations[i].Name = s.config.CitibikeStations[i]
		}
	}

	s.executeTemplate(w, "Citibike", data)
}

func (s *Server) HandleBikesFull(w http.ResponseWriter, r *http.Request) {
	s.executeTemplate(w, "BikesFull", struct{}{})
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
