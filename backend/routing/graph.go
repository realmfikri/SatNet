package routing

import (
	"errors"
	"math"

	"github.com/example/satnet/backend/visibility"
)

// NodeType differentiates between satellites and ground stations.
type NodeType string

const (
	// Satellite represents an on-orbit node.
	Satellite NodeType = "satellite"
	// Ground represents a ground station node.
	Ground NodeType = "ground"
)

// Node describes a satellite or ground station participating in routing.
type Node struct {
	ID       string
	Type     NodeType
	Position visibility.Vector3
}

// Edge captures link characteristics between two nodes.
type Edge struct {
	From       string
	To         string
	LatencyMS  float64
	Throughput float64
}

// Graph stores connectivity and edge weights.
type Graph struct {
	Nodes map[string]Node
	Adj   map[string][]Edge
}

// SpeedOfLightKMPerS defines the propagation speed for latency approximation.
const SpeedOfLightKMPerS = 299792.458

// BuildGraph constructs a bidirectional connectivity graph using line-of-sight rules.
// Latency is approximated as slant range divided by the speed of light (milliseconds),
// while throughput is inversely proportional to latency to represent distance loss.
func BuildGraph(nodes []Node, elevationMask float64) (*Graph, error) {
	g := &Graph{Nodes: make(map[string]Node), Adj: make(map[string][]Edge)}
	for _, n := range nodes {
		if n.ID == "" {
			return nil, errors.New("node ID cannot be empty")
		}
		g.Nodes[n.ID] = n
	}

	addEdge := func(a, b Node) {
		dist := visibility.SlantRange(a.Position, b.Position)
		latency := (dist / SpeedOfLightKMPerS) * 1000
		throughput := 1.0 / (1.0 + latency)
		edge := Edge{From: a.ID, To: b.ID, LatencyMS: latency, Throughput: throughput}
		g.Adj[a.ID] = append(g.Adj[a.ID], edge)
	}

	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			a, b := nodes[i], nodes[j]

			switch {
			case a.Type == Satellite && b.Type == Satellite:
				if visibility.SatelliteToSatelliteVisible(a.Position, b.Position) {
					addEdge(a, b)
					addEdge(b, a)
				}
			case a.Type == Ground && b.Type == Satellite:
				if visibility.GroundToSatelliteVisible(a.Position, b.Position, elevationMask) {
					addEdge(a, b)
					addEdge(b, a)
				}
			case a.Type == Satellite && b.Type == Ground:
				if visibility.GroundToSatelliteVisible(b.Position, a.Position, elevationMask) {
					addEdge(a, b)
					addEdge(b, a)
				}
			default:
				// Ground-to-ground links not supported in this model.
			}
		}
	}

	return g, nil
}

// Clone creates a deep copy of the graph for algorithms that mutate state.
func (g *Graph) Clone() *Graph {
	copyGraph := &Graph{Nodes: make(map[string]Node, len(g.Nodes)), Adj: make(map[string][]Edge, len(g.Adj))}
	for id, node := range g.Nodes {
		copyGraph.Nodes[id] = node
	}
	for id, edges := range g.Adj {
		cloned := make([]Edge, len(edges))
		copy(cloned, edges)
		copyGraph.Adj[id] = cloned
	}
	return copyGraph
}

// RemoveEdge deletes a directed edge from the adjacency list if present.
func (g *Graph) RemoveEdge(from, to string) {
	edges := g.Adj[from]
	filtered := edges[:0]
	for _, e := range edges {
		if e.To != to {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == 0 {
		delete(g.Adj, from)
		return
	}
	g.Adj[from] = filtered
}

// RemoveNode removes a node and any incident edges.
func (g *Graph) RemoveNode(id string) {
	delete(g.Nodes, id)
	delete(g.Adj, id)
	for from := range g.Adj {
		g.RemoveEdge(from, id)
	}
}

// Heuristic returns a straight-line latency estimate in milliseconds for A*.
// It defaults to zero when nodes are not present, yielding Dijkstra behavior.
func (g *Graph) Heuristic(from, to string) float64 {
	src, okSrc := g.Nodes[from]
	dst, okDst := g.Nodes[to]
	if !okSrc || !okDst {
		return 0
	}
	dist := visibility.SlantRange(src.Position, dst.Position)
	return (dist / SpeedOfLightKMPerS) * 1000
}

// Path represents an ordered path with cumulative metrics.
type Path struct {
	Nodes                []string
	LatencyMS            float64
	BottleneckThroughput float64
}

// computePathMetrics evaluates latency and bottleneck throughput along a path.
func (g *Graph) computePathMetrics(sequence []string) (float64, float64, error) {
	if len(sequence) < 2 {
		return 0, math.Inf(1), nil
	}
	totalLatency := 0.0
	bottleneck := math.Inf(1)
	for i := 0; i < len(sequence)-1; i++ {
		from, to := sequence[i], sequence[i+1]
		edgeFound := false
		for _, e := range g.Adj[from] {
			if e.To == to {
				totalLatency += e.LatencyMS
				if e.Throughput < bottleneck {
					bottleneck = e.Throughput
				}
				edgeFound = true
				break
			}
		}
		if !edgeFound {
			return 0, 0, errors.New("path references missing edge")
		}
	}
	return totalLatency, bottleneck, nil
}
