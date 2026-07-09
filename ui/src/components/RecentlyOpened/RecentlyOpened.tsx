import { Component, For, Show } from 'solid-js';
import { FileCode2 } from 'lucide-solid';
import { useRecentFiles } from '../../api/queries';
import './RecentlyOpened.css';

export const RecentlyOpened: Component = () => {
  const recentFilesQuery = useRecentFiles(4);
  const recentFiles = () => recentFilesQuery.data;

  return (
    <div class="recently-opened">
      <div class="recently-opened-header">Recently Opened</div>
      
      <Show when={!recentFilesQuery.isLoading && recentFiles()?.length === 0}>
        <div style={{ padding: '8px', color: 'var(--text-muted)' }}>
          No recently opened files found.
        </div>
      </Show>

      <div class="recently-opened-strip">
        <For each={recentFiles()}>
          {(file) => {
            // Extract just the filename from the path
            const name = file.path.split(/[\/\\]/).pop() || file.path;
            
            return (
              <div class="recently-opened-chip">
                <div class={`recently-opened-chip-icon ${file.language?.toLowerCase() || 'unknown'}`}>
                  <FileCode2 size={14} />
                </div>
                <div>
                  <div class="recently-opened-chip-name">{name}</div>
                  <div class="recently-opened-chip-project">{file.project || 'Global'}</div>
                </div>
              </div>
            );
          }}
        </For>
      </div>
    </div>
  );
};
