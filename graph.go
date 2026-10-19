package searoute

import (
	"maps"
	"math"
	"slices"

	"github.com/openfreightroutes/searoute-data/network"
)

// geojson-path-finder breaks ties by JavaScript object key order, so adjacency keeps insertion order.
type orderedEdges struct {
	keys    [][2]float64
	weights map[[2]float64]float64
}

func (o *orderedEdges) set(k [2]float64, w float64) {
	if _, ok := o.weights[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.weights[k] = w
}

func (o *orderedEdges) delete(k [2]float64) {
	if _, ok := o.weights[k]; ok {
		i := slices.Index(o.keys, k)
		o.keys = slices.Delete(o.keys, i, i+1)
		delete(o.weights, k)
	}
}

func (o *orderedEdges) clone() *orderedEdges {
	return &orderedEdges{keys: slices.Clone(o.keys), weights: maps.Clone(o.weights)}
}

type graph struct {
	vertices        map[[2]float64]*orderedEdges
	sourceCoords    map[[2]float64][2]float64
	compacted       map[[2]float64]*orderedEdges
	compactedCoords map[[2][2]float64][][2]float64
}

func buildGraph(n *network.Network, weight func(a, b [2]float64, pass string) float64) *graph {
	g := &graph{
		vertices:        map[[2]float64]*orderedEdges{},
		sourceCoords:    map[[2]float64][2]float64{},
		compacted:       map[[2]float64]*orderedEdges{},
		compactedCoords: map[[2][2]float64][][2]float64{},
	}
	keys := make([][][2]float64, len(n.Features))
	for i, f := range n.Features {
		for _, c := range f.Coords {
			var k [2]float64
			for ax, x := range c {
				// JavaScript Math.round rounds halves toward +Inf; math.Round rounds them away from zero.
				r := math.Round(x / 1e-5)
				if x/1e-5-r == 0.5 {
					r++
				}
				k[ax] = r * 1e-5
			}
			keys[i] = append(keys[i], k)
			g.sourceCoords[k] = c
		}
	}

	var order [][2]float64
	for i, f := range n.Features {
		for j := 1; j < len(keys[i]); j++ {
			a, b := keys[i][j-1], keys[i][j]
			w := weight(g.sourceCoords[a], g.sourceCoords[b], f.Pass)
			if w == 0 {
				continue
			}
			for _, k := range [][2]float64{a, b} {
				if g.vertices[k] == nil {
					g.vertices[k] = &orderedEdges{weights: map[[2]float64]float64{}}
					order = append(order, k)
				}
			}
			g.vertices[a].set(b, w)
			g.vertices[b].set(a, w)
		}
	}

	for _, k := range order {
		g.compacted[k] = g.vertices[k].clone()
		for _, nb := range g.vertices[k].keys {
			g.compactedCoords[[2][2]float64{k, nb}] = [][2]float64{g.sourceCoords[k]}
		}
	}
	for _, k := range order {
		// Every source edge has its reverse, so shouldCompact reduces to "exactly two neighbours".
		if len(g.vertices[k].keys) != 2 {
			continue
		}
		vertex := g.compacted[k]
		edges := slices.Clone(vertex.keys)
		if len(edges) == 0 {
			continue
		}
		for _, nb := range edges {
			for _, other := range edges {
				if nb == other {
					continue
				}
				for _, p := range [][2][2]float64{{nb, other}, {other, nb}} {
					neighbor := g.compacted[p[0]]
					_, has := neighbor.weights[p[1]]
					back, hasBack := neighbor.weights[k]
					if !has && hasBack {
						neighbor.set(p[1], back+vertex.weights[p[1]])
						g.compactedCoords[p] = slices.Concat(g.compactedCoords[[2][2]float64{p[0], k}], g.compactedCoords[[2][2]float64{k, p[1]}])
					}
				}
			}
		}
		for _, nb := range edges {
			g.compacted[nb].delete(k)
			delete(g.compactedCoords, [2][2]float64{nb, k})
			delete(g.compactedCoords, [2][2]float64{k, nb})
		}
		delete(g.compacted, k)
	}
	return g
}

func (g *graph) findPath(start, finish [2]float64) ([][2]float64, bool) {
	var keys [2][2]float64
	for i, p := range [2][2]float64{start, finish} {
		for ax, x := range p {
			r := math.Round(x / 1e-5)
			if x/1e-5-r == 0.5 {
				r++
			}
			keys[i][ax] = r * 1e-5
		}
	}
	start, finish = keys[0], keys[1]
	if g.vertices[start] == nil || g.vertices[finish] == nil {
		return nil, false
	}

	// Phantom nodes go into a per-call overlay instead of g, so concurrent searches can share g.
	adj := map[[2]float64]*orderedEdges{}
	coords := map[[2][2]float64][][2]float64{}
	for _, n := range [][2]float64{start, finish} {
		if g.compacted[n] != nil || adj[n] != nil {
			continue
		}
		edges := &orderedEdges{weights: map[[2]float64]float64{}}
		incomingW := map[[2]float64]float64{}
		incomingCoords := map[[2]float64][][2]float64{}
		for _, j := range g.vertices[n].keys {
			prev, vk := n, j
			w := g.vertices[prev].weights[vk]
			rw := g.vertices[vk].weights[prev]
			var forkCoords, path [][2]float64
			for adj[vk] == nil && g.compacted[vk] == nil {
				vEdges := g.vertices[vk]
				var next [2]float64
				for _, k := range vEdges.keys {
					if k != prev {
						next = k
						break
					}
				}
				w += vEdges.weights[next]
				rw += g.vertices[next].weights[vk]
				if slices.Contains(path, vk) {
					adj[vk] = vEdges.clone()
					break
				}
				path = append(path, vk)
				forkCoords = append(forkCoords, g.sourceCoords[vk])
				prev, vk = vk, next
			}
			if vk == n {
				continue
			}
			if cur, ok := edges.weights[vk]; !ok || cur > w {
				edges.set(vk, w)
				coords[[2][2]float64{n, vk}] = append([][2]float64{g.sourceCoords[n]}, forkCoords...)
			}
			if cur, ok := incomingW[vk]; !ok || cur > rw {
				incomingW[vk] = rw
				c := append([][2]float64{g.sourceCoords[n]}, forkCoords...)
				slices.Reverse(c)
				incomingCoords[vk] = c
			}
		}
		adj[n] = edges
		// incomingEdges gets exactly the keys of edges, in the same order.
		for _, nb := range edges.keys {
			if adj[nb] == nil {
				adj[nb] = g.compacted[nb].clone()
			}
			adj[nb].set(n, incomingW[nb])
			in := incomingCoords[nb]
			coords[[2][2]float64{nb, n}] = append([][2]float64{g.sourceCoords[nb]}, in[:len(in)-1]...)
		}
	}

	type state struct {
		cost   float64
		node   [2]float64
		parent int
	}
	states := []state{{0, start, -1}}
	heap := []int{0}
	costs := map[[2]float64]float64{start: 0}
	for len(heap) > 0 {
		top := heap[0]
		bottom := heap[len(heap)-1]
		heap = heap[:len(heap)-1]
		if len(heap) > 0 {
			pos := 0
			for pos < len(heap)>>1 {
				left := pos<<1 + 1
				best := heap[left]
				if right := left + 1; right < len(heap) && states[heap[right]].cost < states[best].cost {
					left = right
					best = heap[right]
				}
				if states[best].cost >= states[bottom].cost {
					break
				}
				heap[pos] = best
				pos = left
			}
			heap[pos] = bottom
		}

		s := states[top]
		if s.node == finish {
			var nodes [][2]float64
			for i := top; i >= 0; i = states[i].parent {
				nodes = append(nodes, states[i].node)
			}
			slices.Reverse(nodes)
			var out [][2]float64
			for i := 1; i < len(nodes); i++ {
				key := [2][2]float64{nodes[i-1], nodes[i]}
				c := coords[key]
				if c == nil {
					c = g.compactedCoords[key]
				}
				out = append(out, c...)
			}
			return append(out, g.sourceCoords[finish]), true
		}

		neighbours := adj[s.node]
		if neighbours == nil {
			neighbours = g.compacted[s.node]
		}
		for _, nb := range neighbours.keys {
			c := s.cost + neighbours.weights[nb]
			if seen, ok := costs[nb]; c < math.Inf(1) && (!ok || c < seen) {
				costs[nb] = c
				states = append(states, state{c, nb, top})
				pos := len(heap)
				heap = append(heap, len(states)-1)
				item := heap[pos]
				for pos > 0 {
					parent := (pos - 1) >> 1
					if states[item].cost >= states[heap[parent]].cost {
						break
					}
					heap[pos] = heap[parent]
					pos = parent
				}
				heap[pos] = item
			}
		}
	}
	return nil, false
}
