import { createConnectTransport } from '@connectrpc/connect-web';
import { createClient } from '@connectrpc/connect';
import { pb } from './pb';

// ── Transport ──────────────────────────────────────────

// In dev, Vite proxies /cortex.v1.* requests to the backend.
// In production, ConnectRPC is served on the same origin by PocketBase.
const baseUrl = import.meta.env.VITE_API_URL || '';

const transport = createConnectTransport({
  baseUrl,
  // Inject the PocketBase auth token on every RPC call.
  interceptors: [
    (next) => async (req) => {
      const token = pb.authStore.token;
      if (token) {
        req.header.set('Authorization', `Bearer ${token}`);
      }
      return next(req);
    },
  ],
});

// ── Service Clients ────────────────────────────────────
//
// The actual generated service definitions aren't available yet (they will
// be created by `buf generate`). For now we export a helper that pages can
// use once the generated types are ready. Until code generation runs, the
// pages will use a lightweight fetch-based fallback that talks directly to
// PocketBase collections for data, and ConnectRPC once stubs are generated.

// Generic typed fetch helper for PocketBase REST endpoints.
// This is the fallback used by all pages until ConnectRPC TS stubs are generated.

interface PBListResult<T> {
  page: number;
  perPage: number;
  totalPages: number;
  totalItems: number;
  items: T[];
}

const API_BASE = import.meta.env.VITE_PB_URL || 'http://127.0.0.1:8090';

async function pbFetch<T>(path: string): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  const token = pb.authStore.token;
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const res = await fetch(`${API_BASE}${path}`, { headers });
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }
  return res.json();
}

// ── Collection API helpers ─────────────────────────────

export interface Project {
  id: string;
  name: string;
  path: string;
  description: string;
  github_url: string;
  status: string;
  progress: number;
  technologies: string[];
  last_scanned: string;
  last_activity: string;
  icon_color: string;
  created: string;
  updated: string;
}

export interface Task {
  id: string;
  project: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  assigned_to: string;
  due_date: string;
  linked_files: string[];
  tags: string[];
  created: string;
  updated: string;
}

export interface Handoff {
  id: string;
  project: string;
  from_agent: string;
  to_agent: string;
  title: string;
  context: string;
  status: string;
  included_files: string[];
  prompt_preview: string;
  token_count: number;
  created: string;
  updated: string;
}

export interface VaultEntry {
  id: string;
  project: string;
  category: string;
  title: string;
  content: string;
  tags: string[];
  is_shared: boolean;
  source_agent: string;
  file_path: string;
  version: number;
  created: string;
  updated: string;
}

export interface ActivityLogEntry {
  id: string;
  project: string;
  owner: string;
  action: string;
  subject: string;
  metadata: Record<string, any>;
  created: string;
  updated: string;
}

export interface FileIndexEntry {
  id: string;
  project: string;
  path: string;
  language: string;
  size_bytes: number;
  last_indexed: string;
  created: string;
  updated: string;
}

export interface DaemonStatus {
  ready: boolean;
  version: string;
  uptimeSeconds: number;
  watcherRunning: boolean;
  semanticEnabled: boolean;
}

export interface SearchResult {
  id: string;
  collection: string;
  project_id: string;
  title: string;
  excerpt: string;
  score: number;
}

// ── API Functions ──────────────────────────────────────

// Projects
export async function listProjects(): Promise<Project[]> {
  const res = await pbFetch<PBListResult<Project>>(
    '/api/collections/projects/records?sort=-last_activity&perPage=200'
  );
  return res.items;
}

export async function createProject(data: {
  name: string;
  path?: string;
  description?: string;
  github_url?: string;
}): Promise<Project> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  const token = pb.authStore.token;
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const owner = pb.authStore.record?.id;
  const body = {
    ...data,
    status: 'active',
    progress: 0,
    icon_color: colorFromName(data.name),
    ...(owner ? { owner } : {}),
  };
  const res = await fetch(`${API_BASE}/api/collections/projects/records`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`Create project failed: ${res.status}`);
  return res.json();
}

function colorFromName(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = ((hash << 5) - hash + name.charCodeAt(i)) | 0;
  }
  const h = Math.abs(hash) % 360;
  return `hsl(${h}, 65%, 55%)`;
}

// Tasks
export async function listTasks(params?: {
  projectId?: string;
  status?: string;
}): Promise<Task[]> {
  let filter = '';
  const parts: string[] = [];
  if (params?.projectId) parts.push(`project="${params.projectId}"`);
  if (params?.status) parts.push(`status="${params.status}"`);
  if (parts.length) filter = `&filter=(${parts.join(' && ')})`;
  const res = await pbFetch<PBListResult<Task>>(
    `/api/collections/tasks/records?sort=-updated&perPage=500${filter}`
  );
  return res.items;
}

export async function updateTask(
  id: string,
  data: Partial<Task>
): Promise<Task> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  const token = pb.authStore.token;
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const res = await fetch(`${API_BASE}/api/collections/tasks/records/${id}`, {
    method: 'PATCH',
    headers,
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error(`Update task failed: ${res.status}`);
  return res.json();
}

export async function createTask(data: {
  title: string;
  project?: string;
  description?: string;
  status?: string;
  priority?: string;
  assigned_to?: string;
}): Promise<Task> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  const token = pb.authStore.token;
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const owner = pb.authStore.record?.id;
  const body = {
    status: 'todo',
    priority: 'medium',
    ...data,
    ...(owner ? { owner } : {}),
  };
  const res = await fetch(`${API_BASE}/api/collections/tasks/records`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`Create task failed: ${res.status}`);
  return res.json();
}

// Handoffs
export async function listHandoffs(params?: {
  projectId?: string;
}): Promise<Handoff[]> {
  let filter = '';
  if (params?.projectId) filter = `&filter=(project="${params.projectId}")`;
  const res = await pbFetch<PBListResult<Handoff>>(
    `/api/collections/handoffs/records?sort=-updated&perPage=200${filter}`
  );
  return res.items;
}

export async function createHandoff(data: {
  title: string;
  from_agent: string;
  to_agent: string;
  context: string;
  project?: string;
  included_files?: string[];
}): Promise<Handoff> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  const token = pb.authStore.token;
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const owner = pb.authStore.record?.id;
  const body = {
    status: 'active',
    ...data,
    token_count: Math.floor((data.context?.length || 0) / 4),
    prompt_preview: `# Handoff: ${data.title}\n\nFrom: ${data.from_agent} → ${data.to_agent}\n\n${data.context}`,
    ...(owner ? { owner } : {}),
  };
  const res = await fetch(`${API_BASE}/api/collections/handoffs/records`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`Create handoff failed: ${res.status}`);
  return res.json();
}

// Vault Entries
export async function listVaultEntries(params?: {
  projectId?: string;
  category?: string;
}): Promise<VaultEntry[]> {
  let filter = '';
  const parts: string[] = [];
  if (params?.projectId) parts.push(`project="${params.projectId}"`);
  if (params?.category) parts.push(`category="${params.category}"`);
  if (parts.length) filter = `&filter=(${parts.join(' && ')})`;
  const res = await pbFetch<PBListResult<VaultEntry>>(
    `/api/collections/vault_entries/records?sort=-updated&perPage=500${filter}`
  );
  return res.items;
}

export async function createVaultEntry(data: {
  title: string;
  category: string;
  content?: string;
  project?: string;
  tags?: string[];
  is_shared?: boolean;
}): Promise<VaultEntry> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  const token = pb.authStore.token;
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const owner = pb.authStore.record?.id;
  const body = {
    version: 1,
    ...data,
    ...(owner ? { owner } : {}),
  };
  const res = await fetch(`${API_BASE}/api/collections/vault_entries/records`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`Create vault entry failed: ${res.status}`);
  return res.json();
}

// Activity Log
export async function listActivity(limit = 10): Promise<ActivityLogEntry[]> {
  const res = await pbFetch<PBListResult<ActivityLogEntry>>(
    `/api/collections/activity_log/records?sort=-created&perPage=${limit}`
  );
  return res.items;
}

// File Index (for recently opened)
export async function listRecentFiles(limit = 4): Promise<FileIndexEntry[]> {
  const res = await pbFetch<PBListResult<FileIndexEntry>>(
    `/api/collections/file_index/records?sort=-last_indexed&perPage=${limit}`
  );
  return res.items;
}

// Search (via ConnectRPC — fallback to PocketBase collection search)
export async function searchAll(query: string, scope?: string[]): Promise<SearchResult[]> {
  if (!query.trim()) return [];
  // Search across multiple collections using PocketBase's built-in search
  const results: SearchResult[] = [];
  const collections = scope?.length
    ? scope.map((s) => s.toLowerCase())
    : ['vault_entries', 'tasks', 'handoffs', 'file_index'];

  const searchPromises = collections.map(async (coll) => {
    try {
      const res = await pbFetch<PBListResult<any>>(
        `/api/collections/${coll}/records?filter=(title~"${encodeURIComponent(query)}" || content~"${encodeURIComponent(query)}")&perPage=10`
      );
      for (const item of res.items) {
        results.push({
          id: item.id,
          collection: coll,
          project_id: item.project || '',
          title: item.title || item.path || '',
          excerpt: item.content?.substring(0, 200) || item.description?.substring(0, 200) || '',
          score: 1.0,
        });
      }
    } catch {
      // Some collections might not have title/content fields, skip errors
    }
  });

  await Promise.all(searchPromises);
  return results;
}

// Daemon Status (via ConnectRPC)
export async function getDaemonStatus(): Promise<DaemonStatus> {
  try {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    const token = pb.authStore.token;
    if (token) headers['Authorization'] = `Bearer ${token}`;
    const res = await fetch(`${API_BASE}/cortex.v1.DaemonService/Status`, {
      method: 'POST',
      headers,
      body: '{}',
    });
    if (!res.ok) throw new Error('Daemon status failed');
    const data = await res.json();
    return {
      ready: data.ready ?? false,
      version: data.version ?? '0.0.0',
      uptimeSeconds: Number(data.uptimeSeconds ?? data.uptime_seconds ?? 0),
      watcherRunning: data.watcherRunning ?? data.watcher_running ?? false,
      semanticEnabled: data.semanticEnabled ?? data.semantic_enabled ?? false,
    };
  } catch {
    return {
      ready: false,
      version: '—',
      uptimeSeconds: 0,
      watcherRunning: false,
      semanticEnabled: false,
    };
  }
}

// Scan Project: clones (if needed) and runs the deep analysis for one repo,
// persisting its tech stack, auth, features, knowledge graph and memory.
export async function scanProject(projectId: string): Promise<ScanRepoResult> {
  return cortexFetch<ScanRepoResult>(`/api/cortex/scan/${projectId}`, {
    method: 'POST',
    body: '{}',
  });
}

// ── CORTEX API (providers, knowledge graph) ──────────
async function cortexFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  const token = pb.authStore.token;
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const res = await fetch(`${API_BASE}${path}`, { headers, ...init });
  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`;
    try {
      const body = await res.json();
      if (body?.message) msg = body.message;
    } catch {
      /* ignore */
    }
    throw new Error(msg);
  }
  return res.json();
}

export interface ScanRepoResult {
  name: string;
  project_id: string;
  indexed_files: number;
  frameworks: string[];
  auth_detected: boolean;
  features: number;
  error?: string;
}

export interface ScanReport {
  total_repos: number;
  scanned: number;
  failed: number;
  enriched: boolean;
  results: ScanRepoResult[];
}

export interface ProviderConfig {
  llm_provider: string; // "mistral" | "ollama" | "none"
  mistral_key: string;
  mistral_model: string;
  ollama_chat_model: string;
  embedder: string; // "ollama" | "mistral" | "none"
  ollama_url: string;
  ollama_model: string;
  mistral_emb_model: string;
}

export async function getProviderConfig(): Promise<ProviderConfig> {
  return cortexFetch<ProviderConfig>('/api/cortex/providers');
}

export async function setProviderConfig(cfg: Partial<ProviderConfig>): Promise<ProviderConfig> {
  return cortexFetch<ProviderConfig>('/api/cortex/providers', {
    method: 'POST',
    body: JSON.stringify(cfg),
  });
}

export interface AnalysisGraph {
  nodes: { id: string; label: string; type: string; color: string }[];
  edges: { source: string; target: string; rel: string }[];
}

export async function getKnowledgeGraph(projectId: string): Promise<AnalysisGraph> {
  return cortexFetch<AnalysisGraph>(`/api/cortex/knowledge-graph/${projectId}`);
}

export interface SystemPromptResult {
  project_id: string;
  project_name: string;
  provider: string; // mistral | ollama | heuristic
  prompt: string;
  token_estimate: number;
}

export interface PromptOptions {
  include_tasks?: boolean;
  include_vault?: boolean;
  include_activity?: boolean;
}

// Generate a project-specific agent system prompt using the configured LLM.
export async function generateSystemPrompt(
  projectId: string,
  opts: PromptOptions = {}
): Promise<SystemPromptResult> {
  return cortexFetch<SystemPromptResult>(`/api/cortex/system-prompt/${projectId}`, {
    method: 'POST',
    body: JSON.stringify(opts),
  });
}

// ── MCP connections ────────────────────────────────────

export interface MCPConnection {
  id: string;
  ide: string;
  label: string;
  project_id: string;
  client_name: string;
  enabled: boolean;
  last_used: string;
  connected: boolean;
  created: string;
  endpoint: string;
  token?: string;
  config?: Record<string, any>;
}

export async function listMCPConnections(): Promise<MCPConnection[]> {
  return cortexFetch<MCPConnection[]>('/api/cortex/mcp/connections');
}

export async function createMCPConnection(data: {
  project_id?: string;
  ide?: string;
  label?: string;
}): Promise<MCPConnection> {
  return cortexFetch<MCPConnection>('/api/cortex/mcp/connections', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function deleteMCPConnection(id: string): Promise<{ success: boolean }> {
  return cortexFetch<{ success: boolean }>(`/api/cortex/mcp/connections/${id}`, {
    method: 'DELETE',
  });
}

export async function getMCPConnectionStatus(id: string): Promise<MCPConnection> {
  return cortexFetch<MCPConnection>(`/api/cortex/mcp/connections/${id}/status`);
}

// ── Dashboard Aggregation Helpers ──────────────────────

export interface ProjectBrainStats {
  architectureProgress: number;
  memoryCoverage: number;
  decisionsMade: number;
  tasksLinked: number;
  knowledgeQuality: 'High' | 'Medium' | 'Low';
}

export async function getProjectBrainStats(projectId: string): Promise<ProjectBrainStats> {
  const [vaultEntries, tasks] = await Promise.all([
    listVaultEntries({ projectId }),
    listTasks({ projectId }),
  ]);

  const archEntries = vaultEntries.filter((e) => e.category === 'architecture');
  const decisions = vaultEntries.filter((e) => e.category === 'decision');
  const totalCategories = ['architecture', 'decision', 'roadmap', 'task', 'handoff', 'memory'];
  const coveredCategories = totalCategories.filter((cat) =>
    vaultEntries.some((e) => e.category === cat)
  );

  const archProgress = Math.min(100, Math.round((archEntries.length / Math.max(1, 5)) * 100));
  const memCoverage = Math.min(100, Math.round((coveredCategories.length / totalCategories.length) * 100));
  const tasksLinked = tasks.length;
  const decisionsMade = decisions.length;

  let knowledgeQuality: 'High' | 'Medium' | 'Low' = 'Low';
  if (vaultEntries.length >= 10 && coveredCategories.length >= 4) knowledgeQuality = 'High';
  else if (vaultEntries.length >= 4 && coveredCategories.length >= 2) knowledgeQuality = 'Medium';

  return { architectureProgress: archProgress, memoryCoverage: memCoverage, decisionsMade, tasksLinked, knowledgeQuality };
}

export interface AgentInfo {
  name: string;
  status: 'active' | 'idle' | 'connected' | 'offline';
  icon?: string;
}

export async function getActiveAgents(projectId?: string): Promise<AgentInfo[]> {
  const [vaultEntries, handoffs] = await Promise.all([
    listVaultEntries(projectId ? { projectId } : undefined),
    listHandoffs(projectId ? { projectId } : undefined),
  ]);

  const agentSet = new Map<string, { lastSeen: string; kind: string }>();

  for (const v of vaultEntries) {
    if (v.source_agent) {
      const existing = agentSet.get(v.source_agent);
      if (!existing || v.updated > existing.lastSeen) {
        agentSet.set(v.source_agent, { lastSeen: v.updated, kind: 'vault' });
      }
    }
  }

  for (const h of handoffs) {
    for (const agent of [h.from_agent, h.to_agent]) {
      if (agent) {
        const existing = agentSet.get(agent);
        if (!existing || h.updated > existing.lastSeen) {
          agentSet.set(agent, { lastSeen: h.updated, kind: 'handoff' });
        }
      }
    }
  }

  const now = Date.now();
  const agents: AgentInfo[] = [];

  for (const [name, info] of agentSet) {
    const ageMs = now - new Date(info.lastSeen).getTime();
    const oneHour = 3600_000;
    let status: AgentInfo['status'] = 'offline';
    if (ageMs < oneHour) status = 'active';
    else if (ageMs < oneHour * 24) status = 'connected';
    else status = 'idle';

    agents.push({ name, status });
  }

  // Sort: active first, then connected, then idle, then offline
  const order: Record<string, number> = { active: 0, connected: 1, idle: 2, offline: 3 };
  agents.sort((a, b) => (order[a.status] ?? 4) - (order[b.status] ?? 4));

  return agents;
}

export async function listProjectActivity(projectId: string, limit = 10): Promise<ActivityLogEntry[]> {
  const res = await pbFetch<PBListResult<ActivityLogEntry>>(
    `/api/collections/activity_log/records?sort=-created&perPage=${limit}&filter=(project="${projectId}")`
  );
  return res.items;
}

export interface KnowledgeGraphNode {
  id: string;
  label: string;
  count: number;
  color: string;
}

export function buildKnowledgeGraph(
  vaultEntries: VaultEntry[],
  tasks: Task[],
  handoffs: Handoff[],
  files: FileIndexEntry[]
): KnowledgeGraphNode[] {
  const categories: Record<string, { count: number; color: string }> = {
    Architecture: { count: 0, color: '#3B82F6' },
    Decisions: { count: 0, color: '#F59E0B' },
    Tasks: { count: 0, color: '#22C55E' },
    Handoffs: { count: 0, color: '#E8326E' },
    Files: { count: 0, color: '#8B5CF6' },
    Agents: { count: 0, color: '#06B6D4' },
  };

  for (const v of vaultEntries) {
    if (v.category === 'architecture') categories.Architecture.count++;
    else if (v.category === 'decision') categories.Decisions.count++;
  }

  categories.Tasks.count = tasks.length;
  categories.Handoffs.count = handoffs.length;
  categories.Files.count = files.length;

  const agentNames = new Set<string>();
  for (const v of vaultEntries) if (v.source_agent) agentNames.add(v.source_agent);
  for (const h of handoffs) {
    if (h.from_agent) agentNames.add(h.from_agent);
    if (h.to_agent) agentNames.add(h.to_agent);
  }
  categories.Agents.count = agentNames.size;

  return Object.entries(categories).map(([label, { count, color }]) => ({
    id: label.toLowerCase(),
    label,
    count,
    color,
  }));
}

// Export the transport for future ConnectRPC client usage
export { transport };
