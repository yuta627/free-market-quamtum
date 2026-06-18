import { useEffect, useState } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { confirmPurchase } from "../api/payment";

export default function PaymentCompletePage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [status, setStatus] = useState("処理中...");

  useEffect(() => {
    const paymentIntentId = searchParams.get("payment_intent");
    const productId = searchParams.get("product_id");

    if (!paymentIntentId) {
      setStatus("決済情報が見つかりません");
      return;
    }

    if (productId) {
      confirmPurchase(Number(productId), paymentIntentId)
        .then(() => {
          setStatus("購入が完了しました！");
          setTimeout(() => navigate("/"), 2000);
        })
        .catch(() => {
          setStatus("購入の確定に失敗しました。サポートにお問い合わせください。");
        });
    } else {
      setStatus("購入が完了しました！");
      setTimeout(() => navigate("/"), 2000);
    }
  }, [searchParams, navigate]);

  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", height: "100vh", gap: "1rem" }}>
      <p style={{ fontSize: "1.2rem" }}>{status}</p>
    </div>
  );
}
