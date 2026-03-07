import { useWalletBalance, useStatus } from '../hooks/useApi';
import { Wallet, Copy, ExternalLink } from 'lucide-react';

export function WalletPage() {
  const { data: balance, isLoading } = useWalletBalance();
  const { data: status } = useStatus();

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
        </div>

        {/* Balance */}
        <div className="space-y-3">
          <div className="bg-[var(--color-bg)] rounded-lg p-4 flex items-center justify-between">
            <div>
              <p className="text-xs text-[var(--color-text-muted)]">Balance</p>
              <p className="text-2xl font-bold">
                {isLoading ? '...' : balance ? balance.balance.toFixed(6) : '0.000000'}
              </p>
            </div>
            <span className="text-sm text-[var(--color-text-muted)] font-medium">ETH</span>
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
