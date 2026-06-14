import { Component, For, createResource, createSignal, Show } from 'solid-js';
import {
  Server,
  Activity,
  Box,
  Plug,
  Plus,
  Trash2,
  RefreshCcw,
  Clock,
  Copy,
  Check,
  AlertTriangle,
  BrainCircuit,
  BookOpen,
} from 'lucide-solid';
import {
  getDaemonStatus,
  listProjects,
  listMCPConnections,
  createMCPConnection,
  deleteMCPConnection,
  type MCPConnection,
} from '../../api/client';
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
  { name: 'cortex_get_context', desc: 'Load project characterization + prior AI memory', category: 'Memory' },
  { name: 'cortex_save_memory', desc: 'Persist progress/decisions/notes for next session', category: 'Memory' },
  { name: 'cortex_list_memories', desc: 'List stored memories across IDEs/sessions', category: 'Memory' },
  { name: 'cortex_get_tasks', desc: 'List active tasks for the project', category: 'Project' },
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
        <button class="btn secondary small copy-btn" onClick={copy}>
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
  const [status, { refetch: refetchStatus }] = createResource(() => getDaemonStatus());
  const [projects] = createResource(() => listProjects());
  const [connections, { refetch: refetchConns }] = createResource(() => listMCPConnections());

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
  const endpoint = () => connections()?.[0]?.endpoint || created()?.endpoint || 'http://127.0.0.1:8090/mcp';

  const handleCreate = async () => {
    if (!formProject()) {
      setError('Pick a project to bind this connection to.');
      return;
    }
    setCreating(true);
    setError('');
    try {
      const conn = await createMCPConnection({
        project_id: formProject(),
        ide: formPlatform(),
        label: formLabel() || `${platformLabel(formPlatform())} · ${projectName(formProject())}`,
      });
      setCreated(conn);
      setShowForm(false);
      refetchConns();
    } catch (err: any) {
      setError(err?.message || 'Failed to create connection');
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    await deleteMCPConnection(id);
    if (guideFor() === id) setGuideFor('');
    refetchConns();
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
          <div class="page-title-icon green">
            <Server size={24} />
          </div>
          <div>
            <h1 class="page-title">MCP Server</h1>
            <p class="page-subtitle">Connect any AI client to CORTEX's shared project brain</p>
          </div>
        </div>
        <div class="page-actions">
          <button class="btn secondary" onClick={() => { refetchStatus(); refetchConns(); }}>
            <RefreshCcw size={16} />
            Refresh
          </button>
          <button class="btn primary" onClick={() => { setShowForm(true); setCreated(null); }}>
            <Plus size={16} />
            New Connection
          </button>
        </div>
      </div>

      <div class="mcp-layout">
        {/* Left - Status + tools */}
        <div class="mcp-left-col">
          <div class="mcp-card server-status-card">
            <div class="mcp-card-header">
              <h3>Server Status</h3>
              <Show when={status()?.ready} fallback={<span class="status-badge error">Offline</span>}>
                <span class="status-badge success">Online</span>
              </Show>
            </div>
            <div class="server-metrics">
              <div class="metric-row">
                <div class="metric-label"><Activity size={16} /> Endpoint</div>
                <div class="metric-value mono">/mcp</div>
              </div>
              <div class="metric-row">
                <div class="metric-label"><Box size={16} /> Version</div>
                <div class="metric-value mono">{status()?.version || '0.1.0'}</div>
              </div>
              <div class="metric-row">
                <div class="metric-label"><Plug size={16} /> Connections</div>
                <div class="metric-value">{connections()?.length ?? 0}</div>
              </div>
              <div class="metric-row">
                <div class="metric-label"><BrainCircuit size={16} /> Semantic</div>
                <div class="metric-value">
                  <Show when={status()?.semanticEnabled} fallback={<span class="text-orange">Disabled</span>}>
                    <span class="text-green">Enabled</span>
                  </Show>
                </div>
              </div>
            </div>
          </div>

          <div class="mcp-card tools-card">
            <div class="mcp-card-header">
              <h3>Exposed Tools</h3>
              <span class="badge">{availableTools.length}</span>
            </div>
            <div class="tools-grid">
              <For each={availableTools}>
                {(tool) => (
                  <div class="tool-item">
                    <div class="tool-item-header">
                      <span class="tool-name mono">{tool.name}</span>
                      <span class="tool-category">{tool.category}</span>
                    </div>
                    <p class="tool-desc">{tool.desc}</p>
                  </div>
                )}
              </For>
            </div>
          </div>

          <div class="mcp-card">
            <div class="mcp-card-header"><h3>Supported Clients</h3><span class="badge">{PLATFORMS.length}</span></div>
            <For each={GROUP_ORDER}>
              {(g) => (
                <div class="supported-group">
                  <div class="supported-group-title">{GROUP_LABELS[g]}</div>
                  <div class="supported-chips">
                    <For each={platformsByGroup(g)}>
                      {(p) => <span class="supported-chip">{p.label}</span>}
                    </For>
                  </div>
                </div>
              )}
            </For>
          </div>
        </div>

        {/* Right - Connections */}
        <div class="mcp-right-col">
          <Show when={error()}>
            <div class="mcp-error">{error()}</div>
          </Show>

          {/* Create form */}
          <Show when={showForm()}>
            <div class="mcp-card">
              <div class="mcp-card-header"><h3>New Connection</h3></div>
              <div class="conn-form">
                <label class="conn-field">
                  <span>Project</span>
                  <select value={formProject()} onChange={(e) => setFormProject(e.currentTarget.value)}>
                    <option value="" disabled>Select a project…</option>
                    <For each={projects()}>{(p) => <option value={p.id}>{p.name}</option>}</For>
                  </select>
                </label>
                <label class="conn-field">
                  <span>AI Client / IDE</span>
                  <select value={formPlatform()} onChange={(e) => setFormPlatform(e.currentTarget.value)}>
                    <For each={GROUP_ORDER}>
                      {(g) => (
                        <optgroup label={GROUP_LABELS[g]}>
                          <For each={platformsByGroup(g)}>
                            {(p) => <option value={p.id}>{p.label}</option>}
                          </For>
                        </optgroup>
                      )}
                    </For>
                  </select>
                </label>
                <label class="conn-field">
                  <span>Label (optional)</span>
                  <input value={formLabel()} onInput={(e) => setFormLabel(e.currentTarget.value)} placeholder="My Cursor on work laptop" />
                </label>
                <div class="conn-form-actions">
                  <button class="btn secondary" onClick={() => setShowForm(false)}>Cancel</button>
                  <button class="btn primary" onClick={handleCreate} disabled={creating()}>
                    {creating() ? 'Creating…' : 'Create & Get Config'}
                  </button>
                </div>
              </div>
            </div>
          </Show>

          {/* Newly created connection (token shown once) */}
          <Show when={created()}>
            {(c) => {
              const platform = getPlatform(c().ide) ?? PLATFORMS[0];
              return (
                <div class="mcp-card created-card">
                  <div class="mcp-card-header">
                    <h3>{platform.label} — copy this now</h3>
                    <Check size={18} class="text-green" />
                  </div>
                  <p class="mcp-subtitle">
                    The token below is shown only once. Follow the steps for {platform.label}.
                  </p>

                  <div class="conn-token-row">
                    <span class="conn-token-label">Token</span>
                    <code class="conn-token">{c().token}</code>
                    <button class="icon-btn" onClick={() => copyTok(c().token || '')}>
                      <Show when={!copiedTok()} fallback={<Check size={14} class="text-green" />}>
                        <Copy size={14} />
                      </Show>
                    </button>
                  </div>

                  <IntegrationGuide platform={platform} endpoint={c().endpoint} token={c().token} />
                </div>
              );
            }}
          </Show>

          {/* Connection list */}
          <div class="mcp-card">
            <div class="mcp-card-header">
              <h3>Connections</h3>
              <span class="badge">{connections()?.length ?? 0}</span>
            </div>

            <Show when={(connections()?.length ?? 0) > 0} fallback={
              <div class="conn-empty">No connections yet. Create one to link Claude Code, Cursor, VS Code, Gemini CLI and others to this project's memory.</div>
            }>
              <div class="conn-list">
                <For each={connections()}>
                  {(conn) => {
                    const platform = () => getPlatform(conn.ide);
                    return (
                      <div class="conn-item-wrap">
                        <div class="conn-item">
                          <div class="conn-status-dot" classList={{ connected: conn.connected }} title={conn.connected ? 'Active' : 'Idle'} />
                          <div class="conn-info">
                            <div class="conn-title">
                              {conn.label || platformLabel(conn.ide)}
                              <span class="conn-ide-tag">{platformLabel(conn.ide)}</span>
                              <Show when={conn.client_name}>
                                <span class="conn-client-tag">{conn.client_name}</span>
                              </Show>
                            </div>
                            <div class="conn-meta">
                              <span><BrainCircuit size={12} /> {projectName(conn.project_id)}</span>
                              <span><Clock size={12} /> last used {fmtTime(conn.last_used)}</span>
                              <span classList={{ 'text-green': conn.connected, 'text-orange': !conn.connected }}>
                                {conn.connected ? 'connected' : 'idle'}
                              </span>
                            </div>
                          </div>
                          <button
                            class="icon-btn"
                            title="Setup guide"
                            onClick={() => setGuideFor(guideFor() === conn.id ? '' : conn.id)}
                          >
                            <BookOpen size={16} />
                          </button>
                          <button class="icon-btn danger" onClick={() => handleDelete(conn.id)} title="Revoke">
                            <Trash2 size={16} />
                          </button>
                        </div>
                        <Show when={guideFor() === conn.id && platform()}>
                          <div class="conn-guide-panel">
                            <p class="mcp-subtitle">
                              Setup for {platform()!.label}. The token is only shown at creation — replace
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

          {/* Endpoint note */}
          <div class="mcp-card">
            <div class="mcp-card-header"><h3>Endpoint</h3></div>
            <p class="mcp-subtitle">
              CORTEX speaks MCP over Streamable HTTP. Local IDEs/CLIs connect directly; web clients (Claude.ai, ChatGPT) need a public HTTPS tunnel.
            </p>
            <div class="integration-code">
              <code>{endpoint()}</code>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
