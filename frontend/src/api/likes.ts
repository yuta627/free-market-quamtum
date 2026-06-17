import client from "./client";
import { type Product } from "./products";

export interface LikeHistoryItem {
  product: Product;
  liked: boolean;
}

export const toggleLike = (productId: number) =>
  client
    .post<{ liked: boolean }>(`/products/${productId}/like`)
    .then((r) => r.data.liked);

// 一度でもいいねした商品の履歴（解除済みも含む）を返す。
export const listLikeHistory = () =>
  client
    .get<{ items: LikeHistoryItem[] }>("/likes")
    .then((r) => r.data.items);
