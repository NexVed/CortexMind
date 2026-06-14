import { Component, onMount, onCleanup, createSignal, Show, For } from 'solid-js';
import { ZoomIn, ZoomOut, Maximize2, RotateCcw, Sparkles } from 'lucide-solid';
import { listProjects, listTasks, listVaultEntries, getKnowledgeGraph } from '../../api/client';
import '../shared.css';
import './Graph.css';

const COLORS: Record<string, string> = {
  project: '#3B82F6',       // Blue
  task: '#22C55E',          // Green
  architecture: '#F97316',  // Orange
  decision: '#A855F7',      // Purple
  task_context: '#06B6D4',  // Cyan
  memory: '#F59E0B',        // Yellow
  // Analysis node types (from server-side scan)
  language: '#8B5CF6',
  framework: '#3B82F6',
  database: '#F59E0B',
  tool: '#06B6D4',
  auth: '#EF4444',
  provider: '#EC4899',
  feature: '#22C55E',
  module: '#64748B',
  unknown: '#6B7280',       // Gray
};

const legend = [
  { type: 'Project', color: COLORS.project },
  { type: 'Framework', color: COLORS.framework },
  { type: 'Database', color: COLORS.database },
  { type: 'Auth', color: COLORS.auth },
  { type: 'Feature', color: COLORS.feature },
  { type: 'Module', color: COLORS.module },
];

interface GraphNode {
  id: string;
  x: number;
  y: number;
  vx: number;
  vy: number;
  radius: number;
  color: string;
  label: string;
  type: string;
}

interface Edge {
  source: GraphNode;
  target: GraphNode;
}

export const GraphPage: Component = () => {
  let canvasRef: HTMLCanvasElement | undefined;
  let animationId: number;
  
  const [loading, setLoading] = createSignal(true);
  let nodes: GraphNode[] = [];
  let edges: Edge[] = [];
  
  let scale = 1;
  let offsetX = 0;
  let offsetY = 0;

  let draggedNode: GraphNode | null = null;
  let hoveredNode: GraphNode | null = null;

  async function loadData(w: number, h: number) {
    try {
      const [projects, tasks, entries] = await Promise.all([
        listProjects(),
        listTasks(),
        listVaultEntries()
      ]);

      const nodeMap = new Map<string, GraphNode>();

      const addNode = (id: string, label: string, type: string, radius: number) => {
        if (!nodeMap.has(id)) {
          const color = COLORS[type] || COLORS.unknown;
          nodeMap.set(id, {
            id, label, type, radius, color,
            x: w/2 + (Math.random() - 0.5) * 200,
            y: h/2 + (Math.random() - 0.5) * 200,
            vx: 0, vy: 0
          });
        }
      };

      // Create Project nodes
      for (const p of projects) {
        addNode(`proj_${p.id}`, p.name, 'project', 12);
      }

      // Merge server-side analysis knowledge graphs (tech stack, auth, features).
      await Promise.all(
        projects.map(async (p) => {
          try {
            const g = await getKnowledgeGraph(p.id);
            if (!g?.nodes?.length) return;
            const localId = (nid: string) =>
              nid === 'project' ? `proj_${p.id}` : `ag_${p.id}_${nid}`;
            for (const n of g.nodes) {
              if (n.id === 'project') continue; // hub maps to existing project node
              const radius = n.type === 'feature' || n.type === 'module' ? 7 : 6;
              addNode(localId(n.id), n.label, n.type, radius);
            }
            for (const e of g.edges) {
              const s = nodeMap.get(localId(e.source));
              const t = nodeMap.get(localId(e.target));
              if (s && t) edges.push({ source: s, target: t });
            }
          } catch {
            /* project may not be scanned yet */
          }
        })
      );

      // Create Task nodes
      for (const t of tasks) {
        addNode(`task_${t.id}`, t.title, 'task', 6);
        if (t.project) {
          addNode(`proj_${t.project}`, 'Unknown Project', 'project', 12);
        }
      }

      // Create Vault Entry nodes
      for (const e of entries) {
        addNode(`vault_${e.id}`, e.title, e.category || 'memory', 8);
        if (e.project) {
          addNode(`proj_${e.project}`, 'Unknown Project', 'project', 12);
        }
      }

      nodes = Array.from(nodeMap.values());

      // Create Edges
      for (const t of tasks) {
        if (t.project) {
          const s = nodeMap.get(`task_${t.id}`);
          const tgt = nodeMap.get(`proj_${t.project}`);
          if (s && tgt) edges.push({ source: s, target: tgt });
        }
      }

      for (const e of entries) {
        if (e.project) {
          const s = nodeMap.get(`vault_${e.id}`);
          const tgt = nodeMap.get(`proj_${e.project}`);
          if (s && tgt) edges.push({ source: s, target: tgt });
        }
      }
      
      // Add edges between entries with overlapping tags
      for (let i = 0; i < entries.length; i++) {
        for (let j = i + 1; j < entries.length; j++) {
          const a = entries[i];
          const b = entries[j];
          if (a.tags && b.tags) {
            const overlap = a.tags.some(tag => b.tags!.includes(tag));
            if (overlap) {
              const s = nodeMap.get(`vault_${a.id}`);
              const tgt = nodeMap.get(`vault_${b.id}`);
              if (s && tgt) edges.push({ source: s, target: tgt });
            }
          }
        }
      }

      setLoading(false);
    } catch (err) {
      console.error('Failed to load graph data', err);
      setLoading(false);
    }
  }

  onMount(() => {
    if (!canvasRef) return;
    const ctx = canvasRef.getContext('2d');
    if (!ctx) return;

    const container = canvasRef.parentElement!;
    const rect = container.getBoundingClientRect();
    canvasRef.width = rect.width * 2;
    canvasRef.height = rect.height * 2;
    ctx.scale(2, 2);
    const w = rect.width;
    const h = rect.height;

    loadData(w, h);

    // Mouse Interaction
    let isDragging = false;
    canvasRef.addEventListener('mousedown', (e) => {
      if (hoveredNode) {
        draggedNode = hoveredNode;
        isDragging = true;
      }
    });

    canvasRef.addEventListener('mousemove', (e) => {
      const br = canvasRef!.getBoundingClientRect();
      const mx = (e.clientX - br.left - offsetX) / scale;
      const my = (e.clientY - br.top - offsetY) / scale;

      if (isDragging && draggedNode) {
        draggedNode.x = mx;
        draggedNode.y = my;
        draggedNode.vx = 0;
        draggedNode.vy = 0;
      } else {
        hoveredNode = null;
        for (const node of nodes) {
          const dist = Math.hypot(node.x - mx, node.y - my);
          if (dist < node.radius + 6) { hoveredNode = node; break; }
        }
        canvasRef!.style.cursor = hoveredNode ? 'pointer' : 'default';
      }
    });

    canvasRef.addEventListener('mouseup', () => {
      isDragging = false;
      draggedNode = null;
    });

    canvasRef.addEventListener('mouseleave', () => {
      isDragging = false;
      draggedNode = null;
    });

    // Physics constants
    const K = 0.05; // Spring constant
    const REPULSION = 2500;
    const DAMPING = 0.85;
    const CENTER_PULL = 0.03;
    const IDEAL_LEN = 60;

    function draw() {
      // Physics Step
      if (nodes.length > 0) {
        for (let i = 0; i < nodes.length; i++) {
          for (let j = i + 1; j < nodes.length; j++) {
            const a = nodes[i];
            const b = nodes[j];
            const dx = a.x - b.x;
            const dy = a.y - b.y;
            const distSq = dx*dx + dy*dy;
            if (distSq > 0 && distSq < 100000) {
              const dist = Math.sqrt(distSq);
              const force = REPULSION / distSq;
              const fx = (dx / dist) * force;
              const fy = (dy / dist) * force;
              a.vx += fx; a.vy += fy;
              b.vx -= fx; b.vy -= fy;
            }
          }
          const a = nodes[i];
          a.vx += (w/2 - a.x) * CENTER_PULL;
          a.vy += (h/2 - a.y) * CENTER_PULL;
        }

        for (const edge of edges) {
          const dx = edge.target.x - edge.source.x;
          const dy = edge.target.y - edge.source.y;
          const dist = Math.hypot(dx, dy) || 1;
          const diff = dist - IDEAL_LEN;
          const force = diff * K;
          const fx = (dx / dist) * force;
          const fy = (dy / dist) * force;
          edge.source.vx += fx; edge.source.vy += fy;
          edge.target.vx -= fx; edge.target.vy -= fy;
        }

        for (const node of nodes) {
          if (node === draggedNode) continue;
          node.vx *= DAMPING;
          node.vy *= DAMPING;
          node.x += node.vx;
          node.y += node.vy;
          node.x = Math.max(10, Math.min(w - 10, node.x));
          node.y = Math.max(10, Math.min(h - 10, node.y));
        }
      }

      // Render
      ctx!.clearRect(0, 0, w, h);
      ctx!.save();
      ctx!.translate(offsetX, offsetY);
      ctx!.scale(scale, scale);

      for (const edge of edges) {
        ctx!.beginPath();
        ctx!.moveTo(edge.source.x, edge.source.y);
        ctx!.lineTo(edge.target.x, edge.target.y);
        ctx!.strokeStyle = 'rgba(108, 99, 255, 0.2)';
        ctx!.lineWidth = 1;
        ctx!.stroke();
      }

      for (const node of nodes) {
        const isHovered = node === hoveredNode;
        ctx!.beginPath();
        ctx!.arc(node.x, node.y, node.radius + (isHovered ? 3 : 0), 0, Math.PI * 2);
        ctx!.fillStyle = node.color + (isHovered ? 'FF' : 'AA');
        ctx!.fill();
        ctx!.beginPath();
        ctx!.arc(node.x, node.y, node.radius * 0.6, 0, Math.PI * 2);
        ctx!.fillStyle = node.color;
        ctx!.fill();

        if (isHovered || node.type === 'project') {
          ctx!.font = node.type === 'project' ? '600 12px Inter, sans-serif' : '500 11px Inter, sans-serif';
          const tw = ctx!.measureText(node.label).width;
          const px = node.x - tw / 2;
          const py = node.y - node.radius - 8;
          ctx!.fillStyle = 'rgba(22, 24, 31, 0.85)';
          ctx!.beginPath();
          ctx!.roundRect(px - 6, py - 12, tw + 12, 18, 4);
          ctx!.fill();
          ctx!.fillStyle = '#E8E9F3';
          ctx!.fillText(node.label, px, py);
        }
      }
      ctx!.restore();

      animationId = requestAnimationFrame(draw);
    }
    draw();
  });

  onCleanup(() => { if (animationId) cancelAnimationFrame(animationId); });

  return (
    <div class="graph-page">
      <div class="page-header">
        <div class="page-title-row">
          <div class="page-title-icon pink">
            <Sparkles size={24} />
          </div>
          <div>
            <h1 class="page-title">Knowledge Graph</h1>
            <p class="page-subtitle">Visualize connections across your entire workspace</p>
          </div>
        </div>
      </div>

      <div class="graph-card">
        <div class="graph-controls">
          <button class="graph-btn" title="Zoom In" onClick={() => scale *= 1.2}><ZoomIn size={16} /></button>
          <button class="graph-btn" title="Zoom Out" onClick={() => scale /= 1.2}><ZoomOut size={16} /></button>
          <button class="graph-btn" title="Reset" onClick={() => { scale = 1; offsetX = 0; offsetY = 0; }}><RotateCcw size={16} /></button>
        </div>
        
        <Show when={loading()}>
          <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', color: 'var(--text-muted)'}}>
            Compiling Knowledge Graph...
          </div>
        </Show>

        <canvas ref={canvasRef} style={{ width: '100%', height: '100%', display: 'block', background: 'var(--bg-base)' }} />
        
        <div class="graph-legend">
          <For each={legend}>
            {(item) => (
              <div class="legend-item">
                <div class="legend-dot" style={{ background: item.color }} />
                {item.type}
              </div>
            )}
          </For>
        </div>
      </div>
    </div>
  );
};
