package redmaple

import (
	"log/slog"
	"net/http"

	api "github.com/mpoegel/red-maple/pkg/api"
)

func (s *Server) HandleCitibike(w http.ResponseWriter, r *http.Request) {
	data := api.CitibikePartial{
		Stations: []api.CitibikeStation{},
	}
	for i := 0; i < max(len(s.config.CitibikeStations), 2); i++ {
		name := s.config.CitibikeStations[i]
		numClassics, numEbikes, err := s.citibike.GetNumBikesAtStation(r.Context(), name)
		if err != nil {
			slog.Error("failed to get citibike station status", "err", err, "station", name)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		data.Stations = append(data.Stations, api.CitibikeStation{
			Name:       name,
			TotalBikes: numClassics + numEbikes,
			NumBikes:   numClassics,
			NumEbikes:  numEbikes,
		})
	}

	s.executeTemplate(w, "Citibike", data)
}

func (s *Server) HandleBikesFull(w http.ResponseWriter, r *http.Request) {
	s.executeTemplate(w, "BikesFull", struct{}{})
}
