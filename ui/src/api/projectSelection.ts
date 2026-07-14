import { Accessor, createEffect, createSignal } from 'solid-js';

const STORAGE_KEY = 'cortex.selectedProject';

export interface ProjectChoice {
  id: string;
}

function readStoredProject(): string {
  if (typeof window === 'undefined') return '';
  try {
    return window.localStorage.getItem(STORAGE_KEY) || '';
  } catch {
    return '';
  }
}

const [sharedSelectedProject, setSharedSelectedProject] = createSignal(readStoredProject());

export const selectedProjectId = sharedSelectedProject;

export function setSelectedProjectId(id: string): void {
  const next = id || '';
  setSharedSelectedProject(next);
  if (typeof window === 'undefined') return;
  try {
    if (next) window.localStorage.setItem(STORAGE_KEY, next);
    else window.localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Storage can be unavailable in privacy-restricted browser contexts.
  }
}

export function createProjectSelection(
  urlProject?: Accessor<string | undefined>,
  projects?: Accessor<readonly ProjectChoice[] | undefined>,
) {
  const [selected, setSelected] = createSignal(urlProject?.() || selectedProjectId());

  createEffect(() => {
    const fromUrl = urlProject?.() || '';
    const shared = selectedProjectId();
    const available = projects?.() ?? [];
    const requested = fromUrl || selected() || shared;

    if (!requested && available.length === 0) return;

    const resolved = available.length > 0
      ? (available.some((project) => project.id === requested) ? requested : available[0].id)
      : requested;

    if (resolved && selected() !== resolved) setSelected(resolved);
    if (resolved && shared !== resolved) setSelectedProjectId(resolved);
  });

  const select = (id: string) => {
    setSelected(id);
    setSelectedProjectId(id);
  };

  return { selected, select };
}