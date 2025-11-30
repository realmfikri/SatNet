# SatNet

SatNet is a playground for simulating satellite constellations and visualizing them in 3D. The repository is split into a Go backend that serves simulation data and a CesiumJS/Three.js frontend for interactive exploration.

## Repository layout
- `backend/` — Go module with the API server and simulation logic.
- `frontend/` — Vite-powered web app using CesiumJS and Three.js.
- `docs/` — Reference material covering architecture and usage.
- `scripts/` — Helper scripts for bootstrapping and running both services together.

## Getting started
### Prerequisites
- Go 1.21+
- Node.js 20+

### Backend
```bash
cd backend
# Run the test suite
go test ./...
# Start the API server on :8080
go run ./cmd/api
```

### Frontend
```bash
cd frontend
npm install
# Launch the dev server (defaults to http://localhost:5173)
npm run dev
# Lint and test
npm run lint
npm test
```

### Full-stack workflow
Use the provided scripts to streamline local development:
```bash
# One-time dependency installation
./scripts/setup.sh
# Start both backend and frontend (Ctrl+C to stop)
./scripts/dev.sh
```

## Documentation
- Architecture overview: [`docs/architecture.md`](docs/architecture.md)
- Usage guide: [`docs/usage.md`](docs/usage.md)

## Next steps
- Replace the placeholder simulation with orbital dynamics and link budget modeling.
- Bind the frontend to `/simulation/snapshot` for live constellation views.
- Add CI workflows for linting, testing, and building both stacks.
