import { Component, For, createResource, Show } from 'solid-js';
import {
  FileText,
  ArrowRightLeft,
  Map,
  Scan,
  BrainCircuit,
  Activity
} from 'lucide-solid';
import { listActivity } from '../../api/client';
import './ActivityFeed.css';

// Map actions to icons and colors
function getActivityIconInfo(action: string) {
  if (action.includes('index') || action.includes('file')) return { icon: FileText, color: 'blue' };
  if (action.includes('handoff')) return { icon: ArrowRightLeft, color: 'orange' };
  if (action.includes('roadmap') || action.includes('plan')) return { icon: Map, color: 'purple' };
  if (action.includes('scan')) return { icon: Scan, color: 'green' };
  if (action.includes('ai') || action.includes('context')) return { icon: BrainCircuit, color: 'violet' };
  return { icon: Activity, color: 'gray' };
}

export const ActivityFeed: Component = () => {
  const [activities] = createResource(() => listActivity(5));

  const formatTimeAgo = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays === 1) return 'Yesterday';
    return `${diffDays}d ago`;
  };

  return (
    <div class="activity-feed">
      <div class="activity-feed-header">Recent Activity</div>
      
      <Show when={!activities.loading && activities()?.length === 0}>
        <div style={{ padding: '16px', "text-align": 'center', color: 'var(--text-muted)' }}>
          No recent activity.
        </div>
      </Show>

      <For each={activities()}>
        {(item) => {
          const { icon: Icon, color } = getActivityIconInfo(item.action);
          return (
            <div class="activity-feed-item">
              <div class={`activity-feed-icon ${color}`}>
                <Icon size={16} />
              </div>
              <div class="activity-feed-content">
                <div class="activity-feed-action">{item.action} {item.subject}</div>
                <div class="activity-feed-context">Project {item.project || 'Global'}</div>
              </div>
              <span class="activity-feed-time">{formatTimeAgo(item.created)}</span>
            </div>
          );
        }}
      </For>
    </div>
  );
};
