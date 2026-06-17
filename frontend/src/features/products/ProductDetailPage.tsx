import { useEffect, useState, type FormEvent } from "react";
import { Link, useParams } from "react-router-dom";
import { Heart } from "lucide-react";
import { getProduct, type Product, CONDITION_LABELS } from "../../api/products";
import { askProductQuestion } from "../../api/ai";
import { toggleLike, listLikeHistory } from "../../api/likes";
import { useAuth } from "../auth/AuthContext";
import CheckoutModal from "../../components/CheckoutModal";
import RecommendationSection from "../../components/RecommendationSection";
import styles from "./products.module.css";

interface QAEntry {
  question: string;
  answer: string;
}

function parseImageURLs(raw: string): string[] {
  try {
    return JSON.parse(raw) as string[];
  } catch {
    return [];
  }
}

export default function ProductDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { user } = useAuth();
  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [selectedImg, setSelectedImg] = useState(0);
  const [purchased, setPurchased] = useState(false);
  const [showCheckout, setShowCheckout] = useState(false);
  const [question, setQuestion] = useState("");
  const [qaHistory, setQaHistory] = useState<QAEntry[]>([]);
  const [asking, setAsking] = useState(false);
  const [askError, setAskError] = useState("");
  const [liked, setLiked] = useState(false);

  useEffect(() => {
    if (!id) return;
    getProduct(Number(id))
      .then(setProduct)
      .catch(() => setError("商品の取得に失敗しました"))
      .finally(() => setLoading(false));
  }, [id]);

  useEffect(() => {
    if (!user || !id) return;
    listLikeHistory()
      .then((items) =>
        setLiked(items.some((it) => it.product.id === Number(id) && it.liked))
      )
      .catch(() => {});
  }, [user, id]);

  if (loading) return <div className={styles.page}><p className={styles.status}>読み込み中...</p></div>;
  if (error || !product) return (
    <div className={styles.page}>
      <p className={styles.error}>{error || "商品が見つかりません"}</p>
      <Link to="/">一覧に戻る</Link>
    </div>
  );

  const images = parseImageURLs(product.image_urls);

  const handleCheckoutSuccess = (updated: Product) => {
    setProduct(updated);
    setPurchased(true);
    setShowCheckout(false);
  };

  const handleToggleLike = async () => {
    if (!user) return;
    setLiked((prev) => !prev);
    try {
      await toggleLike(product.id);
    } catch {
      setLiked((prev) => !prev);
    }
  };

  const handleAsk = async (e: FormEvent) => {
    e.preventDefault();
    const q = question.trim();
    if (!q || asking) return;
    setAsking(true);
    setAskError("");
    try {
      const answer = await askProductQuestion(product.id, q);
      setQaHistory((prev) => [...prev, { question: q, answer }]);
      setQuestion("");
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
        "AIへの質問に失敗しました";
      setAskError(msg);
    } finally {
      setAsking(false);
    }
  };

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <Link to="/" className={styles.logo}>フリマアプリ</Link>
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

      <main className={styles.detailMain}>
        <div className={styles.detailImages}>
          <div className={styles.mainImage}>
            {images[selectedImg] ? (
              <img src={images[selectedImg]} alt={product.title} />
            ) : (
              <div className={styles.noImage}>No Image</div>
            )}
          </div>
          {images.length > 1 && (
            <div className={styles.thumbnails}>
              {images.map((url, i) => (
                <img
                  key={i}
                  src={url}
                  alt=""
                  className={i === selectedImg ? styles.thumbActive : styles.thumb}
                  onClick={() => setSelectedImg(i)}
                />
              ))}
            </div>
          )}
        </div>

        <div className={styles.detailInfo}>
          <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: ".5rem" }}>
            <h1 className={styles.detailTitle}>{product.title}</h1>
            {user && (
              <button
                onClick={handleToggleLike}
                aria-label={liked ? "いいねを取り消す" : "いいねする"}
                style={{
                  background: "none",
                  border: "none",
                  cursor: "pointer",
                  padding: ".25rem",
                  flexShrink: 0,
                }}
              >
                <Heart size={26} fill={liked ? "#f60" : "none"} color={liked ? "#f60" : "#888"} />
              </button>
            )}
          </div>
          <p className={styles.detailPrice}>¥{product.price.toLocaleString()}</p>

          <div className={styles.detailMeta}>
            <div className={styles.metaRow}>
              <span>状態</span><span>{CONDITION_LABELS[product.condition]}</span>
            </div>
            <div className={styles.metaRow}>
              <span>出品者</span><span>{product.seller?.name ?? "不明"}</span>
            </div>
            <div className={styles.metaRow}>
              <span>ステータス</span>
              <span className={product.status === "on_sale" ? styles.onSale : styles.sold}>
                {product.status === "on_sale" ? "販売中" : "売り切れ"}
              </span>
            </div>
          </div>

          {product.description && (
            <div className={styles.description}>
              <h3>商品説明</h3>
              <p>{product.description}</p>
            </div>
          )}

          <RecommendationSection productId={product.id} />

          <div className={styles.qaSection}>
            <h3>AIに質問する</h3>
            <p className={styles.qaHint}>商品についての疑問をAIが商品情報をもとに回答します</p>

            {qaHistory.length > 0 && (
              <div className={styles.qaHistory}>
                {qaHistory.map((qa, i) => (
                  <div key={i} className={styles.qaEntry}>
                    <p className={styles.qaQuestion}>Q. {qa.question}</p>
                    <p className={styles.qaAnswer}>{qa.answer}</p>
                  </div>
                ))}
              </div>
            )}

            {askError && <p className={styles.error}>{askError}</p>}

            <form onSubmit={handleAsk} className={styles.qaForm}>
              <input
                type="text"
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                placeholder="例：付属品は揃っていますか？"
                disabled={asking}
                className={styles.qaInput}
              />
              <button type="submit" className={styles.qaSubmitBtn} disabled={asking || !question.trim()}>
                {asking ? "質問中..." : "質問する"}
              </button>
            </form>
          </div>

          {purchased && (
            <p style={{ color: "#2a8f2a", fontWeight: 700, marginBottom: ".75rem" }}>
              購入が完了しました！
            </p>
          )}

          {product.status === "on_sale" && user && user.id !== product.seller_id && (
            <div style={{ display: "flex", flexDirection: "column", gap: ".75rem" }}>
              <button
                className={styles.buyButton}
                onClick={() => setShowCheckout(true)}
              >
                クレジットカード／PayPayで購入する
              </button>
              <Link
                to={`/products/${product.id}/chat`}
                className={styles.buyButton}
                style={{ background: "#fff", color: "#f60", border: "2px solid #f60", textAlign: "center" }}
              >
                出品者に問い合わせる
              </Link>
            </div>
          )}
          {!user && (
            <Link to="/login" className={styles.buyButton}>ログインして購入</Link>
          )}
        </div>
      </main>

      {showCheckout && (
        <CheckoutModal
          product={product}
          onClose={() => setShowCheckout(false)}
          onSuccess={handleCheckoutSuccess}
        />
      )}
    </div>
  );
}
