import { useEffect, useState } from "react";
import { Heart } from "lucide-react";
import { Link } from "react-router-dom";
import ProductCard from "../components/ProductCard";
import { listLikeHistory, toggleLike, type LikeHistoryItem } from "../api/likes";
import { useAuth } from "../features/auth/AuthContext";
import homeStyles from "./Home.module.css";
import styles from "./placeholder.module.css";

export default function Likes() {
  const { user } = useAuth();
  const [items, setItems] = useState<LikeHistoryItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!user) {
      setLoading(false);
      return;
    }
    listLikeHistory()
      .then((all) => setItems(all.filter((it) => it.liked)))
      .finally(() => setLoading(false));
  }, [user]);

  const handleToggleLike = async (productId: number) => {
    // 解除したらリストから即削除
    setItems((prev) => prev.filter((it) => it.product.id !== productId));
    try {
      await toggleLike(productId);
    } catch {
      // 失敗したら元に戻す（再フェッチ）
      listLikeHistory()
        .then((all) => setItems(all.filter((it) => it.liked)))
        .catch(() => {});
    }
  };

  if (!user) {
    return (
      <div className={styles.page}>
        <header className={styles.header}><h1>いいね</h1></header>
        <main className={styles.main}>
          <Heart size={48} strokeWidth={1.2} color="#ddd" />
          <p>ログインするといいねした商品が表示されます</p>
          <Link to="/login" className={styles.logoutBtn}>ログイン</Link>
        </main>
      </div>
    );
  }

  return (
    <div className={styles.page}>
      <header className={styles.header}><h1>いいね</h1></header>
      {loading ? (
        <main className={styles.main}>
          <p>読み込み中...</p>
        </main>
      ) : items.length === 0 ? (
        <main className={styles.main}>
          <Heart size={48} strokeWidth={1.2} color="#ddd" />
          <p>いいねした商品がここに表示されます</p>
        </main>
      ) : (
        <div className={homeStyles.main}>
          <p className={homeStyles.searchNote}>これまでにいいねした商品の履歴です</p>
          <div className={homeStyles.grid}>
            {items.map(({ product, liked }) => (
              <ProductCard
                key={product.id}
                product={product}
                liked={liked}
                onToggleLike={handleToggleLike}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
