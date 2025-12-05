@echo off
echo Building Netly...
cd /d "%~dp0.."

echo [1/4] Server...
go build -o bin\netly-server.exe .\cmd\server || exit /b 1

echo [2/4] Keygen...
go build -o bin\netly-keygen.exe .\cmd\keygen || exit /b 1

echo [3/4] Agent (amd64)...
cd agent && set GOOS=linux&& set GOARCH=amd64&& go build -o ..\bin\uploads\netly-agent-amd64 .\cmd\agent || exit /b 1

echo [4/4] Agent (arm64)...
set GOARCH=arm64&& go build -o ..\bin\uploads\netly-agent-arm64 .\cmd\agent || exit /b 1

cd ..
echo.
echo ✓ Done!
