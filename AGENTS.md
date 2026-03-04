# AGENTS.md

## Cursor Cloud specific instructions

### Overview

Focusd is a Wails v3 desktop app (Go + React/TypeScript). It has two runnable services:

| Service | Command | Port | Notes |
|---------|---------|------|-------|
| Backend API | `go run ./cmd/main.go serve` | 8089 | Requires `.env` with `PASETO_KEYS` and `HMAC_SECRET_KEY` |
| Frontend (Vite) | `npm run dev` (in `frontend/`) | 9245 | Requires Wails bindings generated first |

### Caveats

- **Wails bindings must be generated** before the frontend can build or run: `wails3 generate bindings` from the repo root. Bindings live in `frontend/bindings/` and are gitignored.
- **Frontend `dist/` is required** for Go tests on the root package (`//go:embed all:frontend/dist`). Run `npm run build:dev` in `frontend/` before running `go test ./...` on the root.
- **`.env` file is required** for the backend API server. Minimal dev config:
  ```
  HMAC_SECRET_KEY=dev-mode-secret
  PASETO_KEYS=00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff
  PORT=8089
  ```
- **LLM classifier tests** (`classifier_llm_test.go`) require `GEMINI_API_KEY` and will fail without it. This is expected in CI/dev without API keys.
- **No frontend test files exist** currently; `npm run test` exits with code 1 due to no test files found. This is not a setup issue.
- The full desktop Wails app (`wails3 dev` / `make dev`) requires a display and GTK/WebKit. On headless cloud VMs, use the frontend Vite dev server directly at `localhost:9245` and the backend API server separately.
- `CGO_ENABLED=1` is required for Go compilation (SQLite via `go-sqlite3`).

### Standard commands

See `README.md` for full dev setup. Key commands:
- **Go tests**: `go test ./...` (skip root package if `frontend/dist` not built)
- **Frontend tests**: `cd frontend && npm run test`
- **Frontend build**: `cd frontend && npm run build:dev`
- **Lint**: TypeScript checking via `cd frontend && npx tsc --noEmit`
- **Backend API server**: `go run ./cmd/main.go serve`
- **Frontend dev server**: `cd frontend && npm run dev`
