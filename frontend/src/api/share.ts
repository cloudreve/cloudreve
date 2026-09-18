import { User } from "./user.ts";
import { PaginationResults, Share } from "./explorer.ts";

export interface ShareInfo {
  id: string;
  name?: string;
  source_type?: number;
  remain_downloads?: number;
  visited?: number;
  downloaded?: number;
  expires?: string;
  created_at?: string;
  unlocked: boolean;
  owner: User;
  expired?: boolean;
}

export interface ListShareService {
  page_size: number;
  order_by?: string;
  order_direction?: string;
  next_page_token?: string;
}

export interface ListShareResponse {
  shares: Share[];
  pagination: PaginationResults;
}
