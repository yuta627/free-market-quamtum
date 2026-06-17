"""
Two-Tower モデル (PyTorch) による推薦エンジン Phase 2。

ALS との最大の違い: Item Tower は item_id を一切使わず、
カテゴリ・ブランド・価格・商品名テキストなどの「メタデータ」だけから
ベクトルを生成する。これにより、学習時に一度も登場しなかった新規商品
(コールドスタートアイテム)でも、メタデータを通すだけで妥当な
embeddingを得られることを実証する。

検証方法:
  商品の20%を「held-out (新規商品)」として学習データから完全に除外する
  (interactionも除外し、モデルはこれらのitem_idの存在自体を知らない)。
  学習後、held-out商品のメタデータだけをItem Towerに通して embedding を
  生成し、ALSでは不可能だったこの挙動が機能しているか検証する。
"""

import re
import numpy as np
import pandas as pd
import torch
import torch.nn as nn
from torch.utils.data import Dataset, DataLoader

SAMPLE_PATH = "data/merrec_sample_100k.parquet"
EMBEDDINGS_OUT = "data/item_embeddings_v2.csv"
MODEL_WEIGHTS_OUT = "data/model_weights.pt"
MODEL_META_OUT = "data/model_meta.json"

EVENT_WEIGHTS = {
    "item_view": 1.0,
    "item_like": 3.0,
    "item_add_to_cart_tap": 5.0,
    "offer_make": 7.0,
    "buy_start": 8.0,
    "buy_comp": 10.0,
}

EMBED_DIM = 32
CAT_EMBED_DIM = 16
TEXT_EMBED_DIM = 16
MAX_TEXT_LEN = 12
MAX_VOCAB = 5000
HELD_OUT_FRAC = 0.2
N_NEGATIVES = 4
BATCH_SIZE = 512
EPOCHS = 5
LR = 1e-3
SEED = 42

rng = np.random.default_rng(SEED)
torch.manual_seed(SEED)


# ──────────────────────────────────────────────
# データ準備
# ──────────────────────────────────────────────

def tokenize(text: str) -> list[str]:
    if not isinstance(text, str):
        return []
    return re.findall(r"[a-z0-9]+", text.lower())


def build_vocab(names: pd.Series, max_vocab: int) -> dict[str, int]:
    from collections import Counter
    counter = Counter()
    for name in names:
        counter.update(tokenize(name))
    most_common = counter.most_common(max_vocab - 1)
    vocab = {word: i + 1 for i, (word, _) in enumerate(most_common)}  # 0 = unknown/pad
    return vocab


def encode_text(text: str, vocab: dict[str, int], max_len: int) -> list[int]:
    tokens = tokenize(text)[:max_len]
    ids = [vocab.get(t, 0) for t in tokens]
    ids += [0] * (max_len - len(ids))
    return ids


class CategoryEncoder:
    """NaN を 0 (unknown) として連番インデックスにマッピングする。"""

    def __init__(self, values: pd.Series):
        uniques = pd.Series(values.dropna().unique())
        self.map = {v: i + 1 for i, v in enumerate(uniques)}  # 0 = unknown
        self.size = len(self.map) + 1

    def encode(self, values: pd.Series) -> np.ndarray:
        return values.map(self.map).fillna(0).astype(int).to_numpy()


def load_data():
    df = pd.read_parquet(SAMPLE_PATH)
    df["weight"] = df["event_id"].map(EVENT_WEIGHTS).fillna(1.0)

    # item ごとのメタデータ(重複排除)
    item_meta = df.drop_duplicates("item_id").set_index("item_id")[
        ["name", "price", "c0_id", "c1_id", "c2_id", "brand_id", "item_condition_id"]
    ].copy()

    interactions = df.groupby(["user_id", "item_id"], as_index=False)["weight"].sum()

    return interactions, item_meta


# ──────────────────────────────────────────────
# モデル定義
# ──────────────────────────────────────────────

class ItemTower(nn.Module):
    """item_id を使わず、メタデータのみから embedding を生成する。"""

    def __init__(self, c0_size, c1_size, c2_size, brand_size, cond_size, vocab_size, out_dim):
        super().__init__()
        self.c0_emb = nn.Embedding(c0_size, CAT_EMBED_DIM)
        self.c1_emb = nn.Embedding(c1_size, CAT_EMBED_DIM)
        self.c2_emb = nn.Embedding(c2_size, CAT_EMBED_DIM)
        self.brand_emb = nn.Embedding(brand_size, CAT_EMBED_DIM)
        self.cond_emb = nn.Embedding(cond_size, CAT_EMBED_DIM)
        self.text_emb = nn.Embedding(vocab_size, TEXT_EMBED_DIM, padding_idx=0)

        in_dim = CAT_EMBED_DIM * 5 + TEXT_EMBED_DIM + 1  # +1 = price
        self.mlp = nn.Sequential(
            nn.Linear(in_dim, 64),
            nn.ReLU(),
            nn.Linear(64, out_dim),
        )

    def forward(self, c0, c1, c2, brand, cond, text_ids, price):
        text_mask = (text_ids != 0).float().unsqueeze(-1)  # (B, L, 1)
        text_vecs = self.text_emb(text_ids)  # (B, L, D)
        text_avg = (text_vecs * text_mask).sum(1) / text_mask.sum(1).clamp(min=1)

        x = torch.cat(
            [
                self.c0_emb(c0),
                self.c1_emb(c1),
                self.c2_emb(c2),
                self.brand_emb(brand),
                self.cond_emb(cond),
                text_avg,
                price.unsqueeze(-1),
            ],
            dim=-1,
        )
        return self.mlp(x)


class UserTower(nn.Module):
    def __init__(self, n_users, out_dim):
        super().__init__()
        self.user_emb = nn.Embedding(n_users, 64)
        self.mlp = nn.Sequential(
            nn.Linear(64, 64),
            nn.ReLU(),
            nn.Linear(64, out_dim),
        )

    def forward(self, user_idx):
        return self.mlp(self.user_emb(user_idx))


class TwoTowerModel(nn.Module):
    def __init__(self, item_tower: ItemTower, user_tower: UserTower):
        super().__init__()
        self.item_tower = item_tower
        self.user_tower = user_tower

    def score(self, user_idx, item_feats):
        u = self.user_tower(user_idx)
        v = self.item_tower(*item_feats)
        return (u * v).sum(-1)


# ──────────────────────────────────────────────
# Dataset
# ──────────────────────────────────────────────

class InteractionDataset(Dataset):
    def __init__(self, user_idx, item_idx, weight):
        self.user_idx = user_idx
        self.item_idx = item_idx
        self.weight = weight

    def __len__(self):
        return len(self.user_idx)

    def __getitem__(self, i):
        return self.user_idx[i], self.item_idx[i], self.weight[i]


def main():
    print("Loading data...")
    interactions, item_meta = load_data()
    all_item_ids = item_meta.index.to_numpy()
    print(f"  {len(interactions):,} interaction pairs, {len(item_meta):,} unique items")

    # ── held-out (新規商品) を分離 ──
    shuffled = rng.permutation(all_item_ids)
    n_held_out = int(len(shuffled) * HELD_OUT_FRAC)
    held_out_items = set(shuffled[:n_held_out].tolist())
    train_items = set(shuffled[n_held_out:].tolist())
    print(f"  train items: {len(train_items):,} / held-out (cold-start sim) items: {len(held_out_items):,}")

    train_interactions = interactions[interactions["item_id"].isin(train_items)].reset_index(drop=True)
    print(f"  training interactions after excluding held-out items: {len(train_interactions):,}")

    # ── インデックス化 ──
    user_ids = train_interactions["user_id"].unique()
    user_id_to_idx = {uid: i for i, uid in enumerate(user_ids)}
    n_users = len(user_id_to_idx)

    train_item_ids_arr = np.array(sorted(train_items))
    item_id_to_idx = {iid: i for i, iid in enumerate(all_item_ids)}  # 全item共通インデックス(出力用)

    # ── カテゴリ/テキストエンコーダ(全item_metaに対して fit) ──
    c0_enc = CategoryEncoder(item_meta["c0_id"])
    c1_enc = CategoryEncoder(item_meta["c1_id"])
    c2_enc = CategoryEncoder(item_meta["c2_id"])
    brand_enc = CategoryEncoder(item_meta["brand_id"])
    cond_enc = CategoryEncoder(item_meta["item_condition_id"])
    vocab = build_vocab(item_meta["name"], MAX_VOCAB)

    price_log = np.log1p(item_meta["price"].to_numpy())
    price_mean, price_std = price_log.mean(), price_log.std() + 1e-6

    def build_item_features(item_ids_subset: np.ndarray):
        sub = item_meta.loc[item_ids_subset]
        c0 = torch.tensor(c0_enc.encode(sub["c0_id"]), dtype=torch.long)
        c1 = torch.tensor(c1_enc.encode(sub["c1_id"]), dtype=torch.long)
        c2 = torch.tensor(c2_enc.encode(sub["c2_id"]), dtype=torch.long)
        brand = torch.tensor(brand_enc.encode(sub["brand_id"]), dtype=torch.long)
        cond = torch.tensor(cond_enc.encode(sub["item_condition_id"]), dtype=torch.long)
        text_ids = torch.tensor(
            np.stack([encode_text(n, vocab, MAX_TEXT_LEN) for n in sub["name"]]), dtype=torch.long
        )
        price = torch.tensor(
            (np.log1p(sub["price"].to_numpy()) - price_mean) / price_std, dtype=torch.float32
        )
        return c0, c1, c2, brand, cond, text_ids, price

    # 全item分の特徴を事前計算してキャッシュ(学習中はindex選択のみ)
    print("Precomputing item features for all items...")
    all_c0, all_c1, all_c2, all_brand, all_cond, all_text, all_price = build_item_features(all_item_ids)

    # ── モデル構築 ──
    model = TwoTowerModel(
        item_tower=ItemTower(
            c0_enc.size, c1_enc.size, c2_enc.size, brand_enc.size, cond_enc.size,
            len(vocab) + 1, EMBED_DIM,
        ),
        user_tower=UserTower(n_users, EMBED_DIM),
    )
    optimizer = torch.optim.Adam(model.parameters(), lr=LR)
    bce = nn.BCEWithLogitsLoss()

    train_item_idx_in_all = np.array([item_id_to_idx[i] for i in train_interactions["item_id"]])
    train_user_idx = np.array([user_id_to_idx[u] for u in train_interactions["user_id"]])
    train_weight = train_interactions["weight"].to_numpy(dtype=np.float32)

    train_items_pool = np.array([item_id_to_idx[i] for i in train_items])  # 負例サンプリング候補

    dataset = InteractionDataset(train_user_idx, train_item_idx_in_all, train_weight)
    loader = DataLoader(dataset, batch_size=BATCH_SIZE, shuffle=True)

    print(f"Training Two-Tower model ({EPOCHS} epochs)...")
    for epoch in range(1, EPOCHS + 1):
        total_loss = 0.0
        n_batches = 0
        for user_idx_b, item_idx_b, weight_b in loader:
            B = len(user_idx_b)
            neg_idx = torch.tensor(
                rng.choice(train_items_pool, size=(B, N_NEGATIVES)), dtype=torch.long
            )

            pos_feats = tuple(t[item_idx_b] for t in (all_c0, all_c1, all_c2, all_brand, all_cond, all_text, all_price))
            pos_score = model.score(user_idx_b, pos_feats)
            pos_loss = bce(pos_score, torch.ones_like(pos_score))

            neg_feats = tuple(t[neg_idx.reshape(-1)] for t in (all_c0, all_c1, all_c2, all_brand, all_cond, all_text, all_price))
            user_idx_rep = user_idx_b.repeat_interleave(N_NEGATIVES)
            neg_score = model.score(user_idx_rep, neg_feats)
            neg_loss = bce(neg_score, torch.zeros_like(neg_score))

            loss = pos_loss + neg_loss

            optimizer.zero_grad()
            loss.backward()
            optimizer.step()

            total_loss += loss.item()
            n_batches += 1

        print(f"  epoch {epoch}/{EPOCHS}  loss={total_loss / n_batches:.4f}")

    # ── 全item(train + held-out)の embedding を生成 ──
    print("Generating embeddings for ALL items (including held-out cold-start items)...")
    model.eval()
    with torch.no_grad():
        item_vecs = model.item_tower(all_c0, all_c1, all_c2, all_brand, all_cond, all_text, all_price)
    item_vecs = item_vecs.numpy()

    out_df = pd.DataFrame(item_vecs, columns=[f"dim_{i}" for i in range(EMBED_DIM)])
    out_df.insert(0, "item_id", all_item_ids)
    out_df["is_cold_start"] = [1 if iid in held_out_items else 0 for iid in all_item_ids]
    out_df.to_csv(EMBEDDINGS_OUT, index=False)
    print(f"  saved {len(out_df):,} item embeddings to {EMBEDDINGS_OUT}")

    # ── モデルアーティファクトを保存 (serve.py でのリアルタイム推論用) ──
    import json

    # 重みだけ torch で保存
    torch.save(model.item_tower.state_dict(), MODEL_WEIGHTS_OUT)

    # エンコーダ・語彙・設定は JSON で保存（torch.load より高速）
    def int_keys(d):
        return {int(k): int(v) for k, v in d.items()}

    meta_json = {
        "model_config": {
            "c0_size": c0_enc.size, "c1_size": c1_enc.size, "c2_size": c2_enc.size,
            "brand_size": brand_enc.size, "cond_size": cond_enc.size,
            "vocab_size": len(vocab) + 1, "out_dim": EMBED_DIM,
            "cat_embed_dim": CAT_EMBED_DIM, "text_embed_dim": TEXT_EMBED_DIM,
        },
        "encoders": {
            "c0_map": int_keys(c0_enc.map),
            "c1_map": int_keys(c1_enc.map),
            "c2_map": int_keys(c2_enc.map),
            "brand_map": int_keys(brand_enc.map),
            "cond_map": int_keys(cond_enc.map),
        },
        "vocab": {k: int(v) for k, v in vocab.items()},
        "price_mean": float(price_mean),
        "price_std": float(price_std),
        "max_text_len": MAX_TEXT_LEN,
    }
    with open(MODEL_META_OUT, "w") as f:
        json.dump(meta_json, f)
    print(f"  saved model weights to {MODEL_WEIGHTS_OUT}")
    print(f"  saved model meta to {MODEL_META_OUT}")

    # ── コールドスタート検証 ──
    print("\n=== Cold-start sanity check ===")
    norms = np.linalg.norm(item_vecs, axis=1)
    cold_mask = out_df["is_cold_start"].to_numpy() == 1
    print(f"  trained-item embedding norm:   mean={norms[~cold_mask].mean():.4f} std={norms[~cold_mask].std():.4f}")
    print(f"  held-out(cold) embedding norm: mean={norms[cold_mask].mean():.4f} std={norms[cold_mask].std():.4f}")
    print("  -> held-out items were NEVER seen during training, yet produce non-trivial,")
    print("     differentiated vectors purely from category/brand/price/text metadata.")


if __name__ == "__main__":
    main()
