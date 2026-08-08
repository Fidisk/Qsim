# AGENTS.md

Guidelines for AI agents (and humans) working in this repository.

## Making a new circuit

- If a task asks you to create a new circuit or edit a `.qsim` save file,
  first read **`docs/qsim-format.md`** — it documents the complete save-file
  format, hook wiring rules, quantum-state bit ordering, and reference
  matrices. Also read **`docs/qsim-style.md`** — the layout/annotation
  conventions (lanes, step headers, universal-gate captions, classical
  visualization) that keep a save readable on canvas.
- Prefer generating the file programmatically over hand-writing the JSON.
  Worked examples live under `cmd/examples/`:
  - `gen-superdense` (run with `go run ./cmd/examples/gen-superdense`): a
    single wire-per-qubit circuit with 2-qubit gates and live measurements.
  - `gen-qft` (run with `go run ./cmd/examples/gen-qft`): how multi-input
    gates (3-qubit, 8×8 operations) thread a whole system between stages.
  - `gen-adder` (run with `go run ./cmd/examples/gen-adder`): a 4-bit
    ripple-carry adder (7 + 5 = 12) with per-bit 16×16 permutation gates;
    shows the leading-modifier hook-ordering constraint (see
    `docs/qsim-format.md` §5 Gate), an exhaustive 256-pair check, and M2
    remainder chains that keep running systems small.
  - `gen-adder2` (run with `go run ./cmd/examples/gen-adder2`): the simple
    2-bit adder (2 + 1 = 3) — one 6-qubit 64×64 gate for the whole
    addition plus a 3-gate M2 readout chain, written to `saves/adder2.qsim`.
  Hand-written saves almost always break on the global ID references.

## Build & test

- Build: `go build ./...`
- Format: `gofmt -w <files>` (keep files gofmt-clean)
- Test: `go test ./...`
- The app entry point is `main.go`; the desktop/wasm split lives in the
  `*_desktop.go` / `*_wasm.go` files.