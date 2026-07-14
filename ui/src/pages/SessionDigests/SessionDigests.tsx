import { Component, For, Show, createSignal } from 'solid-js';
import {
  Layers,
  ChevronDown,
  Sparkles,
  Copy,
  Check,
  Code2,
  FileText,
  Clock,
  Cpu,
} from 'lucide-solid';
import {
  type SessionDigest,
} from '../../api/client';
import { useProjects, useSessionDigests, useGenerateSessionDigest } from '../../api/queries';
import { ProjectSelect } from '../../components/ProjectSelect/ProjectSelect';
import { createProjectSelection } from '../../api/projectSelection';
import './SessionDigests.css';

// ── Minimal markdown renderer (headings, bullets, quote, bold) ──
function renderInline(text: string) {
  // Split on **bold** while keeping delimiters.
  const parts = text.split(/(\*\*[^*]+\*\*)/g);
  return (
    <For each={parts}>
      {(part) =>
        part.startsWith('**') && part.endsWith('**') ? (
          <strong>{part.slice(2, -2)}</strong>
        ) : (
          <>{part}</>
        )
      }
    </For>
  );
}

const MarkdownNote: Component<{ md: string }> = (props) => {
  const lines = () => props.md.split('\n');
  return (
    <div class="md-note">
      <For each={lines()}>
        {(line) => {
          const trimmed = line.trimEnd();
          if (trimmed.startsWith('## ')) return <h4 class="md-h">{renderInline(trimmed.slice(3))}</h4>;
          if (trimmed.startsWith('# ')) return <h3 class="md-h1">{renderInline(trimmed.slice(2))}</h3>;
          if (trimmed.startsWith('> ')) return <p class="md-quote">{renderInline(trimmed.slice(2))}</p>;
          if (trimmed.startsWith('- ')) return <div class="md-li">{renderInline(trimmed.slice(2))}</div>;
          if (trimmed.startsWith('#')) return <div class="md-tags">{trimmed}</div>;
          if (trimmed === '') return <div class="md-gap" />;
          return <p class="md-p">{renderInline(trimmed)}</p>;
        }}
      </For>
    </div>
  );
};

// ── Copyable compact-JSON block ─────────────────────────
const CompactJson: Component<{ data: Record<string, any> }> = (props) => {
  const [copied, setCopied] = createSignal(false);
  const text = () => JSON.stringify(props.data);
  const copy = () => {
    navigator.clipboard.writeText(text());
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <div class="compact-json">
      <div class="compact-json-head">
        <span class="compact-json-label">
          <Code2 size={13} /> Agent-to-agent JSON
        </span>
        <button class="btn secondary small" onClick={copy}>
          <Show when={!copied()} fallback={<><Check size={13} /> Copied</>}>
            <Copy size={13} /> Copy
          </Show>
        </button>
      </div>
      <pre class="compact-json-body"><code>{text()}</code></pre>
    </div>
  );
};

export const SessionDigestsPage: Component = () => {
  const [error, setError] = createSignal<string>('');

  const projectsQuery = useProjects();
  const projects = () => projectsQuery.data;
  const projectSelection = createProjectSelection(undefined, projects);
  const selectedProject = projectSelection.selected;
  const setSelectedProject = projectSelection.select;
  const digestsQuery = useSessionDigests(selectedProject);
  const digests = () => digestsQuery.data;
  const genDigestM = useGenerateSessionDigest();
  const isGenerating = () => genDigestM.isPending;

  const handleGenerate = async () => {
    if (!selectedProject()) {
      setError('Pick a project first.');
      return;
    }
    setError('');
    try {
      await genDigestM.mutateAsync({ projectId: selectedProject() });
    } catch (err: any) {
      setError(err?.message || 'Failed to generate digest');
    }
  };

  const formatDate = (iso: string) => {
    if (!iso) return '';
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
  };

  return (
    <div class="digests-page">
      <div class="page-header">
        <div class="page-title-row">
          <div class="page-title-icon violet">
            <Layers size={24} />
          </div>
          <div>
            <h1 class="page-title">Session Digests</h1>
            <p class="page-subtitle">
              Compressed, shareable summaries of what each AI agent did — readable notes plus a
              token-efficient JSON for the next agent.
            </p>
          </div>
        </div>
      </div>

      <div class="digests-toolbar">
        <ProjectSelect
          projects={projects() || []}
          selectedId={selectedProject()}
          onChange={setSelectedProject}
          placeholder="Choose a project…"
        />

        <button
          class="btn primary generate-digest-btn"
          onClick={handleGenerate}
          disabled={isGenerating() || !selectedProject()}
        >
          <Show when={!isGenerating()} fallback={<><Sparkles size={16} class="spin" /> Compressing…</>}>
            <Sparkles size={16} />
            Generate digest
          </Show>
        </button>
      </div>

      <Show when={error()}>
        <div class="digest-error" role="alert">
          {error()}
        </div>
      </Show>

      {/* States */}
      <Show
        when={selectedProject()}
        fallback={
          <div class="digest-empty" role="status">
            <Layers size={40} />
            <h3>Select a project</h3>
            <p>Pick a project above to view its session digests, or generate a new one.</p>
          </div>
        }
      >
        <Show
          when={!digestsQuery.isLoading}
          fallback={
            <div class="digest-skeletons" aria-busy="true" aria-label="Loading digests">
              <For each={[1, 2]}>{() => <div class="digest-skeleton" />}</For>
            </div>
          }
        >
          <Show
            when={(digests() ?? []).length > 0}
            fallback={
              <div class="digest-empty" role="status">
                <FileText size={40} />
                <h3>No digests yet</h3>
                <p>
                  Click <strong>Generate digest</strong> to compress this project's saved agent
                  memories into a session summary. Agents can also create one by calling the{' '}
                  <code>cortex_summarize_session</code> tool.
                </p>
              </div>
            }
          >
            <div class="digest-list">
              <For each={digests()}>
                {(d: SessionDigest) => (
                  <article class="digest-card">
                    <header class="digest-card-head">
                      <h2 class="digest-card-title">{d.title}</h2>
                      <div class="digest-meta">
                        <span class="digest-badge ide">{d.ide || 'unknown'}</span>
                        <span
                          class={`digest-badge provider ${d.provider === 'heuristic' ? 'muted' : ''}`}
                        >
                          <Cpu size={11} /> {d.provider === 'heuristic' ? 'offline' : d.provider}
                        </span>
                        <span class="digest-stat">{d.memory_count} memories</span>
                        <span class="digest-stat">~{d.token_count} tokens</span>
                        <span class="digest-stat">
                          <Clock size={11} /> {formatDate(d.created)}
                        </span>
                      </div>
                    </header>

                    <MarkdownNote md={d.summary_md} />

                    <Show when={d.digest_json}>
                      <CompactJson data={d.digest_json} />
                    </Show>
                  </article>
                )}
              </For>
            </div>
          </Show>
        </Show>
      </Show>
    </div>
  );
};
