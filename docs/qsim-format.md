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
  4-bit ripple-carry adder (7 + 5 = 12) with one 16×16 editable universal
  gate per bit, verified against all 256 input pairs, and M2 remainder
  chains that keep every running system small. Add `-bits 2` to generate
  the 2-bit version (2 + 1 = 3) as `saves/adder2.qsim`. Inputs are normal
  qubits (click to cycle states), not source gates.

## 1. File structure

A `.qsim` file is plain text containing **one JSON object per line**, one
object per window. The engine splits the text on `\n`/`\r` and parses each
line independently (`windows.LoadState`), so a multi-window save is just
multiple lines. Whitespace-only lines are ignored.

Each line must have a `"type"` key: `"Window"`, `"RenderWindow"` or
`"TextWindow"`. Circuit panels are `RenderWindow`s. Before parsing,
`windows.LoadState` migrates older or unnumbered saves to the current
format version — see §3 "Save versioning".

## 2. RenderWindow object

```json
{
  "type": "RenderWindow",
  "saveVersion": "0.8.0",
  "window": { ...base window fields... },
  "cameraZoom": 1,
  "cameraTargetX": 0, "cameraTargetY": 0,
  "cameraOffsetX": 800, "cameraOffsetY": 462.5,
  "cameraRotation": 0,
  "canPan": true, "canSpawn": true,
  "isHorizontalScrolling": false, "isVerticalScrolling": false,
  "isTitleBarVisible": true, "isTitleEditable": false,
  "showGrid": true,
  "components": [ ...component objects... ]
}
```

Base window fields: `name`, `id`, `x`, `y`, `width`, `height`,
`titleBarHeight`, `resizeMinW`, `resizeMinH`, `colorBg`, `colorTitleBar`,
`colorText`, `colorResize`, `colorResizeHover`, `colorBorder` (each color is
`{"r":0,"g":0,"b":0,"a":255}`), `canResize`, `canDrag`, `canZoom`, `priority`.

The camera target is the world point the view is centered on — set it near
the middle of your circuit so the file opens framed. `showGrid` controls the
canvas grid lines and defaults to `true` for older saves; serialize it so
the grid survives a save/load round trip.

## 3. Save versioning

Every `RenderWindow` line carries the format version in `saveVersion`
(currently `globals.SaveVersion`, "0.8.0"). When a file is loaded:

1. `qsim/migration.Migrate` runs first: each line whose `saveVersion` is
   missing (unnumbered) or lower than the current version is rewritten
   through every migration step up to the current version, in order.
   Unnumbered lines are treated as `0.0.0`; malformed lines pass through
   untouched.
2. The migrated text is then parsed normally.

When the serialized format changes, bump `globals.SaveVersion` **and** add
a migration `Step` in `qsim/migration` whose `To` is the new version. Keep
the `Apply` function idempotent: old saves may already contain some of the
fields it sets. The loader also gives sensible defaults to keys absent from
old saves (`showGrid` → true, `probability` → 1, `outcomeProbs`/`hidden` →
zero/absent), so hand-written unnumbered saves still load.

## 3. Common conventions

- **Colors** are `{"r","g","b","a"}` with `uint8` values 0–255.
- **Vectors** are `{"x":0.0,"y":0.0}`.
- **Complex numbers** are `{"real":0.70710677,"imag":0}`.
- **IDs**: every referenceable component has an integer `"id"`; annotation
  components such as `TextBox`/`LineDraw` may omit one. Reference IDs must be
  unique and stable within the file; they are the references used by hooks
  (`targetID`), systems (`hookID`, `infoHookID`) and determinators
  (`hookID`, `qubitSystemID`). On load the engine remaps old→new IDs, so a
  saved reference must match some object's `"id"` in the file.
- **World layout**: positions snap to a 100px grid
  (`config.SnapToGridInterval`). Gate bodies are radius 30; input hooks sit
  `x-300` from the gate center (at `y-50` and `y+50` for 2-input gates);
  the output hook sits `x+300` (a `QubitsSystem` connected to an output hook
  moves its center onto the hook). A source table's output hook sits at
  `x+400`. These distances are sized so the drawn state grid of a ≤4-qubit
  system (2^ceil(n/2)×2^floor(n/2) cells of 100px, centered on the hook)
  clears the producing component. Two hooks connect when within 80px.
- **Authored text**: explanatory text belongs in a `TextBox` component, not
  a bare `Label`. `TextBox` is backgrounded, auto-fits its text when idle,
  and exposes a `[-] [size] [+]` control strip (hold to ramp, or type a
  size) while editing. `Label` remains valid
  for engine/runtime labels, but is not the style-guide choice for authored
  protocol notes.
- **Normal input qubits**: prefer a standalone `QubitsSystem` for a single
  qubit whose state is one of `|0>`, `|1>`, `|+>`, `|->`, `|i>`, or `|-i>`.
  The user can click it to cycle those states without changing its modifier
  ID or wiring. Use `SourceGate` when the source must continuously reassert
  a precise amplitude or when preparing a fine-grained/entangled input.

### Placement: keep every system grid clear of other components

A qubit system's drawn grid has `2^size` cells of 100px, laid out as
`2^ceil(size/2)` columns by `2^floor(size/2)` rows, **centered on the
system**. A system connected to an output hook snaps its center onto that
hook (300px right of the producing gate). The grid therefore occupies:

```
gridX = hookX − GridW(n)/2   gridY = hookY − GridH(n)/2
GridW(n) = 2^ceil(n/2) × 100   GridH(n) = 2^floor(n/2) × 100
```

Place every following component with the shared `cmd/examples/layout`
helpers so nothing sits inside a grid:

```
nextX = layout.AfterOutput(prevGateX, n, margin)
      = prevGateX + 300 + GridW(n)/2 + margin
```

Worked example (the adder generator): a 4-input ADD gate emits a 4-qubit
system, so the first M2 sits `300 + 400/2 + 200 = 700px` later; its 3-qubit
remainder forces the next M2 `300 + 400/2 + 100 = 600px` after that; the
2-qubit remainder needs `300 + 200/2 + 100 = 500px`; the final 1-qubit
carry remainder needs `300 + 200/2 + 350 = 750px` before the next gate (the
350px margin leaves room for the next gate's input hooks 300px left of it).
The pitch therefore shrinks as the remainders shrink.

**Draw order guarantee**: even when components overlap a grid (e.g. a
user-dragged system, or the small logical bit at an M2's C hook, which the
remainder grid's height covers), nothing is hidden: the engine draws every
qubit-system grid as the bottom layer, all other components above them, and
qubit determinators on top. Overlaps are still to be avoided in authored
saves — the formula above is how — but a grid can never erase another
component.

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
- `hidden:true` marks hooks that are not drawn (e.g. the collapse `R` hook
  before a multi-qubit input is measured); the flag is serialized so it
  survives a save/load round trip.
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
- `probability` is the system's **branch weight**: `1` for fresh inputs,
  the product of its unique input systems' probabilities after a gate (a
  system feeding several inputs of one gate counts once), and the input
  probability times the outcome probability after a measurement. It is
  shown on hover; default `1` for saves predating the field.
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
- Editable universal gates (`U`, `cU`) accept **1–8 qubit inputs** (up to a
  256×256 matrix; beyond that the serialized file and the nearest-unitary
  projection become impractical). The editor grows/shrinks the matrix by
  identity padding/truncation.
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
- **Custom operation rule**: a non-standard matrix must be serialized as an
  editable universal gate (`"editable": true`) and have a nearby authored
  `TextBox` explaining its basis order and action. Do not introduce a plain
  non-editable `Gate` with an unexplained custom matrix.
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

### Measurement gates — M/M1, M2, M3 and M4

The app button is labelled `M`; this document calls that branch-producing
gate **M1** so it is not confused with M2.

#### M1 (`Gate` with `isMeasurementGate:true`)

M1 keeps both outcomes alive simultaneously. It emits a `|0>` branch and a
`|1>` branch, each with its probability. Use it when a protocol diagram must
show all possibilities at once. M1 has two output hooks and does not use
`forceMode`.

#### M2 (`CollapseGate`, `normalSystem:false`)

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
- `R` ("remainder") is the normalized conditioned state of every other
  qubit, with the measured modifier removed while preserving the remaining
  modifier order. The hook is always present in the serialized hook list,
  but is hidden/disconnected for a one-qubit input and becomes visible for a
  multi-qubit input.
- `forceMode`: `0` = random (`R` in the UI), `1` = force `|0>`, `2` = force
  `|1>`. Use `1`/`2` when a reproducible save is required. A forced
  zero-probability outcome creates a zero-norm remainder and should be
  avoided. Mode `0` may produce a different result after reload or upstream
  invalidation.
- `outcomeProbs`: the `[p0, p1]` probabilities of the last realization,
  persisted so a saved, already-measured gate (e.g. M4) can keep toggling
  its remainder correctly after load.

#### M3 (`CollapseGate`, `normalSystem:true`)

M3 uses the same random/forced selection and remainder behavior as M2, but
its `C` output is the collapsed measured qubit as a normal one-qubit
`QubitsSystem`, not a `LogicalBit`. Use it when the measured qubit must feed
another quantum gate. Its `R` output is the conditioned remainder.

#### M4 (`M4Gate`)

M4 starts from an M2-style measurement, retains the input determinator (it
does not eat the qubit feeding it), stores the measured input, then flips
the realized result every `swapInterval` frames. Its `C` output is a
logical bit and its `R` output is the conditioned remainder. It is useful
for visualizing changing outcomes when the two outcomes are close in
amplitude. The current engine starts M4 deterministically and alternates the
two forced outcomes; use the `R`/random presentation style for ordinary M2
measurements when a distribution, rather than a deterministic alternation,
is what the protocol needs.

M4 serializes the M2 keys plus its toggle state: `skipRandom`,
`swapInterval`, `frameCount`, `measured`, `result`, `hasRemainder`,
`outcomeProbs`, `inputConsumed`, `storedPos`, `storedSysID`, and
`storedInput` (the input snapshot's `QubitStateManager`). On load,
`storedSysID` is remapped to the loaded input system, so a saved M4 keeps
toggling the correct remainder.

#### Quick comparison — M1, M2, M3 and M4

| Component | Output | Use case |
|---|---|---|
| **M1** (`Gate`, `isMeasurementGate:true`) | both branches at once: a `|0>` system and a `|1>` system, each with its `outcomeProbs` probability | show all possibilities at once — the protocol's alternatives are the point |
| **M2** (`CollapseGate`, `normalSystem:false`) | `C` = `LogicalBit` (classical 0/1); `R` = conditioned remainder (multi-qubit inputs only) | the next step needs one concrete classical result; drives lights/logic gates |
| **M3** (`CollapseGate`, `normalSystem:true`) | `C` = collapsed measured qubit as a normal one-qubit system; `R` = conditioned remainder | the collapsed qubit must continue into another quantum gate |
| **M4** (`M4Gate`) | `C` = `LogicalBit`; `R` = conditioned remainder; the result flips every `swapInterval` frames | close-amplitude outcomes where the changing result must be visualized |

**Which measurement should I use?**

- **M1** shows both outcome branches at once — all possibilities, each with its
  probability. Use it when the protocol's alternatives are the point of the figure.
- **M2** realizes exactly one result. Force modes (`forceMode` `1`/`2`, UI badge
  `0`/`1`) pin the outcome to `|0>`/`|1>` for deterministic cases (reproducible
  saves); mode `R` (`forceMode` `0`) draws a **fresh random sample every tick**,
  so a running circuit's result keeps changing with the outcome distribution —
  use it when the variety of outcomes is the point. Forced modes latch once.
- **M3** selects the outcome exactly like M2, but the result continues as a qubit
  system — a normal one-qubit `QubitsSystem` on `C` instead of a `LogicalBit`.
- **M4** starts from the M2-style measurement and flips the realized result every
  `x` ticks (`swapInterval` frames). It is suited to close-amplitude outcomes
  where the changing result must be visualized, and is normally used in the
  `R`/random presentation form; the engine starts it deterministically and
  alternates the two forced outcomes.

### ControlledUGate — a bit-controlled universal gate

The `cU` gate is a universal gate (see `Gate` above) whose control is a
classical `LogicalBit` instead of a qubit. Its matrix is edited in-app exactly
like the `U` gate (right-click → rename → size/matrix table, `"editable":
true`), but it is applied **only while the control bit is 1**. While the bit
is 0 the merged inputs pass through unchanged, so the output system keeps the
input union's size either way — the control never expands the state.

```json
{
  "type": "ControlledUGate", "id": 71,
  "center": {...}, "radius": 60, "color": {...},
  "isFixed": false, "weight": 100,
  "label": "cU", "inputCount": 2, "editable": true,
  "operation": [ ...2^inputCount x 2^inputCount matrix... ],
  "hooks": [ ...input hooks I0.., in order... ],
  "control": { ...hook, allowLogicalBit:true, label:"C"... },
  "out": { ...hook, isOutput:true, allowQubitSystem:true, label:"O"... }
}
```

- `hooks` holds the qubit inputs `I0..I_{inputCount-1}` in order; `control`
  is the logical control hook (an unhooked control reads as 0, i.e. identity
  passthrough); `out` is the output qubit-system hook.
- Like `Gate`, the inputs merge in hook order (each input system once) and
  are re-ordered to the matrix's basis order by `SwapColumn` — wire the
  inputs so the merged modifier list starts with the inputs in order.
- The one-gate-at-a-time rule counts only the qubit input hooks: the control
  (a logical bit) and the output hook are not gate inputs.


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

### TextBox — an authored annotation

```json
{"type": "TextBox", "center": {"x": -1500, "y": 300}, "width": 420, "height": 36,
 "text": "Step 2 - Entangle Alice's qubit", "fontSize": 20}
```

The saved `width`/`height` are starting points only: the engine refits the
box to its text on every render, and the reader can retune the size live
(see the style guide §3). `id` is optional for annotation components.
Use a `TextBox` for every authored note, header, caption and title; never a
bare `Label`.

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
    - compute every horizontal position with the shared
      `qsim/cmd/examples/layout` helpers (`GridW`, `GridH`, `AfterOutput`)
      so each produced system's grid clears the next component,
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
 4. **Validate geometry and presentation**: check duplicate modifier IDs in
    the initial input systems, calculate every system grid rectangle from its
    `origin.size`, and confirm no gate, measurement, TextBox, light, or
    remainder system lies inside another system grid. Confirm authored text
    is TextBox-backed, custom matrices have `editable:true` and a note, and
    built-in components retain their built-in colors.
 5. **Validate measurements**: choose M1/M2/M3/M4 for the intended teaching
    goal, chain M2/M3 `R` outputs when a remainder continues, and use forced
    modes only when the save must be reproducible. Run the generator's
    exhaustive truth-table check where the protocol has finite classical
    inputs.
