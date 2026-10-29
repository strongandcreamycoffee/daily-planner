// State Management
let tasks = JSON.parse(localStorage.getItem('daily_planner_tasks')) || [
    { id: '1', text: 'Break', date: '2026-10-22', time: '12:00', completed: true, subtasks: [] },
    { id: '2', text: 'test toggle', date: '2026-10-22', time: '11:04', completed: true, subtasks: [] },
    { id: '3', text: 'EOD', date: '2026-10-22', time: '17:10', completed: true, subtasks: [] },
    { id: '4', text: 'marvin ppt', date: '2026-10-28', time: '09:12', completed: false, subtasks: [
        { id: 's1', text: 'Gather slides', completed: true },
        { id: 's2', text: 'Review metrics', completed: false }
    ]}
];

let currentFilter = 'all';
let expandedTaskIds = new Set(); // Tracks expanded cards

// Professional SVG Icon Library Constants
const ICONS = {
    calendar: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect><line x1="16" y1="2" x2="16" y2="6"></line><line x1="8" y1="2" x2="8" y2="6"></line><line x1="3" y1="10" x2="21" y2="10"></line></svg>`,
    clock: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg>`,
    edit: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>`,
    trash: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path><line x1="10" y1="11" x2="10" y2="17"></line><line x1="14" y1="11" x2="14" y2="17"></line></svg>`,
    chevronDown: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>`,
    chevronUp: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="18 15 12 9 6 15"></polyline></svg>`,
    close: `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>`
};

// Initialize App
document.addEventListener('DOMContentLoaded', () => {
    // Set default date/time in add form to current local time
    const now = new Date();
    const todayStr = now.toISOString().split('T')[0];
    const timeStr = now.toTimeString().slice(0, 5);

    document.querySelector('input[name="date"]').value = todayStr;
    document.querySelector('input[name="time"]').value = timeStr;

    renderTasks();
    setupEventListeners();
});

function saveAndRefresh() {
    localStorage.setItem('daily_planner_tasks', JSON.stringify(tasks));
    renderTasks();
}

function setupEventListeners() {
    // Add Form Submit
    document.getElementById('addForm').addEventListener('submit', (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);

        const newTask = {
            id: Date.now().toString(),
            text: formData.get('task'),
            date: formData.get('date'),
            time: formData.get('time'),
            completed: false,
            subtasks: []
        };

        tasks.push(newTask);
        e.target.reset();

        // Reset default date/time again
        const now = new Date();
        document.querySelector('input[name="date"]').value = now.toISOString().split('T')[0];
        document.querySelector('input[name="time"]').value = now.toTimeString().slice(0, 5);

        saveAndRefresh();
    });

    // Edit Form Submit
    document.getElementById('editForm').addEventListener('submit', (e) => {
        e.preventDefault();
        const id = document.getElementById('editId').value;
        const text = document.getElementById('editText').value;
        const date = document.getElementById('editDate').value;
        const time = document.getElementById('editTime').value;

        const task = tasks.find(t => t.id === id);
        if (task) {
            task.text = text;
            task.date = date;
            task.time = time;
            closeModal();
            saveAndRefresh();
        }
    });

    // Reschedule Form Submit
    document.getElementById('rescheduleForm').addEventListener('submit', (e) => {
        e.preventDefault();
        const id = document.getElementById('rescheduleId').value;
        const date = document.getElementById('rescheduleDate').value;
        const time = document.getElementById('rescheduleTime').value;

        const task = tasks.find(t => t.id === id);
        if (task) {
            task.date = date;
            task.time = time;
            closeRescheduleModal();
            saveAndRefresh();
        }
    });
}

// Filter Setter
function setFilter(filter) {
    currentFilter = filter;
    document.querySelectorAll('.filter-btn').forEach(btn => btn.classList.remove('active'));
    event.target.classList.add('active');
    renderTasks();
}

// Check if task is late
function checkIfLate(task) {
    if (task.completed) return false;
    const taskDateTime = new Date(`${task.date}T${task.time}`);
    const now = new Date();
    return taskDateTime < now;
}

// Render Tasks & Progress Bar
function renderTasks() {
    const taskList = document.getElementById('taskList');
    const emptyState = document.getElementById('emptyState');
    taskList.innerHTML = '';

    // Filter tasks
    const filteredTasks = tasks.filter(task => {
        if (currentFilter === 'all' || currentFilter === 'active') return !task.completed;
        if (currentFilter === 'completed') return task.completed;
        return false;
    });

    if (filteredTasks.length === 0) {
        emptyState.style.display = 'block';
    } else {
        emptyState.style.display = 'none';
    }

    // Calculate Today's Progress Stats
    const todayStr = new Date().toISOString().split('T')[0];
    const todaysTasks = tasks.filter(t => t.date === todayStr);
    const totalToday = todaysTasks.length;
    const completedToday = todaysTasks.filter(t => t.completed).length;
    const activeToday = totalToday - completedToday;
    const progressPercent = totalToday === 0 ? 0 : Math.round((completedToday / totalToday) * 100);

    // Update Header Progress UI
    document.getElementById('statsPercent').innerText = `${progressPercent}%`;
    document.getElementById('progressBarFill').style.width = `${progressPercent}%`;
    document.getElementById('statsCountText').innerText = `${activeToday} Active / ${totalToday} Total`;

    // Render each task card
    filteredTasks.forEach(task => {
        const isLate = checkIfLate(task);
        const isExpanded = expandedTaskIds.has(task.id);

        const li = document.createElement('li');
        li.className = `todo-item ${isLate ? 'late' : ''}`;

        // Main Row HTML with Professional SVG Icons (Zero Emojis)
        let html = `
            <div class="task-main-row">
                <div class="task-left">
                    <input type="checkbox" class="task-checkbox" ${task.completed ? 'checked' : ''} onclick="toggleTask('${task.id}')">
                    <div>
                        <span class="task-title ${task.completed ? 'completed' : ''}">${escapeHTML(task.text)}</span>
                        <div class="task-meta">
                            <span class="meta-item">${ICONS.calendar} ${task.date}</span>
                            <span class="meta-item">${ICONS.clock} ${task.time}</span>
                            ${isLate ? '<span class="badge-late">Reschedule</span>' : ''}
                        </div>
                    </div>
                </div>
                <div class="task-actions">
                    <button class="action-btn" onclick="openModal('${task.id}')" title="Edit">${ICONS.edit}</button>
                    <button class="action-btn" onclick="deleteTask('${task.id}')" title="Delete">${ICONS.trash}</button>
                    <button class="action-btn" onclick="toggleExpand('${task.id}')" title="Expand/Collapse">
                        ${isExpanded ? ICONS.chevronUp : ICONS.chevronDown}
                    </button>
                </div>
            </div>
        `;

        // Subtasks Section (if expanded)
        if (isExpanded) {
            html += `
                <div class="subtasks-section">
                    <ul class="subtask-list">
            `;
            if (task.subtasks && task.subtasks.length > 0) {
                task.subtasks.forEach(sub => {
                    html += `
                        <li class="subtask-item">
                            <div class="subtask-left">
                                <input type="checkbox" class="task-checkbox" ${sub.completed ? 'checked' : ''} onclick="toggleSubtask('${task.id}', '${sub.id}')">
                                <span class="subtask-title ${sub.completed ? 'completed' : ''}">${escapeHTML(sub.text)}</span>
                            </div>
                            <button class="action-btn" onclick="deleteSubtask('${task.id}', '${sub.id}')" title="Delete subtask">${ICONS.close}</button>
                        </li>
                    `;
                });
            } else {
                html += `<li style="font-size:12px; color: var(--text-muted); font-style: italic;">No subtasks yet.</li>`;
            }

            html += `
                    </ul>
                    <div class="subtask-add-row">
                        <input type="text" id="subInput_${task.id}" placeholder="Add a subtask..." class="subtask-input" onkeydown="if(event.key==='Enter') addSubtask('${task.id}')">
                        <button class="btn-add-sub" onclick="addSubtask('${task.id}')">Add Subtask</button>
                    </div>
                </div>
            `;
        }

        li.innerHTML = html;
        taskList.appendChild(li);
    });
}

// Task Actions
function toggleTask(id) {
    const task = tasks.find(t => t.id === id);
    if (task) {
        task.completed = !task.completed;
        saveAndRefresh();
    }
}

function deleteTask(id) {
    tasks = tasks.filter(t => t.id !== id);
    expandedTaskIds.delete(id);
    saveAndRefresh();
}

// Expand / Collapse
function toggleExpand(id) {
    if (expandedTaskIds.has(id)) {
        expandedTaskIds.delete(id);
    } else {
        expandedTaskIds.add(id);
    }
    renderTasks();
}

// Subtask Handlers
function addSubtask(taskId) {
    const input = document.getElementById(`subInput_${taskId}`);
    const text = input.value.trim();
    if (!text) return;

    const task = tasks.find(t => t.id === taskId);
    if (task) {
        if (!task.subtasks) task.subtasks = [];
        task.subtasks.push({
            id: 'sub_' + Date.now(),
            text: text,
            completed: false
        });
        saveAndRefresh();
    }
}

function toggleSubtask(taskId, subId) {
    const task = tasks.find(t => t.id === taskId);
    if (task && task.subtasks) {
        const sub = task.subtasks.find(s => s.id === subId);
        if (sub) {
            sub.completed = !sub.completed;
            saveAndRefresh();
        }
    }
}

function deleteSubtask(taskId, subId) {
    const task = tasks.find(t => t.id === taskId);
    if (task && task.subtasks) {
        task.subtasks = task.subtasks.filter(s => s.id !== subId);
        saveAndRefresh();
    }
}

// Edit Modal Functions
function openModal(id) {
    const task = tasks.find(t => t.id === id);
    if (task) {
        document.getElementById('editId').value = task.id;
        document.getElementById('editText').value = task.text;
        document.getElementById('editDate').value = task.date;
        document.getElementById('editTime').value = task.time;
        document.getElementById('editModal').style.display = 'flex';
    }
}

function closeModal() {
    document.getElementById('editModal').style.display = 'none';
}

// Reschedule Modal Functions
function openRescheduleModal(id) {
    const task = tasks.find(t => t.id === id);
    if (task) {
        document.getElementById('rescheduleId').value = task.id;
        document.getElementById('rescheduleTaskTitle').innerText = `Task: "${task.text}"`;
        document.getElementById('rescheduleDate').value = task.date;
        document.getElementById('rescheduleTime').value = task.time;
        document.getElementById('rescheduleModal').style.display = 'flex';
    }
}

function closeRescheduleModal() {
    document.getElementById('rescheduleModal').style.display = 'none';
}

// Utility escape helper
function escapeHTML(str) {
    return str.replace(/[&<>'"]/g,
        tag => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[tag] || tag)
    );
}