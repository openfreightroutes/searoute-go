package searoute_test

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/openfreightroutes/searoute-data/golden"
	"github.com/openfreightroutes/searoute-data/marnet20"
	"github.com/openfreightroutes/searoute-data/marnet50"
	"github.com/openfreightroutes/searoute-data/network"

	searoute "github.com/openfreightroutes/searoute-go"
)

var knownOptions = []string{"units", "restrictions", "via", "allowArctic", "vesselDraftMeters", "speedKnots", "appendOriginDestination", "returnPassages", "maxSnapDistanceKm", "antimeridian", "emissions", "vesselClass", "co2eFactorKgPerKm", "glecInflation", "similarityThreshold", "k"}

func TestGolden(t *testing.T) {
	pending := map[string]string{
		"antimeridian": "antimeridian modes land in Task 6",
		"multi":        "SeaRouteMulti lands in Task 7",
		"alternatives": "SeaRouteAlternatives lands in Task 7",
		"errors":       "via and ports land in Tasks 5 and 7",
		"emissions":    "emissions land in Task 8",
		"networks":     "custom networks land in Task 9",
	}
	networks := map[string]func() *network.Network{"marnet50": marnet50.Network, "marnet20": marnet20.Network}
	routers := map[string]*searoute.Router{}

	for _, file := range []string{"basic", "options", "antimeridian", "multi", "alternatives", "emissions", "errors", "networks"} {
		t.Run(file, func(t *testing.T) {
			if reason, ok := pending[file]; ok {
				t.Skip(reason)
			}
			f, err := golden.FS.Open(file + ".json.gz")
			if err != nil {
				t.Fatal(err)
			}
			zr, err := gzip.NewReader(f)
			if err != nil {
				t.Fatal(err)
			}
			var cases []struct {
				Name    string
				Fn      string
				Network string
				ECA     bool
				Points  []any
				Options json.RawMessage
				Result  json.RawMessage
				Error   *struct{ Name, Side string }
			}
			if err := json.NewDecoder(zr).Decode(&cases); err != nil {
				t.Fatal(err)
			}

			for _, c := range cases {
				t.Run(strings.TrimPrefix(c.Name, file+"/"), func(t *testing.T) {
					var points []searoute.Point
					for _, p := range c.Points {
						ll, ok := p.([]any)
						if !ok {
							t.Skip("port codes need ports.Resolve (Task 7)")
						}
						points = append(points, searoute.Point{Lon: ll[0].(float64), Lat: ll[1].(float64)})
					}
					if c.ECA {
						t.Skip("ECA zones land in Task 8")
					}
					// encoding/json matches keys to field names case-insensitively, so searoute-ts option names decode straight into Options.
					var opts searoute.Options
					if err := json.Unmarshal(c.Options, &opts); err != nil {
						t.Fatal(err)
					}
					var rawOpts map[string]json.RawMessage
					if err := json.Unmarshal(c.Options, &rawOpts); err != nil {
						t.Fatal(err)
					}
					for k := range rawOpts {
						if !slices.Contains(knownOptions, k) {
							t.Fatalf("unknown option %q", k)
						}
					}
					if opts.Antimeridian != "" {
						t.Skip("Task 6")
					}

					router := routers[c.Network]
					if router == nil && c.Network != "marnet100" {
						router, err = searoute.NewRouter(networks[c.Network]())
						if err != nil {
							t.Fatal(err)
						}
						routers[c.Network] = router
					}
					var got any
					var gotErr error
					switch c.Fn {
					case "seaRoute":
						if router == nil {
							got, gotErr = searoute.SeaRoute(points[0], points[1], opts)
						} else {
							got, gotErr = router.SeaRoute(points[0], points[1], opts)
						}
					default:
						t.Skipf("%s is not implemented yet", c.Fn)
					}

					if c.Error != nil {
						var snapErr *searoute.SnapFailedError
						switch c.Error.Name {
						case "SnapFailedError":
							if !errors.As(gotErr, &snapErr) || snapErr.Side != c.Error.Side {
								t.Fatalf("want SnapFailedError on %s, got %v", c.Error.Side, gotErr)
							}
						case "NoRouteError":
							if !errors.Is(gotErr, searoute.ErrNoRoute) {
								t.Fatalf("want ErrNoRoute, got %v", gotErr)
							}
						default:
							if gotErr == nil {
								t.Fatalf("want %s, got no error", c.Error.Name)
							}
						}
						return
					}
					if gotErr != nil {
						t.Fatal(gotErr)
					}

					b, err := json.Marshal(got)
					if err != nil {
						t.Fatal(err)
					}
					var gotAny, wantAny any
					if err := json.Unmarshal(b, &gotAny); err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(c.Result, &wantAny); err != nil {
						t.Fatal(err)
					}
					gotList, ok := gotAny.([]any)
					if !ok {
						gotList = []any{gotAny}
					}
					wantList, ok := wantAny.([]any)
					if !ok {
						wantList = []any{wantAny}
					}
					if len(gotList) != len(wantList) {
						t.Fatalf("got %d routes, want %d", len(gotList), len(wantList))
					}

					for i := range wantList {
						g, w := gotList[i].(map[string]any), wantList[i].(map[string]any)
						gp, wp := g["properties"].(map[string]any), w["properties"].(map[string]any)
						gk, wk := make([]string, 0, len(gp)), make([]string, 0, len(wp))
						for k := range gp {
							gk = append(gk, k)
						}
						for k := range wp {
							wk = append(wk, k)
						}
						slices.Sort(gk)
						slices.Sort(wk)
						if !slices.Equal(gk, wk) {
							t.Fatalf("route %d: properties %v, want %v", i, gk, wk)
						}
						for k, wv := range wp {
							switch wv := wv.(type) {
							case float64:
								gv := gp[k].(float64)
								tol := math.Abs(wv) * 1e-3
								if wv == 0 {
									tol = 1e-9
								}
								if math.Abs(gv-wv) > tol {
									t.Errorf("route %d: %s = %v, want %v", i, k, gv, wv)
								}
							case []any:
								if k != "bbox" {
									if !reflect.DeepEqual(gp[k], wv) {
										t.Errorf("route %d: %s = %v, want %v", i, k, gp[k], wv)
									}
									continue
								}
								for j := range wv {
									if math.Abs(gp[k].([]any)[j].(float64)-wv[j].(float64)) > 1e-6 {
										t.Errorf("route %d: bbox = %v, want %v", i, gp[k], wv)
										break
									}
								}
							default:
								if gp[k] != wv {
									t.Errorf("route %d: %s = %v, want %v", i, k, gp[k], wv)
								}
							}
						}

						gg, wg := g["geometry"].(map[string]any), w["geometry"].(map[string]any)
						if gg["type"] != wg["type"] {
							t.Fatalf("route %d: geometry %v, want %v", i, gg["type"], wg["type"])
						}
						gLines, wLines := []any{gg["coordinates"]}, []any{wg["coordinates"]}
						if wg["type"] == "MultiLineString" {
							gLines, wLines = gg["coordinates"].([]any), wg["coordinates"].([]any)
						}
						if len(gLines) != len(wLines) {
							t.Fatalf("route %d: %d lines, want %d", i, len(gLines), len(wLines))
						}
						for l := range wLines {
							gl, wl := gLines[l].([]any), wLines[l].([]any)
							if len(gl) != len(wl) {
								t.Fatalf("route %d line %d: %d coordinates, want %d", i, l, len(gl), len(wl))
							}
							for j := range wl {
								gc, wc := gl[j].([]any), wl[j].([]any)
								if math.Abs(gc[0].(float64)-wc[0].(float64)) > 1e-6 || math.Abs(gc[1].(float64)-wc[1].(float64)) > 1e-6 {
									t.Fatalf("route %d line %d coordinate %d: %v, want %v", i, l, j, gc, wc)
								}
							}
						}
					}
				})
			}
		})
	}
}
