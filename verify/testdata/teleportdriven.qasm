OPENQASM 2.0;
include "qelib1.inc";
// verify-bundle: TeleportDriven (2 qubits)
qreg q[2];
h q[0]; // H
cx q[0],q[1]; // CX
cx q[0],q[1]; // cX
