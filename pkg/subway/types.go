package subway

type SubwayStop struct {
	ID            string
	Name          string
	Latitude      float64
	Longitude     float64
	LocationType  string
	ParentStation string
}

type StopUpdate struct {
	Stop        SubwayStop
	Arrival     *TripUpdate_StopTimeEvent
	Departure   *TripUpdate_StopTimeEvent
	Destination SubwayStop
}
