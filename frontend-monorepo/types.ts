// types.ts

export interface Subtask {
    id: string;
    text: string;
    completed: boolean;
}

export interface Task {
    id: string;
    text: string;
    date: string; // YYYY-MM-DD format
    time: string; // HH:MM format
    completed: boolean;
    subtasks: Subtask[];
    
    // Optional computed field added dynamically by UI logic
    isLate?: boolean; 
}

export type FilterStatus = 'all' | 'active' | 'completed';

// FullCalendar event shape (if needed later)
export interface CalendarEvent {
    id: string;
    title: string;
    start: string; // ISO datetime string
    backgroundColor?: string;
    borderColor?: string;
    textColor?: string;
}