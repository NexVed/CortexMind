// The daemon passes its credential through the URL fragment, which is never sent
// in HTTP requests or referrers. Keep it in this tab's session only.
const STORAGE_KEY = 'cortex.local-api-token';
let token = '';
const fragment = new URLSearchParams(window.location.hash.slice(1));
token = fragment.get('local_auth') || '';
if (fragment.has('local_auth')) window.history.replaceState(null, '', window.location.pathname + window.location.search);
try {
  token = token || window.sessionStorage.getItem(STORAGE_KEY) || '';
  if (token) window.sessionStorage.setItem(STORAGE_KEY, token);
} catch { /* Private browsing may disable session storage; the in-memory token still works. */ }

export const API_BASE = import.meta.env.VITE_API_URL || '';
export const getLocalToken = () => token;

export function localFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  if (token) headers.set('Authorization', `Bearer ${token}`);
  if (!headers.has('Content-Type') && init.method && init.method !== 'GET') headers.set('Content-Type', 'application/json');
  return fetch(path, { ...init, headers, redirect: 'error' });
}
