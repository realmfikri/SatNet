package routing

import (
	"container/heap"
	"errors"
	"fmt"
)

type nodeCost struct {
	id    string
	cost  float64
	g     float64
	path  []string
	index int
}

type priorityQueue []*nodeCost

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].cost < pq[j].cost }
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*nodeCost)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1
	*pq = old[:n-1]
	return item
}

// ShortestPath returns the latency-optimal path using Dijkstra or A* when a heuristic is provided.
func ShortestPath(g *Graph, start, goal string, heuristic func(string) float64) (Path, error) {
	if heuristic == nil {
		heuristic = func(string) float64 { return 0 }
	}
	if _, ok := g.Nodes[start]; !ok {
		return Path{}, fmt.Errorf("unknown start node %s", start)
	}
	if _, ok := g.Nodes[goal]; !ok {
		return Path{}, fmt.Errorf("unknown goal node %s", goal)
	}

	openSet := &priorityQueue{}
	heap.Init(openSet)
	heap.Push(openSet, &nodeCost{id: start, cost: heuristic(start), g: 0, path: []string{start}})

	visited := make(map[string]float64)

	for openSet.Len() > 0 {
		current := heap.Pop(openSet).(*nodeCost)
		if prev, ok := visited[current.id]; ok && current.g >= prev {
			continue
		}
		visited[current.id] = current.g

		if current.id == goal {
			latency, throughput, err := g.computePathMetrics(current.path)
			if err != nil {
				return Path{}, err
			}
			return Path{Nodes: current.path, LatencyMS: latency, BottleneckThroughput: throughput}, nil
		}

		for _, edge := range g.Adj[current.id] {
			tentativeG := current.g + edge.LatencyMS
			estimate := tentativeG + heuristic(edge.To)
			newPath := append(append([]string{}, current.path...), edge.To)
			heap.Push(openSet, &nodeCost{id: edge.To, cost: estimate, g: tentativeG, path: newPath})
		}
	}

	return Path{}, errors.New("no route available")
}

// KAlternativeRoutes computes up to k loopless shortest paths using Yen's algorithm.
func KAlternativeRoutes(g *Graph, start, goal string, k int) ([]Path, error) {
	if k <= 0 {
		return nil, errors.New("k must be positive")
	}

	base := g.Clone()
	primary, err := ShortestPath(base, start, goal, func(string) float64 { return 0 })
	if err != nil {
		return nil, err
	}

	paths := []Path{primary}
	potential := &priorityQueue{}
	heap.Init(potential)

	for pathIndex := 1; pathIndex < k; pathIndex++ {
		previousPath := paths[pathIndex-1]
		for i := 0; i < len(previousPath.Nodes)-1; i++ {
			spurNode := previousPath.Nodes[i]
			rootPath := previousPath.Nodes[:i+1]

			spurGraph := base.Clone()

			for _, p := range paths {
				if len(p.Nodes) > i && equalPrefix(rootPath, p.Nodes[:i+1]) {
					spurGraph.RemoveEdge(p.Nodes[i], p.Nodes[i+1])
				}
			}
			for _, rootNode := range rootPath[:len(rootPath)-1] {
				spurGraph.RemoveNode(rootNode)
			}

			spurPath, err := ShortestPath(spurGraph, spurNode, goal, func(string) float64 { return 0 })
			if err != nil {
				continue
			}

			newPathNodes := append(append([]string{}, rootPath[:len(rootPath)-1]...), spurPath.Nodes...)
			latency, throughput, err := base.computePathMetrics(newPathNodes)
			if err != nil {
				continue
			}

			heap.Push(potential, &nodeCost{id: "", cost: latency, g: throughput, path: newPathNodes})
		}

		if potential.Len() == 0 {
			break
		}

		candidate := heap.Pop(potential).(*nodeCost)
		paths = append(paths, Path{Nodes: candidate.path, LatencyMS: candidate.cost, BottleneckThroughput: candidate.g})
	}

	return paths, nil
}

func equalPrefix(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
