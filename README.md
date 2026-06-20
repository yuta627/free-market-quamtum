# free-market-quamtum

次世代の技術「量子技術」を用いたフリマアプリ

## デモ

**フロントエンド**: https://free-market-quamtum.vercel.app  
**バックエンドAPI**: https://free-market-backend-1085125624210.asia-northeast1.run.app/api/v1/products

---

## 主な機能

- **商品一覧・検索**: サーバーサイド全文検索（ILIKE）、デバウンス付きリアルタイム検索
- **商品出品**: 画像URL指定、AI自動説明文生成（Gemini API）
- **いいね機能**: オプティミスティックUIによる即時フィードバック
- **購入**: Stripeによるクレジットカード・PayPay決済（PayPay等リダイレクト決済はreturn_urlで処理）
- **オークション**: リアルタイム入札・残り時間表示、出品者による自己入札禁止
- **量子抽選**: オークション終了後、出品者のみが「量子抽選で落札者決定」ボタンを押せる。バックエンドがMLサーバーのQRNG（PennyLane Hadamardゲート）を呼び出し、同額最高入札者の中から落札者を抽選してDBに保存する
- **AIチャット**: 商品ページで質問するとAIが商品情報をもとに回答（Gemini API）
- **量子カーネルレコメンデーション**: 商品詳細ページに「古典（PCA+FAISS）」と「量子カーネル」を切り替えて類似商品を表示
- **アプリ内通知**: 商品が購入されると出品者にアプリ内通知が届く。通知には購入者の配送先住所と配送手順が含まれる
- **配送先住所登録**: マイページで郵便番号・都道府県・市区町村・番地・建物名を登録できる

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
| 決済 | Stripe（カード・PayPay） |
| AI | Gemini API（チャット・説明文生成） |

---

## アプリ概要

以下の２点を意識してアプリを制作しました。

### 1. 実際のフロウを意識した設計・機能

必須条件にない機能でも、実際にフリマアプリを利用する際には必須になるであろう機能をフロウを意識しながら設計、追加しました。
具体的には以下のような必須機能以外の機能を追加しました。
- 決済機能
- 購入した際の通知機能（通知内容には購入者の住所・簡単な配送までの流れを含む）
- いいね機能

### 2. 量子技術を用いる

#### 量子技術とは？
ミクロな世界を記述する最も基本的な学問である量子力学を活用した技術。主なものに新しいコンピュータである量子コンピュータや高感度のセンシング技術などがある。

今回はフリマアプリに実装できる機能に限定しました。具体的には、量子コンピュータ上で実装することで精度や効率を高めることのできる機能、量子の性質を用いた以下のような機能を追加しました。

- 量子カーネル法によるリコメンデーション機能  
  詳細は下記の「レコメンデーション」セクションを参照。
- 量子の偶然性を用いたオークション同額入札時の公平な購入者決定機能

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
              ├── FAISSインデックス（73,696件, 6次元PCAベクトル）
              ├── 量子カーネル回路（PennyLane, 6量子ビット）
              └── QRNG（PennyLane Hadamardゲート）
```

---

## レコメンデーション

商品詳細ページの「あなたへのおすすめ」は、「古典（PCA+FAISS）」と「量子カーネル」の2モードをトグルボタンで切り替えて比較できます。

### パイプライン（共通前処理）

```
MerRecデータセット（100k件）
    │
    ▼
古典 Two-Tower モデル（32次元 embedding）
    │  訓練済みweightはitem_embeddings_v2.csvに保存
    ▼
PCA圧縮（32次元 → 6次元）
    │  説明分散: 74.8%
    │  [-π, π] にスケーリング（AngleEmbedding用）
    ▼
6次元 PCAベクトル（73,696件分をpca_vectors.csvに保存）
```

### 古典モード（PCA + FAISS）

```
PCAベクトル（6次元）
    │
    ▼
FAISSインデックス（IndexFlatIP, L2正規化済み）
    │  クエリ商品と上位k件を内積検索
    ▼
類似商品k件を返却
```

### 量子カーネルモード

```
PCAベクトル（6次元）
    │
    ▼
FAISSで上位50件を候補として高速抽出
    │
    ▼
量子カーネル K(x₁, x₂) = |⟨ψ(x₁)|ψ(x₂)⟩|² を50件全てに計算
    │
    │  特徴マップ ψ(x):
    │    AngleEmbedding(x, rotation="Y")
    │    → CNOTチェーン（量子もつれ生成）
    │    → AngleEmbedding(x, rotation="Z")
    │    → adjoint(ψ(x₂)) との内積を測定
    │
    ▼
量子カーネルスコアで再ランキングして上位k件を返却
```

### 量子カーネル法について

量子カーネル法は、古典的なカーネル法のカーネル関数を量子回路で実装したものです。

- **カーネル値**: K(x₁,x₂) = |⟨ψ(x₁)|ψ(x₂)⟩|² — 量子状態間の内積の二乗
- **特徴空間**: 量子ビット数をnとすると2ⁿ次元のヒルベルト空間（n=6で64次元）
- **FTQC拡張性**: FTQCが実現し量子ビット数が増えると特徴空間が指数関数的に拡大する。現在はn=6だが、NISQ制約が解消されればPCA圧縮なしに32次元以上の入力を直接量子状態へマッピングでき、より豊かな表現が可能になる
- **学習不要**: PQC（Parameterized Quantum Circuit）と異なり、量子カーネル法は回路パラメータの学習が不要。特徴マップを固定した上で類似度を計算するだけでよい

### NISQ制約

- 量子ビット数: 6（シミュレーター）
- 量子カーネル計算: クエリごとに50回の回路実行（FAISS事前フィルタリングで削減）
- 全件推論: 73,696件対応

### QRNG（量子乱数生成）

オークションの同額抽選専用。MLサーバーの `POST /quantum/random` エンドポイントが以下を実行する。

- N量子ビットのHadamardゲートで全重ね合わせ状態を生成
- PauliZ測定でビット列を取得
- 範囲外の値は棄却して再試行（rejection sampling）

バックエンドはQRNGの結果を受け取り、落札者IDをDBに保存する。

---

## 通知

商品が購入されると出品者のアプリ内通知に以下の内容が届く。

- 商品名と金額
- 購入者が住所を登録済みの場合: 配送先（郵便番号・都道府県・市区町村・番地・建物名）と配送手順
- 購入者が住所未登録の場合: メッセージで住所確認を促すメッセージ

通知はDBに保存され、ヘッダーのベルアイコンから確認できる。未読件数はバッジで表示される。

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
| `STRIPE_SECRET_KEY` | Stripe秘密鍵 |
| `GEMINI_API_KEY` | Gemini APIキー（チャット・説明文生成） |

### MLサーバーのembedding再生成

```bash
cd ml
# ハイブリッドQML embeddingとPCAベクトルの生成
pip install scikit-learn torch
python train_qml.py
# → data/qml_embeddings.csv（73,696件）
# → data/pca_vectors.csv（量子カーネル・古典モード用）が生成される
```
