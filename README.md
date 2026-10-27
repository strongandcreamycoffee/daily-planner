# Daily Planner Application

A full-stack web application built to practice Go and API development. It helps users schedule, track, and manage their daily tasks.

The project uses a decoupled monorepo structure. The backend is a REST API written in Go, and the frontend is built with HTML, CSS, and vanilla JavaScript. They communicate using JSON.

## Key Features

- **RESTful API:** The Go backend handles data using GET, POST, PATCH, and DELETE requests.
- **Smart late detection:** The server checks the current time against task deadlines and flags overdue tasks as "Late."
- **Secure database connection:** Firebase Cloud Firestore credentials stay on the backend and are not exposed to the frontend.
- **Lightweight frontend:** The frontend uses vanilla JavaScript without a framework.
- **CORS handling:** Custom Go middleware allows the frontend and backend to communicate during local development.

## Technology Stack

- **Backend:** Go
- **Database:** Firebase Cloud Firestore
- **Frontend:** HTML5, CSS3, vanilla JavaScript
- **Version control:** Git and GitHub

## Run Locally

### Prerequisites

- [Go](https://go.dev/dl/)
- [Git](https://git-scm.com/)
- A Firebase project with Cloud Firestore enabled

### 1. Clone the repository

```bash
git clone https://github.com/strongandcreamycoffee/daily-planner.git
cd daily-planner
```

### 2. Configure and run the backend

The backend needs a Firebase service account key to connect to Firestore.

1. In the Firebase Console, open **Project settings > Service accounts** and generate a private key.
2. Save the downloaded file as `serviceAccountKey.json` in the `backend-monorepo` directory. This file is ignored by Git; do not commit or share it.
3. Install dependencies and start the API:

```bash
cd backend-monorepo
go mod tidy
go run main.go
```

The API listens at `http://localhost:8083`.

### 3. Run the frontend

In a separate terminal, open `frontend-monorepo/index.html` in a browser, or serve the folder with a local server such as the VS Code Live Server extension. Keep the Go backend running so the frontend can fetch tasks.

## Project Structure

```text
daily-planner/
├── backend-monorepo/           # Go API server
│   ├── main.go                 # Server routes, handlers, and CORS middleware
│   ├── go.mod                  # Go module dependencies
│   └── serviceAccountKey.json  # Local Firebase credentials (not committed)
├── frontend-monorepo/          # Client-side interface
│   ├── index.html              # Main entry point
│   ├── style.css               # UI styling
│   └── script.js               # DOM manipulation and API requests
└── README.md                   # Project documentation
```
