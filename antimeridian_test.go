package searoute

import (
	"reflect"
	"testing"
)

func TestUnwrap(t *testing.T) {
	got := unwrapCoords([][2]float64{{170, 0}, {-170, 1}, {-160, 2}})
	want := [][2]float64{{170, 0}, {190, 1}, {200, 2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	plain := [][2]float64{{10, 0}, {20, 1}}
	if got := unwrapCoords(plain); !reflect.DeepEqual(got, plain) {
		t.Fatalf("got %v, want %v", got, plain)
	}
}

func TestSplit(t *testing.T) {
	got := splitAtAntimeridian([][2]float64{{170, 0}, {-170, 1}})
	want := [][][2]float64{{{170, 0}, {180, 0.5}}, {{-180, 0.5}, {-170, 1}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestUnknownAntimeridian(t *testing.T) {
	_, err := SeaRoute(Point{121.5, 31.0}, Point{4.4, 51.9}, Options{Antimeridian: "wrap"})
	if err == nil {
		t.Fatal("want error")
	}
}
