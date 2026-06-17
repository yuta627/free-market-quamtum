import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { getAuction, placeBid, type Auction } from "../api/auctions";
import { useAuth } from "../features/auth/AuthContext";
import styles from "./AuctionDetail.module.css";

function formatRemaining(endsAt: string) {
  const diff = new Date(endsAt).getTime() - Date.now();
  if (diff <= 0) return "終了";
  const h = Math.floor(diff / 3600000);
  const m = Math.floor((diff % 3600000) / 60000);
  if (h >= 24) return `${Math.floor(h / 24)}日${h % 24}時間`;
  if (h > 0) return `${h}時間${m}分`;
  return `${m}分`;
}

export default function AuctionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { user } = useAuth();
  const [auction, setAuction] = useState<Auction | null>(null);
  const [loading, setLoading] = useState(true);
  const [bidAmount, setBidAmount] = useState("");
  const [bidLoading, setBidLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const load = () => {
    if (!id) return;
    setLoading(true);
    getAuction(Number(id))
      .then(setAuction)
      .catch(() => setError("オークションが見つかりません"))
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, [id]);

  const handleBid = async () => {
    if (!auction) return;
    setError("");
    setSuccess("");
    const amount = Number(bidAmount);
    if (!amount || amount <= auction.current_price) {
      setError(`現在価格（¥${auction.current_price.toLocaleString()}）より高い金額を入力してください`);
      return;
    }
    setBidLoading(true);
    try {
      const updated = await placeBid(auction.id, amount);
      setAuction(updated);
      setSuccess(`¥${amount.toLocaleString()} で入札しました！`);
      setBidAmount("");
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? "入札に失敗しました";
      setError(msg);
    } finally {
      setBidLoading(false);
    }
  };

  if (loading) return <div className={styles.page}><p className={styles.status}>読み込み中...</p></div>;
  if (!auction) return <div className={styles.page}><p className={styles.status}>{error || "見つかりません"}</p></div>;

  const remaining = formatRemaining(auction.ends_at);
  const ended = remaining === "終了";

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <Link to="/" className={styles.logo}>フリマアプリ</Link>
        {user && <span className={styles.userName}>{user.name}</span>}
      </header>

      <main className={styles.main}>
        <div className={styles.card}>
          <div className={styles.imgBox}>
            <span className={styles.imgText}>No Image</span>
          </div>

          <div className={styles.body}>
            <div className={styles.auctionBadge}>🔨 オークション</div>
            <h1 className={styles.title}>{auction.product.title}</h1>
            <p className={styles.desc}>{auction.product.description}</p>

            <div className={styles.priceSection}>
              <div className={styles.priceRow}>
                <span className={styles.priceLabel}>現在価格</span>
                <span className={styles.currentPrice}>¥{auction.current_price.toLocaleString()}</span>
              </div>
              <div className={styles.priceRow}>
                <span className={styles.priceLabel}>開始価格</span>
                <span className={styles.startingPrice}>¥{auction.starting_price.toLocaleString()}</span>
              </div>
              <div className={styles.priceRow}>
                <span className={styles.priceLabel}>入札数</span>
                <span className={styles.bidCount}>{auction.bid_count}件</span>
              </div>
              <div className={styles.priceRow}>
                <span className={styles.priceLabel}>残り時間</span>
                <span className={ended ? styles.ended : styles.remaining}>{remaining}</span>
              </div>
            </div>

            {!ended && user && (
              <div className={styles.bidSection}>
                <h3>入札する</h3>
                {error && <p className={styles.error}>{error}</p>}
                {success && <p className={styles.successMsg}>{success}</p>}
                <div className={styles.bidRow}>
                  <input
                    type="number"
                    value={bidAmount}
                    onChange={(e) => setBidAmount(e.target.value)}
                    placeholder={`¥${(auction.current_price + 1).toLocaleString()} 以上`}
                    min={auction.current_price + 1}
                    className={styles.bidInput}
                  />
                  <button
                    onClick={handleBid}
                    disabled={bidLoading}
                    className={styles.bidBtn}
                  >
                    {bidLoading ? "入札中..." : "入札する"}
                  </button>
                </div>
              </div>
            )}
            {!ended && !user && (
              <p className={styles.loginNote}><Link to="/login">ログイン</Link>すると入札できます</p>
            )}
            {ended && <p className={styles.endedNote}>このオークションは終了しました</p>}

            {auction.bids && auction.bids.length > 0 && (
              <div className={styles.bidHistory}>
                <h3>入札履歴</h3>
                {[...auction.bids].reverse().map((bid) => (
                  <div key={bid.id} className={styles.bidHistoryItem}>
                    <span className={styles.bidder}>{bid.bidder?.name ?? "ユーザー"}</span>
                    <span className={styles.bidAmount}>¥{bid.amount.toLocaleString()}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}
