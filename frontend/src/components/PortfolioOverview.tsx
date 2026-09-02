import type { PortfolioSummary } from '../types';
import { formatUSD, formatQuantity, shortenAddress, etherscanAddressLink } from '../utils/format';

interface PortfolioOverviewProps {
  portfolio: PortfolioSummary;
}

export function PortfolioOverview({ portfolio }: PortfolioOverviewProps) {
  return (
    <div className="portfolio-overview">
      <div className="overview-header">
        <div>
          <h2 className="section-title">Portfolio Overview</h2>
          <a
            href={etherscanAddressLink(portfolio.address)}
            target="_blank"
            rel="noopener noreferrer"
            className="address-link"
            title={portfolio.address}
          >
            {shortenAddress(portfolio.address, 6)}
            <span className="external-icon" aria-hidden="true"> ↗</span>
          </a>
        </div>
        {portfolio.is_demo && (
          <span className="demo-badge">DEMO DATA</span>
        )}
      </div>

      <div className="stats-grid">
        <StatCard
          label="Total Value"
          value={formatUSD(portfolio.total_usd)}
          highlight
        />
        <StatCard
          label="ETH Balance"
          value={`${formatQuantity(portfolio.native_balance)} ETH`}
          subValue={formatUSD(portfolio.native_usd)}
        />
        <StatCard
          label="Token Holdings"
          value={String(portfolio.tokens.length)}
          subValue="ERC-20 tokens"
        />
      </div>
    </div>
  );
}

interface StatCardProps {
  label: string;
  value: string;
  subValue?: string;
  highlight?: boolean;
}

function StatCard({ label, value, subValue, highlight }: StatCardProps) {
  return (
    <div className={`stat-card ${highlight ? 'stat-card-highlight' : ''}`}>
      <span className="stat-label">{label}</span>
      <span className="stat-value">{value}</span>
      {subValue && <span className="stat-sub">{subValue}</span>}
    </div>
  );
}
