// API response wrapper
export interface ApiResponse<T> {
  ok: boolean;
  data?: T;
  error_code?: string;
}

// Listing
export type ListingType = 'lessor' | 'lessee';
export type ListingStatus = 'active' | 'inactive';

export interface Listing {
  id: number;
  status: ListingStatus;
  user_id: number;
  channel_id?: number;
  channel_title?: string;
  channel_username?: string;
  channel_followers?: number;
  type: ListingType;
  prices: Record<string, unknown>;
  categories?: string[];
  description?: string;
  created_at: string;
  updated_at: string;
}

// Channel
export interface Channel {
  id: number;
  title: string;
  username?: string;
  photo?: string;
}

// Deal
export type DealStatus =
  | 'draft'
  | 'approved'
  | 'waiting_escrow_deposit'
  | 'escrow_deposited'
  | 'waiting_release'
  | 'released'
  | 'cancelled';

export interface Deal {
  id: number;
  listing_id: number;
  lessor_id: number;
  lessee_id: number;
  type: string;
  duration: number;
  price: number;
  details: Record<string, unknown>;
  lessor_signature?: string;
  lessee_signature?: string;
  status: DealStatus;
  escrow_address?: string;
  escrow_release_time?: string;
  created_at: string;
  updated_at: string;
}

// Deal chat message
export interface DealChat {
  deal_id: number;
  reply_to_chat_id?: number;
  reply_to_message_id?: number;
  replied_message?: string;
  created_at: string;
  updated_at: string;
}
