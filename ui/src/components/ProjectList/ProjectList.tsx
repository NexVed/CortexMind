import { Component, For, Show } from 'solid-js';
import { Plus } from 'lucide-solid';
import { useProjects } from '../../api/queries';
import './ProjectList.css';

function getProgressClass(value: number): string {
  if (value >= 60) return 'high';
  if (value >= 30) return 'medium';
  return 'low';
}

export const ProjectList: Component = () => {
  const projectsQuery = useProjects();
  const projects = () => projectsQuery.data;

  const displayProjects = () => {
    const p = projects();
    if (!p) return [];
    return p.slice(0, 4); // Show only top 4 on dashboard
  };

  return (
    <div class="project-list">
      <div class="project-list-header">
        <span class="project-list-title">Your Projects</span>
        <button class="project-list-add-btn">
          <Plus size={14} />
          New Project
        </button>
      </div>
      
      <Show when={!projectsQuery.isLoading && displayProjects().length === 0}>
        <div style={{ padding: '24px', "text-align": 'center', color: 'var(--text-muted)' }}>
          No projects yet. Click 'New Project' to get started.
        </div>
      </Show>

      <For each={displayProjects()}>
        {(project) => {
          const initial = project.name ? project.name.charAt(0).toUpperCase() : '?';
          const tags = project.technologies || [];
          
          return (
            <div class="project-row">
              <div class="project-avatar" style={{ background: project.icon_color || 'var(--accent)' }}>
                {initial}
              </div>
              <div class="project-info">
                <div class="project-name-row">
                  <span class="project-name">{project.name}</span>
                </div>
                <span class="project-description">{project.description || 'No description'}</span>
                <div class="project-tags">
                  <For each={tags.slice(0, 3)}>
                    {(tag) => (
                      <span class="project-tag">{tag}</span>
                    )}
                  </For>
                  <Show when={tags.length > 3}>
                    <span class="project-tag">+{tags.length - 3}</span>
                  </Show>
                </div>
              </div>
              <div class="project-meta">
                <div class="project-time-row">
                  <span class="project-time">
                    {new Date(project.last_activity || project.updated).toLocaleDateString()}
                  </span>
                  <Show when={project.status === 'active'}>
                    <span class="project-live-dot" />
                  </Show>
                </div>
                <div class="project-progress-row">
                  <span class="project-progress-value">{project.progress}%</span>
                  <div class="project-progress-track">
                    <div
                      class={`project-progress-fill ${getProgressClass(project.progress)}`}
                      style={{ width: `${project.progress}%` }}
                    />
                  </div>
                </div>
              </div>
            </div>
          );
        }}
      </For>
    </div>
  );
};
