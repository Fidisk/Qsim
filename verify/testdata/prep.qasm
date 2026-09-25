OPENQASM 2.0;
include "qelib1.inc";
// verify-bundle: Prep (1 qubits)
qreg q[1];
// verify:prep q[0] = [0.70710677+0i, 0.70710677+0i] (see sidecar)
h q[0]; // H
