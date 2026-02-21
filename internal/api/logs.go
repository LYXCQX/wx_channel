package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"wx_channel/internal/config"
	"wx_channel/internal/response"
	"wx_channel/internal/utils"
)

// LogsService 日志服务
type LogsService struct {
	logFile string
}

// NewLogsService 创建日志服务
func NewLogsService(cfg *config.Config) *LogsService {
	logFile := "logs/wx_channel.log"
	if cfg != nil && strings.TrimSpace(cfg.LogFile) != "" {
		logFile = strings.TrimSpace(cfg.LogFile)
	}
	return &LogsService{
		logFile: logFile,
	}
}

// LogEntry 日志条目结构 (用于解析 JSON 日志)
type LogEntry struct {
	Level   string      `json:"level"`
	Time    string      `json:"time"`
	Message string      `json:"message"`
	Caller  string      `json:"caller,omitempty"`
	Fields  interface{} `json:"fields,omitempty"`
}

// getLogFilePath 获取日志文件路径
func (s *LogsService) getLogFilePath() string {
	if s == nil {
		return filepath.Join(".", "logs", "wx_channel.log")
	}

	// 已经是绝对路径时直接使用
	if filepath.IsAbs(s.logFile) {
		return s.logFile
	}

	// 获取可执行文件所在目录
	baseDir, err := utils.GetBaseDir()
	if err != nil {
		// 如果获取失败，回退到当前目录
		return filepath.Join(".", s.logFile)
	}
	return filepath.Join(baseDir, s.logFile)
}

// GetLogs 获取日志内容 (倒序，支持行数限制)
func (s *LogsService) GetLogs(w http.ResponseWriter, r *http.Request) {
	logPath := s.getLogFilePath()

	// 如果文件不存在
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		response.Success(w, []string{})
		return
	}

	// 打开文件
	file, err := os.Open(logPath)
	if err != nil {
		response.Error(w, 500, "Failed to open log file")
		return
	}
	defer file.Close()

	// 读取参数
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // 默认返回最后 100 行
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	// 简单的读取最后 N 行实现 (如果文件太大，这种方式效率低，但对于普通日志够用)
	// 更好的方式是 seek 到末尾然后向前读
	content, err := io.ReadAll(file)
	if err != nil {
		response.Error(w, 500, "Failed to read log file")
		return
	}

	lines := strings.Split(string(content), "\n")

	// 过滤空行
	var validLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			validLines = append(validLines, line)
		}
	}

	totalLines := len(validLines)
	start := 0
	if totalLines > limit {
		start = totalLines - limit
	}

	// 截取最后 N 行
	result := validLines[start:]

	// 倒序排列 (最新的在最前)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	// 尝试解析为 JSON 对象 (如果日志是 JSON 格式)
	// 由于 logger.go 中目前我们还没强制所有日志为 JSON (file 是 ConsoleWriter NoColor)，所以可能是文本
	// 这里直接返回字符串行
	response.Success(w, result)
}

// DownloadLogs 下载日志文件
func (s *LogsService) DownloadLogs(w http.ResponseWriter, r *http.Request) {
	logPath := s.getLogFilePath()

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		response.Error(w, 404, "Log file not found")
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=wx_channel.log")
	w.Header().Set("Content-Type", "text/plain")
	http.ServeFile(w, r, logPath)
}

// ClearLogs 清空日志
func (s *LogsService) ClearLogs(w http.ResponseWriter, r *http.Request) {
	logPath := s.getLogFilePath()

	// 截断文件
	if err := os.Truncate(logPath, 0); err != nil {
		response.Error(w, 500, "Failed to clear logs: "+err.Error())
		return
	}

	response.Success(w, "Logs cleared")
}

// RegisterRoutes 注册路由
func (s *LogsService) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/logs", s.GetLogs)
	mux.HandleFunc("/api/v1/logs/download", s.DownloadLogs)
	// DELETE 需要单独处理或在 Helper 中处理，ServeMux 不直接支持 Method 匹配，需要在 Handler 内部判断
	// 这里通过 Wrapper 或者在内部判断 Method
	mux.HandleFunc("/api/v1/logs/clear", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete || r.Method == http.MethodPost {
			s.ClearLogs(w, r)
		} else {
			response.Error(w, 405, "Method not allowed")
		}
	})
}
