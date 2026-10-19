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
