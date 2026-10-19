package searoute

import (
	"errors"
	"math"
	"testing"

	"github.com/openfreightroutes/searoute-data/marnet100"
	"github.com/openfreightroutes/searoute-data/network"
)

func twoLineNetwork() *network.Network {
	return &network.Network{Features: []network.Feature{
		{Coords: [][2]float64{{0, 0}, {1, 0}}},
		{Coords: [][2]float64{{0, 1}, {1, 1}}},
	}}
}

func TestSnapNearestFeatureThenVertex(t *testing.T) {
	p := [2]float64{0.2, 0.3}
	vertex, dist, err := snap(twoLineNetwork(), p, "origin", 0)
	if err != nil {
		t.Fatal(err)
	}
	if vertex != [2]float64{0, 0} {
		t.Fatalf("vertex = %v", vertex)
	}
	if dist != haversineKm(p, [2]float64{0, 0}) {
		t.Fatalf("dist = %v", dist)
	}
}

func TestSnapMaxDistance(t *testing.T) {
	p := [2]float64{0.2, 0.3}
	_, _, err := snap(twoLineNetwork(), p, "origin", 10)
	var snapErr *SnapFailedError
	if !errors.As(err, &snapErr) {
		t.Fatalf("err = %v", err)
	}
	if snapErr.Side != "origin" || snapErr.DistanceKm != pointToLineKm(p, [][2]float64{{0, 0}, {1, 0}}) {
		t.Fatalf("err = %+v", snapErr)
	}
}

func TestSnapEmptyNetwork(t *testing.T) {
	_, _, err := snap(&network.Network{}, [2]float64{0, 0}, "destination", 0)
	var snapErr *SnapFailedError
	if !errors.As(err, &snapErr) || !math.IsInf(snapErr.DistanceKm, 1) {
		t.Fatalf("err = %v", err)
	}
	if err.Error() != "Failed to snap destination to the maritime network" {
		t.Fatalf("message = %q", err.Error())
	}
}

func TestSnapMarnet100(t *testing.T) {
	_, dist, err := snap(marnet100.Network(), [2]float64{121.5, 31.0}, "origin", 0)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(dist-26.724991752046467) > 1e-9 {
		t.Fatalf("dist = %v", dist)
	}
}
