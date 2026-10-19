package eca_test

import (
	"testing"

	searoute "github.com/openfreightroutes/searoute-go"
	"github.com/openfreightroutes/searoute-go/eca"
)

func TestECAInit(t *testing.T) {
	if len(eca.Zones) != 5 {
		t.Fatalf("got %d zones", len(eca.Zones))
	}
	r, err := searoute.SeaRoute(searoute.Point{Lon: -74, Lat: 40.5}, searoute.Point{Lon: 4.4, Lat: 51.9}, searoute.Options{Emissions: true})
	if err != nil {
		t.Fatal(err)
	}
	if r.ECAKm == nil || *r.ECAKm <= 0 {
		t.Fatalf("ecaKm %v", r.ECAKm)
	}
}
