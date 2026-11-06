package main

import (
	"context"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

// Global variables shared across handlers
var db *firestore.Client
var ctx context.Context

func main() {
	ctx = context.Background()

	// Initialize Firebase App with Service Account Key
	sa := option.WithAuthCredentialsFile(option.ServiceAccount, "serviceAccountKey.json")
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

	// --- Register Todo Routes ---
	mux.HandleFunc("/api/todos", handleGetTodos)
	mux.HandleFunc("/api/todo/add", handleAddTodo)
	mux.HandleFunc("/api/todo/update", handleUpdateTodo)
	mux.HandleFunc("/api/todo/delete", handleDeleteTodo)
	mux.HandleFunc("/api/todo/toggle", handleToggleTodo)

	// --- Register Dynamic Task/Subtask/Progress Routes ---
	mux.HandleFunc("/api/tasks/", handleTasksDynamicRoutes)

	handler := corsMiddleware(mux)

	port := ":8083"
	log.Printf("API Server running on http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, handler))
}

// Middleware for CORS
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
