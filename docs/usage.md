# Usage

## Backend
1. Ensure Go 1.21+ is installed.
2. From `backend/`, install dependencies and run tests:
   ```bash
   go test ./...
   ```
3. Start the API server:
   ```bash
   go run ./cmd/api
   ```
4. Verify the health endpoint:
   ```bash
   curl http://localhost:8080/health
   ```

## Frontend
1. Ensure Node.js 20+ is installed.
2. From `frontend/`, install dependencies:
   ```bash
   npm install
   ```
3. Start the dev server (defaults to port 5173):
   ```bash
   npm run dev
   ```
4. Run linting and tests:
   ```bash
   npm run lint
   npm test
   ```

## Connecting the stack
- The frontend should eventually call the backend at `http://localhost:8080/simulation/snapshot` to retrieve live constellation data.
- Use the scripts in `scripts/` to coordinate multi-service dev flows.
