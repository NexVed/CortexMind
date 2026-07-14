import {
  Component,
  JSX,
  createContext,
  createSignal,
  useContext,
  onMount,
} from 'solid-js';
import { pb } from './pb';
import { RecordModel } from 'pocketbase';
import { syncGitHubRepos } from './client';
import { queryClient } from './queryClient';
import { qk } from './queries';

// ── Types ──────────────────────────────────────────────

export interface CortexUser {
  id: string;
  email: string;
  displayName: string;
  githubUsername: string;
  githubAvatarUrl: string;
}

interface AuthContextValue {
  user: () => CortexUser | null;
  token: () => string;
  isAuthenticated: () => boolean;
  isLoading: () => boolean;
  error: () => string;
  loginWithGitHub: () => Promise<void>;
  loginWithPassword: (email: string, password: string) => Promise<void>;
  registerWithPassword: (email: string, password: string, displayName?: string) => Promise<void>;
  logout: () => void;
}

// ── Context ────────────────────────────────────────────

const AuthContext = createContext<AuthContextValue>();

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within an AuthProvider');
  return ctx;
}

// ── Helpers ────────────────────────────────────────────

function recordToUser(record: RecordModel): CortexUser {
  return {
    id: record.id,
    email: record.email ?? (record as any).email ?? '',
    displayName: record['display_name'] ?? record['name'] ?? '',
    githubUsername: record['github_username'] ?? '',
    githubAvatarUrl: record['github_avatar_url'] ?? record['avatar'] ?? '',
  };
}

// ── Provider ───────────────────────────────────────────

export const AuthProvider: Component<{ children: JSX.Element }> = (props) => {
  const [user, setUser] = createSignal<CortexUser | null>(null);
  const [isLoading, setIsLoading] = createSignal(true);
  const [error, setError] = createSignal('');

  const token = () => pb.authStore.token ?? '';
  const isAuthenticated = () => pb.authStore.isValid && user() !== null;

  // Restore session from localStorage on mount
  onMount(() => {
    if (pb.authStore.isValid && pb.authStore.record) {
      setUser(recordToUser(pb.authStore.record as RecordModel));
    }
    setIsLoading(false);

    // Listen for auth store changes (e.g. token refresh, logout)
    pb.authStore.onChange((_token, record) => {
      if (record) {
        setUser(recordToUser(record as RecordModel));
      } else {
        setUser(null);
      }
    });
  });

  const loginWithGitHub = async () => {
    setError('');
    setIsLoading(true);
    try {
      // Use the `urlCallback` flow instead of PocketBase's default popup.
      // The default opens a `window.open` popup, which the native desktop
      // webview (Wails/WebView2) cannot host and which crashes the app. With
      // urlCallback we open GitHub in the *system browser* — Wails routes
      // target=_blank to the OS browser — and the SDK completes the login over
      // its realtime channel once the OAuth redirect hits the local daemon.
      // This path also works unchanged in a normal browser.
      const result = await pb.collection('users').authWithOAuth2({
        provider: 'github',
        scopes: ['repo', 'read:user', 'user:email'],
        urlCallback: (url) => {
          window.open(url, '_blank', 'noopener,noreferrer');
        },
      });
      setUser(recordToUser(result.record as RecordModel));

      // Import the user's GitHub repositories as projects. The daemon already
      // stored the GitHub access token on the user record during OAuth, so we
      // trigger the server-side sync (fully paginated, no dependency on the
      // OAuth `meta` reaching the client) and refresh the projects list when it
      // finishes. Fire-and-forget so it doesn't block the UI.
      syncGitHubRepos()
        .then(() => queryClient.invalidateQueries({ queryKey: qk.projects }))
        .catch((err) => console.error('Failed to sync GitHub repos:', err));
    } catch (err: any) {
      const msg =
        err?.message ?? err?.data?.message ?? 'GitHub login failed. Please try again.';
      setError(msg);
      console.error('GitHub OAuth error:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const logout = () => {
    pb.authStore.clear();
    setUser(null);
  };

  // Local (email/password) sign-in — for users who don't want to use GitHub.
  const loginWithPassword = async (email: string, password: string) => {
    setError('');
    setIsLoading(true);
    try {
      const res = await pb.collection('users').authWithPassword(email, password);
      setUser(recordToUser(res.record as RecordModel));
    } catch (err: any) {
      setError(err?.data?.message ?? err?.message ?? 'Invalid email or password.');
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  // Local account creation, then immediate sign-in. No GitHub required.
  const registerWithPassword = async (
    email: string,
    password: string,
    displayName?: string,
  ) => {
    setError('');
    setIsLoading(true);
    try {
      await pb.collection('users').create({
        email,
        password,
        passwordConfirm: password,
        display_name: displayName || email.split('@')[0],
      });
      const res = await pb.collection('users').authWithPassword(email, password);
      setUser(recordToUser(res.record as RecordModel));
    } catch (err: any) {
      setError(err?.data?.message ?? err?.message ?? 'Could not create account.');
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  const value: AuthContextValue = {
    user,
    token,
    isAuthenticated,
    isLoading,
    error,
    loginWithGitHub,
    loginWithPassword,
    registerWithPassword,
    logout,
  };

  return (
    <AuthContext.Provider value={value}>
      {props.children}
    </AuthContext.Provider>
  );
};
