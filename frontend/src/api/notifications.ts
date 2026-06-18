import client from "./client";

export interface Notification {
  id: number;
  user_id: number;
  title: string;
  body: string;
  is_read: boolean;
  created_at: string;
}

export const listNotifications = () =>
  client.get<{ notifications: Notification[]; unread_count: number }>("/notifications")
    .then((r) => r.data);

export const markRead = (id: number) =>
  client.patch(`/notifications/${id}/read`);

export const markAllRead = () =>
  client.patch("/notifications/read-all");
