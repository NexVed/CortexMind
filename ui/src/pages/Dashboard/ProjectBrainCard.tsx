import { Component, Show } from 'solid-js';
import { BrainCircuit } from 'lucide-solid';
import type { ProjectBrainStats } from '../../api/client';

interface Props {
  stats: ProjectBrainStats | null;
  loading: boolean;
}

export const ProjectBrainCard: Component<Props> = (props) => {
  return (
    <div class="dash-brain-card">
      <div class="dash-brain-header">
        <span class="dash-brain-title">Project Brain</span>
        <div class="dash-brain-icon">
          <BrainCircuit size={20} />
        </div>
      </div>

      <Show
        when={!props.loading && props.stats}
        fallback={
          <div class="dash-brain-rows">
            <div class="dash-shimmer dash-shimmer-line" style={{ height: '14px' }} />
            <div class="dash-shimmer dash-shimmer-line" style={{ height: '14px' }} />
            <div class="dash-shimmer dash-shimmer-line" style={{ height: '14px', width: '70%' }} />
          </div>
        }
      >
        <div class="dash-brain-rows">
          {/* Architecture Progress */}
          <div class="dash-brain-row">
            <span class="dash-brain-row-label">Architecture</span>
            <div class="dash-brain-bar-wrapper">
              <div class="dash-brain-bar">
                <div
                  class="dash-brain-bar-fill blue"
                  style={{ width: `${props.stats!.architectureProgress}%` }}
                />
              </div>
              <span class="dash-brain-percent">{props.stats!.architectureProgress}%</span>
            </div>
          </div>

          {/* Memory Coverage */}
          <div class="dash-brain-row">
            <span class="dash-brain-row-label">Memory Coverage</span>
            <div class="dash-brain-bar-wrapper">
              <div class="dash-brain-bar">
                <div
                  class="dash-brain-bar-fill green"
                  style={{ width: `${props.stats!.memoryCoverage}%` }}
                />
              </div>
              <span class="dash-brain-percent">{props.stats!.memoryCoverage}%</span>
            </div>
          </div>

          {/* Decisions Made */}
          <div class="dash-brain-stat-row">
            <span class="dash-brain-stat-label">Decisions Made</span>
            <span class="dash-brain-stat-value">{props.stats!.decisionsMade}</span>
          </div>

          {/* Tasks Linked */}
          <div class="dash-brain-stat-row">
            <span class="dash-brain-stat-label">Tasks Linked</span>
            <span class="dash-brain-stat-value">{props.stats!.tasksLinked}</span>
          </div>

          {/* Knowledge Quality */}
          <div class="dash-brain-stat-row">
            <span class="dash-brain-stat-label">Knowledge Quality</span>
            <span
              class={`dash-brain-quality ${props.stats!.knowledgeQuality.toLowerCase()}`}
            >
              {props.stats!.knowledgeQuality}
            </span>
          </div>
        </div>
      </Show>
    </div>
  );
};
