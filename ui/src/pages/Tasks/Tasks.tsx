import { Component, For, createSignal, Show } from 'solid-js';
import {
  ListTodo,
  Plus,
  Search,
  Filter,
  CheckCircle2,
  Circle,
  Clock,
  MoreHorizontal,
  Calendar,
  User,
} from 'lucide-solid';
import { useTasks, useCreateTask, useUpdateTask } from '../../api/queries';
import { Modal } from '../../components/Modal/Modal';
import './Tasks.css';

export const TasksPage: Component = () => {
  const [searchQuery, setSearchQuery] = createSignal('');
  const [activeFilter, setActiveFilter] = createSignal('all');
  const tasksQuery = useTasks();
  const createTaskM = useCreateTask();
  const updateTaskM = useUpdateTask();

  // ── Create task modal ──
  const [showCreate, setShowCreate] = createSignal(false);
  const [saving, setSaving] = createSignal(false);
  const [formErr, setFormErr] = createSignal('');
  const [fTitle, setFTitle] = createSignal('');
  const [fDesc, setFDesc] = createSignal('');
  const [fPriority, setFPriority] = createSignal('medium');

  const openCreate = () => {
    setFTitle('');
    setFDesc('');
    setFPriority('medium');
    setFormErr('');
    setShowCreate(true);
  };

  const submitCreate = async () => {
    if (!fTitle().trim()) {
      setFormErr('Task title is required.');
      return;
    }
    setSaving(true);
    setFormErr('');
    try {
      await createTaskM.mutateAsync({
        title: fTitle().trim(),
        description: fDesc().trim(),
        priority: fPriority(),
      });
      setShowCreate(false);
    } catch (err: any) {
      setFormErr(err?.message || 'Failed to create task');
    } finally {
      setSaving(false);
    }
  };

  // Derive local state
  const tasks = () => tasksQuery.data ?? [];

  const filteredTasks = () => {
    let result = tasks();
    
    // Apply status filter
    if (activeFilter() !== 'all') {
      result = result.filter(t => t.status === activeFilter());
    }

    // Apply search filter
    if (searchQuery()) {
      const q = searchQuery().toLowerCase();
      result = result.filter(t => 
        t.title.toLowerCase().includes(q) || 
        (t.description && t.description.toLowerCase().includes(q))
      );
    }
    
    return result;
  };

  const toggleTaskStatus = async (id: string, currentStatus: string) => {
    const newStatus = currentStatus === 'done' ? 'todo' : 'done';
    try {
      await updateTaskM.mutateAsync({ id, data: { status: newStatus } });
    } catch (err) {
      console.error('Failed to update task:', err);
    }
  };

  return (
    <div class="tasks-page">
      <div class="page-header">
        <div class="page-title-row">
          <div class="page-title-icon blue">
            <ListTodo size={24} />
          </div>
          <div>
            <h1 class="page-title">Tasks</h1>
            <p class="page-subtitle">Track work across all your projects</p>
          </div>
        </div>
        <div class="page-actions">
          <button class="btn primary" onClick={openCreate}>
            <Plus size={16} />
            New Task
          </button>
        </div>
      </div>

      <div class="tasks-toolbar">
        <div class="search-box">
          <Search size={16} class="search-icon" />
          <input
            type="text"
            placeholder="Search tasks..."
            value={searchQuery()}
            onInput={(e) => setSearchQuery(e.currentTarget.value)}
          />
        </div>
        <div class="tasks-filters">
          <button 
            class={`filter-btn ${activeFilter() === 'all' ? 'active' : ''}`}
            onClick={() => setActiveFilter('all')}
          >
            All
          </button>
          <button 
            class={`filter-btn ${activeFilter() === 'todo' ? 'active' : ''}`}
            onClick={() => setActiveFilter('todo')}
          >
            To Do
          </button>
          <button 
            class={`filter-btn ${activeFilter() === 'in_progress' ? 'active' : ''}`}
            onClick={() => setActiveFilter('in_progress')}
          >
            In Progress
          </button>
          <button 
            class={`filter-btn ${activeFilter() === 'done' ? 'active' : ''}`}
            onClick={() => setActiveFilter('done')}
          >
            Done
          </button>
          <button class="icon-btn">
            <Filter size={16} />
          </button>
        </div>
      </div>

      <Show when={tasksQuery.isLoading}>
        <div class="loading-state">Loading tasks...</div>
      </Show>

      <Show when={!tasksQuery.isLoading && filteredTasks().length === 0}>
        <div class="empty-state">
          <div class="empty-icon"><ListTodo size={32} /></div>
          <h3>No tasks found</h3>
          <p>You're all caught up! Create a new task to get started.</p>
        </div>
      </Show>

      <div class="tasks-list">
        <For each={filteredTasks()}>
          {(task) => {
            const isDone = task.status === 'done';
            
            return (
              <div class={`task-item ${isDone ? 'completed' : ''}`}>
                <button 
                  class="task-checkbox"
                  onClick={() => toggleTaskStatus(task.id, task.status)}
                >
                  <Show when={isDone} fallback={<Circle size={20} />}>
                    <CheckCircle2 size={20} class="text-green" />
                  </Show>
                </button>
                
                <div class="task-content">
                  <div class="task-title-row">
                    <span class="task-title">{task.title}</span>
                    <span class={`task-priority priority-${task.priority || 'medium'}`}>
                      {task.priority || 'medium'}
                    </span>
                  </div>
                  
                  <div class="task-meta">
                    <span class="task-project">{task.project || 'Global'}</span>
                    <Show when={task.due_date}>
                      <span class="task-meta-item">
                        <Calendar size={12} />
                        {new Date(task.due_date).toLocaleDateString()}
                      </span>
                    </Show>
                    <Show when={task.assigned_to}>
                      <span class="task-meta-item">
                        <User size={12} />
                        {task.assigned_to}
                      </span>
                    </Show>
                    <Show when={task.linked_files?.length}>
                      <span class="task-meta-item">
                        {task.linked_files.length} files
                      </span>
                    </Show>
                  </div>
                </div>

                <div class="task-actions">
                  <span class="task-status">{task.status.replace('_', ' ')}</span>
                  <button class="icon-btn">
                    <MoreHorizontal size={16} />
                  </button>
                </div>
              </div>
            );
          }}
        </For>
      </div>

      <Modal open={showCreate()} title="New Task" onClose={() => setShowCreate(false)}>
        <div class="cx-field">
          <label>Title *</label>
          <input
            value={fTitle()}
            onInput={(e) => setFTitle(e.currentTarget.value)}
            placeholder="What needs to be done?"
            autofocus
          />
        </div>
        <div class="cx-field">
          <label>Description</label>
          <textarea
            value={fDesc()}
            onInput={(e) => setFDesc(e.currentTarget.value)}
            placeholder="Optional details"
          />
        </div>
        <div class="cx-field">
          <label>Priority</label>
          <select value={fPriority()} onChange={(e) => setFPriority(e.currentTarget.value)}>
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
          </select>
        </div>
        <Show when={formErr()}>
          <div class="cx-modal-error">{formErr()}</div>
        </Show>
        <div class="cx-modal-actions">
          <button class="btn secondary" onClick={() => setShowCreate(false)}>Cancel</button>
          <button class="btn primary" onClick={submitCreate} disabled={saving()}>
            {saving() ? 'Creating…' : 'Create Task'}
          </button>
        </div>
      </Modal>
    </div>
  );
};
