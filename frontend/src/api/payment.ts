import client from "./client";
import { type Product } from "./products";

export interface CheckoutResponse {
  client_secret: string;
  payment_intent_id: string;
}

export const createCheckout = (productId: number) =>
  client
    .post<CheckoutResponse>(`/products/${productId}/checkout`)
    .then((r) => r.data);

export const confirmPurchase = (productId: number, paymentIntentId: string) =>
  client
    .post<Product>(`/products/${productId}/confirm-purchase`, {
      payment_intent_id: paymentIntentId,
    })
    .then((r) => r.data);
