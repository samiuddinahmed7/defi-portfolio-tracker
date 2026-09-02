import type { PortfolioSummary } from '../types';
import { formatUSD, formatQuantity } from '../utils/format';

interface HoldingsTableProps {
  portfolio: PortfolioSummary;
}

export function HoldingsTable({ portfolio }: HoldingsTableProps) {
  // Build a unified list of holdings, with ETH first.
  const rows = [];

  rows.push({
    symbol: 'ETH',
    name: 'Ether',
    quantity: portfolio.native_balance,
    priceUSD: portfolio.native_usd / (portfolio.native_balance || 1),
    valueUSD: portfolio.native_usd,
    priceSource: 'chainlink',
    isNative: true,
  });

  for (const token of portfolio.tokens) {
    rows.push({
      symbol: token.symbol,
      name: token.name,
      quantity: token.quantity,
      priceUSD: token.price_usd,
      valueUSD: token.value_usd,
      priceSource: token.price_source,
      isNative: false,
    });
  }

  // Sort by USD value descending.
  rows.sort((a, b) => b.valueUSD - a.valueUSD);

  if (rows.length === 0) {
    return (
      <div className="card">
        <h2 className="section-title">Holdings</h2>
        <p className="empty-state">No holdings found for this address.</p>
      </div>
    );
  }

  return (
    <div className="card">
      <h2 className="section-title">Holdings</h2>
      <div className="table-container">
        <table className="holdings-table">
          <thead>
            <tr>
              <th>Asset</th>
              <th className="text-right">Quantity</th>
              <th className="text-right">Price</th>
              <th className="text-right">Value</th>
              <th className="text-right">% Portfolio</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => {
              const pct = portfolio.total_usd > 0
                ? (row.valueUSD / portfolio.total_usd) * 100
                : 0;

              return (
                <tr key={row.symbol}>
                  <td>
                    <div className="asset-cell">
                      <span className="asset-symbol">{row.symbol}</span>
                      <span className="asset-name">{row.name}</span>
                    </div>
                  </td>
                  <td className="text-right mono">
                    {formatQuantity(row.quantity, row.symbol)}
                  </td>
                  <td className="text-right mono">
                    {row.priceUSD > 0 ? formatUSD(row.priceUSD) : (
                      <span className="price-unavailable" title={`Source: ${row.priceSource}`}>
                        N/A
                      </span>
                    )}
                  </td>
                  <td className="text-right mono">{formatUSD(row.valueUSD)}</td>
                  <td className="text-right">
                    <span className="pct-bar-container">
                      <span
                        className="pct-bar-fill"
                        style={{ width: `${Math.min(pct, 100)}%` }}
                      />
                      <span className="pct-label">{pct.toFixed(1)}%</span>
                    </span>
                  </td>
                </tr>
              );
            })}
          </tbody>
          <tfoot>
            <tr className="total-row">
              <td colSpan={3}>Total</td>
              <td className="text-right mono">{formatUSD(portfolio.total_usd)}</td>
              <td />
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  );
}
