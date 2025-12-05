@echo off
title Netly - Full Stack
echo Starting Netly Full Stack...
echo.

start "Netly Backend" cmd /k "cd /d %~dp0..\backend\bin && start.bat"
timeout /t 2 /nobreak >nul

start "Netly Frontend" cmd /k "cd /d %~dp0..\frontend && npm run dev"

echo.
echo ✓ Backend: http://localhost:8081
echo ✓ Frontend: http://localhost:3000
echo.
pause
