/**
 * Ethereum address utilities for the frontend.
 * The backend also validates addresses, but validating client-side first
 * avoids a round trip and gives faster feedback to the user.
 */

const HEX_ADDRESS_RE = /^0x[0-9a-fA-F]{40}$/;

/**
 * Returns true if the string is a well-formed Ethereum address.
 */
export function isValidAddress(address: string): boolean {
  return HEX_ADDRESS_RE.test(address);
}

/**
 * Returns a descriptive validation error message, or null if valid.
 */
export function addressValidationError(address: string): string | null {
  if (!address) return 'Wallet address is required';
  if (!address.startsWith('0x')) return 'Address must start with 0x';
  if (address.length !== 42) return `Address must be 42 characters (got ${address.length})`;
  if (!HEX_ADDRESS_RE.test(address)) return 'Address contains invalid characters';
  return null;
}

/**
 * Returns the lowercase form of an address for consistent comparisons.
 */
export function normalizeAddress(address: string): string {
  return address.toLowerCase();
}
