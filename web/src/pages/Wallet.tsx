import { useWalletBalance, useStatus } from '../hooks/useApi';
import { Wallet, Copy, ExternalLink, DollarSign } from 'lucide-react';
import { Skeleton } from '../components/Skeleton';
import { ErrorState } from '../components/ErrorState';

export function WalletPage() {
  const { data: balance, isLoading, error, refetch } = useWalletBalance();
  const { data: status } = useStatus();

  if (error) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Wallet</h1>
        <ErrorState message={error.message} onRetry={() => refetch()} />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Wallet</h1>

      <div className="glass-card p-6 max-w-lg">
        <div className="flex items-center gap-3 mb-6">
          <div className="p-3 rounded-xl bg-[var(--color-primary-subtle)] border border-[var(--color-border)]">
            <Wallet size={24} className="text-[var(--color-primary)]" />
          </div>
          <div>
            <p className="text-lg font-semibold" style={{ fontFamily: 'var(--font-display)' }}>Ethereum Wallet</p>
            <p className="text-xs text-[var(--color-text-dim)]">Base Sepolia Network</p>
          </div>
        </div>

        {/* Address */}
        <div className="mb-6">
          <p className="text-xs text-[var(--color-text-muted)] mb-1.5">Address</p>
          {isLoading ? (
            <Skeleton className="h-5 w-full" />
          ) : (
            <div className="flex items-center gap-2">
              <p className="font-mono text-sm break-all">{balance?.address || status?.walletAddress || 'N/A'}</p>
              {(balance?.address || status?.walletAddress) && (
                <button
                  onClick={() => navigator.clipboard.writeText(balance?.address || status?.walletAddress || '')}
                  className="shrink-0 text-[var(--color-text-dim)] hover:text-[var(--color-primary)] p-2 -m-1 transition-colors min-w-[44px] min-h-[44px] flex items-center justify-center"
                  title="Copy address"
                  aria-label="Copy wallet address to clipboard"
                >
                  <Copy size={14} />
                </button>
              )}
            </div>
          )}
        </div>

        {/* Balances */}
        <div className="space-y-3">
          {/* ETH Balance */}
          <div className="bg-[var(--color-bg)] rounded-xl p-4 flex items-center justify-between border border-[var(--color-border)]">
            <div>
              <p className="text-xs text-[var(--color-text-muted)]">Balance</p>
              {isLoading ? (
                <Skeleton className="h-8 w-32 mt-1" />
              ) : (
                <p className="text-2xl font-bold" style={{ fontFamily: 'var(--font-display)' }}>
                  {balance ? balance.balance.toFixed(6) : '0.000000'}
                </p>
              )}
            </div>
            <span className="text-sm text-[var(--color-text-dim)] font-medium">ETH</span>
          </div>

          {/* USDC Balance */}
          <div className="bg-[var(--color-bg)] rounded-xl p-4 flex items-center justify-between border border-[var(--color-border)]">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-xl bg-[var(--color-success)]/10 border border-[var(--color-success)]/20">
                <DollarSign size={18} className="text-[var(--color-success)]" />
              </div>
              <div>
                <p className="text-xs text-[var(--color-text-muted)]">USDC Balance</p>
                {isLoading ? (
                  <Skeleton className="h-8 w-32 mt-1" />
                ) : (
                  <p className="text-2xl font-bold" style={{ fontFamily: 'var(--font-display)' }}>
                    {balance?.usdcBalance !== undefined ? balance.usdcBalance.toFixed(6) : '0.000000'}
                  </p>
                )}
              </div>
            </div>
            <span className="text-sm text-[var(--color-text-dim)] font-medium">USDC</span>
          </div>
        </div>

        {/* Explorer link */}
        {(balance?.address || status?.walletAddress) && (
          <a
            href={`https://sepolia.basescan.org/address/${balance?.address || status?.walletAddress}`}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1 text-sm text-[var(--color-primary)] hover:text-[var(--color-primary-light)] transition-colors mt-5"
          >
            View on BaseScan <ExternalLink size={14} />
          </a>
        )}
      </div>
    </div>
  );
}
