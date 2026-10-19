package searoute

import (
	"math"
	"testing"
)

var (
	newYork = [2]float64{-74.04, 40.69}
	london  = [2]float64{-0.13, 51.5}
)

func TestHaversineKm(t *testing.T) {
	got := haversineKm(newYork, london)
	if math.Abs(got-5574.1664065796585) > 1e-9 {
		t.Fatalf("got %v", got)
	}
}

func TestLengthIn(t *testing.T) {
	line := [][2]float64{newYork, london}
	cases := map[Units]float64{
		NauticalMiles: 3009.8090748270292,
		Miles:         3463.626425785698,
	}
	for u, want := range cases {
		got, err := lengthIn(line, u)
		if err != nil || math.Abs(got-want) > 1e-9 {
			t.Fatalf("%s: got %v, %v want %v", u, got, err, want)
		}
	}
	if _, err := lengthIn(line, Units("furlongs")); err == nil {
		t.Fatal("expected error for unknown unit")
	}
}

func TestPointToLineKm(t *testing.T) {
	line := [][2]float64{{-1, 0}, {1, 0}}
	got := pointToLineKm([2]float64{0, 0.5}, line)
	if math.Abs(got-55.59754011676645) > 1e-9 {
		t.Fatalf("got %v", got)
	}
	if got := pointToLineKm([2]float64{1, 0}, line); got != 0 {
		t.Fatalf("vertex: got %v", got)
	}
}

func TestUnitToKm(t *testing.T) {
	cases := map[Units]float64{Kilometers: 1, Meters: 0.001, Miles: 1.609344, NauticalMiles: 1.852}
	for u, want := range cases {
		if got := unitToKm(u); got != want {
			t.Fatalf("%s: got %v want %v", u, got, want)
		}
	}
}
