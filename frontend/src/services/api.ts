import type {
  PortfolioSummary,
  Balance,
  TransactionPage,
  PriceResponse,
} from '../types';

const BASE_URL = import.meta.env.VITE_API_URL ?? '/api/v1';

class ApiError extends Error {
  constructor(
    public status: number,
    public message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

async function fetchJson<T>(url: string): Promise<T> {
  const response = await fetch(url, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (!response.ok) {
    const body = await response.json().catch(() => ({ message: response.statusText }));
    throw new ApiError(response.status, body.message ?? response.statusText);
  }

  return response.json() as Promise<T>;
}

export const api = {
  /**
   * Fetches the full portfolio summary for a wallet address.
   * This is the primary endpoint used by the dashboard.
   */
  getPortfolio(address: string): Promise<PortfolioSummary> {
    return fetchJson(`${BASE_URL}/portfolio/${address}`);
  },

  /**
   * Fetches only the native ETH balance.
   */
  getNativeBalance(address: string): Promise<Balance> {
    return fetchJson(`${BASE_URL}/portfolio/${address}/balance`);
  },

  /**
   * Fetches ERC-20 token balances.
   */
  getTokenBalances(address: string): Promise<{ address: string; tokens: Balance[]; count: number }> {
    return fetchJson(`${BASE_URL}/portfolio/${address}/tokens`);
  },

  /**
   * Fetches paginated transaction history.
   */
  getTransactions(address: string, page = 1, perPage = 20): Promise<TransactionPage> {
    return fetchJson(
      `${BASE_URL}/portfolio/${address}/transactions?page=${page}&per_page=${perPage}`,
    );
  },

  /**
   * Fetches the current USD price for a token symbol.
   */
  getPrice(symbol: string): Promise<PriceResponse> {
    return fetchJson(`${BASE_URL}/prices/${symbol}`);
  },

  /**
   * Health check.
   */
  health(): Promise<{ status: string; version: string; demo: boolean }> {
    return fetchJson(`${BASE_URL}/health`);
  },
};

export { ApiError };
