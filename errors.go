package searoute

import (
	"errors"
	"fmt"
	"math"
)

type SnapFailedError struct {
	Side       string
	DistanceKm float64
}

func (e *SnapFailedError) Error() string {
	msg := "Failed to snap " + e.Side + " to the maritime network"
	if !math.IsInf(e.DistanceKm, 0) && !math.IsNaN(e.DistanceKm) {
		msg += fmt.Sprintf(" (nearest line is %.1f km away)", e.DistanceKm)
	}
	return msg
}

var ErrNoRoute = errors.New("no sea route found between origin and destination")

var ErrUnknownPassage = errors.New("unknown passage")
