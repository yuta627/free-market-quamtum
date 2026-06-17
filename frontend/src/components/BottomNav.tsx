import { NavLink, useNavigate } from "react-router-dom";
import { Home, Heart, PlusCircle, User } from "lucide-react";
import { useAuth } from "../features/auth/AuthContext";
import styles from "./BottomNav.module.css";

const NAV_ITEMS = [
  { to: "/", icon: Home, label: "ホーム" },
  { to: "/likes", icon: Heart, label: "いいね" },
  { to: "/mypage", icon: User, label: "マイページ" },
];

export default function BottomNav() {
  const { user } = useAuth();
  const navigate = useNavigate();

  const handleSell = () => {
    if (!user) {
      navigate("/login");
    } else {
      navigate("/sell");
    }
  };

  return (
    <nav className={styles.nav}>
      {NAV_ITEMS.slice(0, 2).map(({ to, icon: Icon, label }) => (
        <NavLink
          key={to}
          to={to}
          end={to === "/"}
          className={({ isActive }) =>
            `${styles.item} ${isActive ? styles.active : ""}`
          }
        >
          <Icon size={22} />
          <span>{label}</span>
        </NavLink>
      ))}

      <button className={styles.sellBtn} onClick={handleSell} aria-label="出品する">
        <PlusCircle size={32} strokeWidth={1.8} />
        <span>出品</span>
      </button>

      {NAV_ITEMS.slice(2).map(({ to, icon: Icon, label }) => (
        <NavLink
          key={to}
          to={to}
          className={({ isActive }) =>
            `${styles.item} ${isActive ? styles.active : ""}`
          }
        >
          <Icon size={22} />
          <span>{label}</span>
        </NavLink>
      ))}
    </nav>
  );
}
