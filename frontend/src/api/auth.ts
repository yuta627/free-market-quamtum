import client from "./client";

export interface User {
  id: number;
  name: string;
  email: string;
  avatar_url: string;
  bio: string;
  postal_code: string;
  prefecture: string;
  city: string;
  address_line: string;
  building: string;
  created_at: string;
  updated_at: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export const getMe = () => client.get<User>("/me").then((r) => r.data);

export const updateAddress = (data: {
  postal_code: string;
  prefecture: string;
  city: string;
  address_line: string;
  building: string;
}) => client.patch<User>("/me/address", data).then((r) => r.data);

export const signup = (data: {
  name: string;
  email: string;
  password: string;
}) => client.post<AuthResponse>("/auth/signup", data).then((r) => r.data);

export const login = (data: { email: string; password: string }) =>
  client.post<AuthResponse>("/auth/login", data).then((r) => r.data);
