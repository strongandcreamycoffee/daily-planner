@echo off
title Daily Planner Launcher
start "BACKEND :8083" cmd /k "cd /d C:\Users\jguillamar\Documents\daily-planner\backend-monorepo && go run ."
timeout /t 2 >nul
start "FRONTEND :5173" cmd /k "cd /d C:\Users\jguillamar\Documents\daily-planner\frontend-monorepo && npx vite --host 0.0.0.0 --port 5173"