"""
ハイブリッド量子古典リコメンデーション推論API

リコメンデーションは古典Two-Tower → PCA → PQC の量子強化 embedding のみを使用。
（古典単体・QML単体の並列運用は廃止。詳細は train_qml.py 参照）

エンドポイント:
  GET  /recommendations/{item_id}   量子強化 embedding で FAISS 検索
  POST /recommendations/by-meta     タイトル・価格から BoW 代理検索 → FAISS
  POST /quantum/random              PennyLane QRNG (オークション同額抽選用)
  GET  /health
"""

import re
from collections import Counter
from typing import List, Optional

import numpy as np
import pandas as pd
import faiss
import pennylane as qml
from fastapi import FastAPI, HTTPException, Query
from pydantic import BaseModel

HYBRID_EMBEDDINGS_PATH = "data/qml_embeddings.csv"
SAMPLE_PATH = "data/merrec_sample_100k.parquet"

app = FastAPI(title="Flea Market Recommendation API", version="1.0.0")


# ── pydantic models ──

class SimilarItem(BaseModel):
    item_id: int
    score: float
    name: Optional[str] = None
    category: Optional[str] = None
    price: Optional[float] = None
    is_cold_start: bool

class RecommendationResponse(BaseModel):
    query_item_id: int
    results: List[SimilarItem]

class MetaRecommendationRequest(BaseModel):
    title: str
    price: float
    condition: str
    k: int = 10

class MetaRecommendationResponse(BaseModel):
    results: List[SimilarItem]


# ── 起動時に1回だけ読み込み ──

print("Loading hybrid QML embeddings...")
emb_df = pd.read_csv(HYBRID_EMBEDDINGS_PATH)
item_ids = emb_df["item_id"].to_numpy()
vectors = np.ascontiguousarray(
    emb_df.drop(columns=["item_id", "is_cold_start"]).to_numpy(dtype=np.float32)
)
is_cold = emb_df["is_cold_start"].to_numpy()

faiss.normalize_L2(vectors)
item_id_to_pos = {int(iid): i for i, iid in enumerate(item_ids)}

index = faiss.IndexFlatIP(vectors.shape[1])
index.add(vectors)
print(f"FAISS index built: {index.ntotal:,} vectors, dim={vectors.shape[1]}")
print("  (ハイブリッド量子古典 embedding: 古典Two-Tower → PCA → PQC)")

print("Loading item metadata...")
meta = (
    pd.read_parquet(SAMPLE_PATH, columns=["item_id", "name", "c0_name", "price"])
    .drop_duplicates("item_id")
    .set_index("item_id")
)

print("Building content index for cold-start lookup...")
meta_item_ids = meta.index.to_numpy()
meta_names = meta["name"].fillna("").tolist()
meta_prices_log = np.log1p(meta["price"].fillna(0).to_numpy(dtype=np.float32))


def tokenize(text: str):
    if not isinstance(text, str):
        return []
    return re.findall(r"[a-z0-9]+", text.lower())


token_counter: Counter = Counter()
for name in meta_names:
    token_counter.update(set(tokenize(name)))
top_vocab = [w for w, _ in token_counter.most_common(2000)]
vocab_index = {w: i for i, w in enumerate(top_vocab)}
VOCAB_SIZE = len(top_vocab)


def name_to_bow(text: str) -> np.ndarray:
    vec = np.zeros(VOCAB_SIZE, dtype=np.float32)
    for t in tokenize(text):
        idx = vocab_index.get(t)
        if idx is not None:
            vec[idx] = 1.0
    return vec


def find_closest_meta_item_id(title: str, price: float) -> int:
    """BoW でメタデータから最近傍 item_id を返す。"""
    bow = name_to_bow(title)
    bow_norm = np.linalg.norm(bow)
    if bow_norm > 0:
        bow = bow / bow_norm

    CHUNK = 5000
    best_score = -1.0
    best_pos = 0
    price_log = float(np.log1p(max(price, 0)))
    price_std = float(meta_prices_log.std()) + 1e-6

    for start in range(0, len(meta_names), CHUNK):
        chunk_names  = meta_names[start:start + CHUNK]
        chunk_prices = meta_prices_log[start:start + CHUNK]
        bows  = np.stack([name_to_bow(n) for n in chunk_names])
        norms = np.linalg.norm(bows, axis=1, keepdims=True) + 1e-8
        bows  = bows / norms
        text_sim  = bows @ bow
        price_sim = np.clip(1.0 - np.abs(chunk_prices - price_log) / (price_std * 3 + 1e-8), 0, 1)
        score = text_sim * 0.8 + price_sim * 0.2
        local_best = int(np.argmax(score))
        if score[local_best] > best_score:
            best_score = score[local_best]
            best_pos   = start + local_best

    return int(meta_item_ids[best_pos])


def search_by_vector(query_vec: np.ndarray, k: int, exclude_pos: Optional[int] = None) -> List[SimilarItem]:
    q = np.ascontiguousarray(query_vec)
    faiss.normalize_L2(q)
    scores, indices = index.search(q, k + 1)

    results = []
    for score, idx in zip(scores[0], indices[0]):
        if idx == -1 or idx == exclude_pos:
            continue
        candidate_id = int(item_ids[idx])
        row = meta.loc[candidate_id] if candidate_id in meta.index else None
        results.append(SimilarItem(
            item_id=candidate_id,
            score=float(score),
            name=(str(row["name"]) if row is not None else None),
            category=(str(row["c0_name"]) if row is not None else None),
            price=(float(row["price"]) if row is not None else None),
            is_cold_start=bool(is_cold[idx]),
        ))
        if len(results) >= k:
            break
    return results


print("Ready.")


# ── QRNG (PennyLane) — オークション同額抽選用 ──

_qrng_dev = qml.device("default.qubit", wires=8)

@qml.qnode(_qrng_dev)
def _qrng_circuit(n_qubits: int):
    for i in range(n_qubits):
        qml.Hadamard(wires=i)
    return [qml.sample(qml.PauliZ(i)) for i in range(n_qubits)]

def quantum_randint(low: int, high: int):
    n = high - low
    if n <= 0:
        return low, [], 0, 0
    n_qubits = max(n.bit_length(), 1)
    while True:
        samples = _qrng_circuit(n_qubits)
        bits = [int((1 - int(s)) / 2) for s in samples[:n_qubits]]
        value = int("".join(str(b) for b in bits), 2)
        if value < n:
            return low + value, bits, n_qubits, n_qubits


# ── routes ──

class QuantumRandomRequest(BaseModel):
    low: int = 0
    high: int = 100
    purpose: str = ""

class QuantumRandomResponse(BaseModel):
    value: int
    bits: List[int]
    n_qubits: int
    circuit_depth: int
    purpose: str

@app.post("/quantum/random", response_model=QuantumRandomResponse)
def quantum_random(req: QuantumRandomRequest):
    if req.high <= req.low:
        raise HTTPException(status_code=400, detail="high must be greater than low")
    value, bits, n_qubits, depth = quantum_randint(req.low, req.high)
    return QuantumRandomResponse(
        value=value, bits=bits, n_qubits=n_qubits,
        circuit_depth=depth, purpose=req.purpose,
    )

@app.get("/health")
def health():
    return {
        "status": "ok",
        "n_items": int(index.ntotal),
        "embedding_type": "hybrid_quantum_classical",
        "pipeline": "Two-Tower(32d) → PCA(6d) → PQC(6d)",
    }


@app.get("/recommendations/{item_id}", response_model=RecommendationResponse)
def recommendations(item_id: int, k: int = Query(default=10, ge=1, le=50)):
    pos = item_id_to_pos.get(item_id)
    if pos is None:
        raise HTTPException(status_code=404, detail=f"item_id {item_id} not found")

    results = search_by_vector(vectors[pos: pos + 1], k, exclude_pos=pos)
    return RecommendationResponse(query_item_id=item_id, results=results)


@app.post("/recommendations/by-meta", response_model=MetaRecommendationResponse)
def recommendations_by_meta(req: MetaRecommendationRequest):
    k = max(1, min(req.k, 50))

    # BoW でメタデータから最近傍 item_id を特定 → 量子強化 embedding で検索
    proxy_item_id = find_closest_meta_item_id(req.title, req.price)
    proxy_pos = item_id_to_pos.get(proxy_item_id)
    if proxy_pos is None:
        raise HTTPException(status_code=404, detail="No matching item in hybrid index")

    results = search_by_vector(vectors[proxy_pos: proxy_pos + 1], k, exclude_pos=proxy_pos)
    return MetaRecommendationResponse(results=results)
