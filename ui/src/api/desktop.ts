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

function detectWails(): boolean {
  return new URLSearchParams(window.location.search).has('desktop')
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

// emitWindowControl fires a bare Wails event that the Go side listens for to
// minimise / maximise / close the native window. No-op in a browser.
export function emitWindowControl(name: 'wnd:minimise' | 'wnd:toggle-maximise' | 'wnd:close'): void {
  bridge()?.invoke?.(`wails:event:emit:${name}`);
}
