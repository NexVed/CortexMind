import { Component, onMount, onCleanup } from 'solid-js';
import './KnowledgeGraphPreview.css';

interface GraphNode {
  x: number;
  y: number;
  vx: number;
  vy: number;
  radius: number;
  color: string;
}

const COLORS = ['#6C63FF', '#3B82F6', '#22C55E', '#A855F7', '#F97316', '#EF4444', '#F59E0B'];

export const KnowledgeGraphPreview: Component = () => {
  let canvasRef: HTMLCanvasElement | undefined;
  let animationId: number;

  onMount(() => {
    if (!canvasRef) return;
    const ctx = canvasRef.getContext('2d');
    if (!ctx) return;

    const rect = canvasRef.parentElement!.getBoundingClientRect();
    canvasRef.width = rect.width * 2;
    canvasRef.height = rect.height * 2;
    ctx.scale(2, 2);

    const w = rect.width;
    const h = rect.height;

    // Create nodes
    const nodes: GraphNode[] = [];
    for (let i = 0; i < 18; i++) {
      nodes.push({
        x: Math.random() * w,
        y: Math.random() * h,
        vx: (Math.random() - 0.5) * 0.3,
        vy: (Math.random() - 0.5) * 0.3,
        radius: 3 + Math.random() * 4,
        color: COLORS[i % COLORS.length],
      });
    }

    // Create edges between nearby nodes
    const edges: [number, number][] = [];
    for (let i = 0; i < nodes.length; i++) {
      for (let j = i + 1; j < nodes.length; j++) {
        if (Math.random() < 0.25) {
          edges.push([i, j]);
        }
      }
    }

    function draw() {
      ctx!.clearRect(0, 0, w, h);

      // Draw edges
      for (const [i, j] of edges) {
        const a = nodes[i];
        const b = nodes[j];
        const dist = Math.hypot(a.x - b.x, a.y - b.y);
        if (dist < 120) {
          ctx!.beginPath();
          ctx!.moveTo(a.x, a.y);
          ctx!.lineTo(b.x, b.y);
          ctx!.strokeStyle = `rgba(108, 99, 255, ${0.15 * (1 - dist / 120)})`;
          ctx!.lineWidth = 1;
          ctx!.stroke();
        }
      }

      // Draw nodes
      for (const node of nodes) {
        ctx!.beginPath();
        ctx!.arc(node.x, node.y, node.radius, 0, Math.PI * 2);
        ctx!.fillStyle = node.color + '60';
        ctx!.fill();
        ctx!.beginPath();
        ctx!.arc(node.x, node.y, node.radius * 0.5, 0, Math.PI * 2);
        ctx!.fillStyle = node.color;
        ctx!.fill();
      }

      // Update positions
      for (const node of nodes) {
        node.x += node.vx;
        node.y += node.vy;
        if (node.x < 0 || node.x > w) node.vx *= -1;
        if (node.y < 0 || node.y > h) node.vy *= -1;
        node.x = Math.max(0, Math.min(w, node.x));
        node.y = Math.max(0, Math.min(h, node.y));
      }

      animationId = requestAnimationFrame(draw);
    }

    draw();
  });

  onCleanup(() => {
    if (animationId) cancelAnimationFrame(animationId);
  });

  return (
    <div class="knowledge-graph-preview">
      <div class="knowledge-graph-header">
        <span class="knowledge-graph-title">Knowledge Graph</span>
        <span class="knowledge-graph-link">View All</span>
      </div>
      <div class="knowledge-graph-canvas-container">
        <canvas ref={canvasRef} class="knowledge-graph-canvas" />
      </div>
    </div>
  );
};
