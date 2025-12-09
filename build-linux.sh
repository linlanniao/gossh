#!/bin/bash

# 构建 Linux 版本的 gossh 工具
# 使用优化选项减小构建体积

set -e

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}开始构建 Linux 版本...${NC}"

# 设置构建参数
APP_NAME="gossh"
BUILD_DIR="build"
BINARY_NAME="${APP_NAME}-linux-amd64"
OUTPUT_PATH="${BUILD_DIR}/${BINARY_NAME}"

# 创建构建目录
mkdir -p "${BUILD_DIR}"

# 构建参数说明：
# - CGO_ENABLED=0: 禁用 CGO，生成静态二进制文件，减小体积
# - GOOS=linux: 目标操作系统为 Linux
# - GOARCH=amd64: 目标架构为 amd64
# - -ldflags="-s -w": 
#   -s: 去除符号表
#   -w: 去除调试信息
# - -trimpath: 去除文件系统中的路径信息
# - -a: 强制重新构建所有包

echo -e "${YELLOW}正在编译...${NC}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -trimpath \
    -a \
    -o "${OUTPUT_PATH}" \
    .

if [ $? -eq 0 ]; then
    # 获取文件大小
    SIZE=$(du -h "${OUTPUT_PATH}" | cut -f1)
    echo -e "${GREEN}✓ 构建成功！${NC}"
    echo -e "${GREEN}输出文件: ${OUTPUT_PATH}${NC}"
    echo -e "${GREEN}文件大小: ${SIZE}${NC}"
    
    # 显示文件信息
    echo ""
    echo "文件信息:"
    file "${OUTPUT_PATH}"
else
    echo -e "\033[0;31m✗ 构建失败！${NC}"
    exit 1
fi

