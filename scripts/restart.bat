@echo off
echo Stopping server...
taskkill /F /IM netly.exe 2>nul

echo Building backend...
cd backend
go build -o netly.exe ./cmd/server
if %errorlevel% neq 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo Starting server...
start "" netly.exe

echo Done!
timeout /t 2 >nul
