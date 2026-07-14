import { Accessor, createEffect, createSignal } from 'solid-js';

const PREFIX = 'cortex.page.';

function readValue<T>(key: string, fallback: T): T {
  if (typeof window === 'undefined') return fallback;
  try {
    const raw = window.localStorage.getItem(`${PREFIX}${key}`);
    return raw === null ? fallback : JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

/** Keeps lightweight page controls stable when routes unmount and mount again. */
export function createPersistedSignal<T>(key: string, fallback: T): [Accessor<T>, (value: T) => void] {
  const [value, setValue] = createSignal(readValue(key, fallback));
  createEffect(() => {
    if (typeof window === 'undefined') return;
    try {
      window.localStorage.setItem(`${PREFIX}${key}`, JSON.stringify(value()));
    } catch {
      // Storage can be unavailable in privacy-restricted browser contexts.
    }
  });
  return [value, (next: T) => setValue(() => next)] as const;
}
