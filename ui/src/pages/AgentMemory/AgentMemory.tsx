import { Component, For, Show, createSignal, createMemo } from 'solid-js';
import { useSearchParams } from '@solidjs/router';
import {
  BrainCircuit,
  ChevronDown,
  Bot,
  Clock,
  Users,
  Layers,
  MessageSquare,
  X,
  RefreshCw,
} from 'lucide-solid';
import { type AgentMemory } from '../../api/client';
import { useProjects, useAgentMemories } from '../../api/queries';
import { ForceGraph, type FGNode, type FGEdge } from '../../components/ForceGraph/ForceGraph';
import { ProjectSelect } from '../../components/ProjectSelect/ProjectSelect';
import { createProjectSelection } from '../../api/projectSelection';
import { createPersistedSignal } from '../../api/persistedState';
import './AgentMemory.css';

// Category colors for individual memory nodes.
const CATEGORY_COLORS: Record<string, string> = {
  context: '#6C63FF',
  progress: '#3DCB6C',
  decision: '#E8326E',
  note: '#718096',
  handoff: '#F59E0B',
};

const PROJECT_COLOR = '#E8326E';
const SESSION_COLOR = '#06B6D4';

// Known agent brand colors (matched loosely by name).
const AGENT_BRANDS: Record<string, string> = {
  claude: '#D4A574',
  antigravity: '#4F46E5',
  gemini: '#4285F4',
  codex: '#10A37F',
  kiro: '#FF6B35',
  cursor: '#5B5BD6',
  copilot: '#6E40C9',
  windsurf: '#09B6A2',
  aider: '#9333EA',
  cline: '#0EA5E9',
};

function agentColor(name: string): string {
  const lower = name.toLowerCase();
  for (const key of Object.keys(AGENT_BRANDS)) {
    if (lower.includes(key)) return AGENT_BRANDS[key];
  }
  let hash = 0;
  for (let i = 0; i < name.length; i++) hash = ((hash << 5) - hash + name.charCodeAt(i)) | 0;
  return `hsl(${Math.abs(hash) % 360}, 55%, 52%)`;
}

const CATEGORY_LEGEND = [
  { label: 'Project', color: PROJECT_COLOR },
  { label: 'Agent', color: '#4F46E5' },
  { label: 'Session', color: SESSION_COLOR },
  { label: 'Context', color: CATEGORY_COLORS.context },
  { label: 'Progress', color: CATEGORY_COLORS.progress },
  { label: 'Decision', color: CATEGORY_COLORS.decision },
  { label: 'Note', color: CATEGORY_COLORS.note },
  { label: 'Handoff', color: CATEGORY_COLORS.handoff },
];

type NodeType = 'project' | 'agent' | 'session' | 'memory';

interface MemNode {
  id: string;
  type: NodeType;
  label: string;
  color: string;
  agent?: string;
  memory?: AgentMemory;
  sessionMemories?: AgentMemory[];
  sessionDate?: string;
}

function agentName(m: AgentMemory): string {
  return (m.client_name || m.ide || 'unknown').trim() || 'unknown';
}

function fmtDateTime(iso: string): string {
  if (!iso) return 'unknown time';
  try {
    return new Date(iso).toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return iso;
  }
}

function firstLine(s: string): string {
  const t = (s || '').trim().split('\n')[0];
  return t.length > 48 ? t.slice(0, 48) + '…' : t;
}

export const AgentMemoryPage: Component = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const [agentFilter, setAgentFilter] = createPersistedSignal<string>('agent-memory.agent-filter', 'all');
  const [showMemoryNodes, setShowMemoryNodes] = createPersistedSignal('agent-memory.show-memory-nodes', false);
  const [selectedNode, setSelectedNode] = createSignal<MemNode | null>(null);

  const projectsQuery = useProjects();
  const projects = () => projectsQuery.data;
  const projectSelection = createProjectSelection(
    () => typeof searchParams.project === 'string' ? searchParams.project : undefined,
    projects,
  );
  const selectedProject = projectSelection.selected;
  const setSelectedProject = projectSelection.select;
  const memoriesQuery = useAgentMemories(selectedProject);
  const memories = () => memoriesQuery.data;
  const refetch = () => memoriesQuery.refetch();

  const agents = createMemo(() => {
    const set = new Map<string, number>();
    for (const m of memories() ?? []) {
      const n = agentName(m);
      set.set(n, (set.get(n) ?? 0) + 1);
    }
    return Array.from(set.entries()).map(([name, count]) => ({ name, count }));
  });

  const sessionCount = createMemo(() => {
    const set = new Set<string>();
    for (const m of memories() ?? []) {
      set.add(`${agentName(m)}::${m.session_id || m.created.slice(0, 10)}`);
    }
    return set.size;
  });

  const filtered = createMemo<AgentMemory[]>(() => {
    const all = memories() ?? [];
    if (agentFilter() === 'all') return all;
    return all.filter((m) => agentName(m) === agentFilter());
  });

  // ── Build graph data (Project → Agent → Session → Memory) ──
  const built = createMemo(() => {
    const nodes: FGNode[] = [];
    const edges: FGEdge[] = [];
    const byId = new Map<string, MemNode>();
    const mems = filtered();
    if (mems.length === 0) return { nodes, edges, byId };
    const visibleMemoryIds = new Set(
      showMemoryNodes()
        ? [...mems]
            .sort((a, b) => new Date(b.created).getTime() - new Date(a.created).getTime())
            .slice(0, 240)
            .map((m) => m.id)
        : [],
    );

    const add = (data: MemNode, radius: number, labelAlways = false) => {
      nodes.push({ id: data.id, label: data.label, color: data.color, radius, labelAlways });
      byId.set(data.id, data);
    };

    const proj = projects()?.find((p) => p.id === selectedProject());
    add({ id: 'project', type: 'project', label: proj?.name || 'Project', color: PROJECT_COLOR }, 15, true);

    // Group: agent -> session -> memories
    const grouped = new Map<string, Map<string, AgentMemory[]>>();
    for (const m of mems) {
      const a = agentName(m);
      const sess = m.session_id || `day-${m.created.slice(0, 10)}`;
      if (!grouped.has(a)) grouped.set(a, new Map());
      const sMap = grouped.get(a)!;
      if (!sMap.has(sess)) sMap.set(sess, []);
      sMap.get(sess)!.push(m);
    }

    for (const [agent, sessions] of grouped) {
      const aId = `agent:${agent}`;
      add({ id: aId, type: 'agent', label: agent, color: agentColor(agent), agent }, 11, true);
      edges.push({ source: 'project', target: aId });

      for (const [sess, list] of sessions) {
        const sorted = [...list].sort(
          (a, b) => new Date(a.created).getTime() - new Date(b.created).getTime()
        );
        const sessionDate = sorted[0]?.created ?? '';
        const sId = `session:${agent}:${sess}`;
        add(
          {
            id: sId,
            type: 'session',
            label: fmtDateTime(sessionDate),
            color: SESSION_COLOR,
            agent,
            sessionMemories: sorted,
            sessionDate,
          },
          8
        );
        edges.push({ source: aId, target: sId });

        for (const m of list) {
          if (!visibleMemoryIds.has(m.id)) continue;
          const mId = `mem:${m.id}`;
          add(
            {
              id: mId,
              type: 'memory',
              label: m.title || m.category,
              color: CATEGORY_COLORS[m.category] || CATEGORY_COLORS.note,
              agent,
              memory: m,
            },
            5.5
          );
          edges.push({ source: sId, target: mId });
        }
      }
    }
    return { nodes, edges, byId };
  });

  const handleSelect = (id: string | null) => {
    setSelectedNode(id ? built().byId.get(id) ?? null : null);
  };

  const sessionsForAgent = (agent: string) => {
    const set = new Set<string>();
    for (const m of filtered()) {
      if (agentName(m) === agent) set.add(m.session_id || `day-${m.created.slice(0, 10)}`);
    }
    return set.size;
  };
  const memoriesForAgent = (agent: string) =>
    filtered().filter((m) => agentName(m) === agent).length;

  return (
    <div class="codegraph-page">
      <div class="cg-layout">
        {/* Left column — controls & stats (narrow) */}
        <aside class="cg-sidebar">
          <div class="page-header">
            <div class="page-title-row">
              <div class="page-title-icon violet">
                <BrainCircuit size={24} />
              </div>
              <div>
                <h1 class="page-title">Agent Memory Graph</h1>
                <p class="page-subtitle">
                  Everything your AI agents remembered on this project — grouped by agent
                  (Kiro, Antigravity, Gemini, Codex…) and by the date &amp; time of each session.
                </p>
              </div>
            </div>
          </div>

          <div class="codegraph-toolbar">
            <ProjectSelect
              projects={projects() || []}
              selectedId={selectedProject()}
              onChange={(id) => {
                setSelectedProject(id);
                setSearchParams({ project: id }, { replace: true });
                setSelectedNode(null);
                setAgentFilter('all');
              }}
              placeholder="Choose a project…"
            />

            <Show when={agents().length > 0}>
              <label class="am-view-toggle">
                <input
                  type="checkbox"
                  checked={showMemoryNodes()}
                  onChange={(e) => {
                    setShowMemoryNodes(e.currentTarget.checked);
                    setSelectedNode(null);
                  }}
                />
                Show memory details
              </label>
              <div class="custom-select">
                <select
                  value={agentFilter()}
                  onChange={(e) => {
                    setAgentFilter(e.currentTarget.value);
                    setSelectedNode(null);
                  }}
                  aria-label="Filter by agent"
                >
                  <option value="all">All agents</option>
                  <For each={agents()}>
                    {(a) => <option value={a.name}>{a.name} ({a.count})</option>}
                  </For>
                </select>
                <ChevronDown size={16} class="select-icon" />
              </div>
            </Show>

            <button
              class="btn secondary"
              onClick={() => refetch()}
              disabled={!selectedProject() || memoriesQuery.isLoading}
            >
              <RefreshCw size={15} class={memoriesQuery.isLoading ? 'spin' : ''} />
              Refresh
            </button>
          </div>

          {/* Stats bar */}
          <Show when={selectedProject() && (memories()?.length ?? 0) > 0}>
            <div class="cg-stats" role="list">
              <div class="cg-stat" role="listitem"><Users size={14} /> {agents().length} agents</div>
              <div class="cg-stat" role="listitem"><Clock size={14} /> {sessionCount()} sessions</div>
              <div class="cg-stat" role="listitem"><Layers size={14} /> {memories()!.length} memories</div>
              <div class="cg-stat accent" role="listitem">{filtered().length} shown</div>
            </div>
          </Show>
        </aside>

        {/* Right column — graph canvas (wide) */}
        <div class="cg-canvas-wrap">
          <Show when={!selectedProject()}>
            <div class="cg-placeholder" role="status">
              <BrainCircuit size={44} />
              <h3>Select a project</h3>
              <p>Choose a project to explore the memory its AI agents saved over time.</p>
            </div>
          </Show>

          <Show when={selectedProject() && (memories()?.length ?? 0) === 0}>
            <div class="cg-placeholder" role="status">
              <Bot size={44} />
              <h3>{memoriesQuery.isLoading ? 'Loading…' : 'No agent memory yet'}</h3>
              <p>
                Connect an AI client on the <strong>MCP Server</strong> page and let it call
                <code> cortex_save_memory</code>. Saved memories appear here, grouped by agent and session.
              </p>
            </div>
          </Show>

          <Show when={(built().nodes.length ?? 0) > 0}>
            <ForceGraph
              nodes={built().nodes}
              edges={built().edges}
              selectedId={selectedNode()?.id ?? null}
              onSelect={handleSelect}
            />
          </Show>

          {/* Inspector */}
          <Show when={selectedNode()}>
            {(n) => (
              <aside class="cg-inspector" aria-label="Node details">
                <button class="cg-inspector-close" onClick={() => setSelectedNode(null)} aria-label="Close">
                  <X size={15} />
                </button>
                <div class="cg-inspector-type" style={{ color: n().color }}>
                  {n().type}
                </div>

                <Show when={n().type === 'memory' && n().memory}>
                  <h3 class="cg-inspector-title">{n().memory!.title || '(untitled memory)'}</h3>
                  <div class="am-meta-row">
                    <span class="am-chip" style={{ background: n().color + '22', color: n().color }}>
                      {n().memory!.category}
                    </span>
                    <span class="am-chip agent">
                      <Bot size={11} /> {agentName(n().memory!)}
                    </span>
                  </div>
                  <div class="cg-inspector-path">
                    <Clock size={11} /> {fmtDateTime(n().memory!.created)}
                    <Show when={n().memory!.session_id}>
                      {' · session '}{n().memory!.session_id.slice(0, 8)}
                    </Show>
                  </div>
                  <div class="am-memory-content">{n().memory!.content}</div>
                  <Show when={(n().memory!.tags?.length ?? 0) > 0}>
                    <div class="am-tags">
                      <For each={n().memory!.tags}>{(t) => <span class="am-tag">{t}</span>}</For>
                    </div>
                  </Show>
                </Show>

                <Show when={n().type === 'agent'}>
                  <h3 class="cg-inspector-title">{n().label}</h3>
                  <div class="cg-inspector-degree">
                    {sessionsForAgent(n().agent!)} sessions · {memoriesForAgent(n().agent!)} memories
                  </div>
                  <button
                    class="btn secondary small am-focus-btn"
                    onClick={() => {
                      setAgentFilter(n().agent!);
                      setSelectedNode(null);
                    }}
                  >
                    Focus this agent
                  </button>
                </Show>

                <Show when={n().type === 'session' && n().sessionMemories}>
                  <h3 class="cg-inspector-title">Session · {fmtDateTime(n().sessionDate!)}</h3>
                  <div class="cg-inspector-degree">
                    {agentName(n().sessionMemories![0])} · {n().sessionMemories!.length} memories
                  </div>
                  <div class="cg-inspector-sections">
                    <div class="cg-inspector-section">
                      <div class="cg-inspector-section-title">
                        Memories <span class="cg-count">{n().sessionMemories!.length}</span>
                      </div>
                      <ul class="cg-inspector-items">
                        <For each={n().sessionMemories!}>
                          {(m) => (
                            <li
                              onClick={() => setSelectedNode(built().byId.get(`mem:${m.id}`) ?? null)}
                              style={{ cursor: 'pointer' }}
                            >
                              <MessageSquare size={11} /> [{m.category}] {m.title || firstLine(m.content)}
                            </li>
                          )}
                        </For>
                      </ul>
                    </div>
                  </div>
                </Show>

                <Show when={n().type === 'project'}>
                  <h3 class="cg-inspector-title">{n().label}</h3>
                  <div class="cg-inspector-degree">
                    {agents().length} agents · {sessionCount()} sessions · {(memories()?.length ?? 0)} memories
                  </div>
                </Show>
              </aside>
            )}
          </Show>

          <div class="cg-legend">
            <For each={CATEGORY_LEGEND}>
              {(item) => (
                <div class="cg-legend-item">
                  <span class="cg-legend-dot" style={{ background: item.color }} />
                  {item.label}
                </div>
              )}
            </For>
          </div>
        </div>
      </div>
    </div>
  );
};
