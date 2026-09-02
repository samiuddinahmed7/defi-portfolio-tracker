import { describe, it, expect } from 'vitest';
import {
  formatUSD,
  formatQuantity,
  shortenAddress,
  timeAgo,
  etherscanTxLink,
  etherscanAddressLink,
} from '../format';

describe('formatUSD', () => {
  it('formats zero correctly', () => {
    expect(formatUSD(0)).toBe('$0.00');
  });

  it('formats whole dollars', () => {
    expect(formatUSD(1000)).toBe('$1,000.00');
  });

  it('formats cents', () => {
    expect(formatUSD(1.5)).toBe('$1.50');
  });

  it('formats large values with commas', () => {
    expect(formatUSD(1234567.89)).toBe('$1,234,567.89');
  });
});

describe('formatQuantity', () => {
  it('formats stablecoin USDC with 2 decimals', () => {
    const result = formatQuantity(1234.5678, 'USDC');
    expect(result).toBe('1,234.57');
  });

  it('formats ETH with up to 6 decimals', () => {
    const result = formatQuantity(1.123456789, 'ETH');
    // ETH is not a stablecoin, should show more precision
    expect(result).toContain('1.12');
  });

  it('formats LINK tokens', () => {
    const result = formatQuantity(500.25, 'LINK');
    expect(result).toContain('500');
  });

  it('formats zero quantity', () => {
    expect(formatQuantity(0, 'ETH')).toBe('0');
  });
});

describe('shortenAddress', () => {
  it('shortens a full Ethereum address', () => {
    const addr = '0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae';
    const short = shortenAddress(addr);
    expect(short).toBe('0xde0b...7bae');
  });

  it('returns short strings unchanged (under 10 chars)', () => {
    expect(shortenAddress('0x1234')).toBe('0x1234');
  });
});

describe('timeAgo', () => {
  it('returns "just now" for very recent timestamps', () => {
    const now = new Date();
    expect(timeAgo(now)).toBe('just now');
  });

  it('returns minutes for recent timestamps', () => {
    const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000);
    expect(timeAgo(fiveMinutesAgo)).toBe('5m ago');
  });

  it('returns hours for older timestamps', () => {
    const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000);
    expect(timeAgo(twoHoursAgo)).toBe('2h ago');
  });

  it('returns days for old timestamps', () => {
    const threeDaysAgo = new Date(Date.now() - 3 * 24 * 60 * 60 * 1000);
    expect(timeAgo(threeDaysAgo)).toBe('3d ago');
  });
});

describe('etherscan links', () => {
  it('generates a correct tx link', () => {
    const hash = '0xabc123';
    expect(etherscanTxLink(hash)).toBe(`https://etherscan.io/tx/${hash}`);
  });

  it('generates a correct address link', () => {
    const addr = '0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae';
    expect(etherscanAddressLink(addr)).toBe(`https://etherscan.io/address/${addr}`);
  });
});
