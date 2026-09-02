import type { TransactionPage } from '../types';
import {
  formatDate,
  timeAgo,
  shortenAddress,
  formatQuantity,
  etherscanTxLink,
} from '../utils/format';

interface TransactionHistoryProps {
  txPage: TransactionPage;
  walletAddress: string;
  isLoading: boolean;
  onPageChange: (page: number) => void;
}

export function TransactionHistory({
  txPage,
  walletAddress,
  isLoading,
  onPageChange,
}: TransactionHistoryProps) {
  const txs = txPage.transactions ?? [];

  if (txs.length === 0 && !isLoading) {
    return (
      <div className="card">
        <h2 className="section-title">Transaction History</h2>
        <p className="empty-state">No transactions found for this address.</p>
      </div>
    );
  }

  return (
    <div className="card">
      <div className="card-header-row">
        <h2 className="section-title">Transaction History</h2>
        {isLoading && <span className="loading-spinner" />}
      </div>

      <div className="table-container">
        <table className="tx-table">
          <thead>
            <tr>
              <th>Hash</th>
              <th>Date</th>
              <th>Type</th>
              <th>Asset</th>
              <th className="text-right">Amount</th>
              <th>From / To</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {txs.map((tx) => {
              const isSend = tx.from?.toLowerCase() === walletAddress.toLowerCase();
              const isTokenTransfer = tx.type === 'token_transfer';

              const assetSymbol = isTokenTransfer ? (tx.token_symbol ?? '?') : 'ETH';
              const amount = isTokenTransfer
                ? parseFloat(tx.token_value ?? '0') / Math.pow(10, tx.token_decimals ?? 18)
                : tx.normal_value;

              return (
                <tr key={`${tx.hash}-${tx.type}`}>
                  <td>
                    <a
                      href={etherscanTxLink(tx.hash)}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="tx-link"
                      title={tx.hash}
                    >
                      {tx.hash.slice(0, 10)}...
                      <span className="external-icon" aria-hidden="true"> ↗</span>
                    </a>
                  </td>
                  <td title={formatDate(tx.timestamp)}>
                    <span className="date-cell">{timeAgo(tx.timestamp)}</span>
                  </td>
                  <td>
                    <span className={`tx-type-badge tx-type-${tx.type}`}>
                      {tx.type.replace('_', ' ')}
                    </span>
                  </td>
                  <td className="asset-sym">{assetSymbol}</td>
                  <td className={`text-right mono ${isSend ? 'amount-out' : 'amount-in'}`}>
                    {isSend ? '-' : '+'}{formatQuantity(Math.abs(amount), assetSymbol)}
                  </td>
                  <td>
                    <div className="address-pair">
                      <span className="addr-label">From:</span>
                      <span className="addr-value" title={tx.from}>
                        {shortenAddress(tx.from)}
                      </span>
                      <span className="addr-label">To:</span>
                      <span className="addr-value" title={tx.to}>
                        {shortenAddress(tx.to)}
                      </span>
                    </div>
                  </td>
                  <td>
                    <span className={`status-badge status-${tx.status}`}>
                      {tx.status}
                    </span>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      <div className="pagination">
        <button
          className="btn btn-secondary"
          disabled={txPage.page <= 1 || isLoading}
          onClick={() => onPageChange(txPage.page - 1)}
        >
          Previous
        </button>
        <span className="page-info">
          Page {txPage.page} · {txs.length} transactions
        </span>
        <button
          className="btn btn-secondary"
          disabled={!txPage.has_next || isLoading}
          onClick={() => onPageChange(txPage.page + 1)}
        >
          Next
        </button>
      </div>
    </div>
  );
}
