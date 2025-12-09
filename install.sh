#!/bin/bash
set -e

# 创建临时目录
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

# 检测操作系统和架构
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# 映射架构名称
case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "不支持的架构: $ARCH"
        exit 1
        ;;
esac

# 检查操作系统
if [ "$OS" != "linux" ] && [ "$OS" != "darwin" ]; then
    echo "不支持的操作系统: $OS"
    exit 1
fi

# 获取最新版本号
GITHUB_REPO="yourusername/gossh"  # 请替换为实际的仓库地址
LATEST_VERSION=$(curl -s https://api.github.com/repos/$GITHUB_REPO/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo "无法获取最新版本号"
    exit 1
fi

# 下载并解压
DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/$LATEST_VERSION/gossh_${LATEST_VERSION#v}_${OS}_${ARCH}.tar.gz"
cd $TMPDIR
curl -L -o gossh.tar.gz "$DOWNLOAD_URL"
tar -xzf gossh.tar.gz

# 安装到系统目录
sudo cp gossh /usr/local/bin/gossh
sudo chmod +x /usr/local/bin/gossh

echo "安装完成！版本: $LATEST_VERSION"

