import { Component, Show, createSignal, onMount } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { Github, Minus, Square, X } from 'lucide-solid';
import { useAuth } from '../../api/auth';
import { desktopPlatform, emitWindowControl, isWailsDesktop } from '../../api/desktop';
import './Login.css';

const GitHubIcon = () => <Github size={18} aria-hidden="true" />;

export const LoginPage: Component = () => {
	const platform = desktopPlatform();
  const { loginWithGitHub, continueOffline, isAuthenticated, isLoading, error, deviceCode, verificationURL } = useAuth();
  const navigate = useNavigate();
  const [offlineName, setOfflineName] = createSignal('');
  const [browserError, setBrowserError] = createSignal('');

  onMount(() => {
    if (isAuthenticated()) navigate('/', { replace: true });
  });

  const go = async (action: () => Promise<void>) => {
    setBrowserError('');
    try {
      await action();
      if (isAuthenticated()) navigate('/', { replace: true });
    } catch {
      // The authentication provider renders the error.
    }
  };

  const openVerification = async () => {
    setBrowserError('');
    const url = verificationURL();
    if (!url) return;
    if (!isWailsDesktop()) {
      window.open(url, '_blank', 'noopener,noreferrer');
      return;
    }
    try {
      const response = await fetch('/api/auth/github/open-browser', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url }),
      });
      if (!response.ok) {
        const body = await response.json();
        throw new Error(body.error || 'Could not open the default browser');
      }
    } catch (err) {
      setBrowserError(err instanceof Error ? err.message : 'Could not open the default browser');
    }
  };

  return <div class="login-page">
    <Show when={isWailsDesktop()}>
      <header class="login-windowbar">
		<Show when={platform === 'darwin'}>
		  <div class="login-mac-controls" aria-label="Window controls">
			<button class="login-mac-btn close" title="Close" aria-label="Close" onClick={() => emitWindowControl('wnd:close')} />
			<button class="login-mac-btn minimise" title="Minimise" aria-label="Minimise" onClick={() => emitWindowControl('wnd:minimise')} />
			<button class="login-mac-btn maximise" title="Fullscreen" aria-label="Fullscreen" onClick={() => emitWindowControl('wnd:toggle-fullscreen')} />
		  </div>
		</Show>
        <div class="login-windowbar-brand"><img src="/logo.png" alt="" /> <span>CortexMind</span></div>
        <Show when={platform !== 'darwin'}><div class={`login-window-controls ${platform === 'linux' ? 'linux' : ''}`}>
          <button title="Minimize" aria-label="Minimize" onClick={() => emitWindowControl('wnd:minimise')}><Minus size={15} /></button>
          <button title="Maximize" aria-label="Maximize" onClick={() => emitWindowControl('wnd:toggle-maximise')}><Square size={12} /></button>
          <button class="login-window-close" title="Close" aria-label="Close" onClick={() => emitWindowControl('wnd:close')}><X size={16} /></button>
        </div>
		</Show>
      </header>
    </Show>
    <main class="login-card">
      <img src="/logowithname.png" alt="CortexMind" class="login-logo" />
      <p class="login-sub">Choose how you want to use this local workspace.</p>
      <button class="login-github-btn" onClick={() => void go(loginWithGitHub)} disabled={isLoading()}>
        <Show when={!isLoading()} fallback={<span class="login-spinner" />}><GitHubIcon /></Show>
        {isLoading() ? 'Waiting for GitHub...' : 'Connect GitHub account'}
      </button>
      <Show when={deviceCode()}>
        <div class="login-device-flow">
          <span>Enter this code on GitHub</span>
          <code>{deviceCode()}</code>
          <button type="button" onClick={() => void openVerification()}>Open GitHub verification in browser</button>
          <Show when={browserError()}><span class="login-error" role="alert">{browserError()}</span></Show>
        </div>
      </Show>
      <div class="login-divider">or stay offline</div>
      <form class="login-form" onSubmit={(event) => {
        event.preventDefault();
        void go(() => continueOffline(offlineName().trim()));
      }}>
        <input class="login-input" value={offlineName()} onInput={(event) => setOfflineName(event.currentTarget.value)}
          placeholder="Your workspace name" autocomplete="name" disabled={isLoading()} />
        <button class="login-local-btn" type="submit" disabled={isLoading() || !offlineName().trim()}>Create offline workspace</button>
      </form>
      <Show when={error()}><div class="login-error" role="alert">{error()}</div></Show>
    </main>
  </div>;
};
