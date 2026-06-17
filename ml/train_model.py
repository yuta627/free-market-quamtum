"""
MerRecサンプルデータを用いた implicit ALS によるベースライン推薦モデルの学習。

行動ログ(event_id)を暗黙的フィードバックの「重み」に変換し、
user×item の疎行列を構築してALSで学習する。
学習後、各 item_id の埋め込みベクトルを item_embeddings.csv に出力する。
"""

import numpy as np
import pandas as pd
import scipy.sparse as sp
from implicit.als import AlternatingLeastSquares

SAMPLE_PATH = "data/merrec_sample_100k.parquet"
EMBEDDINGS_OUT = "data/item_embeddings.csv"
USER_MAP_OUT = "data/user_id_map.csv"

# 行動の種類ごとの重み付け。購入に近い行動ほど強い「好み」の signal として扱う。
EVENT_WEIGHTS = {
    "item_view": 1.0,
    "item_like": 3.0,
    "item_add_to_cart_tap": 5.0,
    "offer_make": 7.0,
    "buy_start": 8.0,
    "buy_comp": 10.0,
}

N_FACTORS = 32
REGULARIZATION = 0.01
ITERATIONS = 20


def load_interactions() -> pd.DataFrame:
    df = pd.read_parquet(SAMPLE_PATH, columns=["user_id", "item_id", "event_id"])
    df["weight"] = df["event_id"].map(EVENT_WEIGHTS).fillna(1.0)
    # 同一 user×item に複数イベントがある場合は重みを合算する
    grouped = df.groupby(["user_id", "item_id"], as_index=False)["weight"].sum()
    return grouped


def build_index_maps(df: pd.DataFrame):
    user_ids = df["user_id"].unique()
    item_ids = df["item_id"].unique()

    user_id_to_idx = {uid: i for i, uid in enumerate(user_ids)}
    item_id_to_idx = {iid: i for i, iid in enumerate(item_ids)}

    return user_id_to_idx, item_id_to_idx


def build_sparse_matrix(df: pd.DataFrame, user_id_to_idx: dict, item_id_to_idx: dict) -> sp.csr_matrix:
    rows = df["user_id"].map(user_id_to_idx).to_numpy()
    cols = df["item_id"].map(item_id_to_idx).to_numpy()
    vals = df["weight"].to_numpy(dtype=np.float32)

    n_users = len(user_id_to_idx)
    n_items = len(item_id_to_idx)

    matrix = sp.csr_matrix((vals, (rows, cols)), shape=(n_users, n_items))
    return matrix


def main() -> None:
    print("Loading interactions...")
    interactions = load_interactions()
    print(f"  {len(interactions):,} unique (user_id, item_id) pairs")

    user_id_to_idx, item_id_to_idx = build_index_maps(interactions)
    n_users, n_items = len(user_id_to_idx), len(item_id_to_idx)
    print(f"  {n_users:,} unique users, {n_items:,} unique items")

    print("Building sparse user-item matrix...")
    user_item_matrix = build_sparse_matrix(interactions, user_id_to_idx, item_id_to_idx)
    density = user_item_matrix.nnz / (n_users * n_items) * 100
    print(f"  matrix shape: {user_item_matrix.shape}, density: {density:.4f}%")

    print(f"Training ALS (factors={N_FACTORS}, iterations={ITERATIONS})...")
    model = AlternatingLeastSquares(
        factors=N_FACTORS,
        regularization=REGULARIZATION,
        iterations=ITERATIONS,
        random_state=42,
    )
    # implicitは「confidence行列」(= alpha * 1)を期待するため、user-item行列をそのまま渡す
    model.fit(user_item_matrix)

    print("Exporting item embeddings...")
    idx_to_item_id = {idx: item_id for item_id, idx in item_id_to_idx.items()}
    item_factors = model.item_factors  # shape: (n_items, N_FACTORS)

    rows = []
    for idx in range(n_items):
        item_id = idx_to_item_id[idx]
        vec = item_factors[idx]
        rows.append([item_id] + vec.tolist())

    columns = ["item_id"] + [f"dim_{i}" for i in range(N_FACTORS)]
    emb_df = pd.DataFrame(rows, columns=columns)
    emb_df.to_csv(EMBEDDINGS_OUT, index=False)
    print(f"  saved {len(emb_df):,} item embeddings to {EMBEDDINGS_OUT}")

    # user_id <-> index のマッピングも残しておく(推論時に再利用するため)
    user_map_df = pd.DataFrame(
        {"user_id": list(user_id_to_idx.keys()), "user_idx": list(user_id_to_idx.values())}
    )
    user_map_df.to_csv(USER_MAP_OUT, index=False)
    print(f"  saved user id map to {USER_MAP_OUT}")

    print("\nDone.")


if __name__ == "__main__":
    main()
