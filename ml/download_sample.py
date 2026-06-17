"""
MerRec (mercari-us/merrec) データセットから先頭10万行だけを抽出してローカルに保存するスクリプト。

12.7億行ある全データセットを丸ごと読み込むとメモリがクラッシュするため、
以下の方針でサンプリングする:
  1. データセットは 20230501/000000000000.parquet のように複数ファイルに分割されている。
  2. huggingface_hub.hf_hub_download で「先頭の1ファイルだけ」をダウンロードする
     （全2172ファイルをダウンロードしない）。
  3. pyarrow.parquet.ParquetFile でそのファイルを行グループ単位でストリーミング読みし、
     10万行に達したら打ち切ってparquetとして保存する。
"""

import pyarrow.parquet as pq
import pyarrow as pa
from huggingface_hub import hf_hub_download

REPO_ID = "mercari-us/merrec"
SAMPLE_FILE = "20230501/000000000000.parquet"
SAMPLE_SIZE = 100_000
OUTPUT_PATH = "data/merrec_sample_100k.parquet"


def main() -> None:
    print(f"Downloading single shard: {SAMPLE_FILE} ...")
    local_path = hf_hub_download(repo_id=REPO_ID, filename=SAMPLE_FILE, repo_type="dataset")
    print(f"Downloaded to: {local_path}")

    pf = pq.ParquetFile(local_path)
    print(f"Row groups: {pf.num_row_groups}, total rows in this shard: {pf.metadata.num_rows}")

    collected: list[pa.Table] = []
    rows_so_far = 0

    for batch in pf.iter_batches(batch_size=10_000):
        table = pa.Table.from_batches([batch])
        remaining = SAMPLE_SIZE - rows_so_far
        if table.num_rows > remaining:
            table = table.slice(0, remaining)
        collected.append(table)
        rows_so_far += table.num_rows
        print(f"  collected {rows_so_far:,} / {SAMPLE_SIZE:,} rows")
        if rows_so_far >= SAMPLE_SIZE:
            break

    sample = pa.concat_tables(collected)

    import os
    os.makedirs("data", exist_ok=True)
    pq.write_table(sample, OUTPUT_PATH)
    print(f"\nSaved {sample.num_rows:,} rows to {OUTPUT_PATH}")


if __name__ == "__main__":
    main()
