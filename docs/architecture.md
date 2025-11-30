# Architecture Overview

SatNet is organized around a clear boundary between the simulation backend and the 3D visualization frontend.

## Backend (Go)
- Located in `backend/` with a Go module dedicated to the API and simulation logic.
- `cmd/api/main.go` hosts the entrypoint for the HTTP server.
- `internal/api` wires routes for health checks and simulation snapshots.
- `internal/simulation` will house constellation dynamics, contact planning, and other domain logic.

## Frontend (CesiumJS + Three.js)
- Located in `frontend/` as a Vite-powered single-page app.
- `src/main.js` pulls version information from Cesium and Three to verify bundling and dependency wiring.
- Future updates should integrate with the backend snapshot endpoint to visualize orbits and links.

## Docs & Scripts
- `docs/` contains architecture and usage notes to guide onboarding.
- `scripts/` provides helper scripts for local setup and development workflows.
