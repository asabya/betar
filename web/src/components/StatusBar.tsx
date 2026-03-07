import { useStatus } from '../hooks/useApi';
import { Circle, Users, Wallet } from 'lucide-react';

export function StatusBar() {
  const { data: status } = useStatus();

  return (
    <header className="bg-[var(--color-surface)] border-b border-[var(--color-border)] px-4 lg:px-6 py-3 flex items-center justify-between gap-4 flex-wrap">
      <div className="flex items-center gap-2">
        <Circle
          size={10}
          className={status ? 'fill-[var(--color-success)] text-[var(--color-success)]' : 'fill-[var(--color-error)] text-[var(--color-error)]'}
        />
        <span className="text-sm text-[var(--color-text-muted)]">
          {status ? 'Connected' : 'Disconnected'}
        </span>
      </div>

      <div className="flex items-center gap-6 text-sm text-[var(--color-text-muted)]">
        {status && (
          <>
            <span className="flex items-center gap-1.5" title="Connected peers">
              <Users size={14} />
              {status.connectedPeers}
            </span>
            <span className="flex items-center gap-1.5 font-mono text-xs" title="Wallet address">
              <Wallet size={14} />
              {status.walletAddress
                ? `${status.walletAddress.slice(0, 6)}...${status.walletAddress.slice(-4)}`
                : 'N/A'}
            </span>
          </>
        )}
      </div>
    </header>
  );
}
