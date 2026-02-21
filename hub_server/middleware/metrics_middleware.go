package middleware

import (
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ────────────────────────────────────────────
// MetricsStore  全局 API 指标存储
// ────────────────────────────────────────────

// MetricsStore 存储 API 调用指标
type MetricsStore struct {
	mu sync.RWMutex

	// 累计计数
	TotalCalls   atomic.Int64
	SuccessCalls atomic.Int64
	FailedCalls  atomic.Int64

	// 响应时间采样（最近 10000 个请求）
	responseTimes []float64
	rtIndex       int
	rtFull        bool

	// 每端点统计
	endpointStats map[string]*EndpointStat

	// 时序数据采集器
	timeSeries *TimeSeriesCollector
}

// EndpointStat 单个 API 端点的统计
type EndpointStat struct {
	Calls   atomic.Int64
	Errors  atomic.Int64
	TotalMs atomic.Int64
}

// TimeSeriesPoint 单个时序数据点
type TimeSeriesPoint struct {
	Timestamp     time.Time
	Connections   float64 // 从 metrics_cache 读取
	APISuccess    float64
	APIFailed     float64
	AvgRespTimeMs float64
}

// TimeSeriesCollector 时序数据采集器（环形缓冲区）
type TimeSeriesCollector struct {
	mu     sync.RWMutex
	points []TimeSeriesPoint
	maxLen int
}

// ────────────────────────────────────────────
// 全局实例
// ────────────────────────────────────────────

var globalStore *MetricsStore

// InitMetricsStore 初始化全局指标存储
func InitMetricsStore() *MetricsStore {
	store := &MetricsStore{
		responseTimes: make([]float64, 10000),
		endpointStats: make(map[string]*EndpointStat),
		timeSeries: &TimeSeriesCollector{
			points: make([]TimeSeriesPoint, 0, 2880), // 24h × 30s = 2880 个点
			maxLen: 2880,
		},
	}
	globalStore = store

	// 启动后台时序采集 goroutine
	go store.collectLoop()

	return store
}

// GetMetricsStore 获取全局指标存储
func GetMetricsStore() *MetricsStore {
	return globalStore
}

// ────────────────────────────────────────────
// HTTP 中间件
// ────────────────────────────────────────────

// MetricsMiddleware 返回 gorilla/mux 中间件，记录 API 请求
func MetricsMiddleware(store *MetricsStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 只记录 /api/ 请求
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: 200}

			next.ServeHTTP(rw, r)

			elapsed := time.Since(start)
			store.record(r.URL.Path, rw.statusCode, elapsed)
		})
	}
}

// responseWriter 包装 http.ResponseWriter 以捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// ────────────────────────────────────────────
// 记录逻辑
// ────────────────────────────────────────────

func (s *MetricsStore) record(path string, statusCode int, elapsed time.Duration) {
	ms := float64(elapsed.Milliseconds())

	// 累计计数
	s.TotalCalls.Add(1)
	if statusCode >= 200 && statusCode < 400 {
		s.SuccessCalls.Add(1)
	} else {
		s.FailedCalls.Add(1)
	}

	// 响应时间环形缓冲区
	s.mu.Lock()
	s.responseTimes[s.rtIndex] = ms
	s.rtIndex++
	if s.rtIndex >= len(s.responseTimes) {
		s.rtIndex = 0
		s.rtFull = true
	}

	// 每端点计数
	// 将路径参数归一化（例如 /api/admin/user/5 → /api/admin/user/:id）
	normalized := normalizePath(path)
	stat, ok := s.endpointStats[normalized]
	if !ok {
		stat = &EndpointStat{}
		s.endpointStats[normalized] = stat
	}
	s.mu.Unlock()

	stat.Calls.Add(1)
	stat.TotalMs.Add(int64(ms))
	if statusCode >= 400 {
		stat.Errors.Add(1)
	}
}

// normalizePath 将 /api/admin/user/5 归一化为 /api/admin/user/:id
func normalizePath(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if p == "" {
			continue
		}
		// 如果是纯数字或者看起来像 UUID/device ID，则替换为 :id
		if isIDSegment(p) {
			parts[i] = ":id"
		}
	}
	return strings.Join(parts, "/")
}

func isIDSegment(s string) bool {
	if len(s) == 0 {
		return false
	}
	// 纯数字
	allDigit := true
	for _, c := range s {
		if c < '0' || c > '9' {
			allDigit = false
			break
		}
	}
	if allDigit {
		return true
	}
	// 32+ 字符的 hex 字符串（UUID / device ID）
	if len(s) >= 32 {
		allHex := true
		for _, c := range s {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || c == '-') {
				allHex = false
				break
			}
		}
		return allHex
	}
	return false
}

// ────────────────────────────────────────────
// 查询方法（供 controller 调用）
// ────────────────────────────────────────────

// GetSummary 返回 API 调用摘要
func (s *MetricsStore) GetSummary() (total, success, failed int64, successRate float64, avgMs float64) {
	total = s.TotalCalls.Load()
	success = s.SuccessCalls.Load()
	failed = s.FailedCalls.Load()
	if total > 0 {
		successRate = float64(success) / float64(total) * 100
	}
	avgMs = s.getAvgResponseTime()
	return
}

// GetResponseTimePercentiles 返回 P50, P95, P99
func (s *MetricsStore) GetResponseTimePercentiles() (p50, p95, p99 float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var data []float64
	if s.rtFull {
		data = make([]float64, len(s.responseTimes))
		copy(data, s.responseTimes)
	} else {
		data = make([]float64, s.rtIndex)
		copy(data, s.responseTimes[:s.rtIndex])
	}

	if len(data) == 0 {
		return 0, 0, 0
	}

	sort.Float64s(data)
	p50 = percentile(data, 50)
	p95 = percentile(data, 95)
	p99 = percentile(data, 99)
	return
}

func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(p)/100.0*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func (s *MetricsStore) getAvgResponseTime() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := s.rtIndex
	if s.rtFull {
		count = len(s.responseTimes)
	}
	if count == 0 {
		return 0
	}

	var sum float64
	for i := 0; i < count; i++ {
		sum += s.responseTimes[i]
	}
	return math.Round(sum/float64(count)*100) / 100
}

// TopEndpoints 返回调用量最高的 N 个端点
type EndpointInfo struct {
	Path   string  `json:"path"`
	Calls  int64   `json:"calls"`
	Errors int64   `json:"errors"`
	AvgMs  float64 `json:"avgMs"`
}

func (s *MetricsStore) TopEndpoints(n int) []EndpointInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]EndpointInfo, 0, len(s.endpointStats))
	for path, stat := range s.endpointStats {
		calls := stat.Calls.Load()
		avgMs := float64(0)
		if calls > 0 {
			avgMs = float64(stat.TotalMs.Load()) / float64(calls)
		}
		list = append(list, EndpointInfo{
			Path:   path,
			Calls:  calls,
			Errors: stat.Errors.Load(),
			AvgMs:  math.Round(avgMs*100) / 100,
		})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Calls > list[j].Calls
	})

	if len(list) > n {
		list = list[:n]
	}
	return list
}

// ────────────────────────────────────────────
// 时序数据采集
// ────────────────────────────────────────────

// collectLoop 每 30 秒采样一次
func (s *MetricsStore) collectLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 记录上次的计数值用于计算增量
	var lastSuccess, lastFailed int64
	var lastTotalMs int64

	for range ticker.C {
		curSuccess := s.SuccessCalls.Load()
		curFailed := s.FailedCalls.Load()

		deltaSuccess := curSuccess - lastSuccess
		deltaFailed := curFailed - lastFailed
		deltaTotal := deltaSuccess + deltaFailed

		// 计算这个时间窗口的平均响应时间
		var curTotalMs int64
		s.mu.RLock()
		for _, stat := range s.endpointStats {
			curTotalMs += stat.TotalMs.Load()
		}
		s.mu.RUnlock()
		deltaTotalMs := curTotalMs - lastTotalMs

		avgMs := float64(0)
		if deltaTotal > 0 {
			avgMs = float64(deltaTotalMs) / float64(deltaTotal)
		}

		point := TimeSeriesPoint{
			Timestamp:     time.Now(),
			APISuccess:    float64(deltaSuccess),
			APIFailed:     float64(deltaFailed),
			AvgRespTimeMs: math.Round(avgMs*100) / 100,
		}

		s.timeSeries.mu.Lock()
		if len(s.timeSeries.points) >= s.timeSeries.maxLen {
			// 移除最早的点
			s.timeSeries.points = s.timeSeries.points[1:]
		}
		s.timeSeries.points = append(s.timeSeries.points, point)
		s.timeSeries.mu.Unlock()

		lastSuccess = curSuccess
		lastFailed = curFailed
		lastTotalMs = curTotalMs
	}
}

// QueryTimeSeries 查询指定时间范围内的时序数据
func (s *MetricsStore) QueryTimeSeries(duration time.Duration, maxPoints int) []TimeSeriesPoint {
	s.timeSeries.mu.RLock()
	defer s.timeSeries.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	var filtered []TimeSeriesPoint
	for _, p := range s.timeSeries.points {
		if p.Timestamp.After(cutoff) {
			filtered = append(filtered, p)
		}
	}

	// 如果数据点太多，做降采样
	if len(filtered) > maxPoints && maxPoints > 0 {
		step := len(filtered) / maxPoints
		if step < 1 {
			step = 1
		}
		sampled := make([]TimeSeriesPoint, 0, maxPoints)
		for i := 0; i < len(filtered); i += step {
			sampled = append(sampled, filtered[i])
		}
		return sampled
	}

	return filtered
}
