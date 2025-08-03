@echo off

echo 🚀 Starting development server with hot reload...

:: Check if Air is installed
where air >nul 2>nul
if %errorlevel% neq 0 (
    echo ❌ Air is not installed. Installing...
    go install github.com/air-verse/air@latest
)

:: Load environment variables if .env exists
if exist .env (
    echo 📄 Loading environment variables from .env...
    for /f "tokens=*" %%i in (.env) do set %%i
) else (
    echo ⚠️  No .env file found. Using default development settings...
    set ENABLE_TELEMETRY=false
    set ENABLE_PRETTY_LOGS=true
    set LOG_LEVEL=debug
    set SERVICE_NAME=gateway-dev
)

:: Create tmp directory if it doesn't exist
if not exist tmp mkdir tmp

echo 🔥 Starting Air with hot reload...
echo 📝 Watching for changes in:
echo    - *.go files
echo    - internal/ directory
echo    - pkg/ directory
echo    - cmd/ directory
echo.
echo 🌐 Application will be available at: http://localhost:8080
echo 🔄 Press Ctrl+C to stop
echo.

:: Start Air
air