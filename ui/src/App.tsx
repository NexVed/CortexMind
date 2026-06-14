import { Component, Show } from 'solid-js';
import { Route, Router, useNavigate } from '@solidjs/router';
import { AuthProvider, useAuth } from './api/auth';
import { AppLayout } from './layouts/AppLayout';
import { LoginPage } from './pages/Login/Login';
import { DashboardMain } from './pages/Dashboard/Dashboard';
import { ProjectsPage } from './pages/Projects/Projects';
import { VaultsPage } from './pages/Vaults/Vaults';
import { TasksPage } from './pages/Tasks/Tasks';
import { HandoffsPage } from './pages/Handoffs/Handoffs';
import { SearchPage } from './pages/Search/Search';
import { GraphPage } from './pages/Graph/Graph';
import { AIContextPage } from './pages/AIContext/AIContext';
import { MCPServerPage } from './pages/MCPServer/MCPServer';
import { SettingsPage } from './pages/Settings/Settings';

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
            <span style={{ 'font-size': '13px' }}>Loading CORTEX...</span>
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

const App: Component = () => {
  return (
    <AuthProvider>
      <Router>
        <Route path="/login" component={LoginPage} />
        <Route path="/" component={DashboardPage} />
        <Route path="/projects" component={() => <FullWidthPage><ProjectsPage /></FullWidthPage>} />
        <Route path="/vaults" component={() => <FullWidthPage><VaultsPage /></FullWidthPage>} />
        <Route path="/tasks" component={() => <FullWidthPage><TasksPage /></FullWidthPage>} />
        <Route path="/handoffs" component={() => <FullWidthPage><HandoffsPage /></FullWidthPage>} />
        <Route path="/graph" component={() => <FullWidthPage><GraphPage /></FullWidthPage>} />
        <Route path="/ai-context" component={() => <FullWidthPage><AIContextPage /></FullWidthPage>} />
        <Route path="/mcp-server" component={() => <FullWidthPage><MCPServerPage /></FullWidthPage>} />
        <Route path="/settings" component={() => <FullWidthPage><SettingsPage /></FullWidthPage>} />
      </Router>
    </AuthProvider>
  );
};

export default App;
