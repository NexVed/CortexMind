import { Component, For, createSignal, Show, onCleanup } from 'solid-js';
import {
  Server,
  Activity,
  Box,
  Plus,
  Trash2,
  RefreshCcw,
  Clock,
  Copy,
  Check,
  AlertTriangle,
  BrainCircuit,
  BookOpen,
  Terminal,
  Shield,
  Brain,
  Users,
  X,
} from 'lucide-solid';
import {
  type MCPConnection,
} from '../../api/client';
import {
  useDaemonStatus,
  useProjects,
  useMCPConnections,
  useCreateMCPConnection,
  useDeleteMCPConnection,
} from '../../api/queries';
import {
  PLATFORMS,
  GROUP_LABELS,
  platformsByGroup,
  getPlatform,
  buildSnippet,
  type IntegrationGroup,
  type PlatformIntegration,
} from './integrations';
import './MCPServer.css';

const availableTools = [
  {
    name: 'cortex_get_context',
    desc: "Load the project's characterization (system prompt, tech stack, architecture) plus prior AI memory. Call this first.",
    category: 'MEMORY',
    icon: BrainCircuit,
    colorClass: 'purple',
  },
  {
    name: 'cortex_save_memory',
    desc: 'Persist progress, a decision or a note so the next AI session in any IDE can recall it.',
    category: 'MEMORY',
    icon: Brain,
    colorClass: 'blue',
  },
  {
    name: 'cortex_list_memories',
    desc: 'List stored memories for this project from all previous AI sessions and IDEs.',
    category: 'MEMORY',
    icon: BookOpen,
    colorClass: 'indigo',
  },
  {
    name: 'cortex_get_tasks',
    desc: 'List the active (not done) tasks for this project.',
    category: 'TASKS',
    icon: Activity,
    colorClass: 'pink',
  },
  {
    name: 'cortex_summarize_session',
    desc: 'Compress everything you did this session into a stored, token-efficient digest for the next agent.',
    category: 'DIGEST',
    icon: Shield,
    colorClass: 'purple',
  },
];

const GROUP_ORDER: IntegrationGroup[] = ['cli', 'web', 'ide'];

// Renders a platform-specific config + step-by-step setup guide.
const IntegrationGuide: Component<{
  platform: PlatformIntegration;
  endpoint: string;
  token?: string;
}> = (props) => {
  const [copied, setCopied] = createSignal(false);
  const snippet = () => buildSnippet(props.platform, props.endpoint, props.token);

  const copy = () => {
    navigator.clipboard.writeText(snippet());
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div class="integration-guide">
      <Show when={props.platform.requiresPublicUrl}>
        <div class="integration-warning">
          <AlertTriangle size={14} />
          <span>{props.platform.note || 'This client needs a public HTTPS URL — expose CORTEX with a tunnel.'}</span>
        </div>
      </Show>

      <Show when={props.platform.file}>
        <div class="integration-file">
          {props.platform.language === 'bash' ? 'Run in terminal' : props.platform.file}
        </div>
      </Show>

      <div class="conn-config">
        <pre>{snippet()}</pre>
        <button class="copy-btn" onClick={copy}>
          <Show when={!copied()} fallback={<><Check size={14} /> Copied</>}>
            <Copy size={14} /> Copy
          </Show>
        </button>
      </div>

      <ol class="integration-steps">
        <For each={props.platform.steps}>{(step) => <li>{step}</li>}</For>
      </ol>

      <Show when={props.platform.note && !props.platform.requiresPublicUrl}>
        <div class="integration-note">{props.platform.note}</div>
      </Show>
    </div>
  );
};

export const MCPServerPage: Component = () => {
  const statusQuery = useDaemonStatus();
  const status = () => statusQuery.data;
  const projectsQuery = useProjects();
  const projects = () => projectsQuery.data;
  const connectionsQuery = useMCPConnections();
  const connections = () => connectionsQuery.data;
  const createConnM = useCreateMCPConnection();
  const deleteConnM = useDeleteMCPConnection();

  const [lastRefetched, setLastRefetched] = createSignal(new Date());
  const [timeAgo, setTimeAgo] = createSignal('Just now');

  const handleRefetch = () => {
    statusQuery.refetch();
    connectionsQuery.refetch();
    setLastRefetched(new Date());
    setTimeAgo('Just now');
  };

  // Keep the "last updated" label fresh.
  const timer = setInterval(() => {
    const diff = Math.floor((Date.now() - lastRefetched().getTime()) / 60000);
    setTimeAgo(diff < 1 ? 'Just now' : `${diff} min ago`);
  }, 30000);
  onCleanup(() => clearInterval(timer));

  const [showForm, setShowForm] = createSignal(false);
  const [formProject, setFormProject] = createSignal('');
  const [formPlatform, setFormPlatform] = createSignal('cursor');
  const [formLabel, setFormLabel] = createSignal('');
  const [creating, setCreating] = createSignal(false);
  const [created, setCreated] = createSignal<MCPConnection | null>(null);
  const [copiedTok, setCopiedTok] = createSignal(false);
  const [error, setError] = createSignal('');
  const [guideFor, setGuideFor] = createSignal<string>(''); // connection id whose guide is open

  const projectName = (id: string) => projects()?.find((p) => p.id === id)?.name ?? '—';
  const platformLabel = (id: string) => getPlatform(id)?.label ?? id;

  const openForm = () => {
    setCreated(null);
    setError('');
    setFormLabel('');
    setShowForm(true);
  };

  const closeForm = () => {
    setShowForm(false);
    setCreated(null);
    setError('');
  };

  const handleCreate = async () => {
    if (!formProject()) {
      setError('Pick a project to bind this connection to.');
      return;
    }
    setCreating(true);
    setError('');
    try {
      const conn = await createConnM.mutateAsync({
        project_id: formProject(),
        ide: formPlatform(),
        label: formLabel() || `${platformLabel(formPlatform())} · ${projectName(formProject())}`,
      });
      setCreated(conn);
    } catch (err: any) {
      setError(err?.message || 'Failed to create connection');
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    await deleteConnM.mutateAsync(id);
    if (guideFor() === id) setGuideFor('');
  };

  const copyTok = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedTok(true);
    setTimeout(() => setCopiedTok(false), 2000);
  };

  const fmtTime = (iso: string) => (iso ? new Date(iso).toLocaleString() : 'never');

  return (
    <div class="mcp-page">
      <div class="page-header">
        <div class="page-title-row">
          <div class="page-title-icon pink">
            <Server size={24} />
          </div>
          <div>
            <h1 class="page-title">MCP Server</h1>
            <p class="page-subtitle">Connect any AI client to CortexMind's shared project brain.</p>
          </div>
        </div>
        <div class="page-header-actions">
          <button class="btn secondary" onClick={handleRefetch}>
            <RefreshCcw size={16} />
            Refresh
          </button>
          <button class="btn primary-pink" onClick={openForm}>
            <Plus size={16} />
            New Connection
          </button>
        </div>
      </div>

      {/* Status strip */}
      <div class="mcp-status-strip">
        <div class="status-tile">
          <span class="status-tile-label"><Server size={14} /> Server</span>
          <span class="status-tile-value">
            <span class="status-led" classList={{ on: !!status()?.ready }} />
            {status()?.ready ? 'Online' : 'Offline'}
          </span>
        </div>
        <div class="status-tile">
          <span class="status-tile-label"><Activity size={14} /> Endpoint</span>
          <span class="status-tile-value mono">/mcp</span>
        </div>
        <div class="status-tile">
          <span class="status-tile-label"><Box size={14} /> Version</span>
          <span class="status-tile-value">{status()?.version || '—'}</span>
        </div>
        <div class="status-tile">
          <span class="status-tile-label"><Users size={14} /> Connections</span>
          <span class="status-tile-value">{connections()?.length ?? 0}</span>
        </div>
        <div class="status-tile">
          <span class="status-tile-label"><Brain size={14} /> Semantic Search</span>
          <span class="status-tile-value">{status()?.semanticEnabled ? 'Enabled' : 'Disabled'}</span>
        </div>
        <div class="status-tile">
          <span class="status-tile-label"><Clock size={14} /> Updated</span>
          <span class="status-tile-value">{timeAgo()}</span>
        </div>
      </div>

      <div class="mcp-grid">
        {/* Connections — the main column */}
        <div class="mcp-main">
          <div class="mcp-panel">
            <div class="mcp-panel-header">
              <h3>Connections</h3>
              <span class="count-pill">{connections()?.length ?? 0}</span>
            </div>

            <Show
              when={(connections()?.length ?? 0) > 0}
              fallback={
                <div class="conn-empty">
                  <Plug />
                  <h4>No connections yet</h4>
                  <p>Create a connection to link Claude Code, Cursor, VS Code, Gemini CLI and other AI clients to this project's memory.</p>
                  <button class="btn primary-pink" onClick={openForm}>
                    <Plus size={16} /> New Connection
                  </button>
                </div>
              }
            >
              <div class="conn-list">
                <For each={connections()}>
                  {(conn) => {
                    const platform = () => getPlatform(conn.ide);
                    const open = () => guideFor() === conn.id;
                    return (
                      <div class="conn-item-wrap">
                        <div class="conn-item" classList={{ 'guide-open': open() }}>
                          <div
                            class="conn-status-dot"
                            classList={{ connected: conn.connected }}
                            title={conn.connected ? 'Active' : 'Idle'}
                          />
                          <div class="conn-icon">
                            <Show when={platform()?.icon} fallback={<Terminal size={18} class="conn-fallback-icon" />}>
                              <img src={platform()?.icon} class="conn-platform-icon" alt={platformLabel(conn.ide)} />
                            </Show>
                          </div>
                          <div class="conn-info">
                            <div class="conn-title">
                              <span class="conn-label-text">{conn.label || platformLabel(conn.ide)}</span>
                              <span class="conn-ide-tag">{platformLabel(conn.ide)}</span>
                              <Show when={conn.client_name}>
                                <span class="conn-client-tag">{conn.client_name}</span>
                              </Show>
                            </div>
                            <div class="conn-meta">
                              <span><BrainCircuit size={12} /> {projectName(conn.project_id)}</span>
                              <span><Clock size={12} /> last used {fmtTime(conn.last_used)}</span>
                              <span classList={{ 'text-green': conn.connected, 'text-muted': !conn.connected }}>
                                {conn.connected ? 'connected' : 'idle'}
                              </span>
                            </div>
                          </div>
                          <button
                            class="conn-action"
                            classList={{ active: open() }}
                            title="Setup guide"
                            onClick={() => setGuideFor(open() ? '' : conn.id)}
                          >
                            <BookOpen size={16} />
                          </button>
                          <button class="conn-action danger" onClick={() => handleDelete(conn.id)} title="Revoke">
                            <Trash2 size={16} />
                          </button>
                        </div>
                        <Show when={open() && platform()}>
                          <div class="conn-guide-panel">
                            <p class="mcp-hint">
                              Setup for {platform()!.label}. The token is only shown once at creation — replace
                              <code> YOUR_CORTEX_TOKEN </code> with the token you saved, or revoke and create a new connection.
                            </p>
                            <IntegrationGuide platform={platform()!} endpoint={conn.endpoint} />
                          </div>
                        </Show>
                      </div>
                    );
                  }}
                </For>
              </div>
            </Show>
          </div>
        </div>

        {/* Exposed tools — reference aside */}
        <div class="mcp-aside">
          <div class="mcp-panel">
            <div class="mcp-panel-header">
              <h3>Exposed Tools</h3>
              <span class="count-pill">{availableTools.length}</span>
            </div>
            <div class="tools-list">
              <For each={availableTools}>
                {(tool) => {
                  const Icon = tool.icon;
                  return (
                    <div class="tool-row">
                      <div class={`tool-icon ${tool.colorClass}`}>
                        <Icon size={16} />
                      </div>
                      <div class="tool-body">
                        <div class="tool-row-head">
                          <span class="tool-name">{tool.name}</span>
                          <span class={`tool-tag ${tool.colorClass}`}>{tool.category}</span>
                        </div>
                        <p class="tool-desc">{tool.desc}</p>
                      </div>
                    </div>
                  );
                }}
              </For>
            </div>
          </div>
        </div>
      </div>

      {/* New Connection modal (form → success) */}
      <Show when={showForm()}>
        <div class="mcp-modal-overlay" onClick={closeForm}>
          <div class="mcp-modal" onClick={(e) => e.stopPropagation()}>
            <div class="mcp-modal-header">
              <h3>{created() ? 'Connection created' : 'New Connection'}</h3>
              <button class="mcp-modal-close" onClick={closeForm} aria-label="Close">
                <X size={18} />
              </button>
            </div>

            <div class="mcp-modal-body">
              {/* ── Form state ── */}
              <Show when={!created()}>
                <div class="form-step">
                  <div class="form-step-title">1 · Select a project</div>
                  <select class="fancy-input" value={formProject()} onChange={(e) => setFormProject(e.currentTarget.value)}>
                    <option value="" disabled>Choose a project…</option>
                    <For each={projects()}>{(p) => <option value={p.id}>{p.name}</option>}</For>
                  </select>
                </div>

                <div class="form-step">
                  <div class="form-step-title">2 · Choose your AI client</div>
                  <For each={GROUP_ORDER}>
                    {(group) => (
                      <div class="platform-group">
                        <div class="platform-group-title">{GROUP_LABELS[group]}</div>
                        <div class="platform-grid">
                          <For each={platformsByGroup(group)}>
                            {(p) => (
                              <button
                                type="button"
                                class="platform-card"
                                classList={{ active: formPlatform() === p.id }}
                                onClick={() => setFormPlatform(p.id)}
                              >
                                <Show when={formPlatform() === p.id}>
                                  <span class="active-badge"><Check size={11} strokeWidth={3} /></span>
                                </Show>
                                <Show when={p.icon} fallback={<Terminal size={24} class="platform-fallback-icon" />}>
                                  <img src={p.icon} class="platform-icon" alt={p.label} />
                                </Show>
                                <span class="platform-title">{p.label}</span>
                              </button>
                            )}
                          </For>
                        </div>
                      </div>
                    )}
                  </For>
                </div>

                <div class="form-step">
                  <div class="form-step-title">3 · Label <span class="optional-text">(optional)</span></div>
                  <input
                    class="fancy-input"
                    value={formLabel()}
                    onInput={(e) => setFormLabel(e.currentTarget.value)}
                    placeholder="My Cursor on work laptop"
                  />
                </div>

                <Show when={error()}>
                  <div class="mcp-error">{error()}</div>
                </Show>

                <div class="mcp-modal-actions">
                  <button class="btn plain" onClick={closeForm}>Cancel</button>
                  <button class="btn primary-pink" onClick={handleCreate} disabled={creating()}>
                    {creating() ? 'Creating…' : 'Create & get config'}
                  </button>
                </div>
              </Show>

              {/* ── Success state ── */}
              <Show when={created()}>
                {(c) => {
                  const platform = getPlatform(c().ide) ?? PLATFORMS[0];
                  return (
                    <>
                      <div class="created-banner">
                        <Check size={16} />
                        <span>Copy your token now — it is shown only once.</span>
                      </div>

                      <div class="conn-token-row">
                        <span class="conn-token-label">Token</span>
                        <div class="conn-token-container">
                          <code class="conn-token">{c().token}</code>
                          <button class="copy-token-btn" onClick={() => copyTok(c().token || '')}>
                            <Show when={!copiedTok()} fallback={<Check size={14} class="text-green" />}>
                              <Copy size={14} />
                            </Show>
                          </button>
                        </div>
                      </div>

                      <div class="form-step-title">Set up {platform.label}</div>
                      <IntegrationGuide platform={platform} endpoint={c().endpoint} token={c().token} />

                      <div class="mcp-modal-actions">
                        <button class="btn primary-pink" onClick={closeForm}>Done</button>
                      </div>
                    </>
                  );
                }}
              </Show>
            </div>
          </div>
        </div>
      </Show>
    </div>
  );
};

// Small inline plug glyph for the empty state (keeps lucide import list lean).
const Plug: Component = () => (
  <svg width="34" height="34" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
    <path d="M12 22v-5" />
    <path d="M9 8V2" />
    <path d="M15 8V2" />
    <path d="M18 8v5a4 4 0 0 1-4 4h-4a4 4 0 0 1-4-4V8Z" />
  </svg>
);
