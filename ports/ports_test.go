package ports_test

import (
	"errors"
	"testing"

	searoute "github.com/openfreightroutes/searoute-go"
	"github.com/openfreightroutes/searoute-go/ports"
)

func TestResolve(t *testing.T) {
	p, err := ports.Resolve(" cnsha ")
	if err != nil {
		t.Fatal(err)
	}
	if p != (searoute.Point{Lon: 121.473701, Lat: 31.230416}) {
		t.Fatalf("got %v", p)
	}
	_, err = ports.Resolve("XXXXX")
	var perr *ports.UnknownPortError
	if !errors.As(err, &perr) || perr.Code != "XXXXX" || err.Error() != "Unknown port code 'XXXXX'" {
		t.Fatalf("got %v", err)
	}
}

func TestLookup(t *testing.T) {
	p, ok := ports.Lookup("NLRTM")
	if !ok || p.Code != "NLRTM" || p.Name != "Rotterdam" || p.Country != "Netherlands" {
		t.Fatalf("got %+v, %v", p, ok)
	}
}
