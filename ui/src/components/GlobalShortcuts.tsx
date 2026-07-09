import { Component, onCleanup, onMount } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { cycleTheme, toggleSidebarCollapsed, isMac } from '../api/settings';

// GlobalShortcuts registers the keyboard shortcuts advertised on the Settings
// page. It renders nothing; it must live inside the Router so useNavigate works.
export const GlobalShortcuts: Component = () => {
  const navigate = useNavigate();

  const handler = (e: KeyboardEvent) => {
    const mod = isMac ? e.metaKey : e.ctrlKey;
    if (!mod) return;
    const key = e.key.toLowerCase();

    if (key === 'k' && !e.shiftKey) {
      e.preventDefault();
      (document.getElementById('global-search') as HTMLInputElement | null)?.focus();
    } else if (key === 'b' && !e.shiftKey) {
      e.preventDefault();
      toggleSidebarCollapsed();
    } else if (key === 't' && e.shiftKey) {
      e.preventDefault();
      cycleTheme();
    } else if (key === 'n' && !e.shiftKey) {
      e.preventDefault();
      navigate('/projects');
    } else if (key === 'h' && e.shiftKey) {
      e.preventDefault();
      navigate('/handoffs');
    } else if (key === 'g' && e.shiftKey) {
      e.preventDefault();
      navigate('/ai-context');
    } else if (key === '1') {
      e.preventDefault();
      navigate('/');
    } else if (key === '2') {
      e.preventDefault();
      navigate('/projects');
    }
  };

  onMount(() => window.addEventListener('keydown', handler));
  onCleanup(() => window.removeEventListener('keydown', handler));

  return null;
};
