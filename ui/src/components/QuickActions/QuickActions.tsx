import { Component, For } from 'solid-js';
import {
  FolderPlus,
  Scan,
  ArrowRightLeft,
  BrainCircuit,
} from 'lucide-solid';
import './QuickActions.css';

const actions = [
  { icon: FolderPlus, label: 'Add Project', color: 'purple' },
  { icon: Scan, label: 'Scan Repo', color: 'green' },
  { icon: ArrowRightLeft, label: 'New Handoff', color: 'orange' },
  { icon: BrainCircuit, label: 'Generate Context', color: 'blue' },
];

export const QuickActions: Component = () => {
  return (
    <div class="quick-actions">
      <div class="quick-actions-header">Quick Actions</div>
      <div class="quick-actions-grid">
        <For each={actions}>
          {(action) => {
            const Icon = action.icon;
            return (
              <button class="quick-action-card">
                <div class={`quick-action-icon ${action.color}`}>
                  <Icon size={24} />
                </div>
                <span class="quick-action-label">{action.label}</span>
              </button>
            );
          }}
        </For>
      </div>
    </div>
  );
};
