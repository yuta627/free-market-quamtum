"""
MerRec の全商品 (73,696件) を PostgreSQL の products テーブルに投入する。
item_id を product_id として使うことで、ML推薦結果と直接対応させる。
"""
import os
import sys
import psycopg2
import pandas as pd
import numpy as np

DB_CONFIG = {
    "host":     os.getenv("DB_HOST", "localhost"),
    "port":     int(os.getenv("DB_PORT", "5432")),
    "user":     os.getenv("DB_USER", "postgres"),
    "password": os.getenv("DB_PASSWORD", "password"),
    "dbname":   os.getenv("DB_NAME", "fleamarket"),
}

CONDITION_MAP = {
    "New":      "new",
    "Like new": "like_new",
    "Good":     "good",
    "Fair":     "fair",
    "Poor":     "poor",
}

def main():
    print("Loading MerRec parquet...")
    df = pd.read_parquet("data/merrec_sample_100k.parquet")
    items = (
        df[["item_id", "name", "price", "c0_name", "item_condition_name"]]
        .drop_duplicates("item_id")
        .reset_index(drop=True)
    )
    print(f"  {len(items):,} unique items loaded")

    conn = psycopg2.connect(**DB_CONFIG)
    conn.autocommit = False
    cur = conn.cursor()

    # seller_id=1 (出品者テスト) に全商品を紐づける
    cur.execute("SELECT id FROM users LIMIT 1")
    row = cur.fetchone()
    if row is None:
        print("ERROR: users テーブルにユーザーが存在しません。先にGoバックエンドを起動してください。")
        sys.exit(1)
    seller_id = row[0]
    print(f"  Using seller_id={seller_id}")

    # 既存MerRec商品をスキップするため既存IDを取得
    cur.execute("SELECT id FROM products WHERE id > 1000")
    existing = {row[0] for row in cur.fetchall()}
    print(f"  Already in DB: {len(existing):,} items")

    batch = []
    skipped = 0
    BATCH_SIZE = 2000

    for _, row in items.iterrows():
        item_id = int(row["item_id"])
        if item_id in existing:
            skipped += 1
            continue

        name  = str(row["name"])[:200]
        price = max(0, int(row["price"]))
        cond  = CONDITION_MAP.get(str(row["item_condition_name"]), "good")
        desc  = f"{row['c0_name'] or ''} — MerRecデータセット商品".strip()

        batch.append((item_id, seller_id, name, desc, price, cond))

        if len(batch) >= BATCH_SIZE:
            _insert_batch(cur, batch)
            conn.commit()
            total_done = len(existing) + skipped + BATCH_SIZE
            print(f"  inserted up to {total_done:,}...", flush=True)
            batch = []

    if batch:
        _insert_batch(cur, batch)
        conn.commit()

    # シーケンスを最大IDより大きい値に更新（新規商品の自動採番が衝突しないよう）
    cur.execute("SELECT MAX(id) FROM products")
    max_id = cur.fetchone()[0] or 0
    cur.execute(f"SELECT setval(pg_get_serial_sequence('products','id'), {max_id + 1})")
    conn.commit()

    cur.close()
    conn.close()
    print(f"\nDone. skipped={skipped:,}, inserted={len(items)-skipped:,}")
    print(f"Sequence reset to {max_id + 2}")


def _insert_batch(cur, batch):
    cur.executemany(
        """
        INSERT INTO products (id, seller_id, title, description, price, condition, status, image_urls)
        VALUES (%s, %s, %s, %s, %s, %s, 'on_sale', '[]')
        ON CONFLICT (id) DO NOTHING
        """,
        batch,
    )


if __name__ == "__main__":
    main()
