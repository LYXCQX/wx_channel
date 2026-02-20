package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"wx_channel/internal/database"
	"wx_channel/internal/utils"
)

// ChunkedDownloader 处理下载队列（使用 Gopeed 引擎）
type ChunkedDownloader struct {
	queueService  *QueueService
	gopeedService *GopeedService
	settings      *database.SettingsRepository
	downloadDir   string

	mu            sync.RWMutex
	activeItems   map[string]*DownloadState
	progressChan  chan ProgressUpdate
	ctx           context.Context
	cancel        context.CancelFunc
	maxRetries    int
}

// DownloadState 跟踪活动下载的状态
type DownloadState struct {
	QueueItem      *database.QueueItem
	LastUpdateTime time.Time
	BytesPerSecond int64
	CancelFunc     context.CancelFunc
	mu             sync.Mutex
	StartTime      time.Time
}

// ProgressUpdate 表示下载进度更新
type ProgressUpdate struct {
	QueueID         string `json:"queueId"`
	DownloadedSize  int64  `json:"downloadedSize"`
	TotalSize       int64  `json:"totalSize"`
	ChunksCompleted int    `json:"chunksCompleted"`
	ChunksTotal     int    `json:"chunksTotal"`
	Speed           int64  `json:"speed"`
	Status          string `json:"status"`
	ErrorMessage    string `json:"errorMessage,omitempty"`
}

// NewChunkedDownloader 创建一个新的 ChunkedDownloader
func NewChunkedDownloader(queueService *QueueService, gopeedService *GopeedService) *ChunkedDownloader {
	ctx, cancel := context.WithCancel(context.Background())
	settingsRepo := database.NewSettingsRepository()

	// 加载设置
	settings, err := settingsRepo.Load()
	if err != nil {
		settings = database.DefaultSettings()
	}

	return &ChunkedDownloader{
		queueService:  queueService,
		gopeedService: gopeedService,
		settings:      settingsRepo,
		downloadDir:   settings.DownloadDir,
		activeItems:   make(map[string]*DownloadState),
		progressChan:  make(chan ProgressUpdate, 100),
		ctx:           ctx,
		cancel:        cancel,
		maxRetries:    settings.MaxRetries,
	}
}

// ProgressChannel 返回进度更新通道
func (d *ChunkedDownloader) ProgressChannel() <-chan ProgressUpdate {
	return d.progressChan
}

// StartDownload 开始下载队列项目
func (d *ChunkedDownloader) StartDownload(item *database.QueueItem) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 检查是否已在下载
	if _, exists := d.activeItems[item.ID]; exists {
		return fmt.Errorf("download already in progress for item: %s", item.ID)
	}

	// 创建下载上下文
	ctx, cancel := context.WithCancel(d.ctx)

	state := &DownloadState{
		QueueItem:      item,
		LastUpdateTime: time.Now(),
		StartTime:      time.Now(),
		CancelFunc:     cancel,
	}

	d.activeItems[item.ID] = state

	// 在 goroutine 中开始下载
	go d.downloadItem(ctx, state)

	return nil
}

// downloadItem 执行实际下载（使用 Gopeed）
func (d *ChunkedDownloader) downloadItem(ctx context.Context, state *DownloadState) {
	item := state.QueueItem

	// 标记为正在下载
	if err := d.queueService.StartDownload(item.ID); err != nil {
		d.handleError(item.ID, fmt.Errorf("failed to start download: %w", err))
		return
	}

	// 准备下载目录
	downloadPath, err := d.prepareDownloadPath(item)
	if err != nil {
		d.handleError(item.ID, fmt.Errorf("failed to prepare download path: %w", err))
		return
	}

	// 检查 Gopeed 服务是否可用
	if d.gopeedService == nil {
		d.handleError(item.ID, fmt.Errorf("Gopeed service not initialized"))
		return
	}

	// 使用 Gopeed 下载
	utils.Info("[队列下载] 使用 Gopeed 下载: %s", item.Title)
	
	startTime := time.Now()
	lastDownloadedSize := int64(0)
	lastUpdateTime := time.Now()
	
	onProgress := func(progress float64, downloaded int64, total int64) {
		// 每秒更新一次进度
		now := time.Now()
		if now.Sub(lastUpdateTime) < time.Second && progress < 1.0 {
			return
		}
		lastUpdateTime = now
		
		// 计算速度
		elapsed := time.Since(startTime).Seconds()
		var speed int64
		if elapsed > 0 {
			speed = int64(float64(downloaded-lastDownloadedSize) / elapsed)
		}
		
		// 更新状态
		state.mu.Lock()
		state.LastUpdateTime = now
		state.BytesPerSecond = speed
		state.mu.Unlock()
		
		// 计算分片进度（模拟）
		chunksCompleted := int(float64(item.ChunksTotal) * progress)
		
		// 发送进度更新
		d.sendProgress(ProgressUpdate{
			QueueID:         item.ID,
			DownloadedSize:  downloaded,
			TotalSize:       total,
			ChunksCompleted: chunksCompleted,
			ChunksTotal:     item.ChunksTotal,
			Speed:           speed,
			Status:          database.QueueStatusDownloading,
		})
		
		// 更新数据库
		d.queueService.UpdateProgress(item.ID, downloaded, chunksCompleted, speed)
		
		lastDownloadedSize = downloaded
	}
	
	// 调用 Gopeed 同步下载
	err = d.gopeedService.DownloadSync(ctx, item.VideoURL, downloadPath, onProgress)
	if err != nil {
		// 检查是否被取消/暂停
		if ctx.Err() != nil {
			utils.Info("[队列下载] 下载已取消: %s", item.Title)
			return
		}
		d.handleError(item.ID, err)
		return
	}

	// 解密逻辑（如果需要）
	if item.DecryptKey != "" {
		utils.Info("🔐 [队列下载] 开始解密视频: %s", item.Title)
		if err := utils.DecryptFileInPlace(downloadPath, item.DecryptKey, "", 0); err != nil {
			d.handleError(item.ID, fmt.Errorf("解密失败: %w", err))
			return
		}
		utils.Info("✓ [队列下载] 解密完成: %s", item.Title)
	}

	// 验证文件完整性（不严格要求，只警告）
	if err := d.verifyFileIntegrity(downloadPath, item.TotalSize); err != nil {
		utils.Warn("[队列下载] 文件大小不匹配，但下载已完成: %s", item.Title)
	}

	// 标记为完成，传入实际的下载路径
	if err := d.queueService.CompleteDownloadWithPath(item.ID, downloadPath); err != nil {
		d.handleError(item.ID, fmt.Errorf("failed to mark download as completed: %w", err))
		return
	}

	utils.Info("✓ [队列下载] 下载完成: %s", item.Title)

	// 发送完成更新
	d.sendProgress(ProgressUpdate{
		QueueID:         item.ID,
		DownloadedSize:  item.TotalSize,
		TotalSize:       item.TotalSize,
		ChunksCompleted: item.ChunksTotal,
		ChunksTotal:     item.ChunksTotal,
		Speed:           0,
		Status:          database.QueueStatusCompleted,
	})

	// 从活动项目中移除
	d.mu.Lock()
	delete(d.activeItems, item.ID)
	d.mu.Unlock()
}

// prepareDownloadPath 准备项目的下载路径
func (d *ChunkedDownloader) prepareDownloadPath(item *database.QueueItem) (string, error) {
	baseDir, err := utils.GetBaseDir()
	if err != nil {
		return "", err
	}

	// 创建作者文件夹
	authorFolder := utils.CleanFolderName(item.Author)
	downloadDir := filepath.Join(baseDir, d.downloadDir, authorFolder)

	if err := utils.EnsureDir(downloadDir); err != nil {
		return "", fmt.Errorf("failed to create download directory: %w", err)
	}

	// 使用视频ID作为文件名
	filename := item.VideoID
	if filename == "" {
		filename = "video_" + time.Now().Format("20060102_150405")
	}
	filename = utils.EnsureExtension(filename, ".mp4")

	return filepath.Join(downloadDir, filename), nil
}

// verifyFileIntegrity 验证下载的文件大小是否与预期大小匹配
func (d *ChunkedDownloader) verifyFileIntegrity(filePath string, expectedSize int64) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	actualSize := fileInfo.Size()
	if actualSize != expectedSize {
		return fmt.Errorf("file size mismatch: expected %d bytes, got %d bytes", expectedSize, actualSize)
	}

	return nil
}

// handleError 处理下载错误
func (d *ChunkedDownloader) handleError(itemID string, err error) {
	utils.Error("[队列下载] 下载错误 %s: %v", itemID, err)

	// 获取项目详细信息以进行日志记录
	if item, getErr := d.queueService.GetByID(itemID); getErr == nil && item != nil {
		utils.LogDownloadError(item.ID, item.Title, item.Author, item.VideoURL, err, d.maxRetries)
	}

	// 标记为失败
	if markErr := d.queueService.FailDownload(itemID, err.Error()); markErr != nil {
		utils.Error("[队列下载] 标记失败时出错: %v", markErr)
	}

	// 发送错误更新
	d.sendProgress(ProgressUpdate{
		QueueID:      itemID,
		Status:       database.QueueStatusFailed,
		ErrorMessage: err.Error(),
	})

	// 从活动项目中移除
	d.mu.Lock()
	delete(d.activeItems, itemID)
	d.mu.Unlock()
}

// sendProgress 发送进度更新到通道
func (d *ChunkedDownloader) sendProgress(update ProgressUpdate) {
	select {
	case d.progressChan <- update:
	default:
		// 通道已满，跳过更新
	}
}

// PauseDownload 暂停活动下载
func (d *ChunkedDownloader) PauseDownload(itemID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	state, exists := d.activeItems[itemID]
	if !exists {
		return fmt.Errorf("no active download for item: %s", itemID)
	}

	state.CancelFunc()

	// 更新数据库中的状态
	if err := d.queueService.Pause(itemID); err != nil {
		return err
	}

	delete(d.activeItems, itemID)
	return nil
}

// ResumeDownload 恢复暂停的下载
func (d *ChunkedDownloader) ResumeDownload(itemID string) error {
	// 从队列获取项目
	item, err := d.queueService.GetByID(itemID)
	if err != nil {
		return err
	}
	if item == nil {
		return fmt.Errorf("queue item not found: %s", itemID)
	}

	// 在队列服务中恢复
	if err := d.queueService.Resume(itemID); err != nil {
		return err
	}

	// 开始下载
	return d.StartDownload(item)
}

// CancelDownload 取消活动下载
func (d *ChunkedDownloader) CancelDownload(itemID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	state, exists := d.activeItems[itemID]
	if !exists {
		return fmt.Errorf("no active download for item: %s", itemID)
	}

	state.CancelFunc()
	delete(d.activeItems, itemID)

	return nil
}

// GetActiveDownloads 返回活动下载 ID 列表
func (d *ChunkedDownloader) GetActiveDownloads() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ids := make([]string, 0, len(d.activeItems))
	for id := range d.activeItems {
		ids = append(ids, id)
	}
	return ids
}

// GetDownloadState 返回活动下载的状态
func (d *ChunkedDownloader) GetDownloadState(itemID string) (*DownloadState, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	state, exists := d.activeItems[itemID]
	return state, exists
}

// Stop 停止下载器并取消所有活动下载
func (d *ChunkedDownloader) Stop() {
	d.cancel()

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, state := range d.activeItems {
		state.CancelFunc()
	}
	d.activeItems = make(map[string]*DownloadState)

	close(d.progressChan)
}
