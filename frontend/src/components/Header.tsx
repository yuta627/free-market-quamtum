import { Bell, Search } from "lucide-react";
import styles from "./Header.module.css";

interface HeaderProps {
  onSearch?: (q: string) => void;
}

export default function Header({ onSearch }: HeaderProps) {
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
      <button className={styles.notifBtn} aria-label="お知らせ">
        <Bell size={22} />
      </button>
    </header>
  );
}
