import { Component, createSignal, For, createEffect, Show } from 'solid-js';
import {
  Search as SearchIcon,
  Filter,
  FileCode2,
  Archive,
  ListTodo,
  ArrowRightLeft,
  ChevronRight,
  BrainCircuit,
} from 'lucide-solid';
import { searchAll, SearchResult } from '../../api/client';
import './Search.css';

const getCollectionIcon = (collection: string) => {
  switch (collection) {
    case 'file_index': return FileCode2;
    case 'vault_entries': return Archive;
    case 'tasks': return ListTodo;
    case 'handoffs': return ArrowRightLeft;
    default: return SearchIcon;
  }
};

const getCollectionColor = (collection: string) => {
  switch (collection) {
    case 'file_index': return 'blue';
    case 'vault_entries': return 'green';
    case 'tasks': return 'purple';
    case 'handoffs': return 'orange';
    default: return 'gray';
  }
};

const formatCollectionName = (collection: string) => {
  return collection.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
};

export const SearchPage: Component = () => {
  const [query, setQuery] = createSignal('');
  const [isFocused, setIsFocused] = createSignal(false);
  const [results, setResults] = createSignal<SearchResult[]>([]);
  const [isSearching, setIsSearching] = createSignal(false);

  let searchTimeout: any;

  createEffect(() => {
    const q = query();
    if (!q.trim()) {
      setResults([]);
      setIsSearching(false);
      return;
    }

    setIsSearching(true);
    clearTimeout(searchTimeout);
    
    searchTimeout = setTimeout(async () => {
      try {
        const data = await searchAll(q);
        setResults(data);
      } catch (err) {
        console.error("Search failed:", err);
        setResults([]);
      } finally {
        setIsSearching(false);
      }
    }, 300); // 300ms debounce
  });

  return (
    <div class="search-page">
      {/* Hero Header */}
      <div class="search-hero">
        <div class="search-hero-content">
          <div class="search-hero-icon">
            <BrainCircuit size={32} />
          </div>
          <h1 class="search-hero-title">Universal Search</h1>
          <p class="search-hero-subtitle">
            Search across files, tasks, handoffs, and vault knowledge using semantic search.
          </p>
        </div>

        <div class={`search-hero-input-container ${isFocused() ? 'focused' : ''}`}>
          <SearchIcon class="search-hero-input-icon" size={24} />
          <input
            type="text"
            class="search-hero-input"
            placeholder="What are you looking for?"
            value={query()}
            onInput={(e) => setQuery(e.currentTarget.value)}
            onFocus={() => setIsFocused(true)}
            onBlur={() => setIsFocused(false)}
            autofocus
          />
          <button class="search-hero-filter-btn">
            <Filter size={18} />
          </button>
        </div>
      </div>

      <div class="search-content">
        <Show when={isSearching()}>
          <div class="search-loading">Searching your knowledge base...</div>
        </Show>

        <Show when={!isSearching() && query() && results().length === 0}>
          <div class="search-empty">
            <SearchIcon size={48} />
            <h3>No results found</h3>
            <p>We couldn't find anything matching "{query()}"</p>
          </div>
        </Show>

        <Show when={!isSearching() && !query()}>
          <div class="search-suggestions">
            <h3>Try searching for:</h3>
            <div class="suggestion-chips">
              <button onClick={() => setQuery('authentication flow')}>Authentication flow</button>
              <button onClick={() => setQuery('database schema')}>Database schema</button>
              <button onClick={() => setQuery('deployment steps')}>Deployment steps</button>
            </div>
          </div>
        </Show>

        <div class="search-results-list">
          <For each={results()}>
            {(result) => {
              const Icon = getCollectionIcon(result.collection);
              const color = getCollectionColor(result.collection);

              return (
                <div class="search-result-card">
                  <div class={`search-result-icon ${color}`}>
                    <Icon size={20} />
                  </div>
                  <div class="search-result-content">
                    <div class="search-result-header">
                      <span class="search-result-type">{formatCollectionName(result.collection)}</span>
                      <span class="search-result-project">{result.project_id || 'Global'}</span>
                    </div>
                    <h3 class="search-result-title">{result.title}</h3>
                    <p class="search-result-excerpt">
                      {/* Highlight match simply by rendering excerpt. In a real app we'd inject <strong> */}
                      {result.excerpt}
                    </p>
                  </div>
                  <button class="search-result-action">
                    <ChevronRight size={20} />
                  </button>
                </div>
              );
            }}
          </For>
        </div>
      </div>
    </div>
  );
};
