import { Component, Show, For, createSignal, onCleanup } from 'solid-js';
import {
  Search,
  Bell,
  ChevronDown,
  LogOut,
  Sparkles,
  Cpu,
  GitBranch,
  CheckCircle2,
  Inbox,
} from 'lucide-solid';
import { useAuth } from '../../api/auth';
import { type Notification } from '../../api/client';
import { useNotifications } from '../../api/queries';
import './TopBar.css';

export const TopBar: Component = () => {
  const { user, logout } = useAuth();
  const [dropdownOpen, setDropdownOpen] = createSignal(false);
  const [notificationsOpen, setNotificationsOpen] = createSignal(false);

  // Fetch notifications from backend activity_log
  const notificationsQuery = useNotifications(20);
  const backendNotifications = () => notificationsQuery.data;
  const refetch = () => notificationsQuery.refetch();
  const [readIds, setReadIds] = createSignal<Set<string>>(new Set());

  const notifications = () => {
    const items = backendNotifications() ?? [];
    const read = readIds();
    return items.map(n => ({
      ...n,
      unread: n.unread && !read.has(n.id),
    }));
  };

  const hasUnread = () => notifications().some(n => n.unread);

  const markAllRead = () => {
    const all = new Set(readIds());
    for (const n of (backendNotifications() ?? [])) all.add(n.id);
    setReadIds(all);
  };

  const markAsRead = (id: string) => {
    const next = new Set(readIds());
    next.add(id);
    setReadIds(next);
  };

  const toggleNotifications = (e: MouseEvent) => {
    e.stopPropagation();
    setNotificationsOpen(!notificationsOpen());
    setDropdownOpen(false);
    // Refresh on open
    if (!notificationsOpen()) refetch();
  };

  const displayName = () => {
    const u = user();
    return u?.displayName || u?.githubUsername || 'User';
  };

  const avatarUrl = () => user()?.githubAvatarUrl || '';

  const handleLogout = () => {
    logout();
    setDropdownOpen(false);
  };

  const toggleDropdown = () => setDropdownOpen(!dropdownOpen());

  // Close dropdown on outside click
  const handleWindowClick = (e: MouseEvent) => {
    const target = e.target as HTMLElement;
    if (!target.closest('.topbar-user-menu')) {
      setDropdownOpen(false);
    }
    if (!target.closest('.topbar-notifications-wrapper')) {
      setNotificationsOpen(false);
    }
  };
  
  window.addEventListener('click', handleWindowClick);
  onCleanup(() => window.removeEventListener('click', handleWindowClick));

  return (
    <header class="topbar">
      {/* Search */}
      <div class="topbar-search">
        <div class="topbar-search-input">
          <span class="topbar-search-icon">
            <Search size={15} />
          </span>
          <input
            type="text"
            placeholder="Search projects, files, memories, handoffs, decisions..."
            id="global-search"
          />
          <span class="topbar-search-kbd">⌘ K</span>
        </div>
      </div>

      {/* Right Actions */}
      <div class="topbar-actions">
        {/* Notifications */}
        <div class="topbar-notifications-wrapper" style={{ position: 'relative' }}>
          <button
            class="topbar-icon-btn"
            title="Notifications"
            id="notifications-btn"
            onClick={toggleNotifications}
          >
            <Bell size={16} />
            <Show when={hasUnread()}>
              <span class="topbar-notification-dot" />
            </Show>
          </button>

          <Show when={notificationsOpen()}>
            <div class="notifications-panel">
              <div class="notifications-header">
                <h3>Notifications</h3>
                <Show when={notifications().length > 0}>
                  <button class="mark-all-read-btn" onClick={markAllRead}>
                    Mark all as read
                  </button>
                </Show>
              </div>
              <div class="notifications-list">
                <Show when={notifications().length > 0} fallback={
                  <div class="notifications-empty">
                    <Inbox size={28} style={{ color: 'var(--text-muted)', opacity: '0.5' }} />
                    <span>No notifications yet</span>
                  </div>
                }>
                  <For each={notifications()}>
                    {(n) => (
                      <div
                        class={`notification-item ${n.unread ? 'unread' : ''}`}
                        onClick={() => markAsRead(n.id)}
                      >
                        <div class={`notification-category-icon ${n.category}`}>
                          <Show when={n.category === 'digest'}><Sparkles size={14} /></Show>
                          <Show when={n.category === 'scan'}><GitBranch size={14} /></Show>
                          <Show when={n.category === 'handoff'}><Cpu size={14} /></Show>
                          <Show when={n.category === 'task'}><CheckCircle2 size={14} /></Show>
                        </div>
                        <div class="notification-content">
                          <div class="notification-title-row">
                            <span class="notification-title">{n.title}</span>
                            <span class="notification-time">{n.time}</span>
                          </div>
                          <p class="notification-message">{n.message}</p>
                        </div>
                        <Show when={n.unread}>
                          <span class="notification-unread-dot" />
                        </Show>
                      </div>
                    )}
                  </For>
                </Show>
              </div>
            </div>
          </Show>
        </div>

        {/* User Avatar Menu */}
        <div class="topbar-user-menu" style={{ position: 'relative' }}>
          <div 
            class="topbar-user-trigger" 
            onClick={toggleDropdown}
            style={{ display: 'flex', 'align-items': 'center', gap: '8px', cursor: 'pointer' }}
          >
            <Show
              when={avatarUrl()}
              fallback={
                <div class="topbar-avatar" title={displayName()}>
                  {displayName().charAt(0).toUpperCase()}
                </div>
              }
            >
              <img
                class="topbar-avatar-img"
                src={avatarUrl()}
                alt={displayName()}
                title={displayName()}
              />
            </Show>
            <ChevronDown size={14} style={{ color: 'var(--text-muted)' }} />
          </div>

          <Show when={dropdownOpen()}>
            <div class="topbar-dropdown">
              <div class="topbar-dropdown-header">
                <strong>{displayName()}</strong>
                <span>{user()?.email || 'GitHub User'}</span>
              </div>
              <div class="topbar-dropdown-divider"></div>
              <button class="topbar-dropdown-item text-red" onClick={handleLogout}>
                <LogOut size={14} />
                Logout
              </button>
            </div>
          </Show>
        </div>
      </div>
    </header>
  );
};
