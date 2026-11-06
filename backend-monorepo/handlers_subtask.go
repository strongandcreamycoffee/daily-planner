package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// Struct for Subtasks
type Subtask struct {
	ID           string `json:"id"`
	ParentTaskID string `json:"parentTaskId"`
	Text         string `json:"text"`
	Done         bool   `json:"done"`
	CreatedAt    string `json:"createdAt"`
}

// ==========================================
// DYNAMIC ROUTER
// ==========================================

// handleTasksDynamicRoutes routes requests to specific subtask, reschedule, or progress handlers
func handleTasksDynamicRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	parts := strings.Split(path, "/")

	// GET /api/tasks/progress
	if len(parts) == 1 && parts[0] == "progress" {
		if r.Method == http.MethodGet {
			handleGetProgress(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// PATCH /api/tasks/{taskId}/reschedule
	if len(parts) == 2 && parts[1] == "reschedule" {
		if r.Method == http.MethodPatch {
			handleRescheduleTodo(w, r, parts[0]) // Calls function from handlers_todo.go
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Subtask routes: /api/tasks/{taskId}/subtasks or /api/tasks/{taskId}/subtasks/{subtaskId}
	if len(parts) >= 2 && parts[1] == "subtasks" {
		taskId := parts[0]
		if taskId == "" {
			http.Error(w, "Task ID is required", http.StatusBadRequest)
			return
		}

		if len(parts) == 2 {
			if r.Method == http.MethodPost {
				handleAddSubtask(w, r, taskId)
			} else if r.Method == http.MethodGet {
				handleGetSubtasks(w, r, taskId)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		} else if len(parts) == 3 {
			subtaskId := parts[2]
			if subtaskId == "" {
				http.Error(w, "Subtask ID is required", http.StatusBadRequest)
				return
			}

			if r.Method == http.MethodPut {
				handleUpdateSubtask(w, r, taskId, subtaskId)
			} else if r.Method == http.MethodPatch {
				handleToggleSubtask(w, r, taskId, subtaskId)
			} else if r.Method == http.MethodDelete {
				handleDeleteSubtask(w, r, taskId, subtaskId)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

// ==========================================
// PROGRESS STATS
// ==========================================

func handleGetProgress(w http.ResponseWriter, r *http.Request) {
	iter := db.Collection(collectionName).Documents(ctx)
	defer iter.Stop()

	totalTasks := 0
	completedTasks := 0
	activeTasks := 0
	overdueTasks := 0

	now := time.Now()
	todayStr := now.Format("2006-01-02")
	currentTimeStr := now.Format("15:04")

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var t Todo
		if err := doc.DataTo(&t); err != nil {
			continue
		}

		totalTasks++
		if t.Done {
			completedTasks++
		} else {
			activeTasks++
			isPastDate := t.Date < todayStr
			isTodayButLate := (t.Date == todayStr && t.Time < currentTimeStr)
			if isPastDate || isTodayButLate {
				overdueTasks++
			}
		}
	}

	progressPercentage := 0
	if totalTasks > 0 {
		progressPercentage = int((float64(completedTasks) / float64(totalTasks)) * 100)
	}

	response := map[string]interface{}{
		"date":               todayStr,
		"totalTasks":         totalTasks,
		"completedTasks":     completedTasks,
		"activeTasks":        activeTasks,
		"overdueTasks":       overdueTasks,
		"progressPercentage": progressPercentage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ==========================================
// SUBTASK HANDLERS
// ==========================================

func handleAddSubtask(w http.ResponseWriter, r *http.Request, taskId string) {
	docRef := db.Collection(collectionName).Doc(taskId)
	snap, err := docRef.Get(ctx)
	if err != nil || !snap.Exists() {
		http.Error(w, "Parent task not found", http.StatusNotFound)
		return
	}

	type RequestBody struct {
		Text string `json:"text"`
	}
	var req RequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}

	now := time.Now().Format(time.RFC3339)
	subtaskRef := docRef.Collection("subtasks").NewDoc()
	_, err = subtaskRef.Set(ctx, map[string]interface{}{
		"parentTaskId": taskId,
		"text":         req.Text,
		"done":         false,
		"createdAt":    now,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Subtask{
		ID:           subtaskRef.ID,
		ParentTaskID: taskId,
		Text:         req.Text,
		Done:         false,
		CreatedAt:    now,
	})
}

func handleGetSubtasks(w http.ResponseWriter, r *http.Request, taskId string) {
	docRef := db.Collection(collectionName).Doc(taskId)
	snap, err := docRef.Get(ctx)
	if err != nil || !snap.Exists() {
		http.Error(w, "Parent task not found", http.StatusNotFound)
		return
	}

	iter := docRef.Collection("subtasks").OrderBy("createdAt", firestore.Asc).Documents(ctx)
	defer iter.Stop()

	var subtasks []Subtask
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var st Subtask
		if err := doc.DataTo(&st); err != nil {
			continue
		}
		st.ID = doc.Ref.ID
		subtasks = append(subtasks, st)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subtasks)
}

func handleUpdateSubtask(w http.ResponseWriter, r *http.Request, taskId, subtaskId string) {
	type RequestBody struct {
		Text string `json:"text"`
	}
	var req RequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}

	subtaskRef := db.Collection(collectionName).Doc(taskId).Collection("subtasks").Doc(subtaskId)
	snap, err := subtaskRef.Get(ctx)
	if err != nil || !snap.Exists() {
		http.Error(w, "Subtask not found", http.StatusNotFound)
		return
	}

	_, err = subtaskRef.Update(ctx, []firestore.Update{
		{Path: "text", Value: req.Text},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Subtask updated"})
}

func handleToggleSubtask(w http.ResponseWriter, r *http.Request, taskId, subtaskId string) {
	subtaskRef := db.Collection(collectionName).Doc(taskId).Collection("subtasks").Doc(subtaskId)
	snap, err := subtaskRef.Get(ctx)
	if err != nil || !snap.Exists() {
		http.Error(w, "Subtask not found", http.StatusNotFound)
		return
	}

	data := snap.Data()
	currentDone, ok := data["done"].(bool)
	if !ok {
		currentDone = false
	}
	newStatus := !currentDone

	_, err = subtaskRef.Update(ctx, []firestore.Update{
		{Path: "done", Value: newStatus},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Subtask status toggled",
		"done":    newStatus,
	})
}

func handleDeleteSubtask(w http.ResponseWriter, r *http.Request, taskId, subtaskId string) {
	subtaskRef := db.Collection(collectionName).Doc(taskId).Collection("subtasks").Doc(subtaskId)
	snap, err := subtaskRef.Get(ctx)
	if err != nil || !snap.Exists() {
		http.Error(w, "Subtask not found", http.StatusNotFound)
		return
	}

	_, err = subtaskRef.Delete(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Subtask deleted"})
}
