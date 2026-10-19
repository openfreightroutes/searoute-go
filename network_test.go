package searoute_test

import (
	"strings"
	"testing"

	searoute "github.com/openfreightroutes/searoute-go"
)

func TestLoadNetwork(t *testing.T) {
	n, err := searoute.LoadNetwork(strings.NewReader(`{"type":"FeatureCollection","features":[
		{"type":"Feature","properties":{},"geometry":{"type":"LineString","coordinates":[[0,0],[1,1]]}},
		{"type":"Feature","properties":{"pass":"suez"},"geometry":{"type":"LineString","coordinates":[[1,1],[180,2]]}},
		{"type":"Feature","properties":{},"geometry":{"type":"LineString","coordinates":[[180,2],[3,3]]}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(n.Features) != 3 || n.Features[1].Pass != "suez" || n.Features[0].Pass != "" {
		t.Fatalf("features = %+v", n.Features)
	}
	if n.Features[1].Coords[1] != [2]float64{-180, 2} || n.Features[2].Coords[0] != [2]float64{-180, 2} {
		t.Fatalf("lon 180 not rewritten: %+v", n.Features)
	}

	for name, payload := range map[string]string{
		"feature":  `{"type":"Feature"}`,
		"empty":    `{"type":"FeatureCollection","features":[]}`,
		"polygon":  `{"type":"FeatureCollection","features":[{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[0,0],[1,1],[1,0],[0,0]]]}}]}`,
		"not json": `nope`,
	} {
		if _, err := searoute.LoadNetwork(strings.NewReader(payload)); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}
