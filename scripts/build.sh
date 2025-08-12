#!/bin/bash

# MySQL Schema Sync Build Script
# This script provides convenient build commands for development and release

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Application information
APP_NAME="mysql-schema-sync"
BUILD_DIR="build"

# Get version information
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION=$(go version | cut -d' ' -f3)

# Build flags
LDFLAGS="-ldflags \"-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT} -X main.GoVersion=${GO_VERSION}\""

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show help
show_help() {
    echo "MySQL Schema Sync Build Script"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  build           Build for current platform"
    echo "  build-all       Build for all supported platforms"
    echo "  build-linux     Build for Linux platforms"
    echo "  build-darwin    Build for macOS platforms"
    echo "  build-windows   Build for Windows platforms"
    echo "  clean           Clean build artifacts"
    echo "  test            Run tests"
    echo "  release         Create release packages"
    echo "  docker          Build Docker image"
    echo "  version         Show version information"
    echo "  help            Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  BUILD_DIR       Build output directory (default: build)"
    echo "  DOCKER_TAG      Docker image tag (default: mysql-schema-sync:latest)"
    echo ""
}

# Function to show version information
show_version() {
    echo "Application: ${APP_NAME}"
    echo "Version: ${VERSION}"
    echo "Build Time: ${BUILD_TIME}"
    echo "Git Commit: ${GIT_COMMIT}"
    echo "Go Version: ${GO_VERSION}"
}

# Function to build for current platform
build_current() {
    print_status "Building ${APP_NAME} v${VERSION} for current platform..."
    eval "go build ${LDFLAGS} -o ${APP_NAME} ."
    print_success "Build complete: ${APP_NAME}"
}

# Function to build for all platforms
build_all() {
    print_status "Building ${APP_NAME} v${VERSION} for all platforms..."
    
    # Clean and create build directory
    rm -rf ${BUILD_DIR}
    mkdir -p ${BUILD_DIR}
    
    # Define platforms
    platforms=(
        "linux/amd64"
        "linux/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "windows/amd64"
        "windows/arm64"
    )
    
    # Build for each platform
    for platform in "${platforms[@]}"; do
        IFS='/' read -r os arch <<< "$platform"
        output_name="${APP_NAME}-${os}-${arch}"
        
        if [ "$os" = "windows" ]; then
            output_name="${output_name}.exe"
        fi
        
        print_status "Building for ${os}/${arch}..."
        
        if GOOS=$os GOARCH=$arch eval "go build ${LDFLAGS} -o ${BUILD_DIR}/${output_name} ."; then
            print_success "Built ${output_name}"
        else
            print_error "Failed to build for ${os}/${arch}"
            exit 1
        fi
    done
    
    print_success "All builds complete. Binaries available in ${BUILD_DIR}/"
}

# Function to build for Linux
build_linux() {
    print_status "Building ${APP_NAME} v${VERSION} for Linux platforms..."
    mkdir -p ${BUILD_DIR}
    
    GOOS=linux GOARCH=amd64 eval "go build ${LDFLAGS} -o ${BUILD_DIR}/${APP_NAME}-linux-amd64 ."
    GOOS=linux GOARCH=arm64 eval "go build ${LDFLAGS} -o ${BUILD_DIR}/${APP_NAME}-linux-arm64 ."
    
    print_success "Linux builds complete"
}

# Function to build for macOS
build_darwin() {
    print_status "Building ${APP_NAME} v${VERSION} for macOS platforms..."
    mkdir -p ${BUILD_DIR}
    
    GOOS=darwin GOARCH=amd64 eval "go build ${LDFLAGS} -o ${BUILD_DIR}/${APP_NAME}-darwin-amd64 ."
    GOOS=darwin GOARCH=arm64 eval "go build ${LDFLAGS} -o ${BUILD_DIR}/${APP_NAME}-darwin-arm64 ."
    
    print_success "macOS builds complete"
}

# Function to build for Windows
build_windows() {
    print_status "Building ${APP_NAME} v${VERSION} for Windows platforms..."
    mkdir -p ${BUILD_DIR}
    
    GOOS=windows GOARCH=amd64 eval "go build ${LDFLAGS} -o ${BUILD_DIR}/${APP_NAME}-windows-amd64.exe ."
    GOOS=windows GOARCH=arm64 eval "go build ${LDFLAGS} -o ${BUILD_DIR}/${APP_NAME}-windows-arm64.exe ."
    
    print_success "Windows builds complete"
}

# Function to clean build artifacts
clean_build() {
    print_status "Cleaning build artifacts..."
    rm -rf ${BUILD_DIR}
    rm -f ${APP_NAME}
    rm -f ${APP_NAME}.exe
    print_success "Clean complete"
}

# Function to run tests
run_tests() {
    print_status "Running tests..."
    
    print_status "Running unit tests..."
    go test -v -short ./...
    
    print_status "Running integration tests..."
    go test -v -tags=integration ./internal
    
    print_success "All tests passed"
}

# Function to create release packages
create_release() {
    print_status "Creating release packages..."
    
    # Build all platforms first
    build_all
    
    # Create packages directory
    mkdir -p ${BUILD_DIR}/packages
    
    # Define platforms for packaging
    platforms=(
        "linux/amd64"
        "linux/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "windows/amd64"
        "windows/arm64"
    )
    
    # Create packages for each platform
    for platform in "${platforms[@]}"; do
        IFS='/' read -r os arch <<< "$platform"
        binary_name="${APP_NAME}-${os}-${arch}"
        package_name="${APP_NAME}-${VERSION}-${os}-${arch}"
        
        if [ "$os" = "windows" ]; then
            binary_name="${binary_name}.exe"
        fi
        
        print_status "Creating package for ${os}/${arch}..."
        
        # Create package directory
        mkdir -p ${BUILD_DIR}/packages/${package_name}
        
        # Copy files to package
        cp ${BUILD_DIR}/${binary_name} ${BUILD_DIR}/packages/${package_name}/
        cp README.md ${BUILD_DIR}/packages/${package_name}/
        cp LICENSE ${BUILD_DIR}/packages/${package_name}/
        cp CHANGELOG.md ${BUILD_DIR}/packages/${package_name}/
        cp -r examples ${BUILD_DIR}/packages/${package_name}/
        
        # Create archive
        cd ${BUILD_DIR}/packages
        if [ "$os" = "windows" ]; then
            zip -r ${package_name}.zip ${package_name}/
        else
            tar -czf ${package_name}.tar.gz ${package_name}/
        fi
        rm -rf ${package_name}
        cd - > /dev/null
        
        print_success "Created package: ${package_name}"
    done
    
    # Generate checksums
    cd ${BUILD_DIR}/packages
    sha256sum *.tar.gz *.zip > checksums.txt
    cd - > /dev/null
    
    print_success "Release packages created in ${BUILD_DIR}/packages/"
}

# Function to build Docker image
build_docker() {
    DOCKER_TAG=${DOCKER_TAG:-"mysql-schema-sync:latest"}
    
    print_status "Building Docker image: ${DOCKER_TAG}"
    
    docker build -t ${DOCKER_TAG} .
    
    print_success "Docker image built: ${DOCKER_TAG}"
}

# Main script logic
case "${1:-help}" in
    "build")
        build_current
        ;;
    "build-all")
        build_all
        ;;
    "build-linux")
        build_linux
        ;;
    "build-darwin")
        build_darwin
        ;;
    "build-windows")
        build_windows
        ;;
    "clean")
        clean_build
        ;;
    "test")
        run_tests
        ;;
    "release")
        create_release
        ;;
    "docker")
        build_docker
        ;;
    "version")
        show_version
        ;;
    "help"|*)
        show_help
        ;;
esac