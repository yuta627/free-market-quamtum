"""
Two-Tower モデルの item embedding を使った類似アイテム推薦の推論API。

エンドポイント:
  GET  /recommendations/{item_id}   MerRec item_id でFAISS検索 (既存商品)
  POST /recommendations/by-meta     タイトル・価格でBoW最近傍を探しFAISS検索 (新規商品)
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

EMBEDDINGS_PATH = "data/item_embeddings_v2.csv"
QML_EMBEDDINGS_PATH = "data/qml_embeddings.csv"
SAMPLE_PATH = "data/merrec_sample_100k.parquet"

app = FastAPI(title="Flea Market Recommendation API", version="0.3.0")


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

print("Loading item embeddings...")
emb_df = pd.read_csv(EMBEDDINGS_PATH)
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

print("Loading item metadata...")
meta = (
    pd.read_parquet(SAMPLE_PATH, columns=["item_id", "name", "c0_name", "price"])
    .drop_duplicates("item_id")
    .set_index("item_id")
)

print("Building content index for cold-start lookup...")
meta_names = meta["name"].fillna("").tolist()
meta_prices_log = np.log1p(meta["price"].fillna(0).to_numpy(dtype=np.float32))


def tokenize(text: str):
    if not isinstance(text, str):
        return []
    return re.findall(r"[a-z0-9]+", text.lower())


# 上位2000語の語彙でBoWベクトルを構築
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


def find_closest_meta_pos(title: str, price: float) -> int:
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
        chunk_names = meta_names[start:start + CHUNK]
        chunk_prices = meta_prices_log[start:start + CHUNK]
        bows = np.stack([name_to_bow(n) for n in chunk_names])
        norms = np.linalg.norm(bows, axis=1, keepdims=True) + 1e-8
        bows = bows / norms
        text_sim = bows @ bow
        price_sim = np.clip(1.0 - np.abs(chunk_prices - price_log) / (price_std * 3 + 1e-8), 0, 1)
        score = text_sim * 0.8 + price_sim * 0.2
        local_best = int(np.argmax(score))
        if score[local_best] > best_score:
            best_score = score[local_best]
            best_pos = start + local_best

    return best_pos


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


# ── QML インデックス (学習完了後にロード) ──
qml_index = None
qml_item_ids = None
qml_vectors = None
qml_id_to_pos: dict = {}
# QML アイテムの名前・価格 (BoW フォールバック用)
qml_meta_names: list = []
qml_meta_prices: np.ndarray = np.array([])

try:
    print("Loading QML embeddings...")
    qml_df = pd.read_csv(QML_EMBEDDINGS_PATH)
    qml_item_ids = qml_df["item_id"].to_numpy()
    qml_vectors = np.ascontiguousarray(
        qml_df.drop(columns=["item_id", "is_cold_start"]).to_numpy(dtype=np.float32)
    )
    faiss.normalize_L2(qml_vectors)
    qml_index = faiss.IndexFlatIP(qml_vectors.shape[1])
    qml_index.add(qml_vectors)
    qml_id_to_pos = {int(iid): i for i, iid in enumerate(qml_item_ids)}
    # QML アイテムのメタ情報をキャッシュ
    for iid in qml_item_ids:
        row = meta.loc[int(iid)] if int(iid) in meta.index else None
        qml_meta_names.append(str(row["name"]) if row is not None else "")
        qml_meta_prices.tolist()  # placeholder
    qml_meta_prices = np.log1p(np.array(
        [float(meta.loc[int(iid)]["price"]) if int(iid) in meta.index else 0.0 for iid in qml_item_ids],
        dtype=np.float32
    ))
    print(f"QML FAISS index built: {qml_index.ntotal:,} vectors, dim={qml_vectors.shape[1]}")
except FileNotFoundError:
    print(f"QML embeddings not found ({QML_EMBEDDINGS_PATH}). Run train_qml.py first.")

print("Ready.")

# ── QRNG (PennyLane) ──
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
        "qml_n_items": int(qml_index.ntotal) if qml_index else 0,
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
    proxy_pos = find_closest_meta_pos(req.title, req.price)
    results = search_by_vector(vectors[proxy_pos: proxy_pos + 1], k, exclude_pos=proxy_pos)
    return MetaRecommendationResponse(results=results)


# ── QML エンドポイント ──

def _find_closest_qml_pos(title: str, price: float) -> int:
    """QML インデックス内の 5,000 件から BoW で最近傍を探す。"""
    bow = name_to_bow(title)
    bow_norm = np.linalg.norm(bow)
    if bow_norm > 0:
        bow = bow / bow_norm

    CHUNK = 500
    best_score = -1.0
    best_pos = 0
    price_log = float(np.log1p(max(price, 0)))
    price_std = float(qml_meta_prices.std()) + 1e-6

    for start in range(0, len(qml_meta_names), CHUNK):
        chunk_names = qml_meta_names[start:start + CHUNK]
        chunk_prices = qml_meta_prices[start:start + CHUNK]
        bows = np.stack([name_to_bow(n) for n in chunk_names])
        norms = np.linalg.norm(bows, axis=1, keepdims=True) + 1e-8
        bows = bows / norms
        text_sim = bows @ bow
        price_sim = np.clip(1.0 - np.abs(chunk_prices - price_log) / (price_std * 3 + 1e-8), 0, 1)
        score = text_sim * 0.8 + price_sim * 0.2
        local_best = int(np.argmax(score))
        if score[local_best] > best_score:
            best_score = score[local_best]
            best_pos = start + local_best

    return best_pos


@app.get("/recommendations/qml/{item_id}", response_model=RecommendationResponse)
def recommendations_qml(item_id: int, k: int = Query(default=10, ge=1, le=50)):
    """
    PQC (Parameterized Quantum Circuit) で生成した embedding による推薦。

    item_id が QML インデックスにない場合 (アプリ新規商品 / DB ID):
      BoW でタイトル類似度から QML 内最近傍を代理検索し推薦する。

    NISQ 制約:
      - 6 量子ビット / 5,000 商品のみ対応
      - 古典 Two-Tower より embedding 次元が低い (6 vs 32)

    FTQC 完成後:
      - 64+ 量子ビット / 全商品対応
      - 量子カーネル法で類似度計算を O(log N) に短縮
    """
    if qml_index is None:
        raise HTTPException(status_code=503, detail="QML index not loaded. Run train_qml.py first.")

    pos = qml_id_to_pos.get(item_id)

    # QML インデックスに存在しない場合: メタ情報で代理検索
    if pos is None:
        # まず古典インデックスで item_id を探してタイトル・価格を取得
        classical_pos = item_id_to_pos.get(item_id)
        if classical_pos is not None:
            candidate_id_cl = int(item_ids[classical_pos])
            row_cl = meta.loc[candidate_id_cl] if candidate_id_cl in meta.index else None
            proxy_title = str(row_cl["name"]) if row_cl is not None else ""
            proxy_price = float(row_cl["price"]) if row_cl is not None else 0.0
        else:
            # DB 独自商品: item_id そのものをメタ検索
            row_cl = meta.loc[item_id] if item_id in meta.index else None
            proxy_title = str(row_cl["name"]) if row_cl is not None else ""
            proxy_price = float(row_cl["price"]) if row_cl is not None else 0.0

        pos = _find_closest_qml_pos(proxy_title, proxy_price)

    q = np.ascontiguousarray(qml_vectors[pos: pos + 1])
    faiss.normalize_L2(q)
    scores, indices = qml_index.search(q, k + 1)

    results = []
    for score, idx in zip(scores[0], indices[0]):
        if idx == -1 or idx == pos:
            continue
        candidate_id = int(qml_item_ids[idx])
        row = meta.loc[candidate_id] if candidate_id in meta.index else None
        results.append(SimilarItem(
            item_id=candidate_id,
            score=float(score),
            name=(str(row["name"]) if row is not None else None),
            category=(str(row["c0_name"]) if row is not None else None),
            price=(float(row["price"]) if row is not None else None),
            is_cold_start=False,
        ))
        if len(results) >= k:
            break

    return RecommendationResponse(query_item_id=item_id, results=results)
