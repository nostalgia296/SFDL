#!/bin/bash
set -e

# SF轻小说下载器 - 多平台发布脚本

BINARY_NAME="sfdl"
BUILD_DIR="build"
VERSION=${1:-"dev"}
LDFLAGS="-s -w -X main.version=${VERSION}"

if [ "${VERSION}" = "dev" ]; then
    echo "Usage: ./scripts/release.sh <version>"
    echo "Example: ./scripts/release.sh 1.0.0"
    exit 1
fi

echo "Building ${BINARY_NAME} v${VERSION} for all platforms..."

# 清理旧构建
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# 定义平台
platforms=(
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
    "darwin/amd64"
    "darwin/arm64"
    "android/arm64"
)

for platform in "${platforms[@]}"; do
    GOOS=${platform%/*}
    GOARCH=${platform#*/}
    output_name="${BINARY_NAME}-${GOOS}-${GOARCH}"
    
    if [ "${GOOS}" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    echo "Building for ${GOOS}/${GOARCH}..."
    GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${output_name}" ./cmd/sfdl
done

echo "All builds complete!"
echo "Binaries location: ${BUILD_DIR}/"
ls -la "${BUILD_DIR}/"
