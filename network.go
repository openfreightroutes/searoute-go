package searoute

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/openfreightroutes/searoute-data/network"
)

func LoadNetwork(r io.Reader) (*network.Network, error) {
	var fc struct {
		Type     string
		Features []struct {
			Properties struct{ Pass string }
			Geometry   struct {
				Type        string
				Coordinates [][2]float64
			}
		}
	}
	if err := json.NewDecoder(r).Decode(&fc); err != nil {
		return nil, err
	}
	if fc.Type != "FeatureCollection" {
		return nil, errors.New("searoute: network must be a GeoJSON FeatureCollection")
	}
	if len(fc.Features) == 0 {
		return nil, errors.New("searoute: network has no features")
	}
	n := &network.Network{}
	for i, f := range fc.Features {
		if f.Geometry.Type != "LineString" {
			return nil, fmt.Errorf("searoute: feature %d is %q, want LineString", i, f.Geometry.Type)
		}
		for j := range f.Geometry.Coordinates {
			if f.Geometry.Coordinates[j][0] == 180 {
				f.Geometry.Coordinates[j][0] = -180
			}
		}
		n.Features = append(n.Features, network.Feature{Pass: f.Properties.Pass, Coords: f.Geometry.Coordinates})
	}
	return n, nil
}
