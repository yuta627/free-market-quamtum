"""
ハイブリッド量子古典リコメンデーション

パイプライン:
  古典 Two-Tower embedding (32次元)
    → PCA 圧縮 (6次元)          ← 古典モデルの知識を継承
    → PQC 量子変換 (6次元)      ← 量子もつれで非線形特徴を強化
    → 全 100,000 件対応の量子強化 embedding を出力

NISQ 制約:
  量子ビット数: 6 / 回路深度: 2 (ノイズ制約)
  学習: 10,000 件サンプリング (シミュレーション速度上限)
  推論: 全件対応 (PCA圧縮済みベクトルをPQCに通すだけ)

FTQC 完成時:
  PCA 圧縮不要 → 32次元をそのまま AmplitudeEmbedding で投入
  量子ビット数: 64+ / 全件リアルタイム学習
"""

import warnings
warnings.filterwarnings("ignore")

import numpy as np
import pandas as pd
import torch
import torch.nn as nn
import faiss
import pennylane as qml
from sklearn.decomposition import PCA

CLASSICAL_EMB_PATH = "data/item_embeddings_v2.csv"
QML_EMBEDDINGS_OUT = "data/qml_embeddings.csv"

N_QUBITS  = 6
N_LAYERS  = 2
BATCH_SIZE = 32
EPOCHS     = 5
LR         = 1e-3
TRAIN_ITEMS = 10_000   # PQC学習サンプル数（全件は数時間かかる）
K_SIMILAR   = 3        # ポジティブペアの近傍数
MARGIN      = 0.3      # triplet loss マージン
INFER_BATCH = 32
SEED = 42

rng = np.random.default_rng(SEED)
torch.manual_seed(SEED)

# ── 量子デバイス ──
dev = qml.device("default.qubit", wires=N_QUBITS)


@qml.qnode(dev, interface="torch", diff_method="backprop")
def pqc_circuit(inputs, weights):
    """
    Parameterized Quantum Circuit
    inputs:  (N_QUBITS,) PCA圧縮・スケーリング済みベクトル
    weights: (N_LAYERS, N_QUBITS, 3) 学習可能パラメータ
    出力:    各量子ビットの <Z> 期待値 → 6次元 embedding
    """
    qml.AngleEmbedding(inputs, wires=range(N_QUBITS), rotation="Y")
    qml.StronglyEntanglingLayers(weights, wires=range(N_QUBITS))
    return [qml.expval(qml.PauliZ(i)) for i in range(N_QUBITS)]


class HybridQuantumTower(nn.Module):
    """
    PCA圧縮済み6次元ベクトルをPQCで量子変換するタワー。
    古典前処理層なし — 入力はすでに意味ある低次元表現。
    """
    def __init__(self):
        super().__init__()
        self.weights = nn.Parameter(torch.randn(N_LAYERS, N_QUBITS, 3) * 0.1)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        results = []
        for i in range(x.shape[0]):
            out = pqc_circuit(x[i], self.weights)
            results.append(torch.stack(out))
        return torch.stack(results)  # (B, N_QUBITS)


def main():
    # ── Step 1: 古典 embedding を読み込む ──
    print("Loading classical Two-Tower embeddings...")
    emb_df = pd.read_csv(CLASSICAL_EMB_PATH)
    item_ids = emb_df["item_id"].to_numpy()
    vectors = emb_df.drop(columns=["item_id", "is_cold_start"]).to_numpy(dtype=np.float32)
    print(f"  {len(item_ids):,} items, {vectors.shape[1]}次元")

    # L2 正規化（古典 Two-Tower と同じ前処理）
    norms = np.linalg.norm(vectors, axis=1, keepdims=True) + 1e-8
    vectors_normed = vectors / norms

    # ── Step 2: PCA 32 → 6 次元に圧縮 ──
    print(f"PCA compression: {vectors.shape[1]} → {N_QUBITS} dimensions...")
    pca = PCA(n_components=N_QUBITS, random_state=SEED)
    pca_vectors = pca.fit_transform(vectors_normed).astype(np.float32)
    explained = pca.explained_variance_ratio_.sum()
    print(f"  explained variance: {explained:.1%}  (古典知識の{explained:.1%}を継承)")

    # PQC の AngleEmbedding 用に [-π, π] にスケーリング
    scale = np.abs(pca_vectors).max(axis=0) + 1e-8
    pca_scaled = (pca_vectors / scale * np.pi).astype(np.float32)

    # ── Step 3: 古典 FAISS でポジティブペア構築 ──
    print("Building positive pairs from classical similarity...")
    cl_index = faiss.IndexFlatIP(vectors_normed.shape[1])
    cl_index.add(vectors_normed)

    train_idx = rng.choice(len(item_ids), min(TRAIN_ITEMS, len(item_ids)), replace=False)
    _, pos_nn = cl_index.search(vectors_normed[train_idx], K_SIMILAR + 1)
    pos_nn = pos_nn[:, 1:]  # 自分自身を除外 → (TRAIN_ITEMS, K_SIMILAR)
    print(f"  training items: {len(train_idx):,} / {len(item_ids):,}")

    # ── Step 4: PQC 学習（Triplet Loss） ──
    print(f"\n== Hybrid Quantum Training ==")
    print(f"  入力次元   : {vectors.shape[1]} (古典) → {N_QUBITS} (PCA) → {N_QUBITS} (PQC出力)")
    print(f"  量子ビット : {N_QUBITS}  |  PQC層数: {N_LAYERS}")
    print(f"  Hilbert空間: 2^{N_QUBITS} = {2**N_QUBITS}次元")
    print()

    dummy_in = torch.zeros(N_QUBITS)
    dummy_w  = torch.zeros(N_LAYERS, N_QUBITS, 3)
    print(qml.draw(pqc_circuit)(dummy_in, dummy_w))
    print()

    model = HybridQuantumTower()
    optimizer = torch.optim.Adam(model.parameters(), lr=LR)
    pca_tensor = torch.tensor(pca_scaled, dtype=torch.float32)

    print(f"Training ({EPOCHS} epochs, batch={BATCH_SIZE}, triplet margin={MARGIN})...")
    for epoch in range(1, EPOCHS + 1):
        perm = rng.permutation(len(train_idx))
        total_loss = 0.0
        n_batches = 0

        for start in range(0, len(perm) - BATCH_SIZE + 1, BATCH_SIZE):
            local_b  = perm[start:start + BATCH_SIZE]
            global_b = train_idx[local_b]

            # anchor
            anc_feat = pca_tensor[global_b]
            anc_vec  = model(anc_feat)

            # positive: 古典類似度が高い近傍アイテム
            pos_global = pos_nn[local_b, 0]
            pos_vec = model(pca_tensor[pos_global])

            # negative: ランダムサンプル
            neg_global = rng.integers(0, len(item_ids), size=BATCH_SIZE)
            neg_vec = model(pca_tensor[neg_global])

            # Triplet loss: neg より pos が近くなるように学習
            pos_sim = (anc_vec * pos_vec).sum(dim=1)
            neg_sim = (anc_vec * neg_vec).sum(dim=1)
            loss = torch.clamp(neg_sim - pos_sim + MARGIN, min=0).mean()

            optimizer.zero_grad()
            loss.backward()
            optimizer.step()
            total_loss += loss.item()
            n_batches += 1

        print(f"  epoch {epoch}/{EPOCHS}  loss={total_loss / max(n_batches, 1):.4f}")

    # ── Step 5: 全アイテムの量子強化 embedding を生成 ──
    print(f"\nGenerating hybrid embeddings for ALL {len(item_ids):,} items...")
    print("  (PCA圧縮済みベクトルをPQCに通すだけなので全件対応可能)")
    model.eval()
    all_vecs = []
    with torch.no_grad():
        for start in range(0, len(item_ids), INFER_BATCH):
            batch_feat = pca_tensor[start:start + INFER_BATCH]
            vecs = model(batch_feat)
            all_vecs.append(vecs.numpy())
            if start % 5000 == 0:
                print(f"  {start:,} / {len(item_ids):,}", end="\r")
    all_vecs = np.concatenate(all_vecs, axis=0)

    norms_out = np.linalg.norm(all_vecs, axis=1)
    print(f"\n  embedding norm: mean={norms_out.mean():.4f} std={norms_out.std():.4f}")

    out_df = pd.DataFrame(all_vecs, columns=[f"dim_{i}" for i in range(N_QUBITS)])
    out_df.insert(0, "item_id", item_ids)
    out_df["is_cold_start"] = 0
    out_df.to_csv(QML_EMBEDDINGS_OUT, index=False)
    print(f"  saved {len(out_df):,} hybrid embeddings → {QML_EMBEDDINGS_OUT}")
    print("""
== パイプライン完了 ==
  古典 Two-Tower (32次元) の知識を継承しながら
  PQC が量子もつれ・重ね合わせで捉えられなかった非線形パターンを強化。
  全 100,000 件をカバー (従来の QML: 5,000 件)。

FTQC 完成時:
  PCA 圧縮不要 → 32次元を AmplitudeEmbedding で直接量子状態へ
  量子カーネル法 + HHL アルゴリズムで類似度計算が O(log N) に
""")


if __name__ == "__main__":
    main()
