#!/usr/bin/env python3
"""Cross-check Qsim circuits against Qiskit.

Two input kinds (files are the only interface; nothing is executed by Go):

  <name>.verify.json   bundle written by qsim's verify package: rebuilds
                       the circuit, simulates, asserts final fidelity.
  <name>.qsim          save file, parsed directly (raw file IDs, numpy
                       replay mirroring the engine): every replay step is
                       diffed against Qiskit in a step-by-step test log.

A directory target checks every bundle/save inside in batch.

Requires: pip install qiskit

Usage:
    py verify/qiskit_check.py verify/testdata/bell.verify.json [--tol 1e-6]
    py verify/qiskit_check.py saves/superdense.qsim --log check.log
    py verify/qiskit_check.py verify/testdata/

Display (all plain text): single target prints the step log for .qsim,
the verdict, and both final state vectors (expected vs Qiskit); a
directory prints a summary table plus the vectors per bundle.

Exit code 0 = all PASS, 1 = any FAIL, 2 = usage/error.
"""

import argparse
import json
import os
import sys

try:
    import numpy as np
except ImportError:
    sys.stderr.write("error: numpy is required (pip install qiskit)\n")
    sys.exit(2)

try:
    from qiskit import QuantumCircuit
    from qiskit.circuit.library import UnitaryGate
    from qiskit.quantum_info import Statevector
except ImportError:
    sys.stderr.write("error: qiskit is required (pip install qiskit)\n")
    sys.exit(2)

try:
    from qsim_circuit import QsimError, extract as extract_qsim
except ImportError:  # run from outside verify/
    sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
    from qsim_circuit import QsimError, extract as extract_qsim

T = 1 / 2**0.5
REFS_1Q = {
    "h": np.array([[T, T], [T, -T]]),
    "x": np.array([[0, 1], [1, 0]]),
    "y": np.array([[0, -1j], [1j, 0]]),
    "z": np.array([[1, 0], [0, -1]]),
}
REFS_2Q = {
    "cx": np.array([[1, 0, 0, 0], [0, 1, 0, 0], [0, 0, 0, 1], [0, 0, 1, 0]]),
    "cy": np.array([[1, 0, 0, 0], [0, 1, 0, 0], [0, 0, 0, -1j], [0, 0, 1j, 0]]),
    "cz": np.array([[1, 0, 0, 0], [0, 1, 0, 0], [0, 0, 1, 0], [0, 0, 0, -1]]),
}


def classify(mat):
    """Name a unitary matrix, else 'unitary' (exact match, tol 1e-6)."""
    refs = REFS_1Q if mat.shape == (2, 2) else REFS_2Q if mat.shape == (4, 4) else {}
    for name, ref in refs.items():
        if np.allclose(mat, ref, atol=1e-6):
            return name
    return "unitary"


def prep_unitary(a, b):
    """2x2 unitary mapping |0> to a|0> + b|1> (columns orthonormal)."""
    n = abs(a) ** 2 + abs(b) ** 2
    if n == 0:
        raise ValueError("prep state is a zero vector")
    a, b = a / n**0.5, b / n**0.5
    return np.array([[a, -b.conjugate()], [b, a.conjugate()]], dtype=complex)


def apply_checker_op(circ, op):
    """Apply one checker op {gate, qubits, matrix?, state?, label}.

    Qubit lists are little-endian (last listed = matrix MSB), matching
    UnitaryGate and the bundle sidecar convention.
    """
    g, qs = op["gate"], op["qubits"]
    if g == "x":
        circ.x(qs[0])
    elif g in ("h", "y", "z"):
        getattr(circ, g)(qs[0])
    elif g in ("cx", "cy", "cz"):
        getattr(circ, g)(qs[0], qs[1])
    elif g == "prep":
        circ.append(UnitaryGate(prep_unitary(*op["state"])), qs)
    elif g == "unitary":
        circ.append(UnitaryGate(np.asarray(op["matrix"]), label=op.get("label", "U")), qs)
    elif g in ("copy", "identity"):
        pass
    else:
        raise ValueError(f"unknown gate {g!r}")


def simulate(n, ops):
    """Build the circuit incrementally; return statevectors after each op.

    Initial states travel as leading x/prep ops (as the Go extractor
    emits them), so every op list is self-contained starting from |0>.
    """
    circ = QuantumCircuit(n)
    states = []
    for op in ops:
        apply_checker_op(circ, op)
        states.append(np.asarray(Statevector(circ).data, dtype=complex))
    return states


def init_ops(inits):
    """Leading x/prep ops for non-|0> initial states."""
    out = []
    for qi, (a, b) in enumerate(inits):
        if abs(a - 1) < 1e-9 and abs(b) < 1e-9:
            continue
        if abs(a) < 1e-9 and abs(abs(b) - 1) < 1e-9:
            out.append({"gate": "x", "qubits": [qi], "label": "init|1>"})
        else:
            out.append({"gate": "prep", "qubits": [qi], "state": [a, b],
                        "label": "init"})
    return out


def to_checker_ops(qops, qindex):
    """Extract ops are already checker-ready (qiskit-ordered qubits, with
    matrices for customs); normalize container types only."""
    out = []
    for op in qops:
        o = {"gate": op["gate"], "qubits": list(op["qubits"]),
             "label": op.get("label", "")}
        if "matrix" in op:
            o["matrix"] = np.asarray(op["matrix"])
        out.append(o)
    return out


def amp_rows(exp, got, n):
    errs = np.abs(exp - got * np.conj(np.vdot(exp, got)))
    return [{"i": i, "bits": format(i, f"0{n}b"), "exp": exp[i],
             "got": got[i], "err": float(errs[i])} for i in range(exp.size)]


def finish(name, n, exp, got, tol, extra):
    norm = float(np.vdot(exp, exp).real)
    if norm == 0:
        return {"path": extra.get("path", name), "tol": tol,
                "status": "ERROR", "error": "expected state is a zero vector"}
    exp = exp / norm**0.5
    fid = abs(complex(np.vdot(exp, got))) ** 2
    infid = 1.0 - fid
    rows = amp_rows(exp, got, n)
    res = {"name": name, "n_qubits": n, "tol": tol,
           "fidelity": float(fid), "infidelity": float(infid),
           "status": "PASS" if infid <= tol else "FAIL",
           "rows": rows, "worst": max(rows, key=lambda r: r["err"])}
    res.update(extra)
    return res


def check_bundle(path, tol):
    res = {"path": path, "tol": tol, "status": "ERROR"}
    try:
        with open(path, encoding="utf-8") as f:
            b = json.load(f)
        n = b["n_qubits"]
        inits = [(q["init"][0]["real"] + 1j * q["init"][0]["imag"],
                  q["init"][1]["real"] + 1j * q["init"][1]["imag"])
                 for q in b["qubits"]]
        ops = [{"gate": o["gate"], "qubits": o["qubits"],
                "matrix": [[c["real"] + 1j * c["imag"] for c in row]
                           for row in o.get("matrix", [])],
                "state": [(c["real"] + 1j * c["imag"]) for c in o.get("state", [])],
                "label": o.get("label", "")} for o in b["ops"]]
        states = simulate(n, ops)
        exp = np.array([c["real"] + 1j * c["imag"] for c in b["expected"]],
                       dtype=complex)
    except (OSError, ValueError, KeyError) as e:
        res["error"] = f"cannot check bundle: {e}"
        return res
    if exp.shape != states[-1].shape:
        res["error"] = (f"expected has {exp.size} amplitudes, circuit has "
                        f"{states[-1].size}")
        return res
    return finish(b.get("name", os.path.basename(path)), n, exp, states[-1],
                  tol, {"path": path, "n_ops": len(ops),
                        "notes": b.get("notes", []), "ops": b["ops"],
                        "qubits": b.get("qubits", [])})


def check_qsim(path, tol):
    res = {"path": path, "tol": tol, "status": "ERROR"}
    name = os.path.splitext(os.path.basename(path))[0]
    try:
        circ = extract_qsim(path, name)
    except QsimError as e:
        res["error"] = str(e)
        return res
    n = circ["n_qubits"]
    core = to_checker_ops(circ["ops"], circ["qindex"])
    inits = init_ops(circ["inits"])
    ops = inits + core
    try:
        states = simulate(n, ops)
    except ValueError as e:
        res["error"] = f"cannot simulate: {e}"
        return res
    sim_steps = [s for s in circ["steps"][1:] if not s.get("noop")]
    if len(sim_steps) != len(core):
        return {**res, "error": "internal step/op mismatch"}
    for op, step, got in zip(core, sim_steps, states[len(inits):]):
        exp = np.asarray(step["state"], dtype=complex)
        norm = float(np.vdot(exp, exp).real)
        if norm == 0:
            return {**res, "error": f"step {op['label']!r} replayed a zero vector"}
        exp = exp / norm**0.5
        fid = abs(complex(np.vdot(exp, got))) ** 2
        step["gate"] = op["gate"]
        step["qubits"] = list(op["qubits"])
        step["fidelity"] = float(fid)
        step["maxerr"] = float(np.max(np.abs(exp - got)))
    disp_steps = circ["steps"][1:]
    final = finish(name, n, np.asarray(circ["expected"], dtype=complex),
                   states[-1] if states else np.zeros(1 << n), tol,
                   {"path": path, "n_ops": len(ops), "notes": circ["notes"],
                    "steps": disp_steps,
                    "ops": [{"gate": o["gate"], "qubits": o["qubits"],
                             "label": o["label"]} for o in ops],
                    "qubits": [{"qasm": k, "modifier": m,
                                "init": [{"real": a.real, "imag": a.imag},
                                         {"real": b.real, "imag": b.imag}]}
                               for k, (m, (a, b)) in
                               enumerate(zip(circ["modifiers"], circ["inits"]))]})
    return final


def fmt_c(z):
    return f"{z.real:+.6f}{z.imag:+.6f}j"


def print_vectors(out, res):
    """Print both final state vectors: expected (Qsim) vs Qiskit."""
    out.append(f"final statevectors for {res['name']} "
               f"(Qiskit order, qubit 0 = LSB):")
    out.append(f"{'i':>5}  {'bits':<8} {'expected':<22} {'qiskit':<22} "
               f"{'|err|':>10}")
    out.append(f"{'---':>5}  {'--------':<8} {'----------------------':<22} "
               f"{'----------------------':<22} {'----------':>10}")
    rows = res["rows"]
    if len(rows) > 64:
        shown = sorted(rows, key=lambda r: -abs(r["exp"]))[:48]
        shown += [r for r in sorted(rows, key=lambda r: -r["err"])[:16]
                  if r not in shown]
        shown.sort(key=lambda r: r["i"])
        out.append(f"(showing {len(shown)} of {len(rows)} amplitudes)")
    else:
        shown = sorted(rows, key=lambda r: r["i"])
    for r in shown:
        mark = " *" if r == res["worst"] and r["err"] > 0 else ""
        out.append(f"{r['i']:>5}  {r['bits']:<8} {fmt_c(r['exp']):<22} "
                   f"{fmt_c(r['got']):<22} {r['err']:>10.3g}{mark}")


def print_table(out, results):
    cols = ["bundle", "qubits", "ops", "fidelity", "infidelity", "result"]
    rows = []
    for r in results:
        if r["status"] == "ERROR":
            rows.append([str(r.get("name", r["path"])), "-", "-",
                         "-", "-", f"ERROR: {r.get('error', '?')}"])
        else:
            rows.append([str(r["name"]), str(r["n_qubits"]), str(r.get("n_ops", "?")),
                         f"{r['fidelity']:.10f}", f"{r['infidelity']:.3g}",
                         r["status"]])
    widths = [max(len(row[i]) for row in rows + [cols]) for i in range(6)]
    out.append("  ".join(c.ljust(widths[i]) for i, c in enumerate(cols)))
    out.append("  ".join("-" * widths[i] for i in range(6)))
    for row in rows:
        out.append("  ".join(row[i].ljust(widths[i]) for i in range(6)))


def print_steps(out, res):
    out.append(f"step log for {res['name']}: replay state vs Qiskit after each op")
    out.append(f"{'#':>3}  {'op':<12} {'gate':<8} {'qubits':<12} "
               f"{'fidelity':>14}  {'max|err|':>10}")
    out.append(f"{'---':>3}  {'------------':<12} {'--------':<8} {'------------':<12} "
               f"{'--------------':>14}  {'----------':>10}")
    for k, s in enumerate(res["steps"]):
        if "fidelity" in s:
            out.append(f"{k:>3}  {s['label']:<12} {s['gate']:<8} "
                       f"{str(s['qubits']):<12} {s['fidelity']:>14.10f}  "
                       f"{s['maxerr']:>10.3g}")
        else:
            p = ""
            if "p0" in s:
                p = f"p0={s['p0']:.4f}"
            out.append(f"{k:>3}  {s['label']:<12} {'measure':<8} "
                       f"{'':<12} {'(compiled away)':>14}  {p:>10}")


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("target", help=".verify.json bundle, .qsim save, or a directory")
    ap.add_argument("--tol", type=float, default=1e-6,
                    help="max allowed infidelity (default 1e-6)")
    ap.add_argument("--log", metavar="FILE",
                    help="write the text log to FILE as well as stdout")
    args = ap.parse_args()

    if os.path.isdir(args.target):
        paths = sorted(os.path.join(args.target, f) for f in os.listdir(args.target)
                       if f.endswith((".verify.json", ".qsim")))
        if not paths:
            sys.stderr.write(f"error: no bundles or saves in {args.target}\n")
            return 2
        batch = True
    else:
        paths, batch = [args.target], False

    def is_qsim(path):
        if path.endswith(".qsim"):
            return True
        if path.endswith(".json"):
            return False
        try:  # extensionless saves (e.g. saves/Telepo): sniff content
            with open(path, encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    w = json.loads(line)
                    if isinstance(w, dict) and w.get("type") == "RenderWindow":
                        return True
                    return False
        except (OSError, ValueError):
            return False

    def check(p):
        if is_qsim(p):
            return check_qsim(p, args.tol)
        return check_bundle(p, args.tol)

    results = [check(p) for p in paths]
    out = []

    def show(r):
        if r.get("steps"):
            print_steps(out, r)
        out.append(f"{r['name']}: fidelity={r['fidelity']:.12f} "
                   f"infidelity={r['infidelity']:.3g} (tol {args.tol:g}) "
                   f"-> {r['status']}  (worst |err| {r['worst']['err']:.3g} "
                   f"at i={r['worst']['i']})")
        for note in r.get("notes", []):
            out.append(f"  note: {note}")
        print_vectors(out, r)

    if batch:
        print_table(out, results)
        for r in results:
            if r["status"] == "ERROR":
                out.append(f"error: {r.get('name', r['path'])}: "
                           f"{r.get('error', '?')}")
            else:
                out.append("")
                show(r)
    else:
        r = results[0]
        if r["status"] == "ERROR":
            sys.stderr.write(f"error: {r.get('error', '?')}\n")
            return 2
        show(r)

    text = "\n".join(out)
    print(text)
    if args.log:
        with open(args.log, "w", encoding="utf-8") as f:
            f.write(text + "\n")

    if any(r["status"] == "ERROR" for r in results):
        return 2
    return 0 if all(r["status"] == "PASS" for r in results) else 1


if __name__ == "__main__":
    sys.exit(main())
