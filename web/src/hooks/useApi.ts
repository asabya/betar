import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import { useToast } from '../components/Toast';
import type { AgentSpec } from '../api/client';

export function useStatus() {
  return useQuery({ queryKey: ['status'], queryFn: api.getStatus, refetchInterval: 5000 });
}

export function usePeers() {
  return useQuery({ queryKey: ['peers'], queryFn: api.getPeers, refetchInterval: 5000 });
}

export function useAgents() {
  return useQuery({ queryKey: ['agents'], queryFn: api.getAgents, refetchInterval: 5000 });
}

export function useLocalAgents() {
  return useQuery({ queryKey: ['localAgents'], queryFn: api.getLocalAgents, refetchInterval: 5000 });
}

export function useWalletBalance() {
  return useQuery({ queryKey: ['walletBalance'], queryFn: api.getWalletBalance, refetchInterval: 10000 });
}

export function useOrders() {
  return useQuery({ queryKey: ['orders'], queryFn: api.getOrders, refetchInterval: 5000 });
}

export function useSessions(agentId: string | null) {
  return useQuery({
    queryKey: ['sessions', agentId],
    queryFn: () => api.getSessions(agentId!),
    enabled: !!agentId,
    refetchInterval: 5000,
  });
}

export function useSession(agentId: string | null, callerId: string | null) {
  return useQuery({
    queryKey: ['session', agentId, callerId],
    queryFn: () => api.getSession(agentId!, callerId!),
    enabled: !!agentId && !!callerId,
    refetchInterval: 5000,
  });
}

export function useWorkflows() {
  return useQuery({ queryKey: ['workflows'], queryFn: api.getWorkflows, refetchInterval: 5000 });
}

export function useWorkflow(id: string | null) {
  return useQuery({
    queryKey: ['workflow', id],
    queryFn: () => api.getWorkflow(id!),
    enabled: !!id,
    refetchInterval: 5000,
  });
}

export function useReputation(tokenId?: string) {
  return useQuery({
    queryKey: ['reputation', tokenId],
    queryFn: () => api.getReputation(tokenId!),
    enabled: !!tokenId,
    refetchInterval: 30000,
  });
}

export function useRegisterAgent() {
  const qc = useQueryClient();
  const { toast } = useToast();
  return useMutation({
    mutationFn: (spec: AgentSpec) => api.registerAgent(spec),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ['localAgents'] });
      toast(`Agent "${data.name}" registered`, 'success');
    },
    onError: (err: Error) => {
      toast(`Registration failed: ${err.message}`, 'error');
    },
  });
}

export function useExecuteAgent() {
  const { toast } = useToast();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: string }) => api.executeAgent(id, input),
    onSuccess: () => {
      toast('Agent executed successfully', 'success');
    },
    onError: (err: Error) => {
      toast(`Execution failed: ${err.message}`, 'error');
    },
  });
}

export function useCreateOrder() {
  const qc = useQueryClient();
  const { toast } = useToast();
  return useMutation({
    mutationFn: ({ agentId, price }: { agentId: string; price: number }) => api.createOrder(agentId, price),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['orders'] });
      toast('Order created', 'success');
    },
    onError: (err: Error) => {
      toast(`Order failed: ${err.message}`, 'error');
    },
  });
}

export function useCreateWorkflow() {
  const qc = useQueryClient();
  const { toast } = useToast();
  return useMutation({
    mutationFn: ({ agentIds, input }: { agentIds: string[]; input: string }) => api.createWorkflow(agentIds, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['workflows'] });
      toast('Workflow created', 'success');
    },
    onError: (err: Error) => {
      toast(`Workflow failed: ${err.message}`, 'error');
    },
  });
}

export function useCancelWorkflow() {
  const qc = useQueryClient();
  const { toast } = useToast();
  return useMutation({
    mutationFn: (id: string) => api.cancelWorkflow(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['workflows'] });
      toast('Workflow canceled', 'info');
    },
    onError: (err: Error) => {
      toast(`Cancel failed: ${err.message}`, 'error');
    },
  });
}
