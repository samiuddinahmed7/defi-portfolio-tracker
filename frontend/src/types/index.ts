// Central type definitions for the DeFi Portfolio Tracker frontend.
// These mirror the Go backend models.

export interface TokenHolding {
  token_address: string;
  symbol: string;
  name: string;
  quantity: number;
  price_usd: number;
  value_usd: number;
  price_source: string;
}

export interface PortfolioSummary {
  address: string;
  native_balance: number;  // ETH
  native_usd: number;
  tokens: TokenHolding[];
  total_usd: number;
  fetched_at: string;
  is_demo?: boolean;
}

export interface Balance {
  id: number;
  wallet_address: string;
  token_address: string;
  token_symbol: string;
  token_name: string;
  decimals: number;
  raw_balance: string;
  normal_balance: number;
  fetched_at: string;
}

export interface Transaction {
  id: number;
  hash: string;
  wallet_address: string;
  block_number: number;
  timestamp: string;
  from: string;
  to: string;
  value: string;
  normal_value: number;
  gas: number;
  gas_used: number;
  gas_price: string;
  token_address?: string;
  token_symbol?: string;
  token_value?: string;
  token_decimals?: number;
  type: string;  // send | receive | token_transfer
  status: string;
}

export interface TransactionPage {
  transactions: Transaction[];
  total: number;
  page: number;
  per_page: number;
  has_next: boolean;
}

export interface PriceResponse {
  symbol: string;
  price_usd: number;
  source: string;
  fetched_at: string;
}

export interface ApiError {
  error: string;
  message: string;
  code: number;
}

// UI state types
export type LoadingState = 'idle' | 'loading' | 'success' | 'error';

export interface AsyncState<T> {
  data: T | null;
  status: LoadingState;
  error: string | null;
}
