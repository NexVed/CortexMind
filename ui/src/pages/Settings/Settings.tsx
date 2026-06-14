import { Component, For, createSignal, createResource, createEffect, Show } from 'solid-js';
import { Settings, Palette, FolderGit2, BrainCircuit, Server, GitBranch, Keyboard, Info } from 'lucide-solid';
import { getProviderConfig, setProviderConfig, type ProviderConfig } from '../../api/client';
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
  { label: 'Auto-scan repositories', description: 'Automatically scan repositories when added to CORTEX', enabled: true },
  { label: 'Live file watching', description: 'Watch for file changes and update indexes in real-time', enabled: true },
  { label: 'Send anonymous usage data', description: 'Help improve CORTEX by sharing anonymous usage statistics', enabled: false },
  { label: 'Auto-update', description: 'Automatically download and install updates', enabled: true },
];

const appearanceSettings = [
  { label: 'Theme', description: 'Choose your preferred appearance', type: 'select', options: ['Light', 'Dark', 'System'], value: 'Light' },
  { label: 'Sidebar position', description: 'Place the sidebar on the left or right', type: 'select', options: ['Left', 'Right'], value: 'Left' },
  { label: 'Compact mode', description: 'Reduce spacing for more information density', type: 'toggle', enabled: false },
  { label: 'Show file previews', description: 'Display file content previews in search results', type: 'toggle', enabled: true },
];

const shortcuts = [
  { action: 'Global search', keys: '⌘ K' },
  { action: 'New project', keys: '⌘ N' },
  { action: 'New handoff', keys: '⌘ Shift H' },
  { action: 'Generate context', keys: '⌘ Shift G' },
  { action: 'Toggle sidebar', keys: '⌘ B' },
  { action: 'Toggle theme', keys: '⌘ Shift T' },
  { action: 'Navigate to Dashboard', keys: '⌘ 1' },
  { action: 'Navigate to Projects', keys: '⌘ 2' },
];

export const SettingsPage: Component = () => {
  const [activeCategory, setActiveCategory] = createSignal('general');
  const [toggleStates, setToggleStates] = createSignal<Record<string, boolean>>({
    'Auto-scan repositories': true,
    'Live file watching': true,
    'Send anonymous usage data': false,
    'Auto-update': true,
    'Compact mode': false,
    'Show file previews': true,
  });

  const toggle = (label: string) => {
    setToggleStates(prev => ({ ...prev, [label]: !prev[label] }));
  };

  // ── AI Agents / provider configuration ───────────────
  const [providers, { refetch: refetchProviders }] = createResource(getProviderConfig);
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
      const updated = await setProviderConfig(payload);
      setForm({ ...updated });
      setSaveMsg('Saved');
      refetchProviders();
    } catch (err: any) {
      setSaveMsg(err?.message || 'Failed to save');
    } finally {
      setSaving(false);
      setTimeout(() => setSaveMsg(''), 3000);
    }
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
        <h1>Settings</h1>
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
                      class={`toggle-switch ${toggleStates()[setting.label] ? 'active' : ''}`}
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
                        class={`toggle-switch ${toggleStates()[setting.label] ? 'active' : ''}`}
                        onClick={() => toggle(setting.label)}
                      />
                    ) : (
                      <select style={{
                        height: '32px', padding: '0 10px',
                        background: 'var(--bg-base)', border: '1px solid var(--border)',
                        'border-radius': 'var(--r-md)', 'font-size': '12px', color: 'var(--text-primary)',
                        'font-family': 'var(--font-display)',
                      }}>
                        <For each={setting.options!}>{(opt) => <option>{opt}</option>}</For>
                      </select>
                    )}
                  </div>
                )}
              </For>
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
                <div style={{ 'font-size': '18px', 'font-weight': '700', color: 'var(--text-primary)', 'margin-bottom': '4px' }}>CORTEX</div>
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

          {!['general', 'appearance', 'shortcuts', 'about', 'agents'].includes(activeCategory()) && (
            <div class="card">
              <div class="empty-state">
                <Settings size={48} class="empty-state-icon" />
                <div class="empty-state-title">{settingsCategories.find(c => c.id === activeCategory())?.label} Settings</div>
                <div class="empty-state-text">Configuration options for this section will be available soon.</div>
              </div>
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
                <button style={{
                  height: '32px', padding: '0 14px',
                  background: '#FEF2F2', color: '#DC2626',
                  border: '1px solid #FECACA', 'border-radius': 'var(--r-md)',
                  'font-size': '13px', 'font-weight': '500',
                  cursor: 'pointer',
                }}>Reset</button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
