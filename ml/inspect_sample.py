"""抽出したサンプルデータのカラム構成・型・統計情報を確認するスクリプト。"""

import pandas as pd

SAMPLE_PATH = "data/merrec_sample_100k.parquet"


def main() -> None:
    df = pd.read_parquet(SAMPLE_PATH)

    print("=" * 60)
    print(f"shape: {df.shape[0]:,} rows x {df.shape[1]} columns")
    print("=" * 60)

    print("\n--- columns & dtypes ---")
    print(df.dtypes)

    print("\n--- head(5) ---")
    with pd.option_context("display.max_columns", None, "display.width", 200):
        print(df.head(5))

    print("\n--- null counts ---")
    print(df.isnull().sum())

    print("\n--- nunique per column ---")
    print(df.nunique())


if __name__ == "__main__":
    main()
