import { useNavigate } from "react-router-dom";
import { useAuth } from "../features/auth/AuthContext";

export default function HomePage() {
  const { user, clearAuth } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    clearAuth();
    navigate("/login");
  };

  return (
    <div style={{ padding: "2rem" }}>
      <h1>フリマアプリ</h1>
      {user ? (
        <>
          <p>ようこそ、<strong>{user.name}</strong> さん！</p>
          <button onClick={handleLogout}>ログアウト</button>
        </>
      ) : (
        <p>ログインしていません</p>
      )}
    </div>
  );
}
