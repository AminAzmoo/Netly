@echo off
title Netly - Build All
echo Building Netly Full Stack...
echo.

echo [BACKEND] Building...
cd /d "%~dp0..\backend\bin"
call build.bat
if %errorlevel% neq 0 exit /b %errorlevel%

echo.
echo [FRONTEND] Building...
cd /d "%~dp0..\frontend"
call npm run build
if %errorlevel% neq 0 exit /b %errorlevel%

echo.
echo ✓ Build Complete!
pause
