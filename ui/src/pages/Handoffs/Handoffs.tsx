import { Component, For, createSignal, Show } from 'solid-js';
import {
  ArrowRightLeft,
  Plus,
  Search,
  Filter,
  Bot,
  FileCode2,
  Clock,
  MoreVertical,
  ChevronRight,
} from 'lucide-solid';
import { useHandoffs, useCreateHandoff } from '../../api/queries';
import { Modal } from '../../components/Modal/Modal';
import './Handoffs.css';

export const HandoffsPage: Component = () => {
  const [searchQuery, setSearchQuery] = createSignal('');
  const handoffsQuery = useHandoffs();
  const handoffs = () => handoffsQuery.data;
  const createHandoffM = useCreateHandoff();

  // ── Create handoff modal ──
  const [showCreate, setShowCreate] = createSignal(false);
  const [saving, setSaving] = createSignal(false);
  const [formErr, setFormErr] = createSignal('');
  const [fTitle, setFTitle] = createSignal('');
  const [fFrom, setFFrom] = createSignal('');
  const [fTo, setFTo] = createSignal('');
  const [fContext, setFContext] = createSignal('');

  const openCreate = () => {
    setFTitle('');
    setFFrom('');
    setFTo('');
    setFContext('');
    setFormErr('');
    setShowCreate(true);
  };

  const submitCreate = async () => {
    if (!fTitle().trim() || !fFrom().trim() || !fTo().trim()) {
      setFormErr('Title, from-agent and to-agent are required.');
      return;
    }
    setSaving(true);
    setFormErr('');
    try {
      await createHandoffM.mutateAsync({
        title: fTitle().trim(),
        from_agent: fFrom().trim(),
        to_agent: fTo().trim(),
        context: fContext().trim(),
      });
      setShowCreate(false);
    } catch (err: any) {
      setFormErr(err?.message || 'Failed to create handoff');
    } finally {
      setSaving(false);
    }
  };

  const filteredHandoffs = () => {
    const h = handoffs();
    if (!h) return [];
    if (!searchQuery()) return h;
    const q = searchQuery().toLowerCase();
    return h.filter(
      (item) =>
        item.title.toLowerCase().includes(q) ||
        item.from_agent.toLowerCase().includes(q) ||
        item.to_agent.toLowerCase().includes(q)
    );
  };

  return (
    <div class="handoffs-page">
      <div class="page-header">
        <div class="page-title-row">
          <div class="page-title-icon orange">
            <ArrowRightLeft size={24} />
          </div>
          <div>
            <h1 class="page-title">AI Handoffs</h1>
            <p class="page-subtitle">Transfer context seamlessly between agents</p>
          </div>
        </div>
        <div class="page-actions">
          <button class="btn primary" onClick={openCreate}>
            <Plus size={16} />
            New Handoff
          </button>
        </div>
      </div>

      <div class="handoffs-toolbar">
        <div class="search-box">
          <Search size={16} class="search-icon" />
          <input
            type="text"
            placeholder="Search handoffs..."
            value={searchQuery()}
            onInput={(e) => setSearchQuery(e.currentTarget.value)}
          />
        </div>
        <div class="handoffs-filters">
          <button class="filter-btn active">All</button>
          <button class="filter-btn">Active</button>
          <button class="filter-btn">Consumed</button>
          <button class="icon-btn">
            <Filter size={16} />
          </button>
        </div>
      </div>

      <Show when={handoffsQuery.isLoading}>
        <div class="loading-state">Loading handoffs...</div>
      </Show>

      <Show when={!handoffsQuery.isLoading && filteredHandoffs().length === 0}>
        <div class="empty-state">
          <div class="empty-icon"><ArrowRightLeft size={32} /></div>
          <h3>No handoffs found</h3>
          <p>Create a handoff to share context between your AI agents.</p>
        </div>
      </Show>

      <div class="handoffs-grid">
        <For each={filteredHandoffs()}>
          {(item) => (
            <div class="handoff-card">
              <div class="handoff-card-header">
                <div class="handoff-agents">
                  <div class="agent-badge from">
                    <Bot size={14} />
                    {item.from_agent}
                  </div>
                  <ChevronRight size={14} class="agent-arrow" />
                  <div class="agent-badge to">
                    <Bot size={14} />
                    {item.to_agent}
                  </div>
                </div>
                <button class="icon-btn">
                  <MoreVertical size={16} />
                </button>
              </div>

              <h3 class="handoff-title">{item.title}</h3>
              <div class="handoff-project">{item.project || 'Global context'}</div>

              <div class="handoff-meta">
                <div class="meta-item">
                  <FileCode2 size={14} />
                  <span>{item.included_files?.length || 0} files attached</span>
                </div>
                <div class="meta-item">
                  <Bot size={14} />
                  <span>~{item.token_count || 0} tokens</span>
                </div>
                <div class="meta-item">
                  <Clock size={14} />
                  <span>{new Date(item.updated).toLocaleDateString()}</span>
                </div>
              </div>

              <div class="handoff-footer">
                <span class={`handoff-status ${item.status}`}>
                  <span class="status-dot" />
                  {item.status.charAt(0).toUpperCase() + item.status.slice(1)}
                </span>
                <button class="btn secondary small">Resume</button>
              </div>
            </div>
          )}
        </For>
      </div>

      <Modal open={showCreate()} title="New Handoff" onClose={() => setShowCreate(false)}>
        <div class="cx-field">
          <label>Title *</label>
          <input
            value={fTitle()}
            onInput={(e) => setFTitle(e.currentTarget.value)}
            placeholder="Context handoff summary"
            autofocus
          />
        </div>
        <div class="cx-field">
          <label>From agent *</label>
          <input value={fFrom()} onInput={(e) => setFFrom(e.currentTarget.value)} placeholder="claude" />
        </div>
        <div class="cx-field">
          <label>To agent *</label>
          <input value={fTo()} onInput={(e) => setFTo(e.currentTarget.value)} placeholder="cursor" />
        </div>
        <div class="cx-field">
          <label>Context</label>
          <textarea
            value={fContext()}
            onInput={(e) => setFContext(e.currentTarget.value)}
            placeholder="What the next agent needs to know…"
          />
        </div>
        <Show when={formErr()}>
          <div class="cx-modal-error">{formErr()}</div>
        </Show>
        <div class="cx-modal-actions">
          <button class="btn secondary" onClick={() => setShowCreate(false)}>Cancel</button>
          <button class="btn primary" onClick={submitCreate} disabled={saving()}>
            {saving() ? 'Creating…' : 'Create Handoff'}
          </button>
        </div>
      </Modal>
    </div>
  );
};
