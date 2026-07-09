import { Component, For, createMemo, createSignal } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import type { KnowledgeGraphNode } from '../../api/client';

interface Props {
  nodes: KnowledgeGraphNode[];
}

interface BubbleData {
  id: string;
  label: string;
  count: number;
  color: string;
  x: number;
  y: number;
  r: number;
}

export const BubbleKnowledgeCard: Component<Props> = (props) => {
  const navigate = useNavigate();
  const [hoveredId, setHoveredId] = createSignal<string | null>(null);

  // Derive the bubble layout reactively so it updates whenever the dashboard's
  // knowledge-graph data finishes loading (the previous onMount-only layout
  // captured the initial empty state and never refreshed, so counts stuck at 0).
  const bubbles = createMemo<BubbleData[]>(() => {
    const nodes = props.nodes;
    if (!nodes.length) return [];

    const cx = 200;
    const cy = 120;
    const maxCount = Math.max(...nodes.map((n) => n.count), 1);

    // Largest node sits at the center; the rest orbit around it.
    const sorted = [...nodes].sort((a, b) => b.count - a.count);
    const angleStep = (2 * Math.PI) / Math.max(sorted.length - 1, 1);

    return sorted.map((node, i) => {
      // Square-root scale keeps a dominant count (e.g. Files: 116) from
      // ballooning past the card while still showing relative size.
      const r = 22 + Math.sqrt(node.count / maxCount) * 16;
      let x: number, y: number;
      if (i === 0) {
        x = cx;
        y = cy;
      } else {
        const angle = angleStep * (i - 1) - Math.PI / 2;
        const orbitR = 80 + (i % 2 === 0 ? 15 : 0);
        x = cx + Math.cos(angle) * orbitR;
        y = cy + Math.sin(angle) * orbitR;
      }
      return { ...node, x, y, r };
    });
  });

  return (
    <div class="dash-bubble-card">
      <div class="dash-card-header">
        <span class="dash-card-title">Bubble Knowledge</span>
        <span
          class="dash-card-link"
          style={{ cursor: 'pointer' }}
          onClick={() => navigate('/code-graph')}
        >
          View full graph
        </span>
      </div>

      <div class="dash-bubble-container">
        <svg
          class="dash-bubble-svg"
          viewBox="0 0 400 240"
          preserveAspectRatio="xMidYMid meet"
        >
          {/* Connection lines from center to orbiting nodes */}
          <For each={bubbles()}>
            {(bubble, i) => {
              if (i() === 0) return null;
              const center = bubbles()[0];
              if (!center) return null;
              return (
                <line
                  x1={center.x}
                  y1={center.y}
                  x2={bubble.x}
                  y2={bubble.y}
                  stroke={bubble.color}
                  stroke-width="1.5"
                  stroke-opacity="0.2"
                  stroke-dasharray="4 3"
                />
              );
            }}
          </For>

          {/* Bubbles */}
          <For each={bubbles()}>
            {(bubble) => {
              const isHovered = () => hoveredId() === bubble.id;
              const scale = () => (isHovered() ? 1.08 : 1);
              return (
                <g
                  onMouseEnter={() => setHoveredId(bubble.id)}
                  onMouseLeave={() => setHoveredId(null)}
                  style={{ cursor: 'pointer' }}
                >
                  {/* Outer glow */}
                  <circle
                    cx={bubble.x}
                    cy={bubble.y}
                    r={bubble.r + 4}
                    fill={bubble.color}
                    opacity={isHovered() ? 0.15 : 0.08}
                    style={{ transition: 'opacity 200ms ease' }}
                  />
                  {/* Main circle */}
                  <circle
                    cx={bubble.x}
                    cy={bubble.y}
                    r={bubble.r}
                    fill={bubble.color}
                    opacity={isHovered() ? 0.35 : 0.2}
                    stroke={bubble.color}
                    stroke-width="1.5"
                    stroke-opacity={isHovered() ? 0.6 : 0.3}
                    style={{
                      transition: 'opacity 200ms ease, r 200ms ease',
                      transform: `scale(${scale()})`,
                      'transform-origin': `${bubble.x}px ${bubble.y}px`,
                    }}
                  />
                  {/* Label */}
                  <text
                    x={bubble.x}
                    y={bubble.y - 4}
                    text-anchor="middle"
                    dominant-baseline="middle"
                    fill={bubble.color}
                    font-size="10"
                    font-weight="600"
                    font-family="Inter, sans-serif"
                  >
                    {bubble.label}
                  </text>
                  {/* Count */}
                  <text
                    x={bubble.x}
                    y={bubble.y + 10}
                    text-anchor="middle"
                    dominant-baseline="middle"
                    fill={bubble.color}
                    font-size="12"
                    font-weight="700"
                    font-family="Inter, sans-serif"
                    opacity="0.8"
                  >
                    {bubble.count}
                  </text>
                </g>
              );
            }}
          </For>
        </svg>
      </div>
    </div>
  );
};
