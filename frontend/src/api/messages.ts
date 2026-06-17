import client from "./client";
import { type User } from "./auth";

export interface Message {
  id: number;
  product_id: number;
  sender_id: number;
  body: string;
  is_read: boolean;
  created_at: string;
  updated_at: string;
  sender: User;
}

export interface ListMessagesResponse {
  messages: Message[];
}

export const listMessages = (productId: number) =>
  client
    .get<ListMessagesResponse>(`/products/${productId}/messages`)
    .then((r) => r.data);

export const sendMessage = (productId: number, body: string) =>
  client
    .post<Message>(`/products/${productId}/messages`, { body })
    .then((r) => r.data);
