import { usePeers, useStatus } from '../hooks/useApi';
import { Users, Network } from 'lucide-react';

export function Peers() {
  const { data: peers } = usePeers();
  const { data: status } = useStatus();

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between flex-wrap gap-4">
        <h1 className="text-2xl font-bold">Peers</h1>
        <span className="flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
          <Network size={16} className="text-[var(--color-primary)]" />
          {status?.connectedPeers ?? 0} connected
        </span>
      </div>

      {peers && peers.length > 0 ? (
        <div className="space-y-3">
          {peers.map((peer) => (
            <div key={peer.id} className="glass-card p-4">
              <p className="font-mono text-sm break-all mb-2">{peer.id}</p>
              {peer.addrs && peer.addrs.length > 0 && (
                <div className="space-y-1">
                  {peer.addrs.map((addr, i) => (
                    <p key={i} className="font-mono text-xs text-[var(--color-text-dim)] bg-[var(--color-bg)] rounded-lg px-2.5 py-1.5 border border-[var(--color-border)]">
                      {addr}
                    </p>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div className="text-center py-12 text-[var(--color-text-muted)] glass-card">
          <Users size={40} className="mx-auto mb-3 opacity-20" />
          <p>No connected peers</p>
        </div>
      )}
    </div>
  );
}
