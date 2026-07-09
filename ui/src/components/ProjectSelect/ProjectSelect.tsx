import { Component, createSignal, For, Show, onCleanup, createMemo } from 'solid-js';
import { ChevronDown, Search, Check, FolderKanban } from 'lucide-solid';
import './ProjectSelect.css';

interface Project {
  id: string;
  name: string;
  icon_color?: string;
  description?: string;
}

interface ProjectSelectProps {
  projects: Project[];
  selectedId: string;
  onChange: (id: string) => void;
  placeholder?: string;
}

export const ProjectSelect: Component<ProjectSelectProps> = (props) => {
  const [isOpen, setIsOpen] = createSignal(false);
  const [searchQuery, setSearchQuery] = createSignal('');
  let containerRef: HTMLDivElement | undefined;

  const selectedProject = createMemo(() => {
    return props.projects.find((p) => p.id === props.selectedId);
  });

  const filteredProjects = createMemo(() => {
    const query = searchQuery().toLowerCase().trim();
    if (!query) return props.projects;
    return props.projects.filter((p) =>
      p.name.toLowerCase().includes(query)
    );
  });

  const handleSelect = (id: string) => {
    props.onChange(id);
    setIsOpen(false);
    setSearchQuery('');
  };

  const handleOutsideClick = (e: MouseEvent) => {
    if (containerRef && !containerRef.contains(e.target as Node)) {
      setIsOpen(false);
    }
  };

  document.addEventListener('click', handleOutsideClick);
  onCleanup(() => {
    document.removeEventListener('click', handleOutsideClick);
  });

  return (
    <div class="project-select-container" ref={containerRef}>
      {/* Trigger Button */}
      <button
        type="button"
        class={`project-select-trigger ${isOpen() ? 'active' : ''}`}
        onClick={() => setIsOpen(!isOpen())}
      >
        <Show
          when={selectedProject()}
          fallback={<span class="placeholder">{props.placeholder || 'Choose a scanned project...'}</span>}
        >
          {(proj) => (
            <div class="selected-project-display">
              <div
                class="project-select-avatar"
                style={{ background: proj().icon_color || 'var(--accent)' }}
              >
                {proj().name.charAt(0).toUpperCase()}
              </div>
              <span class="project-select-name">{proj().name}</span>
            </div>
          )}
        </Show>
        <ChevronDown size={16} class={`select-chevron ${isOpen() ? 'open' : ''}`} />
      </button>

      {/* Dropdown Panel */}
      <Show when={isOpen()}>
        <div class="project-select-dropdown">
          {/* Search Box */}
          <div class="project-select-search">
            <Search size={14} class="search-icon" />
            <input
              type="text"
              placeholder="Search projects..."
              value={searchQuery()}
              onInput={(e) => setSearchQuery(e.currentTarget.value)}
              onClick={(e) => e.stopPropagation()}
              ref={(el) => setTimeout(() => el?.focus(), 50)}
            />
          </div>

          {/* List of projects */}
          <div class="project-select-list">
            <Show
              when={filteredProjects().length > 0}
              fallback={<div class="no-projects-found">No projects match your search</div>}
            >
              <For each={filteredProjects()}>
                {(proj) => {
                  const isCurrent = proj.id === props.selectedId;
                  return (
                    <div
                      class={`project-select-item ${isCurrent ? 'selected' : ''}`}
                      onClick={() => handleSelect(proj.id)}
                    >
                      <div
                        class="project-select-avatar"
                        style={{ background: proj.icon_color || 'var(--accent)' }}
                      >
                        {proj.name.charAt(0).toUpperCase()}
                      </div>
                      <div class="project-select-item-info">
                        <span class="project-item-name">{proj.name}</span>
                        <Show when={proj.description}>
                          <span class="project-item-desc">{proj.description}</span>
                        </Show>
                      </div>
                      <Show when={isCurrent}>
                        <Check size={14} class="check-icon" />
                      </Show>
                    </div>
                  );
                }}
              </For>
            </Show>
          </div>
        </div>
      </Show>
    </div>
  );
};
