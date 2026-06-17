import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { signup } from "../../api/auth";
import { useAuth } from "./AuthContext";

export default function SignupPage() {
  const navigate = useNavigate();
  const { setAuth } = useAuth();
  const [form, setForm] = useState({ name: "", email: "", password: "" });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const data = await signup(form);
      setAuth(data.token, data.user);
      navigate("/");
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data
          ?.error ?? "登録に失敗しました";
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1>新規登録</h1>
        {error && <p className="auth-error">{error}</p>}
        <form onSubmit={handleSubmit}>
          <label>
            名前
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
              minLength={1}
              maxLength={100}
            />
          </label>
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
            パスワード（8文字以上）
            <input
              type="password"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              required
              minLength={8}
            />
          </label>
          <button type="submit" disabled={loading}>
            {loading ? "処理中..." : "登録する"}
          </button>
        </form>
        <p className="auth-link">
          すでにアカウントをお持ちの方は <Link to="/login">ログイン</Link>
        </p>
      </div>
    </div>
  );
}
