import { Component, Show, onMount, onCleanup } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { useAuth } from '../../api/auth';
import './Login.css';

// Simple inline GitHub SVG icon to avoid extra dependency
const GitHubIcon = () => (
  <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
    <path d="M12 0C5.374 0 0 5.373 0 12c0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23A11.509 11.509 0 0112 5.803c1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576C20.566 21.797 24 17.3 24 12c0-6.627-5.373-12-12-12z" />
  </svg>
);

export const LoginPage: Component = () => {
  const { loginWithGitHub, isAuthenticated, isLoading, error } = useAuth();
  const navigate = useNavigate();
  let canvasRef: HTMLCanvasElement | undefined;
  let animId: number;

  // Redirect if already logged in
  onMount(() => {
    if (isAuthenticated()) {
      navigate('/', { replace: true });
    }
  });

  // Animated background nodes
  onMount(() => {
    if (!canvasRef) return;
    const ctx = canvasRef.getContext('2d');
    if (!ctx) return;

    const resize = () => {
      canvasRef!.width = window.innerWidth * 2;
      canvasRef!.height = window.innerHeight * 2;
      ctx.scale(2, 2);
    };
    resize();

    const w = () => window.innerWidth;
    const h = () => window.innerHeight;

    interface Node {
      x: number;
      y: number;
      vx: number;
      vy: number;
      r: number;
      color: string;
    }

    const COLORS = ['#6C63FF', '#3B82F6', '#22C55E', '#A855F7', '#F97316', '#F59E0B'];
    const nodes: Node[] = [];
    for (let i = 0; i < 30; i++) {
      nodes.push({
        x: Math.random() * w(),
        y: Math.random() * h(),
        vx: (Math.random() - 0.5) * 0.3,
        vy: (Math.random() - 0.5) * 0.3,
        r: 2 + Math.random() * 3,
        color: COLORS[i % COLORS.length],
      });
    }

    const edges: [number, number][] = [];
    for (let i = 0; i < nodes.length; i++) {
      for (let j = i + 1; j < nodes.length; j++) {
        if (Math.random() < 0.15) edges.push([i, j]);
      }
    }

    function draw() {
      ctx!.clearRect(0, 0, w(), h());

      for (const [i, j] of edges) {
        const a = nodes[i],
          b = nodes[j];
        const d = Math.hypot(a.x - b.x, a.y - b.y);
        if (d < 180) {
          ctx!.beginPath();
          ctx!.moveTo(a.x, a.y);
          ctx!.lineTo(b.x, b.y);
          ctx!.strokeStyle = `rgba(108, 99, 255, ${0.06 * (1 - d / 180)})`;
          ctx!.lineWidth = 1;
          ctx!.stroke();
        }
      }

      for (const node of nodes) {
        ctx!.beginPath();
        ctx!.arc(node.x, node.y, node.r, 0, Math.PI * 2);
        ctx!.fillStyle = node.color + '30';
        ctx!.fill();
        ctx!.beginPath();
        ctx!.arc(node.x, node.y, node.r * 0.4, 0, Math.PI * 2);
        ctx!.fillStyle = node.color + '80';
        ctx!.fill();

        node.x += node.vx;
        node.y += node.vy;
        if (node.x < 0 || node.x > w()) node.vx *= -1;
        if (node.y < 0 || node.y > h()) node.vy *= -1;
      }

      animId = requestAnimationFrame(draw);
    }
    draw();
  });

  onCleanup(() => {
    if (animId) cancelAnimationFrame(animId);
  });

  const handleLogin = async () => {
    await loginWithGitHub();
    // After successful login, redirect to dashboard
    if (isAuthenticated()) {
      navigate('/', { replace: true });
    }
  };

  return (
    <div class="login-page">
      {/* Animated gradient orbs */}
      <div class="login-glow-orb" />
      <div class="login-glow-orb" />
      <div class="login-glow-orb" />

      {/* Animated graph background */}
      <canvas ref={canvasRef} class="login-bg-canvas" />

      {/* Glass card */}
      <div class="login-card">
        <div class="login-logo">C</div>
        <div class="login-title">CORTEX</div>
        <div class="login-subtitle">
          The Shared Brain For <strong>AI Development</strong>
        </div>

        <div class="login-divider" />

        <button
          class={`login-github-btn ${isLoading() ? 'loading' : ''}`}
          onClick={handleLogin}
          disabled={isLoading()}
        >
          <Show when={!isLoading()} fallback={<div class="login-spinner" />}>
            <GitHubIcon />
          </Show>
          {isLoading() ? 'Connecting to GitHub...' : 'Sign in with GitHub'}
        </button>

        <Show when={error()}>
          <div class="login-error">{error()}</div>
        </Show>

        <div class="login-features">
          <div class="login-feature">
            <div class="login-feature-dot" />
            Local-first
          </div>
          <div class="login-feature">
            <div class="login-feature-dot" />
            Git-synced memory
          </div>
          <div class="login-feature">
            <div class="login-feature-dot" />
            Multi-agent context
          </div>
        </div>

        <div class="login-footer">
          By signing in, you connect your GitHub account to CORTEX
          <br />
          for repository access and project intelligence.
        </div>
      </div>
    </div>
  );
};
