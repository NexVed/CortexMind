// Helpers for running inside the native Wails desktop shell.
//
// The UI is served from the local daemon's origin, so Wails does NOT inject its
// full JS runtime — but when the window is created with AllowSimpleEventEmit it
// does inject a minimal `window._wails.invoke` bridge. We use that bridge to
// drive the native (frameless) window from the custom titlebar.
//
// IMPORTANT: With a remote URL the bridge is injected *asynchronously* — it may
// not exist when the first SolidJS render runs. We therefore expose a reactive
// signal that re-checks after a short delay so the titlebar appears once the
// bridge is ready.

import { createSignal } from 'solid-js';

interface WailsBridge {
  invoke?: (name: string) => void;
}

function bridge(): WailsBridge | undefined {
  return (window as unknown as { _wails?: WailsBridge })._wails;
}

const DESKTOP_FLAG = 'cortex.desktop';
const PLATFORM_FLAG = 'cortex.platform';

function persistDesktopFromURL(): void {
  if (typeof window === 'undefined') return;
  const params = new URLSearchParams(window.location.search);
  if (params.has('desktop')) sessionStorage.setItem(DESKTOP_FLAG, '1');
  const platform = params.get('platform');
  if (platform === 'windows' || platform === 'darwin' || platform === 'linux') {
    sessionStorage.setItem(PLATFORM_FLAG, platform);
    sessionStorage.setItem(DESKTOP_FLAG, '1');
  }
}

function detectWails(): boolean {
  persistDesktopFromURL();
  return sessionStorage.getItem(DESKTOP_FLAG) === '1'
    || new URLSearchParams(window.location.search).has('desktop')
    || typeof bridge()?.invoke === 'function'
    || /wails/i.test(navigator.userAgent);
}

// Reactive signal — starts with an immediate check, then re-checks a few times
// over the first second to catch late bridge injection.
const [_isDesktop, _setDesktop] = createSignal(detectWails());

// Re-check at 100ms, 300ms, 600ms and 1s after load to catch async injection.
if (!_isDesktop()) {
  const retries = [100, 300, 600, 1000];
  for (const ms of retries) {
    setTimeout(() => {
      if (!_isDesktop() && detectWails()) {
        _setDesktop(true);
      }
    }, ms);
  }
}

// isWailsDesktop is a reactive accessor — SolidJS will re-evaluate any Show/
// Switch that depends on it when the signal flips to true.
export function isWailsDesktop(): boolean {
  return _isDesktop();
}

export type DesktopPlatform = 'windows' | 'darwin' | 'linux' | 'web';

// The native shell supplies its build target explicitly. User-agent detection
// remains only as a useful fallback for development builds.
export function desktopPlatform(): DesktopPlatform {
  const fromQuery = new URLSearchParams(window.location.search).get('platform');
  const platform = fromQuery || sessionStorage.getItem(PLATFORM_FLAG);
  if (platform === 'windows' || platform === 'darwin' || platform === 'linux') return platform;
  if (!isWailsDesktop()) return 'web';
  if (/mac/i.test(navigator.platform)) return 'darwin';
  if (/linux/i.test(navigator.platform)) return 'linux';
  return 'windows';
}

if (typeof document !== 'undefined') {
  document.documentElement.dataset.desktopPlatform = desktopPlatform();
}

// emitWindowControl fires a bare Wails event that the Go side listens for to
// minimise / maximise / close the native window. No-op in a browser.
export function emitWindowControl(name: 'wnd:minimise' | 'wnd:toggle-maximise' | 'wnd:toggle-fullscreen' | 'wnd:close'): void {
  if (name === 'wnd:toggle-maximise' && desktopPlatform() === 'darwin') {
    name = 'wnd:toggle-fullscreen';
  }
  bridge()?.invoke?.(`wails:event:emit:${name}`);
}

export type WindowResizeEdge =
  | 'n-resize'
  | 'ne-resize'
  | 'e-resize'
  | 'se-resize'
  | 's-resize'
  | 'sw-resize'
  | 'w-resize'
  | 'nw-resize';

// Ask Wails to begin a native OS resize gesture from the selected edge/corner.
// This is only available in the desktop shell, so it is intentionally a no-op
// when the UI is opened in a browser.
export function startWindowResize(edge: WindowResizeEdge): void {
  bridge()?.invoke?.(`wails:resize:${edge}`);
}
