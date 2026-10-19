package ports

import (
	"strings"

	dataports "github.com/openfreightroutes/searoute-data/ports"
	searoute "github.com/openfreightroutes/searoute-go"
)

type Port struct {
	Code, Name, Country string
	Point               searoute.Point
}

type UnknownPortError struct {
	Code string
}

func (e *UnknownPortError) Error() string {
	return "Unknown port code '" + e.Code + "'"
}

func Lookup(code string) (Port, bool) {
	normalized := strings.Join(strings.Fields(strings.ToUpper(code)), "")
	p, ok := dataports.All()[normalized]
	if !ok {
		return Port{}, false
	}
	return Port{normalized, p.Name, p.Country, searoute.Point{Lon: p.Coordinates[0], Lat: p.Coordinates[1]}}, true
}

func Resolve(code string) (searoute.Point, error) {
	p, ok := Lookup(code)
	if !ok {
		return searoute.Point{}, &UnknownPortError{Code: code}
	}
	return p.Point, nil
}
