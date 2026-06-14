import { Component, For, Show } from 'solid-js';
import type { AgentInfo } from '../../api/client';

interface Props {
  agents: AgentInfo[];
  loading: boolean;
}

// Agent brand colors and icons
const agentBrands: Record<string, { bg: string; fg: string; letter: string }> = {
  claude: { bg: '#D4A574', fg: '#fff', letter: 'C' },
  codex: { bg: '#10A37F', fg: '#fff', letter: 'Cx' },
  gemini: { bg: '#4285F4', fg: '#fff', letter: 'G' },
  kiro: { bg: '#FF6B35', fg: '#fff', letter: 'K' },
  aider: { bg: '#9333EA', fg: '#fff', letter: 'A' },
  cursor: { bg: '#1A1A2E', fg: '#fff', letter: 'Cu' },
  copilot: { bg: '#000', fg: '#fff', letter: 'Co' },
};

function getAgentBrand(name: string) {
  const lower = name.toLowerCase();
  for (const [key, val] of Object.entries(agentBrands)) {
    if (lower.includes(key)) return val;
  }
  // Generate color from name
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = ((hash << 5) - hash + name.charCodeAt(i)) | 0;
  }
  const h = Math.abs(hash) % 360;
  return { bg: `hsl(${h}, 55%, 50%)`, fg: '#fff', letter: name.charAt(0).toUpperCase() };
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

export const AgentsCard: Component<Props> = (props) => {
  return (
    <div class="dash-agents-card">
      <div class="dash-card-header">
        <span class="dash-card-title">AI Agents</span>
        <span class="dash-card-link">Manage</span>
      </div>

      <Show
        when={!props.loading}
        fallback={
          <div>
            <div class="dash-shimmer dash-shimmer-line" />
            <div class="dash-shimmer dash-shimmer-line" />
            <div class="dash-shimmer dash-shimmer-line" style={{ width: '50%' }} />
          </div>
        }
      >
        <Show when={props.agents.length === 0}>
          <div style={{ padding: '16px', 'text-align': 'center', color: 'var(--text-muted)', 'font-size': '13px' }}>
            No agents detected yet. Agents appear when they interact with vault entries or handoffs.
          </div>
        </Show>

        <div class="dash-agents-list">
          <For each={props.agents}>
            {(agent) => {
              const brand = getAgentBrand(agent.name);
              return (
                <div class="dash-agent-row">
                  <div
                    class="dash-agent-icon"
                    style={{ background: brand.bg, color: brand.fg }}
                  >
                    {brand.letter}
                  </div>
                  <span class="dash-agent-name">{agent.name}</span>
                  <div class={`dash-agent-status ${agent.status}`}>
                    <span class="dash-agent-status-dot" />
                    {capitalize(agent.status)}
                  </div>
                </div>
              );
            }}
          </For>
        </div>
      </Show>
    </div>
  );
};
