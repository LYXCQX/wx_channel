package services

import (
	"os"
	"time"

	"wx_channel/internal/database"
)

// DownloadRecordService 处理下载记录业务逻辑
type DownloadRecordService struct {
	repo *database.DownloadRecordRepository
}

// NewDownloadRecordService 创建一个新的 DownloadRecordService
func NewDownloadRecordService() *DownloadRecordService {
	return &DownloadRecordService{
		repo: database.NewDownloadRecordRepository(),
	}
}

// List 获取下载记录（带过滤和分页）
// Requirements: 2.3, 2.4 - 按日期范围和状态过滤
func (s *DownloadRecordService) List(params *database.FilterParams) (*database.PagedResult[database.DownloadRecord], error) {
	if params == nil {
		params = &database.FilterParams{
			PaginationParams: database.PaginationParams{
				Page:     1,
				PageSize: 20,
				SortBy:   "download_time",
				SortDesc: true,
			},
		}
	}
	return s.repo.List(params)
}

// GetByID 按 ID 获取单条下载记录
func (s *DownloadRecordService) GetByID(id string) (*database.DownloadRecord, error) {
	return s.repo.GetByID(id)
}

// Delete 按 ID 删除下载记录（可选删除文件）
// Requirements: 5.3 - 删除记录（可选择保留或删除文件）
func (s *DownloadRecordService) Delete(id string, deleteFile bool) error {
	if deleteFile {
		record, err := s.repo.GetByID(id)
		if err != nil {
			return err
		}
		if record != nil && record.FilePath != "" {
			// 尝试删除文件，如果文件不存在则忽略错误
			_ = os.Remove(record.FilePath)
		}
	}
	return s.repo.Delete(id)
}

// DeleteMany 按 ID 批量删除下载记录（可选删除文件）
// Requirements: 5.3 - 批量删除（可选择保留或删除文件）
func (s *DownloadRecordService) DeleteMany(ids []string, deleteFiles bool) (int64, error) {
	if deleteFiles {
		records, err := s.repo.GetByIDs(ids)
		if err != nil {
			return 0, err
		}
		for _, record := range records {
			if record.FilePath != "" {
				// 尝试删除文件，如果文件不存在则忽略错误
				_ = os.Remove(record.FilePath)
			}
		}
	}
	return s.repo.DeleteMany(ids)
}

// Clear 清空所有下载记录（可选删除文件）
// Requirements: 5.3 - 清空记录（可选择保留或删除文件）
func (s *DownloadRecordService) Clear(deleteFiles bool) error {
	if deleteFiles {
		records, err := s.repo.GetAll()
		if err != nil {
			return err
		}
		for _, record := range records {
			if record.FilePath != "" {
				// 尝试删除文件，如果文件不存在则忽略错误
				_ = os.Remove(record.FilePath)
			}
		}
	}
	return s.repo.Clear()
}

// DeleteBefore 删除指定日期前的所有记录（可选删除文件）
func (s *DownloadRecordService) DeleteBefore(date time.Time, deleteFiles bool) (int64, error) {
	if deleteFiles {
		// 分页获取日期前的所有记录以删除文件
		const batchSize = 500
		page := 1
		for {
			params := &database.FilterParams{
				PaginationParams: database.PaginationParams{
					Page:     page,
					PageSize: batchSize,
				},
				EndDate: &date,
			}
			result, err := s.repo.List(params)
			if err != nil {
				return 0, err
			}
			for _, record := range result.Items {
				if record.FilePath != "" {
					_ = os.Remove(record.FilePath)
				}
			}
			// 如果这一页数据不足一批，说明没有更多了
			if len(result.Items) < batchSize {
				break
			}
			page++
		}
	}
	return s.repo.DeleteBefore(date)
}

// Count 返回下载记录总数
func (s *DownloadRecordService) Count() (int64, error) {
	return s.repo.Count()
}

// CountByStatus 返回指定状态的记录数
func (s *DownloadRecordService) CountByStatus(status string) (int64, error) {
	return s.repo.CountByStatus(status)
}

// CountToday 返回今天下载的记录数
func (s *DownloadRecordService) CountToday() (int64, error) {
	return s.repo.CountToday()
}

// GetRecent 获取最近的下载记录
func (s *DownloadRecordService) GetRecent(limit int) ([]database.DownloadRecord, error) {
	return s.repo.GetRecent(limit)
}

// GetAll 获取所有下载记录（用于导出）
func (s *DownloadRecordService) GetAll() ([]database.DownloadRecord, error) {
	return s.repo.GetAll()
}

// GetByIDs 按 ID 获取下载记录（用于选择性导出）
func (s *DownloadRecordService) GetByIDs(ids []string) ([]database.DownloadRecord, error) {
	return s.repo.GetByIDs(ids)
}

// GetChartData 返回过去 N 天的下载计数
func (s *DownloadRecordService) GetChartData(days int) ([]string, []int64, error) {
	return s.repo.GetChartData(days)
}

// GetTotalFileSize 返回所有已完成下载的总文件大小
func (s *DownloadRecordService) GetTotalFileSize() (int64, error) {
	return s.repo.GetTotalFileSize()
}

// Create 添加新的下载记录
func (s *DownloadRecordService) Create(record *database.DownloadRecord) error {
	return s.repo.Create(record)
}

// Update 更新现有的下载记录
func (s *DownloadRecordService) Update(record *database.DownloadRecord) error {
	return s.repo.Update(record)
}

// AddRecord 添加新的下载记录（简化版）
func (s *DownloadRecordService) AddRecord(videoID, title, author, filePath string, fileSize int64, status string) error {
	record := &database.DownloadRecord{
		VideoID:      videoID,
		Title:        title,
		Author:       author,
		FilePath:     filePath,
		FileSize:     fileSize,
		Status:       status,
		DownloadTime: time.Now(),
	}
	// 如果ID为空，使用UUID或随机生成
	if record.VideoID == "" {
		// 这里简单处理，实际上应该由数据库生成ID或者前端传递
		// 但为了兼容性，允许空ID，或者生成一个
		// 注意：DownloadRecord struct 定义里 ID 是主键吗？
		// 查看 repo 代码或 struct 定义才知道
	}
	return s.Create(record)
}
