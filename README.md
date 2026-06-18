# free-market-quamtum

量子機械学習（QML）を活用したフリマアプリ。

## デモ

**フロントエンド**: https://free-market-quamtum.vercel.app  
**バックエンドAPI**: https://fleamarket-backend-1085125624210.asia-northeast1.run.app/api/v1/products

---

## 主な機能

- **商品一覧・検索**: サーバーサイド全文検索（ILIKE）、デバウンス付きリアルタイム検索
- **商品出品**: 画像URL指定、AI自動説明文生成（Claude API）
- **いいね機能**: オプティミスティックUIによる即時フィードバック
- **購入**: Stripeによるクレジットカード決済
- **オークション**: リアルタイム入札・残り時間表示、出品者による自己入札禁止
- **量子抽選**: オークション終了後、出品者のみが「量子抽選で落札者決定」ボタンを押せる。バックエンドがMLサーバーのQRNG（PennyLane Hadamardゲート）を呼び出し、同額最高入札者の中から落札者を抽選してDBに保存する
- **AIチャット**: 商品ページで質問するとAIが商品情報をもとに回答
- **ハイブリッド量子古典レコメンデーション**: 商品詳細ページに類似商品を表示

---

## 技術スタック

| レイヤー | 技術 |
|---|---|
| フロントエンド | React + TypeScript + Vite + CSS Modules |
| バックエンド | Go + Gin + GORM v2 |
| データベース | PostgreSQL（Cloud SQL, us-central1） |
| MLサーバー | Python + FastAPI + FAISS + PennyLane |
| インフラ | Cloud Run（asia-northeast1）+ Vercel |
| 認証 | JWT |
| 決済 | Stripe |
| AIチャット | Claude API |

---

## アーキテクチャ

```
ブラウザ (Vercel)
    │
    ▼
Go バックエンド (Cloud Run, asia-northeast1)
    │
    ├─── PostgreSQL (Cloud SQL, us-central1)
    │
    └─── ML推論サーバー (Cloud Run, asia-northeast1)
              ├── ハイブリッドFAISSインデックス（73,696件）
              └── QRNG（PennyLane Hadamardゲート）
```

---

## レコメンデーション

商品詳細ページの「あなたへのおすすめ」は以下のパイプラインで生成しています。

```
MerRecデータセット（100k件）
    │
    ▼
古典 Two-Tower モデル（32次元 embedding）
    │  訓練済みweightはitem_embeddings_v2.csvに保存
    ▼
PCA圧縮（32次元 → 6次元）
    │  説明分散: 74.8%
    ▼
PQC（Parameterized Quantum Circuit）
    │  量子ビット数: 6 / 回路深度: 2
    │  AngleEmbedding + StronglyEntanglingLayers
    │  Triplet lossで学習（ポジティブペア: 古典コサイン類似度上位）
    ▼
6次元 ハイブリッドembedding（73,696件分をCSVに保存）
    │
    ▼
FAISSインデックス（IndexFlatIP）で近傍検索
```

### NISQ制約

- 量子ビット数: 6（シミュレーター上限）
- PQC学習サンプル: 500件（ローカルCPUシミュレーションの速度制約）
- 全件推論: 73,696件対応（PCA圧縮済みベクトルをPQCに通すだけのため）

### QRNG（量子乱数生成）

オークションの同額抽選専用。MLサーバーの `POST /quantum/random` エンドポイントが以下を実行する。

- N量子ビットのHadamardゲートで全重ね合わせ状態を生成
- PauliZ測定でビット列を取得
- 範囲外の値は棄却して再試行（rejection sampling）

バックエンドはQRNGの結果を受け取り、落札者IDをDBに保存する。

---

## データ

- **MerRec**（メルカリ公開データセット）から約100,000件をサンプリングしてML学習に使用
- アプリのシードデータには一部の商品情報を使用

---

## ローカル開発

### 必要環境

- Go 1.25+
- Node.js 18+
- Python 3.10+
- PostgreSQL 15+

### セットアップ

```bash
git clone https://github.com/yuta627/free-market-quamtum.git
cd free-market-quamtum

# バックエンド
cd backend
cp .env.example .env  # 環境変数を設定
go run ./cmd/api/main.go

# フロントエンド
cd frontend
npm install
npm run dev

# MLサーバー
cd ml
pip install fastapi uvicorn faiss-cpu numpy pandas pyarrow pennylane
uvicorn serve:app --host 0.0.0.0 --port 8001
```

### バックエンド環境変数

| 変数名 | 説明 |
|---|---|
| `DB_HOST` | PostgreSQLホスト |
| `DB_USER` | DBユーザー名 |
| `DB_PASSWORD` | DBパスワード |
| `DB_NAME` | DB名 |
| `JWT_SECRET` | JWT署名シークレット |
| `RECOMMENDATION_SERVICE_URL` | MLサーバーURL |
| `QRNG_SERVICE_URL` | QRNGエンドポイントのベースURL（MLサーバーと同じ） |
| `STRIPE_SECRET_KEY` | Stripe秘密鍵 |

### MLサーバーのembedding再生成

```bash
cd ml
# 古典Two-Towerの学習（item_embeddings_v2.csvを生成）
# ※学習スクリプトは別途用意が必要

# ハイブリッドQML embeddingの生成
pip install scikit-learn torch
python train_qml.py
# → data/qml_embeddings.csv が生成される（73,696件）
```
