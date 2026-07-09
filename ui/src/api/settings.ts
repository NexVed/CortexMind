// Device-level client state for CORTEX, owned by a Zustand store.
//
// Zustand (vanilla + `persist`) is the single source of truth for UI
// preferences and handles localStorage automatically. Because this is a
// SolidJS app, we bridge the store into Solid's reactivity with a reconciled
// `createStore` mirror kept in sync via `subscribe`, so components get
// fine-grained reactive reads (`settings.theme`, `settings.toggles[...]`).
//
// The exported helper API is unchanged from the previous implementation, so
// every consumer keeps working — only the internals moved to Zustand.

import { createStore as createZustandStore } from 'zustand/vanilla';
import { persist, createJSONStorage } from 'zustand/middleware';
import { createStore, reconcile } from 'solid-js/store';
import { createRoot } from 'solid-js';

export type Theme = 'Light' | 'Dark' | 'System';
export type SidebarPosition = 'Left' | 'Right';

export interface SettingsData {
  toggles: Record<string, boolean>;
  theme: Theme;
  sidebarPosition: SidebarPosition;
  sidebarCollapsed: boolean;
}

interface SettingsState extends SettingsData {
  setToggle: (label: string, value?: boolean) => void;
  setTheme: (theme: Theme) => void;
  setSidebarPosition: (position: SidebarPosition) => void;
  toggleSidebarCollapsed: () => void;
  cycleTheme: () => void;
  reset: () => void;
}

const STORAGE_KEY = 'cortex.settings';

// Defaults mirror the labels rendered by the Settings page.
const DEFAULT_TOGGLES: Record<string, boolean> = {
  // General
  'Auto-scan repositories': true,
  'Live file watching': true,
  'Send anonymous usage data': false,
  'Auto-update': true,
  // Appearance
  'Compact mode': false,
  'Show file previews': true,
  // Repository defaults
  'Auto-scan on import': true,
  'Deep analysis': false,
  'Include hidden files': false,
  'Extract functions & classes': true,
  // Sync & Git
  'Auto-commit memory bundle': false,
  'Push on export': false,
  'Include vault entries': true,
  'Include agent memories': true,
};

const DEFAULT_DATA: SettingsData = {
  toggles: DEFAULT_TOGGLES,
  theme: 'Light',
  sidebarPosition: 'Left',
  sidebarCollapsed: false,
};

// ── Zustand store (source of truth + persistence) ───────
export const settingsStore = createZustandStore<SettingsState>()(
  persist(
    (set) => ({
      ...DEFAULT_DATA,
      setToggle: (label, value) =>
        set((s) => ({ toggles: { ...s.toggles, [label]: value === undefined ? !s.toggles[label] : value } })),
      setTheme: (theme) => set({ theme }),
      setSidebarPosition: (sidebarPosition) => set({ sidebarPosition }),
      toggleSidebarCollapsed: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
      cycleTheme: () => set((s) => ({ theme: s.theme === 'Dark' ? 'Light' : 'Dark' })),
      reset: () => set({ ...DEFAULT_DATA, toggles: { ...DEFAULT_TOGGLES } }),
    }),
    {
      name: STORAGE_KEY,
      storage: createJSONStorage(() => localStorage),
      // Persist data only, never the action functions.
      partialize: (s) => ({
        toggles: s.toggles,
        theme: s.theme,
        sidebarPosition: s.sidebarPosition,
        sidebarCollapsed: s.sidebarCollapsed,
      }),
      // Merge persisted data over defaults so newly-added toggles appear.
      merge: (persisted, current) => {
        const p = (persisted ?? {}) as Partial<SettingsData>;
        return {
          ...current,
          ...p,
          toggles: { ...DEFAULT_TOGGLES, ...(p.toggles ?? {}) },
        };
      },
    }
  )
);

const pick = (s: SettingsState): SettingsData => ({
  toggles: s.toggles,
  theme: s.theme,
  sidebarPosition: s.sidebarPosition,
  sidebarCollapsed: s.sidebarCollapsed,
});

// ── Solid reactive mirror (read side) ───────────────────
// A reconciled createStore that follows the Zustand store so JSX reads are
// fine-grained reactive.
const settings = createRoot(() => {
  const [state, setState] = createStore<SettingsData>(pick(settingsStore.getState()));
  settingsStore.subscribe((s) => setState(reconcile(pick(s))));
  return state;
});

export { settings };

// ── OS-aware modifier key (for shortcut labels + matching) ──
export const isMac =
  typeof navigator !== 'undefined' && /Mac|iPod|iPhone|iPad/.test(navigator.platform);
export const modKey = isMac ? '\u2318' : 'Ctrl'; // ⌘ vs Ctrl

// ── Appearance application ──────────────────────────────
function applyAppearance() {
  const s = settingsStore.getState();
  const root = document.documentElement;

  const resolved =
    s.theme === 'System'
      ? window.matchMedia('(prefers-color-scheme: dark)').matches
        ? 'dark'
        : 'light'
      : s.theme.toLowerCase();
  if (resolved === 'dark') root.setAttribute('data-theme', 'dark');
  else root.removeAttribute('data-theme');

  root.setAttribute('data-sidebar', s.sidebarPosition.toLowerCase());
  root.setAttribute('data-density', s.toggles['Compact mode'] ? 'compact' : 'comfortable');
  root.setAttribute('data-sidebar-collapsed', s.sidebarCollapsed ? 'true' : 'false');
}

let initialized = false;

// initSettings wires appearance side-effects. Idempotent; call once at boot.
export function initSettings() {
  if (initialized) return;
  initialized = true;

  applyAppearance();
  settingsStore.subscribe(() => applyAppearance());

  window
    .matchMedia('(prefers-color-scheme: dark)')
    .addEventListener('change', () => {
      if (settingsStore.getState().theme === 'System') applyAppearance();
    });
}

// ── Accessors / mutators (stable public API) ────────────
export function isEnabled(label: string): boolean {
  return !!settings.toggles[label];
}
export function setToggle(label: string, value?: boolean) {
  settingsStore.getState().setToggle(label, value);
}
export function setTheme(t: Theme) {
  settingsStore.getState().setTheme(t);
}
export function setSidebarPosition(p: SidebarPosition) {
  settingsStore.getState().setSidebarPosition(p);
}
export function toggleSidebarCollapsed() {
  settingsStore.getState().toggleSidebarCollapsed();
}
export function cycleTheme() {
  settingsStore.getState().cycleTheme();
}
export function resetSettings() {
  settingsStore.getState().reset();
}
