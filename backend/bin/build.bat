@echo off
echo Building Netly Project...
echo.
cd /d "%~dp0.."

echo [1/4] Building Server...
go build -o bin\netly-server.exe .\cmd\server
if %errorlevel% neq 0 exit /b %errorlevel%

echo [2/4] Building Keygen...
go build -o bin\netly-keygen.exe .\cmd\keygen
if %errorlevel% neq 0 exit /b %errorlevel%

echo [3/4] Building Linux Agent (amd64)...
cd agent
set GOOS=linux
set GOARCH=amd64
go build -o ..\bin\uploads\netly-agent-amd64 .\cmd\agent
if %errorlevel% neq 0 exit /b %errorlevel%

echo [4/4] Building Linux Agent (arm64)...
set GOARCH=arm64
go build -o ..\bin\uploads\netly-agent-arm64 .\cmd\agent
if %errorlevel% neq 0 exit /b %errorlevel%

cd ..
echo.
echo ✓ Build Complete!
echo   - Server: bin\netly-server.exe
echo   - Keygen: bin\netly-keygen.exe
echo   - Agent (amd64): bin\uploads\netly-agent-amd64
echo   - Agent (arm64): bin\uploads\netly-agent-arm64
