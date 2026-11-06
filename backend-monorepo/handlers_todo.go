package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
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

const collectionName = "todos"

// ==========================================
// TODO HANDLERS
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
