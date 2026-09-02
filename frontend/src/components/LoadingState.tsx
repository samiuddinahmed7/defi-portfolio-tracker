export function LoadingState({ message = 'Loading portfolio data...' }: { message?: string }) {
  return (
    <div className="loading-container" role="status" aria-live="polite">
      <div className="loading-spinner-large" aria-hidden="true" />
      <p className="loading-text">{message}</p>
      <p className="loading-sub">Fetching on-chain data from Ethereum</p>
    </div>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="error-container" role="alert">
      <span className="error-icon" aria-hidden="true">⚠</span>
      <h3 className="error-title">Something went wrong</h3>
      <p className="error-message">{message}</p>
      {onRetry && (
        <button className="btn btn-primary" onClick={onRetry}>
          Try again
        </button>
      )}
    </div>
  );
}
