package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// WebSocket 连接指标
	WSConnectionsTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "wx_channel_ws_connections_total",
		Help: "当前 WebSocket 连接总数",
	})

	WSMessagesReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "wx_channel_ws_messages_received_total",
		Help: "接收的 WebSocket 消息总数",
	}, []string{"type"})

	WSMessagesSent = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "wx_channel_ws_messages_sent_total",
		Help: "发送的 WebSocket 消息总数",
	}, []string{"type"})

	// API 调用指标
	APICallsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "wx_channel_api_calls_total",
		Help: "API 调用总数",
	}, []string{"key", "status"})

	APICallDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "wx_channel_api_call_duration_seconds",
		Help:    "API 调用耗时（秒）",
		Buckets: prometheus.DefBuckets,
	}, []string{"key"})

	// 重连指标
	ReconnectAttemptsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wx_channel_reconnect_attempts_total",
		Help: "重连尝试总次数",
	})

	ReconnectSuccessTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wx_channel_reconnect_success_total",
		Help: "重连成功总次数",
	})

	// 心跳指标
	HeartbeatsSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wx_channel_heartbeats_sent_total",
		Help: "发送的心跳总数",
	})

	HeartbeatsFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wx_channel_heartbeats_failed_total",
		Help: "失败的心跳总数",
	})

	// 压缩指标
	CompressionBytesIn = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wx_channel_compression_bytes_in_total",
		Help: "压缩前的字节总数",
	})

	CompressionBytesOut = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wx_channel_compression_bytes_out_total",
		Help: "压缩后的字节总数",
	})

	// 负载均衡指标
	LoadBalancerSelections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "wx_channel_load_balancer_selections_total",
		Help: "负载均衡器选择次数",
	}, []string{"client_id"})

	ActiveRequestsPerClient = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "wx_channel_active_requests_per_client",
		Help: "每个客户端的活跃请求数",
	}, []string{"client_id"})
)
