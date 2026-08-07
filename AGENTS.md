# AGENTS.md

Guidelines for AI agents (and humans) working in this repository.

## Making a new circuit

- If a task asks you to create a new circuit or edit a `.qsim` save file,
  first read **`docs/qsim-format.md`** — it documents the complete save-file
  format, hook wiring rules, quantum-state bit ordering, and reference
  matrices.
- Prefer generating the file programmatically over hand-writing the JSON.
  Worked examples live under `cmd/examples/`:
  - `gen-superdense` (run with `go run ./cmd/examples/gen-superdense`): a
    single wire-per-qubit circuit with 2-qubit gates and live measurements.
  - `gen-qft` (run with `go run ./cmd/examples/gen-qft`): how multi-input
    gates (3-qubit, 8×8 operations) thread a whole system between stages.
  Hand-written saves almost always break on the global ID references.

## Build & test

- Build: `go build ./...`
- Format: `gofmt -w <files>` (keep files gofmt-clean)
- Test: `go test ./...`
- The app entry point is `main.go`; the desktop/wasm split lives in the
  `*_desktop.go` / `*_wasm.go` files.