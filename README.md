# free-market-quamtum

量子機械学習（QML）を活用したフリマアプリ。古典的なTwo-Towerモデルと量子回路（PQC）によるレコメンデーションエンジンを搭載しています。

## デモ

**フロントエンド**: https://free-market-quamtum.vercel.app  
**バックエンドAPI**: https://fleamarket-backend-1085125624210.asia-northeast1.run.app/api/v1/products

## 主な機能

- **商品一覧・検索**: サーバーサイド全文検索（ILIKE）、デバウンス付きリアルタイム検索
- **商品出品**: 画像アップロード、AI自動説明文生成（Claude API）
- **いいね機能**: オプティミスティックUIによる即時フィードバック
- **オークション**: リアルタイム入札・残り時間表示
- **AIチャット**: 商品への質問をAIが自動回答
- **レコメンデーション**:
  - 古典Two-Towerモデル（FAISS近傍探索、100K商品対応）
  - 量子機械学習（PQC / 6量子ビット、5,000商品対応）
- **量子乱数クーポン**: QRNG（量子乱数生成）によるクーポン割引率の決定

## 技術スタック

| レイヤー | 技術 |
|---|---|
| フロントエンド | React + TypeScript + Vite + CSS Modules |
| バックエンド | Go + Gin + GORM v2 |
| データベース | PostgreSQL（Cloud SQL） |
| MLサーバー | Python + FastAPI + FAISS + PennyLane（QML） |
| インフラ | Cloud Run（GCP）+ Vercel |
| 認証 | JWT |
| 決済 | Stripe |

## アーキテクチャ

```
ブラウザ (Vercel)
    │
    ▼
Go バックエンド (Cloud Run)
    │
    ├─── PostgreSQL (Cloud SQL)
    │
    └─── ML推論サーバー (Cloud Run)
              ├── Two-Tower FAISS インデックス
              └── QML (PennyLane) インデックス
```

## 量子機械学習について

本アプリでは商品レコメンデーションにParameterized Quantum Circuit（PQC）を採用しています。

- **量子ビット数**: 6 qubits
- **対応商品数**: 5,000件（NISQ制約）
- **フレームワーク**: PennyLane
- **特徴**: 古典Two-Towerモデルより低次元（6次元 vs 32次元）の量子埋め込みで類似商品を探索

FTQC（フォールトトレラント量子コンピュータ）実現後は64量子ビット以上・全商品対応・量子カーネル法によるO(log N)類似度計算を想定しています。

## ローカル開発

### 必要環境

- Go 1.25+
- Node.js 18+
- Python 3.10+
- PostgreSQL 15+
- Docker（オプション）

### セットアップ

```bash
# リポジトリのクローン
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
pip install -r requirements.txt
uvicorn serve:app --host 0.0.0.0 --port 8001
```

### 環境変数（バックエンド）

| 変数名 | 説明 |
|---|---|
| `DB_HOST` | PostgreSQLホスト |
| `DB_USER` | DBユーザー名 |
| `DB_PASSWORD` | DBパスワード |
| `DB_NAME` | DB名 |
| `JWT_SECRET` | JWT署名シークレット |
| `ML_SERVER_URL` | MLサーバーURL |
| `STRIPE_SECRET_KEY` | Stripe秘密鍵 |

## データ

MerRec（メルカリ公開データセット）から約4,000件の商品データをシードデータとして使用しています。
