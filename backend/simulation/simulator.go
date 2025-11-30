package simulation

import (
	"errors"
	"sync"
	"time"

	"github.com/example/satnet/backend/coverage"
	"github.com/example/satnet/backend/routing"
	"github.com/example/satnet/backend/visibility"
)

// EventType enumerates the categories of frontend updates emitted by the simulator.
type EventType string

const (
	// EventTopologyUpdated signals that the connectivity or active satellites changed.
	EventTopologyUpdated EventType = "topology_updated"
	// EventCoverageUpdated indicates coverage metrics were recomputed.
	EventCoverageUpdated EventType = "coverage_updated"
)

// Event is published whenever the simulator recomputes state that should be pushed to the UI.
type Event struct {
	Type     EventType
	Snapshot Snapshot
}

// Satellite represents an on-orbit node with a configurable coverage footprint.
type Satellite struct {
	ID        string
	Position  visibility.Vector3
	Footprint coverage.Footprint
	Active    bool
}

// GroundStation represents a user gateway used as a traffic endpoint.
type GroundStation struct {
	ID       string
	Position visibility.Vector3
}

// TrafficDemand specifies a flow between two nodes for which routing is computed.
type TrafficDemand struct {
	ID     string
	FromID string
	ToID   string
}

// Config wires a simulator with nodes, demands, and modeling parameters.
type Config struct {
	Satellites     []Satellite
	GroundStations []GroundStation
	Traffic        []TrafficDemand
	GridConfig     coverage.GridConfig
	ElevationMask  float64
}

// Snapshot captures the network state and metrics exposed to the frontend.
type Snapshot struct {
	Timestamp          time.Time               `json:"timestamp"`
	ActiveSatellites   []string                `json:"activeSatellites"`
	DisabledSatellites []string                `json:"disabledSatellites"`
	Coverage           coverage.Summary        `json:"coverage"`
	Heatmap            []coverage.HeatmapCell  `json:"heatmap"`
	Routes             map[string]routing.Path `json:"routes"`
}

// Simulator manages network state, recomputes routing/coverage, and broadcasts updates.
type Simulator struct {
	mu            sync.Mutex
	elevationMask float64
	gridConfig    coverage.GridConfig
	satellites    map[string]*Satellite
	ground        map[string]GroundStation
	traffic       []TrafficDemand
	graph         *routing.Graph
	routes        map[string]routing.Path
	events        chan Event
	snapshot      Snapshot
}

// NewSimulator constructs a simulator from the provided configuration and computes the initial state.
func NewSimulator(cfg Config) (*Simulator, error) {
	if err := cfg.GridConfig.Validate(); err != nil {
		return nil, err
	}
	if len(cfg.Satellites) == 0 {
		return nil, errors.New("simulation requires at least one satellite")
	}
	if len(cfg.GroundStations) == 0 {
		return nil, errors.New("simulation requires at least one ground station")
	}

	sats := make(map[string]*Satellite, len(cfg.Satellites))
	for i := range cfg.Satellites {
		sat := cfg.Satellites[i]
		if sat.ID == "" {
			return nil, errors.New("satellite ID cannot be empty")
		}
		if _, exists := sats[sat.ID]; exists {
			return nil, errors.New("duplicate satellite ID")
		}
		sat.Active = true
		sats[sat.ID] = &sat
	}

	ground := make(map[string]GroundStation, len(cfg.GroundStations))
	for _, gs := range cfg.GroundStations {
		if gs.ID == "" {
			return nil, errors.New("ground station ID cannot be empty")
		}
		ground[gs.ID] = gs
	}

	sim := &Simulator{
		elevationMask: cfg.ElevationMask,
		gridConfig:    cfg.GridConfig,
		satellites:    sats,
		ground:        ground,
		traffic:       cfg.Traffic,
		routes:        make(map[string]routing.Path),
		events:        make(chan Event, 8),
	}

	if _, err := sim.recomputeLocked(); err != nil {
		return nil, err
	}

	return sim, nil
}

// NewDemoSimulator builds a simple network useful for manual testing of the API server.
func NewDemoSimulator() *Simulator {
	cfg := Config{
		GridConfig:    coverage.GridConfig{LatStep: 180, LonStep: 360},
		ElevationMask: 0,
		Satellites: []Satellite{
			{ID: "sat-alpha", Position: visibility.Vector3{X: visibility.EarthRadius + 400, Y: 0, Z: 0}, Footprint: coverage.Footprint{CenterLat: 0, CenterLon: 0, RadiusKm: 900, LinkStrength: 1}},
			{ID: "sat-beta", Position: visibility.Vector3{X: visibility.EarthRadius + 800, Y: 200, Z: 0}, Footprint: coverage.Footprint{CenterLat: 45, CenterLon: 90, RadiusKm: 900, LinkStrength: 0.8}},
		},
		GroundStations: []GroundStation{
			{ID: "ground-1", Position: visibility.Vector3{X: visibility.EarthRadius, Y: 0, Z: 0}},
			{ID: "ground-2", Position: visibility.Vector3{X: visibility.EarthRadius, Y: 10, Z: 0}},
		},
		Traffic: []TrafficDemand{{ID: "demo", FromID: "ground-1", ToID: "ground-2"}},
	}

	sim, err := NewSimulator(cfg)
	if err != nil {
		// The demo should never fail; panic to surface configuration issues.
		panic(err)
	}
	return sim
}

// Events exposes a read-only channel of simulator updates for streaming to the frontend.
func (s *Simulator) Events() <-chan Event {
	return s.events
}

// Snapshot returns the latest computed state.
func (s *Simulator) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshot
}

// DisableSatellite marks a satellite inactive and recomputes the network.
func (s *Simulator) DisableSatellite(id string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sat, ok := s.satellites[id]
	if !ok {
		return Snapshot{}, errors.New("unknown satellite")
	}
	sat.Active = false
	return s.recomputeLocked()
}

// RemoveSatellite deletes a satellite entirely and recomputes the network.
func (s *Simulator) RemoveSatellite(id string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.satellites[id]; !ok {
		return Snapshot{}, errors.New("unknown satellite")
	}
	delete(s.satellites, id)
	return s.recomputeLocked()
}

// Recompute forces visibility, routing, and coverage to refresh without altering topology.
func (s *Simulator) Recompute() (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recomputeLocked()
}

func (s *Simulator) recomputeLocked() (Snapshot, error) {
	nodes := make([]routing.Node, 0, len(s.satellites)+len(s.ground))
	activeIDs := make([]string, 0, len(s.satellites))
	disabledIDs := make([]string, 0)
	footprints := make([]coverage.Footprint, 0, len(s.satellites))

	for _, sat := range s.satellites {
		if sat.Active {
			nodes = append(nodes, routing.Node{ID: sat.ID, Type: routing.Satellite, Position: sat.Position})
			activeIDs = append(activeIDs, sat.ID)
			footprints = append(footprints, sat.Footprint)
		} else {
			disabledIDs = append(disabledIDs, sat.ID)
		}
	}
	for _, gs := range s.ground {
		nodes = append(nodes, routing.Node{ID: gs.ID, Type: routing.Ground, Position: gs.Position})
	}

	graph, err := routing.BuildGraph(nodes, s.elevationMask)
	if err != nil {
		return Snapshot{}, err
	}
	s.graph = graph

	routes := make(map[string]routing.Path, len(s.traffic))
	for _, demand := range s.traffic {
		path, err := routing.ShortestPath(graph, demand.FromID, demand.ToID, func(id string) float64 {
			return graph.Heuristic(id, demand.ToID)
		})
		if err == nil {
			routes[demand.ID] = path
		}
	}
	s.routes = routes

	grid, err := coverage.NewCoverageGrid(s.gridConfig)
	if err != nil {
		return Snapshot{}, err
	}
	grid.ApplyFootprints(footprints)
	summary := grid.Summarize()

	snapshot := Snapshot{
		Timestamp:          time.Now().UTC(),
		ActiveSatellites:   activeIDs,
		DisabledSatellites: disabledIDs,
		Coverage:           summary,
		Heatmap:            grid.HeatmapData(),
		Routes:             routes,
	}

	s.snapshot = snapshot

	s.publishEvent(EventTopologyUpdated, snapshot)
	s.publishEvent(EventCoverageUpdated, snapshot)

	return snapshot, nil
}

func (s *Simulator) publishEvent(eventType EventType, snapshot Snapshot) {
	select {
	case s.events <- Event{Type: eventType, Snapshot: snapshot}:
	default:
		// Drop the event when the channel is full to avoid blocking the caller.
	}
}
