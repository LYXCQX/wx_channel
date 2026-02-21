package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"wx_channel/hub_server/cache"
	"wx_channel/hub_server/middleware"
)

// MetricsSummary 监控指标摘要
type MetricsSummary struct {
	Connections       int              `json:"connections"`
	ConnectionsTrend  float64          `json:"connectionsTrend"`
	APICalls          int              `json:"apiCalls"`
	APICallsTrend     float64          `json:"apiCallsTrend"`
	SuccessRate       float64          `json:"successRate"`
	AvgResponseTime   float64          `json:"avgResponseTime"`
	ResponseTimeTrend float64          `json:"responseTimeTrend"`
	HeartbeatsSent    int              `json:"heartbeatsSent"`
	HeartbeatsFailed  int              `json:"heartbeatsFailed"`
	CompressionRate   float64          `json:"compressionRate"`
	BytesSaved        int64            `json:"bytesSaved"`
	DetailedMetrics   []DetailedMetric `json:"detailedMetrics"`
}

// DetailedMetric 详细指标
type DetailedMetric struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// TimeSeriesData 时序数据
type TimeSeriesData struct {
	Connections  TimeSeriesPoints   `json:"connections"`
	APICalls     APICallsPoints     `json:"apiCalls"`
	ResponseTime ResponseTimePoints `json:"responseTime"`
	Endpoints    EndpointsData      `json:"endpoints"`
}

type TimeSeriesPoints struct {
	Labels []string  `json:"labels"`
	Values []float64 `json:"values"`
}

type APICallsPoints struct {
	Labels  []string  `json:"labels"`
	Success []float64 `json:"success"`
	Failed  []float64 `json:"failed"`
}

type ResponseTimePoints struct {
	Labels []string  `json:"labels"`
	P50    []float64 `json:"p50"`
	P95    []float64 `json:"p95"`
	P99    []float64 `json:"p99"`
}

// EndpointsData API 端点统计数据
type EndpointsData struct {
	Labels []string  `json:"labels"`
	Values []float64 `json:"values"`
}

// GetMetricsSummary 获取监控指标摘要
func GetMetricsSummary(w http.ResponseWriter, r *http.Request) {
	// 1. 从客户端推送的 Prometheus 数据获取 WS/心跳/压缩指标
	metricsData, err := cache.GetClientMetrics()
	if err != nil {
		http.Error(w, "获取指标失败", http.StatusInternalServerError)
		return
	}

	clientMetrics := parsePrometheusMetrics(metricsData)

	// WS 连接数
	connections := int(clientMetrics["wx_channel_ws_connections_total"])

	// 心跳
	heartbeatsSent := int(clientMetrics["wx_channel_heartbeats_sent_total"])
	heartbeatsFailed := int(clientMetrics["wx_channel_heartbeats_failed_total"])

	// 压缩
	bytesIn := int64(clientMetrics["wx_channel_compression_bytes_in_total"])
	bytesOut := int64(clientMetrics["wx_channel_compression_bytes_out_total"])
	compressionRate := 0.0
	if bytesIn > 0 {
		compressionRate = float64(bytesIn-bytesOut) / float64(bytesIn) * 100
	}

	// 2. 从服务端 MetricsStore 获取 API 指标
	var apiCalls int64
	var successRate, avgResponseTime float64
	store := middleware.GetMetricsStore()
	if store != nil {
		total, _, _, sr, avg := store.GetSummary()
		apiCalls = total
		successRate = sr
		avgResponseTime = avg
	}

	// 3. 构建详细指标
	p50, p95, p99 := float64(0), float64(0), float64(0)
	if store != nil {
		p50, p95, p99 = store.GetResponseTimePercentiles()
	}

	detailedMetrics := []DetailedMetric{
		{Name: "WebSocket 连接总数", Value: fmt.Sprintf("%d", connections), Description: "当前活跃的 WebSocket 连接数量"},
		{Name: "API 调用总数", Value: fmt.Sprintf("%d", apiCalls), Description: "所有 API 的累计调用次数"},
		{Name: "API 成功率", Value: fmt.Sprintf("%.2f%%", successRate), Description: "API 调用成功的百分比"},
		{Name: "平均响应时间", Value: fmt.Sprintf("%.2fms", avgResponseTime), Description: "API 请求的平均响应时间"},
		{Name: "P50 响应时间", Value: fmt.Sprintf("%.2fms", p50), Description: "50% 的请求在此时间内完成"},
		{Name: "P95 响应时间", Value: fmt.Sprintf("%.2fms", p95), Description: "95% 的请求在此时间内完成"},
		{Name: "P99 响应时间", Value: fmt.Sprintf("%.2fms", p99), Description: "99% 的请求在此时间内完成"},
		{Name: "心跳发送次数", Value: fmt.Sprintf("%d", heartbeatsSent), Description: "发送的心跳消息总数"},
		{Name: "心跳失败次数", Value: fmt.Sprintf("%d", heartbeatsFailed), Description: "失败的心跳消息总数"},
		{Name: "压缩率", Value: fmt.Sprintf("%.2f%%", compressionRate), Description: "数据压缩节省的百分比"},
	}

	summary := MetricsSummary{
		Connections:       connections,
		ConnectionsTrend:  0,
		APICalls:          int(apiCalls),
		APICallsTrend:     0,
		SuccessRate:       successRate,
		AvgResponseTime:   avgResponseTime,
		ResponseTimeTrend: 0,
		HeartbeatsSent:    heartbeatsSent,
		HeartbeatsFailed:  heartbeatsFailed,
		CompressionRate:   compressionRate,
		BytesSaved:        bytesIn - bytesOut,
		DetailedMetrics:   detailedMetrics,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetTimeSeriesData 获取时序数据
func GetTimeSeriesData(w http.ResponseWriter, r *http.Request) {
	timeRange := r.URL.Query().Get("range")
	if timeRange == "" {
		timeRange = "15m"
	}

	duration := parseDuration(timeRange)
	store := middleware.GetMetricsStore()

	data := &TimeSeriesData{}

	if store != nil {
		// 查询真实时序数据
		points := store.QueryTimeSeries(duration, 30)

		labels := make([]string, len(points))
		apiSuccess := make([]float64, len(points))
		apiFailed := make([]float64, len(points))
		respAvg := make([]float64, len(points))
		connectionsValues := make([]float64, len(points))

		for i, p := range points {
			labels[i] = p.Timestamp.Format("15:04")
			apiSuccess[i] = p.APISuccess
			apiFailed[i] = p.APIFailed
			respAvg[i] = p.AvgRespTimeMs
			connectionsValues[i] = p.Connections
		}

		// 如果没有时序数据（刚启动），生成至少一个当前点
		if len(points) == 0 {
			now := time.Now()
			labels = []string{now.Format("15:04")}
			connectionsValues = []float64{0}
			apiSuccess = []float64{0}
			apiFailed = []float64{0}
			respAvg = []float64{0}

			// 尝试从 Prometheus 缓存获取当前连接数
			metricsData, _ := cache.GetClientMetrics()
			if metricsData != "" {
				cm := parsePrometheusMetrics(metricsData)
				connectionsValues[0] = cm["wx_channel_ws_connections_total"]
			}
		}

		data.Connections = TimeSeriesPoints{Labels: labels, Values: connectionsValues}
		data.APICalls = APICallsPoints{Labels: labels, Success: apiSuccess, Failed: apiFailed}
		// 使用当前百分位数作为响应时间线（因为时序点中只有平均值）
		p50, p95, p99 := store.GetResponseTimePercentiles()
		p50Values := make([]float64, len(labels))
		p95Values := make([]float64, len(labels))
		p99Values := make([]float64, len(labels))
		for i := range labels {
			p50Values[i] = p50
			p95Values[i] = p95
			p99Values[i] = p99
		}
		// 如果有真实时序数据，用每个点的平均响应时间来近似
		if len(points) > 0 {
			for i, p := range points {
				if p.AvgRespTimeMs > 0 {
					p50Values[i] = p.AvgRespTimeMs * 0.8
					p95Values[i] = p.AvgRespTimeMs * 1.5
					p99Values[i] = p.AvgRespTimeMs * 2.0
				}
			}
		}
		data.ResponseTime = ResponseTimePoints{Labels: labels, P50: p50Values, P95: p95Values, P99: p99Values}

		// API 端点统计（替代负载均衡分布）
		endpoints := store.TopEndpoints(6)
		epLabels := make([]string, len(endpoints))
		epValues := make([]float64, len(endpoints))
		for i, ep := range endpoints {
			// 简化路径显示
			epLabels[i] = simplifyPath(ep.Path)
			epValues[i] = float64(ep.Calls)
		}
		data.Endpoints = EndpointsData{Labels: epLabels, Values: epValues}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// ────────────────────────────────────────────
// 辅助函数
// ────────────────────────────────────────────

// parsePrometheusMetrics 解析 Prometheus 格式文本
func parsePrometheusMetrics(metricsData string) map[string]float64 {
	metrics := make(map[string]float64)
	if metricsData == "" {
		return metrics
	}

	for _, line := range strings.Split(metricsData, "\n") {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			value, err := strconv.ParseFloat(parts[1], 64)
			if err == nil {
				metrics[parts[0]] = value
			}
		}
	}
	return metrics
}

// parseDuration 解析时间范围
func parseDuration(timeRange string) time.Duration {
	switch timeRange {
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "1h":
		return 1 * time.Hour
	case "6h":
		return 6 * time.Hour
	case "24h":
		return 24 * time.Hour
	default:
		return 15 * time.Minute
	}
}

// simplifyPath 简化 API 路径用于显示
func simplifyPath(path string) string {
	// /api/admin/users → admin/users
	path = strings.TrimPrefix(path, "/api/")
	// /api/auth/login → auth/login
	if len(path) > 20 {
		parts := strings.Split(path, "/")
		if len(parts) > 2 {
			return parts[len(parts)-2] + "/" + parts[len(parts)-1]
		}
	}
	return path
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
