package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// Structs representing data models
type Todo struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Date   string `json:"date"`
	Time   string `json:"time"`
	Done   bool   `json:"done"`
	IsLate bool   `json:"is_late"`
}

type Subtask struct {
	ID           string `json:"id"`
	ParentTaskID string `json:"parentTaskId"`
	Text         string `json:"text"`
	Done         bool   `json:"done"`
	CreatedAt    string `json:"createdAt"`
}

var db *firestore.Client
var ctx context.Context

func main() {
	ctx = context.Background()

	// Initialize Firebase App with Service Account Key
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalf("Error initializing app: %v", err)
	}

	// Get Firestore Client
	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalf("Error creating Firestore client: %v", err)
	}
	defer client.Close()

	db = client
	log.Println("Connected to Cloud Firestore successfully")

	mux := http.NewServeMux()

	// --- Existing Routes ---
	mux.HandleFunc("/api/todos", handleGetTodos)
	mux.HandleFunc("/api/todo/add", handleAddTodo)
	mux.HandleFunc("/api/todo/update", handleUpdateTodo)
	mux.HandleFunc("/api/todo/delete", handleDeleteTodo)
	mux.HandleFunc("/api/todo/toggle", handleToggleTodo)

	// --- New Routes for Subtasks, Rescheduling, and Progress ---
	mux.HandleFunc("/api/tasks/", handleTasksDynamicRoutes)

	handler := corsMiddleware(mux)

	port := ":8083"
	log.Printf("API Server running on http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, handler))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

const collectionName = "todos"

// ==========================================
// EXISTING HANDLERS (Unchanged)
// ==========================================

func handleGetTodos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	iter := db.Collection(collectionName).OrderBy("date", firestore.Asc).Documents(ctx)
	defer iter.Stop()

	var todos []Todo
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
		err = doc.DataTo(&t)
		if err != nil {
			continue
		}
		t.ID = doc.Ref.ID

		isPastDate := t.Date < todayStr
		isTodayButLate := (t.Date == todayStr && t.Time < currentTimeStr)
		t.IsLate = !t.Done && (isPastDate || isTodayButLate)

		todos = append(todos, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todos)
}

func handleAddTodo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type RequestBody struct {
		Text string `json:"text"`
		Date string `json:"date"`
		Time string `json:"time"`
	}

	var req RequestBody
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	_, _, err = db.Collection(collectionName).Add(ctx, map[string]interface{}{
		"text": req.Text,
		"date": req.Date,
		"time": req.Time,
		"done": false,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Task added"})
}

func handleUpdateTodo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type RequestBody struct {
		ID   string `json:"id"`
		Text string `json:"text"`
		Date string `json:"date"`
		Time string `json:"time"`
	}

	var req RequestBody
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	_, err = db.Collection(collectionName).Doc(req.ID).Update(ctx, []firestore.Update{
		{Path: "text", Value: req.Text},
		{Path: "date", Value: req.Date},
		{Path: "time", Value: req.Time},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Task updated"})
}

func handleDeleteTodo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type RequestBody struct {
		ID string `json:"id"`
	}

	var req RequestBody
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	_, err = db.Collection(collectionName).Doc(req.ID).Delete(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Task deleted"})
}

func handleToggleTodo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type RequestBody struct {
		ID string `json:"id"`
	}

	var req RequestBody
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Attempting to toggle task ID: %s", req.ID)
	docRef := db.Collection(collectionName).Doc(req.ID)
	snap, err := docRef.Get(ctx)
	if err != nil {
		log.Printf("Error fetching doc %s: %v", req.ID, err)
		http.Error(w, "Document not found or error fetching", http.StatusNotFound)
		return
	}

	if !snap.Exists() {
		log.Printf("Document %s does not exist in Firestore", req.ID)
		http.Error(w, "Document does not exist", http.StatusNotFound)
		return
	}

	data := snap.Data()
	currentDone, ok := data["done"].(bool)
	if !ok {
		currentDone = false
	}

	newStatus := !currentDone
	log.Printf("Toggling ID %s from %v to %v", req.ID, currentDone, newStatus)

	_, err = docRef.Update(ctx, []firestore.Update{
		{Path: "done", Value: newStatus},
	})

	if err != nil {
		log.Printf("ERROR UPDATING DOC %s: %v", req.ID, err)
		http.Error(w, "Failed to update status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Status toggled"})
}

// ==========================================
// NEW HANDLERS (Subtasks, Reschedule, Progress)
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
			handleRescheduleTodo(w, r, parts[0])
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

// handleGetProgress calculates progress for ALL tasks in the planner
func handleGetProgress(w http.ResponseWriter, r *http.Request) {
	// Query ALL tasks in the collection, not just today's tasks
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
			// Check if this active task is overdue
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

func handleRescheduleTodo(w http.ResponseWriter, r *http.Request, taskId string) {
	type RequestBody struct {
		Date string `json:"date"`
		Time string `json:"time"`
	}
	var req RequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if _, err := time.Parse("2006-01-02", req.Date); err != nil {
		http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	if _, err := time.Parse("15:04", req.Time); err != nil {
		http.Error(w, "Invalid time format. Use HH:MM", http.StatusBadRequest)
		return
	}

	docRef := db.Collection(collectionName).Doc(taskId)
	snap, err := docRef.Get(ctx)
	if err != nil || !snap.Exists() {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	_, err = docRef.Update(ctx, []firestore.Update{
		{Path: "date", Value: req.Date},
		{Path: "time", Value: req.Time},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updatedSnap, err := docRef.Get(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var t Todo
	if err := updatedSnap.DataTo(&t); err != nil {
		http.Error(w, "Failed to parse updated task", http.StatusInternalServerError)
		return
	}
	t.ID = updatedSnap.Ref.ID

	now := time.Now()
	todayStr := now.Format("2006-01-02")
	currentTimeStr := now.Format("15:04")
	isPastDate := t.Date < todayStr
	isTodayButLate := (t.Date == todayStr && t.Time < currentTimeStr)
	t.IsLate = !t.Done && (isPastDate || isTodayButLate)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(t)
}

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
