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

      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-6 max-w-lg">
        <div className="flex items-center gap-3 mb-6">
          <div className="p-3 rounded-xl bg-[var(--color-primary)]/10">
            <Wallet size={24} className="text-[var(--color-primary)]" />
          </div>
          <div>
            <p className="text-lg font-semibold">Ethereum Wallet</p>
            <p className="text-xs text-[var(--color-text-muted)]">Base Sepolia Network</p>
          </div>
        </div>

        {/* Address */}
        <div className="mb-6">
          <p className="text-xs text-[var(--color-text-muted)] mb-1">Address</p>
          {isLoading ? (
            <Skeleton className="h-5 w-full" />
          ) : (
            <div className="flex items-center gap-2">
              <p className="font-mono text-sm break-all">{balance?.address || status?.walletAddress || 'N/A'}</p>
              {(balance?.address || status?.walletAddress) && (
                <button
                  onClick={() => navigator.clipboard.writeText(balance?.address || status?.walletAddress || '')}
                  className="shrink-0 text-[var(--color-text-muted)] hover:text-[var(--color-text)] p-1"
                  title="Copy address"
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
          <div className="bg-[var(--color-bg)] rounded-lg p-4 flex items-center justify-between">
            <div>
              <p className="text-xs text-[var(--color-text-muted)]">Balance</p>
              {isLoading ? (
                <Skeleton className="h-8 w-32 mt-1" />
              ) : (
                <p className="text-2xl font-bold">
                  {balance ? balance.balance.toFixed(6) : '0.000000'}
                </p>
              )}
            </div>
            <span className="text-sm text-[var(--color-text-muted)] font-medium">ETH</span>
          </div>

          {/* USDC Balance */}
          <div className="bg-[var(--color-bg)] rounded-lg p-4 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-green-500/10">
                <DollarSign size={18} className="text-green-400" />
              </div>
              <div>
                <p className="text-xs text-[var(--color-text-muted)]">USDC Balance</p>
                {isLoading ? (
                  <Skeleton className="h-8 w-32 mt-1" />
                ) : (
                  <p className="text-2xl font-bold">
                    {balance?.usdcBalance !== undefined ? balance.usdcBalance.toFixed(6) : '0.000000'}
                  </p>
                )}
              </div>
            </div>
            <span className="text-sm text-[var(--color-text-muted)] font-medium">USDC</span>
          </div>
        </div>

        {/* Explorer link */}
        {(balance?.address || status?.walletAddress) && (
          <a
            href={`https://sepolia.basescan.org/address/${balance?.address || status?.walletAddress}`}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1 text-sm text-[var(--color-primary)] hover:underline mt-4"
          >
            View on BaseScan <ExternalLink size={14} />
          </a>
        )}
      </div>
    </div>
  );
}
