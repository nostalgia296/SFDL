#!/bin/bash
set -e

# SF轻小说下载器 - 构建脚本

BINARY_NAME="sfdl"
BUILD_DIR="build"
VERSION=${1:-"dev"}
LDFLAGS="-s -w -X main.version=${VERSION}"

echo "Building ${BINARY_NAME} v${VERSION}..."

# 清理旧构建
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# 当前平台
echo "Building for current platform..."
go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${BINARY_NAME}" ./cmd/sfdl

echo "Build complete!"
echo "Binary location: ${BUILD_DIR}/${BINARY_NAME}"
