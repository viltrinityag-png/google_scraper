@echo off
setlocal
cd /d "%~dp0"

if not exist ".env" (
  copy ".env.example" ".env" >nul
  echo Created .env from .env.example.
  echo Edit .env and set LEAD_API_KEY before exposing this API publicly.
)

where docker >nul 2>&1
if errorlevel 1 (
  echo Docker is not installed or not on PATH.
  echo Install Docker Desktop, then run this file again.
  pause
  exit /b 1
)

docker compose up --build
