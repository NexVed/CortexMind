import { Component, Show, onMount } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { useAuth } from '../../api/auth';
import './Login.css';

// Inline GitHub mark to avoid an extra dependency.
const GitHubIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
    <path d="M12 0C5.374 0 0 5.373 0 12c0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23A11.509 11.509 0 0112 5.803c1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576C20.566 21.797 24 17.3 24 12c0-6.627-5.373-12-12-12z" />
  </svg>
);

const features = ['Local-first', 'Git-synced memory', 'Multi-agent context'];

export const LoginPage: Component = () => {
  const { loginWithGitHub, isAuthenticated, isLoading, error } = useAuth();
  const navigate = useNavigate();

  onMount(() => {
    if (isAuthenticated()) navigate('/', { replace: true });
  });

  const handleLogin = async () => {
    await loginWithGitHub();
    if (isAuthenticated()) navigate('/', { replace: true });
  };

  return (
    <div class="login-page">
      <main class="login-card">
        <div class="login-mark" aria-hidden="true">C</div>

        <h1 class="login-heading">Sign in to CORTEX</h1>
        <p class="login-sub">The shared brain for AI development.</p>

        <button
          class="login-github-btn"
          onClick={handleLogin}
          disabled={isLoading()}
          aria-busy={isLoading()}
        >
          <Show when={!isLoading()} fallback={<span class="login-spinner" aria-hidden="true" />}>
            <GitHubIcon />
          </Show>
          {isLoading() ? 'Connecting to GitHub…' : 'Continue with GitHub'}
        </button>

        <Show when={error()}>
          <div class="login-error" role="alert">{error()}</div>
        </Show>

        <ul class="login-features">
          {features.map((f) => (
            <li class="login-feature">
              <span class="login-feature-dot" aria-hidden="true" />
              {f}
            </li>
          ))}
        </ul>

        <p class="login-fineprint">
          By continuing, you connect your GitHub account to CORTEX for repository
          access and project intelligence.
        </p>
      </main>
    </div>
  );
};
