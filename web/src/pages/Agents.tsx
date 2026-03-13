import { useState } from 'react';
import { useLocalAgents, useAgents, useRegisterAgent, useExecuteAgent, useReputation } from '../hooks/useApi';
import { Modal } from '../components/Modal';
import { CardSkeleton } from '../components/Skeleton';
import { ErrorState } from '../components/ErrorState';
import { Plus, Play, Bot, Globe, Shield, Star, Hash } from 'lucide-react';
import type { AgentSpec } from '../api/client';

type Tab = 'local' | 'network';

export function Agents() {
  const [tab, setTab] = useState<Tab>('local');
  const [showRegister, setShowRegister] = useState(false);
  const [executeTarget, setExecuteTarget] = useState<{ id: string; name: string } | null>(null);
  const [executeInput, setExecuteInput] = useState('');
  const [executeResult, setExecuteResult] = useState<string | null>(null);

  const { data: localAgents, isLoading: localLoading, error: localError, refetch: refetchLocal } = useLocalAgents();
  const { data: networkAgents, isLoading: networkLoading, error: networkError, refetch: refetchNetwork } = useAgents();
  const registerMut = useRegisterAgent();
  const executeMut = useExecuteAgent();

  const handleRegister = (spec: AgentSpec) => {
    registerMut.mutate(spec, { onSuccess: () => setShowRegister(false) });
  };

  const handleExecute = () => {
    if (!executeTarget) return;
    setExecuteResult(null);
    executeMut.mutate(
      { id: executeTarget.id, input: executeInput },
      {
        onSuccess: (data) => setExecuteResult(data.output),
        onError: (err) => setExecuteResult(`Error: ${err.message}`),
      },
    );
  };

  const isLoading = tab === 'local' ? localLoading : networkLoading;
  const error = tab === 'local' ? localError : networkError;
  const refetch = tab === 'local' ? refetchLocal : refetchNetwork;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between flex-wrap gap-4">
        <h1 className="text-2xl font-bold">Agents</h1>
        <button
          onClick={() => setShowRegister(true)}
          className="flex items-center gap-2 bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white px-4 py-2 rounded-lg text-sm transition-colors"
        >
          <Plus size={16} /> Register Agent
        </button>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-[var(--color-surface)] rounded-lg p-1 w-fit">
        <TabButton active={tab === 'local'} onClick={() => setTab('local')} icon={Bot} label="Local" count={localAgents?.length} />
        <TabButton active={tab === 'network'} onClick={() => setTab('network')} icon={Globe} label="Network" count={networkAgents?.length} />
      </div>

      {/* Error state */}
      {error && <ErrorState message={error.message} onRetry={() => refetch()} />}

      {/* Loading state */}
      {!error && isLoading && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <CardSkeleton />
          <CardSkeleton />
          <CardSkeleton />
        </div>
      )}

      {/* Agent grid */}
      {!error && !isLoading && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {tab === 'local' &&
            localAgents?.map((agent) => (
              <div key={agent.id} className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-5 flex flex-col gap-3">
                <div className="flex items-start justify-between">
                  <div>
                    <p className="font-semibold">{agent.name}</p>
                    <p className="text-xs text-[var(--color-text-muted)] mt-1">{agent.description}</p>
                  </div>
                  <div className="flex items-center gap-1.5">
                    {agent.tokenId && (
                      <span className="flex items-center gap-1 text-xs bg-purple-500/10 text-purple-400 px-2 py-0.5 rounded-full" title={`Token #${agent.tokenId}`}>
                        <Hash size={10} /> {agent.tokenId.length > 6 ? agent.tokenId.slice(0, 6) + '...' : agent.tokenId}
                      </span>
                    )}
                    <span className="text-xs bg-[var(--color-success)]/10 text-[var(--color-success)] px-2 py-0.5 rounded-full">
                      {agent.status || 'active'}
                    </span>
                  </div>
                </div>
                <p className="text-xs font-mono text-[var(--color-text-muted)] break-all">{agent.agentID}</p>
                {agent.tokenId && <ReputationBadge tokenId={agent.tokenId} />}
                <div className="flex items-center justify-between mt-auto pt-2 border-t border-[var(--color-border)]">
                  <span className="text-sm font-medium text-[var(--color-primary)]">
                    {agent.price > 0 ? `${agent.price} USDC` : 'Free'}
                  </span>
                  <button
                    onClick={() => { setExecuteTarget({ id: agent.agentID, name: agent.name }); setExecuteInput(''); setExecuteResult(null); }}
                    className="flex items-center gap-1 text-xs bg-[var(--color-surface-hover)] hover:bg-[var(--color-border)] px-3 py-1.5 rounded-lg transition-colors"
                  >
                    <Play size={12} /> Execute
                  </button>
                </div>
              </div>
            ))}

          {tab === 'network' &&
            networkAgents?.map((agent) => (
              <div key={agent.id} className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-5 flex flex-col gap-3">
                <div className="flex items-start justify-between">
                  <div>
                    <p className="font-semibold">{agent.name}</p>
                    <p className="text-xs font-mono text-[var(--color-text-muted)] mt-1 break-all">{agent.id}</p>
                  </div>
                  <div className="flex items-center gap-1.5">
                    {agent.tokenId && (
                      <span className="flex items-center gap-1 text-xs bg-purple-500/10 text-purple-400 px-2 py-0.5 rounded-full" title={`Token #${agent.tokenId}`}>
                        <Hash size={10} /> {agent.tokenId.length > 6 ? agent.tokenId.slice(0, 6) + '...' : agent.tokenId}
                      </span>
                    )}
                    {agent.protocols?.includes('/x402/libp2p/1.0.0') && (
                      <span className="flex items-center gap-1 text-xs bg-blue-500/10 text-blue-400 px-2 py-0.5 rounded-full">
                        <Shield size={10} /> x402
                      </span>
                    )}
                  </div>
                </div>
                {agent.tokenId && <ReputationBadge tokenId={agent.tokenId} />}
                <div className="flex items-center justify-between mt-auto pt-2 border-t border-[var(--color-border)]">
                  <span className="text-sm font-medium text-[var(--color-primary)]">
                    {agent.price > 0 ? `${agent.price} USDC` : 'Free'}
                  </span>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-[var(--color-text-muted)]">
                      {agent.sellerId.slice(0, 12)}...
                    </span>
                    <button
                      onClick={() => { setExecuteTarget({ id: agent.id, name: agent.name }); setExecuteInput(''); setExecuteResult(null); }}
                      className="flex items-center gap-1 text-xs bg-[var(--color-surface-hover)] hover:bg-[var(--color-border)] px-3 py-1.5 rounded-lg transition-colors"
                    >
                      <Play size={12} /> Execute
                    </button>
                  </div>
                </div>
              </div>
            ))}

          {/* Empty states with CTAs */}
          {tab === 'local' && (!localAgents || localAgents.length === 0) && (
            <div className="col-span-full text-center py-12 text-[var(--color-text-muted)] bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl">
              <Bot size={40} className="mx-auto mb-3 opacity-30" />
              <p className="mb-3">No local agents registered</p>
              <button
                onClick={() => setShowRegister(true)}
                className="inline-flex items-center gap-2 bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white px-4 py-2 rounded-lg text-sm transition-colors"
              >
                <Plus size={16} /> Register your first agent
              </button>
            </div>
          )}

          {tab === 'network' && (!networkAgents || networkAgents.length === 0) && (
            <div className="col-span-full text-center py-12 text-[var(--color-text-muted)] bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl">
              <Globe size={40} className="mx-auto mb-3 opacity-30" />
              <p>No network agents discovered yet</p>
              <p className="text-xs mt-1">Agents will appear as peers connect</p>
            </div>
          )}
        </div>
      )}

      {/* Register Modal */}
      <Modal open={showRegister} onClose={() => setShowRegister(false)} title="Register Agent">
        <RegisterForm onSubmit={handleRegister} loading={registerMut.isPending} />
      </Modal>

      {/* Execute Modal */}
      <Modal open={!!executeTarget} onClose={() => setExecuteTarget(null)} title={`Execute: ${executeTarget?.name ?? ''}`}>
        <div className="space-y-4">
          <textarea
            value={executeInput}
            onChange={(e) => setExecuteInput(e.target.value)}
            placeholder="Enter task input..."
            rows={4}
            className="w-full bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-3 py-2 text-sm resize-none focus:outline-none focus:border-[var(--color-primary)]"
          />
          <button
            onClick={handleExecute}
            disabled={!executeInput.trim() || executeMut.isPending}
            className="w-full bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] disabled:opacity-50 text-white px-4 py-2 rounded-lg text-sm transition-colors"
          >
            {executeMut.isPending ? 'Executing...' : 'Execute'}
          </button>
          {executeResult !== null && (
            <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg p-3 text-sm whitespace-pre-wrap max-h-60 overflow-auto">
              {executeResult}
            </div>
          )}
        </div>
      </Modal>
    </div>
  );
}

function ReputationBadge({ tokenId }: { tokenId: string }) {
  const { data } = useReputation(tokenId);
  if (!data || data.count === 0) return null;
  return (
    <div className="flex items-center gap-1 text-xs text-yellow-400">
      <Star size={12} fill="currentColor" />
      <span>{data.value}{data.decimals > 0 ? `e-${data.decimals}` : ''}</span>
      <span className="text-[var(--color-text-muted)]">({data.count} reviews)</span>
    </div>
  );
}

function TabButton({ active, onClick, icon: Icon, label, count }: {
  active: boolean; onClick: () => void; icon: React.ElementType; label: string; count?: number;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm transition-colors ${
        active ? 'bg-[var(--color-primary)] text-white' : 'text-[var(--color-text-muted)] hover:text-[var(--color-text)]'
      }`}
    >
      <Icon size={16} />
      {label}
      {count !== undefined && (
        <span className={`text-xs px-1.5 py-0.5 rounded-full ${active ? 'bg-white/20' : 'bg-[var(--color-border)]'}`}>
          {count}
        </span>
      )}
    </button>
  );
}

function RegisterForm({ onSubmit, loading }: { onSubmit: (spec: AgentSpec) => void; loading: boolean }) {
  const [form, setForm] = useState<AgentSpec>({ name: '', description: '', price: 0 });

  const update = (field: keyof AgentSpec, value: string | number | boolean) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  };

  return (
    <form
      onSubmit={(e) => { e.preventDefault(); onSubmit(form); }}
      className="space-y-4"
    >
      <Field label="Name" required>
        <input value={form.name} onChange={(e) => update('name', e.target.value)} required className="input" />
      </Field>
      <Field label="Description">
        <textarea value={form.description} onChange={(e) => update('description', e.target.value)} rows={2} className="input resize-none" />
      </Field>
      <Field label="Price (USDC)">
        <input type="number" step="0.01" min="0" value={form.price} onChange={(e) => update('price', parseFloat(e.target.value) || 0)} className="input" />
      </Field>
      <Field label="Model">
        <input value={form.model || ''} onChange={(e) => update('model', e.target.value)} placeholder="gemini-2.5-flash" className="input" />
      </Field>
      <Field label="Provider">
        <select value={form.provider || ''} onChange={(e) => update('provider', e.target.value)} className="input">
          <option value="">Auto</option>
          <option value="google">Google</option>
          <option value="openai">OpenAI</option>
        </select>
      </Field>
      <label className="flex items-center gap-2 text-sm">
        <input type="checkbox" checked={form.x402Support || false} onChange={(e) => update('x402Support', e.target.checked)} />
        Enable x402 payments
      </label>
      <button type="submit" disabled={loading || !form.name} className="w-full bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] disabled:opacity-50 text-white px-4 py-2 rounded-lg text-sm transition-colors">
        {loading ? 'Registering...' : 'Register Agent'}
      </button>
    </form>
  );
}

function Field({ label, required, children }: { label: string; required?: boolean; children: React.ReactNode }) {
  return (
    <div>
      <label className="block text-sm text-[var(--color-text-muted)] mb-1">
        {label} {required && <span className="text-[var(--color-error)]">*</span>}
      </label>
      <div className="[&_.input]:w-full [&_.input]:bg-[var(--color-bg)] [&_.input]:border [&_.input]:border-[var(--color-border)] [&_.input]:rounded-lg [&_.input]:px-3 [&_.input]:py-2 [&_.input]:text-sm [&_.input]:focus:outline-none [&_.input]:focus:border-[var(--color-primary)]">
        {children}
      </div>
    </div>
  );
}
