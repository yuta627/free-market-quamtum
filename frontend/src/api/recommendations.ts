import client from "./client";

export interface RecommendedItem {
  item_id: number;
  score: number;
  name: string;
  category: string;
  price: number;
  is_cold_start: boolean;
}

export const getRecommendations = (productId: number, limit = 10) =>
  client
    .get<{ items: RecommendedItem[] }>(`/products/${productId}/recommendations`, {
      params: { limit },
    })
    .then((r) => r.data.items);

export const getClassicalRecommendations = (productId: number, limit = 10) =>
  client
    .get<{ items: RecommendedItem[] }>(`/products/${productId}/recommendations/classical`, {
      params: { limit },
    })
    .then((r) => r.data.items);

export const getQKernelRecommendations = (productId: number, limit = 10) =>
  client
    .get<{ items: RecommendedItem[] }>(`/products/${productId}/recommendations/qkernel`, {
      params: { limit },
    })
    .then((r) => r.data.items);

