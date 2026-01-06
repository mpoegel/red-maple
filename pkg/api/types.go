package api

type DatetimePartial struct {
	Timestamp string
	AMOrPM    string
	Seconds   string
	Date      string
}

type CitibikePartial struct {
	First  CitibikeStation
	Second CitibikeStation
}

type CitibikeStation struct {
	Name       string
	TotalBikes int
	NumBikes   int
	NumEbikes  int
}
