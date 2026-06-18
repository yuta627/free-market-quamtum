import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { getRecommendations, type RecommendedItem } from "../api/recommendations";
import styles from "./RecommendationSection.module.css";

interface RecommendationSectionProps {
  productId: number;
}

export default function RecommendationSection({ productId }: RecommendationSectionProps) {
  const [items, setItems] = useState<RecommendedItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    getRecommendations(productId, 8)
      .then(setItems)
      .catch(() => setItems([]))
      .finally(() => setLoading(false));
  }, [productId]);

  if (loading) {
    return (
      <div className={styles.section}>
        <h3>あなたへのおすすめ</h3>
        <p className={styles.status}>読み込み中...</p>
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className={styles.section}>
        <h3>あなたへのおすすめ</h3>
        <p className={styles.status}>おすすめなし</p>
      </div>
    );
  }

  return (
    <div className={styles.section}>
      <h3>あなたへのおすすめ</h3>
      <p className={styles.hint}>量子強化AI（古典Two-Tower → PCA → PQC）による推薦</p>
      <div className={styles.scrollRow}>
        {items.map((item) => (
          <Link key={item.item_id} to={`/products/${item.item_id}`} className={styles.card}>
            <div className={styles.imagePlaceholder}>{item.category}</div>
            <p className={styles.name}>{item.name}</p>
            <p className={styles.price}>¥{Math.round(item.price).toLocaleString()}</p>
            {item.is_cold_start && <span className={styles.coldBadge}>NEW</span>}
          </Link>
        ))}
      </div>
    </div>
  );
}
