import { Component, For, Show } from 'solid-js';
import {
  ArrowRightLeft,
  FileText,
  Map,
  Scan,
  BrainCircuit,
  Activity,
} from 'lucide-solid';
import type { ActivityLogEntry } from '../../api/client';

interface Props {
  activities: ActivityLogEntry[];
  loading: boolean;
}

function getIconInfo(action: string) {
  if (action.includes('handoff')) return { icon: ArrowRightLeft, color: 'orange' };
  if (action.includes('architecture') || action.includes('diagram'))
    return { icon: Map, color: 'blue' };
  if (action.includes('decision')) return { icon: BrainCircuit, color: 'purple' };
  if (action.includes('index') || action.includes('file'))
    return { icon: FileText, color: 'green' };
  if (action.includes('scan')) return { icon: Scan, color: 'accent' };
  if (action.includes('task')) return { icon: FileText, color: 'red' };
  return { icon: Activity, color: 'blue' };
}

function formatTimeAgo(dateStr: string): string {
  const d = new Date(dateStr);
  const now = Date.now();
  const diffMs = now - d.getTime();
  const mins = Math.floor(diffMs / 60000);
  const hrs = Math.floor(mins / 60);
  const days = Math.floor(hrs / 24);

  if (mins < 1) return 'Just now';
  if (mins < 60) return `${mins}m ago`;
  if (hrs < 24) return `${hrs}h ago`;
  if (days === 1) return 'Yesterday';
  return `${days}d ago`;
}

export const ActivityFeedCard: Component<Props> = (props) => {
  return (
    <div class="dash-activity-card">
      <div class="dash-card-header">
        <span class="dash-card-title">Activity Feed</span>
        <span class="dash-card-link">View all</span>
      </div>

      <Show
        when={!props.loading}
        fallback={
          <div>
            <div class="dash-shimmer dash-shimmer-line" />
            <div class="dash-shimmer dash-shimmer-line" />
            <div class="dash-shimmer dash-shimmer-line" style={{ width: '60%' }} />
          </div>
        }
      >
        <Show when={props.activities.length === 0}>
          <div style={{ padding: '16px', 'text-align': 'center', color: 'var(--text-muted)', 'font-size': '13px' }}>
            No recent activity for this project.
          </div>
        </Show>

        <div class="dash-activity-list">
          <For each={props.activities}>
            {(item) => {
              const { icon: Icon, color } = getIconInfo(item.action);
              return (
                <div class="dash-activity-item">
                  <div class={`dash-activity-icon ${color}`}>
                    <Icon size={14} />
                  </div>
                  <span class="dash-activity-text">
                    {item.action} {item.subject}
                  </span>
                  <span class="dash-activity-time">{formatTimeAgo(item.created)}</span>
                </div>
              );
            }}
          </For>
        </div>
      </Show>
    </div>
  );
};
