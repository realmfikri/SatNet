#!/usr/bin/env bash
set -euo pipefail

pushd "$(dirname "$0")/.." >/dev/null

echo "Setting up backend dependencies..."
(cd backend && go mod tidy)

echo "Setting up frontend dependencies (npm install)..."
(cd frontend && npm install)

echo "Bootstrap complete."
