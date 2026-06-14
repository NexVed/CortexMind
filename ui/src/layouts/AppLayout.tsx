import { Component, JSX } from 'solid-js';
import { Sidebar } from '../components/Sidebar/Sidebar';
import { TopBar } from '../components/TopBar/TopBar';
import './AppLayout.css';

interface AppLayoutProps {
  children: JSX.Element;
  rightPanel?: JSX.Element;
}

export const AppLayout: Component<AppLayoutProps> = (props) => {
  return (
    <div class="app-layout">
      <Sidebar />
      <div class="app-main-wrapper">
        <TopBar />
        <div class="app-content-area">
          <div class="app-page-card">
            <main class="app-main-content">
              {props.children}
            </main>
            {props.rightPanel && (
              <aside class="app-right-panel">
                {props.rightPanel}
              </aside>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
