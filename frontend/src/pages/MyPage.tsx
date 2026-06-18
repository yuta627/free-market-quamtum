import { useEffect, useState } from "react";
import { User } from "lucide-react";
import { useAuth } from "../features/auth/AuthContext";
import { useNavigate } from "react-router-dom";
import { listMyProducts, listMyPurchases, type Product } from "../api/products";
import { getMe, updateAddress } from "../api/auth";
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
  const [addressForm, setAddressForm] = useState({
    postal_code: "",
    prefecture: "",
    city: "",
    address_line: "",
    building: "",
  });
  const [addressSaving, setAddressSaving] = useState(false);
  const [addressMsg, setAddressMsg] = useState("");

  useEffect(() => {
    if (!user) {
      setLoading(false);
      return;
    }
    Promise.all([listMyProducts(), listMyPurchases(), getMe()])
      .then(([mine, bought, me]) => {
        setMyProducts(mine);
        setPurchases(bought);
        setAddressForm({
          postal_code: me.postal_code ?? "",
          prefecture: me.prefecture ?? "",
          city: me.city ?? "",
          address_line: me.address_line ?? "",
          building: me.building ?? "",
        });
      })
      .finally(() => setLoading(false));
  }, [user]);

  const handleAddressSave = async () => {
    setAddressSaving(true);
    setAddressMsg("");
    try {
      await updateAddress(addressForm);
      setAddressMsg("住所を保存しました");
    } catch {
      setAddressMsg("保存に失敗しました");
    } finally {
      setAddressSaving(false);
    }
  };

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
          <button className={styles.logoutBtn} onClick={() => navigate("/login")}>
            ログイン
          </button>
          <button className={styles.logoutBtn} onClick={() => navigate("/signup")} style={{ marginTop: "0.5rem" }}>
            新規登録
          </button>
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
          <section style={{ padding: "1rem", borderBottom: "1px solid #eee" }}>
            <h2 style={{ fontSize: ".95rem", color: "#333", margin: "0 0 .75rem" }}>配送先住所</h2>
            {[
              { label: "郵便番号", key: "postal_code", placeholder: "1234567" },
              { label: "都道府県", key: "prefecture", placeholder: "東京都" },
              { label: "市区町村", key: "city", placeholder: "渋谷区" },
              { label: "番地", key: "address_line", placeholder: "1-2-3" },
              { label: "建物名", key: "building", placeholder: "○○マンション101号室（任意）" },
            ].map(({ label, key, placeholder }) => (
              <div key={key} style={{ marginBottom: ".5rem" }}>
                <label style={{ fontSize: ".8rem", color: "#555", display: "block", marginBottom: ".25rem" }}>{label}</label>
                <input
                  type="text"
                  placeholder={placeholder}
                  value={addressForm[key as keyof typeof addressForm]}
                  onChange={(e) => setAddressForm((f) => ({ ...f, [key]: e.target.value }))}
                  style={{ width: "100%", padding: ".5rem", border: "1px solid #ddd", borderRadius: "6px", fontSize: ".875rem", boxSizing: "border-box" }}
                />
              </div>
            ))}
            <button
              onClick={handleAddressSave}
              disabled={addressSaving}
              style={{ marginTop: ".5rem", padding: ".5rem 1.25rem", background: "#ff6b35", color: "#fff", border: "none", borderRadius: "6px", fontSize: ".875rem", cursor: "pointer" }}
            >
              {addressSaving ? "保存中..." : "住所を保存"}
            </button>
            {addressMsg && <p style={{ fontSize: ".8rem", color: addressMsg.includes("失敗") ? "#e53e3e" : "#38a169", marginTop: ".5rem" }}>{addressMsg}</p>}
          </section>
          <ProductSection title="購入した商品" products={purchases} />
          <ProductSection title="出品した商品" products={myProducts} />
        </>
      )}
    </div>
  );
}
