import client from "./client";
import { type User } from "./auth";

export type ProductCondition = "new" | "like_new" | "good" | "fair" | "poor";
export type ProductStatus = "on_sale" | "sold" | "draft";

export interface Product {
  id: number;
  seller_id: number;
  buyer_id: number | null;
  title: string;
  description: string;
  price: number;
  status: ProductStatus;
  condition: ProductCondition;
  image_urls: string; // JSON string "[]"
  created_at: string;
  updated_at: string;
  seller: User;
}

export interface ListProductsResponse {
  products: Product[];
  total: number;
  limit: number;
  offset: number;
}

export const listProducts = (params?: {
  status?: string;
  q?: string;
  limit?: number;
  offset?: number;
}) =>
  client
    .get<ListProductsResponse>("/products", { params })
    .then((r) => r.data);

export const getProduct = (id: number) =>
  client.get<Product>(`/products/${id}`).then((r) => r.data);

export const createProduct = (data: {
  title: string;
  description: string;
  price: number;
  condition: ProductCondition;
  image_urls?: string[];
}) => client.post<Product>("/products", data).then((r) => r.data);

export const purchaseProduct = (id: number) =>
  client.post<Product>(`/products/${id}/purchase`).then((r) => r.data);

export const listMyProducts = () =>
  client.get<{ products: Product[] }>("/me/products").then((r) => r.data.products);

export const listMyPurchases = () =>
  client.get<{ products: Product[] }>("/me/purchases").then((r) => r.data.products);

export const CONDITION_LABELS: Record<ProductCondition, string> = {
  new: "新品・未使用",
  like_new: "未使用に近い",
  good: "目立った傷や汚れなし",
  fair: "やや傷や汚れあり",
  poor: "傷や汚れあり",
};
