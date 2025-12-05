@echo off
echo Building Netly...
cd /d "%~dp0..\backend\bin"
call build.bat
if %errorlevel% neq 0 exit /b %errorlevel%

cd /d "%~dp0..\frontend"
call npm run build
if %errorlevel% neq 0 exit /b %errorlevel%

echo.
echo ✓ Done!
pause
