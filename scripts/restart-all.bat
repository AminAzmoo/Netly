@echo off
echo Stopping...
taskkill /IM netly-server.exe /F >nul 2>&1
taskkill /IM node.exe /F >nul 2>&1
timeout /t 2 /nobreak >nul

echo Starting...
call start-all.bat
