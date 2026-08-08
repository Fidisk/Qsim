# Qsim Save Style Guide (.qsim)

A `.qsim` file is a circuit: a stack of JSON objects, one window per line.
The format reference is `docs/qsim-format.md`; this document is the
**style** reference — how to lay out, label and annotate a circuit so a
human can read the protocol off the canvas without decoding JSON.

The running example throughout is `saves/Telepor2.qsim` (quantum
teleportation). It is *functionally* correct but stylistically bare: no
text, no step separation, lanes that meander and overlap. This guide is the
recipe for turning a save like that into a self-documenting diagram.

---

## 1. Golden rules

1. **Reads like a book**: left → right, top → bottom.
2. **One lane per qubit wire**, never crossed.
3. **Steps are separated** by gaps, header boxes and separator lines.
4. **Every universal gate is annotated** — at least once per distinct type.
5. **The classical side tells the story**: lights, logic gates, copy and
   compare gates turn "trust me, it works" into something you can see.
6. **All text lives in a `TextBox`** — never draw text directly on the
   canvas. `Label` draws raw glyphs with no background, which vanish over
   wires and the grid; a `TextBox` is a bordered, backgrounded box that
   stays readable anywhere. Every title, step header, gate caption and
   prose note in a save must be a `TextBox` (see §3 and §4).
7. **Prefer programmatic generation** (see `docs/qsim-format.md` §7 and the
   `cmd/examples/` generators): the style rules live in the generator's
   layout constants and annotation calls, and every hook/ID stays
   consistent automatically.

The TextBox rule applies to authored save annotations. Built-in labels on
gates/hooks, runtime probability readouts, and the app's own controls are
engine-rendered exceptions; do not duplicate them with a second raw text
object.

---

## 2. Quantum layout conventions

- **Position**: everything sits on the 100px grid
  (`config.SnapToGridInterval`). Gate bodies are radius 30; input hooks sit
  at `x−300` (offset `±50` per input), the output hook at `x+300`, a source
  table's output hook at `x+400`. A `QubitsSystem` connected to an output
  hook snaps its center onto the hook.
- **Respect the state grid**: a qubit system's drawn grid is
  `2^ceil(n/2) × 2^floor(n/2)` cells of 100px, **centered on the system**
  (so a 4-qubit system covers 400×400). The hook distances above are sized
  so a ≤4-qubit grid clears the component that produced it — but the grid
  also extends right, so place every following component with the shared
  `cmd/examples/layout` helpers: `GridW(n)`/`GridH(n)` give the footprint
  and `AfterOutput(gateX, n, margin)` returns the next X that clears it.
  The adder generator derives its whole column pitch from these helpers
  (the chain steps shrink as the remainders shrink: 4 → 3 → 2 → 1 qubits).
  Never let two components sit inside one grid.
- **Lanes**: pick one Y per qubit wire and never change it mid-circuit.
  Choose a pitch of **200px** (two grid steps) so 2-input gate hooks
  (`±50`) never collide with a neighbor lane. Keep each qubit's X purely
  increasing — gates on a lane form a column list, not a zigzag.
- **MSB on top**: `origin.modifierIDs[0]` is the most significant bit and
  the top row of every gate matrix (see format doc §6). The topmost lane is
  therefore `modifierIDs[0]`, the next `modifierIDs[1]`, etc.
- **Inputs are normal qubits, not sources**: a single-qubit input can be a
  normal `QubitsSystem` (click to cycle `|0> |1> |+> |-> |i> |-i>`), which
  stays interactive. Only use a `SourceGate` when fine-grained or
  entangled initialization is required (Bell pairs, QFT inputs, ...).
- **Built-in components keep their built-in colors**: never tint a `Light`
  (or other stock component) with a custom color — the save must look like
  the app's own palette (`config.GateColor` etc.).
- **Sources left, sinks right**: inputs at the left edge of their lane;
  measurement / classical output at the right edge.
- **Stage columns**: place gates in the same *stage* at the same X across
  lanes, so the circuit reads as a set of vertical slices (see §3).
- **Camera**: set `cameraTargetX/Y` to the middle of the circuit and a
  comfortable `cameraZoom` (0.2–0.8 for wide circuits) so the file opens
  framed.

### Gestalt grouping

When a protocol is not a straight textbook circuit, group it deliberately
instead of letting every component follow one global line:

- **Proximity**: keep the gate, its short explanation, and its immediate
  measurement/readout close together; leave a larger gap before the next
  protocol phase.
- **Similarity**: use the same TextBox font size and wording pattern for
  peer steps, the same gate color for a gate family, and the built-in Light
  color rather than inventing a second palette.
- **Connectivity**: let actual wires and hook links carry the connection;
  do not draw decorative lines that look like data paths. If a long wire is
  unavoidable, add one unobtrusive separator rather than several arrows.
- **Continuity**: route a continuing qubit or remainder in a consistent
  direction. A remainder chain should read as `R -> next measurement`, not
  jump back across an earlier stage.

Use a small number of `LineDraw` separators to define phase boundaries, not
to outline every component. A line should answer "where does this phase end?"
without competing with the wires that answer "what is connected?".

### Example layout (abstract)

```
 lane 0 ─ S ──────── G ── G ──────── M ────────►  (MSB)
 lane 1 ─ S ──────── G ───────────── M ────────►
 lane 2 ─ S ──────── └──── G ─────── M ────────►
                       └─────────────┘           (2-qubit gate spans lanes)
 classical ──────────── (bits) ─── L ─────────►  (see §6)
```

---

## 3. Steps: separate and explain

Telepor2.qsim's biggest readability gap is that the protocol stages run
into each other with nothing to say what is happening. Use three tools,
always together:

1. **A gap**: leave **200px of empty X** between stages — the eye groups
   gates by whitespace.
2. **A step header**: a `TextBox` above the stage, at the same X as the
   stage's gate column, centered over the lanes it affects:

```json
{"type": "TextBox", "center": {"x": 900, "y": -1600}, "width": 420, "height": 36,
 "text": "Step 2 - Entangle Alice's qubit", "fontSize": 20}
```

   Number the steps (`Step 1`, `Step 2`, ...) so the save can be read as an
   algorithm. Keep one consistent text color/size for step headers and a
   smaller one for captions. A `TextBox` always auto-fits its text on
   render; while editing it shows a **+/− pair above the box** to resize the
   font (which refits the box). A short click steps the size by one; holding
   the button keeps stepping and the repeat interval shrinks, so holding
   ramps the scaling up. The sizes in a save are just a starting point the
   reader can adjust.
3. **A separator line**: a vertical `LineDraw` between stages, from the top
   lane to the bottom classical lane, thin and dim (so it reads as a grid
   line, not a wire):

```json
{"type": "LineDraw",
 "start": {"x": 600, "y": -1600}, "end": {"x": 600, "y": -1000},
 "lineColor": {"r": 90, "g": 90, "b": 90, "a": 255}, "placed": true}
```

### Step header wording

Headers say **what the step does**, not how it is implemented:

| Telepor2 stage (current, unlabeled)      | Suggested header                            |
|------------------------------------------|---------------------------------------------|
| `H` + `CX` on Bell qubits                | `Step 1 - Prepare the Bell pair |00>+|11>`  |
| `CX` with Alice's qubit                  | `Step 2 - Entangle Alice's qubit`           |
| `H` + two `M2` collapses                 | `Step 3 - Alice measures and publishes`     |
| `ControlledGate` corrections             | `Step 4 - Bob corrects his qubit`           |
| `CopyGate` → `CompareGate` → `Light`     | `Step 5 - Verify the teleport`              |

For prose (what to watch, why a step works), use a `TextBox` just under the
header instead of a one-line box:

```json
{"type": "TextBox", "center": {"x": 900, "y": -1500}, "width": 480, "height": 44,
 "text": "CX entangles the message qubit with the Bell pair: the pair's state is now conditioned on Alice's qubit.",
 "fontSize": 14}
```

---

## 4. Universal gates: annotate, at least once per type

A `U` gate (`editable:true`) is an opaque matrix — its label says `U`, its
operation says nothing to the reader. Convention:

1. **Every custom operation is an editable universal gate.** A save never
   uses a plain `Gate` with a custom matrix (e.g. the adder's ADD gates):
   set `"editable": true` so the gate is a universal gate the user can
   inspect and edit.
2. **Rename it** (right-click → rename in the app, or set `"label"` in the
   JSON) to something that names the operation, e.g. `U: Rx(pi/4)`,
   `U: phase 90°`, `ADD bit 0`.
3. **Annotate every distinct type at least once** in the save: one caption
   (`TextBox`) per unique operation matrix, placed under the gate, saying
   what the matrix does in circuit terms. When the same U type is reused
   later, no new caption is needed — the earlier one is referenced
   ("same U as step 2").
4. Keep the caption within the step, e.g.:

```json
{"type": "TextBox", "center": {"x": 1300, "y": -1550}, "width": 460, "height": 34,
 "text": "U: rotate the message qubit by 90 deg around X (Rx(pi/2))",
 "fontSize": 14}
```

   A caption is "the matrix, in words": name the rotation axis/angle, or
   the equivalent standard gate (`H`, `X`, `CZ`, ...) if it is one of
   those in disguise — that is far more useful than the raw matrix.

---

## 5. Measurement choice

Choose the measurement component based on what the diagram is trying to
teach:

- **M1 (`M`)**: show both outcome branches and their probabilities at once.
  Use it when the protocol's alternatives are the point of the figure.
- **M2**: realize one outcome and emit a `LogicalBit`. Use it when the next
  step needs a concrete classical result. Mode `0` (`R`) samples according
  to probability; modes `0`/`1` in the UI force the corresponding outcome
  for a deterministic case. Use forced modes for reproducible saves and
  random mode for distributions.
- **M3**: realize one outcome but emit the measured qubit as a normal
  one-qubit system. Use it when the collapsed qubit continues into a
  quantum gate; its `R` output is the remaining conditioned system.
- **M4**: alternate the realized M2-style outcome every configured number
  of frames. Use it as a visual time-series tool when both outcomes are
  close enough that switching is informative. It normally belongs in the
  random/`R` presentation context, not as a static forced branch.

For M2/M3/M4, chain the `R` output into the next operation when you need to
keep a system's drawn grid small or enforce the one-gate-at-a-time rule. A
filled gate claims its system: unused determinators hide, and determinators
already connected to another gate are disconnected.

## 6. Classical visualization: let the result be seen

The classical side is the protocol's narration. Telepor2.qsim already uses
the right vocabulary (copy, compare, lights) but drops the components
scattered around the canvas; the convention is to arrange them in a
**classical lane below the quantum lanes**, fed right-to-left from the
measurements:

- **`LogicalBit`** — a measured 0/1. It is the *token* everything classical
  consumes; one per measurement output.
- **`Light`** — the verdict. A light on a measurement bit says "this bit
  is 1"; a light on a compare result says "teleport succeeded". One light
  per interesting bit, placed at the end of the classical lane.
- **`LogicGate`** (NOT/AND/OR) — combine bits: e.g. `AND(m1, m2)` as a
  single "both measures matched" success signal instead of two lights.
- **`CopyGate`** — fan out a *quantum* state (or a bit) to several
  consumers without consuming it; used to keep a pristine copy of the
  original qubit for comparison while the original goes through the
  protocol.
- **`CompareGate`** — the proof: `==` of the original state against the
  received state produces a `LogicalBit`; a `Light` on it is the circuit
  asserting "teleportation worked". Keep the two compared wires
  horizontal and converging on the gate.

Telepor2's verification chain restyled: `CopyGate` (pristine copy of the
message qubit) → `CompareGate` (copy vs. Bob's received qubit) →
`LogicalBit` → `Light`, all on one classical lane, with the `Light` labeled
`Teleport OK?`.

Placement rules: classical components go **below** the lowest quantum lane
(start `+300px` below it), read left → right from the `M2` columns, and
their outputs terminate in lights at the right edge — the circuit's last
column should always end in measurement or a light, never in a dangling
bit.

---

## 7. Applying the convention: Telepor2.qsim, restyled

Telepor2.qsim implements the teleportation protocol with 31 components:
3 sources (message `|ψ>` and a Bell pair), `H`/`CX`/`M2` on the quantum
side, `ControlledGate` corrections, and the copy/compare/light verification
chain. The restyle:

| Aspect              | Current `Telepor2.qsim`                | Styled version                                       |
|---------------------|----------------------------------------|------------------------------------------------------|
| Lanes               | qubits wander (`-1400`, `-1200`, `-1000`, systems snap to hooks mid-air) | 3 lanes at `y=-1400/-1200/-1000`, pitch 200, never change Y |
| MSB on top          | `modifierID` 2 is top lane              | keep `modifierID 2` on top; label sources `|psi> A`, `|0> B`, `|0> C` |
| Steps               | none — one long row                    | 5 stages, 200px gaps, `TextBox` headers, `LineDraw` separators |
| Explanatory text    | none                                   | one `TextBox` per step describing the idea (see table in §3) |
| `U` gates           | none present                           | (rule applies whenever a `U` appears: rename + one caption per type, §4) |
| Classical side      | copy/compare/light scattered (`CopyGate` at `(-400,-2200)`, lights at three different spots) | one classical lane at `y=-700` below the qubits: `M2` bits → `LogicGate` → `CopyGate` → `CompareGate` → `Light` labeled `Teleport OK?` |
| Camera              | target `(2623,-1798)`, zoom `0.6` — frames empty space | target = center of the framed circuit, zoom 0.7 |

The resulting save reads as:

```
 |psi>  ─ S ────────────────────────CX──H──M2 ──► bit m1 ──────────────┐
 |0> B  ─ S ──────H──CX ────────────┘   └────M2 ──► bit m2 ── Corrections
 |0> C  ─ S ─────────┘ └───────────────...───────────────────────────► |psi'>
           Step 1        Step 2          Step 3   Step 4      Step 5
          Bell pair     entangle       measure  correct    verify:
          prepare        Alice                     └── CopyGate → CompareGate → (Light)
```

---

## 8. Checklist

Before calling a save "styled":

- [ ] Every qubit has its own horizontal lane; MSB on top; no crossing wires.
- [ ] Sources at the left, measurement/classical output at the right.
- [ ] All centers on the 100px grid (gates/hooks/systems).
- [ ] Every system grid fits its group: calculate the `2^ceil(n/2)` by
      `2^floor(n/2)` footprint before placing the next component; no gate,
      TextBox, light, or remainder system sits inside that rectangle.
- [ ] Stages separated by ≥200px gaps **and** a step `TextBox` header **and**
      a `LineDraw` separator.
- [ ] Every step has a one-line header; complex steps have a `TextBox`
      explaining the idea.
- [ ] Every distinct `U`-gate type is renamed and annotated at least once.
- [ ] Every custom matrix gate has `editable:true` and a nearby TextBox note.
- [ ] No text is drawn outside a `TextBox` (no bare labels on the canvas).
- [ ] Use normal qubits for ordinary `0/1/+/-/i/-i` inputs; reserve sources
      for precise or entangled initialization.
- [ ] Use M1/M2/M3/M4 according to the measurement purpose in §5; chain `R`
      when the remainder must continue or a grid must stay small.
- [ ] Measured bits become `LogicalBit`s shown on `Light`s; protocols that
      need proof use `CopyGate` → `CompareGate` → `Light`.
- [ ] No dangling classical bits; the last column of the circuit is
      measurement or a light.
- [ ] The save opens framed: `cameraTarget` over the circuit center,
      sensible `cameraZoom`.
- [ ] Regenerated (not hand-edited) whenever practical — see
      `docs/qsim-format.md` §7 for the generator workflow.
