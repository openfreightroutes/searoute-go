package searoute

import (
	"math"
	"testing"
)

func TestVesselFactor(t *testing.T) {
	r, err := SeaRoute(Point{-74, 40.5}, Point{4.4, 51.9}, Options{Units: Kilometers, Emissions: true, VesselClass: "panamax"})
	if err != nil {
		t.Fatal(err)
	}
	if got := *r.CO2eTonnes * 1000 / r.Length; math.Abs(got-225.19091329836468) > 1e-9 {
		t.Fatalf("factor %v", got)
	}
}

func TestUnknownVesselClass(t *testing.T) {
	_, err := SeaRoute(Point{-74, 40.5}, Point{4.4, 51.9}, Options{Emissions: true, VesselClass: "dinghy"})
	if err == nil {
		t.Fatal("want error")
	}
}

func TestNoZonesNoECA(t *testing.T) {
	RegisterECAZones(nil)
	r, err := SeaRoute(Point{-74, 40.5}, Point{4.4, 51.9}, Options{Emissions: true})
	if err != nil {
		t.Fatal(err)
	}
	if r.ECAKm != nil || r.ECAFraction != nil {
		t.Fatalf("got %v %v", r.ECAKm, r.ECAFraction)
	}
}
