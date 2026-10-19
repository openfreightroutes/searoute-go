package searoute_test

import (
	"testing"

	searoute "github.com/openfreightroutes/searoute-go"
)

func BenchmarkSeaRoute(b *testing.B) {
	shanghai, rotterdam := searoute.Point{Lon: 121.5, Lat: 31.0}, searoute.Point{Lon: 4.4, Lat: 51.9}
	if _, err := searoute.SeaRoute(shanghai, rotterdam, searoute.Options{}); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		searoute.SeaRoute(shanghai, rotterdam, searoute.Options{})
	}
}
