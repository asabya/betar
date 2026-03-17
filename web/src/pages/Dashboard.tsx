import { memo } from 'react';
import { useStatus, useLocalAgents, useAgents, useWorkflows } from '../hooks/useApi';
import { Bot, Globe, GitBranch, Network, Copy } from 'lucide-react';
import { StatSkeleton, Skeleton } from '../components/Skeleton';
import { ErrorState } from '../components/ErrorState';
import { formatRelative } from '../utils/format';

const StatCard = memo(function StatCard({ label, value, icon: Icon }: { label: string; value: string | number; icon: React.ElementType }) {
  return (
    <div className="glass-card p-4 flex items-center gap-4">
      <div className="p-2.5 rounded-xl bg-[var(--color-primary-subtle)] border border-[var(--color-border)]">
        <Icon size={22} className="text-[var(--color-primary)]" />
      </div>
      <div>
        <p className="text-sm text-[var(--color-text-muted)]">{label}</p>
        <p className="text-xl font-semibold" style={{ fontFamily: 'var(--font-display)' }}>{value}</p>
      </div>
    </div>
  );
});

export function Dashboard() {
  const { data: status, isLoading: statusLoading, error: statusError, refetch: refetchStatus } = useStatus();
  const { data: localAgents, isLoading: agentsLoading } = useLocalAgents();
  const { data: networkAgents } = useAgents();
  const { data: workflows } = useWorkflows();

  const activeWorkflows = workflows?.filter((w) => w.status === 'running').length ?? 0;

  if (statusError) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <ErrorState message="Failed to connect to node" onRetry={() => refetchStatus()} />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>

      {/* Stats grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {statusLoading || agentsLoading ? (
          <>
            <StatSkeleton />
            <StatSkeleton />
            <StatSkeleton />
            <StatSkeleton />
          </>
        ) : (
          <>
            <StatCard label="Connected Peers" value={status?.connectedPeers ?? 0} icon={Network} />
            <StatCard label="Local Agents" value={localAgents?.length ?? 0} icon={Bot} />
            <StatCard label="Network Agents" value={networkAgents?.length ?? 0} icon={Globe} />
            <StatCard label="Active Workflows" value={activeWorkflows} icon={GitBranch} />
          </>
        )}
      </div>

      {/* Node info */}
      {statusLoading ? (
        <div className="glass-card p-5 space-y-3">
          <Skeleton className="h-6 w-40 mb-4" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-3/4" />
          <Skeleton className="h-4 w-1/2" />
        </div>
      ) : status && (
        <div className="glass-card p-5">
          <h2 className="text-lg font-semibold mb-4">Node Information</h2>
          <div className="space-y-3 text-sm">
            <InfoRow label="Peer ID" value={status.peerId} mono copyable />
            <InfoRow label="Wallet" value={status.walletAddress || 'N/A'} mono copyable />
            <InfoRow label="Data Directory" value={status.dataDir} mono />
            <div>
              <span className="text-[var(--color-text-muted)]">Addresses</span>
              <div className="mt-1 space-y-1">
                {status.addresses?.map((addr, i) => (
                  <p key={i} className="font-mono text-xs text-[var(--color-text-dim)] bg-[var(--color-bg)] rounded-lg px-2.5 py-1.5 border border-[var(--color-border)]">
                    {addr}
                  </p>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Local agents summary */}
      {localAgents && localAgents.length > 0 && (
        <div className="glass-card p-5">
          <h2 className="text-lg font-semibold mb-4">Local Agents</h2>
          <div className="space-y-2">
            {localAgents.map((agent) => (
              <div key={agent.id} className="flex items-center justify-between bg-[var(--color-bg)] rounded-xl px-4 py-3 border border-[var(--color-border)]">
                <div>
                  <p className="font-medium">{agent.name}</p>
                  <p className="text-xs text-[var(--color-text-dim)] font-mono">{agent.agentID}</p>
                </div>
                <span className="text-sm text-[var(--color-primary)] font-semibold">
                  {agent.price > 0 ? `${agent.price} USDC` : 'Free'}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Active workflows */}
      {workflows && workflows.filter((w) => w.status === 'running').length > 0 && (
        <div className="glass-card p-5">
          <h2 className="text-lg font-semibold mb-4">Active Workflows</h2>
          <div className="space-y-2">
            {workflows
              .filter((w) => w.status === 'running')
              .map((wf) => {
                const completed = wf.steps.filter((s) => s.status === 'completed').length;
                return (
                  <div key={wf.id} className="bg-[var(--color-bg)] rounded-xl px-4 py-3 border border-[var(--color-border)]">
                    <div className="flex justify-between items-center mb-2">
                      <span className="font-mono text-xs">{wf.id.slice(0, 12)}...</span>
                      <span className="text-xs text-[var(--color-primary)]">
                        {completed}/{wf.steps.length} steps &middot; {formatRelative(wf.createdAt)}
                      </span>
                    </div>
                    <div className="w-full bg-[var(--color-border)] rounded-full h-1.5">
                      <div
                        className="bg-[var(--color-primary)] rounded-full h-1.5 transition-all"
                        style={{ width: `${(completed / wf.steps.length) * 100}%` }}
                      />
                    </div>
                  </div>
                );
              })}
          </div>
        </div>
      )}
    </div>
  );
}

const InfoRow = memo(function InfoRow({ label, value, mono, copyable }: { label: string; value: string; mono?: boolean; copyable?: boolean }) {
  return (
    <div className="flex items-start gap-2">
      <span className="text-[var(--color-text-muted)] min-w-[120px] shrink-0">{label}</span>
      <span className={`break-all ${mono ? 'font-mono text-xs' : ''}`}>{value}</span>
      {copyable && value && (
        <button
          onClick={() => navigator.clipboard.writeText(value)}
          className="shrink-0 text-[var(--color-text-dim)] hover:text-[var(--color-primary)] p-2 -m-1.5 transition-colors min-w-[44px] min-h-[44px] flex items-center justify-center"
          title="Copy to clipboard"
          aria-label={`Copy ${label} to clipboard`}
        >
          <Copy size={14} aria-hidden="true" />
        </button>
      )}
    </div>
  );
});
