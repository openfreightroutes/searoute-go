# searoute-go

Sea routes between two points on the world's shipping lanes. A Go port of [searoute-ts](https://github.com/openfreightroutes/searoute-ts) 2.3.1.

## Install

    go get github.com/openfreightroutes/searoute-go

## Usage

Points are `searoute.Point{Lon, Lat}`. Each `*Route` carries `Length` in `Options.Units` (default `searoute.NauticalMiles`; also `Kilometers`, `Meters`, `Miles`).

    shanghai := searoute.Point{Lon: 121.5, Lat: 31.0}
    rotterdam := searoute.Point{Lon: 4.4, Lat: 51.9}
    singapore := searoute.Point{Lon: 103.8, Lat: 1.3}

    // SeaRoute: shortest route between two points
    route, err := searoute.SeaRoute(shanghai, rotterdam, searoute.Options{Units: searoute.Kilometers})

    // SeaRouteMulti: one route through several points
    route, err = searoute.SeaRouteMulti([]searoute.Point{shanghai, singapore, rotterdam}, searoute.Options{})

    // SeaRouteAlternatives: up to k distinct routes
    routes, err := searoute.SeaRouteAlternatives(shanghai, rotterdam, 3, searoute.Options{})

    // NewRouter: reuse one router on a chosen network
    // (import "github.com/openfreightroutes/searoute-data/marnet50")
    router, err := searoute.NewRouter(marnet50.Network())
    route, err = router.SeaRoute(shanghai, rotterdam, searoute.Options{})

    // LoadNetwork: read your own GeoJSON FeatureCollection of LineStrings
    n, err := searoute.LoadNetwork(file)
    router, err = searoute.NewRouter(n)

The package-level functions use the 100 km network (`marnet100`). `marnet50` and `marnet20` are finer.

`Router` and the package-level functions are safe for concurrent use. `NewRouter` is the only way to make a `Router`.

`OriginSnapKm` and `DestinationSnapKm` are always in kilometres, whatever `Units` is. A zero option value means unset: for example, `SimilarityThreshold: 0` becomes 0.02.

## Ports and ECA zones

`ports.Resolve("CNSHA")` turns a UN/LOCODE into a `Point`. Import `github.com/openfreightroutes/searoute-go/eca` (for its side effect) to register the emission control areas used for ECA distance.

## Licence and attribution

Network data is copyright Eurostat, licensed under EUPL-1.2, and is supplied by [github.com/openfreightroutes/searoute-data](https://github.com/openfreightroutes/searoute-data). This library is a port of searoute-ts (MIT).
