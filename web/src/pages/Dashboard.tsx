import { useStatus, useLocalAgents, useAgents, useWorkflows } from '../hooks/useApi';
import { Bot, Globe, GitBranch, Network, Copy } from 'lucide-react';

function StatCard({ label, value, icon: Icon }: { label: string; value: string | number; icon: React.ElementType }) {
  return (
    <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4 flex items-center gap-4">
      <div className="p-2.5 rounded-lg bg-[var(--color-primary)]/10">
        <Icon size={22} className="text-[var(--color-primary)]" />
      </div>
      <div>
        <p className="text-sm text-[var(--color-text-muted)]">{label}</p>
        <p className="text-xl font-semibold">{value}</p>
      </div>
    </div>
  );
}

export function Dashboard() {
  const { data: status } = useStatus();
  const { data: localAgents } = useLocalAgents();
  const { data: networkAgents } = useAgents();
  const { data: workflows } = useWorkflows();

  const activeWorkflows = workflows?.filter((w) => w.status === 'running').length ?? 0;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>

      {/* Stats grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard label="Connected Peers" value={status?.connectedPeers ?? 0} icon={Network} />
        <StatCard label="Local Agents" value={localAgents?.length ?? 0} icon={Bot} />
        <StatCard label="Network Agents" value={networkAgents?.length ?? 0} icon={Globe} />
        <StatCard label="Active Workflows" value={activeWorkflows} icon={GitBranch} />
      </div>

      {/* Node info */}
      {status && (
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-5">
          <h2 className="text-lg font-semibold mb-4">Node Information</h2>
          <div className="space-y-3 text-sm">
            <InfoRow label="Peer ID" value={status.peerId} mono copyable />
            <InfoRow label="Wallet" value={status.walletAddress || 'N/A'} mono copyable />
            <InfoRow label="Data Directory" value={status.dataDir} mono />
            <div>
              <span className="text-[var(--color-text-muted)]">Addresses</span>
              <div className="mt-1 space-y-1">
                {status.addresses?.map((addr, i) => (
                  <p key={i} className="font-mono text-xs text-[var(--color-text-muted)] bg-[var(--color-bg)] rounded px-2 py-1">
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
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-5">
          <h2 className="text-lg font-semibold mb-4">Local Agents</h2>
          <div className="space-y-2">
            {localAgents.map((agent) => (
              <div key={agent.id} className="flex items-center justify-between bg-[var(--color-bg)] rounded-lg px-4 py-3">
                <div>
                  <p className="font-medium">{agent.name}</p>
                  <p className="text-xs text-[var(--color-text-muted)] font-mono">{agent.agentID}</p>
                </div>
                <span className="text-sm text-[var(--color-primary)] font-medium">
                  {agent.price > 0 ? `${agent.price} USDC` : 'Free'}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Active workflows */}
      {workflows && workflows.filter((w) => w.status === 'running').length > 0 && (
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-5">
          <h2 className="text-lg font-semibold mb-4">Active Workflows</h2>
          <div className="space-y-2">
            {workflows
              .filter((w) => w.status === 'running')
              .map((wf) => {
                const completed = wf.steps.filter((s) => s.status === 'completed').length;
                return (
                  <div key={wf.id} className="bg-[var(--color-bg)] rounded-lg px-4 py-3">
                    <div className="flex justify-between items-center mb-2">
                      <span className="font-mono text-xs">{wf.id.slice(0, 12)}...</span>
                      <span className="text-xs text-[var(--color-warning)]">
                        {completed}/{wf.steps.length} steps
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

function InfoRow({ label, value, mono, copyable }: { label: string; value: string; mono?: boolean; copyable?: boolean }) {
  return (
    <div className="flex items-start gap-2">
      <span className="text-[var(--color-text-muted)] min-w-[120px] shrink-0">{label}</span>
      <span className={`break-all ${mono ? 'font-mono text-xs' : ''}`}>{value}</span>
      {copyable && value && (
        <button
          onClick={() => navigator.clipboard.writeText(value)}
          className="shrink-0 text-[var(--color-text-muted)] hover:text-[var(--color-text)] p-0.5"
          title="Copy"
        >
          <Copy size={14} />
        </button>
      )}
    </div>
  );
}
