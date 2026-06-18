import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import Header from "../components/Header";
import ProductCard from "../components/ProductCard";
import { listProducts, type Product } from "../api/products";
import { listAuctions, finalizeAuction, type Auction } from "../api/auctions";
import { toggleLike, listLikeHistory } from "../api/likes";
import { type QuantumRandomResult } from "../api/quantum";
import { useAuth } from "../features/auth/AuthContext";
import styles from "./Home.module.css";

const TABS = ["おすすめ", "クーポン", "ランキング", "オークション"];

function couponRate(id: number) {
  return [10, 15, 20, 25, 30][id % 5];
}

function formatRemaining(endsAt: string) {
  const ms = new Date(endsAt).getTime() - Date.now();
  if (ms <= 0) return "終了";
  const h = Math.floor(ms / 3600_000);
  const m = Math.floor((ms % 3600_000) / 60_000);
  if (h >= 24) return `${Math.floor(h / 24)}日 ${h % 24}時間`;
  return `${h}時間 ${m}分`;
}

export default function Home() {
  const { user } = useAuth();
  const [activeTab, setActiveTab] = useState(0);
  const [query, setQuery] = useState("");
  const [products, setProducts] = useState<Product[]>([]);
  const [likedIds, setLikedIds] = useState<Set<number>>(new Set());
  const [loading, setLoading] = useState(true);
  const [auctions, setAuctions] = useState<Auction[]>([]);
  const [auctionError, setAuctionError] = useState(false);
  const [qrngResults, setQrngResults] = useState<Record<number, QuantumRandomResult & { winner: string; auction?: Auction }>>({});
  const [qrngLoading, setQrngLoading] = useState<Record<number, boolean>>({});

  useEffect(() => {
    const t = setTimeout(() => {
      setLoading(true);
      listProducts({ q: query || undefined, limit: 40 })
        .then((res) => setProducts(res.products))
        .catch(() => setProducts([]))
        .finally(() => setLoading(false));
    }, query ? 300 : 0);
    return () => clearTimeout(t);
  }, [query]);

  useEffect(() => {
    if (activeTab === 3) {
      setAuctionError(false);
      listAuctions()
        .then((res) => setAuctions(res.auctions ?? []))
        .catch(() => setAuctionError(true));
    }
  }, [activeTab]);

  useEffect(() => {
    if (!user) { setLikedIds(new Set()); return; }
    listLikeHistory()
      .then((items) =>
        setLikedIds(new Set(items.filter((it) => it.liked).map((it) => it.product.id)))
      )
      .catch(() => {});
  }, [user]);

  const handleToggleLike = async (productId: number) => {
    if (!user) return;
    setLikedIds((prev) => {
      const next = new Set(prev);
      next.has(productId) ? next.delete(productId) : next.add(productId);
      return next;
    });
    try {
      await toggleLike(productId);
    } catch {
      setLikedIds((prev) => {
        const next = new Set(prev);
        next.has(productId) ? next.delete(productId) : next.add(productId);
        return next;
      });
    }
  };

  const base = products;

  // Tab-specific derived lists
  const ranked = useMemo(() => [...base].sort((a, b) => b.price - a.price), [base]);
  const couponed = useMemo(() => [...base].sort((a, b) => couponRate(b.id) - couponRate(a.id)), [base]);

  return (
    <div className={styles.page}>
      <Header onSearch={setQuery} />

      <div className={styles.tabs}>
        {TABS.map((tab, i) => (
          <button
            key={tab}
            className={`${styles.tab} ${activeTab === i ? styles.tabActive : ""}`}
            onClick={() => setActiveTab(i)}
          >
            {tab}
          </button>
        ))}
      </div>

      <main className={styles.main}>
        {query && <p className={styles.searchNote}>「{query}」の検索結果</p>}

        {loading ? (
          <p className={styles.placeholder}>読み込み中...</p>
        ) : base.length === 0 ? (
          <p className={styles.placeholder}>商品がありません</p>
        ) : activeTab === 0 ? (
          /* おすすめ */
          <div className={styles.grid}>
            {base.map((p) => (
              <ProductCard key={p.id} product={p} liked={likedIds.has(p.id)} onToggleLike={handleToggleLike} />
            ))}
          </div>
        ) : activeTab === 1 ? (
          /* クーポン */
          <>
            <p className={styles.tabNote}>対象商品にクーポンが適用されます</p>
            <div className={styles.grid}>
              {couponed.map((p) => {
                const rate = couponRate(p.id);
                const discounted = Math.floor(p.price * (1 - rate / 100));
                return (
                  <Link key={p.id} to={`/products/${p.id}`} className={styles.couponCard}>
                    <div className={styles.couponBadge}>{rate}% OFF</div>
                    <div className={styles.couponImg}>
                      <span className={styles.couponImgText}>No Image</span>
                    </div>
                    <div className={styles.couponBody}>
                      <p className={styles.couponTitle}>{p.title}</p>
                      <p className={styles.couponOriginal}>¥{p.price.toLocaleString()}</p>
                      <p className={styles.couponPrice}>¥{discounted.toLocaleString()}</p>
                    </div>
                  </Link>
                );
              })}
            </div>
          </>
        ) : activeTab === 2 ? (
          /* ランキング */
          <>
            <p className={styles.tabNote}>人気順（価格）でランキング表示</p>
            <div className={styles.grid}>
              {ranked.map((p, i) => (
                <ProductCard
                  key={p.id}
                  product={p}
                  liked={likedIds.has(p.id)}
                  onToggleLike={handleToggleLike}
                  statusBadge={`${i + 1}位`}
                />
              ))}
            </div>
          </>
        ) : (
          /* オークション */
          <>
            <p className={styles.tabNote}>入札形式で購入できます。同額入札は量子抽選で落札者を決定します</p>
            {auctionError ? (
              <p className={styles.placeholder}>オークション情報の取得に失敗しました</p>
            ) : auctions.length === 0 ? (
              <p className={styles.placeholder}>現在開催中のオークションはありません</p>
            ) : (
              <div className={styles.grid}>
                {auctions.map((a) => {
                  const remaining = formatRemaining(a.ends_at);
                  const isEnded = remaining === "終了" && a.status === "active" && a.bid_count > 0;
                  const isSeller = user && user.id === a.product.seller.id;
                  const qResult = qrngResults[a.id];
                  const isLoading = qrngLoading[a.id];

                  const handleQuantumDraw = async (e: React.MouseEvent) => {
                    e.preventDefault();
                    setQrngLoading((prev) => ({ ...prev, [a.id]: true }));
                    try {
                      const result = await finalizeAuction(a.id);
                      setQrngResults((prev) => ({ ...prev, [a.id]: { value: 0, bits: [], n_qubits: 0, circuit_depth: 0, purpose: "auction", winner: result.winner?.name ?? `ID:${result.winner_id}`, auction: result } }));
                      setAuctions((prev) => prev.map((au) => au.id === a.id ? result : au));
                    } catch {
                      alert("落札者決定に失敗しました。オークションが終了しているか確認してください。");
                    } finally {
                      setQrngLoading((prev) => ({ ...prev, [a.id]: false }));
                    }
                  };

                  return (
                    <div key={a.id} className={styles.auctionCard}>
                      <Link to={`/auctions/${a.id}`} className={styles.auctionLink}>
                        <div className={styles.auctionImg}>
                          <span className={styles.auctionImgText}>No Image</span>
                        </div>
                        <div className={styles.auctionBody}>
                          <p className={styles.auctionTitle}>{a.product.title}</p>
                          <p className={styles.auctionBid}>現在 ¥{a.current_price.toLocaleString()}</p>
                          <div className={styles.auctionMeta}>
                            <span className={styles.auctionBids}>{a.bid_count}件の入札</span>
                            <span className={remaining === "終了" ? styles.auctionEnded : styles.auctionRemaining}>
                              残り {remaining}
                            </span>
                          </div>
                        </div>
                      </Link>

                      {isEnded && isSeller && !qResult && (
                        <button className={styles.quantumBtn} onClick={handleQuantumDraw} disabled={isLoading}>
                          {isLoading ? "⏳ 量子計算中..." : "⚛️ 量子抽選で落札者決定"}
                        </button>
                      )}
                      {qResult && (
                        <div className={styles.quantumResult}>
                          <p className={styles.quantumTitle}>⚛️ 量子抽選完了</p>
                          {qResult.bits.length > 0 && (
                            <p className={styles.quantumBits}>
                              {qResult.bits.join("")}
                              <span className={styles.quantumBitsMeta}> ({qResult.n_qubits}量子ビット / depth {qResult.circuit_depth})</span>
                            </p>
                          )}
                          <p className={styles.quantumWinner}>🏆 {qResult.winner} が落札</p>
                          <p className={styles.quantumNote}>Hadamardゲートによる真の乱数。予測・再現不可能。</p>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </>
        )}
      </main>
    </div>
  );
}
