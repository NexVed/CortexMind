import { Component } from 'solid-js';

// The new dashboard design integrates all panels into the main grid layout.
// This right panel is kept for structural compatibility with AppLayout
// but is now intentionally empty. All content (activity feed, agents, graph,
// quick actions, timeline) is rendered in DashboardMain's card grid.
export const DashboardRightPanel: Component = () => {
  return null;
};
