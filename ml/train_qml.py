"""
QML レコメンデーションエンジン — PQC (Parameterized Quantum Circuit) による Item Tower

【現在の実装 — NISQ 時代の制約】
  ・量子ビット数: 6 qubits (シミュレータ上限)
  ・回路深度: 2 層 (ノイズ・デコヒーレンスを模倣して浅く保つ)
  ・特徴エンコード: Angle Embedding (価格・コンディション・カテゴリ・テキストを回転角に変換)
  ・出力: 各量子ビットの Pauli-Z 期待値 → 6次元 embedding ベクトル
  ・訓練規模: 計算コスト上限のため 5,000 商品 / 3 epoch に制限

【FTQC 完成時に精度が向上する理由】
  (1) 量子ビット数の拡張
      NISQ: ~50 物理量子ビット (実効 6-8 qubits)
      FTQC: 100万+ 論理量子ビット (誤り訂正済み)
      → Hilbert 空間が 2^6=64 次元 → 2^100 次元に拡大。
        古典モデルでは表現不可能な特徴空間で商品間の類似度を計算できる。

  (2) 量子カーネル法の実用化
      NISQ では量子カーネル行列の計算が O(N^2) で非現実的だが、
      FTQC では HHL アルゴリズム (量子線形代数) を使い O(log N) に短縮。
      全 73,000 商品の類似度行列を量子コンピュータ上で一括計算可能になる。

  (3) 回路深度の増加
      NISQ: depth 2 (ゲートエラーが蓄積するため浅さが必須)
      FTQC: depth 1,000+ (誤り訂正により任意深度が実現)
      → より複雑な特徴変換・非線形性の表現が可能。

  (4) 量子データ再アップロード (Data re-uploading) の完全活用
      各回路層で特徴を再入力する手法は理論上万能近似器だが、
      NISQ では層数制限で表現力が低い。FTQC では制限なく重ねられる。

  (5) 量子優位性が見込まれるタスク
      ・高次元スパースデータ (商品メタデータ) のカーネル計算
      ・フラストレーションのある最適化問題 (コールドスタート最適割当)
      ・量子ウォークによる協調フィルタリングのグラフ探索
"""

import re
import warnings
warnings.filterwarnings("ignore")

import numpy as np
import pandas as pd
import torch
import torch.nn as nn
import pennylane as qml

SAMPLE_PATH = "data/merrec_sample_100k.parquet"
QML_EMBEDDINGS_OUT = "data/qml_embeddings.csv"

# ── ハイパーパラメータ ──
# NISQ 制約: 量子ビット 6、回路深度 2 に限定
# FTQC では N_QUBITS=64+、N_LAYERS=20+ が現実的になる
N_QUBITS = 6
N_LAYERS = 2
EMBED_DIM = N_QUBITS        # Pauli-Z 期待値の次元数 = 量子ビット数
BATCH_SIZE = 32
EPOCHS = 3
LR = 5e-3
MAX_ITEMS = 5000            # シミュレーション速度の上限
N_NEGATIVES = 2
SEED = 42

rng = np.random.default_rng(SEED)
torch.manual_seed(SEED)

# ── 量子デバイス ──
# NISQ: default.qubit シミュレータ (CPU)
# FTQC: ibm_torino 等の実機 or 誤り訂正済みデバイスに差し替えるだけでよい
dev = qml.device("default.qubit", wires=N_QUBITS)


@qml.qnode(dev, interface="torch", diff_method="backprop")
def pqc_circuit(inputs, weights):
    """
    Parameterized Quantum Circuit — Item Tower

    [Angle Embedding]  特徴量を回転角として量子状態にエンコード
    [StronglyEntanglingLayers]  学習可能な回転 + CNOT エンタングル
    [Measurement]  各量子ビットの <Z> 期待値を embedding として出力

    FTQC 拡張ポイント:
      - inputs: 6次元 → 64次元 (より多くの特徴を直接エンコード)
      - weights shape: (2, 6, 3) → (20, 64, 3) (深い回路で複雑な変換)
      - qml.AmplitudeEmbedding に切替で 2^N 個の振幅を一度にエンコード可能
    """
    qml.AngleEmbedding(inputs, wires=range(N_QUBITS), rotation="Y")
    qml.StronglyEntanglingLayers(weights, wires=range(N_QUBITS))
    return [qml.expval(qml.PauliZ(i)) for i in range(N_QUBITS)]


class QuantumItemTower(nn.Module):
    """
    PQC ベースの Item Tower。

    NISQ 制約:
      - N_QUBITS=6 なので特徴を 6次元に圧縮してからエンコード
      - 古典的な線形層で前処理し次元を削減 (量子ビット数に合わせる)

    FTQC 時の改善:
      - 前処理層を除去し、生特徴を直接 AmplitudeEmbedding で投入
      - 古典前処理のボトルネックが消えて量子回路が本来の表現力を発揮
    """
    def __init__(self, in_dim: int):
        super().__init__()
        # 古典前処理: 高次元特徴 → N_QUBITS 次元 (FTQC では不要になる)
        self.pre = nn.Sequential(
            nn.Linear(in_dim, 16),
            nn.Tanh(),
            nn.Linear(16, N_QUBITS),
            nn.Tanh(),  # [-1,1] に正規化 → 回転角として適切
        )
        # 学習可能なPQCパラメータ shape: (N_LAYERS, N_QUBITS, 3)
        self.weights = nn.Parameter(
            torch.randn(N_LAYERS, N_QUBITS, 3) * 0.1
        )

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        # x: (B, in_dim)
        x_compressed = self.pre(x) * torch.pi  # 回転角の範囲を拡大
        # バッチ処理: PennyLane は 1サンプルずつ処理
        results = []
        for i in range(x_compressed.shape[0]):
            out = pqc_circuit(x_compressed[i], self.weights)
            results.append(torch.stack(out))
        return torch.stack(results)  # (B, N_QUBITS)


class UserTower(nn.Module):
    """古典的な User Tower (Two-Tower の再利用)"""
    def __init__(self, n_users: int, out_dim: int):
        super().__init__()
        self.emb = nn.Embedding(n_users, out_dim)

    def forward(self, idx: torch.Tensor) -> torch.Tensor:
        return self.emb(idx)


def tokenize(text: str):
    if not isinstance(text, str):
        return []
    return re.findall(r"[a-z0-9]+", text.lower())


def load_data():
    df = pd.read_parquet(SAMPLE_PATH)
    EVENT_WEIGHTS = {
        "item_view": 1.0, "item_like": 3.0, "item_add_to_cart_tap": 5.0,
        "offer_make": 7.0, "buy_start": 8.0, "buy_comp": 10.0,
    }
    df["weight"] = df["event_id"].map(EVENT_WEIGHTS).fillna(1.0)

    # NISQ 制約: MAX_ITEMS 商品に制限 (シミュレーション速度)
    # FTQC: 制限なし。量子計算機上での並列処理で全件対応可能
    all_item_ids = df["item_id"].unique()
    rng.shuffle(all_item_ids)
    subset_ids = set(all_item_ids[:MAX_ITEMS].tolist())
    df = df[df["item_id"].isin(subset_ids)]

    item_meta = df.drop_duplicates("item_id").set_index("item_id")[
        ["name", "price", "c0_id", "c1_id", "item_condition_id"]
    ].copy()

    interactions = df.groupby(["user_id", "item_id"], as_index=False)["weight"].sum()
    return interactions, item_meta


def build_features(item_meta: pd.DataFrame) -> torch.Tensor:
    """
    特徴量エンジニアリング → N_QUBITS 次元圧縮前の特徴ベクトル

    NISQ: 簡略化した特徴 (price, cond, c0, c1, 2つのテキスト特徴)
    FTQC: 全特徴 (category3階層、brand、vocab5000次元) を AmplitudeEmbedding で直接投入
    """
    # 価格の対数正規化
    price_log = np.log1p(item_meta["price"].fillna(0).to_numpy())
    price_norm = (price_log - price_log.mean()) / (price_log.std() + 1e-6)

    # コンディション (0-1 正規化)
    cond = item_meta["item_condition_id"].fillna(3).astype(float).to_numpy()
    cond_norm = (cond - 1) / 4.0

    # カテゴリ (ハッシュして正規化)
    c0 = item_meta["c0_id"].fillna(0).astype(float).to_numpy()
    c0_norm = (c0 % 100) / 100.0
    c1 = item_meta["c1_id"].fillna(0).astype(float).to_numpy()
    c1_norm = (c1 % 100) / 100.0

    # テキスト特徴: 最頻出2語の出現フラグ
    names = item_meta["name"].fillna("").tolist()
    from collections import Counter
    counter: Counter = Counter()
    for n in names:
        counter.update(tokenize(n))
    top2 = [w for w, _ in counter.most_common(2)]

    def has_word(name, word):
        return 1.0 if word in tokenize(name) else 0.0

    t0 = np.array([has_word(n, top2[0]) for n in names]) if len(top2) > 0 else np.zeros(len(names))
    t1 = np.array([has_word(n, top2[1]) for n in names]) if len(top2) > 1 else np.zeros(len(names))

    feat = np.stack([price_norm, cond_norm, c0_norm, c1_norm, t0, t1], axis=1)
    return torch.tensor(feat, dtype=torch.float32)


def make_batches(interactions, all_item_ids, user_id_to_idx, item_id_to_idx, batch_size):
    rows = interactions[["user_id", "item_id"]].values
    indices = rng.permutation(len(rows))
    batches = []
    for start in range(0, len(indices) - batch_size + 1, batch_size):
        idx = indices[start:start + batch_size]
        u_idx = torch.tensor([user_id_to_idx[rows[i, 0]] for i in idx], dtype=torch.long)
        p_idx = torch.tensor([item_id_to_idx[rows[i, 1]] for i in idx], dtype=torch.long)
        n_idx = torch.tensor(
            [[item_id_to_idx[all_item_ids[j]]
              for j in rng.integers(0, len(all_item_ids), N_NEGATIVES)]
             for _ in idx],
            dtype=torch.long,
        )  # (B, N_NEGATIVES)
        batches.append((u_idx, p_idx, n_idx))
    return batches


def main():
    print("Loading data...")
    interactions, item_meta = load_data()
    all_item_ids = item_meta.index.to_numpy()
    print(f"  {len(interactions):,} interactions / {len(item_meta):,} items (NISQ 制約: {MAX_ITEMS:,} 件上限)")

    user_ids = interactions["user_id"].unique()
    user_id_to_idx = {u: i for i, u in enumerate(user_ids)}
    item_id_to_idx = {iid: i for i, iid in enumerate(all_item_ids)}
    n_users = len(user_id_to_idx)

    feat = build_features(item_meta)  # (N_ITEMS, 6)
    in_dim = feat.shape[1]

    print(f"\n== 量子回路構成 ==")
    print(f"  量子ビット数 : {N_QUBITS} (FTQC では 64+ に拡張)")
    print(f"  PQC 層数     : {N_LAYERS} (FTQC では 20+ 層が実現)")
    print(f"  Hilbert 空間 : 2^{N_QUBITS} = {2**N_QUBITS} 次元")
    print(f"  (FTQC 時)   : 2^64 = {2**64:,} 次元 → 古典不可能な特徴空間")
    print(f"  Embedding 次元: {EMBED_DIM}")
    print()

    # 回路の様子を表示
    dummy_in = torch.zeros(N_QUBITS)
    dummy_w = torch.zeros(N_LAYERS, N_QUBITS, 3)
    print(qml.draw(pqc_circuit)(dummy_in, dummy_w))
    print()

    q_item_tower = QuantumItemTower(in_dim)
    user_tower = UserTower(n_users, EMBED_DIM)

    optimizer = torch.optim.Adam(
        list(q_item_tower.parameters()) + list(user_tower.parameters()), lr=LR
    )
    bce = nn.BCEWithLogitsLoss()

    print(f"Training QML Two-Tower ({EPOCHS} epochs, batch={BATCH_SIZE})...")
    print("  ※ 量子回路のシミュレーションは古典学習より大幅に遅くなります")
    print("    FTQC では実機上でこの処理がリアルタイムに完了します\n")

    for epoch in range(1, EPOCHS + 1):
        batches = make_batches(interactions, all_item_ids, user_id_to_idx, item_id_to_idx, BATCH_SIZE)
        total_loss = 0.0
        n_batches = 0
        for user_idx_b, pos_item_idx_b, neg_item_idxs_b in batches:

            u_vec = user_tower(user_idx_b)               # (B, EMBED_DIM)
            pos_feat = feat[pos_item_idx_b]              # (B, in_dim)
            pos_vec = q_item_tower(pos_feat)             # (B, EMBED_DIM) ← 量子回路

            pos_score = (u_vec * pos_vec).sum(dim=1)
            pos_loss = bce(pos_score, torch.ones_like(pos_score))

            neg_flat = neg_item_idxs_b.view(-1)         # (B * N_NEG,)
            neg_feat = feat[neg_flat]
            neg_vec = q_item_tower(neg_feat)             # (B*N_NEG, EMBED_DIM) ← 量子回路
            u_rep = u_vec.repeat_interleave(N_NEGATIVES, dim=0)
            neg_score = (u_rep * neg_vec).sum(dim=1)
            neg_loss = bce(neg_score, torch.zeros_like(neg_score))

            loss = pos_loss + neg_loss
            optimizer.zero_grad()
            loss.backward()
            optimizer.step()

            total_loss += loss.item()
            n_batches += 1

        print(f"  epoch {epoch}/{EPOCHS}  loss={total_loss / n_batches:.4f}")

    # 全アイテムの量子 embedding を生成
    print("\nGenerating QML embeddings for all items...")
    q_item_tower.eval()
    all_vecs = []
    INFER_BATCH = 16
    with torch.no_grad():
        for start in range(0, len(all_item_ids), INFER_BATCH):
            batch_feat = feat[start:start + INFER_BATCH]
            vecs = q_item_tower(batch_feat)
            all_vecs.append(vecs.numpy())
    all_vecs = np.concatenate(all_vecs, axis=0)

    # 埋め込みの品質確認
    norms = np.linalg.norm(all_vecs, axis=1)
    print(f"  embedding norm: mean={norms.mean():.4f} std={norms.std():.4f}")
    print(f"  (古典 Two-Tower と同じ norm スケールに近いほど品質が高い)")

    out_df = pd.DataFrame(all_vecs, columns=[f"dim_{i}" for i in range(EMBED_DIM)])
    out_df.insert(0, "item_id", all_item_ids)
    out_df["is_cold_start"] = 0
    out_df.to_csv(QML_EMBEDDINGS_OUT, index=False)
    print(f"  saved {len(out_df):,} QML embeddings to {QML_EMBEDDINGS_OUT}")

    print("""
== FTQC 完成時の性能向上シナリオ ==

現在 (NISQ):
  ・ 6 量子ビット = 64 次元 Hilbert 空間
  ・ 5,000 商品のみ (シミュレーション限界)
  ・ 古典前処理で次元圧縮が必要 → 情報損失
  ・ 1 epoch に数分 (CPU シミュレーション)

FTQC 完成後:
  ・ 64 量子ビット = 2^64 ≈ 1.8京 次元 Hilbert 空間
  ・ 全 73,696 商品をリアルタイム処理
  ・ AmplitudeEmbedding で全特徴を損失なく量子状態へ投入
  ・ 量子カーネル法 + HHL アルゴリズムで類似度計算が O(log N) に
  ・ 量子誤り訂正により depth 1,000+ の深い回路が実現
  ・ 学習時間: 数分 → 数秒 (量子並列性)
""")


if __name__ == "__main__":
    main()
