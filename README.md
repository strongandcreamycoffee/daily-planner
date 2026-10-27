# Daily Planner Application

A full-stack web application I built to practice GoLang and API development. It helps users schedule, track, and manage their daily tasks. 

I built this using a decoupled monorepo structure. The backend is a REST API written in Go, and the frontend is built with plain HTML, CSS, and JavaScript. They communicate with each other using JSON. I chose this setup to keep the business logic secure on the server while keeping the frontend lightweight and fast.

## Key Features

- **RESTful API:** The Go backend handles all the data using standard GET, POST, PATCH, and DELETE requests.
- **Smart Late Detection:** The server checks the current time against the task deadline. If a task is overdue, it automatically flags it as "Late" before sending the data to the browser, ensuring the time is always accurate.
- **Secure Database Connection:** I used Firebase Firestore for the database. The sensitive service account keys stay safely on the backend, so they are never exposed to the frontend.
- **Lightweight Frontend:** No heavy frameworks here. I used vanilla JavaScript to keep the project simple and easy to understand.
- **CORS Handling:** I implemented custom middleware in Go to allow the frontend and backend to communicate smoothly during local development.

## Technology Stack

- **Backend:** Go (Golang)
- **Database:** Firebase Cloud Firestore
- **Frontend:** HTML5, CSS3, Vanilla JavaScript
- **Version Control:** Git and GitHub

---

## How to Run Locally

If you want to run this project on your own machine, follow these steps.

### Prerequisites

Make sure you have the following installed on your computer:

- [Go (Golang)](https://go.dev/dl/)
- [Git](https://git-scm.com/)
- A Firebase Project with Cloud Firestore enabled.

### Step 1: Clone the Repository

Open your terminal and clone the project to your computer:

```bash
git clone https://github.com/strongandcreamycoffee/daily-planner.git
cd daily-planner

### Step 2: Set Up the Backend

The backend needs a Firebase Service Account key to connect to the database.

**1. Go to the backend folder:**

```bash
cd backend-monorepo

**2. Get your `serviceAccountKey.json` file** from your Firebase Console (Project Settings > Service Accounts > Generate new private key).

**3. Put the `serviceAccountKey.json` file** directly inside the `backend-monorepo` folder. 
*(Note: This file is ignored by Git so it doesn't get uploaded publicly).*

**4. Download the necessary Go libraries:**

```bash
go mod tidy

### Step 3: Run the Frontend

**1. Open a new terminal window** and go to the frontend folder:

```bash
cd ../frontend-monorepo

**2. Open the `index.html` file** directly in your browser, or use a local server like the VS Code "Live Server" extension. 

**3. Make sure your Go backend is still running** in the other terminal window so the frontend can fetch the tasks.

## Project Structure

Here is how the files are organized in this repository:

```text
daily-planner/
├── backend-monorepo/       # Go API Server
│   ├── main.go             # Server routes, handlers, and CORS middleware
│   ├── go.mod              # Go module dependencies
│   └── serviceAccountKey.json # (Local only) Firebase credentials
│
├── frontend-monorepo/      # Client-side Interface
│   ├── index.html          # Main entry point
│   ├── style.css           # UI styling
│   └── script.js           # DOM manipulation and API fetch logic
│
└── README.md               # Project documentation