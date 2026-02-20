package services

import (
	"context"
	"sync"
	"time"

	"wx_channel/internal/database"
	"wx_channel/internal/utils"
)

// QueueProcessor 自动处理下载队列的后台服务
type QueueProcessor struct {
	queueService    *QueueService
	downloader      *ChunkedDownloader
	maxConcurrent   int
	checkInterval   time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	activeDownloads sync.Map // map[string]bool - 跟踪活动下载
}

// NewQueueProcessor 创建一个新的队列处理器
func NewQueueProcessor(queueService *QueueService, downloader *ChunkedDownloader, maxConcurrent int) *QueueProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	if maxConcurrent <= 0 {
		maxConcurrent = 3 // 默认并发数
	}

	return &QueueProcessor{
		queueService:  queueService,
		downloader:    downloader,
		maxConcurrent: maxConcurrent,
		checkInterval: 5 * time.Second,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start 启动队列处理器
func (p *QueueProcessor) Start() error {
	utils.Info("[QueueProcessor] 启动队列处理器，最大并发: %d", p.maxConcurrent)

	p.wg.Add(1)
	go p.processLoop()

	return nil
}

// Stop 停止队列处理器
func (p *QueueProcessor) Stop() error {
	utils.Info("[QueueProcessor] 停止队列处理器...")

	p.cancel()
	p.wg.Wait()

	utils.Info("[QueueProcessor] 队列处理器已停止")
	return nil
}

// processLoop 主处理循环
func (p *QueueProcessor) processLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.checkInterval)
	defer ticker.Stop()

	// 立即执行一次检查
	p.checkAndStartTasks()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.checkAndStartTasks()
		}
	}
}

// checkAndStartTasks 检查队列并启动新任务
func (p *QueueProcessor) checkAndStartTasks() {
	// 获取当前活动下载数
	activeCount := p.GetActiveCount()

	// 如果已达到并发限制，跳过
	if activeCount >= p.maxConcurrent {
		return
	}

	// 获取待下载任务
	availableSlots := p.maxConcurrent - activeCount
	pendingItems, err := p.queueService.GetByStatus(database.QueueStatusPending)
	if err != nil {
		utils.Error("[QueueProcessor] 获取待下载任务失败: %v", err)
		return
	}

	if len(pendingItems) == 0 {
		return
	}

	utils.Info("[QueueProcessor] 发现 %d 个待下载任务，可用槽位: %d", len(pendingItems), availableSlots)

	// 启动新任务
	for i := 0; i < len(pendingItems) && i < availableSlots; i++ {
		item := pendingItems[i]

		// 检查是否已在下载
		if _, exists := p.activeDownloads.Load(item.ID); exists {
			continue
		}

		// 标记为活动
		p.activeDownloads.Store(item.ID, true)

		// 启动下载
		p.wg.Add(1)
		go p.processTask(&item)
	}
}

// processTask 处理单个下载任务
func (p *QueueProcessor) processTask(item *database.QueueItem) {
	defer p.wg.Done()
	defer p.activeDownloads.Delete(item.ID)

	utils.Info("[QueueProcessor] 开始下载: %s - %s", item.Author, item.Title)

	// 使用 ChunkedDownloader 下载
	if err := p.downloader.StartDownload(item); err != nil {
		utils.Error("[QueueProcessor] 下载启动失败: %s - %v", item.Title, err)

		// 标记失败
		p.queueService.FailDownload(item.ID, err.Error())

		// 检查是否需要重试
		if item.RetryCount < 3 {
			p.queueService.IncrementRetryCount(item.ID)
			p.queueService.UpdateStatus(item.ID, database.QueueStatusPending)
			utils.Info("[QueueProcessor] 将重试: %s (第 %d 次)", item.Title, item.RetryCount+1)
		}
		return
	}

	// 等待下载完成
	// ChunkedDownloader.StartDownload 是异步的，它会在 goroutine 中执行下载
	// 我们需要等待下载完成或失败
	p.waitForDownloadCompletion(item.ID)
}

// waitForDownloadCompletion 等待下载完成
func (p *QueueProcessor) waitForDownloadCompletion(itemID string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute) // 30分钟超时

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-timeout:
			utils.Warn("[QueueProcessor] 下载超时: %s", itemID)
			p.queueService.FailDownload(itemID, "下载超时")
			return
		case <-ticker.C:
			// 检查下载状态
			item, err := p.queueService.GetByID(itemID)
			if err != nil {
				utils.Error("[QueueProcessor] 获取任务状态失败: %v", err)
				return
			}

			if item == nil {
				utils.Warn("[QueueProcessor] 任务不存在: %s", itemID)
				return
			}

			// 检查是否完成或失败
			switch item.Status {
			case database.QueueStatusCompleted:
				utils.Info("[QueueProcessor] 下载完成: %s - %s", item.Author, item.Title)
				return
			case database.QueueStatusFailed:
				utils.Error("[QueueProcessor] 下载失败: %s - %s", item.Title, item.ErrorMessage)

				// 检查是否需要重试
				if item.RetryCount < 3 {
					p.queueService.IncrementRetryCount(itemID)
					p.queueService.UpdateStatus(itemID, database.QueueStatusPending)
					utils.Info("[QueueProcessor] 将重试: %s (第 %d 次)", item.Title, item.RetryCount+1)
				}
				return
			case database.QueueStatusPaused:
				utils.Info("[QueueProcessor] 下载已暂停: %s", item.Title)
				return
			}
		}
	}
}

// GetActiveCount 获取当前活动下载数
func (p *QueueProcessor) GetActiveCount() int {
	count := 0
	p.activeDownloads.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// GetActiveDownloads 获取活动下载的 ID 列表
func (p *QueueProcessor) GetActiveDownloads() []string {
	ids := make([]string, 0)
	p.activeDownloads.Range(func(key, value interface{}) bool {
		if id, ok := key.(string); ok {
			ids = append(ids, id)
		}
		return true
	})
	return ids
}

// SetMaxConcurrent 设置最大并发数
func (p *QueueProcessor) SetMaxConcurrent(max int) {
	if max > 0 {
		p.maxConcurrent = max
		utils.Info("[QueueProcessor] 最大并发数已更新为: %d", max)
	}
}

// SetCheckInterval 设置检查间隔
func (p *QueueProcessor) SetCheckInterval(interval time.Duration) {
	if interval > 0 {
		p.checkInterval = interval
		utils.Info("[QueueProcessor] 检查间隔已更新为: %v", interval)
	}
}
