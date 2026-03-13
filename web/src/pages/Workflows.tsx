import { useState } from 'react';
import { useWorkflows, useWorkflow, useCreateWorkflow, useCancelWorkflow, useAgents, useLocalAgents } from '../hooks/useApi';
import { Modal } from '../components/Modal';
import { ListSkeleton } from '../components/Skeleton';
import { ErrorState } from '../components/ErrorState';
import { Plus, X, ChevronRight, CheckCircle2, XCircle, Clock, Loader2, SkipForward, GitBranch } from 'lucide-react';
import { formatTimestamp } from '../utils/format';

const statusColors: Record<string, string> = {
  pending: 'text-[var(--color-text-muted)]',
  running: 'text-[var(--color-warning)]',
  completed: 'text-[var(--color-success)]',
  failed: 'text-[var(--color-error)]',
  canceled: 'text-[var(--color-text-muted)]',
};

const stepIcons: Record<string, React.ElementType> = {
  pending: Clock,
  running: Loader2,
  completed: CheckCircle2,
  failed: XCircle,
  skipped: SkipForward,
};

export function Workflows() {
  const [showCreate, setShowCreate] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const { data: workflows, isLoading, error, refetch } = useWorkflows();
  const { data: detail } = useWorkflow(selectedId);
  const createMut = useCreateWorkflow();
  const cancelMut = useCancelWorkflow();

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between flex-wrap gap-4">
        <h1 className="text-2xl font-bold">Workflows</h1>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white px-4 py-2 rounded-lg text-sm transition-colors"
        >
          <Plus size={16} /> Create Workflow
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Workflow list */}
        <div className="space-y-3">
          {error ? (
            <ErrorState message={error.message} onRetry={() => refetch()} />
          ) : isLoading ? (
            <ListSkeleton rows={3} />
          ) : workflows && workflows.length > 0 ? (
            workflows.map((wf) => {
              const completed = wf.steps?.filter((s) => s.status === 'completed').length ?? 0;
              const total = wf.steps?.length ?? 0;
              return (
                <button
                  key={wf.id}
                  onClick={() => setSelectedId(wf.id)}
                  className={`w-full text-left bg-[var(--color-surface)] border rounded-xl p-4 hover:bg-[var(--color-surface-hover)] transition-colors flex items-center justify-between ${
                    selectedId === wf.id ? 'border-[var(--color-primary)]' : 'border-[var(--color-border)]'
                  }`}
                >
                  <div className="space-y-1 min-w-0">
                    <p className="font-mono text-sm truncate">{wf.id}</p>
                    <div className="flex items-center gap-3 text-xs">
                      <span className={statusColors[wf.status] || ''}>{wf.status}</span>
                      <span className="text-[var(--color-text-muted)]">{completed}/{total} steps</span>
                      {wf.totalCost && wf.totalCost !== '0' && (
                        <span className="text-[var(--color-text-muted)]">{wf.totalCost} USDC</span>
                      )}
                    </div>
                    <div className="w-full bg-[var(--color-border)] rounded-full h-1 mt-2">
                      <div
                        className="bg-[var(--color-primary)] rounded-full h-1 transition-all"
                        style={{ width: total > 0 ? `${(completed / total) * 100}%` : '0%' }}
                      />
                    </div>
                  </div>
                  <ChevronRight size={16} className="text-[var(--color-text-muted)] shrink-0 ml-3" />
                </button>
              );
            })
          ) : (
            <div className="text-center py-12 text-[var(--color-text-muted)] bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl">
              <GitBranch size={40} className="mx-auto mb-3 opacity-30" />
              <p className="mb-3">No workflows yet</p>
              <button
                onClick={() => setShowCreate(true)}
                className="inline-flex items-center gap-2 bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white px-4 py-2 rounded-lg text-sm transition-colors"
              >
                <Plus size={16} /> Create your first workflow
              </button>
            </div>
          )}
        </div>

        {/* Workflow detail */}
        {detail && (
          <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-5 space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold">Workflow Detail</h2>
              {detail.status === 'running' && (
                <button
                  onClick={() => cancelMut.mutate(detail.id)}
                  disabled={cancelMut.isPending}
                  className="flex items-center gap-1 text-xs text-[var(--color-error)] hover:bg-[var(--color-error)]/10 px-3 py-1.5 rounded-lg transition-colors"
                >
                  <X size={14} /> Cancel
                </button>
              )}
            </div>

            <div className="text-sm space-y-1">
              <p><span className="text-[var(--color-text-muted)]">Status:</span> <span className={statusColors[detail.status] || ''}>{detail.status}</span></p>
              <p><span className="text-[var(--color-text-muted)]">Input:</span> {detail.input}</p>
              {detail.output && <p><span className="text-[var(--color-text-muted)]">Output:</span> {detail.output}</p>}
              <p><span className="text-[var(--color-text-muted)]">Created:</span> {formatTimestamp(detail.createdAt)}</p>
            </div>

            {/* Steps timeline */}
            <div className="space-y-0">
              {detail.steps?.map((step, i) => {
                const StepIcon = stepIcons[step.status] || Clock;
                return (
                  <div key={i} className="flex gap-3">
                    <div className="flex flex-col items-center">
                      <div className={`p-1 rounded-full ${statusColors[step.status]}`}>
                        <StepIcon size={16} className={step.status === 'running' ? 'animate-spin' : ''} />
                      </div>
                      {i < (detail.steps?.length ?? 0) - 1 && (
                        <div className="w-px flex-1 bg-[var(--color-border)] min-h-[20px]" />
                      )}
                    </div>
                    <div className="pb-4 flex-1 min-w-0">
                      <p className="text-sm font-medium">Step {step.index + 1}</p>
                      <p className="text-xs text-[var(--color-text-muted)] font-mono truncate">{step.agentId}</p>
                      {step.input && <p className="text-xs mt-1 text-[var(--color-text-muted)]">Input: {step.input.slice(0, 100)}{step.input.length > 100 ? '...' : ''}</p>}
                      {step.output && <p className="text-xs mt-1">Output: {step.output.slice(0, 200)}{step.output.length > 200 ? '...' : ''}</p>}
                      {step.error && <p className="text-xs mt-1 text-[var(--color-error)]">Error: {step.error}</p>}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>

      {/* Create Modal */}
      <Modal open={showCreate} onClose={() => setShowCreate(false)} title="Create Workflow">
        <CreateWorkflowForm
          onSubmit={(agentIds, input) => {
            createMut.mutate({ agentIds, input }, { onSuccess: () => setShowCreate(false) });
          }}
          loading={createMut.isPending}
        />
      </Modal>
    </div>
  );
}

function CreateWorkflowForm({ onSubmit, loading }: { onSubmit: (agentIds: string[], input: string) => void; loading: boolean }) {
  const { data: localAgents } = useLocalAgents();
  const { data: networkAgents } = useAgents();
  const [selectedAgents, setSelectedAgents] = useState<string[]>([]);
  const [input, setInput] = useState('');

  const allAgents = [
    ...(localAgents?.map((a) => ({ id: a.agentID, name: a.name })) ?? []),
    ...(networkAgents?.map((a) => ({ id: a.id, name: a.name })) ?? []),
  ];

  const toggleAgent = (id: string) => {
    setSelectedAgents((prev) =>
      prev.includes(id) ? prev.filter((a) => a !== id) : [...prev, id],
    );
  };

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-sm text-[var(--color-text-muted)] mb-2">Select agents (in order)</label>
        <div className="space-y-1 max-h-40 overflow-auto">
          {allAgents.map((agent) => (
            <button
              key={agent.id}
              onClick={() => toggleAgent(agent.id)}
              className={`w-full text-left px-3 py-2 rounded-lg text-sm transition-colors ${
                selectedAgents.includes(agent.id)
                  ? 'bg-[var(--color-primary)]/10 text-[var(--color-primary)] border border-[var(--color-primary)]'
                  : 'bg-[var(--color-bg)] border border-[var(--color-border)] hover:border-[var(--color-primary)]'
              }`}
            >
              {selectedAgents.includes(agent.id) && <span className="mr-2">{selectedAgents.indexOf(agent.id) + 1}.</span>}
              {agent.name}
            </button>
          ))}
        </div>
      </div>
      <div>
        <label className="block text-sm text-[var(--color-text-muted)] mb-1">Input</label>
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          rows={3}
          placeholder="Workflow input..."
          className="w-full bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-3 py-2 text-sm resize-none focus:outline-none focus:border-[var(--color-primary)]"
        />
      </div>
      <button
        onClick={() => onSubmit(selectedAgents, input)}
        disabled={loading || selectedAgents.length === 0 || !input.trim()}
        className="w-full bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] disabled:opacity-50 text-white px-4 py-2 rounded-lg text-sm transition-colors"
      >
        {loading ? 'Creating...' : 'Create Workflow'}
      </button>
    </div>
  );
}
