import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  getClassicalRecommendations,
  getQKernelRecommendations,
  type RecommendedItem,
} from "../api/recommendations";
import styles from "./RecommendationSection.module.css";

interface RecommendationSectionProps {
  productId: number;
}

type Mode = "classical" | "qkernel";

export default function RecommendationSection({ productId }: RecommendationSectionProps) {
  const [mode, setMode] = useState<Mode>("classical");
  const [items, setItems] = useState<RecommendedItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    setItems([]);
    const fetch = mode === "qkernel" ? getQKernelRecommendations : getClassicalRecommendations;
    fetch(productId, 8)
      .then(setItems)
      .catch(() => setItems([]))
      .finally(() => setLoading(false));
  }, [productId, mode]);

  return (
    <div className={styles.section}>
      <h3>あなたへのおすすめ</h3>
      <div className={styles.toggleRow}>
        <button
          className={`${styles.toggleBtn} ${mode === "classical" ? styles.toggleActive : ""}`}
          onClick={() => setMode("classical")}
        >
          古典（PCA+FAISS）
        </button>
        <button
          className={`${styles.toggleBtn} ${mode === "qkernel" ? styles.toggleActiveQml : ""}`}
          onClick={() => setMode("qkernel")}
        >
          量子カーネル
        </button>
      </div>
      {loading ? (
        <p className={styles.status}>
          {mode === "qkernel" ? "量子カーネル計算中（数秒かかる場合があります）..." : "読み込み中..."}
        </p>
      ) : items.length === 0 ? (
        <p className={styles.status}>おすすめなし</p>
      ) : (
        <>
          {mode === "qkernel" && (
            <p className={styles.hint}>量子カーネル法 K(x₁,x₂)=|⟨ψ(x₁)|ψ(x₂)⟩|² による推薦</p>
          )}
          <div className={styles.scrollRow}>
            {items.map((item) => (
              <Link
                key={item.item_id}
                to={`/products/${item.item_id}`}
                className={`${styles.card} ${mode === "qkernel" ? styles.qmlCard : ""}`}
              >
                <div className={styles.imagePlaceholder}>{item.category}</div>
                <p className={styles.name}>{item.name}</p>
                <p className={styles.price}>¥{Math.round(item.price).toLocaleString()}</p>
                {mode === "qkernel" && <span className={styles.qmlBadge}>量子</span>}
                {item.is_cold_start && <span className={styles.coldBadge}>NEW</span>}
              </Link>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
