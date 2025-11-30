package simulation

import "time"

type Snapshot struct {
	Timestamp   time.Time `json:"timestamp"`
	ActiveNodes int       `json:"activeNodes"`
	Notes       string    `json:"notes"`
}

type Simulator struct{}

func NewSimulator() *Simulator {
	return &Simulator{}
}

func (s *Simulator) Snapshot() Snapshot {
	return Snapshot{
		Timestamp:   time.Now().UTC(),
		ActiveNodes: 3,
		Notes:       "placeholder topology until real simulation logic lands",
	}
}
