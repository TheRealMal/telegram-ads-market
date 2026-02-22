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
  channel_photo?: string;
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

// Deal (status values match backend entity.DealStatus)
export type DealStatus =
  | 'draft'
  | 'approved'
  | 'waiting_escrow_deposit'
  | 'escrow_deposit_confirmed'
  | 'in_progress'
  | 'waiting_escrow_release'
  | 'escrow_release_confirmed'
  | 'completed'
  | 'waiting_escrow_refund'
  | 'escrow_refund_confirmed'
  | 'expired'
  | 'rejected';

/** User-friendly labels for deal statuses. */
export const DEAL_STATUS_LABEL: Record<DealStatus, string> = {
  draft: 'Draft',
  approved: 'Approved',
  waiting_escrow_deposit: 'Waiting escrow deposit',
  escrow_deposit_confirmed: 'Escrow deposited',
  in_progress: 'In progress',
  waiting_escrow_release: 'Waiting escrow release',
  escrow_release_confirmed: 'Escrow released',
  completed: 'Completed',
  waiting_escrow_refund: 'Waiting escrow refund',
  escrow_refund_confirmed: 'Escrow refunded',
  expired: 'Expired',
  rejected: 'Rejected',
};

export function getDealStatusLabel(status: DealStatus | string): string {
  return DEAL_STATUS_LABEL[status as DealStatus] ?? status;
}

export interface Deal {
  id: number;
  listing_id: number;
  lessor_id: number;
  lessee_id: number;
  channel_id?: number | null;
  type: string;
  duration: number;
  price: number;
  details: Record<string, unknown>;
  lessor_signature?: string;
  lessee_signature?: string;
  lessor_payout_address?: string;
  lessee_payout_address?: string;
  status: DealStatus;
  escrow_address?: string;
  escrow_amount?: number; // nanotons to deposit
  escrow_release_time?: string;
  created_at: string;
  updated_at: string;
}

// Analytics (admin-only)
export interface AnalyticsSnapshot {
  id: number;
  recorded_at: string;
  listings_count: number;
  deals_count: number;
  deals_by_status: Record<string, number>;
  deal_amounts_by_status_ton: Record<string, number>;
  commission_earned_nanoton: number;
  users_count: number;
  avg_listings_per_user: number;
}

export interface LatestSnapshotResponse {
  snapshot: AnalyticsSnapshot | null;
}

export interface AnalyticsHistoryResponse {
  period: string;
  from: string;
  to: string;
  timestamps: number[];
  listings_count: number[];
  deals_count: number[];
  users_count: number[];
  commission_earned_nanoton: number[];
  avg_listings_per_user: number[];
  deals_by_status: Record<string, number[]>;
  deal_amounts_by_status_ton: Record<string, number[]>;
}
