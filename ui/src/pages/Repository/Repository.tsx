import { Component, For, createResource, Show, createSignal } from 'solid-js';
import { useParams } from '@solidjs/router';
import {
  Github, Eye, GitFork, Star, ChevronDown, GitBranch, Tag, Search, Plus, Code, MoreHorizontal,
  Folder, FileText, FileCode2, GitCommit,
  Sparkles, ScanSearch, Copy, MessageSquare, Network, FileKey, Layers, HardDrive, Clock, Scan
} from 'lucide-solid';
import { useProject, useScanProject } from '../../api/queries';
import './Repository.css';

export const RepositoryPage: Component = () => {
  const params = useParams();
  const projectId = params.id;

  const projectQuery = useProject(() => projectId);
  const project = () => projectQuery.data;
  const scanM = useScanProject();
  const [isScanning, setIsScanning] = createSignal(false);

  const handleScan = async () => {
    if (!projectId) return;
    setIsScanning(true);
    try {
      await scanM.mutateAsync(projectId);
    } catch (err) {
      console.error('Scan failed:', err);
    } finally {
      setIsScanning(false);
    }
  };

  const [repoInfo] = createResource(
    () => project()?.github_url,
    async (url) => {
      try {
        const match = url.match(/github\.com\/([^\/]+)\/([^\/]+)/);
        if (!match) return null;
        const [_, owner, repo] = match;
        const cleanRepo = repo.replace('.git', '');
        
        const [repoRes, commitRes] = await Promise.all([
          fetch(`https://api.github.com/repos/${owner}/${cleanRepo}`),
          fetch(`https://api.github.com/repos/${owner}/${cleanRepo}/commits?per_page=1`)
        ]);
        
        return {
          repo: repoRes.ok ? await repoRes.json() : null,
          commit: commitRes.ok ? (await commitRes.json())[0] : null
        };
      } catch {
        return null;
      }
    }
  );

  const [repoFiles] = createResource(
    () => project()?.github_url,
    async (url) => {
      try {
        const match = url.match(/github\.com\/([^\/]+)\/([^\/]+)/);
        if (!match) return [];
        const [_, owner, repo] = match;
        const cleanRepo = repo.replace('.git', '');
        const res = await fetch(`https://api.github.com/repos/${owner}/${cleanRepo}/contents`);
        if (!res.ok) return [];
        const data = await res.json();
        return data.map((item: any) => ({
          name: item.name,
          type: item.type === 'dir' ? 'folder' : 'file',
          message: 'Synced from GitHub',
          date: ''
        })).sort((a: any, b: any) => {
          if (a.type === 'folder' && b.type !== 'folder') return -1;
          if (a.type !== 'folder' && b.type === 'folder') return 1;
          return a.name.localeCompare(b.name);
        });
      } catch {
        return [];
      }
    }
  );

  const [repoReadme] = createResource(
    () => project()?.github_url,
    async (url) => {
      try {
        const match = url.match(/github\.com\/([^\/]+)\/([^\/]+)/);
        if (!match) return null;
        const [_, owner, repo] = match;
        const cleanRepo = repo.replace('.git', '');
        const res = await fetch(`https://api.github.com/repos/${owner}/${cleanRepo}/readme`, {
          headers: {
            'Accept': 'application/vnd.github.v3.html'
          }
        });
        if (!res.ok) return null;
        return await res.text();
      } catch {
        return null;
      }
    }
  );

  return (
    <div class="repo-page">
      <Show when={!projectQuery.isLoading} fallback={<div class="loading-state">Loading repository...</div>}>
        <div class="repo-header">
          <div class="repo-title-area">
            <Github size={24} />
            <h1 class="repo-name">{project()?.name || 'Loading...'}</h1>
          <span class="repo-badge private">Private</span>
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
              <Show when={repoFiles()?.length > 0} fallback={<div style="padding: 24px; text-align: center; color: var(--text-muted);">No files available. The repository might be private or empty.</div>}>
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
          
          <Show when={repoReadme()}>
            <div class="readme-container">
              <div class="readme-header">
                <FileText size={16} /> README.md
              </div>
              <div class="readme-content" innerHTML={repoReadme()!} />
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
                <h4>Clone Repository</h4>
                <p>Clone this repository to your workspace to start working with AI.</p>
              </div>
              <button class="btn secondary w-full"><Copy size={14} /> Clone Repo</button>
            </div>
            
            <div class="ai-card">
              <div class="ai-card-icon-wrap bg-pink-light"><MessageSquare size={18} class="text-pink" /></div>
              <div class="ai-card-content">
                <h4>Chat with Repository</h4>
                <p>Ask questions about your codebase, architecture, dependencies and more.</p>
              </div>
              <button class="btn primary-pink light w-full"><MessageSquare size={14} /> Start Chat</button>
            </div>
          </div>

          <div class="sidebar-section">
            <div class="sidebar-header">
              <h3>Repository Insights</h3>
            </div>
            <div class="insights-list">
              <div class="insight-row"><div class="insight-label"><FileCode2 size={14} /> Language</div><div class="insight-val font-medium">{project()?.technologies?.[0] || 'Unknown'}</div></div>
              <div class="insight-row"><div class="insight-label"><HardDrive size={14} /> Size</div><div class="insight-val font-medium">Unknown</div></div>
              <div class="insight-row"><div class="insight-label"><Layers size={14} /> Files</div><div class="insight-val font-medium">{repoFiles()?.length || 0}</div></div>
              <div class="insight-row"><div class="insight-label"><Network size={14} /> Lines of Code</div><div class="insight-val font-medium">...</div></div>
              <div class="insight-row"><div class="insight-label"><Clock size={14} /> Last Commit</div><div class="insight-val font-medium">...</div></div>
              <div class="insight-row"><div class="insight-label"><FileKey size={14} /> License</div><div class="insight-val font-medium">MIT</div></div>
            </div>
            <p class="insights-note">✨ AI insights are generated after scanning your repository.</p>
          </div>
        </div>
      </div>
      </Show>
    </div>
  );
};
