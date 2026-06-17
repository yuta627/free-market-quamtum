import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type KeyboardEvent,
} from "react";
import { Link, useParams } from "react-router-dom";
import { listMessages, sendMessage, type Message } from "../../api/messages";
import { getProduct, type Product } from "../../api/products";
import { useAuth } from "../auth/AuthContext";
import styles from "./chat.module.css";

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleTimeString("ja-JP", { hour: "2-digit", minute: "2-digit" });
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleDateString("ja-JP", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

function groupByDate(messages: Message[]): { date: string; msgs: Message[] }[] {
  const groups: { date: string; msgs: Message[] }[] = [];
  for (const m of messages) {
    const date = formatDate(m.created_at);
    const last = groups[groups.length - 1];
    if (last && last.date === date) {
      last.msgs.push(m);
    } else {
      groups.push({ date, msgs: [m] });
    }
  }
  return groups;
}

export default function ChatPage() {
  const { id } = useParams<{ id: string }>();
  const productId = Number(id);
  const { user } = useAuth();

  const [product, setProduct] = useState<Product | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [body, setBody] = useState("");
  const [sending, setSending] = useState(false);
  const [loadErr, setLoadErr] = useState("");

  const bottomRef = useRef<HTMLDivElement>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const scrollToBottom = () =>
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });

  const fetchMessages = async () => {
    try {
      const res = await listMessages(productId);
      setMessages(res.messages ?? []);
    } catch {
      // silent for polling
    }
  };

  useEffect(() => {
    getProduct(productId).then(setProduct).catch(() => setLoadErr("商品が見つかりません"));
    fetchMessages();

    // 3秒ごとにポーリング
    pollRef.current = setInterval(fetchMessages, 3000);
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, [productId]);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleSend = async (e?: FormEvent) => {
    e?.preventDefault();
    const trimmed = body.trim();
    if (!trimmed || sending) return;

    setSending(true);
    try {
      const msg = await sendMessage(productId, trimmed);
      setMessages((prev) => [...prev, msg]);
      setBody("");
    } catch {
      // エラーは後続ポーリングで回復
    } finally {
      setSending(false);
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      handleSend();
    }
  };

  if (loadErr) {
    return (
      <div className={styles.page}>
        <p className={styles.errCenter}>{loadErr}</p>
      </div>
    );
  }

  const groups = groupByDate(messages);

  return (
    <div className={styles.page}>
      {/* ヘッダー */}
      <header className={styles.header}>
        <Link to={`/products/${productId}`} className={styles.back}>
          ‹
        </Link>
        <div className={styles.headerInfo}>
          <p className={styles.headerTitle}>{product?.title ?? "読み込み中..."}</p>
          {product && (
            <p className={styles.headerPrice}>¥{product.price.toLocaleString()}</p>
          )}
        </div>
      </header>

      {/* 商品サマリーバー */}
      {product && (
        <div className={styles.productBar}>
          <div className={styles.productBarThumb}>
            <span>📦</span>
          </div>
          <div className={styles.productBarText}>
            <p className={styles.productBarTitle}>{product.title}</p>
            <p className={styles.productBarPrice}>¥{product.price.toLocaleString()}</p>
          </div>
          <Link to={`/products/${productId}`} className={styles.productBarLink}>
            詳細
          </Link>
        </div>
      )}

      {/* メッセージ一覧 */}
      <main className={styles.messageArea}>
        {messages.length === 0 && (
          <p className={styles.empty}>まだメッセージはありません</p>
        )}

        {groups.map((group) => (
          <div key={group.date}>
            <div className={styles.dateSep}>
              <span>{group.date}</span>
            </div>
            {group.msgs.map((m) => {
              const isMine = user?.id === m.sender_id;
              return (
                <div
                  key={m.id}
                  className={isMine ? styles.rowMine : styles.rowTheirs}
                >
                  {!isMine && (
                    <div className={styles.avatar}>
                      {m.sender?.name?.[0] ?? "?"}
                    </div>
                  )}
                  <div className={isMine ? styles.bubbleMine : styles.bubbleTheirs}>
                    {!isMine && (
                      <p className={styles.senderName}>{m.sender?.name}</p>
                    )}
                    <p className={styles.bubbleBody}>{m.body}</p>
                    <p className={styles.bubbleTime}>{formatTime(m.created_at)}</p>
                  </div>
                </div>
              );
            })}
          </div>
        ))}
        <div ref={bottomRef} />
      </main>

      {/* 送信フォーム */}
      <form className={styles.inputArea} onSubmit={handleSend}>
        <textarea
          className={styles.textarea}
          value={body}
          onChange={(e) => setBody(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="メッセージを入力（⌘+Enter で送信）"
          rows={1}
          disabled={sending}
        />
        <button
          type="submit"
          className={styles.sendBtn}
          disabled={!body.trim() || sending}
        >
          送信
        </button>
      </form>
    </div>
  );
}
