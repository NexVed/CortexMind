import { Component, For, createResource, Show, createSignal, createEffect } from 'solid-js';
import { useParams } from '@solidjs/router';
import {
  Github, Eye, GitFork, Star, ChevronDown, GitBranch, Tag, Search, Plus, Code, MoreHorizontal,
  Folder, FileText, FileCode2, GitCommit,
  Sparkles, ScanSearch, Copy, Network, FileKey, Layers, HardDrive, Clock, Scan
} from 'lucide-solid';
import { useProject, useScanProject } from '../../api/queries';
import { getRepositoryInsights, getRepositoryView, getRepositoryImage, scanWorkingTree, type RepositoryView, type WorkingTreeChanges } from '../../api/client';
import { settings } from '../../api/settings';
import './Repository.css';
async function prepareReadme(view: RepositoryView, projectId: string): Promise<string> {
  const document = new DOMParser().parseFromString(view.readme_html, 'text/html');
  const repoBase = `https://github.com/${view.repo.full_name}/`;
  const branch = view.repo.default_branch;
  const rawPrefix = `https://raw.githubusercontent.com/${view.repo.full_name}/${branch}/`;
  const directory = view.readme_path.includes('/') ? view.readme_path.slice(0, view.readme_path.lastIndexOf('/') + 1) : '';
  const base = rawPrefix + directory;
  const images = Array.from(document.querySelectorAll('img'));
  await Promise.all(images.map(async (image, index) => {
    const source = image.getAttribute('src') || '';
    try {
      if (source.startsWith('data:image/')) return;
      let resolved = new URL(source, source.startsWith('/') ? 'https://github.com' : base).href;
      for (const kind of ['blob', 'raw']) resolved = resolved.replace(`${repoBase}${kind}/${branch}/`, rawPrefix);
      if (resolved.startsWith(rawPrefix) && index < 16) {
        const path = decodeURIComponent(resolved.slice(rawPrefix.length).split(/[?#]/)[0]);
        image.setAttribute('src', await getRepositoryImage(projectId, path, branch));
      } else if (resolved.startsWith('https://')) {
        image.setAttribute('src', resolved);
      } else image.removeAttribute('src');
      image.setAttribute('loading', 'lazy');
      image.setAttribute('referrerpolicy', 'no-referrer');
    } catch {
      image.removeAttribute('src');
      image.setAttribute('title', 'Image unavailable');
    }
  }));
  for (const link of document.querySelectorAll('a')) {
    const href = link.getAttribute('href');
    if (!href || href.startsWith('#')) continue;
    try {
      const resolved = new URL(href, `${repoBase}blob/${encodeURIComponent(branch)}/${directory}`);
      if (!['https:', 'http:', 'mailto:'].includes(resolved.protocol)) { link.removeAttribute('href'); continue; }
      link.href = resolved.href;
      link.target = '_blank';
      link.rel = 'noopener noreferrer';
    } catch { link.removeAttribute('href'); }
  }
  return document.body.innerHTML;
}

function readmeDocument(body: string): string {
  const dark = settings.theme === 'Dark' || (settings.theme === 'System' && window.matchMedia('(prefers-color-scheme: dark)').matches);
  return `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src https: data:; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'"><style>
    :root{color-scheme:${dark ? 'dark' : 'light'}}body{font:15px/1.65 system-ui,sans-serif;margin:20px;color:${dark ? '#e6edf3' : '#24292f'};overflow-wrap:anywhere}img{max-width:100%;height:auto}a{color:${dark ? '#58a6ff' : '#0969da'}}pre{padding:16px;overflow:auto;background:${dark ? '#161b22' : '#f6f8fa'};border-radius:6px}code{font:13px/1.6 monospace}table{display:block;max-width:100%;overflow:auto;border-collapse:collapse}td,th{border:1px solid #80808055;padding:6px 12px}blockquote{border-left:3px solid #80808055;padding-left:16px;margin-left:0}h1,h2{border-bottom:1px solid #80808033;padding-bottom:8px}
    </style></head><body>${body}</body></html>`;
}

export const RepositoryPage: Component = () => {
  const params = useParams();
  const projectId = () => params.id;

  const projectQuery = useProject(projectId);
  const project = () => projectQuery.data;
  const scanM = useScanProject();
  const [isScanning, setIsScanning] = createSignal(false);
  const [scanError, setScanError] = createSignal('');
  const [localPath, setLocalPath] = createSignal<string | null>(null);
  const [workingChanges, setWorkingChanges] = createSignal<WorkingTreeChanges | null>(null);
  createEffect(() => { projectId(); setLocalPath(null); setWorkingChanges(null); });
  const [insights, { refetch: refetchInsights }] = createResource(projectId, async (id) => id ? getRepositoryInsights(id) : null);
  const formatSize = (bytes?: number) => { if (!bytes) return '—'; const units = ['B', 'KB', 'MB', 'GB']; let value = bytes; let index = 0; while (value >= 1024 && index < units.length - 1) { value /= 1024; index++; } return `${value.toFixed(index ? 1 : 0)} ${units[index]}`; };

  const handleScan = async () => {
    if (!projectId()) return;
    setIsScanning(true);
    setScanError('');
    try {
      await scanM.mutateAsync(projectId());
      await refetchInsights();
    } catch (err: any) {
      setScanError(err?.message || 'Scan failed');
    } finally {
      setIsScanning(false);
    }
  };

  const handleLocalScan = async () => {
    setIsScanning(true);
    setScanError('');
    setWorkingChanges(null);
    try {
      const result = await scanWorkingTree(projectId(), localPath()?.trim() || project()?.path || '');
      setWorkingChanges(result.changes);
      await Promise.all([projectQuery.refetch(), refetchInsights()]);
    } catch (err: any) { setScanError(err?.message || 'Local checkout scan failed'); }
    finally { setIsScanning(false); }
  };

  const [repoInfo] = createResource(
    () => project()?.github_url ? projectId() : null,
    getRepositoryView
  );
  const repoFiles = () => (repoInfo()?.files || []).map((file) => ({
    name: file.name,
    type: file.type === 'dir' ? 'folder' : 'file',
    message: 'Synced from GitHub',
    date: '',
  })).sort((a, b) => a.type === b.type ? a.name.localeCompare(b.name) : a.type === 'folder' ? -1 : 1);
  const [repoReadme] = createResource(repoInfo, (view) => prepareReadme(view, projectId()));

  return (
    <div class="repo-page">
      <Show when={!projectQuery.isLoading} fallback={<div class="loading-state">Loading repository...</div>}>
        <div class="repo-header">
          <div class="repo-title-area">
            <Github size={24} />
            <h1 class="repo-name">{project()?.name || 'Loading...'}</h1>
          <span class={`repo-badge ${repoInfo()?.repo?.private ? 'private' : ''}`}>
            {project()?.github_url ? (repoInfo.loading ? 'Loading…' : repoInfo()?.repo ? (repoInfo()!.repo.private ? 'Private' : 'Public') : 'Unavailable') : 'Local'}
          </span>
        </div>
        <div class="repo-header-actions">
          <div class="action-group">
            <button class="btn secondary small"><Eye size={14} /> Watch <ChevronDown size={14} /></button>
            <span class="action-count">{repoInfo()?.repo?.subscribers_count || 0}</span>
          </div>
          <div class="action-group">
            <button class="btn secondary small"><GitFork size={14} /> Fork <ChevronDown size={14} /></button>
            <span class="action-count">{repoInfo()?.repo?.forks_count || 0}</span>
          </div>
          <div class="action-group">
            <button class="btn secondary small"><Star size={14} /> Star <ChevronDown size={14} /></button>
            <span class="action-count">{repoInfo()?.repo?.stargazers_count || 0}</span>
          </div>
        </div>
      </div>

      <Show when={repoInfo.error || scanError()}>
        <div role="alert" style={{ color: 'var(--red)', padding: '12px', 'overflow-wrap': 'anywhere' }}>
          {scanError() || repoInfo.error?.message || 'Could not load repository. Reconnect GitHub and try again.'}
        </div>
      </Show>
      <div class="repo-toolbar">
        <div class="branch-selector">
          <button class="btn secondary small"><GitBranch size={14} /> {repoInfo()?.repo?.default_branch || 'main'} <ChevronDown size={14} /></button>
          <div class="branch-stats">
            <span><GitBranch size={14} /> 1 Branch</span>
            <span><Tag size={14} /> 0 Tags</span>
          </div>
        </div>
        
        <div class="file-actions">
          <div class="search-file">
            <Search size={14} class="icon-search" />
            <input type="text" placeholder="Go to file" />
            <span class="shortcut">T</span>
          </div>
          <button class="btn secondary small">Add file <ChevronDown size={14} /></button>
          <button class="btn primary-dark small"><Code size={14} /> Code <ChevronDown size={14} /></button>
          <button class="btn secondary small icon-only"><MoreHorizontal size={14} /></button>
        </div>
      </div>

      <div class="repo-layout">
        <div class="repo-main">
          <div class="file-tree-container">
            <div class="file-tree-header">
              <div class="last-commit-info">
                <img 
                  src={repoInfo()?.commit?.author?.avatar_url || `https://ui-avatars.com/api/?name=${repoInfo()?.commit?.author?.login || repoInfo()?.commit?.commit?.author?.name || 'User'}&background=333&color=fff`} 
                  class="avatar-small" 
                />
                <span class="committer">{repoInfo()?.commit?.author?.login || repoInfo()?.commit?.commit?.author?.name || 'Unknown'}</span>
                <span class="commit-msg" title={repoInfo()?.commit?.commit?.message}>{repoInfo()?.commit?.commit?.message?.split('\n')[0] || 'No commits found'}</span>
              </div>
              <div class="commit-meta">
                <span class="commit-hash">{repoInfo()?.commit?.sha?.substring(0, 7) || '-------'}</span>
                <span class="commit-time">· {repoInfo()?.commit?.commit?.author?.date ? new Date(repoInfo()!.commit!.commit!.author!.date).toLocaleDateString() : ''}</span>
                <span class="commit-total"><GitCommit size={14} /> Commits</span>
              </div>
            </div>
            
            <div class="file-list">
              <Show when={repoFiles()?.length > 0} fallback={<div style="padding: 24px; text-align: center; color: var(--text-muted);">No files available. Scan a local project or check the repository connection.</div>}>
                <For each={repoFiles()}>
                  {(file) => (
                    <div class="file-row">
                      <div class="file-name-col">
                        {file.type === 'folder' ? <Folder size={16} class="icon-folder" fill="var(--text-secondary)" /> : <FileText size={16} class="icon-file" />}
                        <span>{file.name}</span>
                      </div>
                      <div class="file-msg-col">{file.message}</div>
                      <div class="file-date-col">{file.date}</div>
                    </div>
                  )}
                </For>
              </Show>
            </div>
          </div>
          
          <Show when={repoReadme()} fallback={<div class="small" style={{ padding: '16px' }}>{repoReadme.loading ? 'Loading README…' : repoReadme.error ? 'Could not load README.' : 'No README available.'}</div>}>
            <div class="readme-container">
              <div class="readme-header">
                <FileText size={16} /> README.md
              </div>
              <iframe class="readme-frame" title="Repository README" sandbox="allow-popups allow-popups-to-escape-sandbox" srcdoc={readmeDocument(repoReadme()!)} />
            </div>
          </Show>
        </div>

        <div class="repo-sidebar">
          <div class="sidebar-section">
            <div class="sidebar-header ai-assistant-header">
              <Sparkles size={16} class="icon-sparkle" />
              <h3>AI Assistant</h3>
              <span class="badge beta">BETA</span>
            </div>
            
            <div class="ai-card">
              <div class="ai-card-icon-wrap bg-pink-light"><ScanSearch size={18} class="text-pink" /></div>
              <div class="ai-card-content">
                <h4>Scan Repository</h4>
                <p>AI will analyze your codebase, structure, dependencies and generate insights.</p>
              </div>
              <button 
                class={`btn primary-pink light w-full ${isScanning() ? 'scanning' : ''}`}
                onClick={handleScan}
                disabled={isScanning()}
              >
                <Show when={!isScanning()} fallback={<Scan class="spin" size={14} />}>
                  <ScanSearch size={14} />
                </Show>
                {isScanning() ? 'Scanning...' : 'Scan Now'}
              </button>
            </div>
            
            <div class="ai-card">
              <div class="ai-card-icon-wrap bg-red-light"><Copy size={18} class="text-red" /></div>
              <div class="ai-card-content">
                <h4>Local checkout</h4>
                <p>Scan the folder where you cloned this repo to review changes before pushing. This updates MCP's code graph.</p>
              </div>
              <form class="working-tree-form" onSubmit={(event) => { event.preventDefault(); void handleLocalScan(); }}>
                <label for="checkout-path">Cloned repository folder</label>
                <input id="checkout-path" value={localPath() ?? project()?.path ?? ''} onInput={(event) => setLocalPath(event.currentTarget.value)} placeholder="C:\Users\you\source\repo" disabled={isScanning()} />
                <button type="submit" class="btn secondary w-full" disabled={isScanning()}><ScanSearch size={14} /> {isScanning() ? 'Scanning…' : 'Scan local changes'}</button>
              </form>
              <Show when={workingChanges()}>{(changes) => <div class="working-tree-summary" role="status">
                <strong>{changes().branch} · {changes().files.length} changed files{changes().files_truncated ? '+' : ''}</strong>
                <p>{changes().upstream ? `${changes().ahead} commits ahead · ${changes().behind} behind ${changes().upstream} (last fetched)` : 'No upstream configured; local changes shown.'}</p>
                <For each={changes().files.slice(0, 20)}>{(file) => <div><code>{file.untracked ? '??' : file.index_status + file.worktree_status}</code> {file.path}</div>}</For>
                <Show when={changes().files.length > 20}><p>More files are available through MCP.</p></Show>
                <For each={changes().local_commits.slice(0, 5)}>{(commit) => <p>{commit}</p>}</For>
                <p>MCP includes diffs for tracked changes. New files are listed; no pull or push is performed.</p>
              </div>}</Show>
            </div>
            

          </div>

          <div class="sidebar-section">
            <div class="sidebar-header">
              <h3>Repository Insights</h3>
            </div>
            <div class="insights-list">
              <div class="insight-row"><div class="insight-label"><FileCode2 size={14} /> Language</div><div class="insight-val font-medium">{insights()?.language || 'Loading…'}</div></div>
              <div class="insight-row"><div class="insight-label"><HardDrive size={14} /> Size</div><div class="insight-val font-medium">{formatSize(insights()?.size_bytes)}</div></div>
              <div class="insight-row"><div class="insight-label"><Layers size={14} /> Files</div><div class="insight-val font-medium">{insights()?.available ? insights()!.files : '—'}</div></div>
              <div class="insight-row"><div class="insight-label"><Network size={14} /> Lines of Code</div><div class="insight-val font-medium">{insights()?.available ? insights()!.lines_of_code.toLocaleString() : '—'}</div></div>
              <div class="insight-row"><div class="insight-label"><Clock size={14} /> Last Commit</div><div class="insight-val font-medium">{insights()?.last_commit || '—'}</div></div>
              <div class="insight-row"><div class="insight-label"><FileKey size={14} /> License</div><div class="insight-val font-medium">{insights()?.license || 'Loading…'}</div></div>
            </div>
            <p class="insights-note">✨ AI insights are generated after scanning your repository.</p>
          </div>
          <Show when={insights()?.available && insights()!.languages?.length}>
            <div class="sidebar-section language-breakdown">
              <div class="sidebar-header"><h3>Languages</h3></div>
              <div class="language-stack">
                <For each={insights()!.languages.slice(0, 6)}>{(language, index) => <span class={`language-segment language-${index()}`} style={{ width: `${language.percentage}%` }} />}</For>
              </div>
              <div class="language-legend">
                <For each={insights()!.languages.slice(0, 6)}>{(language, index) => <div class="language-item"><span class={`language-dot language-${index()}`} /><strong>{language.name}</strong><span>{language.percentage.toFixed(1)}%</span></div>}</For>
              </div>
            </div>
          </Show>
          </div>
        </div>
      </Show>
    </div>
  );
};






