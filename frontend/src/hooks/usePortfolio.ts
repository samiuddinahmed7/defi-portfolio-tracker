import { useState, useCallback } from 'react';
import { api, ApiError } from '../services/api';
import type { AsyncState, PortfolioSummary, TransactionPage } from '../types';

/**
 * usePortfolio manages all async state for a single wallet address.
 * It fetches portfolio data and transactions in parallel, then merges
 * the results into a unified state object.
 */
export function usePortfolio() {
  const [portfolio, setPortfolio] = useState<AsyncState<PortfolioSummary>>({
    data: null,
    status: 'idle',
    error: null,
  });

  const [transactions, setTransactions] = useState<AsyncState<TransactionPage>>({
    data: null,
    status: 'idle',
    error: null,
  });

  const fetchPortfolio = useCallback(async (address: string) => {
    setPortfolio({ data: null, status: 'loading', error: null });
    setTransactions({ data: null, status: 'loading', error: null });

    // Fetch portfolio and transactions in parallel.
    const [portfolioResult, txResult] = await Promise.allSettled([
      api.getPortfolio(address),
      api.getTransactions(address, 1, 20),
    ]);

    if (portfolioResult.status === 'fulfilled') {
      setPortfolio({ data: portfolioResult.value, status: 'success', error: null });
    } else {
      const err = portfolioResult.reason;
      const message = err instanceof ApiError
        ? `Provider error: ${err.message}`
        : 'Failed to load portfolio. Check the wallet address and try again.';
      setPortfolio({ data: null, status: 'error', error: message });
    }

    if (txResult.status === 'fulfilled') {
      setTransactions({ data: txResult.value, status: 'success', error: null });
    } else {
      const err = txResult.reason;
      const message = err instanceof ApiError ? err.message : 'Failed to load transactions.';
      setTransactions({ data: null, status: 'error', error: message });
    }
  }, []);

  const fetchMoreTransactions = useCallback(async (address: string, page: number) => {
    setTransactions((prev) => ({ ...prev, status: 'loading' }));
    try {
      const page_data = await api.getTransactions(address, page, 20);
      setTransactions({ data: page_data, status: 'success', error: null });
    } catch (err) {
      const message = err instanceof ApiError ? err.message : 'Failed to load transactions.';
      setTransactions((prev) => ({ ...prev, status: 'error', error: message }));
    }
  }, []);

  const reset = useCallback(() => {
    setPortfolio({ data: null, status: 'idle', error: null });
    setTransactions({ data: null, status: 'idle', error: null });
  }, []);

  return {
    portfolio,
    transactions,
    fetchPortfolio,
    fetchMoreTransactions,
    reset,
  };
}
