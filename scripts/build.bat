@echo off
REM MySQL Schema Sync Build Script for Windows
REM This script provides convenient build commands for development and release

setlocal enabledelayedexpansion

REM Application information
set APP_NAME=mysql-schema-sync
set BUILD_DIR=build

REM Get version information
for /f "tokens=*" %%i in ('git describe --tags --always --dirty 2^>nul') do set VERSION=%%i
if "%VERSION%"=="" set VERSION=dev

for /f "tokens=*" %%i in ('powershell -command "Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ'"') do set BUILD_TIME=%%i

for /f "tokens=*" %%i in ('git rev-parse --short HEAD 2^>nul') do set GIT_COMMIT=%%i
if "%GIT_COMMIT%"=="" set GIT_COMMIT=unknown

for /f "tokens=3" %%i in ('go version') do set GO_VERSION=%%i

REM Build flags
set LDFLAGS=-ldflags "-X main.Version=%VERSION% -X main.BuildTime=%BUILD_TIME% -X main.GitCommit=%GIT_COMMIT% -X main.GoVersion=%GO_VERSION%"

REM Function to show help
if "%1"=="help" goto :show_help
if "%1"=="" goto :show_help

REM Function to show version
if "%1"=="version" goto :show_version

REM Function to build for current platform
if "%1"=="build" goto :build_current

REM Function to build for all platforms
if "%1"=="build-all" goto :build_all

REM Function to clean
if "%1"=="clean" goto :clean_build

REM Function to run tests
if "%1"=="test" goto :run_tests

REM Function to create release
if "%1"=="release" goto :create_release

REM Function to build Docker
if "%1"=="docker" goto :build_docker

goto :show_help

:show_help
echo MySQL Schema Sync Build Script for Windows
echo.
echo Usage: %0 [COMMAND]
echo.
echo Commands:
echo   build           Build for current platform
echo   build-all       Build for all supported platforms
echo   clean           Clean build artifacts
echo   test            Run tests
echo   release         Create release packages
echo   docker          Build Docker image
echo   version         Show version information
echo   help            Show this help message
echo.
goto :end

:show_version
echo Application: %APP_NAME%
echo Version: %VERSION%
echo Build Time: %BUILD_TIME%
echo Git Commit: %GIT_COMMIT%
echo Go Version: %GO_VERSION%
goto :end

:build_current
echo [INFO] Building %APP_NAME% v%VERSION% for current platform...
go build %LDFLAGS% -o %APP_NAME%.exe .
if %errorlevel% equ 0 (
    echo [SUCCESS] Build complete: %APP_NAME%.exe
) else (
    echo [ERROR] Build failed
    exit /b 1
)
goto :end

:build_all
echo [INFO] Building %APP_NAME% v%VERSION% for all platforms...

REM Clean and create build directory
if exist %BUILD_DIR% rmdir /s /q %BUILD_DIR%
mkdir %BUILD_DIR%

REM Build for Linux amd64
echo [INFO] Building for linux/amd64...
set GOOS=linux
set GOARCH=amd64
go build %LDFLAGS% -o %BUILD_DIR%\%APP_NAME%-linux-amd64 .
if %errorlevel% neq 0 goto :build_error

REM Build for Linux arm64
echo [INFO] Building for linux/arm64...
set GOOS=linux
set GOARCH=arm64
go build %LDFLAGS% -o %BUILD_DIR%\%APP_NAME%-linux-arm64 .
if %errorlevel% neq 0 goto :build_error

REM Build for macOS amd64
echo [INFO] Building for darwin/amd64...
set GOOS=darwin
set GOARCH=amd64
go build %LDFLAGS% -o %BUILD_DIR%\%APP_NAME%-darwin-amd64 .
if %errorlevel% neq 0 goto :build_error

REM Build for macOS arm64
echo [INFO] Building for darwin/arm64...
set GOOS=darwin
set GOARCH=arm64
go build %LDFLAGS% -o %BUILD_DIR%\%APP_NAME%-darwin-arm64 .
if %errorlevel% neq 0 goto :build_error

REM Build for Windows amd64
echo [INFO] Building for windows/amd64...
set GOOS=windows
set GOARCH=amd64
go build %LDFLAGS% -o %BUILD_DIR%\%APP_NAME%-windows-amd64.exe .
if %errorlevel% neq 0 goto :build_error

REM Build for Windows arm64
echo [INFO] Building for windows/arm64...
set GOOS=windows
set GOARCH=arm64
go build %LDFLAGS% -o %BUILD_DIR%\%APP_NAME%-windows-arm64.exe .
if %errorlevel% neq 0 goto :build_error

echo [SUCCESS] All builds complete. Binaries available in %BUILD_DIR%\
goto :end

:build_error
echo [ERROR] Build failed for %GOOS%/%GOARCH%
exit /b 1

:clean_build
echo [INFO] Cleaning build artifacts...
if exist %BUILD_DIR% rmdir /s /q %BUILD_DIR%
if exist %APP_NAME%.exe del %APP_NAME%.exe
echo [SUCCESS] Clean complete
goto :end

:run_tests
echo [INFO] Running tests...
echo [INFO] Running unit tests...
go test -v -short ./...
if %errorlevel% neq 0 (
    echo [ERROR] Unit tests failed
    exit /b 1
)

echo [INFO] Running integration tests...
go test -v -tags=integration ./internal
if %errorlevel% neq 0 (
    echo [ERROR] Integration tests failed
    exit /b 1
)

echo [SUCCESS] All tests passed
goto :end

:create_release
echo [INFO] Creating release packages...
call :build_all
if %errorlevel% neq 0 exit /b 1

mkdir %BUILD_DIR%\packages

REM Create packages (simplified for Windows batch)
echo [INFO] Creating Windows package...
mkdir %BUILD_DIR%\packages\%APP_NAME%-%VERSION%-windows-amd64
copy %BUILD_DIR%\%APP_NAME%-windows-amd64.exe %BUILD_DIR%\packages\%APP_NAME%-%VERSION%-windows-amd64\
copy README.md %BUILD_DIR%\packages\%APP_NAME%-%VERSION%-windows-amd64\
copy LICENSE %BUILD_DIR%\packages\%APP_NAME%-%VERSION%-windows-amd64\
copy CHANGELOG.md %BUILD_DIR%\packages\%APP_NAME%-%VERSION%-windows-amd64\
xcopy examples %BUILD_DIR%\packages\%APP_NAME%-%VERSION%-windows-amd64\examples\ /e /i

powershell -command "Compress-Archive -Path '%BUILD_DIR%\packages\%APP_NAME%-%VERSION%-windows-amd64' -DestinationPath '%BUILD_DIR%\packages\%APP_NAME%-%VERSION%-windows-amd64.zip'"
rmdir /s /q %BUILD_DIR%\packages\%APP_NAME%-%VERSION%-windows-amd64

echo [SUCCESS] Release packages created in %BUILD_DIR%\packages\
goto :end

:build_docker
echo [INFO] Building Docker image: mysql-schema-sync:latest
docker build -t mysql-schema-sync:latest .
if %errorlevel% equ 0 (
    echo [SUCCESS] Docker image built: mysql-schema-sync:latest
) else (
    echo [ERROR] Docker build failed
    exit /b 1
)
goto :end

:end
endlocal