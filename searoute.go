package searoute

import (
	"sync"

	"github.com/openfreightroutes/searoute-data/marnet100"
)

var defaultRouter = sync.OnceValue(func() *Router {
	r, _ := NewRouter(marnet100.Network())
	return r
})

func SeaRoute(origin, destination Point, opts Options) (*Route, error) {
	return defaultRouter().SeaRoute(origin, destination, opts)
}

func SeaRouteMulti(points []Point, opts Options) (*Route, error) {
	return defaultRouter().SeaRouteMulti(points, opts)
}

func SeaRouteAlternatives(origin, destination Point, k int, opts Options) ([]*Route, error) {
	return defaultRouter().SeaRouteAlternatives(origin, destination, k, opts)
}
