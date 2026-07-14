import { Component, onMount, onCleanup } from 'solid-js';
import { Minus, Square, X, Search, Command } from 'lucide-solid';
import { emitWindowControl } from '../../api/desktop';
import './WindowTitleBar.css';

// WindowTitleBar is the custom titlebar for the frameless native desktop
// window. It provides a draggable region (CSS app-region: drag, honoured on
// Windows via NonClientRegionSupport) and minimise / maximise / close controls
// that drive the native window through the Wails simple-emit bridge.
//
// Rendered only in the desktop shell (see App.tsx). It reserves space for
// itself by setting the global --titlebar-h custom property; every full-height
// layout subtracts that so nothing hides underneath.
export const WindowTitleBar: Component = () => {
  onMount(() => {
    // Increased height for a premium functional bar
    document.documentElement.style.setProperty('--titlebar-h', '48px');
  });
  onCleanup(() => {
    document.documentElement.style.removeProperty('--titlebar-h');
  });

  // Helper to trigger global search palette (Cmd/Ctrl + K)
  const triggerSearch = () => {
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', metaKey: true, ctrlKey: true }));
  };

  return (
    <header class="wtb" onDblClick={() => emitWindowControl('wnd:toggle-maximise')}>
      <div class="wtb-left">
        <div class="wtb-brand">
          <div class="wtb-logo-container">
            <img src="/logo.png" alt="" class="wtb-logo" />
            <div class="wtb-logo-glow"></div>
          </div>
          <span class="wtb-title">CortexMind</span>
        </div>
      </div>

      <div class="wtb-center">
        <button class="wtb-search-bar" onClick={triggerSearch} title="Global Search">
          <Search size={14} class="wtb-search-icon" />
          <span>Search or jump to...</span>
          <div class="wtb-search-shortcut">
            <Command size={11} /> K
          </div>
        </button>
      </div>

      <div class="wtb-right">
        <div class="wtb-controls">
          <button
            class="wtb-btn"
            title="Minimise"
            aria-label="Minimise"
            onClick={() => emitWindowControl('wnd:minimise')}
          >
            <Minus size={15} />
          </button>
          <button
            class="wtb-btn"
            title="Maximise"
            aria-label="Maximise"
            onClick={() => emitWindowControl('wnd:toggle-maximise')}
          >
            <Square size={12} />
          </button>
          <button
            class="wtb-btn wtb-btn--close"
            title="Close"
            aria-label="Close"
            onClick={() => emitWindowControl('wnd:close')}
          >
            <X size={15} />
          </button>
        </div>
      </div>
    </header>
  );
};
