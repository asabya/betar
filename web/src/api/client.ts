const BASE_URL = import.meta.env.VITE_API_URL !== undefined ? import.meta.env.VITE_API_URL : '/api';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API error (${res.status}): ${text}`);
  }
  return res.json();
}

// Types

export interface NodeStatus {
  peerId: string;
  addresses: string[];
  connectedPeers: number;
  walletAddress: string;
  dataDir: string;
}

export interface PeerInfo {
  id: string;
  addrs: string[];
}

export interface AgentListing {
  id: string;
  name: string;
  price: number;
  metadata: string;
  sellerId: string;
  addrs?: string[];
  protocols?: string[];
  timestamp: number;
  tokenId?: string;
}

export interface LocalAgent {
  id: string;
  name: string;
  description: string;
  price: number;
  metadataCID: string;
  agentID: string;
  status: string;
  createdAt: string;
  tokenId?: string;
}

export interface ReputationData {
  count: number;
  value: number;
  decimals: number;
}

export interface AgentSpec {
  name: string;
  description: string;
  image?: string;
  price: number;
  model?: string;
  apiKey?: string;
  services?: { name: string; version?: string }[];
  x402Support?: boolean;
  provider?: string;
  openaiApiKey?: string;
  openaiBaseUrl?: string;
}

export interface Order {
  orderId: string;
  agentId: string;
  buyerId: string;
  sellerId?: string;
  price: number;
  status: string;
  timestamp: number;
}

export interface PaymentRecord {
  paymentId: string;
  txHash: string;
  amount: string;
  payer: string;
  paidAt: string;
}

export interface Exchange {
  requestId: string;
  input: string;
  output: string;
  error?: string;
  timestamp: string;
  payment?: PaymentRecord;
}

export interface Session {
  id: string;
  agentId: string;
  callerId: string;
  createdAt: string;
  updatedAt: string;
  exchanges: Exchange[];
}

export interface WorkflowStep {
  index: number;
  agentId: string;
  status: string;
  input: string;
  output?: string;
  error?: string;
  payment?: PaymentRecord;
  startedAt?: string;
  completedAt?: string;
}

export interface Workflow {
  id: string;
  status: string;
  steps: WorkflowStep[];
  input: string;
  output?: string;
  totalCost: string;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}

export interface WalletBalance {
  address: string;
  balance: number;
  usdcBalance?: number;
}

// API functions

export const api = {
  getStatus: () => request<NodeStatus>('/status'),
  getPeers: () => request<PeerInfo[]>('/peers'),
  getHealth: () => request<{ status: string }>('/health'),

  getAgents: () => request<AgentListing[]>('/agents'),
  getLocalAgents: () => request<LocalAgent[]>('/agents/local'),
  registerAgent: (spec: AgentSpec) =>
    request<LocalAgent>('/agents', { method: 'POST', body: JSON.stringify(spec) }),
  executeAgent: (id: string, input: string) =>
    request<{ output: string }>(`/agents/${encodeURIComponent(id)}/execute`, {
      method: 'POST',
      body: JSON.stringify({ input }),
    }),

  getWalletBalance: () => request<WalletBalance>('/wallet/balance'),

  getOrders: () => request<Order[]>('/orders'),
  createOrder: (agentId: string, price: number) =>
    request<Order>('/orders', { method: 'POST', body: JSON.stringify({ agentId, price }) }),

  getSessions: (agentId: string) => request<Session[]>(`/sessions/${encodeURIComponent(agentId)}`),
  getSession: (agentId: string, callerId: string) =>
    request<Session>(`/sessions/${encodeURIComponent(agentId)}/${encodeURIComponent(callerId)}`),

  createWorkflow: (agentIds: string[], input: string) =>
    request<Workflow>('/workflows', { method: 'POST', body: JSON.stringify({ agentIds, input }) }),
  getWorkflows: () => request<Workflow[]>('/workflows'),
  getWorkflow: (id: string) => request<Workflow>(`/workflows/${encodeURIComponent(id)}`),
  cancelWorkflow: (id: string) =>
    request<Workflow>(`/workflows/${encodeURIComponent(id)}`, { method: 'DELETE' }),

  getReputation: (tokenId: string) =>
    request<ReputationData>(`/agents/reputation/${encodeURIComponent(tokenId)}`),
};
