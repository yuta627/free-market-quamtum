import { useEffect, useState } from "react";
import { User } from "lucide-react";
import { useAuth } from "../features/auth/AuthContext";
import { useNavigate } from "react-router-dom";
import { listMyProducts, listMyPurchases, type Product } from "../api/products";
import ProductCard from "../components/ProductCard";
import homeStyles from "./Home.module.css";
import styles from "./placeholder.module.css";

function ProductSection({ title, products }: { title: string; products: Product[] }) {
  return (
    <section style={{ padding: "1rem" }}>
      <h2 style={{ fontSize: ".95rem", color: "#333", margin: "0 0 .75rem" }}>{title}</h2>
      {products.length === 0 ? (
        <p style={{ color: "#aaa", fontSize: ".85rem" }}>まだありません</p>
      ) : (
        <div className={homeStyles.grid}>
          {products.map((p) => (
            <ProductCard
              key={p.id}
              product={p}
              statusBadge={p.status === "sold" ? "売り切れ" : undefined}
            />
          ))}
        </div>
      )}
    </section>
  );
}

export default function MyPage() {
  const { user, clearAuth } = useAuth();
  const navigate = useNavigate();
  const [myProducts, setMyProducts] = useState<Product[]>([]);
  const [purchases, setPurchases] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!user) {
      setLoading(false);
      return;
    }
    Promise.all([listMyProducts(), listMyPurchases()])
      .then(([mine, bought]) => {
        setMyProducts(mine);
        setPurchases(bought);
      })
      .finally(() => setLoading(false));
  }, [user]);

  const handleLogout = () => {
    clearAuth();
    navigate("/login");
  };

  if (!user) {
    return (
      <div className={styles.page}>
        <header className={styles.header}><h1>マイページ</h1></header>
        <main className={styles.main}>
          <User size={48} strokeWidth={1.2} color="#ddd" />
          <p>ログインしていません</p>
        </main>
      </div>
    );
  }

  return (
    <div className={styles.page} style={{ paddingBottom: "72px" }}>
      <header className={styles.header}><h1>マイページ</h1></header>

      <div style={{ padding: "1rem", display: "flex", alignItems: "center", justifyContent: "space-between" }}>
        <p style={{ margin: 0, fontWeight: 700 }}>{user.name} さん</p>
        <button className={styles.logoutBtn} onClick={handleLogout} style={{ marginTop: 0 }}>
          ログアウト
        </button>
      </div>

      {loading ? (
        <p style={{ textAlign: "center", color: "#aaa" }}>読み込み中...</p>
      ) : (
        <>
          <ProductSection title="購入した商品" products={purchases} />
          <ProductSection title="出品した商品" products={myProducts} />
        </>
      )}
    </div>
  );
}
