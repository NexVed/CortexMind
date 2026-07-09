import { Component, For, createSignal, createEffect, Show } from 'solid-js';
import { Settings, Palette, FolderGit2, BrainCircuit, Server, GitBranch, Keyboard, Info, Trash2, Plus, Plug } from 'lucide-solid';
import { type ProviderConfig, type MCPConnection } from '../../api/client';
import {
  useProviderConfig,
  useSetProviderConfig,
  useMCPConnections,
  useCreateMCPConnection,
  useDeleteMCPConnection,
  useResetAllData,
} from '../../api/queries';
import { settings, isEnabled, setToggle, setTheme, setSidebarPosition, resetSettings, modKey, type Theme, type SidebarPosition } from '../../api/settings';
import { useAuth } from '../../api/auth';
import '../shared.css';

const settingsCategories = [
  { id: 'general', label: 'General', icon: Settings },
  { id: 'appearance', label: 'Appearance', icon: Palette },
  { id: 'repos', label: 'Repository Defaults', icon: FolderGit2 },
  { id: 'agents', label: 'AI Agents', icon: BrainCircuit },
  { id: 'mcp', label: 'MCP Server', icon: Server },
  { id: 'sync', label: 'Sync & Git', icon: GitBranch },
  { id: 'shortcuts', label: 'Keyboard Shortcuts', icon: Keyboard },
  { id: 'about', label: 'About', icon: Info },
];

const generalSettings = [
  { label: 'Auto-scan repositories', description: 'Automatically scan repositories when added to CortexMind', enabled: true },
  { label: 'Live file watching', description: 'Watch for file changes and update indexes in real-time', enabled: true },
  { label: 'Send anonymous usage data', description: 'Help improve CortexMind by sharing anonymous usage statistics', enabled: false },
  { label: 'Auto-update', description: 'Automatically download and install updates', enabled: true },
];

const appearanceSettings = [
  { label: 'Theme', description: 'Choose your preferred appearance', type: 'theme', options: ['Light', 'Dark', 'System'] },
  { label: 'Sidebar position', description: 'Place the sidebar on the left or right', type: 'sidebar', options: ['Left', 'Right'] },
  { label: 'Compact mode', description: 'Reduce spacing for more information density', type: 'toggle' },
  { label: 'Show file previews', description: 'Display file content previews in search results', type: 'toggle' },
];

const repoDefaultSettings = [
  { label: 'Auto-scan on import', description: 'Automatically run a deep scan when a new repository is imported via GitHub' },
  { label: 'Deep analysis', description: 'Use LLM to generate richer summaries during scan (slower but more detailed)' },
  { label: 'Include hidden files', description: 'Index dot-files and dot-directories (e.g. .github, .vscode) during scan' },
  { label: 'Extract functions & classes', description: 'Parse source files to extract function signatures and class definitions' },
];

const syncGitSettings = [
  { label: 'Auto-commit memory bundle', description: 'Automatically commit .cortex/memory.json to the project repo after export' },
  { label: 'Push on export', description: 'Push the committed memory bundle to the remote origin after export' },
  { label: 'Include vault entries', description: 'Include knowledge vault entries in the exported memory bundle' },
  { label: 'Include agent memories', description: 'Include raw agent working-memory entries in the exported bundle' },
];

const shortcuts = [
  { action: 'Global search', keys: `${modKey} K` },
  { action: 'New project', keys: `${modKey} N` },
  { action: 'New handoff', keys: `${modKey} Shift H` },
  { action: 'Generate context', keys: `${modKey} Shift G` },
  { action: 'Toggle sidebar', keys: `${modKey} B` },
  { action: 'Toggle theme', keys: `${modKey} Shift T` },
  { action: 'Navigate to Dashboard', keys: `${modKey} 1` },
  { action: 'Navigate to Projects', keys: `${modKey} 2` },
];

export const SettingsPage: Component = () => {
  const { logout } = useAuth();
  const [activeCategory, setActiveCategory] = createSignal('general');

  // Preferences are backed by the shared, persisted settings store.
  const toggle = (label: string) => setToggle(label);

  const resetM = useResetAllData();

  // ── Reset all data ───────────────────────────────────
  const [resetting, setResetting] = createSignal(false);
  const handleReset = async () => {
    const confirmed = window.confirm(
      'This permanently deletes ALL CortexMind data — projects, memories, handoffs, ' +
      'tasks, digests, and settings. This cannot be undone.\n\nContinue?'
    );
    if (!confirmed) return;
    setResetting(true);
    try {
      await resetM.mutateAsync();
      resetSettings();
      logout();
      window.location.href = '/';
    } catch (err: any) {
      alert(err?.message || 'Failed to reset data');
      setResetting(false);
    }
  };

  // ── AI Agents / provider configuration ───────────────
  const providersQuery = useProviderConfig();
  const providers = () => providersQuery.data;
  const setProviderM = useSetProviderConfig();
  const [form, setForm] = createSignal<ProviderConfig | null>(null);
  const [saving, setSaving] = createSignal(false);
  const [saveMsg, setSaveMsg] = createSignal('');

  createEffect(() => {
    const p = providers();
    if (p && !form()) setForm({ ...p });
  });

  const updateField = (key: keyof ProviderConfig, value: string) => {
    setForm(prev => (prev ? { ...prev, [key]: value } : prev));
  };

  const saveProviders = async () => {
    const f = form();
    if (!f) return;
    setSaving(true);
    setSaveMsg('');
    try {
      // Skip the masked key so the stored secret isn't overwritten with dots.
      const payload: Partial<ProviderConfig> = { ...f };
      if (f.mistral_key && f.mistral_key.includes('•')) delete payload.mistral_key;
      const updated = await setProviderM.mutateAsync(payload);
      setForm({ ...updated });
      setSaveMsg('Saved');
    } catch (err: any) {
      setSaveMsg(err?.message || 'Failed to save');
    } finally {
      setSaving(false);
      setTimeout(() => setSaveMsg(''), 3000);
    }
  };

  // ── MCP connections ──────────────────────────────────
  const mcpQuery = useMCPConnections();
  const mcpConnections = () => mcpQuery.data;
  const createConnM = useCreateMCPConnection();
  const deleteConnM = useDeleteMCPConnection();
  const [creatingConn, setCreatingConn] = createSignal(false);

  const handleCreateConnection = async () => {
    setCreatingConn(true);
    try {
      await createConnM.mutateAsync({ ide: 'generic', label: 'New Connection' });
    } catch { /* ignore */ }
    setCreatingConn(false);
  };

  const handleDeleteConnection = async (id: string) => {
    try {
      await deleteConnM.mutateAsync(id);
    } catch { /* ignore */ }
  };

  const inputStyle = {
    height: '34px', padding: '0 10px', width: '260px',
    background: 'var(--bg-base)', border: '1px solid var(--border)',
    'border-radius': 'var(--r-md)', 'font-size': '13px', color: 'var(--text-primary)',
    'font-family': 'var(--font-display)',
  } as const;

  return (
    <div class="page">
      <div class="page-header">
        <div class="page-title-row" style={{ display: 'flex', 'align-items': 'center', gap: 'var(--s4)' }}>
          <div class="page-title-icon pink">
            <Settings size={24} />
          </div>
          <div>
            <h1 class="page-title" style={{ margin: 0 }}>Settings</h1>
            <p class="page-subtitle" style={{ 'font-size': '13px', color: 'var(--text-secondary)', 'margin-top': '2px' }}>
              Configure application settings, API keys, and model preferences
            </p>
          </div>
        </div>
      </div>

      <div class="split-layout">
        <div class="split-rail">
          <For each={settingsCategories}>
            {(cat) => {
              const Icon = cat.icon;
              return (
                <button
                  class={`split-rail-item ${activeCategory() === cat.id ? 'active' : ''}`}
                  onClick={() => setActiveCategory(cat.id)}
                >
                  <Icon size={16} />
                  {cat.label}
                </button>
              );
            }}
          </For>
        </div>

        <div class="split-content">
          {activeCategory() === 'general' && (
            <div class="card">
              <div class="card-title" style={{ 'margin-bottom': '8px' }}>General</div>
              <For each={generalSettings}>
                {(setting) => (
                  <div class="setting-row">
                    <div class="setting-info">
                      <span class="setting-label">{setting.label}</span>
                      <span class="setting-description">{setting.description}</span>
                    </div>
                    <div
                      class={`toggle-switch ${isEnabled(setting.label) ? 'active' : ''}`}
                      onClick={() => toggle(setting.label)}
                    />
                  </div>
                )}
              </For>
            </div>
          )}

          {activeCategory() === 'appearance' && (
            <div class="card">
              <div class="card-title" style={{ 'margin-bottom': '8px' }}>Appearance</div>
              <For each={appearanceSettings}>
                {(setting) => (
                  <div class="setting-row">
                    <div class="setting-info">
                      <span class="setting-label">{setting.label}</span>
                      <span class="setting-description">{setting.description}</span>
                    </div>
                    {setting.type === 'toggle' ? (
                      <div
                        class={`toggle-switch ${isEnabled(setting.label) ? 'active' : ''}`}
                        onClick={() => toggle(setting.label)}
                      />
                    ) : (
                      <select
                        style={{
                          height: '32px', padding: '0 10px',
                          background: 'var(--bg-base)', border: '1px solid var(--border)',
                          'border-radius': 'var(--r-md)', 'font-size': '12px', color: 'var(--text-primary)',
                          'font-family': 'var(--font-display)',
                        }}
                        value={setting.type === 'theme' ? settings.theme : settings.sidebarPosition}
                        onChange={(e) => {
                          if (setting.type === 'theme') setTheme(e.currentTarget.value as Theme);
                          else setSidebarPosition(e.currentTarget.value as SidebarPosition);
                        }}
                      >
                        <For each={setting.options!}>{(opt) => <option value={opt}>{opt}</option>}</For>
                      </select>
                    )}
                  </div>
                )}
              </For>
            </div>
          )}

          {activeCategory() === 'repos' && (
            <div class="card">
              <div class="card-title" style={{ 'margin-bottom': '4px' }}>Repository Defaults</div>
              <div class="setting-description" style={{ 'margin-bottom': '16px' }}>
                Configure default behavior when importing and scanning repositories.
              </div>
              <For each={repoDefaultSettings}>
                {(setting) => (
                  <div class="setting-row">
                    <div class="setting-info">
                      <span class="setting-label">{setting.label}</span>
                      <span class="setting-description">{setting.description}</span>
                    </div>
                    <div
                      class={`toggle-switch ${isEnabled(setting.label) ? 'active' : ''}`}
                      onClick={() => toggle(setting.label)}
                    />
                  </div>
                )}
              </For>
            </div>
          )}

          {activeCategory() === 'sync' && (
            <div class="card">
              <div class="card-title" style={{ 'margin-bottom': '4px' }}>Sync & Git</div>
              <div class="setting-description" style={{ 'margin-bottom': '16px' }}>
                Control how CortexMind memory bundles are committed and pushed to your repositories.
              </div>
              <For each={syncGitSettings}>
                {(setting) => (
                  <div class="setting-row">
                    <div class="setting-info">
                      <span class="setting-label">{setting.label}</span>
                      <span class="setting-description">{setting.description}</span>
                    </div>
                    <div
                      class={`toggle-switch ${isEnabled(setting.label) ? 'active' : ''}`}
                      onClick={() => toggle(setting.label)}
                    />
                  </div>
                )}
              </For>
            </div>
          )}

          {activeCategory() === 'mcp' && (
            <div class="card">
              <div class="card-title" style={{ 'margin-bottom': '4px' }}>MCP Server Connections</div>
              <div class="setting-description" style={{ 'margin-bottom': '16px' }}>
                Manage IDE connections to the CORTEX MCP server. Each connection binds an IDE to your account.
              </div>

              <Show when={mcpConnections()} fallback={<div class="small">Loading connections…</div>}>
                <Show when={mcpConnections()!.length > 0} fallback={
                  <div class="setting-row" style={{ 'border-bottom': 'none', 'justify-content': 'center' }}>
                    <div class="setting-info" style={{ 'text-align': 'center' }}>
                      <Plug size={32} style={{ color: 'var(--text-muted)', opacity: '0.4', 'margin-bottom': '8px' }} />
                      <span class="setting-label">No connections yet</span>
                      <span class="setting-description">Create a connection to let your IDE communicate with CORTEX.</span>
                    </div>
                  </div>
                }>
                  <For each={mcpConnections()!}>
                    {(conn: MCPConnection) => (
                      <div class="setting-row">
                        <div class="setting-info">
                          <span class="setting-label">{conn.label || conn.ide || 'Connection'}</span>
                          <span class="setting-description">
                            IDE: {conn.ide} · Endpoint: {conn.endpoint || '—'} · Last used: {conn.last_used ? new Date(conn.last_used).toLocaleDateString() : 'never'}
                          </span>
                        </div>
                        <div style={{ display: 'flex', 'align-items': 'center', gap: '8px' }}>
                          <span style={{
                            'font-size': '11px',
                            padding: '2px 8px',
                            'border-radius': 'var(--r-sm)',
                            background: conn.connected ? '#E8FBF5' : 'var(--bg-elevated)',
                            color: conn.connected ? 'var(--green)' : 'var(--text-muted)',
                            'font-weight': '500',
                          }}>
                            {conn.connected ? 'Connected' : 'Offline'}
                          </span>
                          <button
                            style={{
                              background: 'transparent', border: 'none', cursor: 'pointer',
                              color: 'var(--text-muted)', padding: '4px',
                            }}
                            title="Delete connection"
                            onClick={() => handleDeleteConnection(conn.id)}
                          >
                            <Trash2 size={14} />
                          </button>
                        </div>
                      </div>
                    )}
                  </For>
                </Show>
              </Show>

              <div class="setting-row" style={{ 'border-bottom': 'none', 'margin-top': '8px' }}>
                <span />
                <button
                  class="btn primary"
                  style={{ height: '34px', padding: '0 18px', cursor: 'pointer', display: 'flex', 'align-items': 'center', gap: '6px' }}
                  disabled={creatingConn()}
                  onClick={handleCreateConnection}
                >
                  <Plus size={14} />
                  {creatingConn() ? 'Creating…' : 'New Connection'}
                </button>
              </div>
            </div>
          )}

          {activeCategory() === 'shortcuts' && (
            <div class="card">
              <div class="card-title" style={{ 'margin-bottom': '8px' }}>Keyboard Shortcuts</div>
              <For each={shortcuts}>
                {(shortcut) => (
                  <div class="setting-row">
                    <span class="setting-label">{shortcut.action}</span>
                    <span style={{
                      padding: '2px 8px', background: 'var(--bg-elevated)',
                      border: '1px solid var(--border)', 'border-radius': 'var(--r-sm)',
                      'font-size': '12px', 'font-weight': '500', color: 'var(--text-secondary)',
                      'font-family': 'var(--font-mono)',
                    }}>{shortcut.keys}</span>
                  </div>
                )}
              </For>
            </div>
          )}

          {activeCategory() === 'about' && (
            <div class="card">
              <div style={{ 'text-align': 'center', padding: '32px 0' }}>
                <div style={{
                  width: '64px', height: '64px', 'border-radius': 'var(--r-lg)',
                  background: 'linear-gradient(135deg, var(--accent), #8B5CF6)',
                  display: 'flex', 'align-items': 'center', 'justify-content': 'center',
                  color: '#fff', 'font-weight': '700', 'font-size': '24px',
                  margin: '0 auto 16px',
                }}>C</div>
                <div style={{ 'font-size': '18px', 'font-weight': '700', color: 'var(--text-primary)', 'margin-bottom': '4px' }}>CortexMind</div>
                <div style={{ 'font-size': '13px', color: 'var(--text-secondary)', 'margin-bottom': '4px' }}>The Shared Brain For AI Development</div>
                <div class="small">Version 0.1.0-alpha</div>
                <div class="small" style={{ 'margin-top': '16px' }}>Built with ❤️ by NexVed</div>
              </div>
            </div>
          )}

          {activeCategory() === 'agents' && (
            <div class="card">
              <div class="card-title" style={{ 'margin-bottom': '4px' }}>AI Agents & Memory Providers</div>
              <div class="setting-description" style={{ 'margin-bottom': '16px' }}>
                Configure the LLM used to enrich repository analysis and the embedding provider used to build semantic memory during a scan.
              </div>

              <Show when={form()} fallback={<div class="small">Loading provider settings…</div>}>
                {/* LLM enrichment */}
                <div class="setting-row">
                  <div class="setting-info">
                    <span class="setting-label">Analysis LLM</span>
                    <span class="setting-description">Generates richer feature/architecture summaries</span>
                  </div>
                  <select
                    style={inputStyle}
                    value={form()!.llm_provider}
                    onChange={(e) => updateField('llm_provider', e.currentTarget.value)}
                  >
                    <option value="none">None (heuristic only)</option>
                    <option value="mistral">Mistral</option>
                    <option value="ollama">Ollama (local)</option>
                  </select>
                </div>

                <Show when={form()!.llm_provider === 'ollama'}>
                  <div class="setting-row">
                    <div class="setting-info">
                      <span class="setting-label">Ollama Chat Model</span>
                      <span class="setting-description">Model used to generate prompts (e.g. llama3.1, qwen2.5)</span>
                    </div>
                    <input
                      style={inputStyle}
                      value={form()!.ollama_chat_model}
                      onInput={(e) => updateField('ollama_chat_model', e.currentTarget.value)}
                    />
                  </div>
                  <div class="setting-row">
                    <div class="setting-info">
                      <span class="setting-label">Ollama URL</span>
                      <span class="setting-description">Local Ollama server endpoint</span>
                    </div>
                    <input
                      style={inputStyle}
                      value={form()!.ollama_url}
                      onInput={(e) => updateField('ollama_url', e.currentTarget.value)}
                    />
                  </div>
                </Show>

                <Show when={form()!.llm_provider === 'mistral'}>
                  <div class="setting-row">
                    <div class="setting-info">
                      <span class="setting-label">Mistral API Key</span>
                      <span class="setting-description">Stored securely on your user record</span>
                    </div>
                    <input
                      type="password"
                      placeholder="sk-…"
                      style={inputStyle}
                      value={form()!.mistral_key}
                      onInput={(e) => updateField('mistral_key', e.currentTarget.value)}
                    />
                  </div>
                  <div class="setting-row">
                    <div class="setting-info">
                      <span class="setting-label">Mistral Model</span>
                      <span class="setting-description">Chat model for summaries</span>
                    </div>
                    <input
                      style={inputStyle}
                      value={form()!.mistral_model}
                      onInput={(e) => updateField('mistral_model', e.currentTarget.value)}
                    />
                  </div>
                </Show>

                {/* Embeddings */}
                <div class="setting-row">
                  <div class="setting-info">
                    <span class="setting-label">Memory Embeddings</span>
                    <span class="setting-description">Provider used to vectorize project memory</span>
                  </div>
                  <select
                    style={inputStyle}
                    value={form()!.embedder}
                    onChange={(e) => updateField('embedder', e.currentTarget.value)}
                  >
                    <option value="none">Disabled</option>
                    <option value="ollama">Ollama (local)</option>
                    <option value="mistral">Mistral</option>
                  </select>
                </div>

                <Show when={form()!.embedder === 'ollama'}>
                  <div class="setting-row">
                    <div class="setting-info">
                      <span class="setting-label">Ollama URL</span>
                      <span class="setting-description">Local Ollama server endpoint</span>
                    </div>
                    <input
                      style={inputStyle}
                      value={form()!.ollama_url}
                      onInput={(e) => updateField('ollama_url', e.currentTarget.value)}
                    />
                  </div>
                  <div class="setting-row">
                    <div class="setting-info">
                      <span class="setting-label">Ollama Embedding Model</span>
                      <span class="setting-description">e.g. bge-m3, nomic-embed-text</span>
                    </div>
                    <input
                      style={inputStyle}
                      value={form()!.ollama_model}
                      onInput={(e) => updateField('ollama_model', e.currentTarget.value)}
                    />
                  </div>
                </Show>

                <Show when={form()!.embedder === 'mistral'}>
                  <div class="setting-row">
                    <div class="setting-info">
                      <span class="setting-label">Mistral Embedding Model</span>
                      <span class="setting-description">e.g. mistral-embed (uses the key above)</span>
                    </div>
                    <input
                      style={inputStyle}
                      value={form()!.mistral_emb_model}
                      onInput={(e) => updateField('mistral_emb_model', e.currentTarget.value)}
                    />
                  </div>
                </Show>

                <div class="setting-row" style={{ 'border-bottom': 'none', 'margin-top': '8px' }}>
                  <span class="small" style={{ color: saveMsg() === 'Saved' ? 'var(--green, #22C55E)' : 'var(--text-muted)' }}>
                    {saveMsg()}
                  </span>
                  <button
                    class="btn primary"
                    style={{ height: '34px', padding: '0 18px', cursor: 'pointer' }}
                    disabled={saving()}
                    onClick={saveProviders}
                  >
                    {saving() ? 'Saving…' : 'Save Providers'}
                  </button>
                </div>
              </Show>
            </div>
          )}

          {activeCategory() === 'general' && (
            <div class="card" style={{ 'margin-top': '16px', 'border-color': '#FECACA' }}>
              <div class="card-title" style={{ color: 'var(--red)', 'margin-bottom': '8px' }}>Danger Zone</div>
              <div class="setting-row">
                <div class="setting-info">
                  <span class="setting-label">Reset all data</span>
                  <span class="setting-description">Delete all CORTEX data including memories, handoffs, and settings.</span>
                </div>
                <button
                  onClick={handleReset}
                  disabled={resetting()}
                  style={{
                    height: '32px', padding: '0 14px',
                    background: '#FEF2F2', color: '#DC2626',
                    border: '1px solid #FECACA', 'border-radius': 'var(--r-md)',
                    'font-size': '13px', 'font-weight': '500',
                    cursor: resetting() ? 'not-allowed' : 'pointer',
                    opacity: resetting() ? '0.6' : '1',
                  }}>{resetting() ? 'Resetting…' : 'Reset'}</button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
