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
	names := slices.Clone(opts.Restrictions)
	if !opts.AllowArctic {
		names = append(names, "northwest", "northeast")
	}
	slices.Sort(names)
	names = slices.Compact(names)

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

	table := passages.Load()
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
		bb, ok := table.BBoxes[name]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownPassage, name)
		}
		blocks = append(blocks, bb...)
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
	if opts.AppendOriginDestination {
		route.Coordinates = slices.Concat([][2]float64{o}, path, [][2]float64{d})
	}
	return route, nil
}
