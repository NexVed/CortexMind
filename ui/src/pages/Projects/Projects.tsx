import { Component, For, createSignal, createResource, Show } from 'solid-js';
import {
  FolderKanban,
  Plus,
  Search,
  MoreVertical,
  Activity,
  Github,
  Play,
  CheckCircle2,
  Scan,
} from 'lucide-solid';
import { listProjects, scanProject } from '../../api/client';
import './Projects.css';

export const ProjectsPage: Component = () => {
  const [searchQuery, setSearchQuery] = createSignal('');
  const [projects, { mutate, refetch }] = createResource(() => listProjects());
  const [scanning, setScanning] = createSignal<Record<string, boolean>>({});

  const filteredProjects = () => {
    const p = projects();
    if (!p) return [];
    if (!searchQuery()) return p;
    const q = searchQuery().toLowerCase();
    return p.filter(
      (pr) =>
        pr.name.toLowerCase().includes(q) ||
        (pr.description && pr.description.toLowerCase().includes(q))
    );
  };

  const handleScan = async (projectId: string) => {
    setScanning((prev) => ({ ...prev, [projectId]: true }));
    try {
      await scanProject(projectId);
      refetch();
    } catch (err) {
      console.error('Scan failed:', err);
    } finally {
      setScanning((prev) => ({ ...prev, [projectId]: false }));
    }
  };

  return (
    <div class="projects-page">
      <div class="page-header">
        <div class="page-title-row">
          <div class="page-title-icon purple">
            <FolderKanban size={24} />
          </div>
          <div>
            <h1 class="page-title">Projects</h1>
            <p class="page-subtitle">Manage your local and remote codebases</p>
          </div>
        </div>
        <div class="page-actions">
          <button class="btn primary">
            <Plus size={16} />
            Add Project
          </button>
        </div>
      </div>

      <div class="projects-toolbar">
        <div class="search-box">
          <Search size={16} class="search-icon" />
          <input
            type="text"
            placeholder="Search projects..."
            value={searchQuery()}
            onInput={(e) => setSearchQuery(e.currentTarget.value)}
          />
        </div>
        <div class="projects-filters">
          <button class="filter-btn active">All</button>
          <button class="filter-btn">Active</button>
          <button class="filter-btn">Archived</button>
        </div>
      </div>

      <Show when={projects.loading}>
        <div class="loading-state">Loading projects...</div>
      </Show>

      <Show when={!projects.loading && filteredProjects().length === 0}>
        <div class="empty-state">
          <div class="empty-icon"><FolderKanban size={32} /></div>
          <h3>No projects found</h3>
          <p>Get started by adding your first project repository.</p>
          <button class="btn primary mt-4">
            <Plus size={16} />
            Add Project
          </button>
        </div>
      </Show>

      <div class="projects-list">
        <For each={filteredProjects()}>
          {(project) => {
            const initial = project.name ? project.name.charAt(0).toUpperCase() : '?';
            const tags = project.technologies || [];
            const isScanning = scanning()[project.id];
            
            return (
              <div class="project-list-item">
                <div class="project-list-item-left">
                  <div class="project-card-avatar" style={{ background: project.icon_color || 'var(--accent)' }}>
                    {initial}
                  </div>
                  <div class="project-list-item-info">
                    <h3 class="project-card-name">{project.name}</h3>
                    <div class="project-card-status">
                      <Show
                        when={project.status === 'active'}
                        fallback={<span class="status-dot archived" />}
                      >
                        <span class="status-dot active" />
                      </Show>
                      {project.status}
                    </div>
                  </div>
                </div>

                <div class="project-list-item-desc">
                  <p class="project-card-desc">{project.description || 'No description provided.'}</p>
                </div>

                <div class="project-list-item-metrics">
                  <div class="metric">
                    <Activity size={14} />
                    <span>{project.progress}% indexed</span>
                  </div>
                  <Show when={project.github_url}>
                    <div class="metric">
                      <Github size={14} />
                      <span>Remote synced</span>
                    </div>
                  </Show>
                </div>

                <div class="project-list-item-actions">
                  <span class="project-card-time">
                    Updated {new Date(project.updated).toLocaleDateString()}
                  </span>
                  <button 
                    class={`btn secondary small ${isScanning ? 'scanning' : ''}`}
                    onClick={() => handleScan(project.id)}
                    disabled={isScanning}
                  >
                    <Show when={!isScanning} fallback={<Scan class="spin" size={14} />}>
                      <Scan size={14} />
                    </Show>
                    {isScanning ? 'Scanning...' : 'Scan'}
                  </button>
                  <button class="btn primary small">
                    <Play size={14} />
                    Open
                  </button>
                  <button class="icon-btn" aria-label="More options">
                    <MoreVertical size={16} />
                  </button>
                </div>
              </div>
            );
          }}
        </For>
      </div>
    </div>
  );
};
