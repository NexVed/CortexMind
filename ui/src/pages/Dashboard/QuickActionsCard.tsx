import { Component, For } from 'solid-js';
import {
  FilePlus,
  BrainCircuit,
  ArrowRightLeft,
  Upload,
  Sparkles,
} from 'lucide-solid';

const quickActions = [
  { icon: FilePlus, label: 'New Task' },
  { icon: BrainCircuit, label: 'New Decision' },
  { icon: ArrowRightLeft, label: 'New Handoff' },
  { icon: Upload, label: 'Upload File' },
  { icon: Sparkles, label: 'Ask Cortex (AI)' },
];

export const QuickActionsCard: Component = () => {
  return (
    <div class="dash-quick-card">
      <span class="dash-card-title">Quick Actions</span>

      <div class="dash-quick-list">
        <For each={quickActions}>
          {(action) => {
            const Icon = action.icon;
            return (
              <button class="dash-quick-item" id={`quick-${action.label.toLowerCase().replace(/\s+/g, '-')}`}>
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
