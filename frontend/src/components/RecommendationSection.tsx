import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { getRecommendations, getQMLRecommendations, type RecommendedItem } from "../api/recommendations";
import styles from "./RecommendationSection.module.css";

interface RecommendationSectionProps {
  productId: number;
}

export default function RecommendationSection({ productId }: RecommendationSectionProps) {
  const [items, setItems] = useState<RecommendedItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [useQML, setUseQML] = useState(false);

  useEffect(() => {
    setLoading(true);
    const fetch = useQML ? getQMLRecommendations(productId, 8) : getRecommendations(productId, 8);
    fetch
      .then(setItems)
      .catch(() => setItems([]))
      .finally(() => setLoading(false));
  }, [productId, useQML]);

  if (loading) {
    return (
      <div className={styles.section}>
        <h3>あなたへのおすすめ</h3>
        <p className={styles.status}>読み込み中...</p>
      </div>
    );
  }

  if (items.length === 0 && !loading) {
    return (
      <div className={styles.section}>
        <h3>あなたへのおすすめ</h3>
        <div className={styles.toggleRow}>
          <button
            className={`${styles.toggleBtn} ${!useQML ? styles.toggleActive : ""}`}
            onClick={() => setUseQML(false)}
          >
            古典 Two-Tower
          </button>
          <button
            className={`${styles.toggleBtn} ${useQML ? styles.toggleActiveQml : ""}`}
            onClick={() => setUseQML(true)}
          >
            ⚛ QML (PQC)
          </button>
        </div>
        <p className={styles.status}>{useQML ? "QMLインデックスに該当なし（NISQ 5,000件制限）" : "おすすめなし"}</p>
      </div>
    );
  }

  return (
    <div className={styles.section}>
      <h3>あなたへのおすすめ</h3>
      <div className={styles.toggleRow}>
        <button
          className={`${styles.toggleBtn} ${!useQML ? styles.toggleActive : ""}`}
          onClick={() => setUseQML(false)}
        >
          古典 Two-Tower
        </button>
        <button
          className={`${styles.toggleBtn} ${useQML ? styles.toggleActiveQml : ""}`}
          onClick={() => setUseQML(true)}
        >
          ⚛ QML (PQC)
        </button>
      </div>
      {useQML ? (
        <p className={styles.hint}>量子回路（PQC / 6量子ビット）による推薦 — NISQ試験実装</p>
      ) : (
        <p className={styles.hint}>AIが商品の特徴から似ている商品を提案します</p>
      )}
      <div className={styles.scrollRow}>
        {items.map((item) => (
          <Link key={item.item_id} to={`/products/${item.item_id}`} className={`${styles.card} ${useQML ? styles.qmlCard : ""}`}>
            <div className={styles.imagePlaceholder}>{item.category}</div>
            <p className={styles.name}>{item.name}</p>
            <p className={styles.price}>¥{Math.round(item.price).toLocaleString()}</p>
            {item.is_cold_start && <span className={styles.coldBadge}>NEW</span>}
            {useQML && <span className={styles.qmlBadge}>⚛ QML</span>}
          </Link>
        ))}
      </div>
    </div>
  );
}
