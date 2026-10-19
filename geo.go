package searoute

import (
	"fmt"
	"math"
)

type Units string

const (
	Kilometers    Units = "kilometers"
	Meters        Units = "meters"
	Miles         Units = "miles"
	NauticalMiles Units = "nauticalmiles"
)

const earthRadiusM = 6371008.8

func haversineKm(a, b [2]float64) float64 {
	dLat := math.Mod(b[1]-a[1], 360) * math.Pi / 180
	dLon := math.Mod(b[0]-a[0], 360) * math.Pi / 180
	lat1 := math.Mod(a[1], 360) * math.Pi / 180
	lat2 := math.Mod(b[1], 360) * math.Pi / 180
	h := math.Pow(math.Sin(dLat/2), 2) + math.Pow(math.Sin(dLon/2), 2)*math.Cos(lat1)*math.Cos(lat2)
	return 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h)) * (earthRadiusM / 1e3)
}

func unitToKm(u Units) float64 {
	switch u {
	case Kilometers:
		return 1
	case Meters:
		return 0.001
	case Miles:
		return 1.609344
	case NauticalMiles:
		return 1.852
	}
	return 0
}

func lengthIn(coords [][2]float64, u Units) (float64, error) {
	kmPerUnit := unitToKm(u)
	if kmPerUnit == 0 {
		return 0, fmt.Errorf("%s units is invalid", u)
	}
	total := 0.0
	for i := 0; i < len(coords)-1; i++ {
		total += haversineKm(coords[i], coords[i+1]) / kmPerUnit
	}
	return total, nil
}

func pointToLineKm(p [2]float64, line [][2]float64) float64 {
	kmPerRad := earthRadiusM / 1e3
	degPerRad := 360 / (2 * math.Pi)
	best := math.Inf(1)
	for i := 0; i < len(line)-1; i++ {
		start, stop := line[i], line[i+1]
		var hit [2]float64
		if stop == p {
			hit = stop
		} else if start == p {
			hit = start
		} else {
			var v [3][3]float64
			for k, pos := range [3][2]float64{start, stop, p} {
				lat := math.Mod(pos[1], 360) * math.Pi / 180
				lng := math.Mod(pos[0], 360) * math.Pi / 180
				v[k] = [3]float64{math.Cos(lat) * math.Cos(lng), math.Cos(lat) * math.Sin(lng), math.Sin(lat)}
			}
			a, b, c := v[0], v[1], v[2]
			axis := [3]float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
			target := [3]float64{axis[1]*c[2] - axis[2]*c[1], axis[2]*c[0] - axis[0]*c[2], axis[0]*c[1] - axis[1]*c[0]}
			if axis == [3]float64{} {
				if a[0]*b[0]+a[1]*b[1]+a[2]*b[2] > 0 {
					hit = stop
				} else {
					hit = p
				}
			} else if target == [3]float64{} {
				hit = stop
			} else {
				cross := [3]float64{target[1]*axis[2] - target[2]*axis[1], target[2]*axis[0] - target[0]*axis[2], target[0]*axis[1] - target[1]*axis[0]}
				mag := math.Sqrt(math.Pow(cross[0], 2) + math.Pow(cross[1], 2) + math.Pow(cross[2], 2))
				i1 := [3]float64{cross[0] / mag, cross[1] / mag, cross[2] / mag}
				i2 := [3]float64{-i1[0], -i1[1], -i1[2]}
				in := i2
				if c[0]*i1[0]+c[1]*i1[1]+c[2]*i1[2] > c[0]*i2[0]+c[1]*i2[1]+c[2]*i2[2] {
					in = i1
				}
				axisMag := math.Sqrt(math.Pow(axis[0], 2) + math.Pow(axis[1], 2) + math.Pow(axis[2], 2))
				n := [3]float64{axis[0] / axisMag, axis[1] / axisMag, axis[2] / axisMag}
				aXi := [3]float64{a[1]*in[2] - a[2]*in[1], a[2]*in[0] - a[0]*in[2], a[0]*in[1] - a[1]*in[0]}
				iXb := [3]float64{in[1]*b[2] - in[2]*b[1], in[2]*b[0] - in[0]*b[2], in[0]*b[1] - in[1]*b[0]}
				cmpAI := aXi[0]*n[0] + aXi[1]*n[1] + aXi[2]*n[2]
				cmpIB := iXb[0]*n[0] + iXb[1]*n[1] + iXb[2]*n[2]
				if cmpAI >= 0 && cmpIB >= 0 {
					z := math.Min(math.Max(in[2], -1), 1)
					hit = [2]float64{
						math.Mod(math.Atan2(in[1], in[0]), 2*math.Pi) * 180 / math.Pi,
						math.Mod(math.Asin(z), 2*math.Pi) * 180 / math.Pi,
					}
				} else if a[0]*c[0]+a[1]*c[1]+a[2]*c[2] > b[0]*c[0]+b[1]*c[1]+b[2]*c[2] {
					hit = start
				} else {
					hit = stop
				}
			}
		}
		d := haversineKm(p, hit) / kmPerRad * degPerRad
		if d < best {
			best = d
		}
	}
	return best / degPerRad * kmPerRad
}
