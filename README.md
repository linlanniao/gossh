# gossh - 批量 SSH 连接工具

gossh 是一个类似 ansible 的批量 SSH 连接工具，支持批量连接到多台 Linux 服务器执行命令或脚本。

## 项目背景与意义

在生产环境中，当需要管理大规模服务器集群时，传统的 Ansible 工具往往会遇到性能瓶颈。以管理超过 3000 台服务器为例：

### Ansible 的性能问题

- **资源消耗高**：在 4 核 8G 的 Ansible 控制机上，管理 3000 台机器时单核 CPU 占用率高达 100%
- **并发能力弱**：即使设置 50 并发数，4 核 CPU 也会完全占满
- **执行速度慢**：执行简单的 `ansible -m ping` 命令，3000 台机器需要将近 **15 分钟**

### gossh 的解决方案

gossh 专注于解决大规模服务器管理的性能问题：

- **提取核心功能**：将 Ansible 最常用的功能（ping、shell、文件上传等）独立实现，去除不必要的开销
- **兼容 Ansible 配置**：尽量兼容 `ansible.cfg`，无需修改现有 Ansible 配置即可覆盖常用功能
- **显著性能提升**：经过实践验证，简单的 ping 命令从 **15 分钟减少到 1 分钟**，性能提升约 **15 倍**

gossh 特别适合需要频繁执行批量操作的大规模生产环境，在保持与 Ansible 相似的使用体验的同时，大幅提升执行效率。

## 主要特性

- ✅ 支持 SSH key 认证和密码认证
- ✅ 并发执行，提高效率
- ✅ 支持从文件或命令行参数读取主机列表
- ✅ 支持执行命令和脚本文件
- ✅ 支持 Become 模式（类似 ansible 的 sudo 执行）
- ✅ 支持批量上传文件
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

### 一键安装（推荐）

```bash
curl -fsSL https://raw.githubusercontent.com/linlanniao/gossh/main/install.sh | bash
```

### 从 GitHub Releases 手动下载

访问 [GitHub Releases](https://github.com/linlanniao/gossh/releases) 下载对应平台的预编译二进制文件，解压后手动复制到系统 PATH 目录。

## 使用方法

### run 命令 - 批量执行命令

```bash
# 使用 -g 指定组名，只对指定分组的主机执行命令
gossh run -i ansible_hosts -g test -u root -c "df -h"
gossh run -i hosts.ini -g web_servers -u root -k ~/.ssh/id_rsa -c "uptime"

# 使用 -g all 选择所有分组的主机
gossh run -i hosts.txt -g all -u root -k ~/.ssh/id_rsa -c "uptime"

# 从目录读取所有 Ansible hosts 文件并聚合，选择所有分组
gossh run -i ansible_hosts -g all -u root -k ~/.ssh/id_rsa -c "uptime"

# 从命令行参数指定主机执行命令（逗号分隔，也需要指定 -g）
gossh run -i "192.168.1.10,192.168.1.11" -g all -u root -c "df -h"

# 使用 become 模式（sudo）执行命令
gossh run -i hosts.txt -g all -u root -c "systemctl restart nginx" --become
gossh run -i hosts.txt -g all -u root -c "whoami" --become --become-user appuser

# 指定并发数
gossh run -i hosts.txt -g all -u root -c "ls -la" -f 10
```

### script 命令 - 批量执行脚本文件

脚本会先上传到远程主机的临时目录，然后执行，执行完成后自动清理临时文件。

```bash
# 使用 -g 指定组名，只对指定分组的主机执行脚本
gossh script -i ansible_hosts -g test -u root -s deploy.sh
gossh script -i hosts.ini -g web_servers -u root -k ~/.ssh/id_rsa -s backup.sh

# 使用 -g all 选择所有分组的主机执行脚本
gossh script -i hosts.txt -g all -u root -k ~/.ssh/id_rsa -s deploy.sh

# 从目录读取所有 Ansible hosts 文件并聚合
gossh script -i ansible_hosts -g all -u root -k ~/.ssh/id_rsa -s deploy.sh

# 从命令行参数指定主机执行脚本（逗号分隔，也需要指定 -g）
gossh script -i "192.168.1.10,192.168.1.11" -g all -u root -s deploy.sh

# 使用 become 模式（sudo）执行脚本
gossh script -i hosts.txt -g all -u root -s deploy.sh --become
gossh script -i hosts.txt -g all -u root -s deploy.sh --become --become-user appuser

# 指定并发数
gossh script -i hosts.txt -g all -u root -s deploy.sh -f 10
```

### upload 命令 - 批量上传文件

```bash
# 使用 -g 指定组名，只对指定分组的主机上传文件
gossh upload -i ansible_hosts -g test -u root -l app.tar.gz -r /tmp/app.tar.gz
gossh upload -i hosts.ini -g web_servers -u root -k ~/.ssh/id_rsa -l config.conf -r /etc/config.conf

# 使用 -g all 选择所有分组的主机上传文件
gossh upload -i hosts.txt -g all -u root -k ~/.ssh/id_rsa -l app.tar.gz -r /tmp/app.tar.gz

# 从目录读取所有 Ansible hosts 文件并聚合
gossh upload -i ansible_hosts -g all -u root -k ~/.ssh/id_rsa -l app.tar.gz -r /tmp/app.tar.gz

# 从命令行参数指定主机上传文件（逗号分隔，也需要指定 -g）
gossh upload -i "192.168.1.10,192.168.1.11" -g all -u root -l app.tar.gz -r /tmp/app.tar.gz

# 指定文件权限
gossh upload -i hosts.txt -g all -u root -l script.sh -r /tmp/script.sh --mode 0755

# 指定并发数
gossh upload -i hosts.txt -g all -u root -l app.tar.gz -r /tmp/app.tar.gz -f 10
```

### ping 命令 - 测试 SSH 连接

```bash
# 使用 -g 指定组名，只测试指定分组的主机
gossh ping -i ansible_hosts -g test -u root
gossh ping -i hosts.ini -g web_servers -u root -k ~/.ssh/id_rsa

# 使用 -g all 测试所有分组的主机
gossh ping -i hosts.txt -g all -u root -k ~/.ssh/id_rsa

# 从目录读取所有 Ansible hosts 文件并聚合测试
gossh ping -i ansible_hosts -g all -u root -k ~/.ssh/id_rsa

# 从命令行参数指定主机测试连接（逗号分隔，也需要指定 -g）
gossh ping -i "192.168.1.10,192.168.1.11" -g all -u root

# 指定并发数
gossh ping -i hosts.txt -g all -u root -f 10
```

### list-host 命令 - 列出所有主机 IP 地址

```bash
# 使用 -g 指定组名，只列出指定分组的主机
gossh list-host -i ansible_hosts -g test
gossh list-host -i hosts.ini -g web_servers

# 使用 -g all 列出所有分组的主机
gossh list-host -i hosts.txt -g all

# 从目录列出所有主机
gossh list-host -i ansible_hosts -g all

# 从命令行参数列出主机（逗号分隔，也需要指定 -g）
gossh list-host -i "192.168.1.10,192.168.1.11" -g all

# 指定输出格式（ip/full/json）
gossh list-host -i ansible_hosts -g test --format full

# 一行输出（逗号分隔）
gossh list-host -i ansible_hosts -g test --one-line
```

### list-group 命令 - 列出所有组名

```bash
# 从文件列出所有组
gossh list-group -i ansible_hosts
gossh list-group -i hosts.ini

# 从目录列出所有组（递归读取所有文件）
gossh list-group -i hosts_dir/

# 从 ansible.cfg 配置的 inventory 列出所有组
gossh list-group

# 一行输出（逗号分隔）
gossh list-group -i ansible_hosts --one-line
```

### 参数说明

#### 全局参数（所有命令通用）

**主机列表相关**

- `-i, --inventory`: 主机列表（文件路径、目录路径或逗号分隔的主机列表）。如果指定目录，会递归读取目录下所有子文件并聚合，例如: `-i hosts.ini` 或 `-i hosts_dir/` 或 `-i 192.168.1.10,192.168.1.11`
- `-g, --group`: Ansible INI 格式的分组名称（必需）。使用 `-g all` 表示选择所有分组，支持逗号分隔的多个组，例如: `-g test` 或 `-g web_servers` 或 `-g all` 或 `-g test,web_servers`

**认证相关**

- `-u, --user`: SSH 用户名（可从 ansible.cfg 的 remote_user 读取）
- `-k, --key`: SSH 私钥路径（优先使用，可从 ansible.cfg 的 private_key_file 读取）
- `-p, --password`: SSH 密码（如果未提供 key）
- `-P, --port`: SSH 端口（默认: 22）

**执行相关**

- `-f, --forks`: 并发执行数量（默认: 5，可从 ansible.cfg 的 forks 读取）
- `-T, --timeout`: 连接超时时间（默认: 30s，可从 ansible.cfg 的 timeout 读取），例如: `30s`, `1m`, `2m30s`
- `--config-file`: 指定 ansible.cfg 配置文件路径。如果未指定，将按以下顺序查找：1) 环境变量 ANSIBLE_CONFIG 2) 当前目录及父目录的 ansible.cfg 3) ~/.ansible.cfg

#### run 命令专用参数

- `-c, --command`: 要执行的命令（必需）
- `--become`: 使用 sudo 执行命令（类似 ansible 的 become）
- `--become-user`: 使用 sudo 切换到指定用户执行命令（默认: root）
- `--show-output`: 显示命令输出（默认: true）
- `--log-dir`: 日志目录路径（可选，JSON 格式）。会自动生成文件名：run-时间戳.log

#### script 命令专用参数

- `-s, --script`: 要执行的脚本文件路径（必需）
- `--become`: 使用 sudo 执行脚本（类似 ansible 的 become）
- `--become-user`: 使用 sudo 切换到指定用户执行脚本（默认: root）
- `--show-output`: 显示命令输出（默认: true）
- `--log-dir`: 日志目录路径（可选，JSON 格式）。会自动生成文件名：script-时间戳.log

#### upload 命令专用参数

- `-l, --local`: 本地文件路径（必需）
- `-r, --remote`: 远程文件路径（必需）
- `--mode`: 文件权限（默认: 0644）
- `--show-output`: 显示命令输出（默认: true）
- `--log-dir`: 日志目录路径（可选，JSON 格式）。会自动生成文件名：upload-时间戳.log

#### list-host 命令专用参数

- `--format`: 输出格式: ip（仅 IP 地址）、full（完整信息）、json（JSON 格式），默认: ip
- `--one-line`: 一行输出（逗号分隔）

#### list-group 命令专用参数

- `--one-line`: 一行输出（逗号分隔）

### 主机列表文件格式

#### 普通格式

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

#### Ansible INI 格式

支持 Ansible 的 INI 格式主机文件，可以使用分组：

```ini
[web_servers]
192.168.1.10
192.168.1.11:2222
root@192.168.1.12

[db_servers]
192.168.1.20
192.168.1.21
```

**注意**：目前不支持 Ansible 的 `[group:children]` 子分组格式，仅支持直接定义主机列表的分组。

使用方式：

- `-i hosts.ini -g web_servers`: 只对 web_servers 分组的主机执行
- `-i ansible_hosts -g all`: 读取目录下所有文件并聚合所有主机
- `-i ansible_hosts -g web_servers`: 从目录中读取，但只对指定分组执行

## 示例

### 示例 1: 检查所有服务器的磁盘使用情况

```bash
gossh run -i hosts.txt -g all -u root -k ~/.ssh/id_rsa -c "df -h"
```

### 示例 2: 批量执行部署脚本

```bash
gossh script -i hosts.txt -g all -u deploy -k ~/.ssh/deploy_key -s deploy.sh -f 10
```

### 示例 2.1: 使用 become 模式执行需要权限的命令

```bash
# 使用 sudo 执行需要 root 权限的命令
gossh run -i hosts.txt -g all -u deploy -c "systemctl restart nginx" --become

# 切换到指定用户执行命令
gossh run -i hosts.txt -g all -u deploy -c "whoami" --become --become-user appuser
```

### 示例 2.2: 批量上传文件

```bash
# 上传文件到远程主机
gossh upload -i hosts.txt -g all -u root -l app.tar.gz -r /tmp/app.tar.gz

# 上传脚本文件并设置执行权限
gossh upload -i hosts.txt -g all -u root -l deploy.sh -r /tmp/deploy.sh --mode 0755
```

### 示例 3: 快速检查服务器状态

```bash
gossh run -i "server1,server2,server3" -g all -u admin -c "uptime && free -h"
```

### 示例 4: 使用密码认证

```bash
gossh run -i hosts.txt -g all -u root -p "your_password" -c "whoami"
```

### 示例 5: 测试 SSH 连接

```bash
# 测试所有主机的 SSH 连接是否可用
gossh ping -i hosts.txt -g all -u root -k ~/.ssh/id_rsa

# 快速测试几个主机的连接
gossh ping -i "192.168.1.10,192.168.1.11,192.168.1.12" -g all -u admin
```

### 示例 6: 列出主机和组

```bash
# 列出所有主机 IP
gossh list-host -i hosts.txt -g all

# 列出所有组名
gossh list-group -i hosts.txt

# 一行输出
gossh list-host -i hosts.txt -g all --one-line
gossh list-group -i hosts.txt --one-line
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
5. **脚本执行**: `script` 命令会将脚本上传到远程主机的 `/tmp/gossh_script_*.sh` 临时文件，执行完成后自动清理
6. **Become 模式**: 使用 `--become` 参数时，确保 SSH 用户有 sudo 权限且配置了无密码 sudo（或使用 `-p` 提供密码）

## 开发

```bash
# 运行测试
go test ./...

# 构建
go build -o gossh .

# 安装到系统
go install .
```

## 许可证

见 LICENSE 文件
