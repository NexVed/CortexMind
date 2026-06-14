import { Component } from 'solid-js';

interface Props {
  daemonReady: boolean;
  projectCount: number;
  totalMemories: number;
  handoffCount: number;
  decisionCount: number;
  teamMembers: number;
}

export const StatusBar: Component<Props> = (props) => {
  return (
    <div class="dash-status-bar">
      <div class="dash-status-left">
        <span class={`dash-status-dot ${props.daemonReady ? '' : 'offline'}`} />
        <span>{props.daemonReady ? 'All systems operational' : 'Daemon offline'}</span>
      </div>

      <div class="dash-status-stats">
        <div class="dash-status-stat">
          Projects <strong>{props.projectCount}</strong>
        </div>
        <div class="dash-status-stat">
          Total Memories <strong>{props.totalMemories.toLocaleString()}</strong>
        </div>
        <div class="dash-status-stat">
          Handoffs <strong>{props.handoffCount}</strong>
        </div>
        <div class="dash-status-stat">
          Decisions <strong>{props.decisionCount}</strong>
        </div>
        <div class="dash-status-stat">
          Team Members <strong>{props.teamMembers}</strong>
        </div>
      </div>

      <div class="dash-status-right">
        <div class="dash-status-server">
          <span class="dash-status-server-label">MCP Server</span>
          <span class={`dash-status-server-value ${props.daemonReady ? 'running' : 'stopped'}`}>
            {props.daemonReady ? 'Running' : 'Stopped'}
          </span>
        </div>
      </div>
    </div>
  );
};
