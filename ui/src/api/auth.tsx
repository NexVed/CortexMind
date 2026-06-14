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
      const result = await pb.collection('users').authWithOAuth2({ 
        provider: 'github',
        scopes: ['repo', 'read:user', 'user:email']
      });
      setUser(recordToUser(result.record as RecordModel));
      
      if (result.meta?.accessToken) {
        // Run sync asynchronously so it doesn't block the UI
        syncGitHubRepos(result.meta.accessToken).catch(err => {
          console.error('Failed to sync GitHub repos:', err);
        });
      }
    } catch (err: any) {
      const msg =
        err?.message ?? err?.data?.message ?? 'GitHub login failed. Please try again.';
      setError(msg);
      console.error('GitHub OAuth error:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const syncGitHubRepos = async (token: string) => {
    const res = await fetch('https://api.github.com/user/repos?per_page=100&sort=updated', {
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: 'application/vnd.github.v3+json',
      },
    });
    if (!res.ok) throw new Error(`GitHub API error: ${res.statusText}`);
    const repos = await res.json();

    const existingProjects = await pb.collection('projects').getFullList();
    const existingUrls = new Set(existingProjects.map(p => p.github_url).filter(Boolean));

    const colors = ['#E8326E', '#3B82F6', '#22C55E', '#F59E0B', '#A855F7'];

    for (let i = 0; i < repos.length; i++) {
      const repo = repos[i];
      if (!existingUrls.has(repo.html_url)) {
        await pb.collection('projects').create({
          name: repo.name,
          description: repo.description || 'Synced from GitHub',
          github_url: repo.html_url,
          status: 'active',
          progress: 0,
          owner: pb.authStore.record?.id,
          icon_color: colors[i % colors.length],
        });
      }
    }
  };

  const logout = () => {
    pb.authStore.clear();
    setUser(null);
  };

  const value: AuthContextValue = {
    user,
    token,
    isAuthenticated,
    isLoading,
    error,
    loginWithGitHub,
    logout,
  };

  return (
    <AuthContext.Provider value={value}>
      {props.children}
    </AuthContext.Provider>
  );
};
