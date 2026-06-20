"""
ハイブリッド量子古典リコメンデーション推論API

エンドポイント:
  GET  /recommendations/{item_id}          PQC embedding で FAISS 検索
  GET  /recommendations/qkernel/{item_id}  量子カーネル法で再ランキング
  POST /recommendations/by-meta            タイトル・価格から BoW 代理検索 → FAISS
  POST /quantum/random                     PennyLane QRNG (オークション同額抽選用)
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
PCA_VECTORS_PATH       = "data/pca_vectors.csv"
SAMPLE_PATH = "data/merrec_sample_100k.parquet"

N_QUBITS = 6

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


print("Loading PCA vectors for quantum kernel and classical search...")

try:
    pca_df = pd.read_csv(PCA_VECTORS_PATH)
    pca_item_ids = pca_df["item_id"].to_numpy()
    pca_vectors = pca_df.drop(columns=["item_id"]).to_numpy(dtype=np.float32)
    pca_id_to_pos = {int(iid): i for i, iid in enumerate(pca_item_ids)}
    pca_vectors_norm = np.ascontiguousarray(pca_vectors.copy())
    faiss.normalize_L2(pca_vectors_norm)
    classical_index = faiss.IndexFlatIP(pca_vectors_norm.shape[1])
    classical_index.add(pca_vectors_norm)
    print(f"  PCA vectors loaded: {len(pca_item_ids):,} items, dim={pca_vectors.shape[1]}")
    print(f"  Classical FAISS index built: {classical_index.ntotal:,} vectors")
    QK_AVAILABLE = True
except FileNotFoundError:
    print("  pca_vectors.csv not found — /recommendations/classical and /qkernel will be unavailable")
    QK_AVAILABLE = False
    pca_vectors = None
    pca_id_to_pos = {}
    classical_index = None
    pca_vectors_norm = None


# ── 量子カーネル法 ──
# 特徴マップ: AngleEmbedding(Y) → CNOTチェーン → AngleEmbedding(Z)
# カーネル値: K(x1,x2) = |<ψ(x1)|ψ(x2)>|² = |000...0>を測定する確率
# パラメータ学習なし。FTQCで量子ビット数を増やすだけでスケールアップ可能。

_qk_dev = qml.device("default.qubit", wires=N_QUBITS)

def _feature_map(x):
    """IQP風量子特徴マップ。データを量子状態|ψ(x)⟩に変換。"""
    qml.AngleEmbedding(x, wires=range(N_QUBITS), rotation="Y")
    for i in range(N_QUBITS - 1):
        qml.CNOT(wires=[i, i + 1])
    qml.AngleEmbedding(x, wires=range(N_QUBITS), rotation="Z")

@qml.qnode(_qk_dev)
def _kernel_circuit(x1, x2):
    """
    K(x1,x2) = |<ψ(x1)|ψ(x2)>|²
    feature_map(x1)を適用してからadjoint(feature_map)(x2)を適用。
    |000...0>を測定する確率がカーネル値。
    """
    _feature_map(x1)
    qml.adjoint(_feature_map)(x2)
    return qml.probs(wires=range(N_QUBITS))

def quantum_kernel(x1: np.ndarray, x2: np.ndarray) -> float:
    """量子カーネル値を返す。値が1に近いほど量子特徴空間で近い。"""
    return float(_kernel_circuit(x1, x2)[0])


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
        "methods": {
            "pqc": "Two-Tower(32d) → PCA(6d) → PQC変換 → FAISS検索",
            "qkernel": f"Two-Tower(32d) → PCA(6d) → 量子カーネルK(x1,x2)=|<ψ(x1)|ψ(x2)>|² → 再ランキング  [available={QK_AVAILABLE}]",
        },
        "quantum_kernel": {
            "feature_map": "AngleEmbedding(Y) → CNOTチェーン → AngleEmbedding(Z)",
            "n_qubits": N_QUBITS,
            "hilbert_space_dim": 2 ** N_QUBITS,
            "trainable_params": 0,
        },
    }


@app.get("/recommendations/{item_id}", response_model=RecommendationResponse)
def recommendations(item_id: int, k: int = Query(default=10, ge=1, le=50)):
    pos = item_id_to_pos.get(item_id)
    if pos is None:
        raise HTTPException(status_code=404, detail=f"item_id {item_id} not found")

    results = search_by_vector(vectors[pos: pos + 1], k, exclude_pos=pos)
    return RecommendationResponse(query_item_id=item_id, results=results)


@app.get("/recommendations/classical/{item_id}", response_model=RecommendationResponse)
def recommendations_classical(item_id: int, k: int = Query(default=10, ge=1, le=50)):
    """
    古典レコメンデーション。
    PCA圧縮済み6次元ベクトルをそのままFAISSで近傍検索する。
    量子変換なし。量子カーネル法との比較用。
    """
    if not QK_AVAILABLE:
        raise HTTPException(status_code=503, detail="pca_vectors.csv が見つかりません。")

    pca_pos = pca_id_to_pos.get(item_id)
    if pca_pos is None:
        raise HTTPException(status_code=404, detail=f"item_id {item_id} not found")

    q = np.ascontiguousarray(pca_vectors_norm[pca_pos: pca_pos + 1])
    scores, indices = classical_index.search(q, k + 1)

    results = []
    for score, idx in zip(scores[0], indices[0]):
        if idx == -1 or idx == pca_pos:
            continue
        candidate_id = int(pca_item_ids[idx])
        row = meta.loc[candidate_id] if candidate_id in meta.index else None
        pos_in_main = item_id_to_pos.get(candidate_id)
        results.append(SimilarItem(
            item_id=candidate_id,
            score=float(score),
            name=(str(row["name"]) if row is not None else None),
            category=(str(row["c0_name"]) if row is not None else None),
            price=(float(row["price"]) if row is not None else None),
            is_cold_start=bool(is_cold[pos_in_main]) if pos_in_main is not None else False,
        ))
        if len(results) >= k:
            break
    return RecommendationResponse(query_item_id=item_id, results=results)


@app.get("/recommendations/qkernel/{item_id}", response_model=RecommendationResponse)
def recommendations_qkernel(item_id: int, k: int = Query(default=10, ge=1, le=50)):
    """
    量子カーネル法によるレコメンデーション。

    処理:
      1. FAISSでPQC embeddingから候補50件を取得（高速）
      2. 候補それぞれと量子カーネル K(x1,x2)=|<ψ(x1)|ψ(x2)>|² を計算
      3. カーネル値の降順で再ランキングして上位k件を返す

    量子カーネルの特性:
      特徴マップ: AngleEmbedding(Y) → CNOTチェーン → AngleEmbedding(Z)
      カーネル値: 量子特徴空間|ψ(x)⟩での内積の二乗
      FTQCで量子ビット数を増やすと2^n次元の特徴空間になり古典計算不可能な類似度を計算できる
    """
    if not QK_AVAILABLE:
        raise HTTPException(status_code=503, detail="pca_vectors.csv が見つかりません。train_qml.py を再実行してください。")

    pos = item_id_to_pos.get(item_id)
    if pos is None:
        raise HTTPException(status_code=404, detail=f"item_id {item_id} not found")

    pca_pos = pca_id_to_pos.get(item_id)
    if pca_pos is None:
        raise HTTPException(status_code=404, detail=f"item_id {item_id} not found in PCA vectors")

    # Step 1: FAISSで候補50件を取得
    CANDIDATES = 50
    q = np.ascontiguousarray(vectors[pos: pos + 1])
    faiss.normalize_L2(q)
    scores, indices = index.search(q, CANDIDATES + 1)

    candidate_positions = [
        int(idx) for idx in indices[0]
        if idx != -1 and idx != pos
    ][:CANDIDATES]

    # Step 2: 量子カーネルで再ランキング
    query_pca = pca_vectors[pca_pos]
    reranked = []
    for cand_pos in candidate_positions:
        cand_item_id = int(item_ids[cand_pos])
        cand_pca_pos = pca_id_to_pos.get(cand_item_id)
        if cand_pca_pos is None:
            continue
        cand_pca = pca_vectors[cand_pca_pos]
        k_score = quantum_kernel(query_pca, cand_pca)
        reranked.append((cand_pos, k_score))

    reranked.sort(key=lambda x: x[1], reverse=True)

    # Step 3: 上位k件をSimilarItemに変換
    results = []
    for cand_pos, k_score in reranked[:k]:
        candidate_id = int(item_ids[cand_pos])
        row = meta.loc[candidate_id] if candidate_id in meta.index else None
        results.append(SimilarItem(
            item_id=candidate_id,
            score=k_score,
            name=(str(row["name"]) if row is not None else None),
            category=(str(row["c0_name"]) if row is not None else None),
            price=(float(row["price"]) if row is not None else None),
            is_cold_start=bool(is_cold[cand_pos]),
        ))

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
