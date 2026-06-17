import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { listProducts, type Product, CONDITION_LABELS } from "../../api/products";
import { useAuth } from "../auth/AuthContext";
import styles from "./products.module.css";

function parseImageURLs(raw: string): string[] {
  try {
    return JSON.parse(raw) as string[];
  } catch {
    return [];
  }
}

export default function ProductListPage() {
  const { user } = useAuth();
  const [products, setProducts] = useState<Product[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    listProducts({ status: "on_sale", limit: 40 })
      .then((res) => {
        setProducts(res.products ?? []);
        setTotal(res.total);
      })
      .catch(() => setError("商品の取得に失敗しました"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <h1 className={styles.logo}>フリマアプリ</h1>
        <div className={styles.headerActions}>
          {user ? (
            <>
              <span className={styles.userName}>{user.name}</span>
              <Link to="/sell" className={styles.btnPrimary}>出品する</Link>
            </>
          ) : (
            <Link to="/login" className={styles.btnPrimary}>ログイン</Link>
          )}
        </div>
      </header>

      <main className={styles.main}>
        <div className={styles.listHeader}>
          <h2>出品中の商品 <span className={styles.count}>({total}件)</span></h2>
        </div>

        {loading && <p className={styles.status}>読み込み中...</p>}
        {error && <p className={styles.error}>{error}</p>}

        {!loading && products.length === 0 && (
          <p className={styles.status}>商品がまだありません</p>
        )}

        <div className={styles.grid}>
          {products.map((p) => {
            const images = parseImageURLs(p.image_urls);
            return (
              <Link key={p.id} to={`/products/${p.id}`} className={styles.card}>
                <div className={styles.cardImage}>
                  {images[0] ? (
                    <img src={images[0]} alt={p.title} />
                  ) : (
                    <div className={styles.noImage}>No Image</div>
                  )}
                </div>
                <div className={styles.cardBody}>
                  <p className={styles.cardTitle}>{p.title}</p>
                  <p className={styles.cardCondition}>{CONDITION_LABELS[p.condition]}</p>
                  <p className={styles.cardPrice}>¥{p.price.toLocaleString()}</p>
                </div>
              </Link>
            );
          })}
        </div>
      </main>
    </div>
  );
}
