import { Component } from 'solid-js';
import { Sparkles } from 'lucide-solid';

interface Props {
  decisions: number;
  files: number;
  tasks: number;
}

export const ContextGeneratorCard: Component<Props> = (props) => {
  return (
    <div class="dash-context-card">
      <span class="dash-card-title">Context Generator</span>
      <span class="dash-context-subtitle">
        Generate AI-ready context for any agent.
      </span>

      {/* Current Goal */}
      <div class="dash-context-goal">
        <span class="dash-context-goal-badge">Current Goal</span>
        <span class="dash-context-goal-text">
          {props.tasks > 0
            ? `${props.tasks} active task${props.tasks !== 1 ? 's' : ''} to complete`
            : 'No active tasks'}
        </span>
      </div>

      {/* Stats */}
      <div class="dash-context-stats">
        <div class="dash-context-stat">
          <span class="dash-context-stat-label">Recent Decisions</span>
          <span class="dash-context-stat-value">{props.decisions}</span>
        </div>
        <div class="dash-context-stat">
          <span class="dash-context-stat-label">Relevant Files</span>
          <span class="dash-context-stat-value">{props.files}</span>
        </div>
        <div class="dash-context-stat">
          <span class="dash-context-stat-label">Active Tasks</span>
          <span class="dash-context-stat-value">{props.tasks}</span>
        </div>
      </div>

      {/* Generate Button */}
      <button class="dash-generate-btn" id="generate-context-btn">
        <Sparkles size={16} />
        Generate Context
      </button>
    </div>
  );
};
