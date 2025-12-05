@echo off
title Netly - Restart
echo Restarting Netly Full Stack...
echo.

echo Stopping processes...
taskkill /FI "WINDOWTITLE eq Netly Backend*" /F >nul 2>&1
taskkill /FI "WINDOWTITLE eq Netly Frontend*" /F >nul 2>&1
taskkill /IM netly-server.exe /F >nul 2>&1
taskkill /IM node.exe /F >nul 2>&1
timeout /t 2 /nobreak >nul

echo Starting services...
start "Netly Backend" cmd /k "cd /d %~dp0..\backend\bin && start.bat"
timeout /t 2 /nobreak >nul

start "Netly Frontend" cmd /k "cd /d %~dp0..\frontend && npm run dev"

echo.
echo ✓ Restarted successfully
pause
