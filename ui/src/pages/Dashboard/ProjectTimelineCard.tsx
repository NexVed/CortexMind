import { Component, For, createMemo } from 'solid-js';
import type { VaultEntry, Handoff } from '../../api/client';

interface Props {
  vaultEntries: VaultEntry[];
  handoffs: Handoff[];
  projectId: string;
}

interface TimelineEvent {
  date: string;
  dateFormatted: string;
  text: string;
  type: 'decision' | 'task' | 'handoff' | 'memory';
  color: string;
}

const typeColors: Record<string, string> = {
  decision: 'accent',
  task: 'blue',
  handoff: 'purple',
  memory: 'green',
};

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

export const ProjectTimelineCard: Component<Props> = (props) => {
  const events = createMemo<TimelineEvent[]>(() => {
    const items: TimelineEvent[] = [];

    // Vault entries as timeline items
    for (const v of props.vaultEntries) {
      if (v.project && v.project !== props.projectId) continue;

      let type: TimelineEvent['type'] = 'memory';
      if (v.category === 'decision') type = 'decision';
      else if (v.category === 'architecture') type = 'memory';
      else if (v.category === 'task') type = 'task';

      items.push({
        date: v.created || v.updated,
        dateFormatted: formatDate(v.created || v.updated),
        text: v.title,
        type,
        color: typeColors[type],
      });
    }

    // Handoffs as timeline items
    for (const h of props.handoffs) {
      if (h.project && h.project !== props.projectId) continue;

      items.push({
        date: h.created || h.updated,
        dateFormatted: formatDate(h.created || h.updated),
        text: h.title || `Handoff from ${h.from_agent}`,
        type: 'handoff',
        color: typeColors.handoff,
      });
    }

    // Sort by date descending
    items.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime());

    return items.slice(0, 6);
  });

  return (
    <div class="dash-timeline-card">
      <div class="dash-card-header">
        <span class="dash-card-title">Project Timeline</span>
        <span class="dash-card-link">View roadmap</span>
      </div>

      <div class="dash-timeline-list">
        <For each={events()} fallback={
          <div style={{ padding: '16px', 'text-align': 'center', color: 'var(--text-muted)', 'font-size': '13px' }}>
            No timeline events yet.
          </div>
        }>
          {(event) => (
            <div class="dash-timeline-item">
              <span class={`dash-timeline-dot ${event.color}`} />
              <span class="dash-timeline-date">{event.dateFormatted}</span>
              <span class="dash-timeline-text">{event.text}</span>
              <span class={`dash-timeline-badge ${event.type}`}>{event.type}</span>
            </div>
          )}
        </For>
      </div>
    </div>
  );
};
