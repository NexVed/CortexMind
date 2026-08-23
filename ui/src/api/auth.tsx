import { Component, JSX, createContext, createSignal, useContext, onMount } from 'solid-js';
import { isWailsDesktop } from './desktop';

export interface CortexUser { id: string; email: string; displayName: string; githubUsername: string; githubAvatarUrl: string; githubId: string; provider: string; offline: boolean; }

interface AuthContextValue {
  user: () => CortexUser | null;
  token: () => string;
  isAuthenticated: () => boolean;
  isLoading: () => boolean;
  error: () => string;
  deviceCode: () => string;
  verificationURL: () => string;
  loginWithGitHub: () => Promise<void>;
  continueOffline: (displayName: string) => Promise<void>;
  logout: () => Promise<void>;
}

interface DeviceStartResponse {
  url: string;
  user_code: string;
  expires_in: number;
}

const AuthContext = createContext<AuthContextValue>();
const API = import.meta.env.VITE_API_URL || '';

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within an AuthProvider');
  return ctx;
}

function asUser(value: any): CortexUser {
  return {
    id: value.id || value.ID || '',
    email: value.email || value.Email || '',
    displayName: value.display_name || value.displayName || value.DisplayName || value.username || value.Username || 'Offline',
    githubUsername: value.username || value.Username || '',
    githubAvatarUrl: value.avatar_url || value.avatarUrl || value.AvatarURL || '',
    githubId: value.github_id || value.githubId || value.GitHubID || '',
    provider: value.provider || value.Provider || '',
    offline: !!(value.offline ?? value.Offline),
  };
}

async function request(path: string, init?: RequestInit): Promise<any> {
  const response = await fetch(API + path, { headers: { 'Content-Type': 'application/json' }, ...init });
  const body = await response.json();
  if (!response.ok) throw new Error(body.error || 'Request failed');
  return body;
}

export const AuthProvider: Component<{ children: JSX.Element }> = (props) => {
  const [user, setUser] = createSignal<CortexUser | null>(null);
  const [isLoading, setLoading] = createSignal(true);
  const [error, setError] = createSignal('');
  const [deviceCode, setDeviceCode] = createSignal('');
  const [verificationURL, setVerificationURL] = createSignal('');

  const restore = async () => {
    const body = await request('/api/auth/session');
    if (body.auth_error) throw new Error(body.auth_error);
    setUser(body.user ? asUser(body.user) : null);
  };

  onMount(() => {
    void restore().catch((err: Error) => setError(err.message)).finally(() => setLoading(false));
  });

  const loginWithGitHub = async () => {
    setError('');
    setDeviceCode('');
    setVerificationURL('');
    setLoading(true);
    try {
      const endpoint = isWailsDesktop()
        ? '/api/auth/github/start?open_browser=1'
        : '/api/auth/github/start';
      const start = await request(endpoint, { method: 'POST' }) as DeviceStartResponse;
      setDeviceCode(start.user_code);
      setVerificationURL(start.url);
      // Web builds retain the normal browser behaviour. The native desktop
      // shell asks the daemon to use the user's default browser instead.
      if (!isWailsDesktop()) window.open(start.url, '_blank', 'noopener,noreferrer');

      const deadline = Date.now() + Math.max(60, start.expires_in || 900) * 1000;
      while (!user() && Date.now() < deadline) {
        await new Promise((resolve) => window.setTimeout(resolve, 1000));
        await restore();
      }
      if (!user()) throw new Error('GitHub sign-in timed out. Please try again.');
      setDeviceCode('');
      setVerificationURL('');
    } catch (err: any) {
      setError(err.message || 'Unable to sign in with GitHub.');
      throw err;
    } finally {
      setLoading(false);
    }
  };

  const continueOffline = async (displayName: string) => {
    setError('');
    setLoading(true);
    try {
      const body = await request('/api/auth/offline', { method: 'POST', body: JSON.stringify({ display_name: displayName }) });
      setUser(asUser(body.user));
    } catch (err: any) {
      setError(err.message);
      throw err;
    } finally {
      setLoading(false);
    }
  };

  const logout = async () => {
    await request('/api/auth/logout', { method: 'POST' });
    setUser(null);
  };

  return <AuthContext.Provider value={{
    user,
    token: () => '',
    isAuthenticated: () => user() !== null,
    isLoading,
    error,
    deviceCode,
    verificationURL,
    loginWithGitHub,
    continueOffline,
    logout,
  }}>{props.children}</AuthContext.Provider>;
};
