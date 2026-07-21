import { Component, Show, For, createSignal } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { Search, Bell, Minus, Square, X } from 'lucide-solid';
import { emitWindowControl, isWailsDesktop } from '../../api/desktop';
import { searchAll, type SearchResult } from '../../api/client';
import './TopBar.css';

export const TopBar: Component = () => {
  const navigate = useNavigate();
  const [query, setQuery] = createSignal('');
  const [results, setResults] = createSignal<SearchResult[]>([]);
  const [open, setOpen] = createSignal(false);
  const [searching, setSearching] = createSignal(false);
  let searchTimer: number | undefined;
  const runSearch = (value: string) => { setQuery(value); window.clearTimeout(searchTimer); if (!value.trim()) { setResults([]); setOpen(false); return; } setOpen(true); searchTimer = window.setTimeout(async () => { setSearching(true); try { setResults(await searchAll(value)); } finally { setSearching(false); } }, 180); };
  const resultRoute = (result: SearchResult) => result.collection === 'projects' || result.collection === 'file_index' ? `/repository/${result.project_id || result.id}` : result.collection === 'tasks' ? '/tasks' : result.collection === 'handoffs' ? '/handoffs' : result.collection === 'agent_memories' ? '/agent-memory' : result.collection === 'session_digests' ? '/digests' : '/vaults';
  const openResult = (result: SearchResult) => { navigate(resultRoute(result)); setQuery(''); setOpen(false); };
  return <header class="topbar"><div class="topbar-search"><div class="topbar-search-input"><span class="topbar-search-icon"><Search size={15} /></span><input type="text" placeholder="Search projects, files, memories, handoffs, decisions..." id="global-search" value={query()} onInput={(event) => runSearch(event.currentTarget.value)} onFocus={() => setOpen(!!query().trim())} /><span class="topbar-search-kbd">Ctrl K</span><Show when={open()}><div class="global-search-results"><Show when={searching()}><div class="global-search-status">Searching local workspace...</div></Show><Show when={!searching() && results().length === 0}><div class="global-search-status">No matching local results.</div></Show><For each={results()}>{(result) => <button class="global-search-result" onClick={() => openResult(result)}><span class="global-search-type">{result.collection.replace('_', ' ')}</span><span class="global-search-copy"><strong>{result.title}</strong><small>{result.excerpt.slice(0, 100)}</small></span></button>}</For></div></Show></div></div><div class="topbar-actions"><button class="topbar-icon-btn" title="Notifications" aria-label="Notifications"><Bell size={16} /></button><Show when={isWailsDesktop()}><div class="topbar-window-controls"><button title="Minimize" aria-label="Minimize" onClick={() => emitWindowControl('wnd:minimise')}><Minus size={15} /></button><button title="Maximize" aria-label="Maximize" onClick={() => emitWindowControl('wnd:toggle-maximise')}><Square size={12} /></button><button class="topbar-window-close" title="Close" aria-label="Close" onClick={() => emitWindowControl('wnd:close')}><X size={16} /></button></div></Show></div></header>;
};