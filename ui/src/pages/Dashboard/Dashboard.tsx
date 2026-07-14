import { Component, createMemo, Show, For, createEffect } from 'solid-js';
import { useSearchParams, useNavigate } from '@solidjs/router';
import {
  Star,
  CheckCircle2,
  GitBranch,
  Clock,
  ChevronDown,
  ChevronRight,
  Plus,
  Calendar,
  Github,
} from 'lucide-solid';
import { useAuth } from '../../api/auth';
import {
  buildKnowledgeGraph,
  buildCodeGraphKnowledgeGraph,
  type Project,
  type KnowledgeGraphNode,
} from '../../api/client';
import {
  useProjects,
  useTasks,
  useHandoffs,
  useVaultEntries,
  useRecentFiles,
  useDaemonStatus,
  useProjectBrainStats,
  useActiveAgents,
  useProjectActivity,
  useCodeGraph,
} from '../../api/queries';
import { ProjectBrainCard } from './ProjectBrainCard';
import { ActivityFeedCard } from './ActivityFeedCard';
import { AgentsCard } from './AgentsCard';
import { BubbleKnowledgeCard } from './BubbleKnowledgeCard';
import { ContextGeneratorCard } from './ContextGeneratorCard';
import { ProjectTimelineCard } from './ProjectTimelineCard';
import { QuickActionsCard } from './QuickActionsCard';
import { StatusBar } from './StatusBar';
import './Dashboard.css';
import { createProjectSelection } from '../../api/projectSelection';

// ── Dashboard Main ─────────────────────────────────────

export const DashboardMain: Component = () => {
  const { user } = useAuth();
  const navigate = useNavigate();
  
  const [searchParams, setSearchParams] = useSearchParams();

  // ── Project selection and server state ─────────────────
  const projectsQuery = useProjects();
  const projects = () => projectsQuery.data;
  const projectSelection = createProjectSelection(
    () => typeof searchParams.project === 'string' ? searchParams.project : undefined,
    projects,
  );
  const selectedProjectId = () => projectSelection.selected() || null;
  const setSelectedProjectId = (id: string) => {
    projectSelection.select(id);
    setSearchParams({ project: id }, { replace: true });
  };
  const daemonQuery = useDaemonStatus();
  const daemonStatus = () => daemonQuery.data;

  // Pin the initial project so it doesn't jump around when backend ordering changes
  createEffect(() => {
    const p = projects();
    if (p && p.length > 0 && !selectedProjectId()) {
      const initial = p.find(pr => pr.status === 'active') ?? p[0];
      setSelectedProjectId(initial.id);
    }
  });

  // ── Derived: active project ──
  const activeProject = createMemo<Project | null>(() => {
    const p = projects();
    if (!p?.length) return null;
    const selected = selectedProjectId();
    if (selected) {
      return p.find((pr) => pr.id === selected) ?? p[0];
    }
    return p.find((pr) => pr.status === 'active') ?? p[0];
  });

  const projectId = createMemo(() => activeProject()?.id ?? '');
  const projectScope = () => ({ projectId: projectId() });

  // ── Scoped data (per-project) ──────────────────────
  // Keep every dashboard card on the active project's query key. Without
  // this, the bubble graph and file totals showed data from all repositories.
  const tasksQuery = useTasks(projectScope);
  const tasks = () => tasksQuery.data;
  const handoffsQuery = useHandoffs(projectScope);
  const handoffs = () => handoffsQuery.data;
  const vaultQuery = useVaultEntries(projectScope);
  const vaultEntries = () => vaultQuery.data;
  const filesQuery = useRecentFiles(500, projectId);
  const allFiles = () => filesQuery.data;
  const brainStatsQuery = useProjectBrainStats(projectId);
  const brainStats = () => brainStatsQuery.data;
  const agentsQuery = useActiveAgents(projectId);
  const agents = () => agentsQuery.data;
  const projectActivityQuery = useProjectActivity(projectId, 5);
  const projectActivity = () => projectActivityQuery.data;
  const codeGraphQuery = useCodeGraph(projectId);
  const codeGraph = () => codeGraphQuery.data;

  // Prefer the actual persisted code graph. The memory summary remains a
  // fallback for projects scanned before automatic graph building existed.
  const graphNodes = createMemo<KnowledgeGraphNode[]>(() => {
    const g = codeGraph();
    if (g?.built) return buildCodeGraphKnowledgeGraph(g);
    return buildKnowledgeGraph(
      vaultEntries() ?? [],
      tasks() ?? [],
      handoffs() ?? [],
      allFiles() ?? [],
    );
  });
  const graphSource = createMemo(() => (codeGraph()?.built ? 'Code graph' : 'Project memory'));

  // ── Aggregated counts ─────────────────────────────
  const projectCount = createMemo(() => projects()?.length ?? 0);
  const totalMemories = createMemo(() => vaultEntries()?.length ?? 0);
  const handoffCount = createMemo(() => handoffs()?.length ?? 0);
  const decisionCount = createMemo(() =>
    (vaultEntries() ?? []).filter((e) => e.category === 'decision').length
  );
  const teamMembers = createMemo(() => {
    const agents_list = new Set<string>();
    for (const v of vaultEntries() ?? []) {
      if (v.source_agent) agents_list.add(v.source_agent);
    }
    for (const h of handoffs() ?? []) {
      if (h.from_agent) agents_list.add(h.from_agent);
      if (h.to_agent) agents_list.add(h.to_agent);
    }
    return agents_list.size;
  });

  // ── Active project stats ─────────────────────────
  const projectProgress = createMemo(() => activeProject()?.progress ?? 0);
  const openTasks = createMemo(() =>
    (tasks() ?? []).filter(
      (t) =>
        t.project === projectId() &&
        (t.status === 'todo' || t.status === 'in_progress')
    ).length
  );
  const activeAgentCount = createMemo(
    () => (agents() ?? []).filter((a) => a.status === 'active' || a.status === 'connected').length
  );

  // ── Time formatting ──────────────────────────────
  const lastUpdated = createMemo(() => {
    const p = activeProject();
    if (!p?.last_activity) return '—';
    const d = new Date(p.last_activity);
    const now = Date.now();
    const diffH = Math.floor((now - d.getTime()) / 3600000);
    if (diffH < 1) return 'Just now';
    if (diffH < 24) return `${diffH}h ago`;
    return `${Math.floor(diffH / 24)}d ago`;
  });

  // ── Scan state (real data, no fabricated commit/branch) ──
  const isScanned = createMemo(() => !!activeProject()?.last_scanned);
  const lastScanned = createMemo(() => {
    const p = activeProject();
    if (!p?.last_scanned) return 'Never scanned';
    return new Date(p.last_scanned).toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  });
  const githubSlug = createMemo(() => {
    const url = activeProject()?.github_url;
    if (!url) return 'Not linked';
    return url.replace('https://github.com/', '').replace(/\.git$/, '');
  });

  // ── Related projects as breadcrumbs ───────────────
  const breadcrumbProjects = createMemo(() => {
    const p = projects();
    if (!p) return [];
    return p.filter((pr) => pr.status === 'active').slice(0, 4);
  });

  // Colors for breadcrumbs
  const breadcrumbColors = ['#E8326E', '#3B82F6', '#22C55E', '#F59E0B', '#A855F7'];

  // ── Reorder projects to keep active first ────────
  const orderedProjects = createMemo(() => {
    const p = projects();
    if (!p || p.length === 0) return [];
    const activeId = projectId();
    const activeProj = p.find(proj => proj.id === activeId);
    if (!activeProj) return p;
    return [activeProj, ...p.filter(proj => proj.id !== activeId)];
  });

  return (
    <div class="dashboard-v2">
      {/* ── Breadcrumbs Row ──────────────────────────── */}
      <div class="dash-breadcrumbs-container">
        <div class="dash-breadcrumbs">
          <button class="dash-tab-plus" title="Add project" onClick={() => navigate('/projects')}><Plus size={16} /></button>
          
          <For each={orderedProjects()}>
            {(proj) => {
              const isActive = () => proj.id === projectId();
              return (
                <button 
                  class={`dash-tab ${isActive() ? 'active' : 'inactive'}`}
                  onClick={() => setSelectedProjectId(proj.id)}
                >
                  <div class={`dash-tab-icon ${isActive() ? 'dark-bg' : 'light-bg'}`}>
                    <span style={{ color: isActive() ? 'var(--bg-surface)' : 'var(--text-primary)', 'font-size': '10px' }}>
                      {proj.name?.charAt(0)?.toUpperCase() || 'P'}
                    </span>
                  </div>
                  {proj.name}
                  <Show when={isActive()} fallback={<ChevronRight size={14} class="ml-1" />}>
                    <ChevronDown size={14} class="ml-1" />
                  </Show>
                </button>
              );
            }}
          </For>
        </div>

        <div class="dash-timeframe-container">
          <div class="dash-timeframe">
            <Calendar size={14} />
            <span>Last updated</span>
            <span style={{ color: 'var(--text-primary)', 'font-weight': '500' }}>
              {lastUpdated()}
            </span>
          </div>
        </div>
      </div>

      {/* ── Top Row: Project Info + Project Brain ─────── */}
      <div class="dash-top-row">
        {/* Project Info Card */}
        <div class="dash-project-info">
          <div class="dash-project-header">
            <div
              class="dash-project-avatar"
              style={{
                background: activeProject()?.icon_color || 'var(--primary-dark)',
                color: '#fff',
                'border-radius': '50%'
              }}
            >
              {activeProject()?.name?.charAt(0)?.toUpperCase() || 'A'}
            </div>
            <div class="dash-project-title-block">
              <div class="dash-project-name">
                {activeProject()?.name || 'No project selected'}
                <Star size={18} color="#F59E0B" fill="#F59E0B" />
              </div>
              <div class="dash-project-desc">
                {activeProject()?.description || 'No description yet.'}
              </div>
            </div>
            <div class="dash-project-status">
              <CheckCircle2 size={14} />
              {activeProject()?.status === 'active'
                ? 'On Track'
                : activeProject()?.status || 'On Track'}
            </div>
          </div>

          {/* Stat Boxes Row */}
          <div class="dash-project-stats-boxes">
            <div class="dash-stat-box">
              <div class="dash-stat-box-label accent">Progress</div>
              <div class="dash-stat-box-value">{projectProgress()}%</div>
              <div class="dash-stat-box-bar">
                <div
                  class="dash-stat-box-bar-fill accent"
                  style={{ width: `${projectProgress()}%` }}
                />
              </div>
            </div>
            
            <div class="dash-stat-box">
              <div class="dash-stat-box-label accent">Open Tasks</div>
              <div class="dash-stat-box-value">{openTasks()}</div>
            </div>

            <div class="dash-stat-box">
              <div class="dash-stat-box-label accent">Active Agents</div>
              <div class="dash-agent-avatars" style={{ "margin-top": "8px" }}>
                <For each={(agents() ?? []).slice(0, 3)}>
                  {(agent, i) => (
                    <div
                      class="dash-agent-avatar"
                      style={{
                        background: breadcrumbColors[i() % breadcrumbColors.length],
                        'z-index': 10 - i(),
                      }}
                      title={agent.name}
                    >
                      {agent.name?.charAt(0)?.toUpperCase() || '?'}
                    </div>
                  )}
                </For>
                <Show when={activeAgentCount() > 0}>
                  <span class="dash-agent-avatar-count">+{activeAgentCount()}</span>
                </Show>
              </div>
            </div>

            <div class="dash-stat-box">
              <div class="dash-stat-box-label accent">Last Updated</div>
              <div class="dash-stat-box-value" style={{ "font-size": "16px" }}>
                {lastUpdated()}
              </div>
            </div>
          </div>

          {/* Git Info Box */}
          <div class="dash-git-info-box">
            <div class="dash-git-icon">
              <Github size={24} />
            </div>
            <div class="dash-git-item" style={{ flex: 2 }}>
              <span class="dash-git-label">GitHub</span>
              <Show
                when={activeProject()?.github_url}
                fallback={<span class="dash-git-value" style={{ 'font-size': '13px' }}>Not linked</span>}
              >
                <a
                  class="dash-git-value"
                  href={activeProject()!.github_url}
                  target="_blank"
                  rel="noreferrer"
                  style={{ 'font-size': '13px', color: 'inherit', 'text-decoration': 'none' }}
                >
                  {githubSlug()}
                </a>
              </Show>
            </div>
            <div class="dash-git-item" style={{ flex: 1.5 }}>
              <span class="dash-git-label">Last Scanned</span>
              <span class="dash-git-value" style={{ 'font-size': '13px' }}>{lastScanned()}</span>
            </div>
            <div class="dash-git-item" style={{ flex: 1 }}>
              <span class="dash-git-label">Indexed</span>
              <span class="dash-git-value">{projectProgress()}%</span>
            </div>
            <div class="dash-git-item" style={{ flex: 1 }}>
              <span class="dash-git-label">Status</span>
              <div class="dash-sync-status">
                <span class="dash-sync-dot" style={{ background: isScanned() ? undefined : 'var(--text-muted)' }} />
                <span class="dash-git-value">{isScanned() ? 'Scanned' : 'Not scanned'}</span>
              </div>
            </div>
          </div>
        </div>

        {/* Project Brain Card */}
        <ProjectBrainCard stats={brainStats() ?? null} loading={brainStatsQuery.isLoading} />
      </div>

      {/* ── Middle Row: Activity · Agents · Bubble Graph ─ */}
      <div class="dash-mid-row">
        <ActivityFeedCard
          activities={projectActivity() ?? []}
          loading={projectActivityQuery.isLoading}
        />
        <AgentsCard agents={agents() ?? []} loading={agentsQuery.isLoading} />
        <BubbleKnowledgeCard nodes={graphNodes()} source={graphSource()} />
      </div>

      {/* ── Bottom Row: Context · Timeline · Quick Actions ── */}
      <div class="dash-bottom-row">
        <ContextGeneratorCard
          decisions={decisionCount()}
          files={(allFiles() ?? []).length}
          tasks={(tasks() ?? []).filter(
            (t) => t.project === projectId() && t.status !== 'done'
          ).length}
          onGenerate={() =>
            navigate(projectId() ? `/ai-context?project=${projectId()}` : '/ai-context')
          }
        />
        <ProjectTimelineCard
          vaultEntries={vaultEntries() ?? []}
          handoffs={handoffs() ?? []}
          projectId={projectId()}
        />
        <QuickActionsCard onNavigate={(path) => navigate(path)} />
      </div>

      {/* ── Status Bar ──────────────────────────────── */}
      <StatusBar
        daemonReady={daemonStatus()?.ready ?? false}
        projectCount={projectCount()}
        totalMemories={totalMemories()}
        handoffCount={handoffCount()}
        decisionCount={decisionCount()}
        teamMembers={teamMembers()}
      />
    </div>
  );
};
