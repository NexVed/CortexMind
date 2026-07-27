import { Component, For } from 'solid-js';
import { startWindowResize, WindowResizeEdge } from '../../api/desktop';
import './WindowResizeHandles.css';

const resizeEdges: WindowResizeEdge[] = [
  'n-resize',
  'ne-resize',
  'e-resize',
  'se-resize',
  's-resize',
  'sw-resize',
  'w-resize',
  'nw-resize',
];

// A frameless Wails window has no system frame to grab. These transparent
// targets forward a pointer-down from each outer edge/corner to Wails, which
// starts the normal OS resize drag without adding any visual chrome.
export const WindowResizeHandles: Component = () => (
  <div class="window-resize-handles" aria-hidden="true">
    <For each={resizeEdges}>
      {(edge) => (
        <div
          class={`window-resize-handle window-resize-handle--${edge}`}
          onMouseDown={(event) => {
            event.preventDefault();
            startWindowResize(edge);
          }}
        />
      )}
    </For>
  </div>
);