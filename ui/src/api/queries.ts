// Centralized server-state layer built on @tanstack/solid-query.
//
// Every remote read is a query (keyed + cached + deduped), every remote write
// is a mutation that invalidates the query keys it affects. Pages consume these
// hooks instead of calling the raw client + createResource, so a single fetch of
// e.g. the project list is shared across the whole app.

import { useQuery, useMutation, type QueryClient } from '@tanstack/solid-query';
import type { Accessor } from 'solid-js';
import { queryClient } from './queryClient';
import {
  listProjects,
  getProject,
  createProject,
  listTasks,
  createTask,
  updateTask,
  listHandoffs,
  createHandoff,
  listVaultEntries,
  createVaultEntry,
  listActivity,
  listProjectActivity,
  listRecentFiles,
  listNotifications,
  getDaemonStatus,
  getProviderConfig,
  setProviderConfig,
  scanProject,
  buildCodeGraph,
  getCodeGraph,
  generateSessionDigest,
  listSessionDigests,
  listAgentMemories,
  listMCPConnections,
  createMCPConnection,
  deleteMCPConnection,
  resetAllData,
  getProjectBrainStats,
  getActiveAgents,
  getSystemPrompt,
  saveSystemPrompt,
  syncGitHubRepos,
  type Task,
  type ProviderConfig,
} from './client';

// ── Query keys ─────────────────────────────────────────
// One place that owns every cache key, so invalidation stays consistent.
export const qk = {
  projects: ['projects'] as const,
  project: (id: string) => ['projects', id] as const,
  tasks: (params?: { projectId?: string; status?: string }) => ['tasks', params ?? {}] as const,
  handoffs: (params?: { projectId?: string }) => ['handoffs', params ?? {}] as const,
  vaultEntries: (params?: { projectId?: string; category?: string }) => ['vaultEntries', params ?? {}] as const,
  activity: (limit: number) => ['activity', limit] as const,
  notifications: (limit: number) => ['notifications', limit] as const,
  projectActivity: (projectId: string, limit: number) => ['projectActivity', projectId, limit] as const,
  recentFiles: (limit: number, projectId = '') => ['recentFiles', limit, projectId] as const,
  daemonStatus: ['daemonStatus'] as const,
  providerConfig: ['providerConfig'] as const,
  mcpConnections: ['mcpConnections'] as const,
  sessionDigests: (projectId: string) => ['sessionDigests', projectId] as const,
  codeGraph: (projectId: string) => ['codeGraph', projectId] as const,
  agentMemories: (projectId: string) => ['agentMemories', projectId] as const,
  brainStats: (projectId: string) => ['brainStats', projectId] as const,
  activeAgents: (projectId: string) => ['activeAgents', projectId] as const,
  systemPrompt: (projectId: string) => ['systemPrompt', projectId] as const,
};

const invalidate = (client: QueryClient, key: readonly unknown[]) =>
  client.invalidateQueries({ queryKey: key });

// ── Queries ────────────────────────────────────────────

export function useProjects() {
  return useQuery(() => ({
    queryKey: qk.projects,
    queryFn: listProjects,
    refetchInterval: 30_000,
  }));
}

export function useProject(id: Accessor<string>) {
  return useQuery(() => ({
    queryKey: qk.project(id()),
    queryFn: () => getProject(id()),
    enabled: !!id(),
  }));
}

export function useTasks(params?: Accessor<{ projectId?: string; status?: string } | undefined>) {
  return useQuery(() => {
    const scope = params?.();
    return {
      queryKey: qk.tasks(scope),
      queryFn: () => listTasks(scope),
      enabled: params ? !!scope?.projectId : true,
      refetchInterval: params ? 30_000 : undefined,
    };
  });
}

export function useHandoffs(params?: Accessor<{ projectId?: string } | undefined>) {
  return useQuery(() => {
    const scope = params?.();
    return {
      queryKey: qk.handoffs(scope),
      queryFn: () => listHandoffs(scope),
      enabled: params ? !!scope?.projectId : true,
      refetchInterval: params ? 30_000 : undefined,
    };
  });
}

export function useVaultEntries(params?: Accessor<{ projectId?: string; category?: string } | undefined>) {
  return useQuery(() => {
    const scope = params?.();
    return {
      queryKey: qk.vaultEntries(scope),
      queryFn: () => listVaultEntries(scope),
      enabled: params ? !!scope?.projectId : true,
      refetchInterval: params ? 5_000 : undefined,
    };
  });
}

export function useActivity(limit = 5) {
  return useQuery(() => ({ queryKey: qk.activity(limit), queryFn: () => listActivity(limit) }));
}

export function useNotifications(limit = 20) {
  return useQuery(() => ({ queryKey: qk.notifications(limit), queryFn: () => listNotifications(limit) }));
}

export function useProjectActivity(projectId: Accessor<string>, limit = 10) {
  return useQuery(() => ({
    queryKey: qk.projectActivity(projectId(), limit),
    queryFn: () => (projectId() ? listProjectActivity(projectId(), limit) : Promise.resolve([])),
    enabled: !!projectId(),
    refetchInterval: 5_000,
  }));
}

export function useRecentFiles(limit = 4, projectId?: Accessor<string>) {
  return useQuery(() => {
    const id = projectId?.() ?? '';
    return {
      queryKey: qk.recentFiles(limit, id),
      queryFn: () => listRecentFiles(limit, id || undefined),
      enabled: projectId ? !!id : true,
      refetchInterval: projectId ? 30_000 : undefined,
    };
  });
}

export function useSystemPrompt(projectId: Accessor<string>) {
  return useQuery(() => ({
    queryKey: qk.systemPrompt(projectId()),
    queryFn: () => getSystemPrompt(projectId()),
    enabled: !!projectId(),
    refetchInterval: 5_000,
  }));
}

export function useSaveSystemPrompt() {
  return useMutation(() => ({
    mutationFn: (vars: { projectId: string; prompt: string }) =>
      saveSystemPrompt(vars.projectId, vars.prompt),
    onSuccess: (_res, vars) => invalidate(queryClient, qk.systemPrompt(vars.projectId)),
  }));
}

export function useDaemonStatus() {
  return useQuery(() => ({
    queryKey: qk.daemonStatus,
    queryFn: getDaemonStatus,
    refetchInterval: 30_000,
  }));
}

export function useProviderConfig() {
  return useQuery(() => ({ queryKey: qk.providerConfig, queryFn: getProviderConfig }));
}

export function useMCPConnections() {
  return useQuery(() => ({ queryKey: qk.mcpConnections, queryFn: listMCPConnections }));
}

export function useSessionDigests(projectId: Accessor<string>) {
  return useQuery(() => ({
    queryKey: qk.sessionDigests(projectId()),
    queryFn: () => listSessionDigests(projectId()),
    enabled: !!projectId(),
    refetchInterval: 5_000,
  }));
}

export function useCodeGraph(projectId: Accessor<string>) {
  return useQuery(() => ({
    queryKey: qk.codeGraph(projectId()),
    queryFn: () => getCodeGraph(projectId()),
    enabled: !!projectId(),
    refetchInterval: 30_000,
  }));
}

export function useAgentMemories(projectId: Accessor<string>) {
  return useQuery(() => ({
    queryKey: qk.agentMemories(projectId()),
    queryFn: () => listAgentMemories(projectId()),
    enabled: !!projectId(),
    refetchInterval: 5_000,
  }));
}

export function useProjectBrainStats(projectId: Accessor<string>) {
  return useQuery(() => ({
    queryKey: qk.brainStats(projectId()),
    queryFn: () => (projectId() ? getProjectBrainStats(projectId()) : Promise.resolve(null)),
    enabled: !!projectId(),
    refetchInterval: 30_000,
  }));
}

export function useActiveAgents(projectId: Accessor<string>) {
  return useQuery(() => ({
    queryKey: qk.activeAgents(projectId()),
    queryFn: () => getActiveAgents(projectId() || undefined),
    enabled: !!projectId(),
    refetchInterval: 5_000,
  }));
}

// ── Mutations ──────────────────────────────────────────

export function useCreateProject() {
  return useMutation(() => ({
    mutationFn: createProject,
    onSuccess: () => invalidate(queryClient, qk.projects),
  }));
}

export function useSyncGitHubRepos() {
  return useMutation(() => ({
    mutationFn: syncGitHubRepos,
    onSuccess: () => invalidate(queryClient, qk.projects),
  }));
}

export function useScanProject() {
  return useMutation(() => ({
    mutationFn: (projectId: string) => scanProject(projectId),
    onSuccess: (_res, projectId) => {
      invalidate(queryClient, qk.projects);
      invalidate(queryClient, qk.project(projectId));
      invalidate(queryClient, qk.codeGraph(projectId));
      invalidate(queryClient, ['tasks']);
      invalidate(queryClient, ['handoffs']);
      invalidate(queryClient, ['vaultEntries']);
      invalidate(queryClient, ['recentFiles']);
      invalidate(queryClient, ['brainStats']);
      invalidate(queryClient, ['activeAgents']);
      invalidate(queryClient, ['projectActivity']);
    },
  }));
}

export function useCreateVaultEntry() {
  return useMutation(() => ({
    mutationFn: createVaultEntry,
    onSuccess: () => invalidate(queryClient, ['vaultEntries']),
  }));
}

export function useCreateTask() {
  return useMutation(() => ({
    mutationFn: createTask,
    onSuccess: () => invalidate(queryClient, ['tasks']),
  }));
}

export function useUpdateTask() {
  return useMutation(() => ({
    mutationFn: (vars: { id: string; data: Partial<Task> }) => updateTask(vars.id, vars.data),
    onSuccess: () => invalidate(queryClient, ['tasks']),
  }));
}

export function useCreateHandoff() {
  return useMutation(() => ({
    mutationFn: createHandoff,
    onSuccess: () => invalidate(queryClient, ['handoffs']),
  }));
}

export function useSetProviderConfig() {
  return useMutation(() => ({
    mutationFn: (cfg: Partial<ProviderConfig>) => setProviderConfig(cfg),
    onSuccess: () => invalidate(queryClient, qk.providerConfig),
  }));
}

export function useBuildCodeGraph() {
  return useMutation(() => ({
    mutationFn: (projectId: string) => buildCodeGraph(projectId),
    onSuccess: (res, projectId) => {
      queryClient.setQueryData(qk.codeGraph(projectId), res);
    },
  }));
}

export function useGenerateSessionDigest() {
  return useMutation(() => ({
    mutationFn: (vars: { projectId: string; sessionId?: string }) =>
      generateSessionDigest(vars.projectId, vars.sessionId),
    onSuccess: (_res, vars) => invalidate(queryClient, qk.sessionDigests(vars.projectId)),
  }));
}

export function useCreateMCPConnection() {
  return useMutation(() => ({
    mutationFn: createMCPConnection,
    onSuccess: () => invalidate(queryClient, qk.mcpConnections),
  }));
}

export function useDeleteMCPConnection() {
  return useMutation(() => ({
    mutationFn: (id: string) => deleteMCPConnection(id),
    onSuccess: () => invalidate(queryClient, qk.mcpConnections),
  }));
}

export function useResetAllData() {
  return useMutation(() => ({
    mutationFn: resetAllData,
    onSuccess: () => queryClient.clear(),
  }));
}





