package main

import (
	"log"

	"github.com/example/satnet/backend/internal/api"
)

func main() {
	server := api.NewServer(":8080")
	if err := server.Start(); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
