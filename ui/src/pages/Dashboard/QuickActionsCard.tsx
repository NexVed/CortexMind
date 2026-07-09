import { Component, For } from 'solid-js';
import {
  FilePlus,
  BrainCircuit,
  ArrowRightLeft,
  FolderPlus,
  Sparkles,
} from 'lucide-solid';

interface Props {
  // Called with a route path when a quick action is clicked.
  onNavigate: (path: string) => void;
}

const quickActions = [
  { icon: FilePlus, label: 'New Task', path: '/tasks' },
  { icon: BrainCircuit, label: 'New Decision', path: '/vaults' },
  { icon: ArrowRightLeft, label: 'New Handoff', path: '/handoffs' },
  { icon: FolderPlus, label: 'Add Project', path: '/projects' },
  { icon: Sparkles, label: 'Build System Prompt', path: '/ai-context' },
];

export const QuickActionsCard: Component<Props> = (props) => {
  return (
    <div class="dash-quick-card">
      <span class="dash-card-title">Quick Actions</span>

      <div class="dash-quick-list">
        <For each={quickActions}>
          {(action) => {
            const Icon = action.icon;
            return (
              <button
                class="dash-quick-item"
                id={`quick-${action.label.toLowerCase().replace(/\s+/g, '-')}`}
                onClick={() => props.onNavigate(action.path)}
              >
                <Icon size={16} />
                {action.label}
              </button>
            );
          }}
        </For>
      </div>
    </div>
  );
};
