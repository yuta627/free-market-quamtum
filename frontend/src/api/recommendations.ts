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

export const getQMLRecommendations = (productId: number, limit = 10) =>
  client
    .get<{ items: RecommendedItem[] }>(`/products/${productId}/recommendations/qml`, {
      params: { limit },
    })
    .then((r) => r.data.items);
