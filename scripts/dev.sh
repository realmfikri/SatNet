#!/usr/bin/env bash
set -euo pipefail

pushd "$(dirname "$0")/.." >/dev/null

echo "Starting backend API on :8080"
(cd backend && go run ./cmd/api) &
BACKEND_PID=$!

echo "Starting frontend on :5173"
(cd frontend && npm run dev) &
FRONTEND_PID=$!

trap "echo 'Stopping services'; kill $BACKEND_PID $FRONTEND_PID" SIGINT SIGTERM
wait
