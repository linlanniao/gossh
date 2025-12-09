#!/bin/bash

# 构建多个平台的 gossh 工具
# 使用优化选项减小构建体积

set -e

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  构建多平台 gossh 工具${NC}"
echo -e "${BLUE}========================================${NC}"

# 设置构建参数
APP_NAME="gossh"
BUILD_DIR="build"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date +"%Y-%m-%d %H:%M:%S")

# 创建构建目录
mkdir -p "${BUILD_DIR}"

# 构建目标平台列表
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# 构建函数
build_platform() {
    local platform=$1
    local os=$(echo $platform | cut -d'/' -f1)
    local arch=$(echo $platform | cut -d'/' -f2)
    local ext=""
    
    if [ "$os" = "windows" ]; then
        ext=".exe"
    fi
    
    local binary_name="${APP_NAME}-${os}-${arch}${ext}"
    local output_path="${BUILD_DIR}/${binary_name}"
    
    echo ""
    echo -e "${YELLOW}正在构建 ${os}/${arch}...${NC}"
    
    CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build \
        -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}'" \
        -trimpath \
        -a \
        -o "${output_path}" \
        .
    
    if [ $? -eq 0 ]; then
        local size=$(du -h "${output_path}" | cut -f1)
        echo -e "${GREEN}✓ ${os}/${arch} 构建成功 (${size})${NC}"
    else
        echo -e "${RED}✗ ${os}/${arch} 构建失败${NC}"
        return 1
    fi
}

# 构建所有平台
for platform in "${PLATFORMS[@]}"; do
    build_platform "$platform"
done

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}所有平台构建完成！${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "构建文件列表:"
ls -lh "${BUILD_DIR}"/* | awk '{print $9, "(" $5 ")"}'

