package subway

type SubwayStop struct {
	ID                string
	Name              string
	Latitude          float64
	Longitude         float64
	LocationType      string
	ParentStation     string
	AreTrainsStopping int
}

const (
	TrainsNotStopping   int = 0
	TrainsStoppingNorth int = 1
	TrainsStoppingSouth int = 2
)

type StopUpdate struct {
	Stop        SubwayStop
	Arrival     *TripUpdate_StopTimeEvent
	Departure   *TripUpdate_StopTimeEvent
	Destination SubwayStop
}

type TrainUpdate struct {
	NextStop SubwayStop
	IsAtStop bool
}
