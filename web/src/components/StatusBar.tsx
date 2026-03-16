import { useStatus } from '../hooks/useApi';
import { Circle, Users, Wallet } from 'lucide-react';

export function StatusBar() {
  const { data: status } = useStatus();

  return (
    <header className="bg-[var(--color-chrome-light)] backdrop-blur-xl border-b border-[var(--color-border)] px-4 lg:px-6 py-3 flex items-center justify-between gap-4 flex-wrap">
      <div className="flex items-center gap-2" role="status" aria-label={status ? 'Node connected' : 'Node disconnected'}>
        <Circle
          size={8}
          aria-hidden="true"
          className={status
            ? 'fill-[var(--color-success)] text-[var(--color-success)]'
            : 'fill-[var(--color-error)] text-[var(--color-error)]'}
          style={status ? { filter: 'drop-shadow(0 0 4px rgba(34,197,94,0.5))' } : undefined}
        />
        <span className="text-sm text-[var(--color-text-muted)]">
          {status ? 'Connected' : 'Disconnected'}
        </span>
      </div>

      <div className="flex items-center gap-6 text-sm text-[var(--color-text-muted)]">
        {status && (
          <>
            <span className="flex items-center gap-1.5" aria-label={`${status.connectedPeers} connected peers`}>
              <Users size={14} aria-hidden="true" />
              {status.connectedPeers}
            </span>
            <span className="flex items-center gap-1.5 font-mono text-xs" aria-label={`Wallet address: ${status.walletAddress || 'not available'}`}>
              <Wallet size={14} aria-hidden="true" />
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
