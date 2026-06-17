import client from "./client";

export interface Bid {
  id: number;
  auction_id: number;
  bidder_id: number;
  amount: number;
  created_at: string;
  bidder?: { id: number; name: string };
}

export interface Auction {
  id: number;
  product_id: number;
  starting_price: number;
  current_price: number;
  bid_count: number;
  ends_at: string;
  status: string;
  winner_id?: number;
  winner?: { id: number; name: string };
  product: {
    id: number;
    title: string;
    description: string;
    condition: string;
    image_urls: string;
    seller: { id: number; name: string };
  };
  bids?: Bid[];
}

export const listAuctions = (limit = 20, offset = 0) =>
  client.get<{ auctions: Auction[]; total: number }>("/auctions", { params: { limit, offset } })
    .then((r) => r.data);

export const getAuction = (id: number) =>
  client.get<Auction>(`/auctions/${id}`).then((r) => r.data);

export const createAuction = (data: {
  title: string;
  description: string;
  condition: string;
  image_urls: string[];
  starting_price: number;
  ends_at: string;
}) => client.post<Auction>("/auctions", data).then((r) => r.data);

export const placeBid = (auctionId: number, amount: number) =>
  client.post<Auction>(`/auctions/${auctionId}/bid`, { amount }).then((r) => r.data);

export const finalizeAuction = (auctionId: number) =>
  client.post<Auction>(`/auctions/${auctionId}/finalize`).then((r) => r.data);
