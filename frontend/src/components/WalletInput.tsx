import React, { useState } from 'react';
import { addressValidationError } from '../utils/ethereum';

interface WalletInputProps {
  onSubmit: (address: string) => void;
  isLoading: boolean;
}

const DEMO_ADDRESS = '0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae';

export function WalletInput({ onSubmit, isLoading }: WalletInputProps) {
  const [address, setAddress] = useState('');
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = address.trim();
    const validationError = addressValidationError(trimmed);
    if (validationError) {
      setError(validationError);
      return;
    }
    setError(null);
    onSubmit(trimmed.toLowerCase());
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setAddress(e.target.value);
    if (error) setError(null);
  };

  const loadDemo = () => {
    setAddress(DEMO_ADDRESS);
    setError(null);
    onSubmit(DEMO_ADDRESS);
  };

  return (
    <div className="wallet-input-card">
      <h1 className="app-title">DeFi Portfolio Tracker</h1>
      <p className="app-subtitle">
        Enter an Ethereum wallet address to view its on-chain holdings and transaction history.
      </p>

      <form onSubmit={handleSubmit} className="wallet-form">
        <div className="input-group">
          <input
            type="text"
            value={address}
            onChange={handleChange}
            placeholder="0x..."
            className={`address-input ${error ? 'input-error' : ''}`}
            disabled={isLoading}
            spellCheck={false}
            autoComplete="off"
          />
          <button
            type="submit"
            className="btn btn-primary"
            disabled={isLoading || !address.trim()}
          >
            {isLoading ? 'Loading...' : 'Track Portfolio'}
          </button>
        </div>
        {error && <p className="error-message">{error}</p>}
      </form>

      <div className="demo-hint">
        <span>No address handy? </span>
        <button className="btn-link" onClick={loadDemo} disabled={isLoading}>
          Load demo wallet
        </button>
      </div>
    </div>
  );
}
