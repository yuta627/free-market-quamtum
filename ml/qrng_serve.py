"""
QRNG (Quantum Random Number Generator) API

Hadamardゲートで量子ビットを重ね合わせ状態にし、
測定による波束収縮で真の乱数を生成する。
古典的な擬似乱数と異なり、原理的に予測・再現不可能。
"""

from typing import List

from fastapi import FastAPI
from pydantic import BaseModel
from qiskit import QuantumCircuit
from qiskit_aer import AerSimulator

app = FastAPI(title="QRNG API", version="1.0.0")
simulator = AerSimulator()


class RandomRequest(BaseModel):
    low: int = 0
    high: int = 100
    purpose: str = ""


class RandomResponse(BaseModel):
    value: int
    bits: List[int]
    n_qubits: int
    circuit_depth: int
    purpose: str


def quantum_randint(low: int, high: int):
    """rejection sampling で [low, high) の均一分布を保証"""
    n = high - low
    if n <= 0:
        return low, [], 0, 0
    n_qubits = max(n.bit_length(), 1)

    qc = QuantumCircuit(n_qubits, n_qubits)
    for i in range(n_qubits):
        qc.h(i)
    qc.measure(list(range(n_qubits)), list(range(n_qubits)))

    depth = qc.depth()

    while True:
        result = simulator.run(qc, shots=1).result()
        bits_str = list(result.get_counts().keys())[0]
        bits = [int(b) for b in bits_str]
        value = int(bits_str, 2)
        if value < n:
            return low + value, bits, n_qubits, depth


@app.get("/health")
def health():
    return {"status": "ok", "backend": "AerSimulator"}


@app.post("/quantum/random", response_model=RandomResponse)
def quantum_random(req: RandomRequest):
    value, bits, n_qubits, depth = quantum_randint(req.low, req.high)
    return RandomResponse(
        value=value,
        bits=bits,
        n_qubits=n_qubits,
        circuit_depth=depth,
        purpose=req.purpose,
    )
