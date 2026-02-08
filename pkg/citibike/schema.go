package citibike

type StationStatusResponse struct {
	Data struct {
		Stations []StationStatus `json:"stations"`
	} `json:"data"`
	LastUpdated int    `json:"last_updated"`
	TimeToLive  int    `json:"ttl"`
	Version     string `json:"version"`
}

type StationStatus struct {
	NumDocksDisabled      int `json:"num_docks_disabled"`
	VehicleTypesAvailable []struct {
		VehicleTypeID string `json:"vehicle_type_id"`
		Count         int    `json:"count"`
	} `json:"vehicle_types_available"`
	IsReturning        int    `json:"is_returning"`
	IsInstalled        int    `json:"is_installed"`
	StationID          string `json:"station_id"`
	NumEBikesAvailable int    `json:"num_ebikes_available"`
	NumDocksAvailable  int    `json:"num_docks_available"`
	NumBikesDisabled   int    `json:"num_bikes_disabled"`
	NumBikesAvailable  int    `json:"num_bikes_available"`
	LastReported       int    `json:"last_reported"`
	IsRenting          int    `json:"is_renting"`
}

type StationInformationResponse struct {
	Data struct {
		Stations []StationInfo `json:"stations"`
	} `json:"data"`
	LastUpdated int    `json:"last_updated"`
	TimeToLive  int    `json:"ttl"`
	Version     string `json:"version"`
}

type StationInfo struct {
	Longitude  float64 `json:"lon"`
	ShortName  string  `json:"short_name"`
	StationID  string  `json:"station_id"`
	RentalURIs struct {
		Android string `json:"android"`
		IOS     string `json:"ios"`
	} `json:"rental_uris"`
	RegionID string  `json:"region_id"`
	Name     string  `json:"name"`
	Capacity int     `json:"capacity"`
	Latitude float64 `json:"lat"`
}

type VehicleTypesResponse struct {
	Data struct {
		VehicleTypes []struct {
			VehicleTypeID  string `json:"vehicle_type_id"`
			PropulsionType string `json:"propulsion_type"`
			FormFactor     string `json:"form_factor"`
		} `json:"vehicle_types"`
	} `json:"data"`
	LastUpdated int    `json:"last_updated"`
	TimeToLive  int    `json:"ttl"`
	Version     string `json:"version"`
}
