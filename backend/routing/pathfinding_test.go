package routing

import (
	"math"
	"testing"

	"github.com/example/satnet/backend/visibility"
)

func testNodes() []Node {
	er := visibility.EarthRadius
	diag := er / math.Sqrt2
	return []Node{
		{ID: "ground-a", Type: Ground, Position: visibility.Vector3{X: er, Y: 0, Z: 0}},
		{ID: "ground-b", Type: Ground, Position: visibility.Vector3{X: diag, Y: diag, Z: 0}},
		{ID: "sat-alpha", Type: Satellite, Position: visibility.Vector3{X: er + 1500, Y: er + 1500, Z: 0}},
		{ID: "sat-beta", Type: Satellite, Position: visibility.Vector3{X: er + 4000, Y: er + 4000, Z: 0}},
		{ID: "sat-gamma", Type: Satellite, Position: visibility.Vector3{X: er + 1500, Y: er + 4000, Z: 0}},
	}
}

func TestShortestPathPrefersLowerLatency(t *testing.T) {
	g, err := BuildGraph(testNodes(), 0)
	if err != nil {
		t.Fatalf("failed to build graph: %v", err)
	}

	heuristic := func(n string) float64 { return g.Heuristic(n, "ground-b") }
	path, err := ShortestPath(g, "ground-a", "ground-b", heuristic)
	if err != nil {
		t.Fatalf("expected path, got error: %v", err)
	}

	expected := []string{"ground-a", "sat-alpha", "ground-b"}
	if len(path.Nodes) != len(expected) {
		t.Fatalf("unexpected path length: %v", path.Nodes)
	}
	for i, n := range expected {
		if path.Nodes[i] != n {
			t.Fatalf("expected node %s at %d, got %s", n, i, path.Nodes[i])
		}
	}
	if path.LatencyMS <= 0 || path.BottleneckThroughput <= 0 {
		t.Fatalf("path metrics not computed: %+v", path)
	}
}

func TestKAlternativeRoutesProvidesBackup(t *testing.T) {
	g, err := BuildGraph(testNodes(), 0)
	if err != nil {
		t.Fatalf("failed to build graph: %v", err)
	}

	paths, err := KAlternativeRoutes(g, "ground-a", "ground-b", 3)
	if err != nil {
		t.Fatalf("expected routes, got error: %v", err)
	}

	if len(paths) < 2 {
		t.Fatalf("expected at least two routes, got %d", len(paths))
	}

	primary := paths[0]
	backup := paths[1]
	if equalPrefix(primary.Nodes, backup.Nodes) {
		t.Fatalf("backup route should differ from primary: %v vs %v", primary.Nodes, backup.Nodes)
	}
}

func TestFailureHandlingReroutes(t *testing.T) {
	g, err := BuildGraph(testNodes(), 0)
	if err != nil {
		t.Fatalf("failed to build graph: %v", err)
	}

	degraded := g.Clone()
	degraded.RemoveNode("sat-alpha")

	path, err := ShortestPath(degraded, "ground-a", "ground-b", func(string) float64 { return 0 })
	if err != nil {
		t.Fatalf("expected reroute despite failure, got error: %v", err)
	}

	for _, n := range path.Nodes {
		if n == "sat-alpha" {
			t.Fatalf("failed path should not include removed satellite: %v", path.Nodes)
		}
	}

	if path.LatencyMS <= 0 || path.BottleneckThroughput <= 0 || math.IsInf(path.BottleneckThroughput, 0) {
		t.Fatalf("invalid path metrics after reroute: %+v", path)
	}
}
