import { useEffect, useRef, useState } from "react";
import { Bell, Search } from "lucide-react";
import { useAuth } from "../features/auth/AuthContext";
import { listNotifications, markAllRead, markRead, type Notification } from "../api/notifications";
import styles from "./Header.module.css";

interface HeaderProps {
  onSearch?: (q: string) => void;
}

export default function Header({ onSearch }: HeaderProps) {
  const { user } = useAuth();
  const [unreadCount, setUnreadCount] = useState(0);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!user) { setUnreadCount(0); return; }
    listNotifications()
      .then((res) => {
        setNotifications(res.notifications ?? []);
        setUnreadCount(res.unread_count);
      })
      .catch(() => {});
  }, [user]);

  // パネル外クリックで閉じる
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const handleOpen = async () => {
    setOpen((prev) => !prev);
    if (!open && user && unreadCount > 0) {
      await markAllRead().catch(() => {});
      setUnreadCount(0);
      setNotifications((prev) => prev.map((n) => ({ ...n, is_read: true })));
    }
  };

  const handleMarkRead = async (id: number) => {
    await markRead(id).catch(() => {});
    setNotifications((prev) =>
      prev.map((n) => (n.id === id ? { ...n, is_read: true } : n))
    );
  };

  return (
    <header className={styles.header}>
      <div className={styles.searchWrap}>
        <Search size={16} className={styles.searchIcon} />
        <input
          className={styles.searchInput}
          placeholder="商品を検索"
          onChange={(e) => onSearch?.(e.target.value)}
        />
      </div>

      {user && (
        <div className={styles.notifWrap} ref={ref}>
          <button className={styles.notifBtn} aria-label="お知らせ" onClick={handleOpen}>
            <Bell size={22} />
            {unreadCount > 0 && (
              <span className={styles.badge}>{unreadCount > 9 ? "9+" : unreadCount}</span>
            )}
          </button>

          {open && (
            <div className={styles.panel}>
              <p className={styles.panelTitle}>お知らせ</p>
              {notifications.length === 0 ? (
                <p className={styles.empty}>通知はありません</p>
              ) : (
                notifications.map((n) => (
                  <div
                    key={n.id}
                    className={`${styles.item} ${n.is_read ? styles.itemRead : styles.itemUnread}`}
                    onClick={() => handleMarkRead(n.id)}
                  >
                    <p className={styles.itemTitle}>{n.title}</p>
                    <p className={styles.itemBody}>{n.body}</p>
                    <p className={styles.itemTime}>
                      {new Date(n.created_at).toLocaleString("ja-JP")}
                    </p>
                  </div>
                ))
              )}
            </div>
          )}
        </div>
      )}
    </header>
  );
}
