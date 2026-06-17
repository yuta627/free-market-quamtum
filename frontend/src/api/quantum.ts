import client from "./client";

export interface QuantumRandomResult {
  value: number;
  bits: number[];
  n_qubits: number;
  circuit_depth: number;
  purpose: string;
}

export const getQuantumRandom = (low: number, high: number, purpose = "") =>
  client
    .get<QuantumRandomResult>("/quantum/random", { params: { low, high, purpose } })
    .then((r) => r.data);

export interface PriceSuggestResult {
  suggested_price: number;
  candidates: { price: number; score: number }[];
  similar_prices: number[];
  solver: string;
}

export const suggestPrice = (title: string, condition: string) =>
  client
    .post<PriceSuggestResult>("/quantum/price-suggest", { title, condition })
    .then((r) => r.data);
