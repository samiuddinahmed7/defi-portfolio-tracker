import { useState } from 'react';
import { WalletInput } from '../components/WalletInput';
import { PortfolioOverview } from '../components/PortfolioOverview';
import { TokenAllocation } from '../components/TokenAllocation';
import { HoldingsTable } from '../components/HoldingsTable';
import { TransactionHistory } from '../components/TransactionHistory';
import { LoadingState, ErrorState } from '../components/LoadingState';
import { usePortfolio } from '../hooks/usePortfolio';

export function Dashboard() {
  const [address, setAddress] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const { portfolio, transactions, fetchPortfolio, fetchMoreTransactions, reset } = usePortfolio();

  const handleSearch = (addr: string) => {
    setAddress(addr);
    setCurrentPage(1);
    fetchPortfolio(addr);
  };

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
    fetchMoreTransactions(address, page);
  };

  const handleReset = () => {
    setAddress('');
    setCurrentPage(1);
    reset();
  };

  const isInitial = portfolio.status === 'idle';
  const isLoading = portfolio.status === 'loading';
  const hasError = portfolio.status === 'error';
  const hasData = portfolio.status === 'success' && portfolio.data;

  return (
    <div className="dashboard">
      <nav className="navbar">
        <div className="navbar-brand" onClick={handleReset} role="button" tabIndex={0}>
          <span className="navbar-logo" aria-hidden="true">📊</span>
          <span className="navbar-title">DeFi Portfolio Tracker</span>
        </div>
        {hasData && (
          <button className="btn btn-ghost" onClick={handleReset}>
            Search another wallet
          </button>
        )}
      </nav>

      <main className="main-content">
        {isInitial && (
          <div className="hero">
            <WalletInput onSubmit={handleSearch} isLoading={false} />
          </div>
        )}

        {isLoading && (
          <LoadingState />
        )}

        {hasError && (
          <ErrorState
            message={portfolio.error ?? 'An unknown error occurred.'}
            onRetry={handleReset}
          />
        )}

        {hasData && portfolio.data && (
          <div className="portfolio-layout">
            {/* Top: search bar for switching wallets */}
            <div className="search-bar-compact">
              <WalletInput
                onSubmit={handleSearch}
                isLoading={portfolio.status === 'loading'}
              />
            </div>

            <PortfolioOverview portfolio={portfolio.data} />

            <div className="two-col">
              <TokenAllocation portfolio={portfolio.data} />
              <HoldingsTable portfolio={portfolio.data} />
            </div>

            {transactions.status === 'loading' && (
              <LoadingState message="Loading transactions..." />
            )}
            {transactions.status === 'error' && (
              <ErrorState message={transactions.error ?? 'Failed to load transactions.'} />
            )}
            {transactions.status === 'success' && transactions.data && (
              <TransactionHistory
                txPage={transactions.data}
                walletAddress={address}
                isLoading={transactions.status === 'loading'}
                onPageChange={handlePageChange}
              />
            )}
          </div>
        )}
      </main>

      <footer className="footer">
        <p>
          Personal learning project · Data from Etherscan, Alchemy, and Chainlink price feeds ·
          Not financial advice
        </p>
      </footer>
    </div>
  );
}
