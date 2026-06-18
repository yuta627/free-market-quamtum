import { useEffect, useState } from "react";
import { loadStripe } from "@stripe/stripe-js";
import {
  Elements,
  PaymentElement,
  useStripe,
  useElements,
} from "@stripe/react-stripe-js";
import { createCheckout, confirmPurchase } from "../api/payment";
import { type Product } from "../api/products";
import styles from "./CheckoutModal.module.css";

const stripePromise = loadStripe(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY ?? "");

interface CheckoutModalProps {
  product: Product;
  onClose: () => void;
  onSuccess: (updated: Product) => void;
}

export default function CheckoutModal({ product, onClose, onSuccess }: CheckoutModalProps) {
  const [clientSecret, setClientSecret] = useState("");
  const [paymentIntentId, setPaymentIntentId] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    createCheckout(product.id)
      .then((res) => {
        setClientSecret(res.client_secret);
        setPaymentIntentId(res.payment_intent_id);
      })
      .catch((err: unknown) => {
        const msg =
          (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
          "決済の準備に失敗しました";
        setError(msg);
      })
      .finally(() => setLoading(false));
  }, [product.id]);

  return (
    <div className={styles.overlay} onClick={onClose}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <div className={styles.header}>
          <h2>お支払い</h2>
          <button className={styles.closeBtn} onClick={onClose} aria-label="閉じる">×</button>
        </div>

        <div className={styles.summary}>
          <p className={styles.productTitle}>{product.title}</p>
          <p className={styles.productPrice}>¥{product.price.toLocaleString()}</p>
        </div>

        {loading && <p className={styles.status}>決済を準備しています...</p>}
        {error && <p className={styles.error}>{error}</p>}

        {clientSecret && (
          <Elements stripe={stripePromise} options={{ clientSecret }}>
            <PaymentForm
              productId={product.id}
              paymentIntentId={paymentIntentId}
              onSuccess={onSuccess}
            />
          </Elements>
        )}
      </div>
    </div>
  );
}

function PaymentForm({
  productId,
  paymentIntentId,
  onSuccess,
}: {
  productId: number;
  paymentIntentId: string;
  onSuccess: (updated: Product) => void;
}) {
  const stripe = useStripe();
  const elements = useElements();
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!stripe || !elements) return;

    setSubmitting(true);
    setError("");

    const { error: confirmError } = await stripe.confirmPayment({
      elements,
      confirmParams: {
        return_url: `${window.location.origin}/payment-complete?product_id=${productId}`,
      },
      redirect: "if_required",
    });

    if (confirmError) {
      setError(confirmError.message ?? "決済に失敗しました");
      setSubmitting(false);
      return;
    }

    try {
      const updated = await confirmPurchase(productId, paymentIntentId);
      onSuccess(updated);
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
        "購入の確定に失敗しました";
      setError(msg);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className={styles.form}>
      <PaymentElement options={{ layout: "tabs" }} />
      {error && <p className={styles.error}>{error}</p>}
      <button type="submit" className={styles.payBtn} disabled={!stripe || submitting}>
        {submitting ? "処理中..." : "支払う"}
      </button>
    </form>
  );
}
