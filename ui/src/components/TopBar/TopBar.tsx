import { Component, Show, createSignal, onCleanup } from 'solid-js';
import {
  Search,
  Sun,
  Moon,
  Bell,
  Settings,
  ChevronDown,
  LogOut,
} from 'lucide-solid';
import { useAuth } from '../../api/auth';
import './TopBar.css';

export const TopBar: Component = () => {
  const { user, logout } = useAuth();
  const [dropdownOpen, setDropdownOpen] = createSignal(false);

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
        <button class="topbar-icon-btn" title="Notifications" id="notifications-btn">
          <Bell size={16} />
          <span class="topbar-notification-dot" />
        </button>

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
