package cache

import (
	"sync"
	"time"
)

// metricsCache 缓存客户端推送的监控数据
var (
	metricsCache      = make(map[string]string) // clientID -> metrics
	metricsCacheMutex sync.RWMutex
	metricsCacheTTL   = 60 * time.Second // 缓存 60 秒
	metricsTimestamp  = make(map[string]time.Time)
)

// cleanExpired 清理过期数据（必须在写锁下调用）
func cleanExpired() {
	now := time.Now()
	for clientID, timestamp := range metricsTimestamp {
		if now.Sub(timestamp) > metricsCacheTTL {
			delete(metricsCache, clientID)
			delete(metricsTimestamp, clientID)
		}
	}
}

// UpdateClientMetrics 更新客户端的监控数据
func UpdateClientMetrics(clientID string, metricsData string) {
	metricsCacheMutex.Lock()
	defer metricsCacheMutex.Unlock()

	metricsCache[clientID] = metricsData
	metricsTimestamp[clientID] = time.Now()
}

// GetClientMetrics 获取客户端的监控数据
func GetClientMetrics() (string, error) {
	// 先用写锁清理过期数据
	metricsCacheMutex.Lock()
	cleanExpired()
	metricsCacheMutex.Unlock()

	// 再用读锁读取
	metricsCacheMutex.RLock()
	defer metricsCacheMutex.RUnlock()

	if len(metricsCache) == 0 {
		return "", nil
	}

	// 返回第一个客户端的数据
	for _, metrics := range metricsCache {
		return metrics, nil
	}

	return "", nil
}

// GetAllClientMetrics 获取所有客户端的监控数据
func GetAllClientMetrics() map[string]string {
	// 先用写锁清理过期数据
	metricsCacheMutex.Lock()
	cleanExpired()
	metricsCacheMutex.Unlock()

	// 再用读锁读取
	metricsCacheMutex.RLock()
	defer metricsCacheMutex.RUnlock()

	result := make(map[string]string, len(metricsCache))
	for clientID, metrics := range metricsCache {
		result[clientID] = metrics
	}

	return result
}
