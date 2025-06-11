package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// 配置结构体
type Config struct {
	// 目录配置
	UploadDir string `json:"upload_dir"`
	TargetDir string `json:"target_dir"`
	BackupDir string `json:"backup_dir"`

	// 服务配置
	ServiceName string `json:"service_name"`
	Port        string `json:"port"`
	MaxFileSize int64  `json:"max_file_size"` // 单位: MB

	// 功能开关
	EnableBackup    bool `json:"enable_backup"`
	EnableService   bool `json:"enable_service"`
	EnableCleanup   bool `json:"enable_cleanup"`
	CleanupInterval int  `json:"cleanup_interval"` // 小时
	FileMaxAge      int  `json:"file_max_age"`     // 小时

	// 权限配置
	DirPermission  string `json:"dir_permission"`
	FilePermission string `json:"file_permission"`
	ExecPermission string `json:"exec_permission"`

	// 界面配置
	Title       string   `json:"title"`
	Description string   `json:"description"`
	AcceptTypes []string `json:"accept_types"`
}

// 默认配置
func getDefaultConfig() *Config {
	return &Config{
		UploadDir:       "./uploads",
		TargetDir:       "/opt/myapp",
		BackupDir:       "/opt/myapp/backup",
		ServiceName:     "myapp",
		Port:            ":8080",
		MaxFileSize:     100, // MB
		EnableBackup:    true,
		EnableService:   true,
		EnableCleanup:   true,
		CleanupInterval: 1,  // 1小时
		FileMaxAge:      24, // 24小时
		DirPermission:   "0755",
		FilePermission:  "0644",
		ExecPermission:  "0755",
		Title:           "🚀 灵心巧手 - 上位机程序升级",
		Description:     "支持 .tar.gz, .zip, 可执行文件的程序升级系统",
		AcceptTypes:     []string{".tar.gz", ".zip", ".gz", "application/x-executable", "application/octet-stream"},
	}
}

// 全局配置实例
var appConfig *Config

type UpgradeHandler struct{}

// 动态HTML模板
const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>{{.Config.Title}}</title>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; }
        .upload-form { margin: 20px 0; }
        .form-group { margin: 15px 0; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input[type="file"] { width: 100%; padding: 10px; border: 2px dashed #ddd; border-radius: 4px; display: block; width: 100%; padding: 10px 0;}
        input[type="submit"] { background: #007cba; color: white; padding: 12px 30px; border: none; border-radius: 4px; cursor: pointer; font-size: 16px; }
        input[type="submit"]:hover { background: #005a87; }
        .status { padding: 15px; margin: 15px 0; border-radius: 4px; }
        .success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
        .error { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
        .info { background: #d1ecf1; color: #0c5460; border: 1px solid #bee5eb; }
        .logs { background: #f8f9fa; border: 1px solid #dee2e6; padding: 15px; border-radius: 4px; font-family: monospace; white-space: pre-wrap; max-height: 300px; overflow-y: auto; }
        .config { background: #fff3cd; border: 1px solid #ffeaa7; padding: 10px; border-radius: 4px; font-size: 12px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>{{.Config.Title}}</h1>
        
        <div class="config">
            <strong>当前配置:</strong> 目标目录: {{.Config.TargetDir}} | 服务: {{.Config.ServiceName}} | 最大文件: {{.Config.MaxFileSize}}MB
        </div>
        
        {{if .Message}}
        <div class="status {{.MessageType}}">
            {{.Message}}
        </div>
        {{end}}
        
        {{if .Logs}}
        <div class="logs">{{.Logs}}</div>
        {{end}}
        
        <form class="upload-form" enctype="multipart/form-data" action="/upload" method="post">
            <div class="form-group">
                <label for="file">选择程序文件 ({{.Config.Description}}):</label>
                <input type="file" name="file" id="file" required accept="{{.AcceptTypesStr}}">
            </div>
            <div class="form-group">
                <input type="submit" value="上传并升级程序">
            </div>
        </form>
        
        <div class="info">
            <strong>升级流程说明:</strong><br>
            {{if .Config.EnableService}}1. 停止当前服务 ({{.Config.ServiceName}})<br>{{end}}
            {{if .Config.EnableBackup}}2. 备份现有程序到 {{.Config.BackupDir}}<br>{{end}}
            3. 部署新程序到 {{.Config.TargetDir}}<br>
            4. 设置权限 (目录:{{.Config.DirPermission}}, 文件:{{.Config.FilePermission}}, 可执行:{{.Config.ExecPermission}})<br>
            {{if .Config.EnableService}}5. 启动服务并验证状态<br>{{end}}
        </div>
    </div>
</body>
</html>
`

type PageData struct {
	Config         *Config
	Message        string
	MessageType    string
	Logs           string
	AcceptTypesStr string
}

func (h *UpgradeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("upload").Parse(htmlTemplate))
	data := PageData{
		Config:         appConfig,
		AcceptTypesStr: strings.Join(appConfig.AcceptTypes, ","),
	}
	tmpl.Execute(w, data)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// 使用配置中的文件大小限制
	maxSize := appConfig.MaxFileSize << 20 // MB to bytes
	r.ParseMultipartForm(maxSize)

	file, handler, err := r.FormFile("file")
	if err != nil {
		showResult(w, "上传失败: "+err.Error(), "error", "")
		return
	}
	defer file.Close()

	log.Printf("开始上传文件: %s, 大小: %d bytes", handler.Filename, handler.Size)

	// 创建上传目录
	if err := os.MkdirAll(appConfig.UploadDir, getPermission(appConfig.DirPermission)); err != nil {
		showResult(w, "创建上传目录失败: "+err.Error(), "error", "")
		return
	}

	// 保存上传的文件
	uploadPath := filepath.Join(appConfig.UploadDir, handler.Filename)
	dst, err := os.Create(uploadPath)
	if err != nil {
		showResult(w, "创建文件失败: "+err.Error(), "error", "")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		showResult(w, "保存文件失败: "+err.Error(), "error", "")
		return
	}

	// 执行升级
	logs, err := performUpgrade(uploadPath, handler.Filename)
	if err != nil {
		showResult(w, "升级失败: "+err.Error(), "error", logs)
		return
	}

	showResult(w, "程序升级成功！", "success", logs)
}

func performUpgrade(filePath, filename string) (string, error) {
	var logs strings.Builder

	logs.WriteString(fmt.Sprintf("开始升级程序: %s\n", filename))
	logs.WriteString(fmt.Sprintf("时间: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	logs.WriteString(fmt.Sprintf("配置: 目标=%s, 服务=%s\n\n", appConfig.TargetDir, appConfig.ServiceName))

	step := 1

	// 1. 停止服务（可选）
	if appConfig.EnableService {
		logs.WriteString(fmt.Sprintf("%d. 停止当前服务 (%s)...\n", step, appConfig.ServiceName))
		if err := runCommand("systemctl", "stop", appConfig.ServiceName); err != nil {
			logs.WriteString(fmt.Sprintf("   警告: 停止服务失败 (可能服务不存在): %v\n", err))
		} else {
			logs.WriteString("   ✓ 服务已停止\n")
		}
		step++
	}

	// 2. 创建必要目录
	logs.WriteString(fmt.Sprintf("\n%d. 创建必要目录...\n", step))
	dirs := []string{appConfig.TargetDir}
	if appConfig.EnableBackup {
		dirs = append(dirs, appConfig.BackupDir)
	}

	dirPerm := getPermission(appConfig.DirPermission)
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return logs.String(), fmt.Errorf("创建目录 %s 失败: %v", dir, err)
		}
		logs.WriteString(fmt.Sprintf("   ✓ 目录 %s 已准备 (权限:%s)\n", dir, appConfig.DirPermission))
	}
	step++

	// 3. 备份现有程序（可选）
	if appConfig.EnableBackup {
		logs.WriteString(fmt.Sprintf("\n%d. 备份现有程序...\n", step))
		backupPath := filepath.Join(appConfig.BackupDir, fmt.Sprintf("backup_%s.tar.gz", time.Now().Format("20060102_150405")))
		if err := runCommand("tar", "-czf", backupPath, "-C", appConfig.TargetDir, "."); err != nil {
			logs.WriteString(fmt.Sprintf("   警告: 备份失败 (可能没有现有程序): %v\n", err))
		} else {
			logs.WriteString(fmt.Sprintf("   ✓ 备份已保存到: %s\n", backupPath))
		}
		step++
	}

	// 4. 部署新程序
	logs.WriteString(fmt.Sprintf("\n%d. 部署新程序...\n", step))
	if err := deployProgram(filePath, filename, &logs); err != nil {
		return logs.String(), err
	}
	step++

	// 5. 设置权限
	logs.WriteString(fmt.Sprintf("\n%d. 设置程序权限...\n", step))
	if err := setPermissions(appConfig.TargetDir, &logs); err != nil {
		return logs.String(), err
	}
	step++

	// 6. 启动服务（可选）
	if appConfig.EnableService {
		logs.WriteString(fmt.Sprintf("\n%d. 启动服务 (%s)...\n", step, appConfig.ServiceName))
		if err := runCommand("systemctl", "start", appConfig.ServiceName); err != nil {
			logs.WriteString(fmt.Sprintf("   警告: 启动服务失败: %v\n", err))
			logs.WriteString("   请手动启动程序或检查服务配置\n")
		} else {
			logs.WriteString("   ✓ 服务已启动\n")

			// 等待一下再检查状态
			time.Sleep(2 * time.Second)
			if err := runCommand("systemctl", "is-active", appConfig.ServiceName); err != nil {
				logs.WriteString("   警告: 服务状态检查失败，请手动验证\n")
			} else {
				logs.WriteString("   ✓ 服务运行正常\n")
			}
		}
	}

	logs.WriteString(fmt.Sprintf("\n升级完成时间: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	return logs.String(), nil
}

func deployProgram(filePath, filename string, logs *strings.Builder) error {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".gz":
		if strings.HasSuffix(strings.ToLower(filename), ".tar.gz") {
			// tar.gz 文件
			logs.WriteString("   解压 tar.gz 文件...\n")
			if err := runCommand("tar", "-xzf", filePath, "-C", appConfig.TargetDir); err != nil {
				return fmt.Errorf("解压 tar.gz 失败: %v", err)
			}
		} else {
			// 单个 .gz 文件
			logs.WriteString("   解压 gz 文件...\n")
			outputPath := filepath.Join(appConfig.TargetDir, strings.TrimSuffix(filename, ".gz"))
			cmd := exec.Command("sh", "-c", fmt.Sprintf("gunzip -c %s > %s", filePath, outputPath))
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("解压 gz 文件失败: %v", err)
			}
		}
	case ".zip":
		logs.WriteString("   解压 zip 文件...\n")
		if err := runCommand("unzip", "-o", filePath, "-d", appConfig.TargetDir); err != nil {
			return fmt.Errorf("解压 zip 失败: %v", err)
		}
	default:
		// 直接复制可执行文件
		logs.WriteString("   复制可执行文件...\n")
		targetPath := filepath.Join(appConfig.TargetDir, filename)
		if err := copyFile(filePath, targetPath); err != nil {
			return fmt.Errorf("复制文件失败: %v", err)
		}
	}

	logs.WriteString("   ✓ 程序部署完成\n")
	return nil
}

func setPermissions(targetDir string, logs *strings.Builder) error {
	dirPerm := getPermission(appConfig.DirPermission)
	filePerm := getPermission(appConfig.FilePermission)
	execPerm := getPermission(appConfig.ExecPermission)

	// 遍历目录，为可执行文件设置权限
	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 为所有文件设置适当权限
		if info.IsDir() {
			os.Chmod(path, dirPerm)
		} else {
			// 检查是否为可执行文件
			if isExecutable(path) {
				os.Chmod(path, execPerm)
				logs.WriteString(fmt.Sprintf("   ✓ 设置可执行权限 (%s): %s\n", appConfig.ExecPermission, path))
			} else {
				os.Chmod(path, filePerm)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("设置权限失败: %v", err)
	}

	return nil
}

func isExecutable(filePath string) bool {
	// 检查文件是否为可执行文件
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// 读取文件头部判断是否为 ELF 文件
	header := make([]byte, 4)
	if _, err := file.Read(header); err != nil {
		return false
	}

	// ELF 魔数: 0x7F 'E' 'L' 'F'
	if len(header) >= 4 && header[0] == 0x7F && header[1] == 'E' && header[2] == 'L' && header[3] == 'F' {
		return true
	}

	// 也可以检查文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == "" || ext == ".bin" || ext == ".exe"
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func showResult(w http.ResponseWriter, message, messageType, logs string) {
	tmpl := template.Must(template.New("upload").Parse(htmlTemplate))
	data := PageData{
		Config:         appConfig,
		Message:        message,
		MessageType:    messageType,
		Logs:           logs,
		AcceptTypesStr: strings.Join(appConfig.AcceptTypes, ","),
	}
	tmpl.Execute(w, data)
}

// 工具函数：将字符串权限转换为 os.FileMode
func getPermission(permStr string) os.FileMode {
	if perm, err := strconv.ParseUint(permStr, 8, 32); err == nil {
		return os.FileMode(perm)
	}
	return 0755 // 默认权限
}

// 加载配置文件
func loadConfig(configPath string) (*Config, error) {
	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("配置文件不存在，创建默认配置: %s", configPath)
		defaultConfig := getDefaultConfig()
		if err := saveConfig(configPath, defaultConfig); err != nil {
			log.Printf("创建默认配置文件失败: %v", err)
		}
		return defaultConfig, nil
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &config, nil
}

// 保存配置文件
func saveConfig(configPath string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

// 从环境变量覆盖配置
func overrideConfigFromEnv(config *Config) {
	if val := os.Getenv("UPLOAD_DIR"); val != "" {
		config.UploadDir = val
	}
	if val := os.Getenv("TARGET_DIR"); val != "" {
		config.TargetDir = val
	}
	if val := os.Getenv("BACKUP_DIR"); val != "" {
		config.BackupDir = val
	}
	if val := os.Getenv("SERVICE_NAME"); val != "" {
		config.ServiceName = val
	}
	if val := os.Getenv("PORT"); val != "" {
		config.Port = val
	}
	if val := os.Getenv("MAX_FILE_SIZE"); val != "" {
		if size, err := strconv.ParseInt(val, 10, 64); err == nil {
			config.MaxFileSize = size
		}
	}
	if val := os.Getenv("ENABLE_BACKUP"); val != "" {
		config.EnableBackup = val == "true"
	}
	if val := os.Getenv("ENABLE_SERVICE"); val != "" {
		config.EnableService = val == "true"
	}
	if val := os.Getenv("TITLE"); val != "" {
		config.Title = val
	}
}

func main() {
	// 命令行参数
	var (
		configPath  = flag.String("config", "./config.json", "配置文件路径")
		port        = flag.String("port", "", "服务端口 (覆盖配置文件)")
		targetDir   = flag.String("target", "", "目标目录 (覆盖配置文件)")
		serviceName = flag.String("service", "", "服务名称 (覆盖配置文件)")
		genConfig   = flag.Bool("gen-config", false, "生成默认配置文件并退出")
	)
	flag.Parse()

	// 生成配置文件
	if *genConfig {
		defaultConfig := getDefaultConfig()
		if err := saveConfig(*configPath, defaultConfig); err != nil {
			log.Fatalf("生成配置文件失败: %v", err)
		}
		log.Printf("已生成默认配置文件: %s", *configPath)
		return
	}

	// 加载配置
	var err error
	appConfig, err = loadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 从环境变量覆盖配置
	overrideConfigFromEnv(appConfig)

	// 从命令行参数覆盖配置
	if *port != "" {
		appConfig.Port = *port
	}
	if *targetDir != "" {
		appConfig.TargetDir = *targetDir
	}
	if *serviceName != "" {
		appConfig.ServiceName = *serviceName
	}

	// 确保端口格式正确
	if !strings.HasPrefix(appConfig.Port, ":") {
		appConfig.Port = ":" + appConfig.Port
	}

	// 检查是否以 root 权限运行
	if os.Geteuid() != 0 && appConfig.EnableService {
		log.Println("警告: 建议以 root 权限运行以确保能够操作系统服务")
	}

	// 启动清理任务（可选）
	if appConfig.EnableCleanup {
		go func() {
			interval := time.Duration(appConfig.CleanupInterval) * time.Hour
			maxAge := time.Duration(appConfig.FileMaxAge) * time.Hour
			for {
				time.Sleep(interval)
				cleanupOldFiles(appConfig.UploadDir, maxAge)
			}
		}()
	}

	// 设置路由
	http.Handle("/", &UpgradeHandler{})
	http.HandleFunc("/upload", uploadHandler)

	// 启动服务器
	log.Printf("程序升级系统启动成功")
	log.Printf("配置文件: %s", *configPath)
	log.Printf("访问地址: http://localhost%s", appConfig.Port)
	log.Printf("目标目录: %s", appConfig.TargetDir)
	log.Printf("服务名称: %s", appConfig.ServiceName)
	log.Printf("备份功能: %v", appConfig.EnableBackup)
	log.Printf("服务管理: %v", appConfig.EnableService)
	log.Printf("文件清理: %v", appConfig.EnableCleanup)

	if err := http.ListenAndServe(appConfig.Port, nil); err != nil {
		log.Fatal("启动服务器失败:", err)
	}
}

func cleanupOldFiles(dir string, maxAge time.Duration) {
	log.Printf("开始清理旧文件: %s (超过 %v)", dir, maxAge)
	count := 0
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if time.Since(info.ModTime()) > maxAge {
			if err := os.Remove(path); err == nil {
				count++
				log.Printf("清理文件: %s", path)
			}
		}
		return nil
	})
	log.Printf("清理完成，共删除 %d 个文件", count)
}
