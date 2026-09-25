OPENQASM 2.0;
include "qelib1.inc";
// verify-bundle: Superdense (2 qubits)
qreg q[2];
h q[0]; // H
cx q[0],q[1]; // CX
// verify:custom U0 on q[1],q[0] (see sidecar)
// verify:custom U1 on q[1],q[0] (see sidecar)
cx q[0],q[1]; // CX
// verify:custom U2 on q[1],q[0] (see sidecar)
