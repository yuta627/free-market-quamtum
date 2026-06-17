import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { createProduct, type ProductCondition, CONDITION_LABELS } from "../../api/products";
import { createAuction } from "../../api/auctions";
import { generateDescription } from "../../api/ai";
import { useAuth } from "../auth/AuthContext";
import styles from "./products.module.css";

const CONDITIONS = Object.entries(CONDITION_LABELS) as [ProductCondition, string][];

export default function SellPage() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const [mode, setMode] = useState<"normal" | "auction">("normal");
  const [form, setForm] = useState({
    title: "",
    description: "",
    price: "",
    condition: "good" as ProductCondition,
    imageUrl: "",
  });
  const [auctionDuration, setAuctionDuration] = useState("24");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [aiLoading, setAiLoading] = useState(false);
  const [aiKeywords, setAiKeywords] = useState("");

  if (!user) {
    navigate("/login");
    return null;
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    const price = Number(form.price);
    if (isNaN(price) || price < 0) {
      setError("価格は0以上の数値を入力してください");
      return;
    }
    setLoading(true);
    try {
      const imageURLs = form.imageUrl.trim() ? [form.imageUrl.trim()] : [];
      if (mode === "auction") {
        const endsAt = new Date(Date.now() + Number(auctionDuration) * 60 * 60 * 1000).toISOString();
        const auction = await createAuction({
          title: form.title,
          description: form.description,
          condition: form.condition,
          image_urls: imageURLs,
          starting_price: price,
          ends_at: endsAt,
        });
        navigate(`/auctions/${auction.id}`);
      } else {
        const product = await createProduct({
          title: form.title,
          description: form.description,
          price,
          condition: form.condition,
          image_urls: imageURLs,
        });
        navigate(`/products/${product.id}`);
      }
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
        "出品に失敗しました";
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  const set = (key: keyof typeof form) => (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>
  ) => setForm((f) => ({ ...f, [key]: e.target.value }));

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <Link to="/" className={styles.logo}>フリマアプリ</Link>
        <span className={styles.userName}>{user.name}</span>
      </header>

      <main className={styles.formMain}>
        <div className={styles.formCard}>
          <h1>商品を出品する</h1>

          {/* 出品モード切り替え */}
          <div className={styles.modeToggle}>
            <button
              type="button"
              className={`${styles.modeBtn} ${mode === "normal" ? styles.modeBtnActive : ""}`}
              onClick={() => setMode("normal")}
            >
              通常出品
            </button>
            <button
              type="button"
              className={`${styles.modeBtn} ${mode === "auction" ? styles.modeBtnAuctionActive : ""}`}
              onClick={() => setMode("auction")}
            >
              🔨 オークション出品
            </button>
          </div>

          {error && <p className={styles.formError}>{error}</p>}

          <form onSubmit={handleSubmit} className={styles.form}>
            <label className={styles.label}>
              商品名 <span className={styles.required}>必須</span>
              <input
                type="text"
                value={form.title}
                onChange={set("title")}
                required
                maxLength={200}
                placeholder="例：iPhone 14 Pro 128GB"
              />
            </label>

            <label className={styles.label}>
              商品の説明
              <div className={styles.aiRow}>
                <input
                  type="text"
                  value={aiKeywords}
                  onChange={(e) => setAiKeywords(e.target.value)}
                  placeholder="キーワード（例：美品、付属品あり、ほぼ未使用）"
                  className={styles.aiKeywordInput}
                />
                <button
                  type="button"
                  className={styles.aiBtn}
                  disabled={aiLoading || !form.title}
                  onClick={async () => {
                    setAiLoading(true);
                    try {
                      const desc = await generateDescription(form.title, aiKeywords);
                      setForm((f) => ({ ...f, description: desc }));
                    } catch {
                      setError("AI生成に失敗しました");
                    } finally {
                      setAiLoading(false);
                    }
                  }}
                >
                  {aiLoading ? "生成中..." : "AIで説明文を自動生成する"}
                </button>
              </div>
              <textarea
                value={form.description}
                onChange={set("description")}
                rows={5}
                placeholder="商品の状態・付属品・購入時期など"
              />
            </label>

            <label className={styles.label}>
              状態 <span className={styles.required}>必須</span>
              <select value={form.condition} onChange={set("condition")} required>
                {CONDITIONS.map(([val, label]) => (
                  <option key={val} value={val}>{label}</option>
                ))}
              </select>
            </label>

            <label className={styles.label}>
              {mode === "auction" ? "開始価格（円）" : "販売価格（円）"}
              <span className={styles.required}>必須</span>
              <input
                type="number"
                value={form.price}
                onChange={set("price")}
                required
                min={0}
                placeholder="0"
              />
            </label>

            {mode === "auction" && (
              <label className={styles.label}>
                オークション期間 <span className={styles.required}>必須</span>
                <select value={auctionDuration} onChange={(e) => setAuctionDuration(e.target.value)}>
                  <option value="1">1時間</option>
                  <option value="3">3時間</option>
                  <option value="6">6時間</option>
                  <option value="12">12時間</option>
                  <option value="24">24時間</option>
                  <option value="48">48時間</option>
                  <option value="72">72時間</option>
                </select>
              </label>
            )}

            <label className={styles.label}>
              画像URL（任意）
              <input
                type="url"
                value={form.imageUrl}
                onChange={set("imageUrl")}
                placeholder="https://example.com/image.jpg"
              />
            </label>

            <button type="submit" className={mode === "auction" ? styles.auctionSubmitBtn : styles.submitBtn} disabled={loading}>
              {loading ? "出品中..." : mode === "auction" ? "🔨 オークションに出品する" : "出品する"}
            </button>
          </form>
        </div>
      </main>
    </div>
  );
}
