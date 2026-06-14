import { Component, For, createResource, Show, createEffect, createSignal } from 'solid-js';
import { Check } from 'lucide-solid';
import { listTasks, updateTask, Task } from '../../api/client';
import './TodaysTasks.css';

export const TodaysTasks: Component = () => {
  const [serverTasks, { mutate }] = createResource(() => listTasks());
  const [tasks, setTasks] = createSignal<Task[]>([]);

  createEffect(() => {
    if (serverTasks()) {
      // Filter for "today's" tasks (in a real app, this would check due_date, but for now just show top active/recent tasks)
      setTasks(serverTasks()!.slice(0, 5));
    }
  });

  const completed = () => tasks().filter(t => t.status === 'done').length;
  const total = () => tasks().length;

  const toggleTask = async (id: string, currentStatus: string) => {
    const newStatus = currentStatus === 'done' ? 'todo' : 'done';
    
    // Optimistic UI update
    setTasks(prev => prev.map(t => t.id === id ? { ...t, status: newStatus } : t));
    
    try {
      await updateTask(id, { status: newStatus });
    } catch (err) {
      console.error("Failed to update task", err);
      // Revert on failure
      setTasks(prev => prev.map(t => t.id === id ? { ...t, status: currentStatus } : t));
    }
  };

  // SVG ring calculations
  const radius = 16;
  const circumference = 2 * Math.PI * radius;
  const progress = () => completed() / Math.max(total(), 1);
  const dashOffset = () => circumference * (1 - progress());

  return (
    <div class="todays-tasks">
      <div class="todays-tasks-header">Today's Tasks</div>
      
      <Show when={!serverTasks.loading && tasks().length === 0}>
        <div style={{ padding: '16px', "text-align": 'center', color: 'var(--text-muted)' }}>
          No tasks for today. You're all caught up!
        </div>
      </Show>

      <div class="todays-tasks-list">
        <For each={tasks()}>
          {(task) => {
            const isDone = task.status === 'done';
            return (
              <div class="todays-task-row">
                <div
                  class={`todays-task-checkbox ${isDone ? 'checked' : ''}`}
                  onClick={() => toggleTask(task.id, task.status)}
                >
                  {isDone && <Check size={10} />}
                </div>
                <span class={`todays-task-title ${isDone ? 'completed' : ''}`}>
                  {task.title}
                </span>
                <span class="todays-task-project">{task.project || 'Global'}</span>
              </div>
            );
          }}
        </For>
      </div>
      <div class="todays-tasks-footer">
        <div class="todays-tasks-ring">
          <svg width="40" height="40" viewBox="0 0 40 40">
            <circle
              class="todays-tasks-ring-bg"
              cx="20" cy="20" r={radius}
            />
            <circle
              class="todays-tasks-ring-fill"
              cx="20" cy="20" r={radius}
              stroke-dasharray={circumference}
              stroke-dashoffset={dashOffset()}
            />
          </svg>
        </div>
        <div class="todays-tasks-counter">
          <span class="todays-tasks-count-number">{completed()}/{Math.max(total(), 1)}</span>
          <span class="todays-tasks-count-label">Tasks Done</span>
        </div>
      </div>
    </div>
  );
};
