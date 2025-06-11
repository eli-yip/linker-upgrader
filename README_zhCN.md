# 🚀 Linker Universal Program Upgrade System

一个简单、安全、可配置的 Web 程序升级系统，支持多种文件格式的自动部署和服务管理。

[English](README_EN.md) | 中文

## ✨ 特性

- 🌐 **Web 界面**: 友好的用户界面，支持拖拽上传
- 📦 **多格式支持**: `.tar.gz`、`.zip`、`.gz`、可执行文件
- 🔧 **服务管理**: 自动停止/启动 systemd 服务
- 💾 **智能备份**: 自动备份现有程序，支持版本回滚
- 🔐 **权限管理**: 自动设置文件权限和可执行权限
- ⚙️ **高度可配置**: 支持配置文件、环境变量、命令行参数
- 🧹 **自动清理**: 定时清理临时文件
- 📊 **实时日志**: 详细的升级过程日志展示
- 🛡️ **安全检查**: 文件类型验证和大小限制

## 🎯 适用场景

- **上位机程序升级**: 工业控制系统程序远程升级
- **边缘设备部署**: IoT 设备程序自动更新
- **服务器应用升级**: 生产环境应用程序热更新
- **CI/CD 部署**: 持续集成/持续部署流水线
- **嵌入式系统**: 嵌入式设备程序升级

## 📋 系统要求

- **操作系统**: Linux (Ubuntu 18.04+, CentOS 7+, 其他发行版)
- **Go 版本**: 1.18 或更高版本
- **系统权限**: 建议以 root 权限运行 (用于服务管理)
- **系统工具**: `tar`, `unzip`, `systemctl` (可选)

## 🚀 快速开始

### 1. 下载和安装

```bash
# 克隆仓库
git clone https://github.com/linker-bot/linker-upgrader.git
cd upgrade-system

# 编译程序
go build -o upgrade-system main.go

# TBD
# 或者下载预编译的二进制文件
wget https://github.com/linker-bot/linker-upgrader/releases/latest/download/upgrade-system-linux-amd64
chmod +x upgrade-system-linux-amd64
```

### 2. 生成配置文件

```bash
# 生成默认配置文件
./upgrade-system -gen-config

# 配置文件将保存到 config.json
```

### 3. 启动服务

```bash
# 使用默认配置启动
./upgrade-system

# 指定端口启动
./upgrade-system -port 8080

# 指定配置文件启动
./upgrade-system -config /path/to/config.json
```

### 4. 访问 Web 界面

打开浏览器访问：`http://localhost:8080`

## ⚙️ 配置详解

### 配置文件 (config.json)

```json
{
  "upload_dir": "./uploads",                    // 上传临时目录
  "target_dir": "/opt/myapp",                   // 目标程序目录  
  "backup_dir": "/opt/myapp/backup",            // 备份目录
  "service_name": "myapp",                      // systemd 服务名
  "port": ":8080",                             // 服务端口
  "max_file_size": 100,                        // 最大文件大小 (MB)
  "enable_backup": true,                       // 启用备份功能
  "enable_service": true,                      // 启用服务管理
  "enable_cleanup": true,                      // 启用文件清理
  "cleanup_interval": 1,                       // 清理间隔 (小时)
  "file_max_age": 24,                         // 文件保留时间 (小时)
  "dir_permission": "0755",                    // 目录权限
  "file_permission": "0644",                   // 文件权限
  "exec_permission": "0755",                   // 可执行文件权限
  "title": "🚀 灵心巧手 - 上位机程序升级",      // 页面标题
  "description": "支持多种格式的程序升级系统",   // 页面描述
  "accept_types": [                           // 支持的文件类型
    ".tar.gz", ".zip", ".gz",
    "application/x-executable",
    "application/octet-stream"
  ]
}
```

### 环境变量配置

```bash
# 基本配置
export TARGET_DIR="/opt/production"
export SERVICE_NAME="prod-service"
export PORT="9090"
export MAX_FILE_SIZE="200"

# 功能开关
export ENABLE_BACKUP="true"
export ENABLE_SERVICE="true"
export ENABLE_CLEANUP="false"

# 界面定制
export TITLE="生产环境升级系统"
```

### 命令行参数

```bash
./upgrade-system -h
  -config string
        配置文件路径 (default "./config.json")
  -gen-config
        生成默认配置文件并退出
  -port string
        服务端口 (覆盖配置文件)
  -service string
        服务名称 (覆盖配置文件)
  -target string
        目标目录 (覆盖配置文件)
```

## 🔄 升级流程

系统会根据配置自动执行以下步骤：

1. **📤 文件上传**: 验证文件类型和大小
2. **⏹️ 停止服务**: 优雅停止当前运行的服务 (可选)
3. **💾 备份程序**: 备份现有程序到备份目录 (可选)
4. **📦 解压部署**: 根据文件类型自动解压或复制
5. **🔐 设置权限**: 自动设置目录和文件权限
6. **▶️ 启动服务**: 启动服务并验证状态 (可选)
7. **📊 状态报告**: 显示详细的升级日志

## 🛠️ 高级用法

### Systemd 服务配置

创建 systemd 服务文件 `/etc/systemd/system/upgrade-system.service`:

```ini
[Unit]
Description=Program Upgrade System
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/upgrade-system
ExecStart=/opt/upgrade-system/upgrade-system -config /etc/upgrade-system/config.json
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启用服务：

```bash
sudo systemctl enable upgrade-system
sudo systemctl start upgrade-system
```

### Docker 部署

#### 使用预构建镜像

```bash
# 拉取最新版本
docker pull ghcr.io/linker-bot/linker-upgrader:latest

# 运行容器
docker run -d -p 6110:6110 \
  --name linker-upgrader \
  -v /opt/myapp:/opt/myapp \
  -v ./config.json:/etc/linker-upgrader/config.json \
  ghcr.io/linker-bot/linker-upgrader:latest \
  -config /etc/linker-upgrader/config.json
```

#### 使用 Docker Compose

```yaml
services:
  linker-upgrader:
    image: ghcr.io/linker-bot/linker-upgrader:latest
    container_name: linker-upgrader
    ports:
      - "6110:6110"
    volumes:
      - /opt/myapp:/opt/myapp
      - ./config.json:/etc/linker-upgrader/config.json
    command: ["-config", "/etc/linker-upgrader/config.json"]
    restart: unless-stopped
```

### Nginx 反向代理

```nginx
server {
    listen 80;
    server_name upgrade.example.com;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        
        # 支持大文件上传
        client_max_body_size 200M;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
}
```

## 🔒 安全注意事项

- **权限管理**: 建议以最小权限原则运行
- **网络安全**: 在生产环境中使用 HTTPS 和身份认证
- **文件验证**: 上传前验证文件的完整性和来源
- **备份策略**: 定期清理备份文件，避免磁盘空间不足
- **日志监控**: 监控升级日志，及时发现异常情况

## 📚 API 文档

### Web 界面

- `GET /` - 主页面，显示上传表单
- `POST /upload` - 处理文件上传和程序升级

### 响应格式

成功响应会显示包含以下信息的页面：
- 升级状态 (成功/失败)
- 详细的操作日志
- 当前配置信息

## 🐛 故障排除

### 常见问题

**Q: 服务启动失败？**
```bash
# 检查端口是否被占用
netstat -tlnp | grep :8080

# 检查权限
ls -la upgrade-system
```

**Q: 文件上传失败？**
```bash
# 检查磁盘空间
df -h

# 检查上传目录权限
ls -la uploads/
```

**Q: 服务管理失败？**
```bash
# 检查 systemd 服务状态
systemctl status myapp

# 检查用户权限
id
```

### 日志查看

```bash
# 查看程序日志
tail -f /var/log/upgrade-system.log

# 查看 systemd 日志
journalctl -u upgrade-system -f
```

## 🤝 贡献指南

我们欢迎各种形式的贡献！

### 开发环境设置

```bash
# 克隆仓库
git clone https://github.com/yourusername/upgrade-system.git
cd upgrade-system

# 安装依赖
go mod tidy

# 运行测试
go test ./...

# 运行程序
go run main.go
```


### 代码规范

- 遵循 Go 官方代码规范
- 添加适当的注释和文档
- 编写单元测试
- 使用 `gofmt` 格式化代码

## 📄 许可证

本项目采用 Apachev2 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

---

⭐ 如果这个项目对你有帮助，请给我们一个 Star！