package subway

type TrainLine string

const (
	OneTrain     TrainLine = "1"
	TwoTrain     TrainLine = "2"
	ThreeTrain   TrainLine = "3"
	FourTrain    TrainLine = "4"
	FiveTrain    TrainLine = "5"
	SixTrain     TrainLine = "6"
	SevenTrain   TrainLine = "7"
	ATrain       TrainLine = "A"
	BTrain       TrainLine = "B"
	CTrain       TrainLine = "C"
	DTrain       TrainLine = "D"
	ETrain       TrainLine = "E"
	FTrain       TrainLine = "F"
	GTrain       TrainLine = "G"
	JTrain       TrainLine = "J"
	LTrain       TrainLine = "L"
	MTrain       TrainLine = "M"
	NTrain       TrainLine = "N"
	QTrain       TrainLine = "Q"
	RTrain       TrainLine = "R"
	WTrain       TrainLine = "W"
	ZTrain       TrainLine = "Z"
	STrain       TrainLine = "S"
	UnknownTrain TrainLine = "n/a"
)

func ParseTrainLine(s string) TrainLine {
	switch s {
	case "1":
		return OneTrain
	case "2":
		return TwoTrain
	case "3":
		return ThreeTrain
	case "4":
		return FourTrain
	case "5":
		return FiveTrain
	case "6":
		return SixTrain
	case "7":
		return SevenTrain
	case "A":
		return ATrain
	case "B":
		return BTrain
	case "C":
		return CTrain
	case "D":
		return DTrain
	case "E":
		return ETrain
	case "F":
		return FTrain
	case "G":
		return GTrain
	case "J":
		return JTrain
	case "L":
		return LTrain
	case "M":
		return MTrain
	case "N":
		return NTrain
	case "Q":
		return QTrain
	case "R":
		return RTrain
	case "W":
		return WTrain
	case "Z":
		return ZTrain
	case "S":
		return STrain
	default:
		return UnknownTrain
	}
}

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
