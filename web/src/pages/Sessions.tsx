import { useState } from 'react';
import { useLocalAgents, useSessions, useSession } from '../hooks/useApi';
import { ChevronRight, Clock, MessageSquare, CreditCard } from 'lucide-react';
import { ListSkeleton } from '../components/Skeleton';
import { ErrorState } from '../components/ErrorState';
import { formatTimestamp } from '../utils/format';

export function Sessions() {
  const { data: agents } = useLocalAgents();
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [selectedCaller, setSelectedCaller] = useState<{ agentId: string; callerId: string } | null>(null);
  const { data: sessions, isLoading: sessionsLoading, error: sessionsError, refetch: refetchSessions } = useSessions(selectedAgent);
  const { data: sessionDetail } = useSession(selectedCaller?.agentId ?? null, selectedCaller?.callerId ?? null);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Sessions</h1>

      {/* Agent selector */}
      <div className="flex flex-wrap gap-2">
        {agents?.map((agent) => (
          <button
            key={agent.id}
            onClick={() => { setSelectedAgent(agent.id); setSelectedCaller(null); }}
            className={`px-4 py-2 rounded-lg text-sm transition-colors border ${
              selectedAgent === agent.id
                ? 'bg-[var(--color-primary)] text-white border-[var(--color-primary)]'
                : 'bg-[var(--color-surface)] border-[var(--color-border)] text-[var(--color-text-muted)] hover:border-[var(--color-primary)]'
            }`}
          >
            {agent.name}
          </button>
        ))}
        {(!agents || agents.length === 0) && (
          <p className="text-[var(--color-text-muted)] text-sm">No local agents. Register an agent first.</p>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Session list */}
        {selectedAgent && (
          <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-5">
            <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
              <MessageSquare size={18} /> Sessions
            </h2>
            {sessionsError ? (
              <ErrorState message={sessionsError.message} onRetry={() => refetchSessions()} />
            ) : sessionsLoading ? (
              <ListSkeleton rows={3} />
            ) : sessions && sessions.length > 0 ? (
              <div className="space-y-2">
                {sessions.map((sess) => (
                  <button
                    key={sess.id}
                    onClick={() => setSelectedCaller({ agentId: sess.agentId, callerId: sess.callerId })}
                    className={`w-full text-left bg-[var(--color-bg)] rounded-lg px-4 py-3 hover:bg-[var(--color-surface-hover)] transition-colors flex items-center justify-between ${
                      selectedCaller?.callerId === sess.callerId ? 'ring-1 ring-[var(--color-primary)]' : ''
                    }`}
                  >
                    <div>
                      <p className="text-sm font-mono break-all">{sess.callerId}</p>
                      <p className="text-xs text-[var(--color-text-muted)] mt-1">
                        {sess.exchanges?.length ?? 0} exchanges
                      </p>
                    </div>
                    <ChevronRight size={16} className="text-[var(--color-text-muted)] shrink-0" />
                  </button>
                ))}
              </div>
            ) : (
              <p className="text-sm text-[var(--color-text-muted)]">No sessions for this agent</p>
            )}
          </div>
        )}

        {/* Session detail */}
        {sessionDetail && (
          <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-5">
            <h2 className="text-lg font-semibold mb-4">Exchange History</h2>
            <div className="space-y-4">
              {sessionDetail.exchanges?.map((ex, i) => (
                <div key={ex.requestId || i} className="bg-[var(--color-bg)] rounded-lg p-4 space-y-2">
                  <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)]">
                    <Clock size={12} />
                    {formatTimestamp(ex.timestamp)}
                  </div>
                  <div>
                    <p className="text-xs text-[var(--color-info)] font-medium mb-1">Input</p>
                    <p className="text-sm bg-[var(--color-surface)] rounded p-2">{ex.input}</p>
                  </div>
                  {ex.output && (
                    <div>
                      <p className="text-xs text-[var(--color-success)] font-medium mb-1">Output</p>
                      <p className="text-sm bg-[var(--color-surface)] rounded p-2 whitespace-pre-wrap">{ex.output}</p>
                    </div>
                  )}
                  {ex.error && (
                    <div>
                      <p className="text-xs text-[var(--color-error)] font-medium mb-1">Error</p>
                      <p className="text-sm bg-[var(--color-surface)] rounded p-2 text-[var(--color-error)]">{ex.error}</p>
                    </div>
                  )}
                  {ex.payment && (
                    <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)] pt-2 border-t border-[var(--color-border)]">
                      <CreditCard size={12} />
                      <span>{ex.payment.amount} USDC</span>
                      <span className="font-mono">{ex.payment.txHash.slice(0, 10)}...</span>
                    </div>
                  )}
                </div>
              ))}
              {(!sessionDetail.exchanges || sessionDetail.exchanges.length === 0) && (
                <p className="text-sm text-[var(--color-text-muted)]">No exchanges in this session</p>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
