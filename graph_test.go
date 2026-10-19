package searoute

import (
	"reflect"
	"sync"
	"testing"

	"github.com/openfreightroutes/searoute-data/network"
)

func testWeight(a, b [2]float64, pass string) float64 {
	if pass == "suez" {
		return 0
	}
	return haversineKm(a, b)
}

func TestFindPathTieBreak(t *testing.T) {
	g := buildGraph(&network.Network{Features: []network.Feature{
		{Coords: [][2]float64{{0, -2}, {0, -1}}},
		{Coords: [][2]float64{{0, -1}, {1, 0}}},
		{Coords: [][2]float64{{1, 0}, {0, 1}}},
		{Coords: [][2]float64{{0, -1}, {-1, 0}}},
		{Coords: [][2]float64{{-1, 0}, {0, 1}}},
		{Coords: [][2]float64{{0, 1}, {0, 2}}},
	}}, testWeight)
	got, ok := g.findPath([2]float64{0, -2}, [2]float64{0, 2})
	want := [][2]float64{{0, -2}, {0, -1}, {1, 0}, {0, 1}, {0, 2}}
	if !ok || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, %v want %v", got, ok, want)
	}
}

var phantomNetwork = &network.Network{Features: []network.Feature{
	{Coords: [][2]float64{{0, 0}, {1, 0}, {2, 0}, {3, 0}}},
	{Coords: [][2]float64{{3, 0}, {3, 1}}},
	{Coords: [][2]float64{{3, 0}, {4, 0}}},
}}

func TestFindPathPhantom(t *testing.T) {
	got, ok := buildGraph(phantomNetwork, testWeight).findPath([2]float64{1, 0}, [2]float64{4, 0})
	want := [][2]float64{{1, 0}, {2, 0}, {3, 0}, {4, 0}}
	if !ok || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, %v want %v", got, ok, want)
	}
}

func TestFindPathBlocked(t *testing.T) {
	g := buildGraph(&network.Network{Features: []network.Feature{
		{Pass: "suez", Coords: [][2]float64{{0, 0}, {1, 0}}},
		{Coords: [][2]float64{{1, 0}, {2, 0}}},
	}}, testWeight)
	if got, ok := g.findPath([2]float64{0, 0}, [2]float64{2, 0}); ok {
		t.Fatalf("expected no path, got %v", got)
	}
}

func TestFindPathRounding(t *testing.T) {
	g := buildGraph(&network.Network{Features: []network.Feature{
		{Coords: [][2]float64{{0, 0}, {1, 0}}},
		{Coords: [][2]float64{{1.0000001, 0}, {2, 0}}},
	}}, testWeight)
	got, ok := g.findPath([2]float64{0, 0}, [2]float64{2, 0})
	if !ok || len(got) != 3 {
		t.Fatalf("got %v, %v", got, ok)
	}
}

func TestFindPathRoundsNegativeHalfLikeJS(t *testing.T) {
	g := buildGraph(&network.Network{Features: []network.Feature{
		{Coords: [][2]float64{{-0.000025, 0}, {1, 0}}},
	}}, testWeight)
	if g.vertices[[2]float64{-0.00002, 0}] == nil {
		t.Fatalf("expected key {-0.00002 0}, got vertices %v", g.vertices)
	}
	got, ok := g.findPath([2]float64{-0.000025, 0}, [2]float64{1, 0})
	if want := [][2]float64{{-0.000025, 0}, {1, 0}}; !ok || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, %v want %v", got, ok, want)
	}
}

func TestFindPathDoesNotMutate(t *testing.T) {
	g := buildGraph(phantomNetwork, testWeight)
	searches := [][2][2]float64{
		{{1, 0}, {4, 0}},
		{{1, 0}, {4, 0}},
		{{2, 0}, {1, 0}},
	}
	wants := make([][][2]float64, len(searches))
	for i, s := range searches {
		got, gotOK := g.findPath(s[0], s[1])
		want, wantOK := buildGraph(phantomNetwork, testWeight).findPath(s[0], s[1])
		if gotOK != wantOK || !reflect.DeepEqual(got, want) {
			t.Fatalf("%v: got %v, %v want %v, %v", s, got, gotOK, want, wantOK)
		}
		wants[i] = want
		if s == searches[2] && !reflect.DeepEqual(want, [][2]float64{{2, 0}, {1, 0}}) {
			t.Fatalf("%v: got %v want [[2 0] [1 0]]", s, want)
		}
	}

	var wg sync.WaitGroup
	for range 8 {
		for i, s := range searches {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if got, _ := g.findPath(s[0], s[1]); !reflect.DeepEqual(got, wants[i]) {
					t.Errorf("concurrent %v: got %v want %v", s, got, wants[i])
				}
			}()
		}
	}
	wg.Wait()
}
