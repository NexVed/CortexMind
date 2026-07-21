import { Component, For, Show } from 'solid-js';
import { useNavigate, useLocation } from '@solidjs/router';
import { LayoutDashboard, FolderKanban, BookOpen, Bot, Settings, Puzzle, Layers, Network, Brain, Github, LogOut } from 'lucide-solid';
import { useAuth } from '../../api/auth';
import './Sidebar.css';

const navItems: { id: string; path: string; label: string; icon: any; badge?: string }[] = [
  { id: 'workspace', path: '/', label: 'Workspace', icon: LayoutDashboard },
  { id: 'projects', path: '/projects', label: 'Projects', icon: FolderKanban },
  { id: 'vaults', path: '/vaults', label: 'Context', icon: BookOpen },
  { id: 'agents', path: '/ai-context', label: 'Agents', icon: Bot },
  { id: 'digests', path: '/digests', label: 'Digests', icon: Layers },
  { id: 'codegraph', path: '/code-graph', label: 'Code Graph', icon: Network },
  { id: 'agentmemory', path: '/agent-memory', label: 'Agent Memory', icon: Brain },
  { id: 'settings', path: '/settings', label: 'Settings', icon: Settings },
  { id: 'integrations', path: '/mcp-server', label: 'Integrations', icon: Puzzle },
];

export const Sidebar: Component = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuth();
  const displayName = () => user()?.displayName || user()?.githubUsername || 'User';
  const accountName = () => user()?.offline ? 'Offline workspace' : `@${user()?.githubUsername || 'GitHub connected'}`;
  const isActive = (path: string) => path === '/' ? location.pathname === '/' : location.pathname.startsWith(path);

  return <nav class="sidebar"><div class="sidebar-brand"><img src="/logowithname.png" alt="CortexMind" class="sidebar-logo-img" /></div><div class="sidebar-nav"><For each={navItems}>{(item) => { const Icon = item.icon; return <button class={`sidebar-nav-item ${isActive(item.path) ? 'active' : ''}`} onClick={() => navigate(item.path)} id={`nav-${item.id}`}><Icon size={18} /><span class="sidebar-nav-label">{item.label}</span></button>; }}</For></div><div class="sidebar-bottom"><div class="sidebar-account"><div class="sidebar-account-heading"><Github size={13} /> {user()?.offline ? 'Local profile' : 'GitHub account'}</div><div class="sidebar-account-profile"><Show when={user()?.githubAvatarUrl} fallback={<div class="sidebar-account-avatar">{displayName().charAt(0).toUpperCase()}</div>}><img class="sidebar-account-avatar" src={user()!.githubAvatarUrl} alt={displayName()} /></Show><div><strong>{displayName()}</strong><span>{accountName()}</span></div></div><Show when={!user()?.offline && user()?.githubId}><div class="sidebar-account-id">GitHub ID {user()!.githubId}</div></Show><button class="sidebar-account-logout" onClick={() => void logout()}><LogOut size={14} /> Sign out</button></div></div></nav>;
};