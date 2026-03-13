import { useState } from 'react';
import { useOrders, useCreateOrder, useAgents } from '../hooks/useApi';
import { Modal } from '../components/Modal';
import { ListSkeleton } from '../components/Skeleton';
import { ErrorState } from '../components/ErrorState';
import { Plus, ShoppingCart } from 'lucide-react';
import { formatTimestamp } from '../utils/format';

const statusColors: Record<string, string> = {
  pending: 'bg-yellow-500/10 text-yellow-500',
  accepted: 'bg-blue-500/10 text-blue-500',
  completed: 'bg-green-500/10 text-green-500',
  cancelled: 'bg-red-500/10 text-red-500',
};

export function Orders() {
  const [showCreate, setShowCreate] = useState(false);
  const { data: orders, isLoading, error, refetch } = useOrders();
  const { data: agents } = useAgents();
  const createMut = useCreateOrder();

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between flex-wrap gap-4">
        <h1 className="text-2xl font-bold">Orders</h1>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white px-4 py-2 rounded-lg text-sm transition-colors"
        >
          <Plus size={16} /> Create Order
        </button>
      </div>

      {error ? (
        <ErrorState message={error.message} onRetry={() => refetch()} />
      ) : isLoading ? (
        <ListSkeleton rows={3} />
      ) : orders && orders.length > 0 ? (
        <div className="space-y-3">
          {orders.map((order) => (
            <div key={order.orderId} className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4">
              <div className="flex items-start justify-between flex-wrap gap-2">
                <div className="space-y-1">
                  <p className="font-mono text-sm">{order.orderId}</p>
                  <p className="text-xs text-[var(--color-text-muted)]">
                    Agent: <span className="font-mono">{order.agentId}</span>
                  </p>
                  <p className="text-xs text-[var(--color-text-muted)]">
                    {formatTimestamp(order.timestamp)}
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-sm font-medium text-[var(--color-primary)]">{order.price} USDC</span>
                  <span className={`text-xs px-2 py-0.5 rounded-full ${statusColors[order.status] || ''}`}>
                    {order.status}
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="text-center py-12 text-[var(--color-text-muted)] bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl">
          <ShoppingCart size={40} className="mx-auto mb-3 opacity-30" />
          <p className="mb-3">No orders yet</p>
          <button
            onClick={() => setShowCreate(true)}
            className="inline-flex items-center gap-2 bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white px-4 py-2 rounded-lg text-sm transition-colors"
          >
            <Plus size={16} /> Create your first order
          </button>
        </div>
      )}

      <Modal open={showCreate} onClose={() => setShowCreate(false)} title="Create Order">
        <CreateOrderForm
          agents={agents ?? []}
          onSubmit={(agentId, price) => {
            createMut.mutate({ agentId, price }, { onSuccess: () => setShowCreate(false) });
          }}
          loading={createMut.isPending}
        />
      </Modal>
    </div>
  );
}

function CreateOrderForm({ agents, onSubmit, loading }: {
  agents: { id: string; name: string }[];
  onSubmit: (agentId: string, price: number) => void;
  loading: boolean;
}) {
  const [agentId, setAgentId] = useState('');
  const [price, setPrice] = useState(0);

  return (
    <form onSubmit={(e) => { e.preventDefault(); onSubmit(agentId, price); }} className="space-y-4">
      <div>
        <label className="block text-sm text-[var(--color-text-muted)] mb-1">Agent</label>
        <select
          value={agentId}
          onChange={(e) => setAgentId(e.target.value)}
          required
          className="w-full bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-[var(--color-primary)]"
        >
          <option value="">Select agent...</option>
          {agents.map((a) => (
            <option key={a.id} value={a.id}>{a.name} ({a.id.slice(0, 12)}...)</option>
          ))}
        </select>
      </div>
      <div>
        <label className="block text-sm text-[var(--color-text-muted)] mb-1">Price (USDC)</label>
        <input
          type="number"
          step="0.01"
          min="0"
          value={price}
          onChange={(e) => setPrice(parseFloat(e.target.value) || 0)}
          required
          className="w-full bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-[var(--color-primary)]"
        />
      </div>
      <button
        type="submit"
        disabled={loading || !agentId}
        className="w-full bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] disabled:opacity-50 text-white px-4 py-2 rounded-lg text-sm transition-colors"
      >
        {loading ? 'Creating...' : 'Create Order'}
      </button>
    </form>
  );
}
