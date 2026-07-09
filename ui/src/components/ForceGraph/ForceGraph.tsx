import { Component, onMount, onCleanup, createEffect } from 'solid-js';
import { ZoomIn, ZoomOut, RotateCcw } from 'lucide-solid';
import './ForceGraph.css';

export interface FGNode {
  id: string;
  label: string;
  color: string;
  radius?: number;      // default 6
  labelAlways?: boolean; // always show label (e.g. hub / directory nodes)
}

export interface FGEdge {
  source: string;
  target: string;
  strong?: boolean;      // draw a slightly heavier link
}

interface Props {
  nodes: FGNode[];
  edges: FGEdge[];
  selectedId?: string | null;
  highlight?: string;              // substring to emphasize; others dim
  onSelect?: (id: string | null) => void;
}

// Internal particle.
interface P {
  id: string;
  label: string;
  color: string;
  radius: number;
  labelAlways: boolean;
  x: number;
  y: number;
  vx: number;
  vy: number;
}

/**
 * ForceGraph is a minimal, Obsidian-style bubble graph rendered on a canvas.
 * The layout runs a cooling force simulation that settles within ~1s and then
 * freezes (no perpetual animation). Supports drag, scroll-zoom, pan, hover
 * highlighting of a node and its neighbours, and click-to-select.
 */
export const ForceGraph: Component<Props> = (props) => {
  let canvas: HTMLCanvasElement | undefined;
  let ctx: CanvasRenderingContext2D | null = null;
  let raf = 0;
  let ready = false;

  let nodes: P[] = [];
  let links: { s: P; t: P; strong: boolean }[] = [];
  const posById = new Map<string, { x: number; y: number }>();
  const neighbors = new Map<string, Set<string>>();

  let W = 0;
  let H = 0;
  const DPR = Math.min(window.devicePixelRatio || 1, 2);

  let scale = 1;
  let offX = 0;
  let offY = 0;
  let alpha = 1;

  let hover: P | null = null;
  let dragNode: P | null = null;
  let panning = false;
  let panStart = { x: 0, y: 0, ox: 0, oy: 0 };
  let downAt = { x: 0, y: 0 };
  let moved = false;

  // Physics constants — tuned to settle quickly and stay calm.
  const CENTER = 0.045;
  const REPULSION = 320;
  const LINK_K = 0.06;
  const IDEAL = 48;
  const FRICTION = 0.8;
  const COOL = 0.985;
  const ALPHA_MIN = 0.02;

  const reheat = () => {
    alpha = Math.max(alpha, 0.9);
  };

  const rebuild = () => {
    const byId = new Map<string, P>();
    nodes = props.nodes.map((n) => {
      const prev = posById.get(n.id);
      const baseW = W || 800;
      const baseH = H || 500;
      const p: P = {
        id: n.id,
        label: n.label,
        color: n.color,
        radius: n.radius ?? 6,
        labelAlways: !!n.labelAlways,
        x: prev?.x ?? baseW / 2 + (Math.random() - 0.5) * Math.min(baseW, baseH) * 0.5,
        y: prev?.y ?? baseH / 2 + (Math.random() - 0.5) * Math.min(baseW, baseH) * 0.5,
        vx: 0,
        vy: 0,
      };
      byId.set(n.id, p);
      return p;
    });
    links = [];
    neighbors.clear();
    for (const e of props.edges) {
      const s = byId.get(e.source);
      const t = byId.get(e.target);
      if (!s || !t) continue;
      links.push({ s, t, strong: !!e.strong });
      if (!neighbors.has(s.id)) neighbors.set(s.id, new Set());
      if (!neighbors.has(t.id)) neighbors.set(t.id, new Set());
      neighbors.get(s.id)!.add(t.id);
      neighbors.get(t.id)!.add(s.id);
    }
    // Drop stale saved positions for removed nodes.
    for (const id of [...posById.keys()]) if (!byId.has(id)) posById.delete(id);
    reheat();
  };

  const resize = () => {
    if (!canvas || !ctx) return;
    const r = canvas.getBoundingClientRect();
    W = r.width;
    H = r.height;
    canvas.width = Math.max(1, Math.round(W * DPR));
    canvas.height = Math.max(1, Math.round(H * DPR));
    ctx.setTransform(DPR, 0, 0, DPR, 0, 0);
  };

  const tick = () => {
    if (alpha > ALPHA_MIN && nodes.length > 0) {
      for (let i = 0; i < nodes.length; i++) {
        const a = nodes[i];
        for (let j = i + 1; j < nodes.length; j++) {
          const b = nodes[j];
          let dx = a.x - b.x;
          let dy = a.y - b.y;
          let d2 = dx * dx + dy * dy;
          if (d2 === 0) {
            dx = Math.random() - 0.5;
            dy = Math.random() - 0.5;
            d2 = dx * dx + dy * dy;
          }
          if (d2 < 62500) {
            const dist = Math.sqrt(d2);
            const f = (REPULSION / d2) * alpha;
            const fx = (dx / dist) * f;
            const fy = (dy / dist) * f;
            a.vx += fx;
            a.vy += fy;
            b.vx -= fx;
            b.vy -= fy;
          }
        }
        a.vx += (W / 2 - a.x) * CENTER * alpha;
        a.vy += (H / 2 - a.y) * CENTER * alpha;
      }
      for (const l of links) {
        const dx = l.t.x - l.s.x;
        const dy = l.t.y - l.s.y;
        const dist = Math.hypot(dx, dy) || 1;
        const f = (dist - IDEAL) * LINK_K * alpha;
        const fx = (dx / dist) * f;
        const fy = (dy / dist) * f;
        l.s.vx += fx;
        l.s.vy += fy;
        l.t.vx -= fx;
        l.t.vy -= fy;
      }
      for (const n of nodes) {
        if (n === dragNode) continue;
        n.vx *= FRICTION;
        n.vy *= FRICTION;
        n.x += n.vx;
        n.y += n.vy;
        posById.set(n.id, { x: n.x, y: n.y });
      }
      alpha *= COOL;
    }
    draw();
    raf = requestAnimationFrame(tick);
  };

  const draw = () => {
    if (!ctx) return;
    ctx.clearRect(0, 0, W, H);
    ctx.save();
    ctx.translate(offX, offY);
    ctx.scale(scale, scale);

    const q = (props.highlight || '').trim().toLowerCase();
    const sel = props.selectedId ?? null;
    const activeId = hover?.id ?? sel ?? null;
    const activeSet = activeId ? neighbors.get(activeId) : null;

    // Links
    for (const l of links) {
      const related = !!activeId && (l.s.id === activeId || l.t.id === activeId);
      ctx.beginPath();
      ctx.moveTo(l.s.x, l.s.y);
      ctx.lineTo(l.t.x, l.t.y);
      ctx.strokeStyle = related ? 'rgba(110,110,135,0.6)' : 'rgba(140,140,160,0.16)';
      ctx.lineWidth = related ? 1.4 : l.strong ? 1.1 : 0.9;
      ctx.stroke();
    }

    // Nodes
    for (const n of nodes) {
      const isActive = activeId === n.id;
      const isNeighbor = !!activeSet && activeSet.has(n.id);
      const match = q.length > 0 && n.label.toLowerCase().includes(q);
      const dim = (!!activeId && !isActive && !isNeighbor) || (q.length > 0 && !match);
      ctx.globalAlpha = dim ? 0.2 : 1;

      ctx.beginPath();
      ctx.arc(n.x, n.y, n.radius, 0, Math.PI * 2);
      ctx.fillStyle = n.color;
      ctx.fill();

      if (sel === n.id || (match && q.length > 0)) {
        ctx.lineWidth = 2;
        ctx.strokeStyle = 'rgba(20,20,26,0.9)';
        ctx.stroke();
      }

      const showLabel = !dim && (n.labelAlways || isActive || isNeighbor || match || scale > 1.35);
      if (showLabel) {
        const fontPx = n.labelAlways ? 11.5 : 10.5;
        ctx.font = `${n.labelAlways ? '600' : '500'} ${fontPx}px Inter, system-ui, sans-serif`;
        ctx.textAlign = 'center';
        ctx.fillStyle = isActive ? 'rgba(20,20,26,0.95)' : 'rgba(90,92,112,0.9)';
        ctx.fillText(n.label, n.x, n.y + n.radius + 11);
        ctx.textAlign = 'left';
      }
      ctx.globalAlpha = 1;
    }
    ctx.restore();
  };

  // ── Interaction ──
  const toWorld = (clientX: number, clientY: number) => {
    const r = canvas!.getBoundingClientRect();
    return { x: (clientX - r.left - offX) / scale, y: (clientY - r.top - offY) / scale };
  };
  const nodeAt = (wx: number, wy: number): P | null => {
    // Iterate in reverse so topmost drawn wins.
    for (let i = nodes.length - 1; i >= 0; i--) {
      const n = nodes[i];
      if (Math.hypot(n.x - wx, n.y - wy) <= n.radius + 4) return n;
    }
    return null;
  };

  const onDown = (e: MouseEvent) => {
    downAt = { x: e.clientX, y: e.clientY };
    moved = false;
    const { x, y } = toWorld(e.clientX, e.clientY);
    const hit = nodeAt(x, y);
    if (hit) {
      dragNode = hit;
    } else {
      panning = true;
      panStart = { x: e.clientX, y: e.clientY, ox: offX, oy: offY };
    }
  };
  const onMove = (e: MouseEvent) => {
    if (Math.abs(e.clientX - downAt.x) + Math.abs(e.clientY - downAt.y) > 3) moved = true;
    if (dragNode) {
      const { x, y } = toWorld(e.clientX, e.clientY);
      dragNode.x = x;
      dragNode.y = y;
      dragNode.vx = 0;
      dragNode.vy = 0;
      posById.set(dragNode.id, { x, y });
      reheat();
    } else if (panning) {
      offX = panStart.ox + (e.clientX - panStart.x);
      offY = panStart.oy + (e.clientY - panStart.y);
    } else {
      const { x, y } = toWorld(e.clientX, e.clientY);
      hover = nodeAt(x, y);
      if (canvas) canvas.style.cursor = hover ? 'pointer' : 'grab';
    }
  };
  const onUp = (e: MouseEvent) => {
    if (!moved) {
      const { x, y } = toWorld(e.clientX, e.clientY);
      const hit = nodeAt(x, y);
      props.onSelect?.(hit ? hit.id : null);
    }
    dragNode = null;
    panning = false;
  };
  const onLeave = () => {
    dragNode = null;
    panning = false;
    hover = null;
  };
  const onWheel = (e: WheelEvent) => {
    e.preventDefault();
    const r = canvas!.getBoundingClientRect();
    const mx = e.clientX - r.left;
    const my = e.clientY - r.top;
    const factor = e.deltaY < 0 ? 1.12 : 1 / 1.12;
    const next = Math.max(0.25, Math.min(4, scale * factor));
    offX = mx - ((mx - offX) * next) / scale;
    offY = my - ((my - offY) * next) / scale;
    scale = next;
  };

  const zoomBy = (factor: number) => {
    const cx = W / 2;
    const cy = H / 2;
    const next = Math.max(0.25, Math.min(4, scale * factor));
    offX = cx - ((cx - offX) * next) / scale;
    offY = cy - ((cy - offY) * next) / scale;
    scale = next;
  };
  const resetView = () => {
    scale = 1;
    offX = 0;
    offY = 0;
    reheat();
  };

  onMount(() => {
    if (!canvas) return;
    ctx = canvas.getContext('2d');
    canvas.style.cursor = 'grab';
    requestAnimationFrame(() => {
      resize();
      ready = true;
      rebuild();
      raf = requestAnimationFrame(tick);
    });

    const ro = new ResizeObserver(() => resize());
    if (canvas.parentElement) ro.observe(canvas.parentElement);

    canvas.addEventListener('mousedown', onDown);
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
    canvas.addEventListener('mouseleave', onLeave);
    canvas.addEventListener('wheel', onWheel, { passive: false });

    onCleanup(() => {
      if (raf) cancelAnimationFrame(raf);
      ro.disconnect();
      canvas?.removeEventListener('mousedown', onDown);
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
      canvas?.removeEventListener('mouseleave', onLeave);
      canvas?.removeEventListener('wheel', onWheel);
    });
  });

  // Rebuild whenever the incoming nodes/edges change.
  createEffect(() => {
    // Track props reactively.
    props.nodes;
    props.edges;
    if (ready) rebuild();
  });

  return (
    <>
      <canvas ref={canvas} class="fg-canvas" />
      <div class="fg-controls">
        <button class="fg-ctrl" title="Zoom in" onClick={() => zoomBy(1.2)}><ZoomIn size={15} /></button>
        <button class="fg-ctrl" title="Zoom out" onClick={() => zoomBy(1 / 1.2)}><ZoomOut size={15} /></button>
        <button class="fg-ctrl" title="Reset view" onClick={resetView}><RotateCcw size={15} /></button>
      </div>
    </>
  );
};
