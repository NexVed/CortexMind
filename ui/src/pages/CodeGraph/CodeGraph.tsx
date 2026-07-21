import { Component, For, Show, createSignal, createMemo } from 'solid-js';
import {
  Network,
  ChevronDown,
  Boxes,
  RefreshCw,
  Search,
  FileCode2,
  Folder,
  Package,
  FunctionSquare,
  Box,
  X,
} from 'lucide-solid';
import {
  type CodeGraphNode,
} from '../../api/client';
import { useProjects, useCodeGraph, useBuildCodeGraph } from '../../api/queries';
import { ForceGraph, type FGNode, type FGEdge } from '../../components/ForceGraph/ForceGraph';
import { ProjectSelect } from '../../components/ProjectSelect/ProjectSelect';
import { createProjectSelection } from '../../api/projectSelection';
import { createPersistedSignal } from '../../api/persistedState';
import './CodeGraph.css';

const TYPE_COLORS: Record<string, string> = {
  dir: '#718096',
  file: '#6C63FF',
  function: '#3DCB6C',
  class: '#E8326E',
  package: '#F59E0B',
};

const legend = [
  { type: 'dir', label: 'Directory' },
  { type: 'file', label: 'File' },
  { type: 'function', label: 'Function' },
  { type: 'class', label: 'Class' },
  { type: 'package', label: 'Package' },
];

export const CodeGraphPage: Component = () => {
  const [error, setError] = createSignal('');
  const [showPackages, setShowPackages] = createPersistedSignal('code-graph.show-packages.v2', true);
  const [showSymbols, setShowSymbols] = createPersistedSignal('code-graph.show-symbols.v2', true);
  const MAX_VISIBLE_NODES = 220;
  const [search, setSearch] = createPersistedSignal('code-graph.search', '');
  const [selectedNode, setSelectedNode] = createSignal<CodeGraphNode | null>(null);
  type GraphFocus = 'dirs' | 'files' | 'functions' | 'classes' | 'packages' | 'internal' | 'external' | 'cycles' | 'orphans';
  const [focus, setFocus] = createSignal<GraphFocus | null>(null);

  const projectsQuery = useProjects();
  const projects = () => projectsQuery.data;
  const projectSelection = createProjectSelection(undefined, projects);
  const selectedProject = projectSelection.selected;
  const setSelectedProject = projectSelection.select;
  const graphQuery = useCodeGraph(selectedProject);
  const graph = () => graphQuery.data;
  const buildM = useBuildCodeGraph();
  const building = () => buildM.isPending;

  const stats = () => graph()?.stats;

  const handleBuild = async () => {
    if (!selectedProject()) {
      setError('Pick a project first.');
      return;
    }
    setError('');
    try {
      await buildM.mutateAsync(selectedProject());
    } catch (err: any) {
      setError(err?.message || 'Failed to build code graph');
    }
  };

  function visibleTypes(packs: boolean, syms: boolean): Set<string> {
    const t = new Set(['dir', 'file']);
    if (packs) t.add('package');
    if (syms) {
      t.add('function');
      t.add('class');
    }
    return t;
  }

  const cycleNodes = (edges: { source: string; target: string; rel: string }[]) => {
    const adjacent = new Map<string, string[]>();
    for (const edge of edges) {
      if (edge.rel !== 'imports') continue;
      adjacent.set(edge.source, [...(adjacent.get(edge.source) ?? []), edge.target]);
    }
    const visiting = new Set<string>();
    const visited = new Set<string>();
    const result = new Set<string>();
    const visit = (node: string, trail: string[]) => {
      if (visiting.has(node)) {
        for (const member of trail.slice(trail.indexOf(node))) result.add(member);
        return;
      }
      if (visited.has(node)) return;
      visiting.add(node);
      for (const next of adjacent.get(node) ?? []) visit(next, [...trail, node]);
      visiting.delete(node);
      visited.add(node);
    };
    for (const node of adjacent.keys()) visit(node, []);
    return result;
  };

  const focusedIds = createMemo<Set<string> | null>(() => {
    const g = graph();
    const active = focus();
    if (!g || !active) return null;
    if (active === 'dirs' || active === 'files' || active === 'functions' || active === 'classes' || active === 'packages') {
      const type = active === 'dirs' ? 'dir' : active === 'files' ? 'file' : active === 'functions' ? 'function' : active === 'classes' ? 'class' : 'package';
      return new Set(g.nodes.filter((node) => node.type === type).map((node) => node.id));
    }
    if (active === 'orphans') return new Set(g.nodes.filter((node) => node.degree === 0).map((node) => node.id));
    if (active === 'cycles') return cycleNodes(g.edges);
    const relationship = active === 'internal' ? 'imports' : 'depends_on';
    return new Set(g.edges.filter((edge) => edge.rel === relationship).flatMap((edge) => [edge.source, edge.target]));
  });

  const toggleFocus = (next: GraphFocus) => {
    if (next === 'functions' || next === 'classes') setShowSymbols(true);
    if (next === 'packages' || next === 'external') setShowPackages(true);
    setSelectedNode(null);
    setFocus((current) => current === next ? null : next);
  };
  // ── Reactive graph data for the shared ForceGraph ──
  const visibleNodes = createMemo<CodeGraphNode[]>(() => {
    const g = graph();
    if (!g || !g.built) return [];
    const types = visibleTypes(showPackages(), showSymbols());
    const ids = focusedIds();
    const candidates = g.nodes.filter((n) => types.has(n.type) && (!ids || ids.has(n.id)));
    if (candidates.length <= MAX_VISIBLE_NODES) return candidates;

    const ranked = (type?: string) => candidates
      .filter((node) => !type || node.type === type)
      .sort((a, b) => b.degree - a.degree || a.id.localeCompare(b.id));

    // The overview must represent every graph layer. A pure degree ranking
    // favours files, which previously hid functions/classes on first load.
    if (!focus()) {
      const selected: CodeGraphNode[] = [];
      const selectedIDs = new Set<string>();
      const add = (nodes: CodeGraphNode[], limit: number) => {
        for (const node of nodes) {
          if (selected.length >= MAX_VISIBLE_NODES || selectedIDs.has(node.id)) continue;
          selectedIDs.add(node.id);
          selected.push(node);
          if (--limit === 0) break;
        }
      };
      add(ranked('dir'), 60);
      add(ranked('package'), 25);
      add(ranked('function'), 75);
      add(ranked('class'), 45);
      add(ranked(), MAX_VISIBLE_NODES - selected.length);
      return selected;
    }

    return ranked().slice(0, MAX_VISIBLE_NODES);
  });

  const includedIds = createMemo(() => new Set(visibleNodes().map((n) => n.id)));

  const fgNodes = createMemo<FGNode[]>(() => {
    const g = graph();
    if (!g || !g.built) return [];
    const ids = includedIds();
    return visibleNodes()
      .filter((n) => ids.has(n.id))
      .map((n) => ({
        id: n.id,
        label: n.label,
        color: TYPE_COLORS[n.type] || '#6B7280',
        radius: Math.max(4, Math.min(14, 4 + Math.sqrt(n.degree) * 1.6)),
        labelAlways: n.type === 'dir',
      }));
  });

  const fgEdges = createMemo<FGEdge[]>(() => {
    const g = graph();
    if (!g || !g.built) return [];
    const ids = includedIds();
    return g.edges
      .filter((e) => ids.has(e.source) && ids.has(e.target))
      .map((e) => ({ source: e.source, target: e.target, strong: e.rel === 'depends_on' }));
  });

  const handleSelect = (id: string | null) => {
    if (!id) {
      setSelectedNode(null);
      return;
    }
    setSelectedNode(graph()?.nodes.find((n) => n.id === id) ?? null);
  };

  // Node inspector data derived from the full graph.
  const nodeDetail = () => {
    const sel = selectedNode();
    const g = graph();
    if (!sel || !g) return null;
    const defines: string[] = [];
    const dependsOn: string[] = [];
    const imports: string[] = [];
    const usedBy: string[] = [];
    const labelOf = (id: string) => g.nodes.find((n) => n.id === id)?.label || id;
    const pathOf = (id: string) => g.nodes.find((n) => n.id === id)?.path || labelOf(id);
    for (const e of g.edges) {
      if (e.source === sel.id) {
        if (e.rel === 'defines') defines.push(labelOf(e.target));
        else if (e.rel === 'depends_on') dependsOn.push(pathOf(e.target));
        else if (e.rel === 'imports') imports.push(labelOf(e.target));
      } else if (e.target === sel.id && e.rel === 'depends_on') {
        usedBy.push(pathOf(e.source));
      }
    }
    return { defines, dependsOn, imports, usedBy };
  };

  return (
    <div class="codegraph-page">
      <div class="cg-layout">
        {/* Left column — controls & stats (narrow) */}
        <aside class="cg-sidebar">
          <div class="page-header">
            <div class="page-title-row">
              <div class="page-title-icon violet">
                <Network size={24} />
              </div>
              <div>
                <h1 class="page-title">Code Graph</h1>
                <p class="page-subtitle">
                  A persistent map of your codebase — directories, files, symbols and their
                  dependencies. Shared with agents as queryable memory.
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
                setSelectedNode(null);
                setFocus(null);
              }}
              placeholder="Choose a project…"
            />

            <button class="btn primary" onClick={handleBuild} disabled={building() || !selectedProject()}>
              <Show when={!building()} fallback={<><RefreshCw size={15} class="spin" /> Building…</>}>
                <Boxes size={15} />
                {graph()?.built ? 'Rebuild graph' : 'Build graph'}
              </Show>
            </button>

            <div class="cg-search">
              <Search size={14} />
              <input
                type="search"
                placeholder="Highlight node…"
                value={search()}
                onInput={(e) => setSearch(e.currentTarget.value)}
                aria-label="Search nodes"
              />
            </div>

            <div class="cg-density-note">
              {graph()?.built ? 'Showing ' + visibleNodes().length + ' of ' + graph()!.nodes.length + ' nodes' : 'Overview mode'}
            </div>
            <div class="cg-toggle-group">
              <label class={`cg-toggle ${showPackages() ? 'on' : ''}`}>
                <input type="checkbox" checked={showPackages()} onChange={(e) => setShowPackages(e.currentTarget.checked)} />
                Packages
              </label>
              <label class={`cg-toggle ${showSymbols() ? 'on' : ''}`}>
                <input type="checkbox" checked={showSymbols()} onChange={(e) => setShowSymbols(e.currentTarget.checked)} />
                Symbols
              </label>
            </div>
          </div>

          <Show when={error()}>
            <div class="cg-error" role="alert">{error()}</div>
          </Show>

          {/* Stats bar */}
          <Show when={stats() && graph()?.built}>
            <div class="cg-stats" role="list" aria-label="Graph filters">
              <button class={`cg-stat ${focus() === 'dirs' ? 'active' : ''}`} type="button" role="listitem" aria-pressed={focus() === 'dirs'} onClick={() => toggleFocus('dirs')}><Folder size={14} /> {stats()!.dirs} dirs</button>
              <button class={`cg-stat ${focus() === 'files' ? 'active' : ''}`} type="button" role="listitem" aria-pressed={focus() === 'files'} onClick={() => toggleFocus('files')}><FileCode2 size={14} /> {stats()!.files} files</button>
              <button class={`cg-stat ${focus() === 'functions' ? 'active' : ''}`} type="button" role="listitem" aria-pressed={focus() === 'functions'} onClick={() => toggleFocus('functions')}><FunctionSquare size={14} /> {stats()!.functions} functions</button>
              <button class={`cg-stat ${focus() === 'classes' ? 'active' : ''}`} type="button" role="listitem" aria-pressed={focus() === 'classes'} onClick={() => toggleFocus('classes')}><Box size={14} /> {stats()!.classes} classes</button>
              <button class={`cg-stat ${focus() === 'packages' ? 'active' : ''}`} type="button" role="listitem" aria-pressed={focus() === 'packages'} onClick={() => toggleFocus('packages')}><Package size={14} /> {stats()!.packages} packages</button>
              <button class={`cg-stat ${focus() === 'internal' ? 'active' : ''}`} type="button" role="listitem" aria-pressed={focus() === 'internal'} onClick={() => toggleFocus('internal')}>{stats()!.internal_deps} internal deps</button>
              <button class={`cg-stat ${focus() === 'external' ? 'active' : ''}`} type="button" role="listitem" aria-pressed={focus() === 'external'} onClick={() => toggleFocus('external')}>{stats()!.external_deps} external deps</button>
              <button class={`cg-stat ${focus() === 'cycles' ? 'active' : ''}`} type="button" role="listitem" aria-pressed={focus() === 'cycles'} disabled={stats()!.cycles === 0} onClick={() => toggleFocus('cycles')}>{stats()!.cycles} cycles</button>
              <button class={`cg-stat ${focus() === 'orphans' ? 'active' : ''}`} type="button" role="listitem" aria-pressed={focus() === 'orphans'} onClick={() => toggleFocus('orphans')}>{stats()!.orphans} orphans</button>
            </div>
          </Show>
        </aside>

        {/* Right column — graph canvas (wide) */}
        <div class="cg-canvas-wrap">
          <Show when={!selectedProject()}>
            <div class="cg-placeholder" role="status">
              <Network size={44} />
              <h3>Select a project</h3>
              <p>Choose a project and build its code graph to explore the structure.</p>
            </div>
          </Show>

          <Show when={selectedProject() && !graph()?.built}>
            <div class="cg-placeholder" role="status">
              <Boxes size={44} />
              <h3>{graphQuery.isLoading ? 'Loading…' : 'No graph yet'}</h3>
              <p>
                Click <strong>Build graph</strong> to compile this project's indexed files into a
                codebase memory graph. (The project must be scanned first.)
              </p>
            </div>
          </Show>

          <Show when={graph()?.built}>
            <ForceGraph
              nodes={fgNodes()}
              edges={fgEdges()}
              selectedId={selectedNode()?.id ?? null}
              highlight={search()}
              onSelect={handleSelect}
            />
          </Show>

          {/* Node inspector */}
          <Show when={selectedNode()}>
            {(n) => (
              <aside class="cg-inspector" aria-label="Node details">
                <button class="cg-inspector-close" onClick={() => setSelectedNode(null)} aria-label="Close">
                  <X size={15} />
                </button>
                <div class="cg-inspector-type" style={{ color: TYPE_COLORS[n().type] }}>
                  {n().type}
                </div>
                <h3 class="cg-inspector-title">{n().label}</h3>
                <Show when={n().path}>
                  <div class="cg-inspector-path mono">{n().path}{n().line ? `:${n().line}` : ''}</div>
                </Show>
                <div class="cg-inspector-degree">{n().degree} connections</div>

                <Show when={nodeDetail()}>
                  {(d) => (
                    <div class="cg-inspector-sections">
                      <InspectorList title="Defines" items={d().defines} />
                      <InspectorList title="Depends on" items={d().dependsOn} accent />
                      <InspectorList title="Imports" items={d().imports} />
                      <InspectorList title="Used by" items={d().usedBy} accent />
                    </div>
                  )}
                </Show>
              </aside>
            )}
          </Show>

          <div class="cg-legend">
            <For each={legend}>
              {(item) => (
                <div class="cg-legend-item">
                  <span class="cg-legend-dot" style={{ background: TYPE_COLORS[item.type] }} />
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

const InspectorList: Component<{ title: string; items: string[]; accent?: boolean }> = (props) => (
  <Show when={props.items.length > 0}>
    <div class="cg-inspector-section">
      <div class="cg-inspector-section-title">
        {props.title} <span class="cg-count">{props.items.length}</span>
      </div>
      <ul class="cg-inspector-items">
        <For each={props.items.slice(0, 20)}>
          {(it) => <li class={`mono ${props.accent ? 'accent' : ''}`}>{it}</li>}
        </For>
      </ul>
    </div>
  </Show>
);
