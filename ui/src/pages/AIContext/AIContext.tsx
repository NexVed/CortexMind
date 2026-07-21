import { Component, Show, createEffect, createMemo, createSignal } from 'solid-js';
import { useSearchParams } from '@solidjs/router';
import {
  Archive,
  BrainCircuit,
  Check,
  Copy,
  FileCode2,
  ListTodo,
  Save,
  Sparkles,
} from 'lucide-solid';
import { generateSystemPrompt } from '../../api/client';
import { useAgentMemories, useProjects, useSaveSystemPrompt, useSystemPrompt } from '../../api/queries';
import { ProjectSelect } from '../../components/ProjectSelect/ProjectSelect';
import { createProjectSelection } from '../../api/projectSelection';
import { createPersistedSignal } from '../../api/persistedState';
import './AIContext.css';

const placeholderPrompt =
  'Write the instructions you want every AI coding agent to follow for this project. You can include architecture, conventions, research findings, constraints, and working agreements.';

export const AIContextPage: Component = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const [includeTasks, setIncludeTasks] = createPersistedSignal('ai-context.include-tasks', true);
  const [includeVault, setIncludeVault] = createPersistedSignal('ai-context.include-vault', true);
  const [includeActivity, setIncludeActivity] = createPersistedSignal('ai-context.include-activity', false);
  const [isGenerating, setIsGenerating] = createSignal(false);
  const [copied, setCopied] = createSignal(false);
  const [saved, setSaved] = createSignal(false);
  const [prompt, setPrompt] = createSignal('');
  const [provider, setProvider] = createSignal('');
  const [tokenEstimate, setTokenEstimate] = createSignal(0);
  const [error, setError] = createSignal('');
  let loadedProject = '';

  const projectsQuery = useProjects();
  const projects = () => projectsQuery.data;
  const projectSelection = createProjectSelection(
    () => typeof searchParams.project === 'string' ? searchParams.project : undefined,
    projects,
  );
  const selectedProject = projectSelection.selected;
  const setSelectedProject = projectSelection.select;
  const promptQuery = useSystemPrompt(selectedProject);
  const memoriesQuery = useAgentMemories(selectedProject);
  const savePrompt = useSaveSystemPrompt();
  const latestSessionContext = createMemo(() =>
    (memoriesQuery.data || []).find((memory) => memory.category === 'context')
  );

  // Only apply a response to the project it belongs to. This prevents the
  // previous project's prompt from appearing while the new project loads.
  createEffect(() => {
    const id = selectedProject();
    const data = promptQuery.data;
    if (!id) {
      loadedProject = '';
      setPrompt('');
      setProvider('');
      setTokenEstimate(0);
      setSaved(false);
      return;
    }
    if (data?.project_id === id) {
      loadedProject = id;
      setPrompt(data.prompt || '');
      setProvider(data.prompt ? data.provider : '');
      setTokenEstimate(data.token_estimate || 0);
      setSaved(false);
    } else if (loadedProject !== id) {
      setPrompt('');
      setProvider('');
      setTokenEstimate(0);
      setSaved(false);
    }
  });

  const handlePromptChange = (value: string) => {
    setPrompt(value);
    setSaved(false);
    setProvider(value ? 'custom' : '');
    setTokenEstimate(Math.round(value.length / 4));
  };

  const handleSave = async () => {
    if (!selectedProject()) {
      setError('Pick a project first.');
      return;
    }
    setError('');
    setSaved(false);
    try {
      const res = await savePrompt.mutateAsync({
        projectId: selectedProject(),
        prompt: prompt(),
      });
      setPrompt(res.prompt);
      setProvider(res.prompt ? 'custom' : '');
      setTokenEstimate(res.token_estimate);
      setSaved(true);
    } catch (err: any) {
      setError(err?.message || 'Failed to save prompt');
    }
  };

  const handleGenerate = async () => {
    if (!selectedProject()) {
      setError('Pick a project first.');
      return;
    }
    setIsGenerating(true);
    setError('');
    setSaved(false);
    try {
      const res = await generateSystemPrompt(selectedProject(), {
        include_tasks: includeTasks(),
        include_vault: includeVault(),
        include_activity: includeActivity(),
        preview: true,
      });
      setPrompt(res.prompt);
      setProvider(res.provider);
      setTokenEstimate(res.token_estimate);
    } catch (err: any) {
      setError(err?.message || 'Failed to generate prompt');
    } finally {
      setIsGenerating(false);
    }
  };

  const handleCopy = async () => {
    if (!prompt()) return;
    await navigator.clipboard.writeText(prompt());
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div class="ai-context-page">
      <div class="page-header">
        <div class="page-title-row">
          <div class="page-title-icon violet">
            <BrainCircuit size={24} />
          </div>
          <div>
            <h1 class="page-title">AI Agent System Prompt</h1>
            <p class="page-subtitle">
              Write and maintain the project instructions that your coding agents should always use
            </p>
          </div>
        </div>
      </div>

      <div class="context-layout">
        <div class="context-config">
          <div class="config-card">
            <h3>Target Project</h3>
            <ProjectSelect
              projects={projects() || []}
              selectedId={selectedProject()}
              onChange={(id) => {
                setSelectedProject(id);
                setSearchParams({ project: id }, { replace: true });
              }}
              placeholder="Choose a project..."
            />
            <p class="config-help">
              Your prompt is saved separately for each project and is also available through MCP.
            </p>
          </div>

          <Show when={latestSessionContext()}>
            {(memory) => (
              <div class="config-card session-context-card">
                <h3>Latest Session Context</h3>
                <p class="session-context-title">{memory().title}</p>
                <p class="session-context-meta">
                  Saved {new Date(memory().updated || memory().created).toLocaleString()}
                </p>
                <pre class="session-context-content">{memory().content}</pre>
                <p class="config-help">
                  This complete hand-off is saved by <code>cortex_save_memory</code>. Generate a baseline to include it in the editable system prompt.
                </p>
              </div>
            )}
          </Show>

          <div class="config-card">
            <h3>Optional Baseline Sources</h3>
            <p class="config-help">Use these only when you want CORTEX to draft or refresh a starting point.</p>
            <div class="source-options">
              <label class="source-option" classList={{ active: includeTasks() }}>
                <div class="source-info">
                  <div class="source-icon blue"><ListTodo size={16} /></div>
                  <div>
                    <div class="source-name">Active Tasks</div>
                    <div class="source-desc">Current sprint and uncompleted tasks</div>
                  </div>
                </div>
                <input
                  type="checkbox"
                  checked={includeTasks()}
                  onChange={(e) => setIncludeTasks(e.currentTarget.checked)}
                />
                <div class="custom-toggle" />
              </label>

              <label class="source-option" classList={{ active: includeVault() }}>
                <div class="source-info">
                  <div class="source-icon green"><Archive size={16} /></div>
                  <div>
                    <div class="source-name">Vault Entries</div>
                    <div class="source-desc">Architectural decisions and rules</div>
                  </div>
                </div>
                <input
                  type="checkbox"
                  checked={includeVault()}
                  onChange={(e) => setIncludeVault(e.currentTarget.checked)}
                />
                <div class="custom-toggle" />
              </label>

              <label class="source-option" classList={{ active: includeActivity() }}>
                <div class="source-info">
                  <div class="source-icon orange"><FileCode2 size={16} /></div>
                  <div>
                    <div class="source-name">Recent Activity</div>
                    <div class="source-desc">Recently changed files and commits</div>
                  </div>
                </div>
                <input
                  type="checkbox"
                  checked={includeActivity()}
                  onChange={(e) => setIncludeActivity(e.currentTarget.checked)}
                />
                <div class="custom-toggle" />
              </label>
            </div>
          </div>

          <button
            class="btn primary large save-btn"
            onClick={handleSave}
            disabled={savePrompt.isPending || !selectedProject()}
          >
            <Show when={!savePrompt.isPending} fallback={<><Save size={18} class="spin" /> Saving...</>}>
              <Save size={18} />
              Save My System Prompt
            </Show>
          </button>

          <button
            class="btn secondary large generate-btn"
            onClick={handleGenerate}
            disabled={isGenerating() || !selectedProject()}
          >
            <Show when={!isGenerating()} fallback={<><Sparkles size={18} class="spin" /> Generating baseline...</>}>
              <Sparkles size={18} />
              Generate Baseline
            </Show>
          </button>

          <Show when={saved()}>
            <div class="prompt-saved"><Check size={14} /> Saved for this project</div>
          </Show>
          <Show when={error()}>
            <div class="prompt-error">{error()}</div>
          </Show>
        </div>

        <div class="context-preview">
          <div class="preview-header">
            <h3>System Prompt</h3>
            <div class="preview-actions">
              <Show when={provider()}>
                <span class="provider-badge">
                  {provider() === 'custom' ? 'Manual' : 'Generated baseline'}
                </span>
              </Show>
              <span class="token-count">~{tokenEstimate()} tokens</span>
              <button class="btn secondary small" onClick={handleCopy} disabled={!prompt()}>
                <Show when={!copied()} fallback={<><Check size={14} class="text-green" /> Copied</>}>
                  <Copy size={14} /> Copy to Clipboard
                </Show>
              </button>
            </div>
          </div>
          <div class="preview-editor">
            <textarea
              class="prompt-textarea"
              value={prompt()}
              onInput={(e) => handlePromptChange(e.currentTarget.value)}
              placeholder={placeholderPrompt}
              aria-label="Project system prompt"
              spellcheck={false}
            />
          </div>
        </div>
      </div>
    </div>
  );
};


