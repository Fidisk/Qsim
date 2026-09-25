OPENQASM 2.0;
include "qelib1.inc";
// verify-bundle: Custom (2 qubits)
qreg q[2];
x q[1]; // X
// verify:custom U0 on q[1],q[0] (see sidecar)
