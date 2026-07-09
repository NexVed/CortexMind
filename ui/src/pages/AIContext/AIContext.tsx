import { Component, For, createSignal, Show } from 'solid-js';
import { useSearchParams } from '@solidjs/router';
import {
  BrainCircuit,
  Bot,
  FileCode2,
  ListTodo,
  Archive,
  ChevronDown,
  Sparkles,
  Copy,
  Check,
} from 'lucide-solid';
import { generateSystemPrompt } from '../../api/client';
import { useProjects } from '../../api/queries';
import { ProjectSelect } from '../../components/ProjectSelect/ProjectSelect';
import './AIContext.css';

const placeholderPrompt =
  'Select a scanned project and click "Generate System Prompt" to compile a tailored, copy-paste-ready prompt from its tech stack, authentication, features, structure, decisions and active tasks — then paste it into ChatGPT, Claude or any coding agent. No AI API required.';

export const AIContextPage: Component = () => {
  const [searchParams] = useSearchParams();
  const [selectedProject, setSelectedProject] = createSignal<string>(
    typeof searchParams.project === 'string' ? searchParams.project : ''
  );
  const [includeTasks, setIncludeTasks] = createSignal(true);
  const [includeVault, setIncludeVault] = createSignal(true);
  const [includeActivity, setIncludeActivity] = createSignal(false);
  const [isGenerating, setIsGenerating] = createSignal(false);
  const [copied, setCopied] = createSignal(false);
  const [prompt, setPrompt] = createSignal<string>('');
  const [provider, setProvider] = createSignal<string>('');
  const [tokenEstimate, setTokenEstimate] = createSignal<number>(0);
  const [error, setError] = createSignal<string>('');

  // Fetch real projects for the dropdown
  const projectsQuery = useProjects();
  const projects = () => projectsQuery.data;

  const promptLines = () => (prompt() ? prompt().split('\n') : placeholderPrompt.split('\n'));

  const handleGenerate = async () => {
    if (!selectedProject()) {
      setError('Pick a project first.');
      return;
    }
    setIsGenerating(true);
    setError('');
    try {
      const res = await generateSystemPrompt(selectedProject(), {
        include_tasks: includeTasks(),
        include_vault: includeVault(),
        include_activity: includeActivity(),
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

  const handleCopy = () => {
    if (!prompt()) return;
    navigator.clipboard.writeText(prompt());
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
            <h1 class="page-title">AI Agent Prompt Builder</h1>
            <p class="page-subtitle">Compile a copy-paste system prompt from your scanned codebase — paste it into ChatGPT, Claude or any agent</p>
          </div>
        </div>
      </div>

      <div class="context-layout">
        {/* Left Column - Configuration */}
        <div class="context-config">
          <div class="config-card">
            <h3>Target Project</h3>
            <ProjectSelect
              projects={projects() || []}
              selectedId={selectedProject()}
              onChange={setSelectedProject}
              placeholder="Choose a scanned project…"
            />
          </div>

          <div class="config-card">
            <h3>Include Knowledge Sources</h3>
            <div class="source-options">
              <label class={`source-option ${includeTasks() ? 'active' : ''}`}>
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

              <label class={`source-option ${includeVault() ? 'active' : ''}`}>
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

              <label class={`source-option ${includeActivity() ? 'active' : ''}`}>
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
            class={`btn primary large generate-btn ${isGenerating() ? 'generating' : ''}`}
            onClick={handleGenerate}
            disabled={isGenerating()}
          >
            <Show when={!isGenerating()} fallback={<><Sparkles size={18} class="spin" /> Generating Prompt...</>}>
              <Sparkles size={18} />
              Generate System Prompt
            </Show>
          </button>

          <Show when={error()}>
            <div class="prompt-error">{error()}</div>
          </Show>
        </div>

        {/* Right Column - Preview */}
        <div class="context-preview">
          <div class="preview-header">
            <h3>System Prompt</h3>
            <div class="preview-actions">
              <Show when={provider()}>
                <span class="provider-badge">Built locally</span>
              </Show>
              <span class="token-count">~{tokenEstimate() || Math.round(placeholderPrompt.length / 4)} tokens</span>
              <button class="btn secondary small" onClick={handleCopy} disabled={!prompt()}>
                <Show when={!copied()} fallback={<><Check size={14} class="text-green" /> Copied</>}>
                  <Copy size={14} /> Copy to Clipboard
                </Show>
              </button>
            </div>
          </div>
          <div class="preview-editor">
            <div class="preview-lines">
              <For each={promptLines()}>
                {(line) => (
                  <div class="preview-line">
                    <span class="line-content">{line || ' '}</span>
                  </div>
                )}
              </For>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
