import client from "./client";

export const generateDescription = (title: string, keywords: string) =>
  client
    .post<{ description: string }>("/ai/generate-description", { title, keywords })
    .then((r) => r.data.description);

export const askProductQuestion = (productId: number, question: string) =>
  client
    .post<{ answer: string }>(`/products/${productId}/ask`, { question })
    .then((r) => r.data.answer);
