package searoute

import "math"

func unwrapCoords(c [][2]float64) [][2]float64 {
	if len(c) == 0 {
		return nil
	}
	out := [][2]float64{c[0]}
	offset := 0.0
	for i := 1; i < len(c); i++ {
		lon := c[i][0] + offset
		prev := out[i-1][0]
		for lon-prev > 180 {
			offset -= 360
			lon -= 360
		}
		for lon-prev <= -180 {
			offset += 360
			lon += 360
		}
		out = append(out, [2]float64{lon, c[i][1]})
	}
	return out
}

func splitAtAntimeridian(coords [][2]float64) [][][2]float64 {
	if len(coords) == 0 {
		return nil
	}
	u := unwrapCoords(coords)
	panelOf := func(lon float64) float64 { return math.Floor((lon + 180) / 360) }

	var segments [][][2]float64
	panel := panelOf(u[0][0])
	current := [][2]float64{{u[0][0] - 360*panel, u[0][1]}}
	for i := 1; i < len(u); i++ {
		prev, cur := u[i-1], u[i]
		curPanel := panelOf(cur[0])
		if curPanel == panel {
			p := [2]float64{cur[0] - 360*panel, cur[1]}
			if current[len(current)-1] != p {
				current = append(current, p)
			}
			continue
		}

		boundary := 180 + 360*(panel-1)
		if curPanel > panel {
			boundary = 180 + 360*panel
		}
		t := (boundary - prev[0]) / (cur[0] - prev[0])
		lat := prev[1] + t*(cur[1]-prev[1])

		p := [2]float64{boundary - 360*panel, lat}
		if current[len(current)-1] != p {
			current = append(current, p)
		}
		segments = append(segments, current)

		panel = curPanel
		current = [][2]float64{{boundary - 360*panel, lat}}
		p = [2]float64{cur[0] - 360*panel, cur[1]}
		if current[len(current)-1] != p {
			current = append(current, p)
		}
	}
	return append(segments, current)
}

func (r *Route) applyAntimeridian(mode string) {
	switch mode {
	case "unwrap":
		r.Coordinates = unwrapCoords(r.Coordinates)
	case "split":
		r.MultiCoordinates = splitAtAntimeridian(r.Coordinates)
		r.Coordinates = nil
	}
}
