import { Component, For, createSignal, Show } from 'solid-js';
import {
  Archive,
  Plus,
  Search,
  Book,
  FileText,
  Map,
  Shield,
  MoreVertical,
  Calendar,
  Share2,
} from 'lucide-solid';
import { useVaultEntries, useCreateVaultEntry } from '../../api/queries';
import { Modal } from '../../components/Modal/Modal';
import './Vaults.css';

const categories = [
  { id: 'all', label: 'All Entries', icon: Book },
  { id: 'architecture', label: 'Architecture', icon: Map },
  { id: 'decision', label: 'Decisions', icon: Shield },
  { id: 'task', label: 'Tasks context', icon: FileText },
];

export const VaultsPage: Component = () => {
  const [activeCategory, setActiveCategory] = createSignal('all');
  const [searchQuery, setSearchQuery] = createSignal('');

  // We re-fetch when activeCategory changes if we pass it as source,
  // but for simple filtering it's better to fetch all and filter client side.
  const entriesQuery = useVaultEntries();
  const entries = () => entriesQuery.data;
  const createEntryM = useCreateVaultEntry();

  // ── Create entry modal ──
  const [showCreate, setShowCreate] = createSignal(false);
  const [saving, setSaving] = createSignal(false);
  const [formErr, setFormErr] = createSignal('');
  const [fTitle, setFTitle] = createSignal('');
  const [fCategory, setFCategory] = createSignal('architecture');
  const [fContent, setFContent] = createSignal('');

  const openCreate = () => {
    setFTitle('');
    setFCategory(activeCategory() !== 'all' ? activeCategory() : 'architecture');
    setFContent('');
    setFormErr('');
    setShowCreate(true);
  };

  const submitCreate = async () => {
    if (!fTitle().trim()) {
      setFormErr('Title is required.');
      return;
    }
    setSaving(true);
    setFormErr('');
    try {
      await createEntryM.mutateAsync({
        title: fTitle().trim(),
        category: fCategory(),
        content: fContent().trim(),
      });
      setShowCreate(false);
    } catch (err: any) {
      setFormErr(err?.message || 'Failed to create entry');
    } finally {
      setSaving(false);
    }
  };

  const filteredEntries = () => {
    let result = entries() || [];
    
    if (activeCategory() !== 'all') {
      result = result.filter(e => e.category === activeCategory());
    }
    
    if (searchQuery()) {
      const q = searchQuery().toLowerCase();
      result = result.filter(e => 
        e.title.toLowerCase().includes(q) || 
        (e.content && e.content.toLowerCase().includes(q))
      );
    }
    
    return result;
  };

  const getCategoryIcon = (catId: string) => {
    const cat = categories.find((c) => c.id === catId);
    return cat ? cat.icon : FileText;
  };

  return (
    <div class="vaults-page">
      <div class="page-header">
        <div class="page-title-row">
          <div class="page-title-icon green">
            <Archive size={24} />
          </div>
          <div>
            <h1 class="page-title">Knowledge Vault</h1>
            <p class="page-subtitle">Persistent memory and architecture context</p>
          </div>
        </div>
        <div class="page-actions">
          <button class="btn primary" onClick={openCreate}>
            <Plus size={16} />
            New Entry
          </button>
        </div>
      </div>

      <div class="vaults-layout">
        {/* Sidebar filters */}
        <aside class="vaults-sidebar">
          <div class="search-box">
            <Search size={16} class="search-icon" />
            <input
              type="text"
              placeholder="Search vault..."
              value={searchQuery()}
              onInput={(e) => setSearchQuery(e.currentTarget.value)}
            />
          </div>

          <div class="vaults-categories">
            <div class="category-header">Categories</div>
            <For each={categories}>
              {(cat) => {
                const Icon = cat.icon;
                return (
                  <button
                    class={`category-btn ${activeCategory() === cat.id ? 'active' : ''}`}
                    onClick={() => setActiveCategory(cat.id)}
                  >
                    <Icon size={16} />
                    <span>{cat.label}</span>
                  </button>
                );
              }}
            </For>
          </div>
        </aside>

        {/* Main Grid */}
        <div class="vaults-content">
          <Show when={entriesQuery.isLoading}>
            <div class="loading-state">Loading vault entries...</div>
          </Show>

          <Show when={!entriesQuery.isLoading && filteredEntries().length === 0}>
            <div class="empty-state">
              <div class="empty-icon"><Archive size={32} /></div>
              <h3>No entries found</h3>
              <p>Add knowledge to the vault to persist it across sessions.</p>
            </div>
          </Show>

          <div class="vaults-grid">
            <For each={filteredEntries()}>
              {(entry) => {
                const Icon = getCategoryIcon(entry.category);
                return (
                  <div class="vault-card">
                    <div class="vault-card-header">
                      <div class="vault-card-category">
                        <Icon size={14} />
                        <span>{entry.category.charAt(0).toUpperCase() + entry.category.slice(1)}</span>
                      </div>
                      <button class="icon-btn">
                        <MoreVertical size={16} />
                      </button>
                    </div>

                    <h3 class="vault-card-title">{entry.title}</h3>
                    
                    {/* Render a plain text preview of the HTML/markdown content */}
                    <p class="vault-card-preview">
                      {entry.content ? entry.content.replace(/<[^>]*>?/gm, '').substring(0, 120) + '...' : 'No content.'}
                    </p>

                    <Show when={entry.tags && entry.tags.length > 0}>
                      <div class="vault-card-tags">
                        <For each={entry.tags!.slice(0, 3)}>
                          {(tag) => <span class="vault-tag">{tag}</span>}
                        </For>
                        <Show when={entry.tags!.length > 3}>
                          <span class="vault-tag">+{entry.tags!.length - 3}</span>
                        </Show>
                      </div>
                    </Show>

                    <div class="vault-card-footer">
                      <div class="vault-card-meta">
                        <Calendar size={12} />
                        <span>{new Date(entry.updated).toLocaleDateString()}</span>
                      </div>
                      <Show when={entry.is_shared}>
                        <div class="vault-card-meta shared" title="Shared with team">
                          <Share2 size={12} />
                          <span>Shared</span>
                        </div>
                      </Show>
                    </div>
                  </div>
                );
              }}
            </For>
          </div>
        </div>
      </div>

      <Modal open={showCreate()} title="New Vault Entry" onClose={() => setShowCreate(false)}>
        <div class="cx-field">
          <label>Title *</label>
          <input
            value={fTitle()}
            onInput={(e) => setFTitle(e.currentTarget.value)}
            placeholder="e.g. Auth flow decision"
            autofocus
          />
        </div>
        <div class="cx-field">
          <label>Category</label>
          <select value={fCategory()} onChange={(e) => setFCategory(e.currentTarget.value)}>
            <option value="architecture">Architecture</option>
            <option value="decision">Decision</option>
            <option value="task">Task context</option>
            <option value="roadmap">Roadmap</option>
            <option value="memory">Memory</option>
          </select>
        </div>
        <div class="cx-field">
          <label>Content</label>
          <textarea
            value={fContent()}
            onInput={(e) => setFContent(e.currentTarget.value)}
            placeholder="Knowledge to persist across sessions…"
          />
        </div>
        <Show when={formErr()}>
          <div class="cx-modal-error">{formErr()}</div>
        </Show>
        <div class="cx-modal-actions">
          <button class="btn secondary" onClick={() => setShowCreate(false)}>Cancel</button>
          <button class="btn primary" onClick={submitCreate} disabled={saving()}>
            {saving() ? 'Saving…' : 'Create Entry'}
          </button>
        </div>
      </Modal>
    </div>
  );
};
