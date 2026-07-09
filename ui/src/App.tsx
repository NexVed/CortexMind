import { Component, Show } from 'solid-js';
import { Route, Router, useNavigate } from '@solidjs/router';
import { QueryClientProvider } from '@tanstack/solid-query';
import { queryClient } from './api/queryClient';
import { AuthProvider, useAuth } from './api/auth';
import { AppLayout } from './layouts/AppLayout';
import { GlobalShortcuts } from './components/GlobalShortcuts';
import { initSettings } from './api/settings';
import { LoginPage } from './pages/Login/Login';
import { DashboardMain } from './pages/Dashboard/Dashboard';
import { ProjectsPage } from './pages/Projects/Projects';
import { VaultsPage } from './pages/Vaults/Vaults';
import { TasksPage } from './pages/Tasks/Tasks';
import { HandoffsPage } from './pages/Handoffs/Handoffs';
import { SearchPage } from './pages/Search/Search';
import { AIContextPage } from './pages/AIContext/AIContext';
import { MCPServerPage } from './pages/MCPServer/MCPServer';
import { SessionDigestsPage } from './pages/SessionDigests/SessionDigests';
import { CodeGraphPage } from './pages/CodeGraph/CodeGraph';
import { AgentMemoryPage } from './pages/AgentMemory/AgentMemory';
import { SettingsPage } from './pages/Settings/Settings';
import { RepositoryPage } from './pages/Repository/Repository';

// ── Route Guard ────────────────────────────────────────

const ProtectedRoute: Component<{ children: any }> = (props) => {
  const { isAuthenticated, isLoading } = useAuth();
  const navigate = useNavigate();

  // While checking session, show a loading shimmer
  return (
    <Show
      when={!isLoading()}
      fallback={
        <div style={{
          display: 'flex',
          'align-items': 'center',
          'justify-content': 'center',
          height: '100vh',
          width: '100vw',
          background: 'var(--bg-base)',
        }}>
          <div style={{
            display: 'flex',
            'flex-direction': 'column',
            'align-items': 'center',
            gap: '16px',
            color: 'var(--text-muted)',
            'font-family': 'var(--font-display)',
          }}>
            <div style={{
              width: '48px',
              height: '48px',
              'border-radius': '14px',
              background: 'linear-gradient(135deg, var(--accent), #8B5CF6)',
              display: 'flex',
              'align-items': 'center',
              'justify-content': 'center',
              color: '#fff',
              'font-weight': '700',
              'font-size': '20px',
            }}>C</div>
            <span style={{ 'font-size': '13px' }}>Loading CortexMind...</span>
          </div>
        </div>
      }
    >
      <Show
        when={isAuthenticated()}
        fallback={(() => { navigate('/login', { replace: true }); return null; })()}
      >
        {props.children}
      </Show>
    </Show>
  );
};

// ── Page Wrappers ──────────────────────────────────────

const DashboardPage: Component = () => (
  <ProtectedRoute>
    <AppLayout>
      <DashboardMain />
    </AppLayout>
  </ProtectedRoute>
);

const FullWidthPage: Component<{ children: any }> = (props) => (
  <ProtectedRoute>
    <AppLayout>
      {props.children}
    </AppLayout>
  </ProtectedRoute>
);

// ── App ────────────────────────────────────────────────

// RootShell wraps every route: it applies persisted UI preferences and mounts
// the global keyboard shortcut handler (which needs Router context).
const RootShell: Component<{ children?: any }> = (props) => {
  initSettings();
  return (
    <>
      <GlobalShortcuts />
      {props.children}
    </>
  );
};

const App: Component = () => {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <Router root={RootShell}>
          <Route path="/login" component={LoginPage} />
          <Route path="/" component={DashboardPage} />
          <Route path="/projects" component={() => <FullWidthPage><ProjectsPage /></FullWidthPage>} />
          <Route path="/vaults" component={() => <FullWidthPage><VaultsPage /></FullWidthPage>} />
          <Route path="/tasks" component={() => <FullWidthPage><TasksPage /></FullWidthPage>} />
          <Route path="/handoffs" component={() => <FullWidthPage><HandoffsPage /></FullWidthPage>} />
          <Route path="/ai-context" component={() => <FullWidthPage><AIContextPage /></FullWidthPage>} />
          <Route path="/digests" component={() => <FullWidthPage><SessionDigestsPage /></FullWidthPage>} />
          <Route path="/code-graph" component={() => <FullWidthPage><CodeGraphPage /></FullWidthPage>} />
          <Route path="/agent-memory" component={() => <FullWidthPage><AgentMemoryPage /></FullWidthPage>} />
          <Route path="/mcp-server" component={() => <FullWidthPage><MCPServerPage /></FullWidthPage>} />
          <Route path="/settings" component={() => <FullWidthPage><SettingsPage /></FullWidthPage>} />
          <Route path="/repository/:id" component={() => <FullWidthPage><RepositoryPage /></FullWidthPage>} />
        </Router>
      </AuthProvider>
    </QueryClientProvider>
  );
};

export default App;
