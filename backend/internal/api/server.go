package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/example/satnet/backend/simulation"
)

type Server struct {
	addr string
	sim  *simulation.Simulator
}

type healthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

type simulationResponse struct {
	Message  string              `json:"message"`
	Snapshot simulation.Snapshot `json:"snapshot"`
}

func NewServer(addr string) *Server {
	return &Server{
		addr: addr,
		sim:  simulation.NewDemoSimulator(),
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/simulation/snapshot", s.snapshotHandler)

	srv := &http.Server{
		Addr:         s.addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	log.Printf("API server listening on %s", s.addr)
	return srv.ListenAndServe()
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, healthResponse{Status: "ok", Time: time.Now().UTC().Format(time.RFC3339)})
}

func (s *Server) snapshotHandler(w http.ResponseWriter, r *http.Request) {
	snap := s.sim.Snapshot()
	writeJSON(w, simulationResponse{Message: "current simulation state", Snapshot: snap})
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to write response: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
