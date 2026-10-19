package searoute

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"

	"github.com/openfreightroutes/searoute-data/network"
	"github.com/openfreightroutes/searoute-data/passages"
)

type Options struct {
	Units                   Units
	Restrictions, Via       []string
	AllowArctic             bool
	VesselDraftMeters       float64
	SpeedKnots              float64
	AppendOriginDestination bool
	ReturnPassages          bool
	MaxSnapDistanceKm       float64
	Antimeridian            string
	Emissions               bool
	VesselClass             string
	CO2eFactorKgPerKm       float64
	GLECInflation           float64
	SimilarityThreshold     float64
}

var CanalMaxDraftM = passages.Load().CanalMaxDraftM

type Router struct {
	network *network.Network
	mu      sync.Mutex
	graphs  map[string]*graph
}

func NewRouter(n *network.Network) (*Router, error) {
	if n == nil || len(n.Features) == 0 {
		return nil, errors.New("network has no features")
	}
	return &Router{network: n, graphs: map[string]*graph{}}, nil
}

func (r *Router) SeaRoute(origin, destination Point, opts Options) (*Route, error) {
	units := opts.Units
	if units == "" {
		units = NauticalMiles
	}
	if opts.Antimeridian != "" && opts.Antimeridian != "unwrap" && opts.Antimeridian != "split" {
		return nil, fmt.Errorf("invalid antimeridian option %q: want \"unwrap\" or \"split\"", opts.Antimeridian)
	}
	table := passages.Load()
	for _, name := range slices.Concat(opts.Restrictions, opts.Via) {
		if _, ok := table.BBoxes[name]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownPassage, name)
		}
	}

	names := slices.Clone(opts.Restrictions)
	if opts.VesselDraftMeters > 0 {
		for _, canal := range []string{"panama", "suez", "kiel", "corinth"} {
			if opts.VesselDraftMeters > CanalMaxDraftM[canal] {
				names = append(names, canal)
			}
		}
	}
	if !opts.AllowArctic {
		names = append(names, "northwest", "northeast")
	}
	slices.Sort(names)
	names = slices.Compact(names)

	if len(opts.Via) > 0 {
		viaCanon := map[string]bool{}
		for _, name := range opts.Via {
			if alias, ok := table.Aliases[name]; ok {
				name = alias
			}
			viaCanon[name] = true
		}
		for _, name := range opts.Restrictions {
			if alias, ok := table.Aliases[name]; ok {
				name = alias
			}
			if viaCanon[name] {
				return nil, fmt.Errorf("cannot route via %q while it is also restricted: %w", name, ErrNoRoute)
			}
		}
		legOpts := opts
		legOpts.Via = nil
		legOpts.Restrictions = nil
		legOpts.AllowArctic = true
		legOpts.VesselDraftMeters = 0
		legOpts.Antimeridian = ""
		for _, name := range names {
			canonical := name
			if alias, ok := table.Aliases[name]; ok {
				canonical = alias
			}
			if !viaCanon[canonical] {
				legOpts.Restrictions = append(legOpts.Restrictions, name)
			}
		}

		waypoints := []Point{origin}
		for _, name := range opts.Via {
			lon, lat := 0.0, 0.0
			for _, bb := range table.BBoxes[name] {
				lon += (bb[0] + bb[2]) / 2
				lat += (bb[1] + bb[3]) / 2
			}
			n := float64(len(table.BBoxes[name]))
			waypoints = append(waypoints, Point{lon / n, lat / n})
		}
		waypoints = append(waypoints, destination)

		var legs []*Route
		for i := 0; i < len(waypoints)-1; i++ {
			leg, err := r.SeaRoute(waypoints[i], waypoints[i+1], legOpts)
			if err != nil {
				return nil, err
			}
			legs = append(legs, leg)
		}

		first, last := legs[0], legs[len(legs)-1]
		route := &Route{
			Coordinates:       slices.Clone(first.Coordinates),
			Units:             units,
			BBox:              [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)},
			OriginSnapKm:      first.OriginSnapKm,
			DestinationSnapKm: last.DestinationSnapKm,
		}
		for i, leg := range legs {
			route.Length += leg.Length
			if i == 0 {
				continue
			}
			next := leg.Coordinates
			if route.Coordinates[len(route.Coordinates)-1] == next[0] {
				next = next[1:]
			}
			route.Coordinates = append(route.Coordinates, next...)
		}
		for _, c := range route.Coordinates {
			route.BBox[0] = min(route.BBox[0], c[0])
			route.BBox[1] = min(route.BBox[1], c[1])
			route.BBox[2] = max(route.BBox[2], c[0])
			route.BBox[3] = max(route.BBox[3], c[1])
		}
		gcKm := haversineKm([2]float64{origin.Lon, origin.Lat}, [2]float64{destination.Lon, destination.Lat})
		totalKm := route.Length * unitToKm(units)
		route.GreatCircleLength = gcKm / unitToKm(units)
		route.DetourRatio = 1
		if gcKm > 0 {
			route.DetourRatio = totalKm / gcKm
		}
		if opts.SpeedKnots > 0 {
			hours := totalKm / (opts.SpeedKnots * 1.852)
			route.DurationHours = &hours
		}
		if opts.ReturnPassages {
			route.Passages = []string{}
			for _, leg := range legs {
				for _, name := range leg.Passages {
					if !slices.Contains(route.Passages, name) {
						route.Passages = append(route.Passages, name)
					}
				}
			}
		}
		route.applyAntimeridian(opts.Antimeridian)
		return route, nil
	}

	o := [2]float64{origin.Lon, origin.Lat}
	d := [2]float64{destination.Lon, destination.Lat}
	oVertex, oKm, err := snap(r.network, o, "origin", opts.MaxSnapDistanceKm)
	if err != nil {
		return nil, err
	}
	dVertex, dKm, err := snap(r.network, d, "destination", opts.MaxSnapDistanceKm)
	if err != nil {
		return nil, err
	}
	if oVertex == dVertex {
		return nil, fmt.Errorf("origin and destination snap to the same network vertex (%.1f km and %.1f km away); they are closer together than the network resolution: %w", oKm, dKm, ErrNoRoute)
	}

	native := map[string]bool{}
	var blocks [][4]float64
	for _, name := range names {
		canonical := name
		if alias, ok := table.Aliases[name]; ok {
			canonical = alias
		}
		if slices.Contains(table.Native, canonical) {
			native[canonical] = true
			continue
		}
		blocks = append(blocks, table.BBoxes[name]...)
	}
	key := strings.Join(names, ",")
	r.mu.Lock()
	g := r.graphs[key]
	if g == nil {
		g = buildGraph(r.network, func(a, b [2]float64, pass string) float64 {
			if native[pass] {
				return 0
			}
			for _, bb := range blocks {
				if segmentIntersectsBBox(a, b, bb) {
					return 0
				}
			}
			return haversineKm(a, b)
		})
		r.graphs[key] = g
	}
	r.mu.Unlock()

	path, ok := g.findPath(oVertex, dVertex)
	if !ok || len(path) < 2 {
		return nil, ErrNoRoute
	}
	length, err := lengthIn(path, units)
	if err != nil {
		return nil, err
	}
	lenKm, _ := lengthIn(path, Kilometers)
	gcKm := haversineKm(o, d)

	route := &Route{
		Coordinates:       path,
		Length:            length,
		Units:             units,
		BBox:              [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)},
		GreatCircleLength: gcKm / unitToKm(units),
		DetourRatio:       1,
		OriginSnapKm:      oKm,
		DestinationSnapKm: dKm,
	}
	for _, c := range path {
		route.BBox[0] = min(route.BBox[0], c[0])
		route.BBox[1] = min(route.BBox[1], c[1])
		route.BBox[2] = max(route.BBox[2], c[0])
		route.BBox[3] = max(route.BBox[3], c[1])
	}
	if gcKm > 0 {
		route.DetourRatio = lenKm / gcKm
	}
	if opts.SpeedKnots > 0 {
		hours := lenKm / (opts.SpeedKnots * 1.852)
		route.DurationHours = &hours
	}
	if opts.ReturnPassages {
		route.Passages = []string{}
		for i, a := range path {
			b := a
			if i+1 < len(path) {
				b = path[i+1]
			}
			for _, name := range table.ReportOrder {
				if slices.Contains(route.Passages, name) {
					continue
				}
				for _, bb := range table.BBoxes[name] {
					if segmentIntersectsBBox(a, b, bb) {
						route.Passages = append(route.Passages, name)
						break
					}
				}
			}
		}
	}
	if opts.AppendOriginDestination {
		route.Coordinates = slices.Concat([][2]float64{o}, path, [][2]float64{d})
	}
	route.applyAntimeridian(opts.Antimeridian)
	return route, nil
}
