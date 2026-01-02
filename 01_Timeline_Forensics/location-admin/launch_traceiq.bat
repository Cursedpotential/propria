@echo off
title TraceIQ Location Manager
echo Starting TraceIQ Agent Gateway + UI...
cd /d "C:\Users\matts\location-admin"

:: Start the Combined Server (API + Vite)
start /b npm run start:all

:: Wait for server to spin up
timeout /t 5 /nobreak >nul

:: Open the browser
start http://localhost:5173

:: Keep window open so server runs
echo TraceIQ is running. Close this window to stop.
pause