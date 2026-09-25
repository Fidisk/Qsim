"""Parse a .qsim save and replay its unitary pipeline step by step (numpy).

Independent of the Go verify package: raw file IDs are resolved directly
(the save is self-consistent) and the state math is reimplemented with
numpy, mirroring the engine (Merge = Kronecker first-operand-high,
SwapColumn brings gate qubits front in hook order, Multiply = U on the
leading qubits). The checker diffs every replay step against Qiskit.

Supported subset: SourceGate and standalone 1-qubit inputs, unitary Gate
matrices of any size, CopyGate, classically-controlled gates whose control
is unhooked (passthrough), constant (button-driven: applied or skipped) or
measurement-driven, and terminal/strippable measurements. Anything else on
the quantum path is an error.

Classical controls via deferred measurement: an M2 collapse whose C-bit
drives controls compiles to quantum control(s) on the measured qubit, and
measurement-output systems (remainders, M1 branches, M3 collapsed qubits)
are transparent downstream — their qubits resolve to the uncollapsed input
state. Both sides (this replay and Qiskit) simulate the compiled,
measurement-free circuit, which is equivalent by the deferred-measurement
principle. Outcome statistics are NOT verified. Soundness guards, all
errors: a collapsed/output qubit feeding two different gates (fan-out of
one state object), and a control qubit whose carrier also feeds a unitary
gate (its state at control time would be order-dependent).
"""

import json

import numpy as np


class QsimError(Exception):
    pass


T = 1 / 2**0.5
REFS = {
    (2, 2): {
        "h": np.array([[T, T], [T, -T]]),
        "x": np.array([[0, 1], [1, 0]]),
        "y": np.array([[0, -1j], [1j, 0]]),
        "z": np.array([[1, 0], [0, -1]]),
    },
    (4, 4): {
        "cx": np.array([[1, 0, 0, 0], [0, 1, 0, 0], [0, 0, 0, 1], [0, 0, 1, 0]]),
        "cy": np.array([[1, 0, 0, 0], [0, 1, 0, 0], [0, 0, 0, -1j], [0, 0, 1j, 0]]),
        "cz": np.array([[1, 0, 0, 0], [0, 1, 0, 0], [0, 0, 1, 0], [0, 0, 0, -1]]),
    },
}

CTRL_1Q = ["x", "y", "z"]  # ControlledGate Kind 0/1/2


def _c(z):
    return complex(z["real"], z["imag"])


def _mat(op):
    return np.array([[ _c(v) for v in row] for row in op], dtype=complex)


def classify(mat):
    refs = REFS.get(mat.shape, {})
    for name, ref in refs.items():
        if np.allclose(mat, ref, atol=1e-6):
            return name
    return "unitary"


def load_window(path):
    with open(path, encoding="utf-8") as f:
        text = f.read()
    for line in text.replace("\r", "\n").split("\n"):
        line = line.strip()
        if not line:
            continue
        try:
            w = json.loads(line)
        except ValueError:
            continue
        if isinstance(w, dict) and w.get("type") == "RenderWindow":
            return w
    raise QsimError("save contains no circuit window")


class State:
    """amps indexed MSB-first over mods (mods[0] = most significant bit)."""

    def __init__(self, amps, mods):
        self.amps = np.asarray(amps, dtype=complex)
        self.mods = list(mods)

    def copy(self):
        return State(self.amps.copy(), list(self.mods))


def merge(a, b):
    return State(np.kron(a.amps, b.amps), a.mods + b.mods)


def swap_front(st, order):
    """Permute so mods run [order..., rest...], like sequential SwapColumn."""
    n = len(st.mods)
    pos = {m: k for k, m in enumerate(st.mods)}
    rest = [m for m in st.mods if m not in set(order)]
    if len(order) + len(rest) != n:
        raise QsimError(f"gate input modifier not in state: {order}")
    new_mods = list(order) + rest
    out = np.zeros_like(st.amps)
    for i in range(1 << n):
        j = 0
        for k2, m in enumerate(new_mods):
            bit = (i >> (n - 1 - pos[m])) & 1
            j |= bit << (n - 1 - k2)
        out[j] = st.amps[i]
    return State(out, new_mods)


def apply_unitary(st, mods, mat):
    st = swap_front(st, mods)
    n = int(round(np.log2(mat.shape[0])))
    rest = len(st.mods) - n
    full = np.kron(mat, np.eye(1 << rest, dtype=complex)) if rest else mat
    return State(full @ st.amps, st.mods)


def apply_controlled(st, ctrl, tgts, mat):
    """Apply controlled-mat (control first) and restore modifier order."""
    order = [ctrl] + list(tgts)
    keep = list(st.mods)
    st = swap_front(st, order)
    n = int(round(np.log2(mat.shape[0])))
    dim = 1 << n
    cu = np.zeros((2 * dim, 2 * dim), dtype=complex)
    cu[:dim, :dim] = np.eye(dim)
    cu[dim:, dim:] = mat
    rest = len(st.mods) - (n + 1)
    full = np.kron(cu, np.eye(1 << rest, dtype=complex)) if rest else cu
    return swap_front(State(full @ st.amps, st.mods), keep)


def controlled_matrix(mat):
    n = mat.shape[0]
    cu = np.zeros((2 * n, 2 * n), dtype=complex)
    cu[:n, :n] = np.eye(n)
    cu[n:, n:] = mat
    return cu


def outcome_prob(m, mod, k):
    """P(measuring mod == k) from a manager state."""
    n = len(m.mods)
    pos = m.mods.index(mod)
    return float(sum(abs(a) ** 2 for i, a in enumerate(m.amps)
                     if ((i >> (n - 1 - pos)) & 1) == k))


def normalize(v):
    v = np.asarray(v, dtype=complex)
    n = float(np.vdot(v, v).real)
    if n == 0:
        raise QsimError("zero-vector input state")
    return v / np.sqrt(n)


def gate_hooks(comp):
    """Return (input_hooks, output_hooks) for gate-like components."""
    t = comp["type"]
    if t == "Gate":
        n = comp.get("inputCount", 0)
        hs = comp.get("hooks", [])
        if comp.get("isMeasurementGate"):
            return hs[:1], hs[1:]
        return hs[:n], hs[n:n + 1]
    if t == "CollapseGate" or t == "M4Gate":
        hs = comp.get("hooks", [])
        return hs[:1], hs[1:]
    if t == "ControlledGate":
        return [comp.get("inQubit")], [comp.get("outHook")]
    if t == "ControlledUGate":
        return list(comp.get("hooks", [])), [comp.get("out")]
    return [], []


def comp_hooks(c):
    t = c.get("type")
    if t == "Gate" or t == "CollapseGate" or t == "M4Gate":
        return c.get("hooks", [])
    if t == "SourceGate":
        return [c.get("outHook")]
    if t == "CopyGate":
        return [c.get("inHook"), c.get("outHook")]
    if t == "ControlledGate":
        return [c.get("inQubit"), c.get("inControl"), c.get("outHook")]
    if t == "ControlledUGate":
        return list(c.get("hooks", [])) + [c.get("control"), c.get("out")]
    if t == "CompareGate":
        return [c.get("inA"), c.get("inB"), c.get("outHook")]
    return []


def extract(path, name=None):
    """Extract the circuit: returns dict with qubits/init/ops/steps/notes.

    ops are checker-ready: {gate, qubits (qiskit order, little-endian),
    matrix?, label}. steps[i] = {label, gate, qubits, state} with state the
    FULL n-qubit vector in Qiskit order after step i.
    """
    w = load_window(path)
    comps = w.get("components", [])
    systems = {c["id"]: c for c in comps if c.get("type") == "QubitsSystem"
               and "id" in c}
    if not systems:
        raise QsimError("save contains no qubit systems")
    bits = {c["id"]: c for c in comps if c.get("type") == "LogicalBit"
            and "id" in c}
    dets = {}
    for s in systems.values():
        for d in s.get("qubitDeterminators", []):
            if "id" in d:
                dets[d["id"]] = (d, s["id"])

    # Normalize dangling links up front, mirroring the loader's remap
    # cleanup (and the engine, which treats them as inert): a hooked hook
    # whose target names no known object becomes unconnected, and likewise
    # for determinator/system back-links naming unknown hooks.
    known_ids = set(systems) | set(bits) | set(dets)
    hook_ids = {h["id"] for c in comps for h in comp_hooks(c)
                if h and "id" in h}
    for c in comps:
        for h in comp_hooks(c):
            if h and h.get("isHooked") and h.get("targetID") not in known_ids:
                h["isHooked"] = False
                h["targetID"] = 0
    for s in systems.values():
        if s.get("hookID") not in hook_ids:
            s["hookID"] = 0
        if s.get("infoHookID") not in hook_ids:
            s["infoHookID"] = 0
        for d in s.get("qubitDeterminators", []):
            if d.get("hookID") not in hook_ids:
                d["hookID"] = 0

    managers = {}   # sys id -> State (never mutated after creation)
    ver = {}        # modifier -> State holding its current value (SSA style:
                    # each replayed output overwrites its mods; copies and
                    # transparent measurement outputs share objects without
                    # forking the version, so every qubit has exactly one
                    # current holder by construction)
    is_init = set()
    exclusive = {}  # sys id -> [(desc, gate id)] (unitary-class consumers)
    skip_meas = set()  # measurement comp ids with unconnected inputs
    meas_comps = []  # collapse/M1 comps with hooked inputs, in order
    meas_probs = {}  # collapse/M1 comp id -> (p0, p1) at replay time
    compiled = set()  # collapse ids whose C-bit drove a compiled control
    meas_parent = {}  # meas output sys id -> measurement input sys id
    meas_children = {}  # collapse comp id -> [output sys ids]
    meas = {}       # collapse comp id -> {"sys","mod","bit","label"}
    bit_drivers = {}  # bit id -> [("meas",mid)|("const",v)|("logic",)]
    notes = []

    hook_owner = {}
    for c in comps:
        for h in comp_hooks(c):
            if h and "id" in h:
                hook_owner[h["id"]] = c

    def is_live(h):
        return bool(h) and bool(h.get("isHooked"))

    def is_spare(c):
        t = c.get("type")
        if t == "Gate":
            return not any(is_live(h) for h in c.get("hooks", []))
        if t == "CopyGate":
            return not is_live(c.get("inHook")) and not is_live(c.get("outHook"))
        if t == "ControlledGate":
            return not is_live(c.get("inQubit")) and not is_live(c.get("outHook"))
        if t == "ControlledUGate":
            return not any(is_live(h) for h in c.get("hooks", [])) \
                and not is_live(c.get("out"))
        return False

    spare = {c.get("id") for c in comps
             if c.get("type") in ("Gate", "CopyGate", "ControlledGate",
                                   "ControlledUGate") and is_spare(c)}

    def producer_hook_id(s):
        hid = s.get("hookID", 0)
        if hid:
            return hid
        for c in comps:
            for h in comp_hooks(c):
                if (h or {}).get("targetID") == s["id"]:
                    return h["id"]
        return 0

    # --- producers / seeds ---
    for s in systems.values():
        hid = producer_hook_id(s)
        if not hid:
            o = s.get("origin") or {}
            amps = [_c(a) for a in o.get("amplitudes", [])]
            mods = list(o.get("modifierIDs", []))
            if len(amps) != 2 or len(mods) != 1:
                raise QsimError(
                    f"standalone {len(amps)}-amp system (id {s['id']}) is not "
                    "supported; use 1-qubit inputs")
            st = State(normalize(amps), mods)
            managers[s["id"]] = st
            for mod in mods:
                ver[mod] = st
            is_init.add(s["id"])
            continue
        owner = hook_owner.get(hid, {})
        t = owner.get("type")
        if t == "SourceGate":
            amps = [_c(a) for a in owner.get("amplitude", [])]
            if len(amps) != 2:
                raise QsimError(f"source {owner.get('label')!r} is not 1-qubit")
            st = State(normalize(amps), [owner["modifierID"]])
            managers[s["id"]] = st
            for mod in st.mods:
                ver[mod] = st
            is_init.add(s["id"])
        elif t == "Gate" and owner.get("isMeasurementGate"):
            meas_children.setdefault(owner["id"], []).append(s["id"])
        elif t == "CollapseGate" or t == "M4Gate":
            meas_children.setdefault(owner["id"], []).append(s["id"])
        elif t in ("Gate", "CopyGate", "ControlledGate", "ControlledUGate"):
            pass  # replayed later
        else:
            raise QsimError(
                f"qubit system (id {s['id']}) is produced by unsupported {t}")

    # --- consumers (pass 1: everything but control resolution) ---
    def input_sys(h, what):
        if not is_live(h):
            raise QsimError(f"{what} is unconnected")
        dd = dets.get(h.get("targetID"))
        if dd is None:
            raise QsimError(f"{what} links to a non-qubit target")
        return dd[1], dd[0]["modifierID"]

    for c in comps:
        t = c.get("type")
        label = str(c.get("label", t))
        if c.get("id") in spare:
            notes.append(f"{label}: unconnected spare, skipped")
            continue
        if t == "Gate" and c.get("isMeasurementGate"):
            hs, _ = gate_hooks(c)
            if not hs or not is_live(hs[0]):
                notes.append(f"M1 {label}: input unconnected, skipped")
                skip_meas.add(c["id"])
                meas_children.pop(c["id"], None)
                continue
            sys, _ = input_sys(hs[0], f"M1 {label} input")
            for child in meas_children.get(c["id"], []):
                meas_parent[child] = sys
            meas_comps.append(c)
        elif t == "Gate":
            ins, _ = gate_hooks(c)
            if len(ins) < c.get("inputCount", 0):
                raise QsimError(f"gate {label!r} is missing input hooks")
            for i in range(c.get("inputCount", 0)):
                sys, _ = input_sys(ins[i], f"gate {label} input I{i}")
                exclusive.setdefault(sys, []).append((f"gate {label}", c["id"]))
        elif t == "CopyGate":
            h = c.get("inHook") or {}
            if h.get("isHooked") or h.get("targetID"):
                sys = systems.get(h.get("targetID"))
                if sys is None:
                    raise QsimError(f"copy gate {label!r} input links to a "
                                    "non-system target")
            # Read-only mirror (may share its system); inputless copies
            # produce nothing (checked at replay).
        elif t == "CollapseGate":
            ins, outs = gate_hooks(c)
            if not ins or not is_live(ins[0]):
                notes.append(f"{label}: input unconnected, skipped")
                skip_meas.add(c["id"])
                meas_children.pop(c["id"], None)
                continue
            sys, mod = input_sys(ins[0], f"{label} input")
            for child in meas_children.get(c["id"], []):
                meas_parent[child] = sys
            cbit = (outs[0] or {}).get("targetID") if outs else 0
            if cbit in bits and not c.get("normalSystem", False):
                meas[c["id"]] = {"sys": sys, "mod": mod, "bit": cbit,
                                 "label": label}
                bit_drivers.setdefault(cbit, []).append(("meas", c["id"]))
            meas_comps.append(c)
        elif t == "ControlledGate":
            sys, _ = input_sys(c.get("inQubit"), f"controlled gate {label} input")
            exclusive.setdefault(sys, []).append(
                (f"controlled gate {label}", c["id"]))
        elif t == "ControlledUGate":
            ins, _ = gate_hooks(c)
            for i, h in enumerate(ins[:c.get("inputCount", 0)]):
                sys, _ = input_sys(h, f"controlled-U {label} input I{i}")
                exclusive.setdefault(sys, []).append(
                    (f"controlled-U {label}", c["id"]))
        elif t == "LogicButton":
            h = c.get("outHook") or {}
            outbit = c.get("outputID", 0)
            for bid in {h.get("targetID", 0), outbit}:
                if bid in bits:
                    bit_drivers.setdefault(bid, []).append(
                        ("const", c.get("value", 0)))
        elif t == "LogicGate":
            h = c.get("outHook") or {}
            outbit = c.get("outputID", 0)
            for bid in {h.get("targetID", 0), outbit}:
                if bid in bits:
                    bit_drivers.setdefault(bid, []).append(("logic",))
        elif t == "M4Gate":
            # Read-only like a collapse (input continues, outputs strip if
            # terminal), but the C-bit oscillates: usable by nothing.
            ins, outs = gate_hooks(c)
            if not ins or not is_live(ins[0]):
                notes.append(f"{label}: input unconnected, skipped")
                skip_meas.add(c["id"])
                meas_children.pop(c["id"], None)
                continue
            sys, mod = input_sys(ins[0], f"{label} input")
            for child in meas_children.get(c["id"], []):
                meas_parent[child] = sys
            cbit = (outs[0] or {}).get("targetID") if outs else 0
            if cbit in bits:
                bit_drivers.setdefault(cbit, []).append(("m4",))
            meas_comps.append(c)
        elif t == "DecomposeGate":
            if any(is_live(h) for h in comp_hooks(c)):
                raise QsimError(f"decompose gate {label!r} is connected but "
                                "has no operation")

    for sys, who in sorted(exclusive.items()):
        gates = {g: d for d, g in who}
        if len(gates) > 1:
            raise QsimError(
                f"qubit system (id {sys}) feeds {len(gates)} gates "
                f"({', '.join(sorted(gates.values()))}); one system feeds "
                "one gate")

    # --- control resolution (pass 2: drivers are all registered) ---
    def distinct_gates(sys):
        return {g for _, g in exclusive.get(sys, [])}

    def control_carrier(mid, what, self_gid=0):
        """Produced system holding the measured qubit, walked transparent
        through measurement outputs. Must be free of unitary consumers
        (other than the controlled gate itself) so its stored state is
        time-independent."""
        sys = meas[mid]["sys"]
        seen = set()
        while sys in meas_parent:
            if sys in seen:
                raise QsimError(f"{what}: measurement chain cycle")
            seen.add(sys)
            sys = meas_parent[sys]
        others = distinct_gates(sys) - {self_gid}
        if others:
            raise QsimError(
                f"{what}: control qubit carrier system (id {sys}) also feeds "
                f"gate(s) {sorted(others)}; deferred measurement needs it "
                "measurement-only")
        return sys

    controls = {}  # gate comp id -> ("qctl", mid) | ("const", v)
    for c in comps:
        t = c.get("type")
        if t not in ("ControlledGate", "ControlledUGate"):
            continue
        if c.get("id") in spare:
            continue
        label = str(c.get("label", t))
        h = (c.get("inControl") if t == "ControlledGate"
             else c.get("control")) or {}
        what = f"controlled gate {label!r}" if t == "ControlledGate" \
            else f"controlled-U {label!r}"
        if not h.get("isHooked"):
            controls[c["id"]] = ("const", 0)
            continue
        bid = h.get("targetID")
        if bid not in bits:
            controls[c["id"]] = ("const", 0)  # engine inputBit nil -> 0
            notes.append(f"{what}: control targets a non-bit, reads 0")
            continue
        drvs = bit_drivers.get(bid, [])
        if not drvs:
            v = bits[bid].get("value", 0)
            controls[c["id"]] = ("const", v)
            notes.append(f"{what}: undriven control bit reads stored value {v}")
            continue
        kinds = {d[0] for d in drvs}
        if kinds == {"const"} and len({d[1] for d in drvs}) == 1:
            controls[c["id"]] = ("const", drvs[0][1])
        elif kinds == {"meas"} and len(drvs) == 1:
            mid = drvs[0][1]
            control_carrier(mid, what, c["id"])
            controls[c["id"]] = ("qctl", mid)
            compiled.add(mid)
        elif kinds == {"m4"}:
            raise QsimError(
                f"{what} control bit is driven by an oscillating (M4) "
                "measurement and has no static value")
        else:
            raise QsimError(
                f"{what} control bit value is not statically known "
                "(only measurement-compiled, button-driven, or stored bits "
                "verify)")

    # --- qubit mapping (save order over init systems) ---
    modifiers, inits = [], []
    for c in comps:
        if c.get("type") == "QubitsSystem" and c["id"] in is_init:
            m = managers[c["id"]]
            if len(m.mods) != 1 or len(m.amps) != 2:
                raise QsimError(
                    f"input system (id {c['id']}) is not a single qubit")
            if m.mods[0] in modifiers:
                raise QsimError(f"modifier {m.mods[0]} is initialized twice")
            modifiers.append(m.mods[0])
            inits.append((complex(m.amps[0]), complex(m.amps[1])))
    if not modifiers:
        raise QsimError("no source or standalone input systems found")
    n = len(modifiers)
    qindex = {m: k for k, m in enumerate(modifiers)}

    def resolve_mgr(sys_id):
        """Manager object holding sys_id's qubits, transparent through
        measurement outputs (deferred measurement: remainders/branches see
        the uncollapsed input state)."""
        seen = set()
        while sys_id in meas_parent:
            if sys_id in seen:
                raise QsimError("measurement chain cycle")
            seen.add(sys_id)
            sys_id = meas_parent[sys_id]
        return managers.get(sys_id)

    def full_state():
        """Tensor product of current holders in Qiskit order."""
        out = np.zeros(1 << n, dtype=complex)
        owners = {}
        for mod, m in ver.items():
            k = m.mods.index(mod)
            owners[mod] = (m, k)
        if len(owners) != n:
            raise QsimError("lost track of qubits during replay")
        for full in range(1 << n):
            amp = complex(1)
            per = {}
            for qi, mod in enumerate(modifiers):
                m, k = owners[mod]
                bit = (full >> qi) & 1
                per[id(m)] = per.get(id(m), [m, 0])
                per[id(m)][1] |= bit << (len(m.mods) - 1 - k)
            for m, idx in per.values():
                amp *= m.amps[idx]
            out[full] = amp
        return out

    uses = {}  # id(manager object) -> set of gate ids (fan-out guard)
    compiled = set()  # collapse ids whose C-bit drove a compiled control

    def use(mgr, gid, what):
        s = uses.setdefault(id(mgr), set())
        s.add(gid)
        if len(s) > 1:
            raise QsimError(
                f"{what}: one state feeds two gates (fan-out of a collapsed "
                "or remainder state is not verifiable)")

    ops, steps = [], []
    steps.append({"label": "init", "gate": "init", "qubits": [],
                  "state": full_state()})

    # --- replay in dependency order ---
    def out_system(c, label):
        t = c["type"]
        oh = (c.get("outHook") or {}) if t == "CopyGate" else {}
        if t != "CopyGate":
            _, outs = gate_hooks(c)
            oh = outs[0] if outs else {}
        otid = (oh or {}).get("targetID", 0)
        if not otid:
            raise QsimError(f"{label} output is unconnected")
        if otid not in systems:
            raise QsimError(f"{label} output links to a non-system target")
        return otid

    pending = [c for c in comps if c.get("id") not in spare
               and c.get("id") not in skip_meas
               and (c.get("type") in ("Gate", "CopyGate", "ControlledGate",
                                      "ControlledUGate", "CollapseGate",
                                      "M4Gate"))]
    pending.sort(key=lambda c: c.get("id", 0))

    def job_inputs(c, label):
        """(sys, mod) inputs of a replay job (raises wiring errors)."""
        t = c["type"]
        if t == "Gate":
            ins, _ = gate_hooks(c)
            return [input_sys(h, f"gate {label} input I{i}")
                    for i, h in enumerate(ins[:c.get("inputCount", 0)])]
        if t == "ControlledGate":
            return [input_sys(c.get("inQubit"),
                              f"controlled gate {label} input")]
        if t == "ControlledUGate":
            ins, _ = gate_hooks(c)
            return [input_sys(h, f"controlled-U {label} input I{i}")
                    for i, h in enumerate(ins[:c.get("inputCount", 0)])]
        if t == "CopyGate":
            h = c.get("inHook") or {}
            intid = h.get("targetID", 0)
            if intid and intid in systems:
                return [(intid, 0)]
            return []
        return []

    def walk_root(sys):
        seen = set()
        while sys in meas_parent:
            if sys in seen:
                return sys
            seen.add(sys)
            sys = meas_parent[sys]
        return sys

    produced_set = set()
    for c in pending:
        t = c["type"]
        if t in ("CollapseGate", "M4Gate") or \
                (t == "Gate" and c.get("isMeasurementGate")):
            continue
        try:
            produced_set.add(out_system(c, str(c.get("label", t))))
        except QsimError:
            pass

    def produced():
        out = set()
        for c in pending:
            try:
                out.add(out_system(c, str(c.get("label", c.get("type")))))
            except QsimError:
                pass
        return out

    def stuck_reason():
        prod = produced()
        import os
        dbg = os.environ.get("QVERIFY_DEBUG", "")
        for c in pending:
            if id(c) in done:
                continue
            label = str(c.get("label", c.get("type")))
            try:
                inps = job_inputs(c, label)
            except QsimError as e:
                return str(e)
            for s, _ in inps:
                if resolve_mgr(s) is None and s not in prod:
                    return (f"{label} is fed by system {s}, which has no "
                            f"producer in the circuit (stale output of a "
                            f"skipped measurement?)")
            if dbg:
                missing = [s for s, _ in inps if resolve_mgr(s) is None]
                print(f"DBG stuck {label} id={c.get('id')} missing inputs {missing} "
                      f"(all have producers)")
        return "circuit has a dependency cycle"

    done = set()
    while len(done) < len(pending):
        progress = False
        for c in pending:
            if id(c) in done:
                continue
            t = c["type"]
            label = str(c.get("label", t))
            if t in ("CollapseGate", "M4Gate") or \
                    (t == "Gate" and c.get("isMeasurementGate")):
                # Measurements compile away (deferred measurement): log the
                # step with the unchanged state, no manager changes. Record
                # the outcome distribution: deterministic outcomes verify
                # exactly, random ones only at protocol level.
                ins, _ = gate_hooks(c)
                msys, mmod = input_sys(ins[0], f"{label} input")
                mgr = resolve_mgr(msys)
                if mgr is None:
                    if walk_root(msys) in produced_set:
                        continue  # input system not replayed yet
                    if c["id"] in compiled:
                        raise QsimError(
                            f"{label}: a control was compiled from this "
                            f"measurement, but its input has no live state")
                    notes.append(f"{label}: input has no live state, skipped")
                    skip_meas.add(c["id"])
                    meas_children.pop(c["id"], None)
                    done.add(id(c))
                    progress = True
                    continue
                p0 = outcome_prob(mgr, mmod, 0)
                steps.append({"label": label, "gate": "measure", "qubits": [],
                              "state": full_state(), "noop": True,
                              "p0": p0, "p1": 1.0 - p0})
                if t != "M4Gate":
                    meas_probs[c["id"]] = (p0, 1.0 - p0)
                done.add(id(c))
                progress = True
                continue
            out_sys = out_system(c, label)
            if t == "Gate":
                ins, _ = gate_hooks(c)
                inps = [input_sys(h, f"gate {label} input I{i}")
                        for i, h in enumerate(ins[:c.get("inputCount", 0)])]
                mgrs = [(resolve_mgr(s), mod) for s, mod in inps]
                if any(m is None for m, _ in mgrs):
                    continue
                uniq, order, seen = [], [], set()
                for (m, _), (_, mod) in zip(mgrs, inps):
                    order.append(mod)
                    if id(m) not in seen:
                        seen.add(id(m))
                        uniq.append(m)
                res = State(np.array([1], dtype=complex), [])
                for m in uniq:
                    res = merge(res, m)
                mat = _mat(c["operation"])
                res = apply_unitary(res, order, mat)
                managers[out_sys] = res
                for m in uniq:
                    use(m, c["id"], f"gate {label}")
                name = classify(mat)
                qs = [qindex[mod] for mod in order]
                emit = {"gate": name, "qubits": qs, "label": label}
                if name == "unitary":
                    emit["qubits"] = qs[::-1]
                    emit["matrix"] = mat
            elif t == "CopyGate":
                h = c.get("inHook") or {}
                intid = h.get("targetID", 0)
                if not intid or intid not in systems:
                    fed = any(g != c["id"] for _, g in exclusive.get(out_sys, []))
                    if fed:
                        raise QsimError(
                            f"copy {label!r} has no input but its output "
                            "feeds computation; the state is undetermined")
                    notes.append(f"copy {label}: no input, output ignored")
                    done.add(id(c))
                    progress = True
                    continue
                m = resolve_mgr(intid)
                if m is None:
                    continue
                # Read-only mirror: no use recorded (sanctioned sharing).
                managers[out_sys] = m.copy()
                notes.append(f"copy {label}: state carried through")
                emit = {"gate": "copy", "qubits": [], "label": label}
                steps.append({"label": label, "gate": "copy", "qubits": [],
                              "state": full_state(), "noop": False})
                ops.append(emit)
                done.add(id(c))
                progress = True
                continue
            elif t in ("ControlledGate", "ControlledUGate"):
                kind, ctl = controls[c["id"]]
                if t == "ControlledGate":
                    inps = [input_sys(c.get("inQubit"),
                                      f"controlled gate {label} input")]
                else:
                    ins, _ = gate_hooks(c)
                    inps = [input_sys(h, f"controlled-U {label} input I{i}")
                            for i, h in enumerate(ins[:c.get("inputCount", 0)])]
                mgrs = [(resolve_mgr(s), mod) for s, mod in inps]
                if any(m is None for m, _ in mgrs):
                    continue
                if kind == "const" and ctl == 0:
                    res = merge_all(mgrs, inps, None,
                                    use, c["id"], label)
                    managers[out_sys] = res
                    notes.append(f"{label}: control 0, passed through")
                    emit = {"gate": "identity", "qubits": [], "label": label}
                elif kind == "const":
                    if t == "ControlledGate":
                        mat = [np.array([[0, 1], [1, 0]], dtype=complex),
                               np.array([[0, -1j], [1j, 0]], dtype=complex),
                               np.array([[1, 0], [0, -1]], dtype=complex)
                               ][c.get("kind", 0)]
                        name = CTRL_1Q[c.get("kind", 0)]
                    else:
                        mat = _mat(c["operation"])
                        name = classify(mat)
                    res = merge_all(mgrs, inps, mat, use, c["id"], label)
                    managers[out_sys] = res
                    qs = [qindex[mod] for _, mod in inps]
                    emit = {"gate": name, "qubits": qs, "label": label}
                    if name == "unitary":
                        emit["qubits"] = qs[::-1]
                        emit["matrix"] = mat
                else:  # quantum control via deferred measurement
                    mid = ctl
                    compiled.add(mid)
                    cmod = meas[mid]["mod"]
                    # Current holder: sequential compiled controls chain
                    # through the version map (restriction checked up front).
                    cmgr = ver[cmod]
                    if t == "ControlledGate":
                        mat = [np.array([[0, 1], [1, 0]], dtype=complex),
                               np.array([[0, -1j], [1j, 0]], dtype=complex),
                               np.array([[1, 0], [0, -1]], dtype=complex)
                               ][c.get("kind", 0)]
                        # Compiled to a real quantum control: cx/cy/cz, not
                        # the 1q x/y/z names (those mean single-qubit downstream).
                        name = ["cx", "cy", "cz"][c.get("kind", 0)]
                    else:
                        mat = _mat(c["operation"])
                    uniq, seen = [cmgr], {id(cmgr)}
                    for (m, _) in mgrs:
                        if id(m) not in seen:
                            seen.add(id(m))
                            uniq.append(m)
                    res = State(np.array([1], dtype=complex), [])
                    for m in uniq:
                        res = merge(res, m)
                    res = apply_controlled(res, cmod,
                                           [mod for _, mod in inps], mat)
                    managers[out_sys] = res
                    for m in uniq:
                        use(m, c["id"], f"{label} (control/target)")
                    qc = qindex[cmod]
                    qt = [qindex[mod] for _, mod in inps]
                    if t == "ControlledGate":
                        emit = {"gate": name, "qubits": [qc] + qt,
                                "label": label}
                    else:
                        emit = {"gate": "unitary",
                                "qubits": qt[::-1] + [qc],
                                "matrix": controlled_matrix(mat),
                                "label": label}
            for mod in res.mods:
                ver[mod] = res
            steps.append({"label": label, "gate": emit["gate"],
                          "qubits": list(emit["qubits"]),
                          "state": full_state(), "noop": False})
            ops.append(emit)
            done.add(id(c))
            progress = True
        if not progress:
            raise QsimError(stuck_reason())
    for c in meas_comps:
        label = str(c.get("label", c.get("type")))
        det = ""
        probs = meas_probs.get(c["id"])
        if probs is not None and max(probs) > 1 - 1e-9:
            det = f"deterministic outcome {0 if probs[0] > 0.5 else 1}, exact; "
        children = meas_children.get(c["id"], [])
        fed = any(exclusive.get(s) for s in children)
        if c["id"] in compiled:
            base = "control(s) compiled to quantum control (deferred measurement)"
        elif fed:
            base = "remainder feeds downstream transparently (deferred measurement)"
        else:
            base = "terminal readout stripped, pre-measurement state compared"
        tail = "" if det else "; outcome statistics not verified"
        notes.append(f"{label}: {det}{base}{tail}")
    return {"name": name or str(path),
            "n_qubits": n, "modifiers": modifiers, "inits": inits,
            "qindex": qindex, "ops": ops, "steps": steps, "notes": notes,
            "expected": steps[-1]["state"]}


def merge_all(mgrs, inps, mat, use, gid, label):
    """Merge input managers, optionally apply mat, for passthrough/const."""
    uniq, order, seen = [], [], set()
    for (m, _), (_, mod) in zip(mgrs, inps):
        order.append(mod)
        if id(m) not in seen:
            seen.add(id(m))
            uniq.append(m)
    res = State(np.array([1], dtype=complex), [])
    for m in uniq:
        res = merge(res, m)
    if mat is not None:
        res = apply_unitary(res, order, mat)
    else:
        res = swap_front(res, order)
    for m in uniq:
        use(m, gid, label)
    return res
