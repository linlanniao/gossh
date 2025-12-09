# gossh - 批量 SSH 连接工具

gossh 是一个类似 ansible 的批量 SSH 连接工具，支持批量连接到多台 Linux 服务器执行命令或脚本。

## 主要特性

- ✅ 支持 SSH key 认证和密码认证
- ✅ 并发执行，提高效率
- ✅ 支持从文件或命令行参数读取主机列表
- ✅ 支持执行命令和脚本文件
- ✅ 支持连接测试（ping 功能）
- ✅ 详细的执行结果输出
- ✅ 可配置并发数量

## 安装

### 从源码构建

```bash
go build -o gossh .
```

或者直接使用：

```bash
go run .
```

### 从 GitHub Releases 下载

访问 [GitHub Releases](https://github.com/yourusername/gossh/releases) 下载对应平台的预编译二进制文件。

### 使用 GoReleaser 构建

项目使用 [GoReleaser](https://goreleaser.com) 进行自动化构建和发布：

```bash
# 安装 GoReleaser
go install github.com/goreleaser/goreleaser@latest

# 本地测试构建（不发布）
goreleaser build --snapshot

# 发布新版本（需要设置 GITHUB_TOKEN）
goreleaser release
```

GoReleaser 会自动为 Linux 和 macOS 平台构建二进制文件，并生成相应的压缩包。

## 使用方法

### run 命令 - 批量执行命令或脚本

```bash
# 从文件读取主机列表执行命令
gossh run -f hosts.txt -u root -k ~/.ssh/id_rsa -c "uptime"

# 从命令行参数指定主机执行命令
gossh run -H "192.168.1.10,192.168.1.11" -u root -c "df -h"

# 执行脚本文件
gossh run -f hosts.txt -u root -s deploy.sh

# 指定并发数
gossh run -f hosts.txt -u root -c "ls -la" --concurrency 10
```

### ping 命令 - 测试 SSH 连接

```bash
# 从文件读取主机列表测试连接
gossh ping -f hosts.txt -u root -k ~/.ssh/id_rsa

# 从命令行参数指定主机测试连接
gossh ping -H "192.168.1.10,192.168.1.11" -u root

# 指定并发数
gossh ping -f hosts.txt -u root --concurrency 10
```

### version 命令 - 查看版本信息

```bash
# 查看版本信息
gossh version

# 或使用简写
gossh v
```

版本信息包括：
- 版本号
- Git 提交哈希
- 分支名
- 构建用户
- 构建日期
- Go 版本和平台信息

### 参数说明

#### 主机列表相关
- `-f, --file`: 主机列表文件路径
- `-H, --hosts`: 主机列表（逗号分隔），例如: `192.168.1.10,192.168.1.11`

#### 认证相关
- `-u, --user`: SSH 用户名（必需）
- `-k, --key`: SSH 私钥路径（优先使用）
- `-p, --password`: SSH 密码（如果未提供 key）
- `-P, --port`: SSH 端口（默认: 22）

#### 执行相关
- `-c, --command`: 要执行的命令
- `-s, --script`: 要执行的脚本文件路径
- `--concurrency`: 并发执行数量（默认: 5）
- `--show-output`: 显示命令输出（默认: true）

### 主机列表文件格式

主机列表文件支持以下格式：

```
# 注释行
192.168.1.10              # 默认端口 22
192.168.1.11:22           # 指定端口
root@192.168.1.12         # 指定用户
admin@192.168.1.13:2222   # 指定用户和端口
```

每行一个主机，支持：
- 空行和以 `#` 开头的注释行会被忽略
- 格式：`[user@]host[:port]`
- 如果不指定用户，使用 `-u` 参数指定的用户
- 如果不指定端口，使用 `-P` 参数指定的端口（默认 22）

## 示例

### 示例 1: 检查所有服务器的磁盘使用情况

```bash
gossh run -f hosts.txt -u root -k ~/.ssh/id_rsa -c "df -h"
```

### 示例 2: 批量执行部署脚本

```bash
gossh run -f hosts.txt -u deploy -k ~/.ssh/deploy_key -s deploy.sh --concurrency 10
```

### 示例 3: 快速检查服务器状态

```bash
gossh run -H "server1,server2,server3" -u admin -c "uptime && free -h"
```

### 示例 4: 使用密码认证

```bash
gossh run -f hosts.txt -u root -p "your_password" -c "whoami"
```

### 示例 5: 测试 SSH 连接

```bash
# 测试所有主机的 SSH 连接是否可用
gossh ping -f hosts.txt -u root -k ~/.ssh/id_rsa

# 快速测试几个主机的连接
gossh ping -H "192.168.1.10,192.168.1.11,192.168.1.12" -u admin
```

## 输出格式

### run 命令输出

工具会显示每个主机的执行结果：

```
================================================================================
执行结果汇总
================================================================================

[✓] 192.168.1.10 - 执行成功
标准输出:
 15:30:45 up 10 days,  2:15,  1 user,  load average: 0.05, 0.10, 0.15

--------------------------------------------------------------------------------

[✗] 192.168.1.11 - 执行失败 (退出码: 1)
标准错误:
command not found

--------------------------------------------------------------------------------

总计: 2 台主机 | 成功: 1 | 失败: 1
================================================================================
```

### ping 命令输出

连接测试会显示每个主机的连接状态和延迟：

```
================================================================================
SSH 连接测试结果
================================================================================

[✓] 192.168.1.10 - 连接成功 (延迟: 45ms)
[✓] 192.168.1.11 - 连接成功 (延迟: 52ms)
[✗] 192.168.1.12 - 连接失败
  错误: dial tcp 192.168.1.12:22: connect: connection refused

--------------------------------------------------------------------------------
总计: 3 台主机 | 成功: 2 | 失败: 1
================================================================================
```

## 注意事项

1. **安全性**: 当前版本使用 `InsecureIgnoreHostKey()`，生产环境建议实现 host key 验证
2. **SSH Key**: 如果未指定 key 路径，工具会尝试使用 `~/.ssh/id_rsa`
3. **并发控制**: 默认并发数为 5，可以根据网络和服务器性能调整
4. **错误处理**: 连接失败或执行失败的主机会在结果中标记，不会中断其他主机的执行

## 开发

```bash
# 运行测试
go test ./...

# 构建
go build -o gossh .

# 安装到系统
go install .

# 使用 GoReleaser 本地构建（测试）
goreleaser build --snapshot
```

### 版本信息

版本信息通过构建时的 ldflags 注入到二进制文件中，包括：
- `Version`: Git 标签版本
- `Revision`: Git 完整提交哈希
- `Branch`: Git 分支名
- `BuildUser`: 构建用户
- `BuildDate`: 构建日期

这些信息可以通过 `gossh version` 命令查看。

## 许可证

见 LICENSE 文件

