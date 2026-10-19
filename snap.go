package searoute

import (
	"math"

	"github.com/openfreightroutes/searoute-data/network"
)

func snap(n *network.Network, p [2]float64, side string, maxKm float64) (vertex [2]float64, distKm float64, err error) {
	nearest := -1
	nearestLineKm := math.Inf(1)
	for i, f := range n.Features {
		d := pointToLineKm(p, f.Coords)
		if d < nearestLineKm {
			nearestLineKm = d
			nearest = i
		}
	}
	if nearest < 0 {
		return vertex, 0, &SnapFailedError{Side: side, DistanceKm: math.Inf(1)}
	}
	if maxKm > 0 && nearestLineKm > maxKm {
		return vertex, 0, &SnapFailedError{Side: side, DistanceKm: nearestLineKm}
	}

	distKm = math.Inf(1)
	for _, c := range n.Features[nearest].Coords {
		d := haversineKm(p, c)
		if d < distKm {
			distKm = d
			vertex = c
		}
	}
	return vertex, distKm, nil
}
