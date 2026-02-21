package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"wx_channel/internal/config"
	"wx_channel/internal/database"
	"wx_channel/internal/response"
	"wx_channel/internal/services"
	"wx_channel/internal/utils"

	"github.com/qtgolang/SunnyNet/SunnyNet"
)

// BatchHandler 批量下载处理器
type BatchHandler struct {
	downloadService *services.DownloadRecordService
	gopeedService   *services.GopeedService // Injected Gopeed Service
	mu              sync.RWMutex
	tasks           []BatchTask
	running         bool
	cancelFunc      context.CancelFunc // 用于取消时立即中断下载
}

// BatchTask 批量下载任务
type BatchTask struct {
	ID              string  `json:"id"`
	URL             string  `json:"url"`
	Title           string  `json:"title"`
	AuthorName      string  `json:"authorName,omitempty"`      // 兼容旧格式
	Author          string  `json:"author,omitempty"`          // 新格式
	Key             string  `json:"key,omitempty"`             // 加密密钥（新方式，后端生成解密数组）
	DecryptorPrefix string  `json:"decryptorPrefix,omitempty"` // 解密前缀（旧方式，前端传递）
	PrefixLen       int     `json:"prefixLen,omitempty"`
	Status          string  `json:"status"` // pending, downloading, done, failed
	Error           string  `json:"error,omitempty"`
	Progress        float64 `json:"progress,omitempty"`
	DownloadedMB    float64 `json:"downloadedMB,omitempty"`
	TotalMB         float64 `json:"totalMB,omitempty"`
	// 额外字段用于下载记录（批量下载JSON格式）
	Duration   string `json:"duration,omitempty"`   // 时长字符串，如 "00:22"
	SizeMB     string `json:"sizeMB,omitempty"`     // 大小字符串，如 "28.77MB"
	Cover      string `json:"cover,omitempty"`      // 封面URL（批量下载格式）
	Resolution string `json:"resolution,omitempty"` // 分辨率
	PageSource string `json:"pageSource,omitempty"` // 页面来源（batch_console/batch_feed/batch_home等）
	// 统计数据字段
	PlayCount    string `json:"playCount,omitempty"`    // 播放量（字符串格式）
	LikeCount    string `json:"likeCount,omitempty"`    // 点赞数（字符串格式）
	CommentCount string `json:"commentCount,omitempty"` // 评论数（字符串格式）
	FavCount     string `json:"favCount,omitempty"`     // 收藏数（字符串格式）
	ForwardCount string `json:"forwardCount,omitempty"` // 转发数（字符串格式）
	CreateTime   string `json:"createTime,omitempty"`   // 创建时间
	IPRegion     string `json:"ipRegion,omitempty"`     // IP所在地
	// 兼容数据库导出格式
	VideoURL   string `json:"videoUrl,omitempty"`   // 视频URL（数据库格式）
	CoverURL   string `json:"coverUrl,omitempty"`   // 封面URL（数据库格式）
	DecryptKey string `json:"decryptKey,omitempty"` // 解密密钥（数据库格式）
	DurationMs int64  `json:"durationMs,omitempty"` // 时长毫秒（数据库格式，字段名为duration但类型是int64）
	Size       int64  `json:"size,omitempty"`       // 大小字节（数据库格式）
}

// GetAuthor 获取作者名称，兼容两种字段
func (t *BatchTask) GetAuthor() string {
	if t.Author != "" {
		return t.Author
	}
	return t.AuthorName
}

// GetURL 获取视频URL，兼容两种格式
func (t *BatchTask) GetURL() string {
	if t.URL != "" {
		return t.URL
	}
	return t.VideoURL
}

// GetKey 获取解密密钥，兼容两种格式
func (t *BatchTask) GetKey() string {
	if t.Key != "" {
		return t.Key
	}
	return t.DecryptKey
}

// Handle implements router.Interceptor
func (h *BatchHandler) Handle(Conn *SunnyNet.HttpConn) bool {
	// Defensive checks
	if h == nil {
		return false
	}
	if Conn == nil || Conn.Request == nil || Conn.Request.URL == nil {
		return false
	}

	// Debug log
	// utils.Info("BatchHandler checking: %s", Conn.Request.URL.Path)

	if h.HandleBatchStart(Conn) {
		return true
	}
	if h.HandleBatchProgress(Conn) {
		return true
	}
	if h.HandleBatchCancel(Conn) {
		return true
	}
	if h.HandleBatchResume(Conn) {
		return true
	}
	if h.HandleBatchClear(Conn) {
		return true
	}
	if h.HandleBatchFailed(Conn) {
		return true
	}
	return false
}

// GetCover 获取封面URL，兼容两种格式
func (t *BatchTask) GetCover() string {
	if t.Cover != "" {
		return t.Cover
	}
	return t.CoverURL
}

// NewBatchHandler 创建批量下载处理器
func NewBatchHandler(cfg *config.Config, gopeedService *services.GopeedService) *BatchHandler {
	return &BatchHandler{
		downloadService: services.NewDownloadRecordService(),
		gopeedService:   gopeedService,
		tasks:           make([]BatchTask, 0),
	}
}

// getConfig 获取当前配置（动态获取最新配置）
func (h *BatchHandler) getConfig() *config.Config {
	return config.Get()
}

// getDownloadsDir 获取解析后的下载目录
func (h *BatchHandler) getDownloadsDir() (string, error) {
	cfg := h.getConfig()
	return cfg.GetResolvedDownloadsDir()
}

// HandleBatchStart 处理批量下载开始请求
func (h *BatchHandler) HandleBatchStart(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/batch_start" {
		return false
	}

	utils.Info("📥 [批量下载] 收到 batch_start 请求")

	// 处理 CORS 预检请求
	if Conn.Request.Method == "OPTIONS" {
		h.sendSuccessResponse(Conn, map[string]interface{}{"message": "OK"})
		return true
	}

	// 只处理 POST 请求
	if Conn.Request.Method != "POST" {
		h.sendErrorResponse(Conn, fmt.Errorf("method not allowed: %s", Conn.Request.Method))
		return true
	}

	// 授权校验
	if h.getConfig() != nil && h.getConfig().SecretToken != "" {
		if Conn.Request.Header.Get("X-Local-Auth") != h.getConfig().SecretToken {
			h.sendErrorResponse(Conn, fmt.Errorf("unauthorized"))
			return true
		}
	}

	utils.Info("📥 [批量下载] 开始读取请求体...")

	// 检查请求体是否为空
	if Conn.Request.Body == nil {
		err := fmt.Errorf("request body is nil")
		utils.HandleError(err, "读取batch_start请求体")
		h.sendErrorResponse(Conn, err)
		return true
	}

	body, err := io.ReadAll(Conn.Request.Body)
	if err != nil {
		utils.HandleError(err, "读取batch_start请求体")
		h.sendErrorResponse(Conn, err)
		return true
	}
	defer Conn.Request.Body.Close()

	bodySize := len(body)
	utils.Info("📥 [批量下载] 请求体大小: %.2f MB", float64(bodySize)/(1024*1024))

	var req struct {
		Videos          []BatchTask `json:"videos"`
		ForceRedownload bool        `json:"forceRedownload"`
		PageSource      string      `json:"pageSource,omitempty"` // 页面来源
	}

	utils.Info("📥 [批量下载] 开始解析 JSON...")
	if err := json.Unmarshal(body, &req); err != nil {
		utils.HandleError(err, "解析batch_start JSON")
		h.sendErrorResponse(Conn, err)
		return true
	}
	utils.Info("📥 [批量下载] JSON 解析完成，视频数: %d", len(req.Videos))

	// 判断批量下载来源
	pageSource := req.PageSource
	if pageSource == "" {
		// 如果请求体中没有指定，则通过请求头判断
		origin := Conn.Request.Header.Get("Origin")
		referer := Conn.Request.Header.Get("Referer")

		if strings.Contains(origin, "channels.weixin.qq.com") || strings.Contains(referer, "channels.weixin.qq.com") {
			// 从视频号页面发起的请求，尝试从Referer中提取页面类型
			if strings.Contains(referer, "/web/pages/feed") {
				pageSource = "batch_feed"
			} else if strings.Contains(referer, "/web/pages/home") {
				pageSource = "batch_home"
			} else if strings.Contains(referer, "/web/pages/profile") {
				pageSource = "batch_profile"
			} else if strings.Contains(referer, "/web/pages/s") {
				pageSource = "batch_search" // 搜索页面批量下载
			} else {
				pageSource = "batch_channels" // 默认标记为视频号批量下载
			}
		} else {
			// 从Web控制台发起的请求
			pageSource = "batch_console"
		}
	}
	utils.Info("📥 [批量下载] 来源: %s", pageSource)

	if len(req.Videos) == 0 {
		h.sendErrorResponse(Conn, fmt.Errorf("视频列表为空"))
		return true
	}

	// 初始化任务
	h.mu.Lock()
	h.tasks = make([]BatchTask, len(req.Videos))
	for i, v := range req.Videos {
		h.tasks[i] = BatchTask{
			ID:              v.ID,
			URL:             v.URL,
			Title:           v.Title,
			AuthorName:      v.GetAuthor(), // 兼容 author 和 authorName
			Author:          v.Author,
			Key:             v.Key,
			DecryptorPrefix: v.DecryptorPrefix,
			PrefixLen:       v.PrefixLen,
			Status:          "pending",
			// 保留额外字段
			Duration:     v.Duration,
			SizeMB:       v.SizeMB,
			Cover:        v.Cover,
			Resolution:   v.Resolution,
			PageSource:   pageSource, // 保存页面来源
			PlayCount:    v.PlayCount,
			LikeCount:    v.LikeCount,
			CommentCount: v.CommentCount,
			FavCount:     v.FavCount,
			ForwardCount: v.ForwardCount,
			CreateTime:   v.CreateTime,
			IPRegion:     v.IPRegion,
		}
	}
	h.running = true
	h.mu.Unlock()

	// 获取并发数配置
	concurrency := 5 // 默认值（与配置默认值一致）
	if h.getConfig() != nil && h.getConfig().DownloadConcurrency > 0 {
		concurrency = h.getConfig().DownloadConcurrency
	}

	utils.Info("🚀 [批量下载] 开始下载 %d 个视频，并发数: %d", len(req.Videos), concurrency)

	// 启动后台下载
	go h.startBatchDownload(req.ForceRedownload)

	h.sendSuccessResponse(Conn, map[string]interface{}{
		"total":       len(req.Videos),
		"concurrency": concurrency,
	})
	return true
}

// startBatchDownload 开始批量下载（并发版本）
func (h *BatchHandler) startBatchDownload(forceRedownload bool) {
	// 创建可取消的 context
	ctx, cancel := context.WithCancel(context.Background())
	h.mu.Lock()
	h.cancelFunc = cancel
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		h.running = false
		h.cancelFunc = nil
		h.mu.Unlock()
		cancel() // 确保释放资源
	}()

	// 获取下载目录
	downloadsDir, err := h.getDownloadsDir()
	if err != nil {
		utils.HandleError(err, "获取下载目录")
		return
	}

	// 获取并发数
	concurrency := 5 // 默认值（与配置默认值一致）
	if h.getConfig() != nil && h.getConfig().DownloadConcurrency > 0 {
		concurrency = h.getConfig().DownloadConcurrency
	}
	if concurrency < 1 {
		concurrency = 1
	}

	// 创建任务通道
	taskChan := make(chan int, len(h.tasks))
	var wg sync.WaitGroup

	// 启动 worker
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for taskIdx := range taskChan {
				// 检查是否取消
				select {
				case <-ctx.Done():
					return
				default:
				}

				h.mu.Lock()
				task := &h.tasks[taskIdx]
				task.Status = "downloading"
				h.mu.Unlock()

				utils.Info("📥 [Worker %d] 开始下载: %s", workerID, task.Title)

				// 下载视频
				err := h.downloadVideo(ctx, task, downloadsDir, forceRedownload, taskIdx)

				h.mu.Lock()
				if err != nil {
					task.Status = "failed"
					task.Error = err.Error()
					task.Progress = 0
					utils.Error("❌ [Worker %d] 失败: %s - %v", workerID, task.Title, err)
				} else {
					task.Status = "done"
					task.Progress = 100
					utils.Info("✅ [Worker %d] 完成: %s", workerID, task.Title)
				}
				h.mu.Unlock()
			}
		}(w)
	}

	// 分发任务（只处理 pending 状态的任务，跳过 done 和 failed）
	pendingCount := 0
	for i := range h.tasks {
		h.mu.RLock()
		taskStatus := h.tasks[i].Status
		h.mu.RUnlock()

		// 只处理 pending 状态的任务
		if taskStatus != "pending" {
			continue
		}

		select {
		case <-ctx.Done():
			close(taskChan)
			wg.Wait()
			utils.Info("⏹️ [批量下载] 已取消")
			return
		case taskChan <- i:
			pendingCount++
		}
	}
	close(taskChan)

	if pendingCount == 0 {
		utils.Info("ℹ️ [批量下载] 没有待处理的任务（所有任务已完成或失败）")
		return
	}
	utils.Info("📋 [批量下载] 开始处理 %d 个待处理任务", pendingCount)

	// 等待所有 worker 完成
	wg.Wait()

	// 统计结果
	h.mu.RLock()
	done, failed := 0, 0
	for _, t := range h.tasks {
		if t.Status == "done" {
			done++
		} else if t.Status == "failed" {
			failed++
		}
	}
	h.mu.RUnlock()

	utils.Info("✅ [批量下载] 全部完成！成功: %d, 失败: %d", done, failed)
}

// downloadVideo 下载单个视频（带重试和断点续传）
func (h *BatchHandler) downloadVideo(ctx context.Context, task *BatchTask, downloadsDir string, forceRedownload bool, taskIdx int) error {
	// 创建作者目录
	authorFolder := utils.CleanFolderName(task.GetAuthor())
	savePath := filepath.Join(downloadsDir, authorFolder)
	if err := utils.EnsureDir(savePath); err != nil {
		return fmt.Errorf("创建作者目录失败: %v", err)
	}

	// 优先使用视频ID进行去重检查（如果提供了视频ID）
	if !forceRedownload && task.ID != "" && h.downloadService != nil {
		if exists, err := h.downloadService.GetByID(task.ID); err == nil && exists != nil {
			// DB记录中已存在该视频ID，说明已下载过，尝试查找文件
			// 使用包含ID的文件名查找
			filenameWithID := utils.GenerateVideoFilename(task.Title, task.ID)
			filenameWithID = utils.EnsureExtension(filenameWithID, ".mp4")
			filePathWithID := filepath.Join(savePath, filenameWithID)
			if _, err := os.Stat(filePathWithID); err == nil {
				utils.Info("⏭️ [批量下载] 视频ID已存在记录中，文件已存在，跳过: ID=%s, 文件名=%s", task.ID, filenameWithID)
				// 文件已存在也保存记录（标记为已完成）
				h.saveDownloadRecord(task, filePathWithID, "completed")
				return nil
			}
		}
	} else if h.downloadService == nil {
		utils.Warn("downloadService is nil, skipping DB check")
	}

	// 生成文件名：优先使用视频ID确保唯一性
	cleanFilename := utils.GenerateVideoFilename(task.Title, task.ID)
	cleanFilename = utils.EnsureExtension(cleanFilename, ".mp4")
	filePath := filepath.Join(savePath, cleanFilename)

	// 检查文件是否已存在（作为备用检查，主要检查已通过ID完成）
	if !forceRedownload {
		if _, err := os.Stat(filePath); err == nil {
			utils.Info("⏭️ [批量下载] 文件已存在，跳过: %s", cleanFilename)
			// 文件已存在也保存记录（标记为已完成）
			h.saveDownloadRecord(task, filePath, "completed")
			return nil
		}
	}

	// 使用配置的重试次数
	maxRetries := 3
	if h.getConfig() != nil {
		maxRetries = h.getConfig().DownloadRetryCount
	}
	if maxRetries < 1 {
		maxRetries = 3
	}
	var lastErr error

	for retry := 0; retry < maxRetries; retry++ {
		// 检查是否取消
		select {
		case <-ctx.Done():
			return fmt.Errorf("下载已取消")
		default:
		}

		if retry > 0 {
			// 指数退避 + 随机抖动
			baseDelay := time.Duration(1<<uint(retry)) * time.Second // 2s, 4s, 8s...
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
			delay := baseDelay + jitter
			utils.Info("🔄 [批量下载] 等待 %v 后重试 (%d/%d): %s", delay, retry, maxRetries-1, task.Title)

			select {
			case <-ctx.Done():
				return fmt.Errorf("下载已取消")
			case <-time.After(delay):
			}
		}

		// 使用配置的超时时间
		timeout := 10 * time.Minute
		if h.getConfig() != nil && h.getConfig().DownloadTimeout > 0 {
			timeout = h.getConfig().DownloadTimeout
		}
		downloadCtx, cancel := context.WithTimeout(ctx, timeout)
		err := h.downloadVideoOnce(downloadCtx, task, filePath, taskIdx)
		cancel()

		if err == nil {
			// 下载成功，保存到下载记录数据库
			h.saveDownloadRecord(task, filePath, "completed")
			return nil
		}

		lastErr = err
		utils.LogDownloadRetry(task.ID, task.Title, retry+1, maxRetries, err)
		utils.Warn("⚠️ [批量下载] 下载失败 (尝试 %d/%d): %v", retry+1, maxRetries, err)

		// 如果不支持断点续传或是加密视频，清理临时文件
		resumeEnabled := h.getConfig() != nil && h.getConfig().DownloadResumeEnabled
		if task.DecryptorPrefix != "" || !resumeEnabled {
			os.Remove(filePath + ".tmp")
		}
	}

	// 记录最终失败的详细错误
	utils.LogDownloadError(task.ID, task.Title, task.GetAuthor(), task.URL, lastErr, maxRetries)
	return fmt.Errorf("下载失败（已重试 %d 次）: %v", maxRetries, lastErr)
}

// downloadVideoOnce 执行一次下载尝试（支持断点续传）
func (h *BatchHandler) downloadVideoOnce(ctx context.Context, task *BatchTask, filePath string, taskIdx int) error {
	// 使用 Gopeed 下载
	if h.gopeedService == nil {
		return fmt.Errorf("Gopeed下载服务未初始化")
	}

	// 开始下载
	utils.Info("🚀 [批量下载] 使用 Gopeed 下载: %s", task.Title)

	// 创建临时文件路径（Gopeed 会处理，这里我们只需要传递最终路径，
	// 但 GopeedService.DownloadSync 还没有实现自动重命名？
	// 让我们看看 GopeedService.DownloadSync 的实现。
	// 它是直接调用 CreateDirect，并没有阻塞直到完成？
	// 之前的 gopeed_service.go 实现是轮询状态直到 DownloadStatusDone。
	// 所以是阻塞的。

	// 注意：Gopeed 下载的临时文件名处理可能需要注意。
	// 如果我们传递 filePath，Gopeed 会直接下载到那个路径（或所在目录）。

	onProgress := func(progress float64, downloaded int64, total int64) {
		h.mu.Lock()
		defer h.mu.Unlock()

		// 确保任务索引有效
		if taskIdx >= 0 && taskIdx < len(h.tasks) {
			task := &h.tasks[taskIdx]

			// 只在下载中状态更新，避免覆盖完成状态
			if task.Status == "downloading" {
				task.Progress = progress * 100 // 转换为百分比
				task.DownloadedMB = float64(downloaded) / (1024 * 1024)
				task.TotalMB = float64(total) / (1024 * 1024)
				// 也可以根据需要计算 SizeMB 字符串
				if total > 0 {
					task.SizeMB = fmt.Sprintf("%.2fMB", task.TotalMB)
				}
				
				// 每10%输出一次日志
				if int(task.Progress)%10 == 0 && task.Progress > 0 {
					utils.Info("📊 [批量下载] %s 进度: %.1f%% (%.2f/%.2f MB)", 
						task.Title, task.Progress, task.DownloadedMB, task.TotalMB)
				}
			}
		}
	}

	// 获取单文件连接数配置
	connections := 8 // 默认值
	if h.getConfig() != nil && h.getConfig().DownloadConnections > 0 {
		connections = h.getConfig().DownloadConnections
	}

	err := h.gopeedService.DownloadSync(ctx, task.URL, filePath, connections, onProgress)
	if err != nil {
		return err
	}

	// 解密逻辑（如果需要）
	needDecrypt := task.Key != "" || (task.DecryptorPrefix != "" && task.PrefixLen > 0)
	if needDecrypt {
		utils.Info("🔐 [批量下载] 开始解密视频...")
		// 原地解密（不需要额外的临时文件，因为 gopeed 已经下载了完整文件）
		if err := utils.DecryptFileInPlace(filePath, task.GetKey(), task.DecryptorPrefix, task.PrefixLen); err != nil {
			return fmt.Errorf("解密失败: %v", err)
		}
		utils.Info("✓ [批量下载] 解密完成")
	}

	return nil
}

// saveDownloadRecord 保存下载记录到数据库
func (h *BatchHandler) saveDownloadRecord(task *BatchTask, filePath string, status string) {
	// 检查DB中是否已存在记录
	if h.downloadService != nil {
		if existing, err := h.downloadService.GetByID(task.ID); err == nil && existing != nil {
			utils.Info("📝 [下载记录] 记录已存在(DB)，跳过保存: %s - %s", task.Title, task.GetAuthor())
			return
		}
	}

	// 获取文件大小
	var fileSize int64 = 0
	if stat, err := os.Stat(filePath); err == nil {
		fileSize = stat.Size()
	}

	// 解析时长字符串为毫秒 (格式: "00:22" 或 "1:23:45")
	duration := parseDurationToMs(task.Duration)

	// 尝试从浏览记录获取更多信息（分辨率、封面等）
	resolution := task.Resolution
	coverURL := task.Cover
	if resolution == "" || coverURL == "" {
		browseRepo := database.NewBrowseHistoryRepository()
		if browseRecord, err := browseRepo.GetByID(task.ID); err == nil && browseRecord != nil {
			if resolution == "" {
				resolution = browseRecord.Resolution
			}
			if coverURL == "" {
				coverURL = browseRecord.CoverURL
			}
			// 如果时长为0，也从浏览记录获取
			if duration == 0 {
				duration = browseRecord.Duration
			}
		}
	}

	// 创建下载记录
	// 使用格式化后的文件名作为标题，确保与实际文件名一致
	cleanTitle := utils.CleanFilename(task.Title)
	record := &database.DownloadRecord{
		ID:           task.ID,
		VideoID:      task.ID,
		Title:        cleanTitle,
		Author:       task.GetAuthor(),
		CoverURL:     coverURL,
		Duration:     duration,
		FileSize:     fileSize,
		FilePath:     filePath,
		Format:       "mp4",
		Resolution:   resolution,
		Status:       status,
		DownloadTime: time.Now(),
	}

	// 保存到数据库
	if h.downloadService != nil {
		if err := h.downloadService.Create(record); err != nil {
			// 如果是重复记录，尝试更新
			if strings.Contains(err.Error(), "UNIQUE constraint") {
				if updateErr := h.downloadService.Update(record); updateErr != nil {
					utils.Warn("更新下载记录失败: %v", updateErr)
				}
			} else {
				utils.Warn("保存下载记录失败: %v", err)
			}
		} else {
			utils.Info("📝 [下载记录] 已保存(DB): %s - %s", task.Title, task.GetAuthor())
		}
	}
}

// parseDurationToMs 解析时长字符串为毫秒
// 支持格式: "00:22", "1:23", "1:23:45"
func parseDurationToMs(duration string) int64 {
	if duration == "" {
		return 0
	}

	parts := strings.Split(duration, ":")
	var totalSeconds int64 = 0

	switch len(parts) {
	case 2: // MM:SS
		minutes, _ := strconv.ParseInt(parts[0], 10, 64)
		seconds, _ := strconv.ParseInt(parts[1], 10, 64)
		totalSeconds = minutes*60 + seconds
	case 3: // HH:MM:SS
		hours, _ := strconv.ParseInt(parts[0], 10, 64)
		minutes, _ := strconv.ParseInt(parts[1], 10, 64)
		seconds, _ := strconv.ParseInt(parts[2], 10, 64)
		totalSeconds = hours*3600 + minutes*60 + seconds
	}

	return totalSeconds * 1000 // 转换为毫秒
}

// HandleBatchProgress 处理批量下载进度查询请求
func (h *BatchHandler) HandleBatchProgress(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/batch_progress" {
		return false
	}

	// 处理 CORS 预检请求
	if Conn.Request.Method == "OPTIONS" {
		h.sendSuccessResponse(Conn, map[string]interface{}{"message": "OK"})
		return true
	}

	// 授权校验
	if h.getConfig() != nil && h.getConfig().SecretToken != "" {
		if Conn.Request.Header.Get("X-Local-Auth") != h.getConfig().SecretToken {
			h.sendErrorResponse(Conn, fmt.Errorf("unauthorized"))
			return true
		}
	}

	h.mu.RLock()
	total := len(h.tasks)
	done, failed, running := 0, 0, 0
	var downloadingTasks []map[string]interface{}
	var allTasks []map[string]interface{}
	isRunning := h.running // 检查是否正在运行

	for _, t := range h.tasks {
		taskInfo := map[string]interface{}{
			"id":           t.ID,
			"title":        t.Title,
			"authorName":   t.GetAuthor(),
			"status":       t.Status,
			"progress":     t.Progress,
			"downloadedMB": t.DownloadedMB,
			"totalMB":      t.TotalMB,
			"error":        t.Error,
		}
		allTasks = append(allTasks, taskInfo)

		switch t.Status {
		case "done":
			done++
		case "failed":
			failed++
		case "downloading":
			// 只有在真正运行中时才统计为 running
			if isRunning {
				running++
				downloadingTasks = append(downloadingTasks, taskInfo)
			}
		}
	}
	h.mu.RUnlock()

	response := map[string]interface{}{
		"total":   total,
		"done":    done,
		"failed":  failed,
		"running": running,
		"tasks":   allTasks,
	}

	// 返回所有正在下载的任务（并发模式下可能有多个）
	if len(downloadingTasks) > 0 {
		response["currentTasks"] = downloadingTasks
		// 兼容旧版本，返回第一个
		response["currentTask"] = downloadingTasks[0]
	}

	h.sendSuccessResponse(Conn, response)
	return true
}

// HandleBatchCancel 处理批量下载取消请求
func (h *BatchHandler) HandleBatchCancel(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/batch_cancel" {
		return false
	}

	// 处理 CORS 预检请求
	if Conn.Request.Method == "OPTIONS" {
		h.sendSuccessResponse(Conn, map[string]interface{}{"message": "OK"})
		return true
	}

	// 授权校验
	if h.getConfig() != nil && h.getConfig().SecretToken != "" {
		if Conn.Request.Header.Get("X-Local-Auth") != h.getConfig().SecretToken {
			h.sendErrorResponse(Conn, fmt.Errorf("unauthorized"))
			return true
		}
	}

	h.mu.Lock()
	if h.running && h.cancelFunc != nil {
		h.cancelFunc() // 立即取消所有正在进行的下载
		h.running = false

		// 将正在下载的任务状态更新为 pending（表示已取消，但保留在列表中）
		// 这样前端可以通过 running=0 判断下载已取消
		// 注意：保留进度以支持断点续传
		for i := range h.tasks {
			if h.tasks[i].Status == "downloading" {
				h.tasks[i].Status = "pending"
				// 不重置进度，保留已下载的进度以支持断点续传
			}
		}
	}
	h.mu.Unlock()

	utils.Info("⏹️ [批量下载] 用户取消下载")

	h.sendSuccessResponse(Conn, map[string]interface{}{
		"message": "下载已取消",
	})
	return true
}

// HandleBatchFailed 处理导出失败清单请求
func (h *BatchHandler) HandleBatchFailed(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/batch_failed" {
		return false
	}

	// 处理 CORS 预检请求
	if Conn.Request.Method == "OPTIONS" {
		h.sendSuccessResponse(Conn, map[string]interface{}{"message": "OK"})
		return true
	}

	// 授权校验
	if h.getConfig() != nil && h.getConfig().SecretToken != "" {
		if Conn.Request.Header.Get("X-Local-Auth") != h.getConfig().SecretToken {
			h.sendErrorResponse(Conn, fmt.Errorf("unauthorized"))
			return true
		}
	}

	h.mu.RLock()
	failedTasks := make([]BatchTask, 0)
	for _, t := range h.tasks {
		if t.Status == "failed" {
			failedTasks = append(failedTasks, t)
		}
	}
	h.mu.RUnlock()

	if len(failedTasks) == 0 {
		h.sendSuccessResponse(Conn, map[string]interface{}{
			"failed": 0,
		})
		return true
	}

	// 导出失败清单
	// 获取下载目录
	downloadsDir, err := h.getDownloadsDir()
	if err != nil {
		h.sendErrorResponse(Conn, err)
		return true
	}
	timestamp := time.Now().Format("20060102_150405")
	exportFile := filepath.Join(downloadsDir, fmt.Sprintf("failed_videos_%s.json", timestamp))

	data, err := json.MarshalIndent(failedTasks, "", "  ")
	if err != nil {
		h.sendErrorResponse(Conn, err)
		return true
	}

	if err := os.WriteFile(exportFile, data, 0644); err != nil {
		h.sendErrorResponse(Conn, err)
		return true
	}

	utils.Info("📄 [批量下载] 失败清单已导出: %s", exportFile)

	h.sendSuccessResponse(Conn, map[string]interface{}{
		"failed": len(failedTasks),
		"json":   exportFile,
	})
	return true
}

// HandleBatchResume 处理继续下载请求（从pending状态恢复）
func (h *BatchHandler) HandleBatchResume(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/batch_resume" {
		return false
	}

	// 处理 CORS 预检请求
	if Conn.Request.Method == "OPTIONS" {
		h.sendSuccessResponse(Conn, map[string]interface{}{"message": "OK"})
		return true
	}

	// 授权校验
	if h.getConfig() != nil && h.getConfig().SecretToken != "" {
		if Conn.Request.Header.Get("X-Local-Auth") != h.getConfig().SecretToken {
			h.sendErrorResponse(Conn, fmt.Errorf("unauthorized"))
			return true
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// 检查是否有待处理的任务
	// 包括 pending 状态的任务，以及 failed 状态但错误为"下载已取消"的任务
	pendingCount := 0
	for i := range h.tasks {
		if h.tasks[i].Status == "pending" {
			pendingCount++
		} else if h.tasks[i].Status == "failed" && h.tasks[i].Error == "下载已取消" {
			// 将因取消而失败的任务重置为 pending 状态，以便继续下载
			// 注意：保留进度以支持断点续传
			h.tasks[i].Status = "pending"
			h.tasks[i].Error = ""
			// 不重置进度，保留已下载的进度以支持断点续传
			pendingCount++
		}
	}

	if pendingCount == 0 {
		h.sendErrorResponse(Conn, fmt.Errorf("没有待处理的任务"))
		return true
	}

	// 如果已经在运行，返回错误
	if h.running {
		h.sendErrorResponse(Conn, fmt.Errorf("下载正在进行中，无法继续"))
		return true
	}

	// 读取请求体获取 forceRedownload 参数
	var req struct {
		ForceRedownload bool `json:"forceRedownload"`
	}
	if Conn.Request.Body != nil {
		body, _ := io.ReadAll(Conn.Request.Body)
		json.Unmarshal(body, &req)
		Conn.Request.Body.Close()
	}

	// 启动下载
	h.running = true
	forceRedownload := req.ForceRedownload

	utils.Info("▶️ [批量下载] 继续下载 %d 个待处理任务", pendingCount)

	// 启动后台下载
	go h.startBatchDownload(forceRedownload)

	h.sendSuccessResponse(Conn, map[string]interface{}{
		"message": "继续下载已启动",
		"pending": pendingCount,
	})
	return true
}

// HandleBatchClear 处理清除任务请求
func (h *BatchHandler) HandleBatchClear(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/batch_clear" {
		return false
	}

	// 处理 CORS 预检请求
	if Conn.Request.Method == "OPTIONS" {
		h.sendSuccessResponse(Conn, map[string]interface{}{"message": "OK"})
		return true
	}

	// 授权校验
	if h.getConfig() != nil && h.getConfig().SecretToken != "" {
		if Conn.Request.Header.Get("X-Local-Auth") != h.getConfig().SecretToken {
			h.sendErrorResponse(Conn, fmt.Errorf("unauthorized"))
			return true
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// 如果正在运行，先取消
	if h.running && h.cancelFunc != nil {
		h.cancelFunc()
		h.running = false
	}

	// 清除所有任务
	taskCount := len(h.tasks)
	h.tasks = nil
	h.cancelFunc = nil

	utils.Info("🗑️ [批量下载] 已清除所有任务（%d 个）", taskCount)

	h.sendSuccessResponse(Conn, map[string]interface{}{
		"message": "任务已清除",
		"cleared": taskCount,
	})
	return true
}

// sendSuccessResponse 发送成功响应
func (h *BatchHandler) sendSuccessResponse(Conn *SunnyNet.HttpConn, data interface{}) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	headers.Set("Pragma", "no-cache")
	headers.Set("Expires", "0")
	headers.Set("X-Content-Type-Options", "nosniff")
	if h.getConfig() != nil && len(h.getConfig().AllowedOrigins) > 0 {
		origin := Conn.Request.Header.Get("Origin")
		if origin != "" {
			for _, o := range h.getConfig().AllowedOrigins {
				if o == origin {
					headers.Set("Access-Control-Allow-Origin", origin)
					headers.Set("Vary", "Origin")
					headers.Set("Access-Control-Allow-Headers", "Content-Type, X-Local-Auth")
					headers.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
					break
				}
			}
		}
	}

	Conn.StopRequest(200, string(response.SuccessJSON(data)), headers)
}

// sendErrorResponse 发送错误响应
func (h *BatchHandler) sendErrorResponse(Conn *SunnyNet.HttpConn, err error) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Content-Type-Options", "nosniff")
	if h.getConfig() != nil && len(h.getConfig().AllowedOrigins) > 0 {
		origin := Conn.Request.Header.Get("Origin")
		if origin != "" {
			for _, o := range h.getConfig().AllowedOrigins {
				if o == origin {
					headers.Set("Access-Control-Allow-Origin", origin)
					headers.Set("Vary", "Origin")
					headers.Set("Access-Control-Allow-Headers", "Content-Type, X-Local-Auth")
					headers.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
					break
				}
			}
		}
	}
	Conn.StopRequest(500, string(response.ErrorJSON(500, err.Error())), headers)
}
