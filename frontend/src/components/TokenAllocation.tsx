import type { PortfolioSummary } from '../types';
import { formatUSD } from '../utils/format';

interface TokenAllocationProps {
  portfolio: PortfolioSummary;
}

// Colour palette for the allocation bars. Uses CSS custom properties so both
// light and dark themes get readable colours.
const COLOURS = [
  '#6366f1', // indigo
  '#8b5cf6', // violet
  '#06b6d4', // cyan
  '#10b981', // emerald
  '#f59e0b', // amber
  '#ef4444', // red
  '#3b82f6', // blue
  '#ec4899', // pink
];

interface Slice {
  symbol: string;
  name: string;
  valueUSD: number;
  percentage: number;
  colour: string;
}

function buildSlices(portfolio: PortfolioSummary): Slice[] {
  if (portfolio.total_usd === 0) return [];

  const items: Array<{ symbol: string; name: string; valueUSD: number }> = [];

  // Add native ETH.
  if (portfolio.native_usd > 0) {
    items.push({ symbol: 'ETH', name: 'Ether', valueUSD: portfolio.native_usd });
  }

  // Add tokens.
  for (const token of portfolio.tokens) {
    if (token.value_usd > 0) {
      items.push({ symbol: token.symbol, name: token.name, valueUSD: token.value_usd });
    }
  }

  // Sort by value descending.
  items.sort((a, b) => b.valueUSD - a.valueUSD);

  // Group small holdings into "Other".
  const threshold = portfolio.total_usd * 0.02; // < 2% → Other
  const main = items.filter((i) => i.valueUSD >= threshold);
  const other = items.filter((i) => i.valueUSD < threshold);
  const otherTotal = other.reduce((s, i) => s + i.valueUSD, 0);

  const slices: Slice[] = main.map((item, idx) => ({
    symbol: item.symbol,
    name: item.name,
    valueUSD: item.valueUSD,
    percentage: (item.valueUSD / portfolio.total_usd) * 100,
    colour: COLOURS[idx % COLOURS.length],
  }));

  if (otherTotal > 0) {
    slices.push({
      symbol: 'Other',
      name: 'Other tokens',
      valueUSD: otherTotal,
      percentage: (otherTotal / portfolio.total_usd) * 100,
      colour: '#94a3b8',
    });
  }

  return slices;
}

export function TokenAllocation({ portfolio }: TokenAllocationProps) {
  const slices = buildSlices(portfolio);

  if (slices.length === 0) {
    return (
      <div className="card">
        <h2 className="section-title">Token Allocation</h2>
        <p className="empty-state">No token holdings to display.</p>
      </div>
    );
  }

  return (
    <div className="card">
      <h2 className="section-title">Token Allocation</h2>
      <div className="allocation-container">
        {/* Stacked bar chart */}
        <div className="allocation-bar" role="img" aria-label="Token allocation chart">
          {slices.map((slice) => (
            <div
              key={slice.symbol}
              className="allocation-segment"
              style={{ width: `${slice.percentage}%`, backgroundColor: slice.colour }}
              title={`${slice.symbol}: ${slice.percentage.toFixed(1)}%`}
            />
          ))}
        </div>

        {/* Legend */}
        <div className="allocation-legend">
          {slices.map((slice) => (
            <div key={slice.symbol} className="legend-item">
              <span
                className="legend-dot"
                style={{ backgroundColor: slice.colour }}
              />
              <span className="legend-symbol">{slice.symbol}</span>
              <span className="legend-pct">{slice.percentage.toFixed(1)}%</span>
              <span className="legend-value">{formatUSD(slice.valueUSD)}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
