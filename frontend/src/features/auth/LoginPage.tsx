import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { login } from "../../api/auth";
import { useAuth } from "./AuthContext";

export default function LoginPage() {
  const navigate = useNavigate();
  const { setAuth } = useAuth();
  const [form, setForm] = useState({ email: "", password: "" });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const data = await login(form);
      setAuth(data.token, data.user);
      navigate("/");
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data
          ?.error ?? "ログインに失敗しました";
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1>ログイン</h1>
        {error && <p className="auth-error">{error}</p>}
        <form onSubmit={handleSubmit}>
          <label>
            メールアドレス
            <input
              type="email"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
              required
            />
          </label>
          <label>
            パスワード
            <input
              type="password"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              required
            />
          </label>
          <button type="submit" disabled={loading}>
            {loading ? "処理中..." : "ログイン"}
          </button>
        </form>
        <p className="auth-link">
          アカウントをお持ちでない方は <Link to="/signup">新規登録</Link>
        </p>
      </div>
    </div>
  );
}
