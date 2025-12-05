@echo off
title Netly
cd /d "%~dp0..\backend"

start /b "" bin\netly-server.exe
timeout /t 2 /nobreak >nul

cd ..\frontend
start /b "" cmd /c "npm run dev"

echo Backend: http://localhost:8081
echo Frontend: http://localhost:3000
echo.
echo Press Ctrl+C to stop
pause >nul
