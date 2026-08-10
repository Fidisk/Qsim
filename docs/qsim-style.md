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
  Never let two components sit inside one grid — the engine draws grids
  beneath everything else, so an overlap is visible but never hides another
  component; keep it out of reach anyway.
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

### Grouped protocol layout recipe

The adder (`cmd/examples/gen-adder`) is the canonical grouped circuit:
inputs on the left, one block per bit (gate + caption + measurement
chain), separators between blocks. The recipe:

1. **Place the inputs group.** One normal `QubitsSystem` per qubit at a
   common leftmost X (the generator's `srcX`), each on its own lane —
   200px pitch, MSB on top — with register captions as `TextBox`es above
   and below the group. Nothing else shares this column.
2. **Per stage, place one group containing the gate, its caption, and
   its measurement chain.** The stage's gates share one X column; the
   caption `TextBox` goes directly under the gate; the chain (`M2` →
   `LogicalBit` → `Light`) follows immediately right, each step spaced
   with `layout.AfterOutput` so no remainder grid covers the next
   component. Keep the group tight: no gap inside it may exceed the
   inter-stage gap.
3. **Then the readout group.** On a classical lane `+300px` below the
   lowest quantum lane, read left → right from the `M2` columns:
   `LogicalBit`s into `LogicGate`/`CopyGate` → `CompareGate` → a `Light`
   at the right edge. The circuit's last column is always a light;
   per-bit lights may also sit on their lanes inside the stage group, as
   the adder does.
4. **Separate groups with whitespace and one separator line.** Leave
   ≥200px of empty X between groups and draw one thin dim vertical
   `LineDraw` (90/90/90) from the top lane to the classical lane at that
   gap. One line per phase boundary, never around a single component.
5. **Let the wires cross the gaps.** A remainder `R` chain or bit link
   may span the whitespace into the next group — the wire is the
   connection, the gap is only a visual pause.

```
 inputs group            stage group (one per step)          readout group
 ──────────────           ──────────────────────────────      ───────────────
  lane 0 ─ I ─────────►   ── G ── M2 ── bit ─────────────►    ─ Copy ─ Compare ─ L
  lane 1 ─ I ─────────►   ── G ── M2 ── bit ─────────────►    ────────────────►
  lane 2 ─ I ─────────►   ── └─ G ── M2 ── bit ──────────►    (classical lane,
  (register captions      caption (same group, under G)       +300px below)
   above/below)                       │
               │            ≥200px gap + separator; the R chain
          ≥200px gap      and bit links may cross the gap
          + separator
```

Spacing, in one place:

- Lane pitch 200px (two grid steps); every center on the 100px snap grid.
- Stage pitch: `nextX = layout.AfterOutput(prevGateX, n, margin)` =
  `prevGateX + 300 + GridW(n)/2 + margin`, with `GridW(n) = 2^ceil(n/2) × 100`
  (see `docs/qsim-format.md` §3). Adder margins: 200px after a 4-qubit
  output (pitch 700), 100px after 3- and 2-qubit remainders (pitch 600
  and 500), and 350px before the next gate (pitch 750) so that gate's
  input hooks, 300px left of it, stay clear.
- Inter-stage whitespace: ≥200px — the §3 floor; the grid formula often
  demands more, which is fine.
- Readout lane: `+300px` below the lowest quantum lane.

The four Gestalt principles, applied concretely to this recipe:

- **Proximity**: the gate, caption and measurement chain sit inside one
  group with no internal gap exceeding the ≥200px inter-stage gap, so
  each stage binds into a single unit before the eye reaches the next.
- **Similarity**: peer stages reuse the same `TextBox` sizes (20pt step
  header, 14pt caption) and the built-in gate/light colors, so parallel
  groups read as parallel phases without re-reading labels.
- **Connectivity**: only real hook links carry the data — the `R` chain
  and the bit→light links may cross the gaps, while the one dim
  separator per phase touches no hook, so it reads as a boundary, not a
  path.
- **Continuity**: a continuing qubit or remainder always flows
  left → right (`R` → next measurement or gate) on its lane and never
  jumps back across an earlier group.

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
   smaller one for captions (see the size table in §3). A `TextBox` always
   auto-fits its text on render, so the `width`/`height` in a save are only
   starting points. While editing, a **`[-] [size] [+]` strip** sits above
   the box: a short click on **+/−** steps the font size by one; holding a
   button keeps stepping with an accelerating ramp (the repeat interval
   shrinks and the step size doubles, up to a 64px cap; the size clamps at
   6 with no upper bound). The middle **size field** shows the live size —
   click it and type a new number (Enter commits, Esc cancels). Every
   change refits the box, so any saved size is just a starting point the
   reader can re-tune.
3. **A separator line**: a vertical `LineDraw` between stages, from the top
   lane to the bottom classical lane, thin and dim (so it reads as a grid
   line, not a wire):

```json
{"type": "LineDraw",
 "start": {"x": 600, "y": -1600}, "end": {"x": 600, "y": -1000},
 "lineColor": {"r": 90, "g": 90, "b": 90, "a": 255}, "placed": true}
```

### Separator line discipline

A separator's only job is to answer "where does this phase end?" — so use
it **at major stage boundaries only**, where the 200px gap plus header
already say a new phase starts but the eye could still merge two stages.
Between short adjacent stages, the gap and header alone are enough; a
separator no one needs is noise.

- **How many**: at most one per stage boundary, and a handful per save
  (about 4–6). If a protocol would need more, it is over-fragmented:
  merge minor steps into phases and separate the phases, not the
  micro-steps. Too many separators is worse than none — the canvas reads
  as a fence instead of a circuit.
- **Color**: dim mid-gray, e.g. `{"r": 90, "g": 90, "b": 90, "a": 255}` —
  darker than the wire palette, lighter than the background grid. Never
  white (that is the `LineDraw` default and reads as a signal), never a
  bright or saturated color.
- **Thickness**: a `LineDraw` has no thickness field — it always renders
  at 2px — so dimness is conveyed by color alone. If a separator looks
  as bold as a hook, its color is wrong.
- **Geometry**: vertical, crossing every lane at the boundary X, from
  above the top lane to below the bottom classical lane. Place it in the
  empty gap, clear of gates, hooks and boxes.

**A separator must never be mistaken for a wire.** Wires are the curved,
hooked, blue/pink lines that carry signals; a separator carries nothing.
So a separator must not: run horizontally along a lane (horizontal = a
wire); use the wire palette (hook blue, output-hook pink, or anything
bright enough to catch the eye as a signal); touch or cross a hook, gate
or system (crossing a gate's position reads as a data path); or end on a
component. Bare ends, straight line, vertical, dim gray — if a reader
could imagine a qubit or bit traveling on it, it is wrong.

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

### Text size by role

Pick the font size from the role, not the available space; a reader
should see the hierarchy at a glance without reading a word:

| Role              | `fontSize` | Where it lives                                            |
|-------------------|------------|-----------------------------------------------------------|
| Title             | 24–28      | one per save, above the first lane, centered over the circuit |
| Step header       | 18–20      | above each stage, centered over its lanes (item 2)        |
| Caption           | 14–16      | directly under the gate or step it explains, inside its group |
| Footnote / note   | 12–14      | prose, caveats, cross-references ("same U as step 2")     |

- **Captions stay short and close**: 1–2 lines, placed **inside the
  group they explain** — under the gate, within the step's X span, never
  straddling a gap or a separator (proximity beats arrows). A `TextBox`
  has no word-wrap; a long string just widens the box, so break lines
  yourself (`\n`). Text that would run 3+ lines is a note, not a caption:
  shorten it and move the detail to a footnote box.
- **Hierarchy**: the title is the largest text on the canvas; everything
  else steps down (headers, then captions, then notes). Resist raising a
  header to title size or a caption to header size.
- **Mechanics** (see item 2): `TextBox` always auto-fits its box to its
  text, so the sizes above are starting points. While editing, the
  `[-] [size] [+]` strip above the box shows the live size in the middle
  field (click to type a number, Enter commits) and the **+/−** buttons
  step the size by one — hold either to ramp, and the box refits on every
  change.

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
  step needs a concrete classical result. The UI badge shows `R`, `0`, or
  `1` for the force mode: `R` samples according to probability, while the
  `0`/`1` badges (ForceMode `1`/`2`) force the corresponding outcome for a
  deterministic case. Use forced modes for reproducible saves and random
  mode for distributions.
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

**Which measurement should I use?**

- **M1** — shows both outcome branches at once (all possibilities). Pick it when
  the alternatives are the point of the figure.
- **M2** — realizes one result. Force modes (badges `0`/`1`) pin a deterministic
  `|0>`/`|1>` for reproducible saves; `R` mode samples randomly per measurement
  for varied distributions.
- **M3** — the same selection as M2, but the outcome continues as a qubit system
  (feeds quantum gates).
- **M4** — flips the M2-style result every `x` ticks. Suited to close-amplitude
  outcomes where the changing result must be visualized; normally used in the
  `R`/random presentation form.

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
- **`ControlledUGate` (`cU`)** — the classical-control tool: a universal
  matrix (edited like the `U` gate) that applies only while a hooked
  `LogicalBit` is 1. Use it for classically conditioned corrections — "apply
  this operator only if that measured bit was set" — the quantum-lane mirror
  of a `LogicGate`.

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
- [ ] At most one dim, vertical `LineDraw` per stage boundary, a handful
      per save, never colored or shaped like a wire (see §3).
- [ ] Text sizes follow the §3 role table; captions are 1–2 lines and sit
      inside the group they explain.
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
