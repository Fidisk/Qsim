# Qsim Save File Format (.qsim)

This document describes how to author a `.qsim` circuit file so the Qsim
engine loads it correctly. Read it whenever you are asked to create a new
circuit. Fully worked examples live under `cmd/examples/` — build the circuit
with the engine's own constructors and serialize it, instead of hand-writing
JSON:

- `cmd/examples/gen-superdense/main.go`
  (`go run ./cmd/examples/gen-superdense`): a linear single-wire-per-qubit
  circuit with 2-qubit gates, collapse measurement and lights.
- `cmd/examples/gen-qft/main.go` (`go run ./cmd/examples/gen-qft`): a 3-qubit
  QFT showing how multi-input gates (3-qubit, 8×8 operations) thread a whole
  entangled system between stages.
- `cmd/examples/gen-adder/main.go` (`go run ./cmd/examples/gen-adder`): a
  4-bit ripple-carry adder (7 + 5 = 12) with one 16×16 permutation gate per
  bit, verified against all 256 input pairs, and M2 remainder chains that
  keep every running system small.
- `cmd/examples/gen-adder2/main.go` (`go run ./cmd/examples/gen-adder2`):
  the simple 2-bit adder (2 + 1 = 3): a single 6-qubit 64×64 gate for the
  whole addition plus a 3-gate M2 readout chain.

## 1. File structure

A `.qsim` file is plain text containing **one JSON object per line**, one
object per window. The engine splits the text on `\n`/`\r` and parses each
line independently (`windows.LoadState`), so a multi-window save is just
multiple lines. Whitespace-only lines are ignored.

Each line must have a `"type"` key: `"Window"`, `"RenderWindow"` or
`"TextWindow"`. Circuit panels are `RenderWindow`s.

## 2. RenderWindow object

```json
{
  "type": "RenderWindow",
  "window": { ...base window fields... },
  "cameraZoom": 1,
  "cameraTargetX": 0, "cameraTargetY": 0,
  "cameraOffsetX": 800, "cameraOffsetY": 462.5,
  "cameraRotation": 0,
  "canPan": true, "canSpawn": true,
  "isHorizontalScrolling": false, "isVerticalScrolling": false,
  "isTitleBarVisible": true, "isTitleEditable": false,
  "components": [ ...component objects... ]
}
```

Base window fields: `name`, `id`, `x`, `y`, `width`, `height`,
`titleBarHeight`, `resizeMinW`, `resizeMinH`, `colorBg`, `colorTitleBar`,
`colorText`, `colorResize`, `colorResizeHover`, `colorBorder` (each color is
`{"r":0,"g":0,"b":0,"a":255}`), `canResize`, `canDrag`, `canZoom`, `priority`.

The camera target is the world point the view is centered on — set it near
the middle of your circuit so the file opens framed.

## 3. Common conventions

- **Colors** are `{"r","g","b","a"}` with `uint8` values 0–255.
- **Vectors** are `{"x":0.0,"y":0.0}`.
- **Complex numbers** are `{"real":0.70710677,"imag":0}`.
- **IDs**: every component has an integer `"id"`. IDs must be unique and
  stable within the file; they are the references used by hooks
  (`targetID`), systems (`hookID`, `infoHookID`) and determinators
  (`hookID`, `qubitSystemID`). On load the engine remaps old→new IDs, so a
  saved reference must match some object's `"id"` in the file.
- **World layout**: positions snap to a 100px grid
  (`config.SnapToGridInterval`). Gate bodies are radius 30; input hooks sit
  `x-150` from the gate center (at `y-50` and `y+50` for 2-input gates);
  the output hook sits `x+150` (a `QubitsSystem` connected to an output hook
  moves its center onto the hook). Two hooks connect when within 80px.

## 4. Hook objects (the wiring mechanism)

Every connection in the engine is a `Hook` pointing at a target:

```json
{
  "type": "Hook", "id": 19,
  "center": {"x": -1250, "y": -150},
  "radius": 30, "color": {...},
  "isFixed": false, "weight": 100,
  "isHooked": true, "targetID": 15,
  "isOutput": false,
  "label": "I0",
  "allowQubitSystem": false, "allowLogicalBit": false
}
```

- `isHooked`/`targetID` are the actual link: `isHooked:true` and a nonzero
  `targetID` pointing at the target component's `id`.
- `isOutput:true` marks hooks that emit a new system/bit (gate `O`, source
  `O`, collapse `C`/`R`); `isOutput:false` hooks consume.
- `allowQubitSystem` hooks accept a `QubitsSystem` (or a determinator of
  one); `allowLogicalBit` hooks accept a `LogicalBit` (double-ring look).
- Labels are purely cosmetic: gate inputs are `I0`, `I1`, …; gate output
  `O`; collapse `I`, `C`, `R`; logic gates `A`, `B`, `O`; light `I`.
- When the engine loads a file it **rebuilds links from the hook side**
  (`remapReferences`): a hook whose `targetID` is a system/determinator/bit
  sets that object's back-reference. `LogicalBit.hookID` from the file is
  intentionally ignored — bits are re-linked from the hooks that point at
  them, and a bit may fan out to several hooks.

## 5. Component types

### SourceGate — emits a fixed single-qubit state

```json
{
  "type": "SourceGate", "id": 10,
  "center": {"x": -1500, "y": -150}, "radius": 90,
  "color": {...}, "isFixed": false, "weight": 100,
  "label": "|0> A",
  "amplitude": [{"real": 1, "imag": 0}, {"real": 0, "imag": 0}],
  "modifierID": 0,
  "outHook": { ...hook, isOutput:true, allowQubitSystem:true, label:"O"... }
}
```

`amplitude[0]` is `|0>`, `amplitude[1]` is `|1>`. `modifierID` is the global
qubit identity (must be unique per qubit wire and must match the `modifierID`
used inside the connected system's `origin.modifierIDs`). The `outHook`
targets a `QubitsSystem` whose `infoHookID` points back at the source hook.
While the app runs, the source re-copies its amplitude into the hooked
system every frame.

### QubitsSystem — a multi-qubit state grid

```json
{
  "type": "QubitsSystem", "id": 14,
  "center": {"x": -1300, "y": -150}, "radius": 30,
  "color": {...}, "isFixed": false, "weight": 0,
  "hookID": 11, "infoHookID": 11, "isLogical": false,
  "origin": {
    "amplitudes": [{"real": 1, "imag": 0}, {"real": 0, "imag": 0}],
    "modifierIDs": [0],
    "size": 1
  },
  "qubitDeterminators": [
    {"center": {...}, "radius": 15, "color": {...},
     "modifierID": 0, "id": 15, "hookID": 19, "qubitSystemID": 14}
  ]
}
```

- `origin` is the **source of truth**: amplitudes in the order
  `|mod0 mod1 ...>` with `modifierIDs[0]` the **most significant bit** (a
  2-qubit state index `i = q0<<1 | q1`). Size must equal the number of
  modifiers and `len(amplitudes) == 1<<size`.
- `hookID` links to the output hook of the gate/source that produced it
  (0 when the system is a free source). `infoHookID` links a read-only info
  hook (e.g. a `SourceGate` output or a compare input).
- `qubitDeterminators` are the per-qubit handles the user plugs into gate
  inputs. Each has `modifierID` (matching one entry of
  `origin.modifierIDs`), its own `id`, `hookID` (the gate input hook it is
  plugged into, 0 if free) and `qubitSystemID`.
- On load the system is rebuilt with `QubitsSystem.Assign(origin)`, which
  regenerates determinators and the animated qubit visuals (random orbits —
  cosmetic); the serialized `qubitDeterminators` entries are then used to
  restore exact positions, `hookID`s and `modifierID`s.

### Gate — a unitary operation

```json
{
  "type": "Gate", "id": 21,
  "center": {...}, "radius": 30, "color": {...},
  "isFixed": false, "weight": 100,
  "label": "H", "inputCount": 1, "outputCount": 1,
  "isMeasurementGate": false, "measureResult": 0, "editable": false,
  "operation": [
    [{"real": 0.70710677, "imag": 0}, {"real": 0.70710677, "imag": 0}],
    [{"real": 0.70710677, "imag": 0}, {"real": -0.70710677, "imag": 0}]
  ],
  "hooks": [ ...input hooks I0.., then the output hook O... ]
}
```

- `operation` is the `2^inputCount × 2^inputCount` matrix (row-major,
  complex). Input order in the hook list matters: input `I0` becomes the
  first (high) column when a multi-input gate runs, then qubits are
  re-ordered to the matrix's basis order by `SwapColumn`.
- **Critical wiring constraint**: `Gate.CalculateOutPut` merges the input
  parent systems in hook order (each system once), then calls
  `SwapColumn(i, FindID(I_i))` for each input. That swap sequence only
  leaves the inputs where the matrix expects them when the inputs are
  already the **leading modifiers** of the merged system (the swap is a
  no-op for them); inputs that sit mid-system get scrambled. So wire gates
  so that the hook order makes the merged modifier list start with the
  inputs in order — e.g. by feeding fresh single-qubit systems first and
  ordering hooks `(fresh..., existing...)` so the merged list grows as
  `[I0, I1, ..., In-1, rest...]`. This is why the adder generator uses one
  4-input gate per bit with hooks `(c_{i+1}, a_i, b_i, c_i)`.
- Reference matrices: H `[[1,1],[1,-1]]/√2`, X `[[0,1],[1,0]]`, Z
  `[[1,0],[0,-1]]`, CNOT (control qubit 0 → target qubit 1)
  `[[1,0,0,0],[0,1,0,0],[0,0,0,1],[0,0,1,0]]`, CZ
  `[[1,0,0,0],[0,1,0,0],[0,0,1,0],[0,0,0,-1]]`.
- A gate with `inputCount` hooked inputs computes its output. To act on one
  qubit of an already-merged 2-qubit system, use a 2-input gate whose
  matrix is the Kronecker product (e.g. `X⊗I`) and feed both determinators
  of the system — see the superdense example.
- `isMeasurementGate:true` is the "M" gate: it needs no `operation` (empty
  list) and has **two** output hooks (|0> and |1> branches). Its input hook
  is `I`.

### CollapseGate — the M2/M3 measurement

```json
{
  "type": "CollapseGate", "id": 53,
  "center": {...}, "radius": 30, "color": {...},
  "isFixed": false, "weight": 100,
  "label": "M2", "inputCount": 1,
  "forceMode": 2, "normalSystem": false,
  "hooks": [ ...hook I..., ...hook C..., ...hook R... ]
}
```

- One input `I` (takes a qubit determinator). Hooks: `C` and `R` (labels),
  in that order in the `hooks` array.
- `normalSystem:false` = **M2**: the `C` output is a `LogicalBit` (classical
  0/1) and `C` has `allowLogicalBit:true`, `isOutput:true`. The bit drives
  lights/logic gates.
- `normalSystem:true` = **M3**: the `C` output is a collapsed normal
  single-qubit `QubitsSystem` (`allowQubitSystem:true`).
- `R` ("remainder") is the conditioned state of the other qubits; only
  exists for multi-qubit inputs (hidden otherwise, `isHooked:false`).
- `forceMode`: 0 = random outcome, 1 = force |0>, 2 = force |1>. Using
  force modes makes a save's measured bits deterministic.

### LogicalBit — a classical 0/1 value

```json
{
  "type": "LogicalBit", "id": 62,
  "center": {...}, "radius": 15, "color": {...},
  "value": 1, "hookID": 63
}
```

`hookID` is informational on save; on load links are rebuilt from hooks that
`targetID` this bit (one hook for a plain chain, several for fan-out).

### Light — displays a bit

```json
{
  "type": "Light", "id": 65,
  "center": {...}, "radius": 40, "color": {...},
  "isFixed": false, "weight": 100,
  "inHook": { ...hook, allowLogicalBit:true, isOutput:false, label:"I"... }
}
```

Shines yellow while the hooked bit's `value` is 1.

### Other components

`M4Gate` (frame-cycling measurement), `CopyGate`, `CompareGate`,
`ControlledGate`, `LogicButton`, `LogicGate` (NOT/AND/OR over bits), `InfoTable`,
`Button`, `ToggleButton`, `Input`, `Label`, `TextBox`, `LineDraw`, `Circle`
are all serialized with analogous fields (see `windows/serialize.go` for the
exact key list of each).

## 6. Measurement bit ordering

`Gate.MeasureOutput` and `CollapseGate` compute the probability that a
measured qubit reads 0 as the sum of `|amp|²` over all indices whose bit at
position `(size-1-pos)` is 0, where `pos` is the index of the qubit's
`modifierID` in `origin.modifierIDs`. So `modifierIDs[0]` is the **most
significant bit** — index 0 of the amplitudes list is `|00...0>`, and the
top row of the gate's grid is the MSB.

## 7. Building a new circuit — recommended workflow

1. **Prefer programmatic generation.** Write a small Go program like
   `cmd/examples/gen-superdense/main.go`:
   - create a `windows.NewRenderWindow`, `rw.PushComponent(...)` every
     component,
   - construct sources/gates/systems with the `components.New*`
     constructors (they lay out hooks and register IDs),
   - wire with `hook.Connect(det)`, `hook.ConnectInfo(qs)`, output
     `hook.Connect(qs)`,
   - assign exact states with `qubits.QubitStateManager`
     (`NewQubitStateManagerFrom`, `Merge`, `Multiply`, `SwapColumn`) so the
     serialized `origin` amplitudes are correct,
   - write `rw.SaveState()` to a file under `saves/`.
   This guarantees every ID reference is consistent, which is the most
   error-prone part of hand-editing.
2. **When hand-editing JSON**, copy a working save (e.g. `saves/Test.qsim`
   or the generated `saves/superdense.qsim`) and modify, keeping: unique
   ids; `hookID`/`targetID`/`infoHookID` pairs consistent; `modifierIDs`
   consistent between `origin`, determinators, and sources; `size` and
   amplitude counts exact powers of two.
3. **Validate**: load the file through `windows.LoadState` (see the
   round-trip check at the end of the example generator) and confirm every
   `isHooked` hook resolves to a live object.
