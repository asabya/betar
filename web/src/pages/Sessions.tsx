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
            onClick={() => { setSelectedAgent(agent.agentID); setSelectedCaller(null); }}
            className={`px-4 py-2 rounded-xl text-sm transition-all border ${
              selectedAgent === agent.agentID
                ? 'bg-[var(--color-primary-subtle)] text-[var(--color-primary-light)] border-[var(--color-border-hover)]'
                : 'bg-[var(--color-surface)] border-[var(--color-border)] text-[var(--color-text-muted)] hover:border-[var(--color-border-hover)]'
            }`}
          >
            {agent.name}
          </button>
        ))}
        {(!agents || agents.length === 0) && (
          <p className="text-[var(--color-text-dim)] text-sm">No local agents. Register an agent first.</p>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Session list */}
        {selectedAgent && (
          <div className="glass-card p-5">
            <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
              <MessageSquare size={18} className="text-[var(--color-primary)]" /> Sessions
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
                    className={`w-full text-left bg-[var(--color-bg)] rounded-xl px-4 py-3 hover:bg-[var(--color-bg-elevated)] transition-colors flex items-center justify-between border ${
                      selectedCaller?.callerId === sess.callerId ? 'border-[var(--color-border-hover)]' : 'border-[var(--color-border)]'
                    }`}
                  >
                    <div>
                      <p className="text-sm font-mono break-all">{sess.callerId}</p>
                      <p className="text-xs text-[var(--color-text-dim)] mt-1">
                        {sess.exchanges?.length ?? 0} exchanges
                      </p>
                    </div>
                    <ChevronRight size={16} className="text-[var(--color-text-dim)] shrink-0" />
                  </button>
                ))}
              </div>
            ) : (
              <p className="text-sm text-[var(--color-text-dim)]">No sessions for this agent</p>
            )}
          </div>
        )}

        {/* Session detail */}
        {sessionDetail && (
          <div className="glass-card p-5">
            <h2 className="text-lg font-semibold mb-4">Exchange History</h2>
            <div className="space-y-4">
              {sessionDetail.exchanges?.map((ex, i) => (
                <div key={ex.requestId || i} className="bg-[var(--color-bg)] rounded-xl p-4 space-y-2 border border-[var(--color-border)]">
                  <div className="flex items-center gap-2 text-xs text-[var(--color-text-dim)]">
                    <Clock size={12} />
                    {formatTimestamp(ex.timestamp)}
                  </div>
                  <div>
                    <p className="text-xs text-[var(--color-info)] font-medium mb-1">Input</p>
                    <p className="text-sm bg-[var(--color-surface-solid)] rounded-lg p-2.5 border border-[var(--color-border)]">{ex.input}</p>
                  </div>
                  {ex.output && (
                    <div>
                      <p className="text-xs text-[var(--color-success)] font-medium mb-1">Output</p>
                      <p className="text-sm bg-[var(--color-surface-solid)] rounded-lg p-2.5 border border-[var(--color-border)] whitespace-pre-wrap">{ex.output}</p>
                    </div>
                  )}
                  {ex.error && (
                    <div>
                      <p className="text-xs text-[var(--color-error)] font-medium mb-1">Error</p>
                      <p className="text-sm bg-[var(--color-surface-solid)] rounded-lg p-2.5 border border-[var(--color-error)]/20 text-[var(--color-error)]">{ex.error}</p>
                    </div>
                  )}
                  {ex.payment && (
                    <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)] pt-2 border-t border-[var(--color-border)]">
                      <CreditCard size={12} className="text-[var(--color-primary)]" />
                      <span className="text-[var(--color-primary)]">{ex.payment.amount} USDC</span>
                      <span className="font-mono text-[var(--color-text-dim)]">{ex.payment.txHash.slice(0, 10)}...</span>
                    </div>
                  )}
                </div>
              ))}
              {(!sessionDetail.exchanges || sessionDetail.exchanges.length === 0) && (
                <p className="text-sm text-[var(--color-text-dim)]">No exchanges in this session</p>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
