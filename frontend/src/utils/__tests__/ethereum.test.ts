import { describe, it, expect } from 'vitest';
import { isValidAddress, addressValidationError, normalizeAddress } from '../ethereum';

describe('isValidAddress', () => {
  it('accepts a valid lowercase address', () => {
    expect(isValidAddress('0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae')).toBe(true);
  });

  it('accepts a checksummed address', () => {
    expect(isValidAddress('0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2')).toBe(true);
  });

  it('rejects missing 0x prefix', () => {
    expect(isValidAddress('de0b295669a9fd93d5f28d9ec85e40f4cb697bae')).toBe(false);
  });

  it('rejects an address that is too short', () => {
    expect(isValidAddress('0x1234')).toBe(false);
  });

  it('rejects an address that is too long', () => {
    expect(isValidAddress('0x' + 'a'.repeat(42))).toBe(false);
  });

  it('rejects non-hex characters', () => {
    expect(isValidAddress('0xde0b295669a9fd93d5f28d9ec85e40f4cb697bZZ')).toBe(false);
  });

  it('rejects empty string', () => {
    expect(isValidAddress('')).toBe(false);
  });
});

describe('addressValidationError', () => {
  it('returns null for a valid address', () => {
    expect(addressValidationError('0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae')).toBeNull();
  });

  it('returns a descriptive message for empty input', () => {
    const err = addressValidationError('');
    expect(err).not.toBeNull();
    expect(typeof err).toBe('string');
  });

  it('returns a message about 0x prefix if missing', () => {
    const err = addressValidationError('de0b295669a9fd93d5f28d9ec85e40f4cb697bae');
    expect(err).toContain('0x');
  });
});

describe('normalizeAddress', () => {
  it('lowercases a checksummed address', () => {
    const mixed = '0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2';
    expect(normalizeAddress(mixed)).toBe(mixed.toLowerCase());
  });

  it('is idempotent on already lowercase addresses', () => {
    const addr = '0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae';
    expect(normalizeAddress(addr)).toBe(addr);
  });
});
