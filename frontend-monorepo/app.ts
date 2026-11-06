import type { Task, FilterStatus } from './types.js';

// CONFIG & STATE

const API_URL = 'http://localhost:8083'; // Connects to your Go server

let tasks: Task[] = [];
let currentFilter: FilterStatus = 'all';
const expandedTaskIds = new Set<string>();
let calendarInstance: any = null;

// HELPERS

const $ = <T extends HTMLElement>(id: string) => document.getElementById(id) as T | null;
const $input = (id: string) => $(id) as HTMLInputElement | null;
const $query = <T extends HTMLElement>(sel: string) => document.querySelector(sel) as T | null;

function escapeHTML(str: string): string {
  return str.replace(/[&<>'"]/g, tag => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;',
    "'": '&#39;', '"': '&quot;'
  }[tag] || tag));
}

function formatTime12(date: Date): string {
  let hours = date.getHours();
  const ampm = hours >= 12 ? 'PM' : 'AM';
  hours = hours % 12 || 12;
  const minutes = date.getMinutes().toString().padStart(2, '0');
  return `${hours}:${minutes} ${ampm}`;
}

function isTaskLate(task: Task): boolean {
  if (task.completed) return false;
  return new Date(`${task.date}T${task.time}`) < new Date();
}

function getTodayStr(): string { return new Date().toISOString().split('T')[0]; }
function getNowTime(): string { return new Date().toTimeString().slice(0, 5); }

// API COMMUNICATION (Replaces localStorage)

// Fetch all tasks from Firebase via Go backend
async function fetchTodos(): Promise<void> {
  try {
    const res = await fetch(`${API_URL}/api/todos`);
    const data = await res.json();
    const list = Array.isArray(data) ? data : [];  // guards against null
    
    tasks = list.map((t: any) => ({
      id: t.id,
      text: t.text,
      date: t.date,
      time: t.time,
      completed: t.done,
      subtasks: []
    }));
    
    renderTasks();
    refreshCalendar();
  } catch (err) {
    console.error("Failed to fetch tasks:", err);
  }
}

// Helper to send requests and then refresh the list
async function apiRequest(url: string, method: string, body: object): Promise<boolean> {
  try {
    const res = await fetch(`${API_URL}${url}`, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    if (res.ok) {
      await fetchTodos(); // Refresh list after successful change
      return true;
    }
    return false;
  } catch (err) {
    console.error("API Error:", err);
    return false;
  }
}

// ICONS

const svg = (paths: string, size = 14) =>
  `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${paths}</svg>`;

const ICONS = {
  calendar: svg(`<rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/>`),
  clock: svg(`<circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>`),
  edit: svg(`<path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>`),
  trash: svg(`<polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/>`),
  chevronDown: svg(`<polyline points="6 9 12 15 18 9"/>`),
  chevronUp: svg(`<polyline points="18 15 12 9 6 15"/>`),
  close: svg(`<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>`, 12),
};

// FULLCALENDAR

declare global { interface Window { FullCalendar: any; } }

function getCalendarEvents() {
  return tasks.map(task => ({
    id: task.id,
    title: task.text,
    start: `${task.date}T${task.time}:00`,
    backgroundColor: task.completed ? '#e8f8f0' : '#4aba82',
    borderColor: 'transparent',
    textColor: task.completed ? '#7a8a80' : '#ffffff',
  }));
}

function initCalendar(): void {
  const calendarEl = $('calendar');
  if (!calendarEl) return;
  calendarInstance = new window.FullCalendar.Calendar(calendarEl, {
    initialView: 'dayGridMonth',
    slotLabelFormat: { hour: 'numeric', minute: '2-digit', meridiem: true },
    dayMaxEvents: 2,
    headerToolbar: { left: 'prev,next today', center: 'title', right: '' },
    height: 'auto',
    editable: true,
    events: getCalendarEvents(),
    eventContent: (arg: any) => ({
      html: `<div class="custom-event-wrapper">
        <span class="custom-event-title">${arg.event.title}</span>
        <span class="custom-event-time">${formatTime12(arg.event.start)}</span>
      </div>`
    }),
    eventClick: (info: any) => openModal(info.event.id),
    eventDrop: async (info: any) => {
      const taskId = info.event.id;
      const newStart = info.event.start;
      const newDate = newStart.toISOString().split('T')[0];
      const newTime = newStart.toTimeString().slice(0, 5);
      
      // Update via API
      const success = await apiRequest('/api/todo/update', 'PUT', {
        id: taskId, text: info.event.title, date: newDate, time: newTime
      });
      
      if (!success) info.revert(); // Revert UI if API failed
    }
  });
  calendarInstance.render();
}

function refreshCalendar(): void {
  if (calendarInstance) {
    calendarInstance.removeAllEvents();
    calendarInstance.addEventSource(getCalendarEvents());
  }
}

function toggleCalendarView(): void {
  const panel = $('calendarPanel');
  const grid = $query<HTMLElement>('.dashboard-grid');
  const btn = $('calendarToggleBtn');
  if (!panel || !grid || !btn) return;
  const isVisible = panel.style.visibility === 'visible';
  panel.style.visibility = isVisible ? 'hidden' : 'visible';
  panel.style.position = isVisible ? 'absolute' : 'relative';
  grid.classList.toggle('calendar-active', !isVisible);
  btn.innerText = isVisible ? '📅 Show Calendar' : '❌ Hide Calendar';
  if (!isVisible && calendarInstance) setTimeout(() => calendarInstance.updateSize(), 50);
}

// TASK OPERATIONS (Now Async & API-Driven)

async function toggleTask(id: string): Promise<void> {
  await apiRequest('/api/todo/toggle', 'PATCH', { id });
}

async function deleteTask(id: string): Promise<void> {
  expandedTaskIds.delete(id);
  await apiRequest('/api/todo/delete', 'DELETE', { id });
}

function toggleExpand(id: string): void {
  if (expandedTaskIds.has(id)) expandedTaskIds.delete(id);
  else expandedTaskIds.add(id);
  renderTasks();
}

// MODALS
function setModalDisplay(id: string, show: boolean): void {
  const modal = $(id);
  if (modal) modal.style.setProperty('display', show ? 'flex' : 'none', 'important');
}

function openModal(id: string): void {
  const task = tasks.find(t => t.id === id);
  if (!task) return;
  $input('editId')!.value = task.id;
  $input('editText')!.value = task.text;
  $input('editDate')!.value = task.date;
  $input('editTime')!.value = task.time;
  setModalDisplay('editModal', true);
}

function closeModal(): void { setModalDisplay('editModal', false); }

function openRescheduleModal(id: string): void {
  const task = tasks.find(t => t.id === id);
  if (!task) return;
  $input('rescheduleId')!.value = task.id;
  const titleEl = $('rescheduleTaskTitle');
  if (titleEl) titleEl.innerText = `Task: "${task.text}"`;
  $input('rescheduleDate')!.value = task.date;
  $input('rescheduleTime')!.value = task.time;
  setModalDisplay('rescheduleModal', true);
}

function closeRescheduleModal(): void { setModalDisplay('rescheduleModal', false); }

// RENDERING & STATS

function setFilter(filter: FilterStatus): void {
  currentFilter = filter;
  document.querySelectorAll('.filter-btn').forEach(btn => btn.classList.remove('active'));
  $query(`.filter-btn[data-filter="${filter}"]`)?.classList.add('active');
  renderTasks();
}

function updateStats(total: number, completed: number): void {
  const percent = total === 0 ? 0 : Math.round((completed / total) * 100);
  const active = total - completed;
  const percentEl = $('statsPercent');
  const barEl = $('progressBarFill');
  const countEl = $('statsCountText');
  if (percentEl) percentEl.innerText = `${percent}%`;
  if (barEl) barEl.style.width = `${percent}%`;
  if (countEl) countEl.innerText = `${active} Active / ${total} Total`;
}

function renderTaskItem(task: Task): string {
  const isLate = isTaskLate(task);
  const isExpanded = expandedTaskIds.has(task.id);

  let html = `
    <div class="task-main-row">
      <div class="task-left">
        <input type="checkbox" class="task-checkbox" ${task.completed ? 'checked' : ''}>
        <div>
          <span class="task-title ${task.completed ? 'completed' : ''}">${escapeHTML(task.text)}</span>
          <div class="task-meta">
            <span class="meta-item">${ICONS.calendar} ${task.date}</span>
            <span class="meta-item">${ICONS.clock} ${task.time}</span>
            ${isLate ? `<button class="btn-reschedule-inline" data-action="reschedule" data-id="${task.id}">RESCHEDULE</button>` : ''}
          </div>
        </div>
      </div>
      <div class="task-actions">
        <button class="action-btn" data-action="edit" data-id="${task.id}" title="Edit">${ICONS.edit}</button>
        <button class="action-btn" data-action="delete" data-id="${task.id}" title="Delete">${ICONS.trash}</button>
        <button class="action-btn" data-action="expand" data-id="${task.id}" title="Expand/Collapse">${isExpanded ? ICONS.chevronUp : ICONS.chevronDown}</button>
      </div>
    </div>`;

  if (isExpanded) {
    html += `<div class="subtasks-section"><ul class="subtask-list"><li style="font-size:12px;color:var(--text-muted);font-style:italic;">No subtasks yet.</li></ul></div>`;
  }

  return html;
}

function renderTasks(): void {
  const taskList = $('taskList');
  const emptyState = $('emptyState');
  if (!taskList || !emptyState) return;
  taskList.innerHTML = '';

  const filtered = tasks.filter(t => {
    if (currentFilter === 'all') return true;
    if (currentFilter === 'active') return !t.completed;
    return t.completed;
  });

  emptyState.style.display = filtered.length === 0 ? 'block' : 'none';

  const todayTasks = tasks.filter(t => t.date === getTodayStr());
  const completedToday = todayTasks.filter(t => t.completed).length;
  updateStats(todayTasks.length, completedToday);

  filtered.forEach(task => {
    const li = document.createElement('li');
    li.className = `todo-item ${isTaskLate(task) ? 'late' : ''}`;
    li.setAttribute('data-id', task.id);
    li.innerHTML = renderTaskItem(task);
    taskList.appendChild(li);
  });
}

// EVENT LISTENERS (Now Async)

function setupFormListeners(): void {
  const addForm = $('addForm');
  const editForm = $('editForm');
  const rescheduleForm = $('rescheduleForm');

  addForm?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target as HTMLFormElement);
    const success = await apiRequest('/api/todo/add', 'POST', {
      text: fd.get('task') as string,
      date: fd.get('date') as string,
      time: fd.get('time') as string
    });
    if (success) {
      (e.target as HTMLFormElement).reset();
      $query<HTMLInputElement>('input[name="date"]')!.value = getTodayStr();
      $query<HTMLInputElement>("input[name='time']")!.value = getNowTime();
    }
  });

  editForm?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const id = $input('editId')?.value || '';
    const text = $input('editText')?.value || '';
    const date = $input('editDate')?.value || '';
    const time = $input('editTime')?.value || '';
    const success = await apiRequest('/api/todo/update', 'PUT', { id, text, date, time });
    if (success) closeModal();
  });

  rescheduleForm?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const id = $input('rescheduleId')?.value || '';
    const text = tasks.find(t => t.id === id)?.text || ''; // Need existing text to update
    const date = $input('rescheduleDate')?.value || '';
    const time = $input('rescheduleTime')?.value || '';
    const success = await apiRequest('/api/todo/update', 'PUT', { id, text, date, time });
    if (success) closeRescheduleModal();
  });

  $('closeEditBtn')?.addEventListener('click', closeModal);
  $('closeReschedBtn')?.addEventListener('click', closeRescheduleModal);
}

function setupClickDelegation(): void {
  document.body.addEventListener('click', (e) => {
    const target = e.target as HTMLElement;

    const filterBtn = target.closest<HTMLElement>('[data-filter]');
    if (filterBtn?.dataset.filter) { setFilter(filterBtn.dataset.filter as FilterStatus); return; }

    const calToggle = $('calendarToggleBtn');
    if (calToggle && (target === calToggle || calToggle.contains(target))) { toggleCalendarView(); return; }

    if (target instanceof HTMLInputElement && target.classList.contains('task-checkbox')) {
      const li = target.closest('li.todo-item');
      if (!li) return;
      toggleTask(li.getAttribute('data-id')!);
      e.preventDefault();
      return;
    }

    const actionBtn = target.closest<HTMLElement>('[data-action][data-id]');
    if (!actionBtn) return;
    const id = actionBtn.getAttribute('data-id')!;
    const action = actionBtn.getAttribute('data-action')!;

    switch (action) {
      case 'edit': openModal(id); break;
      case 'delete': deleteTask(id); break;
      case 'expand': toggleExpand(id); break;
      case 'reschedule': openRescheduleModal(id); break;
    }
  });
}

// INIT

async function init(): Promise<void> {
  const dateInput = $query<HTMLInputElement>('input[name="date"]');
  const timeInput = $query<HTMLInputElement>("input[name='time']");
  if (dateInput) dateInput.value = getTodayStr();
  if (timeInput) timeInput.value = getNowTime();

  await fetchTodos(); // Fetch from Firebase instead of localStorage
  setupFormListeners();
  setupClickDelegation();
  initCalendar();
}

document.addEventListener('DOMContentLoaded', init);