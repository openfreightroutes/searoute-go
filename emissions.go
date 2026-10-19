package searoute

import (
	"fmt"
	"math"
	"slices"
	"sync"
)

type ECAZone struct {
	Name   string
	BBoxes [][4]float64
}

var (
	ecaMu    sync.RWMutex
	ecaZones []ECAZone
)

func RegisterECAZones(zones []ECAZone) {
	ecaMu.Lock()
	defer ecaMu.Unlock()
	ecaZones = slices.Clone(zones)
}

var vesselClasses = map[string]struct{ fuelTonnesPerDay, serviceSpeedKnots float64 }{
	"feeder":      {40, 16},
	"handysize":   {25, 13},
	"panamax":     {45, 14},
	"postpanamax": {90, 18},
	"capesize":    {55, 14},
	"vlcc":        {70, 15},
}

func (r *Route) applyEmissions(coords [][2]float64, lenKm float64, opts Options) error {
	if !opts.Emissions {
		return nil
	}

	ecaMu.RLock()
	zones := ecaZones
	ecaMu.RUnlock()
	if len(zones) > 0 {
		inside := 0.0
		u := unwrapCoords(coords)
		for i := 0; i < len(u)-1; i++ {
			a, b := u[i], u[i+1]
			segKm := haversineKm(a, b)
			if segKm == 0 {
				continue
			}
			steps := max(1, math.Ceil(segKm/5))
			stepKm := segKm / steps
			for s := 0.0; s < steps; s++ {
				t := (s + 0.5) / steps
				lon := math.Mod(math.Mod(a[0]+(b[0]-a[0])*t+180, 360)+360, 360) - 180
				lat := a[1] + (b[1]-a[1])*t
				for _, z := range zones {
					found := false
					for _, bb := range z.BBoxes {
						if lon >= bb[0] && lon <= bb[2] && lat >= bb[1] && lat <= bb[3] {
							found = true
							break
						}
					}
					if found {
						inside += stepKm
						break
					}
				}
			}
		}
		fraction := 0.0
		if lenKm > 0 {
			fraction = inside / lenKm
		}
		r.ECAKm = &inside
		r.ECAFraction = &fraction
	}

	factor := opts.CO2eFactorKgPerKm
	if factor <= 0 && opts.VesselClass != "" {
		v, ok := vesselClasses[opts.VesselClass]
		if !ok {
			return fmt.Errorf("unknown vessel class %q", opts.VesselClass)
		}
		factor = v.fuelTonnesPerDay * 3.114 * 1000 / (24 * v.serviceSpeedKnots * 1.852)
	}
	if factor > 0 {
		tonnes := lenKm * (1 + opts.GLECInflation) * factor / 1000
		r.CO2eTonnes = &tonnes
	}
	return nil
}
