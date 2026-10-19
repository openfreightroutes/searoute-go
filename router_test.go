package searoute

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"slices"
	"sync"
	"testing"

	"github.com/openfreightroutes/searoute-data/marnet100"
)

func TestShanghaiRotterdam(t *testing.T) {
	r, err := SeaRoute(Point{121.5, 31.0}, Point{4.4, 51.9}, Options{Units: Kilometers, ReturnPassages: true})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Passages, []string{"malacca", "babelmandeb", "suez", "gibraltar", "dover"}) {
		t.Fatalf("passages %v", r.Passages)
	}
	if math.Abs(r.Length-19752.85044899524) > 19752.85044899524*1e-3 {
		t.Fatalf("length %v", r.Length)
	}
	if len(r.Coordinates) != 112 {
		t.Fatalf("got %d coordinates", len(r.Coordinates))
	}
}

func TestSameVertex(t *testing.T) {
	_, err := SeaRoute(Point{2.1734, 41.3851}, Point{2.1854, 41.3851}, Options{})
	if !errors.Is(err, ErrNoRoute) {
		t.Fatalf("err = %v", err)
	}
}

func TestUnknownUnits(t *testing.T) {
	if _, err := SeaRoute(Point{121.5, 31.0}, Point{4.4, 51.9}, Options{Units: Units("furlongs")}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestMarshalJSONOrder(t *testing.T) {
	r, err := SeaRoute(Point{121.5, 31.0}, Point{4.4, 51.9}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte(`{"type":"Feature","properties":{"length":`)) {
		t.Fatalf("got %s", b[:80])
	}
}

func TestConcurrentRestrictionSets(t *testing.T) {
	router, err := NewRouter(marnet100.Network())
	if err != nil {
		t.Fatal(err)
	}
	sets := [][]string{nil, {"suez"}}
	var want []*Route
	for _, s := range sets {
		r, err := SeaRoute(Point{121.5, 31.0}, Point{4.4, 51.9}, Options{Restrictions: s})
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, r)
	}
	if reflect.DeepEqual(want[0], want[1]) {
		t.Fatal("restricting suez did not change the route")
	}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := router.SeaRoute(Point{121.5, 31.0}, Point{4.4, 51.9}, Options{Restrictions: sets[i%2]})
			if err != nil {
				t.Error(err)
				return
			}
			if !reflect.DeepEqual(got, want[i%2]) {
				t.Errorf("goroutine %d: route differs from the sequential one", i)
			}
		}()
	}
	wg.Wait()
}

func TestNewRouterEmpty(t *testing.T) {
	if _, err := NewRouter(nil); err == nil {
		t.Fatal("expected an error for a nil network")
	}
}

func TestSegmentIntersectsBBox(t *testing.T) {
	suez := [4]float64{32.0, 29.5, 32.9, 31.5}
	if !segmentIntersectsBBox([2]float64{31, 30}, [2]float64{34, 30}, suez) {
		t.Fatal("segment through the box not detected")
	}
	if segmentIntersectsBBox([2]float64{31, 32}, [2]float64{34, 32}, suez) {
		t.Fatal("segment north of the box reported")
	}
	if !segmentIntersectsBBox([2]float64{32.5, 30}, [2]float64{32.5, 30}, suez) {
		t.Fatal("point inside the box not detected")
	}
}

func TestUnknownPassage(t *testing.T) {
	for _, o := range []Options{{Restrictions: []string{"atlantis"}}, {Via: []string{"atlantis"}}} {
		if _, err := SeaRoute(Point{121.5, 31.0}, Point{4.4, 51.9}, o); !errors.Is(err, ErrUnknownPassage) {
			t.Fatalf("%+v: err = %v", o, err)
		}
	}
}

func TestBlockedSnapVertex(t *testing.T) {
	r, err := SeaRoute(Point{32.55, 30.5}, Point{4.4, 51.9}, Options{Restrictions: []string{"suez"}, ReturnPassages: true})
	if err != nil {
		if !errors.Is(err, ErrNoRoute) {
			t.Fatal(err)
		}
		return
	}
	if slices.Contains(r.Passages, "suez") {
		t.Fatalf("passages %v", r.Passages)
	}
}

func TestDraftBlocksPanama(t *testing.T) {
	r, err := SeaRoute(Point{-74.0, 40.5}, Point{-118.3, 33.7}, Options{VesselDraftMeters: 15.3, ReturnPassages: true})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(r.Passages, "panama") {
		t.Fatalf("passages %v", r.Passages)
	}
}

func TestViaContradiction(t *testing.T) {
	_, err := SeaRoute(Point{121.5, 31.0}, Point{4.4, 51.9}, Options{Via: []string{"suez"}, Restrictions: []string{"suez"}})
	if !errors.Is(err, ErrNoRoute) {
		t.Fatalf("err = %v", err)
	}
}

func TestMultiTooFew(t *testing.T) {
	if _, err := SeaRouteMulti([]Point{{121.5, 31.0}}, Options{}); err == nil {
		t.Fatal("want error")
	}
}

func TestAlternativesK(t *testing.T) {
	routes, err := SeaRouteAlternatives(Point{121.5, 31.0}, Point{4.4, 51.9}, 3, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) == 0 || len(routes) > 3 || routes[0].Variant != "baseline" {
		t.Fatalf("got %d routes", len(routes))
	}
	for i := 1; i < len(routes); i++ {
		if routes[i].Length < routes[i-1].Length {
			t.Fatalf("lengths not ascending at %d", i)
		}
	}
}
