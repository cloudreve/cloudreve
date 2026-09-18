import { PaginationResults } from "./explorer.ts";

export interface ListDavAccountsService {
  page_size: number;
  next_page_token?: string;
}

export interface DavAccount {
  id: string;
  created_at: string;
  name: string;
  uri: string;
  password: string;
  options?: string;
}

export interface ListDavAccountsResponse {
  accounts: DavAccount[];
  pagination?: PaginationResults;
}

export const DavAccountOption = {
  readonly: 0,
  proxy: 1,
  disable_sys_files: 2,
};

export interface CreateDavAccountService {
  name: string;
  uri: string;
  readonly?: boolean;
  proxy?: boolean;
  disable_sys_files?: boolean;
}
