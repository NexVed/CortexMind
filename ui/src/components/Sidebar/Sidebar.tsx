import { Component, createSignal, For, createResource, Show } from 'solid-js';
import { useNavigate, useLocation } from '@solidjs/router';
import {
  LayoutDashboard,
  FolderKanban,
  BookOpen,
  Bot,
  Sparkles,
  Search,
  Users,
  Settings,
  Puzzle,
  ChevronDown,
} from 'lucide-solid';
import { useAuth } from '../../api/auth';
import { getDaemonStatus } from '../../api/client';
import './Sidebar.css';

const navItems = [
  { id: 'workspace', path: '/', label: 'Workspace', icon: LayoutDashboard },
  { id: 'projects', path: '/projects', label: 'Projects', icon: FolderKanban },
  { id: 'vaults', path: '/vaults', label: 'Context', icon: BookOpen },
  { id: 'agents', path: '/ai-context', label: 'Agents', icon: Bot },
  { id: 'knowledge', path: '/graph', label: 'Knowledge', icon: Sparkles, badge: 'BETA' },
  { id: 'divider-1', path: '', label: '', icon: null },
  { id: 'team', path: '/handoffs', label: 'Team', icon: Users },
  { id: 'settings', path: '/settings', label: 'Settings', icon: Settings },
  { id: 'integrations', path: '/mcp-server', label: 'Integrations', icon: Puzzle },
];

export const Sidebar: Component = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuth();
  const [daemonStatus] = createResource(() => getDaemonStatus());
  const [showUserMenu, setShowUserMenu] = createSignal(false);

  const isActive = (path: string) => {
    if (path === '/') return location.pathname === '/';
    return location.pathname.startsWith(path);
  };

  const displayName = () => {
    const u = user();
    return u?.displayName || u?.githubUsername || 'User';
  };

  const workspaceName = () => {
    const u = user();
    return u?.displayName ? `${u.displayName}'s workspace` : 'Team workspace';
  };

  return (
    <nav class="sidebar">
      {/* Brand */}
      <div class="sidebar-brand">
        <div class="sidebar-logo">C</div>
        <span class="sidebar-brand-name">CORTEX</span>
      </div>

      {/* Navigation */}
      <div class="sidebar-nav">
        <For each={navItems}>
          {(item) => {
            if (item.id.startsWith('divider')) {
              return <div class="sidebar-divider" />;
            }
            const Icon = item.icon!;
            return (
              <button
                class={`sidebar-nav-item ${isActive(item.path) ? 'active' : ''}`}
                onClick={() => navigate(item.path)}
                id={`nav-${item.id}`}
              >
                <Icon size={18} />
                <span class="sidebar-nav-label">{item.label}</span>
                {item.badge && <span class="sidebar-nav-badge">{item.badge}</span>}
              </button>
            );
          }}
        </For>
      </div>

      {/* Workspace info */}
      <div class="sidebar-bottom">
        <div
          class="sidebar-workspace"
          onClick={() => setShowUserMenu(!showUserMenu())}
        >
          <Show
            when={user()?.githubAvatarUrl}
            fallback={<div class="sidebar-workspace-avatar">{displayName().charAt(0)}</div>}
          >
            <img
              class="sidebar-workspace-avatar-img"
              src={user()!.githubAvatarUrl}
              alt={displayName()}
            />
          </Show>
          <div class="sidebar-workspace-info">
            <span class="sidebar-workspace-name">{displayName()}</span>
            <span class="sidebar-workspace-sub">{workspaceName()}</span>
          </div>
        </div>
      </div>
    </nav>
  );
};
