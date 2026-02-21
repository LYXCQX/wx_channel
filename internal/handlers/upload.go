package handlers

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
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
	"wx_channel/internal/utils" // Import websocket package
	"wx_channel/internal/websocket"
	"wx_channel/pkg/util"

	"github.com/fatih/color"
	"github.com/qtgolang/SunnyNet/SunnyNet"
)

// UploadHandler 文件上传处理器
type UploadHandler struct {
	downloadService *services.DownloadRecordService
	gopeedService   *services.GopeedService // Injected Gopeed Service
	chunkSem        chan struct{}
	mergeSem        chan struct{}
	wsHub           *websocket.Hub
	activeDownloads sync.Map // map[string]context.CancelFunc
}

// NewUploadHandler 创建上传处理器
func NewUploadHandler(cfg *config.Config, wsHub *websocket.Hub, gopeedService *services.GopeedService) *UploadHandler {
	ch := cfg.UploadChunkConcurrency
	if ch <= 0 {
		ch = 4
	}
	mg := cfg.UploadMergeConcurrency
	if mg <= 0 {
		mg = 1
	}
	return &UploadHandler{
		downloadService: services.NewDownloadRecordService(),
		gopeedService:   gopeedService,
		chunkSem:        make(chan struct{}, ch),
		mergeSem:        make(chan struct{}, mg),
		wsHub:           wsHub,
	}
}

// getConfig 获取当前配置（动态获取最新配置）
func (h *UploadHandler) getConfig() *config.Config {
	return config.Get()
}

// getDownloadsDir 获取解析后的下载目录
func (h *UploadHandler) getDownloadsDir() (string, error) {
	cfg := h.getConfig()
	if cfg == nil {
		return "", fmt.Errorf("config is nil")
	}
	return cfg.GetResolvedDownloadsDir()
}

// Handle implements router.Interceptor
func (h *UploadHandler) Handle(Conn *SunnyNet.HttpConn) bool {
	// Critical nil check
	if Conn == nil || Conn.Request == nil || Conn.Request.URL == nil {
		return false
	}

	if h.HandleInitUpload(Conn) {
		return true
	}
	if h.HandleUploadChunk(Conn) {
		return true
	}
	if h.HandleCompleteUpload(Conn) {
		return true
	}
	if h.HandleUploadStatus(Conn) {
		return true
	}
	if h.HandleSaveVideo(Conn) {
		return true
	}
	if h.HandleSaveCover(Conn) {
		return true
	}
	if h.HandleCancelDownload(Conn) {
		return true
	}
	if h.HandleDownloadVideo(Conn) {
		return true
	}
	return false
}

// HandleCancelDownload 处理取消下载请求
func (h *UploadHandler) HandleCancelDownload(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/cancel_download" {
		return false
	}

	// 允许 POST 或 GET 请求
	if Conn.Request.Method != "POST" && Conn.Request.Method != "GET" {
		h.sendErrorResponse(Conn, fmt.Errorf("method not allowed: %s", Conn.Request.Method))
		return true
	}

	var videoId string

	if Conn.Request.Method == "GET" {
		videoId = Conn.Request.URL.Query().Get("videoId")
	} else {
		// POST 请求解析Body
		body, err := io.ReadAll(Conn.Request.Body)
		if err == nil {
			var req struct {
				VideoID string `json:"videoId"`
			}
			json.Unmarshal(body, &req)
			videoId = req.VideoID
		}
		_ = Conn.Request.Body.Close()
	}

	if videoId == "" {
		h.sendErrorResponse(Conn, fmt.Errorf("missing videoId"))
		return true
	}

	utils.Info("⏹️ [取消下载] 收到取消请求: %s", videoId)

	// 查找并调用取消函数
	if cancel, ok := h.activeDownloads.Load(videoId); ok {
		if cancelFunc, ok := cancel.(context.CancelFunc); ok {
			cancelFunc()
			utils.Info("Found and executed cancel function for %s", videoId)
		}
		h.activeDownloads.Delete(videoId)
		h.sendSuccessResponse(Conn)
	} else {
		utils.Warn("No active download found for %s to cancel", videoId)
		// 即使没找到也返回成功，可能是已经完成了
		h.sendSuccessResponse(Conn)
	}

	return true
}

// HandleInitUpload 处理分片上传初始化请求
func (h *UploadHandler) HandleInitUpload(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/init_upload" {
		return false
	}

	if h.getConfig() != nil && h.getConfig().SecretToken != "" {
		if Conn.Request.Header.Get("X-Local-Auth") != h.getConfig().SecretToken {
			headers := http.Header{}
			headers.Set("Content-Type", "application/json")
			headers.Set("X-Content-Type-Options", "nosniff")
			Conn.StopRequest(401, `{"success":false,"error":"unauthorized"}`, headers)
			return true
		}
	}
	if h.getConfig() != nil && len(h.getConfig().AllowedOrigins) > 0 {
		origin := Conn.Request.Header.Get("Origin")
		if origin != "" {
			allowed := false
			for _, o := range h.getConfig().AllowedOrigins {
				if o == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				headers.Set("X-Content-Type-Options", "nosniff")
				Conn.StopRequest(403, `{"success":false,"error":"forbidden_origin"}`, headers)
				return true
			}
		}
	}

	// 获取下载目录
	downloadsDir, err := h.getDownloadsDir()
	if err != nil {
		utils.HandleError(err, "获取下载目录")
		h.sendErrorResponse(Conn, err)
		return true
	}

	uploadsRoot := filepath.Join(downloadsDir, ".uploads")
	if err := utils.EnsureDir(uploadsRoot); err != nil {
		utils.HandleError(err, "创建上传目录")
	}

	// 生成 uploadId
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		utils.HandleError(err, "生成随机数")
		h.sendErrorResponse(Conn, err)
		return true
	}
	uploadId := fmt.Sprintf("%x", b)
	utils.Info("🔄 init_upload: 生成 uploadId = %s", uploadId)

	// 创建临时目录
	upDir := filepath.Join(uploadsRoot, uploadId)
	if err := os.MkdirAll(upDir, 0755); err != nil {
		utils.HandleError(err, "创建上传目录")
		utils.LogUploadInit(uploadId, false)
		h.sendErrorResponse(Conn, err)
		return true
	}

	// 记录上传初始化成功
	utils.LogUploadInit(uploadId, true)

	// 使用 JSON 编码确保正确转义
	responseData := map[string]interface{}{
		"success":  true,
		"uploadId": uploadId,
	}
	utils.Info("✅ init_upload: 返回响应: %v", responseData)
	h.sendJSONResponse(Conn, 200, responseData)
	return true
}

// HandleUploadChunk 处理分片上传请求
func (h *UploadHandler) HandleUploadChunk(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/upload_chunk" {
		return false
	}

	// 并发限流（分片）
	if h.chunkSem != nil {
		h.chunkSem <- struct{}{}
		defer func() { <-h.chunkSem }()
	}

	if h.getConfig() != nil && h.getConfig().SecretToken != "" {
		if Conn.Request.Header.Get("X-Local-Auth") != h.getConfig().SecretToken {
			headers := http.Header{}
			headers.Set("Content-Type", "application/json")
			headers.Set("X-Content-Type-Options", "nosniff")
			Conn.StopRequest(401, `{"success":false,"error":"unauthorized"}`, headers)
			return true
		}
	}
	if h.getConfig() != nil && len(h.getConfig().AllowedOrigins) > 0 {
		origin := Conn.Request.Header.Get("Origin")
		if origin != "" {
			allowed := false
			for _, o := range h.getConfig().AllowedOrigins {
				if o == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				headers.Set("X-Content-Type-Options", "nosniff")
				Conn.StopRequest(403, `{"success":false,"error":"forbidden_origin"}`, headers)
				return true
			}
		}
	}

	// 解析multipart表单
	err := Conn.Request.ParseMultipartForm(h.getConfig().MaxUploadSize)
	if err != nil {
		utils.HandleError(err, "解析multipart表单")
		h.sendErrorResponse(Conn, err)
		return true
	}

	uploadId := Conn.Request.FormValue("uploadId")
	indexStr := Conn.Request.FormValue("index")
	totalStr := Conn.Request.FormValue("total")

	if uploadId == "" || indexStr == "" || totalStr == "" {
		h.sendErrorResponse(Conn, fmt.Errorf("missing fields"))
		return true
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		utils.HandleError(err, "解析索引")
		h.sendErrorResponse(Conn, err)
		return true
	}

	total, err := strconv.Atoi(totalStr)
	if err != nil {
		utils.HandleError(err, "解析总数")
		h.sendErrorResponse(Conn, err)
		return true
	}

	utils.Info("[分片上传] 接收分片: uploadId=%s, 分片索引=%d/%d", uploadId, index+1, total)

	file, _, err := Conn.Request.FormFile("chunk")
	if err != nil {
		utils.HandleError(err, "获取分片文件")
		h.sendErrorResponse(Conn, err)
		return true
	}
	defer file.Close()

	checksum := Conn.Request.FormValue("checksum")
	algo := strings.ToLower(Conn.Request.FormValue("algo"))
	if algo == "" {
		algo = "md5"
	}
	var expectedSize int64 = -1
	if sz := Conn.Request.FormValue("size"); sz != "" {
		if v, convErr := strconv.ParseInt(sz, 10, 64); convErr == nil {
			expectedSize = v
		}
	}

	downloadsDir, err := h.getDownloadsDir()
	if err != nil {
		utils.HandleError(err, "获取下载目录")
		h.sendErrorResponse(Conn, err)
		return true
	}

	uploadsRoot := filepath.Join(downloadsDir, ".uploads")
	upDir := filepath.Join(uploadsRoot, uploadId)

	if _, err := os.Stat(upDir); os.IsNotExist(err) {
		h.sendErrorResponse(Conn, fmt.Errorf("uploadId not found"))
		return true
	}

	partPath := filepath.Join(upDir, fmt.Sprintf("%06d.part", index))
	out, err := os.Create(partPath)
	if err != nil {
		utils.HandleError(err, "创建分片文件")
		h.sendErrorResponse(Conn, err)
		return true
	}
	defer out.Close()

	var written int64
	if checksum != "" {
		switch algo {
		case "md5":
			hsh := md5.New()
			n, err := io.Copy(io.MultiWriter(out, hsh), file)
			if err != nil {
				utils.HandleError(err, "写入分片数据")
				h.sendErrorResponse(Conn, err)
				return true
			}
			sum := fmt.Sprintf("%x", hsh.Sum(nil))
			if !strings.EqualFold(sum, checksum) {
				_ = out.Close()
				_ = os.Remove(partPath)
				utils.Error("[分片上传] 校验失败: uploadId=%s, 分片索引=%d, 算法=%s, 期望=%s, 实际=%s", uploadId, index, algo, checksum, sum)
				h.sendErrorResponse(Conn, fmt.Errorf("checksum_mismatch"))
				return true
			}
			written = n
			utils.Info("[分片上传] 校验通过: uploadId=%s, 分片索引=%d, 算法=%s, 大小=%.2fMB", uploadId, index, algo, float64(written)/(1024*1024))
		case "sha256":
			hsh := sha256.New()
			n, err := io.Copy(io.MultiWriter(out, hsh), file)
			if err != nil {
				utils.HandleError(err, "写入分片数据")
				h.sendErrorResponse(Conn, err)
				return true
			}
			sum := fmt.Sprintf("%x", hsh.Sum(nil))
			if !strings.EqualFold(sum, checksum) {
				_ = out.Close()
				_ = os.Remove(partPath)
				utils.Error("[分片上传] 校验失败: uploadId=%s, 分片索引=%d, 算法=%s, 期望=%s, 实际=%s", uploadId, index, algo, checksum, sum)
				h.sendErrorResponse(Conn, fmt.Errorf("checksum_mismatch"))
				return true
			}
			written = n
			utils.Info("[分片上传] 校验通过: uploadId=%s, 分片索引=%d, 算法=%s, 大小=%.2fMB", uploadId, index, algo, float64(written)/(1024*1024))
		default:
			h.sendErrorResponse(Conn, fmt.Errorf("unsupported_algo"))
			return true
		}
	} else {
		n, err := io.Copy(out, file)
		if err != nil {
			utils.HandleError(err, "写入分片数据")
			h.sendErrorResponse(Conn, err)
			return true
		}
		written = n
	}

	// 尺寸校验（可选字段 + 上限保护）
	if expectedSize >= 0 && written != expectedSize {
		_ = out.Close()
		_ = os.Remove(partPath)
		utils.Error("[分片上传] 尺寸不匹配: uploadId=%s, 分片索引=%d, 期望=%d, 实际=%d", uploadId, index, expectedSize, written)
		h.sendErrorResponse(Conn, fmt.Errorf("size_mismatch"))
		return true
	}
	if h.getConfig() != nil && h.getConfig().ChunkSize > 0 && written > h.getConfig().ChunkSize*2 { // 容忍放宽至2倍
		_ = out.Close()
		_ = os.Remove(partPath)
		utils.Error("[分片上传] 分片过大: uploadId=%s, 分片索引=%d, 大小=%d, 限制=%d", uploadId, index, written, h.getConfig().ChunkSize*2)
		h.sendErrorResponse(Conn, fmt.Errorf("chunk_too_large"))
		return true
	}
	if err != nil {
		utils.HandleError(err, "写入分片数据")
		h.sendErrorResponse(Conn, err)
		return true
	}

	sizeMB := float64(written) / (1024 * 1024)
	utils.Info("[分片上传] 分片已保存: uploadId=%s, 分片索引=%d/%d, 大小=%.2fMB, 路径=%s", uploadId, index+1, total, sizeMB, partPath)

	// 记录分片上传成功
	utils.LogUploadChunk(uploadId, index, total, sizeMB, true)

	h.sendSuccessResponse(Conn)
	return true
}

// HandleCompleteUpload 处理分片上传完成请求
func (h *UploadHandler) HandleCompleteUpload(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/complete_upload" {
		return false
	}

	// 并发限流（合并）
	if h.mergeSem != nil {
		h.mergeSem <- struct{}{}
		defer func() { <-h.mergeSem }()
	}

	if h.getConfig() != nil && h.getConfig().SecretToken != "" {
		if Conn.Request.Header.Get("X-Local-Auth") != h.getConfig().SecretToken {
			headers := http.Header{}
			headers.Set("Content-Type", "application/json")
			headers.Set("X-Content-Type-Options", "nosniff")
			Conn.StopRequest(401, `{"success":false,"error":"unauthorized"}`, headers)
			return true
		}
	}
	if h.getConfig() != nil && len(h.getConfig().AllowedOrigins) > 0 {
		origin := Conn.Request.Header.Get("Origin")
		if origin != "" {
			allowed := false
			for _, o := range h.getConfig().AllowedOrigins {
				if o == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				headers.Set("X-Content-Type-Options", "nosniff")
				Conn.StopRequest(403, `{"success":false,"error":"forbidden_origin"}`, headers)
				return true
			}
		}
	}

	body, err := io.ReadAll(Conn.Request.Body)
	if err != nil {
		utils.HandleError(err, "读取complete_upload请求体")
		h.sendErrorResponse(Conn, err)
		return true
	}

	if err := Conn.Request.Body.Close(); err != nil {
		utils.HandleError(err, "关闭请求体")
	}

	var req struct {
		UploadId   string `json:"uploadId"`
		Total      int    `json:"total"`
		Filename   string `json:"filename"`
		AuthorName string `json:"authorName"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		utils.HandleError(err, "解析complete_upload JSON")
		h.sendErrorResponse(Conn, err)
		return true
	}

	if req.UploadId == "" || req.Total <= 0 || req.Filename == "" {
		utils.Error("[分片合并] 缺少必要字段: uploadId=%s, total=%d, filename=%s", req.UploadId, req.Total, req.Filename)
		h.sendErrorResponse(Conn, fmt.Errorf("missing fields"))
		return true
	}
	utils.Info("[分片合并] 开始合并: uploadId=%s, 文件名=%s, 作者=%s, 分片数=%d", req.UploadId, req.Filename, req.AuthorName, req.Total)

	downloadsDir, err := h.getDownloadsDir()
	if err != nil {
		utils.HandleError(err, "获取下载目录")
		h.sendErrorResponse(Conn, err)
		return true
	}

	uploadsRoot := filepath.Join(downloadsDir, ".uploads")
	upDir := filepath.Join(uploadsRoot, req.UploadId)

	// 目标作者目录
	authorFolder := utils.CleanFolderName(req.AuthorName)
	savePath := filepath.Join(downloadsDir, authorFolder)

	if err := utils.EnsureDir(savePath); err != nil {
		utils.HandleError(err, "创建作者目录")
		h.sendErrorResponse(Conn, err)
		return true
	}

	// 清理文件名
	cleanFilename := utils.CleanFilename(req.Filename)
	cleanFilename = utils.EnsureExtension(cleanFilename, ".mp4")

	// 冲突处理
	base := filepath.Base(cleanFilename)
	ext := filepath.Ext(cleanFilename)
	baseName := strings.TrimSuffix(base, ext)
	finalPath := filepath.Join(savePath, cleanFilename)
	if _, err := os.Stat(finalPath); err == nil {
		// 文件已存在，生成唯一文件名
		for i := 1; i < 1000; i++ {
			candidate := filepath.Join(savePath, fmt.Sprintf("%s(%d)%s", baseName, i, ext))
			if _, existsErr := os.Stat(candidate); os.IsNotExist(existsErr) {
				finalPath = candidate
				break
			}
		}
	}

	// 合并分片
	out, err := os.Create(finalPath)
	if err != nil {
		utils.HandleError(err, "创建目标文件")
		h.sendErrorResponse(Conn, err)
		return true
	}
	defer out.Close()

	// 基本存在性与数量校验
	for i := 0; i < req.Total; i++ {
		partPath := filepath.Join(upDir, fmt.Sprintf("%06d.part", i))
		if _, err := os.Stat(partPath); err != nil {
			utils.Error("[分片合并] 分片缺失: uploadId=%s, 分片索引=%d, 路径=%s", req.UploadId, i, partPath)
			h.sendErrorResponse(Conn, fmt.Errorf("missing_part_%06d", i))
			return true
		}
	}

	var totalWritten int64
	for i := 0; i < req.Total; i++ {
		partPath := filepath.Join(upDir, fmt.Sprintf("%06d.part", i))
		in, err := os.Open(partPath)
		if err != nil {
			utils.HandleError(err, "打开分片文件")
			h.sendErrorResponse(Conn, err)
			return true
		}

		n, err := io.Copy(out, in)
		in.Close()
		if err != nil {
			utils.HandleError(err, "合并分片数据")
			h.sendErrorResponse(Conn, err)
			return true
		}
		totalWritten += n
	}

	// 清理临时目录
	os.RemoveAll(upDir)

	fileSize := float64(totalWritten) / (1024 * 1024)
	utils.Info("[分片合并] 合并完成: uploadId=%s, 文件名=%s, 作者=%s, 路径=%s, 大小=%.2fMB, 分片数=%d", req.UploadId, req.Filename, req.AuthorName, finalPath, fileSize, req.Total)
	color.Green("✓ 分片视频已保存: %s (%.2f MB)", finalPath, fileSize)

	// 记录分片合并成功
	utils.LogUploadMerge(req.UploadId, req.Filename, req.AuthorName, req.Total, fileSize, true)

	responseData := map[string]interface{}{
		"success": true,
		"path":    finalPath,
		"size":    fileSize,
	}
	utils.Info("✅ complete_upload: 返回响应: %v", responseData)
	h.sendJSONResponse(Conn, 200, responseData)
	return true
}

// HandleSaveVideo 处理直接保存视频文件请求
func (h *UploadHandler) HandleSaveVideo(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/save_video" {
		return false
	}

	if h.getConfig() != nil && h.getConfig().SecretToken != "" {
		if Conn.Request.Header.Get("X-Local-Auth") != h.getConfig().SecretToken {
			headers := http.Header{}
			headers.Set("Content-Type", "application/json")
			headers.Set("X-Content-Type-Options", "nosniff")
			Conn.StopRequest(401, `{"success":false,"error":"unauthorized"}`, headers)
			return true
		}
	}
	if h.getConfig() != nil && len(h.getConfig().AllowedOrigins) > 0 {
		origin := Conn.Request.Header.Get("Origin")
		if origin != "" {
			allowed := false
			for _, o := range h.getConfig().AllowedOrigins {
				if o == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				headers.Set("X-Content-Type-Options", "nosniff")
				Conn.StopRequest(403, `{"success":false,"error":"forbidden_origin"}`, headers)
				return true
			}
		}
	}

	utils.Info("🔄 save_video: 开始处理请求")

	// 解析multipart表单
	err := Conn.Request.ParseMultipartForm(h.getConfig().MaxUploadSize)
	if err != nil {
		utils.HandleError(err, "解析表单数据")
		h.sendErrorResponse(Conn, err)
		return true
	}

	utils.Info("✅ save_video: 表单解析成功")

	file, header, err := Conn.Request.FormFile("video")
	if err != nil {
		utils.HandleError(err, "获取视频文件")
		h.sendErrorResponse(Conn, err)
		return true
	}
	defer file.Close()

	utils.Info("接收上传: %s, 报告大小: %d bytes", header.Filename, header.Size)

	filename := Conn.Request.FormValue("filename")
	authorName := Conn.Request.FormValue("authorName")
	isEncrypted := Conn.Request.FormValue("isEncrypted") == "true"

	// 创建作者文件夹路径
	authorFolder := utils.CleanFolderName(authorName)

	downloadsDir, err := h.getDownloadsDir()
	if err != nil {
		utils.HandleError(err, "获取下载目录")
		h.sendErrorResponse(Conn, err)
		return true
	}
	savePath := filepath.Join(downloadsDir, authorFolder)

	utils.Info("保存目录: %s", savePath)
	if err := utils.EnsureDir(savePath); err != nil {
		utils.HandleError(err, "创建文件夹")
		h.sendErrorResponse(Conn, err)
		return true
	}

	// 清理文件名
	cleanFilename := utils.CleanFilename(filename)
	cleanFilename = utils.EnsureExtension(cleanFilename, ".mp4")

	// 生成唯一文件名
	filePath := filepath.Join(savePath, cleanFilename)
	if _, statErr := os.Stat(filePath); statErr == nil {
		base := strings.TrimSuffix(cleanFilename, filepath.Ext(cleanFilename))
		ext := filepath.Ext(cleanFilename)
		for i := 1; i < 1000; i++ {
			candidate := filepath.Join(savePath, fmt.Sprintf("%s(%d)%s", base, i, ext))
			if _, existsErr := os.Stat(candidate); os.IsNotExist(existsErr) {
				filePath = candidate
				break
			}
		}
	}

	// 保存文件
	out, err := os.Create(filePath)
	if err != nil {
		utils.HandleError(err, "创建目标文件")
		h.sendErrorResponse(Conn, err)
		return true
	}
	defer out.Close()

	// 流式拷贝
	if seeker, ok := file.(io.Seeker); ok {
		_, _ = seeker.Seek(0, io.SeekStart)
	}

	written, err := io.Copy(out, file)
	if err != nil {
		utils.HandleError(err, "写入视频数据")
		h.sendErrorResponse(Conn, err)
		return true
	}

	fileSize := float64(written) / (1024 * 1024)
	statusMsg := ""
	if isEncrypted {
		statusMsg = " [已解密]"
	}
	utils.Info("✓ 视频已保存: %s (%.2f MB)%s", filePath, fileSize, statusMsg)

	// 记录直接上传成功
	utils.LogDirectUpload(filename, authorName, fileSize, isEncrypted, true)

	responseData := map[string]interface{}{
		"success": true,
		"path":    filePath,
		"size":    fileSize,
	}
	utils.Info("✅ save_video: 返回响应: %v", responseData)
	h.sendJSONResponse(Conn, 200, responseData)
	return true
}

// HandleSaveCover 处理保存封面图片请求
func (h *UploadHandler) HandleSaveCover(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/save_cover" {
		return false
	}

	// 授权校验
	if h.getConfig() != nil && h.getConfig().SecretToken != "" {
		if Conn.Request.Header.Get("X-Local-Auth") != h.getConfig().SecretToken {
			headers := http.Header{}
			headers.Set("Content-Type", "application/json")
			headers.Set("X-Content-Type-Options", "nosniff")
			Conn.StopRequest(401, `{"success":false,"error":"unauthorized"}`, headers)
			return true
		}
	}
	if h.getConfig() != nil && len(h.getConfig().AllowedOrigins) > 0 {
		origin := Conn.Request.Header.Get("Origin")
		if origin != "" {
			allowed := false
			for _, o := range h.getConfig().AllowedOrigins {
				if o == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				headers.Set("X-Content-Type-Options", "nosniff")
				Conn.StopRequest(403, `{"success":false,"error":"forbidden_origin"}`, headers)
				return true
			}
		}
	}

	// 只处理 POST 请求
	if Conn.Request.Method != "POST" {
		h.sendErrorResponse(Conn, fmt.Errorf("method not allowed: %s", Conn.Request.Method))
		return true
	}

	body, err := io.ReadAll(Conn.Request.Body)
	if err != nil {
		utils.HandleError(err, "读取save_cover请求体")
		h.sendErrorResponse(Conn, err)
		return true
	}
	defer Conn.Request.Body.Close()

	var req struct {
		CoverURL  string `json:"coverUrl"`
		VideoID   string `json:"videoId"`
		Title     string `json:"title"`
		Author    string `json:"author"`
		ForceSave bool   `json:"forceSave"` // 是否强制保存（即使文件已存在）
	}

	if err := json.Unmarshal(body, &req); err != nil {
		utils.HandleError(err, "解析save_cover JSON")
		h.sendErrorResponse(Conn, err)
		return true
	}

	if req.CoverURL == "" {
		h.sendErrorResponse(Conn, fmt.Errorf("封面URL不能为空"))
		return true
	}

	// 创建作者目录
	authorFolder := utils.CleanFolderName(req.Author)
	if authorFolder == "" {
		authorFolder = "未知作者"
	}

	downloadsDir, err := h.getDownloadsDir()
	if err != nil {
		utils.HandleError(err, "获取下载目录")
		h.sendErrorResponse(Conn, err)
		return true
	}
	savePath := filepath.Join(downloadsDir, authorFolder)

	if err := utils.EnsureDir(savePath); err != nil {
		utils.HandleError(err, "创建作者目录")
		h.sendErrorResponse(Conn, err)
		return true
	}

	// 生成文件名：使用视频标题，如果没有则使用视频ID
	var filename string
	if req.Title != "" {
		filename = utils.CleanFilename(req.Title)
	} else if req.VideoID != "" {
		filename = "cover_" + req.VideoID
	} else {
		filename = "cover_" + fmt.Sprintf("%d", time.Now().Unix())
	}

	// 确保文件扩展名
	filename = utils.EnsureExtension(filename, ".jpg")
	coverPath := filepath.Join(savePath, filename)

	// 检查文件是否已存在
	if !req.ForceSave {
		if _, err := os.Stat(coverPath); err == nil {
			utils.Info("⏭️ [封面下载] 文件已存在，跳过: %s", filename)
			relativePath, _ := filepath.Rel(downloadsDir, coverPath)
			responseData := map[string]interface{}{
				"success":      true,
				"path":         coverPath,
				"message":      "文件已存在",
				"relativePath": relativePath,
			}
			responseBytes, _ := json.Marshal(responseData)
			h.sendJSONResponse(Conn, 200, responseBytes)
			return true
		}
	}

	// 下载封面图片
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Get(req.CoverURL)
	if err != nil {
		utils.HandleError(err, "下载封面图片")
		h.sendErrorResponse(Conn, fmt.Errorf("下载封面失败: %v", err))
		return true
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.sendErrorResponse(Conn, fmt.Errorf("下载封面失败: HTTP %d", resp.StatusCode))
		return true
	}

	// 保存文件
	out, err := os.Create(coverPath)
	if err != nil {
		utils.HandleError(err, "创建封面文件")
		h.sendErrorResponse(Conn, err)
		return true
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		utils.HandleError(err, "写入封面数据")
		h.sendErrorResponse(Conn, err)
		return true
	}

	fileSize := float64(written) / 1024 // KB
	relativePath, _ := filepath.Rel(downloadsDir, coverPath)

	utils.Info("✓ [封面下载] 封面已保存: %s (%.2f KB)", relativePath, fileSize)

	responseData := map[string]interface{}{
		"success":      true,
		"path":         coverPath,
		"relativePath": relativePath,
		"size":         fileSize,
	}
	responseBytes, err := json.Marshal(responseData)
	if err != nil {
		utils.HandleError(err, "生成响应JSON")
		h.sendErrorResponse(Conn, err)
		return true
	}
	h.sendJSONResponse(Conn, 200, responseBytes)
	return true
}

// HandleDownloadVideo 处理从URL下载视频请求
func (h *UploadHandler) HandleDownloadVideo(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/download_video" {
		return false
	}

	// 授权校验
	if h.getConfig() != nil && h.getConfig().SecretToken != "" {
		if Conn.Request.Header.Get("X-Local-Auth") != h.getConfig().SecretToken {
			headers := http.Header{}
			headers.Set("Content-Type", "application/json")
			headers.Set("X-Content-Type-Options", "nosniff")
			Conn.StopRequest(401, `{"success":false,"error":"unauthorized"}`, headers)
			return true
		}
	}
	if h.getConfig() != nil && len(h.getConfig().AllowedOrigins) > 0 {
		origin := Conn.Request.Header.Get("Origin")
		if origin != "" {
			allowed := false
			for _, o := range h.getConfig().AllowedOrigins {
				if o == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				headers.Set("X-Content-Type-Options", "nosniff")
				Conn.StopRequest(403, `{"success":false,"error":"forbidden_origin"}`, headers)
				return true
			}
		}
	}

	// 只处理 POST 请求
	if Conn.Request.Method != "POST" {
		h.sendErrorResponse(Conn, fmt.Errorf("method not allowed: %s", Conn.Request.Method))
		return true
	}

	// check body
	if Conn.Request.Body == nil {
		utils.Error("Handler request body is nil")
		h.sendErrorResponse(Conn, fmt.Errorf("request body is nil"))
		return true
	}

	body, err := io.ReadAll(Conn.Request.Body)
	if err != nil {
		utils.HandleError(err, "读取download_video请求体")
		h.sendErrorResponse(Conn, err)
		return true
	}
	defer Conn.Request.Body.Close()

	var req struct {
		VideoURL     string `json:"videoUrl"`
		VideoID      string `json:"videoId"`
		Title        string `json:"title"`
		Author       string `json:"author"`
		Key          string `json:"key"`        // 解密key（可选）
		ForceSave    bool   `json:"forceSave"`  // 是否强制保存（即使文件已存在）
		Resolution   string `json:"resolution"` // 分辨率字符串（如 "1080x1920" 或 "1080p"）
		Width        int    `json:"width"`      // 视频宽度（可选）
		Height       int    `json:"height"`     // 视频高度（可选）
		FileFormat   string `json:"fileFormat"` // 文件格式（如 "hd", "sd" 等）
		LikeCount    int64  `json:"likeCount"`
		CommentCount int64  `json:"commentCount"`
		ForwardCount int64  `json:"forwardCount"`
		FavCount     int64  `json:"favCount"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		utils.HandleError(err, "解析download_video JSON")
		h.sendErrorResponse(Conn, err)
		return true
	}

	if req.VideoURL == "" {
		h.sendErrorResponse(Conn, fmt.Errorf("视频URL不能为空"))
		return true
	}

	// 创建作者目录
	authorFolder := utils.CleanFolderName(req.Author)
	if authorFolder == "" {
		authorFolder = "未知作者"
	}

	downloadsDir, err := h.getDownloadsDir()
	if err != nil {
		utils.HandleError(err, "获取下载目录")
		h.sendErrorResponse(Conn, err)
		return true
	}
	savePath := filepath.Join(downloadsDir, authorFolder)

	if err := utils.EnsureDir(savePath); err != nil {
		utils.HandleError(err, "创建作者目录")
		h.sendErrorResponse(Conn, err)
		return true
	}

	// 优先使用视频ID进行去重检查（如果提供了视频ID）
	if !req.ForceSave && req.VideoID != "" && h.downloadService != nil {
		if exists, err := h.downloadService.GetByID(req.VideoID); err == nil && exists != nil {
			// DB记录中已存在该视频ID，说明已下载过，跳过下载
			utils.Info("⏭️ [视频下载] 视频ID已存在记录中(DB)，跳过下载: ID=%s", req.VideoID)
			responseData := map[string]interface{}{
				"success": true,
				"skipped": true,
				"message": "视频已下载（基于ID检查）",
			}
			responseBytes, _ := json.Marshal(responseData)
			h.sendJSONResponse(Conn, 200, responseBytes)
			return true
		}
	}

	// 生成文件名：优先使用视频ID确保唯一性
	filename := utils.GenerateVideoFilename(req.Title, req.VideoID)

	// 检查文件名中是否已经包含分辨率信息（避免重复添加）
	hasResolutionInFilename := false
	if req.Width > 0 && req.Height > 0 {
		resolutionPattern := fmt.Sprintf("_%dx%d", req.Width, req.Height)
		hasResolutionInFilename = strings.Contains(filename, resolutionPattern)
	} else if req.Resolution != "" {
		cleanResolution := strings.ReplaceAll(req.Resolution, " ", "")
		cleanResolution = strings.ReplaceAll(cleanResolution, "×", "x")
		cleanResolution = strings.ReplaceAll(cleanResolution, "X", "x")
		hasResolutionInFilename = strings.Contains(filename, "_"+cleanResolution) || strings.Contains(filename, cleanResolution)
	}

	// 如果有分辨率信息且文件名中还没有，添加到文件名中（与前端命名方式一致）
	if !hasResolutionInFilename && (req.FileFormat != "" || req.Width > 0 || req.Height > 0 || req.Resolution != "") {
		var qualityInfo string
		if req.FileFormat != "" {
			qualityInfo = req.FileFormat
		} else {
			qualityInfo = "quality"
		}

		// 优先使用 width 和 height，其次使用 resolution 字符串
		if req.Width > 0 && req.Height > 0 {
			qualityInfo += fmt.Sprintf("_%dx%d", req.Width, req.Height)
		} else if req.Resolution != "" {
			// 清理分辨率字符串，移除空格和特殊字符
			cleanResolution := strings.ReplaceAll(req.Resolution, " ", "")
			cleanResolution = strings.ReplaceAll(cleanResolution, "×", "x")
			cleanResolution = strings.ReplaceAll(cleanResolution, "X", "x")
			qualityInfo += "_" + cleanResolution
		}

		// 在添加分辨率信息前，需要先移除扩展名
		base := strings.TrimSuffix(filename, filepath.Ext(filename))
		ext := filepath.Ext(filename)
		if ext == "" {
			ext = ".mp4"
		}
		filename = base + "_" + qualityInfo + ext
		utils.Info("📐 [视频下载] 添加分辨率信息到文件名: %s", qualityInfo)
	} else if hasResolutionInFilename {
		utils.Info("📐 [视频下载] 文件名中已包含分辨率信息，跳过添加")
	}

	// 确保文件扩展名
	filename = utils.EnsureExtension(filename, ".mp4")
	videoPath := filepath.Join(savePath, filename)

	// 检查文件是否已存在（作为备用检查，主要检查已通过ID完成）
	if !req.ForceSave {
		if stat, err := os.Stat(videoPath); err == nil {
			// 文件已存在，返回成功但不重新下载
			fileSize := float64(stat.Size()) / (1024 * 1024)
			relativePath, _ := filepath.Rel(downloadsDir, videoPath)
			utils.Info("⏭️ [视频下载] 文件已存在，跳过: %s", relativePath)

			// 注意：不再手动保存下载记录，因为队列系统已经处理了记录保存
			// 移除重复的记录调用以避免数据库中出现重复记录

			responseData := map[string]interface{}{
				"success":      true,
				"path":         videoPath,
				"relativePath": relativePath,
				"size":         fileSize,
				"skipped":      true,
				"message":      "文件已存在，跳过下载",
			}
			responseBytes, _ := json.Marshal(responseData)
			h.sendJSONResponse(Conn, 200, responseBytes)
			return true
		}
	}

	// 判断是否需要解密
	needDecrypt := req.Key != ""

	// 临时文件路径
	tmpPath := videoPath + ".tmp"

	// 进度回调
	var lastLogTime time.Time
	onProgress := func(progress float64, downloaded int64, total int64) {
		// 每秒打印一次日志，避免刷屏
		now := time.Now()
		if now.Sub(lastLogTime) >= time.Second {
			// 转换为MB
			downloadedMB := float64(downloaded) / (1024 * 1024)
			totalMB := float64(total) / (1024 * 1024)
			percentage := progress * 100

			utils.Info("📥 [视频下载] 进度: %.2f%% (%.2f/%.2f MB)", percentage, downloadedMB, totalMB)

			// 发送 WebSocket 事件
			if h.wsHub != nil {
				// api_client.js 只识别 type='cmd'
				h.wsHub.Broadcast(map[string]interface{}{
					"type": "cmd",
					"data": map[string]interface{}{
						"action": "download_progress",
						"payload": map[string]interface{}{
							"videoId":    req.VideoID,
							"percentage": percentage, // 前端 expect "percentage"
							"downloaded": downloaded,
							"total":      total,
							"speed":      0,
						},
					},
				})
			}
			lastLogTime = now
		}
	}

	// 使用 Gopeed 下载
	utils.Info("🚀 [视频下载] 使用 Gopeed 引擎: %s", req.Title)

	// 创建 Context (支持取消)
	ctx, cancel := context.WithCancel(Conn.Request.Context())
	h.activeDownloads.Store(req.VideoID, cancel)
	defer h.activeDownloads.Delete(req.VideoID)
	// 注意：这里不要由 defer 调用 cancel()，因为 DownloadSync 是阻塞的
	// 如果在这里 defer cancel()，当函数返回时会取消，但此时下载可能已经完成
	// 而且我们希望在 DownloadSync 返回前保持 context 有效
	// 只有在发生错误或外部取消时，ctx 才会结束
	// 实际上，defer cancel() 是良好的实践，确保不泄露
	defer cancel()

	// 执行下载
	// 注意：GopeedService.DownloadSync 默认使用 30 分钟超时（如果在内部配置了）
	// 这里我们可以传入带有超时的 context
	downloadCtx, downloadCancel := context.WithTimeout(ctx, 30*time.Minute)
	defer downloadCancel()

	// 获取单文件连接数配置
	connections := 8 // 默认值
	cfg := config.Get()
	if cfg != nil && cfg.DownloadConnections > 0 {
		connections = cfg.DownloadConnections
	}

	err = h.gopeedService.DownloadSync(downloadCtx, req.VideoURL, tmpPath, connections, onProgress)
	if err != nil {
		utils.Error("❌ [视频下载] Gopeed 下载失败: %v", err)
		h.sendErrorResponse(Conn, fmt.Errorf("下载失败: %v", err))
		return true
	}

	// 验证文件存在
	stat, err := os.Stat(tmpPath)
	if err != nil || stat.Size() == 0 {
		utils.Error("❌ [视频下载] 下载文件无效")
		h.sendErrorResponse(Conn, fmt.Errorf("下载文件无效"))
		return true
	}

	// 解密
	if needDecrypt {
		utils.Info("🔐 [视频下载] 开始解密...")
		if err := utils.DecryptFileInPlace(tmpPath, req.Key, "", 0); err != nil {
			utils.Error("❌ [视频下载] 解密失败: %v", err)
			h.sendErrorResponse(Conn, fmt.Errorf("解密失败: %v", err))
			return true
		}
		utils.Info("✓ [视频下载] 解密完成")
	}

	// 重命名为最终文件
	if err := os.Rename(tmpPath, videoPath); err != nil {
		os.Remove(tmpPath)
		utils.Error("❌ [视频下载] 重命名文件失败: %v", err)
		h.sendErrorResponse(Conn, fmt.Errorf("重命名文件失败: %v", err))
		return true
	}

	fileSize := float64(stat.Size()) / (1024 * 1024)
	relativePath, _ := filepath.Rel(downloadsDir, videoPath)

	statusMsg := ""
	if needDecrypt {
		statusMsg = " [已解密]"
	}
	utils.Info("✓ [视频下载] 视频已保存" + statusMsg)

	// 保存下载记录
	if h.downloadService != nil {
		record := &database.DownloadRecord{
			ID:           req.VideoID,
			VideoID:      req.VideoID,
			Title:        req.Title,
			Author:       req.Author,
			Duration:     0, // 暂时无法获取准确时长，除非前端传递
			FileSize:     int64(stat.Size()),
			FilePath:     videoPath,
			Format:       "mp4",
			Resolution:   req.Resolution,
			Status:       database.DownloadStatusCompleted,
			DownloadTime: time.Now(),
			LikeCount:    req.LikeCount,
			CommentCount: req.CommentCount,
			ForwardCount: req.ForwardCount,
			FavCount:     req.FavCount,
		}
		if err := h.downloadService.Create(record); err != nil {
			utils.Error("保存下载记录失败: %v", err)
		} else {
			utils.Info("已保存下载记录: %s", record.Title)
		}
	}

	responseData := map[string]interface{}{
		"success":      true,
		"path":         videoPath,
		"relativePath": relativePath,
		"size":         fileSize,
		"decrypted":    needDecrypt,
	}
	responseBytes, err := json.Marshal(responseData)
	if err != nil {
		utils.HandleError(err, "生成响应JSON")
		h.sendErrorResponse(Conn, err)
		return true
	}
	h.sendJSONResponse(Conn, 200, responseBytes)
	return true
}

// abs 返回 int64 的绝对值
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// downloadVideoWithRetry 执行一次视频下载尝试（支持重试和断点续传）
func (h *UploadHandler) downloadVideoWithRetry(ctx context.Context, client *http.Client, httpReq *http.Request, req struct {
	VideoURL  string
	VideoID   string
	Title     string
	Author    string
	Key       string
	ForceSave bool
}, videoPath string, needDecrypt bool, resumeOffset int64, written *int64, expectedTotalSize *int64) error {
	tmpPath := videoPath + ".tmp"
	prefixLen := int64(131072) // 128KB 加密前缀长度

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		// 确保错误时也尝试关闭响应体（如果存在）
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return fmt.Errorf("请求失败: %v", err)
	}

	// 确保响应体总是被关闭
	defer func() {
		if resp != nil && resp.Body != nil {
			// 尝试完全读取并关闭，避免连接泄漏
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	// 包装 resp.Body 以显示进度
	if req.Title != "" { // 只对有标题的请求（真实下载）显示进度
		resp.Body = &utils.ProgressReader{
			Ctx:    ctx, // 传递上下文以支持取消
			Reader: resp.Body,
			Total:  resp.ContentLength,
			OnProgress: func(current, total int64) {
				if total > 0 {
					percent := float64(current) / float64(total) * 100
					// 使用 \r 在同一行刷新? 不，标准日志会换行。
					// 这里的日志系统是 utils.Info，通常会换行。
					// 为了避免刷屏，我们在 ProgressReader 中已经限制了1秒一次。
					// 但如果是日志文件，\r 没用。
					// 可以在这里再次控制频率或格式。
					utils.Info("📥 [视频下载] 进度: %.2f%% (%.2f/%.2f MB)",
						percent, float64(current)/(1024*1024), float64(total)/(1024*1024))

					// 广播进度到 WebSocket
					if h.wsHub != nil {
						h.wsHub.BroadcastCommand("download_progress", map[string]interface{}{
							"videoUrl":   req.VideoURL,
							"videoId":    req.VideoID,
							"title":      req.Title,
							"current":    current,
							"total":      total,
							"percentage": percent,
						})
					}
				} else {
					utils.Info("📥 [视频下载] 已下载: %.2f MB", float64(current)/(1024*1024))
				}
			},
		}
	}

	// 检查响应状态（支持 200 和 206 Partial Content）
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		// 读取并丢弃响应体，确保连接可以复用
		io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// 如果服务器不支持 Range，重新下载
	if resumeOffset > 0 && resp.StatusCode != http.StatusPartialContent {
		utils.Warn("⚠️ [视频下载] 服务器不支持断点续传，需要重新下载")
		return fmt.Errorf("服务器不支持断点续传")
	}

	// 计算期望的总文件大小
	if resp.StatusCode == http.StatusPartialContent {
		// 断点续传：总大小 = 已下载 + Content-Length
		if resp.ContentLength > 0 {
			*expectedTotalSize = resumeOffset + resp.ContentLength
		} else {
			*expectedTotalSize = -1 // 未知大小
		}
	} else {
		// 完整下载：总大小 = Content-Length
		*expectedTotalSize = resp.ContentLength
	}

	if *expectedTotalSize > 0 {
		sizeMB := float64(*expectedTotalSize) / (1024 * 1024)
		utils.Info("📦 [视频下载] 期望文件大小: %.2f MB", sizeMB)
	}

	// 打开/创建文件（断点续传时追加，否则创建新文件）
	var out *os.File
	if resumeOffset > 0 {
		out, err = os.OpenFile(tmpPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("打开文件失败（断点续传）: %v", err)
		}
	} else {
		out, err = os.Create(tmpPath)
		if err != nil {
			return fmt.Errorf("创建文件失败: %v", err)
		}
	}
	defer out.Close()

	*written = 0

	if needDecrypt {
		if resumeOffset >= prefixLen {
			// 断点续传：已下载部分 >= 128KB，直接复制剩余数据（未加密）
			utils.Info("🔐 [视频下载] 加密视频断点续传，复制剩余数据（未加密部分）...")
			n, err := io.Copy(out, resp.Body)
			if err != nil {
				return fmt.Errorf("写入视频数据失败: %v", err)
			}
			*written = n
		} else {
			// 从头开始下载：需要解密前128KB
			utils.Info("🔐 [视频下载] 开始解密下载...")

			// 解析 key 为 uint64
			seed, err := utils.ParseKey(req.Key)
			if err != nil {
				return fmt.Errorf("解析密钥失败: %v", err)
			}

			// 生成 128KB 解密数组
			decryptorPrefix := util.GenerateDecryptorArray(seed, int(prefixLen))
			utils.Info("🔑 [视频下载] 从 key 生成解密数组，长度: %d bytes", len(decryptorPrefix))

			// 读取前缀数据
			prefixData := make([]byte, prefixLen)
			n, err := io.ReadFull(resp.Body, prefixData)
			if err != nil && err != io.ErrUnexpectedEOF {
				return fmt.Errorf("读取前缀失败: %v", err)
			}
			prefixData = prefixData[:n]

			utils.Info("📖 [视频下载] 读取前缀: %d bytes", n)

			// 解密前缀
			decryptedPrefix := util.XorDecrypt(prefixData, decryptorPrefix)

			// 写入解密后的前缀
			nw, err := out.Write(decryptedPrefix)
			if err != nil {
				return fmt.Errorf("写入解密前缀失败: %v", err)
			}
			*written += int64(nw)

			utils.Info("✓ [视频下载] 前缀解密完成")

			// 复制剩余数据（未加密）
			buf := make([]byte, 32*1024)
			for {
				select {
				case <-ctx.Done():
					return fmt.Errorf("下载已取消")
				default:
				}

				nr, er := resp.Body.Read(buf)
				if nr > 0 {
					nw, ew := out.Write(buf[0:nr])
					if ew != nil {
						return fmt.Errorf("写入视频数据失败: %v", ew)
					}
					*written += int64(nw)
					if nr != nw {
						return fmt.Errorf("写入不完整: 期望 %d, 实际 %d", nr, nw)
					}
				}
				if er != nil {
					if er != io.EOF {
						return fmt.Errorf("读取视频数据失败: %v", er)
					}
					break
				}
			}
		}
	} else {
		// 普通下载（非加密）
		utils.Info("📥 [视频下载] 开始下载...")
		n, err := io.Copy(out, resp.Body)
		if err != nil {
			return fmt.Errorf("写入视频数据失败: %v", err)
		}
		*written = n
	}

	// 关闭文件
	if err := out.Close(); err != nil {
		return fmt.Errorf("关闭文件失败: %v", err)
	}

	// 验证文件大小
	if *written == 0 {
		os.Remove(tmpPath)
		return fmt.Errorf("下载的文件为空")
	}

	// 验证实际写入的大小（对于断点续传，需要加上已下载的部分）
	actualWritten := *written
	if resumeOffset > 0 {
		actualWritten = resumeOffset + *written
	}

	// 如果知道期望大小，验证是否匹配
	if *expectedTotalSize > 0 {
		if actualWritten != *expectedTotalSize {
			diff := abs(actualWritten - *expectedTotalSize)
			diffPercent := float64(diff) / float64(*expectedTotalSize) * 100
			// 如果差异超过1%，认为下载不完整
			if diffPercent > 1.0 {
				return fmt.Errorf("下载不完整: 期望 %d bytes (%.2f MB), 实际 %d bytes (%.2f MB), 差异 %.2f%%",
					*expectedTotalSize, float64(*expectedTotalSize)/(1024*1024),
					actualWritten, float64(actualWritten)/(1024*1024),
					diffPercent)
			}
			// 差异在1%以内，记录警告但继续
			if diff > 0 {
				utils.Warn("⚠️ [视频下载] 文件大小略有差异: 期望 %.2f MB, 实际 %.2f MB, 差异 %.2f%%",
					float64(*expectedTotalSize)/(1024*1024),
					float64(actualWritten)/(1024*1024),
					diffPercent)
			}
		} else {
			utils.Info("✓ [视频下载] 下载大小验证通过: %.2f MB", float64(actualWritten)/(1024*1024))
		}
	}

	return nil
}

// HandleUploadStatus 查询已上传的分片列表
func (h *UploadHandler) HandleUploadStatus(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/upload_status" {
		return false
	}

	if h.getConfig() != nil && h.getConfig().SecretToken != "" {
		if Conn.Request.Header.Get("X-Local-Auth") != h.getConfig().SecretToken {
			headers := http.Header{}
			headers.Set("Content-Type", "application/json")
			headers.Set("X-Content-Type-Options", "nosniff")
			Conn.StopRequest(401, `{"success":false,"error":"unauthorized"}`, headers)
			return true
		}
	}
	if h.getConfig() != nil && len(h.getConfig().AllowedOrigins) > 0 {
		origin := Conn.Request.Header.Get("Origin")
		if origin != "" {
			allowed := false
			for _, o := range h.getConfig().AllowedOrigins {
				if o == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				headers.Set("X-Content-Type-Options", "nosniff")
				Conn.StopRequest(403, `{"success":false,"error":"forbidden_origin"}`, headers)
				return true
			}
		}
	}

	body, err := io.ReadAll(Conn.Request.Body)
	if err != nil {
		h.sendErrorResponse(Conn, err)
		return true
	}
	_ = Conn.Request.Body.Close()

	var req struct {
		UploadId string `json:"uploadId"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		h.sendErrorResponse(Conn, err)
		return true
	}
	if req.UploadId == "" {
		h.sendErrorResponse(Conn, fmt.Errorf("missing_uploadId"))
		return true
	}

	downloadsDir, err := h.getDownloadsDir()
	if err != nil {
		h.sendErrorResponse(Conn, err)
		return true
	}
	upDir := filepath.Join(downloadsDir, ".uploads", req.UploadId)
	entries, err := os.ReadDir(upDir)
	if err != nil {
		h.sendErrorResponse(Conn, err)
		return true
	}

	parts := []int{}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".part") && len(name) >= 10 {
			idxStr := strings.TrimSuffix(name, ".part")
			if n, convErr := strconv.Atoi(strings.TrimLeft(idxStr, "0")); convErr == nil {
				parts = append(parts, n)
			} else if idxStr == "000000" { // 0 特判
				parts = append(parts, 0)
			}
		}
	}

	resp := map[string]interface{}{"success": true, "parts": parts}
	h.sendJSONResponse(Conn, 200, resp)
	return true
}

// sendSuccessResponse 发送成功响应
func (h *UploadHandler) sendSuccessResponse(Conn *SunnyNet.HttpConn) {
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
	Conn.StopRequest(200, string(response.SuccessJSON(nil)), headers)
}

// sendJSONResponse 发送JSON响应 (Assuming body is the Data part of standard response, or needs to be wrapped)
// CAUTION: The existing callsites pass a full object like {"success": true, "uploadId": ...}.
// We need to change the semantic. 'body' passed here should be treated as the 'Data' field content if we want consistent structure.
// However, the existing callsites manually construct {"success": true...}. We should refactor callsites first?
// Let's refactor sendJSONResponse to accept interface{} instead of []byte and encoding it.
// sendJSONResponse 发送JSON响应
func (h *UploadHandler) sendJSONResponse(Conn *SunnyNet.HttpConn, statusCode int, data interface{}) {
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

	var respBytes []byte
	if b, ok := data.([]byte); ok {
		respBytes = b
	} else {
		var err error
		respBytes, err = json.Marshal(data)
		if err != nil {
			utils.Error("JSON marshaling failed: %v", err)
			// Fallback error
			Conn.StopRequest(500, `{"success":false,"error":"internal_error"}`, headers)
			return
		}
	}

	Conn.StopRequest(statusCode, string(respBytes), headers)
}

// sendErrorResponse 发送错误响应
func (h *UploadHandler) sendErrorResponse(Conn *SunnyNet.HttpConn, err error) {
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

// 注意：saveDownloadRecord 方法已被移除
// 原因：该方法创建的下载记录使用未格式化的文件名（包含 ？ 字符），
// 而队列系统的 CompleteDownload() 方法使用格式化的文件名（？ 替换为 _），
// 导致出现重复记录且文件名格式不一致。
// 现在统一使用队列系统的 CompleteDownload() 方法来创建下载记录。
