/**
 * Formatting utilities for the portfolio dashboard.
 * Keeping formatting in one place makes it easy to adjust regional settings.
 */

/**
 * Formats a USD value with 2 decimal places and a $ prefix.
 * Values over $1,000 use comma separators for readability.
 */
export function formatUSD(value: number): string {
  if (!isFinite(value) || isNaN(value)) return '$0.00';

  if (Math.abs(value) < 0.01 && value !== 0) {
    return `$${value.toFixed(6)}`;
  }

  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}

/**
 * Formats a token quantity with appropriate decimal places.
 * Small quantities get more decimals; large ones fewer.
 */
export function formatQuantity(value: number, symbol?: string): string {
  if (!isFinite(value) || isNaN(value)) return '0';

  // Stablecoins are usually displayed as whole dollars.
  const stables = new Set(['USDC', 'USDT', 'DAI', 'BUSD', 'FRAX']);
  if (symbol && stables.has(symbol.toUpperCase())) {
    return new Intl.NumberFormat('en-US', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(value);
  }

  if (value === 0) return '0';
  if (Math.abs(value) >= 1000) {
    return new Intl.NumberFormat('en-US', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(value);
  }
  if (Math.abs(value) >= 1) {
    return value.toFixed(4);
  }
  return value.toFixed(6);
}

/**
 * Shortens an Ethereum address to 0x1234...5678 format.
 */
export function shortenAddress(address: string, chars = 4): string {
  if (!address || address.length < 10) return address;
  return `${address.slice(0, 2 + chars)}...${address.slice(-chars)}`;
}

/**
 * Formats a timestamp as a locale-aware date+time string.
 */
export function formatDate(timestamp: string | Date): string {
  const d = typeof timestamp === 'string' ? new Date(timestamp) : timestamp;
  if (isNaN(d.getTime())) return 'Unknown';
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(d);
}

/**
 * Returns a human-readable relative time string ("2 hours ago").
 */
export function timeAgo(timestamp: string | Date): string {
  const d = typeof timestamp === 'string' ? new Date(timestamp) : timestamp;
  const diffMs = Date.now() - d.getTime();
  const diffSecs = Math.floor(diffMs / 1000);
  const diffMins = Math.floor(diffSecs / 60);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffSecs < 60) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  return formatDate(d);
}

/**
 * Builds a link to the Etherscan transaction detail page.
 */
export function etherscanTxLink(hash: string): string {
  return `https://etherscan.io/tx/${hash}`;
}

/**
 * Builds a link to the Etherscan address page.
 */
export function etherscanAddressLink(address: string): string {
  return `https://etherscan.io/address/${address}`;
}
