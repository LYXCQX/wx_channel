package cloud

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"wx_channel/internal/config"
	"wx_channel/internal/metrics"
	"wx_channel/internal/utils"
	hubws "wx_channel/internal/websocket"

	"github.com/gorilla/websocket"
)

// Connector 云端连接器
type Connector struct {
	cfg   *config.Config
	local *hubws.Hub
	conn  *websocket.Conn
	mu    sync.Mutex

	clientID string
	ctx      context.Context
	cancel   context.CancelFunc

	// 重连策略
	retryCount int
	maxRetries int           // 最大重试次数（0 = 无限重试）
	baseDelay  time.Duration // 基础延迟
	maxDelay   time.Duration // 最大延迟
}

// NewConnector 创建云端连接器
func NewConnector(cfg *config.Config, localHub *hubws.Hub) *Connector {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Connector{
		cfg:        cfg,
		local:      localHub,
		clientID:   cfg.MachineID,
		ctx:        ctx,
		cancel:     cancel,
		maxRetries: 0,                  // 0 = 无限重试
		baseDelay:  1 * time.Second,    // 基础延迟 1 秒
		maxDelay:   2 * time.Minute,    // 最大延迟 2 分钟
	}

	if c.clientID == "" {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "unknown"
		}
		c.clientID = fmt.Sprintf("%s-%d", hostname, time.Now().Unix()%10000)
	}

	return c
}

// Start 启动连接器
func (c *Connector) Start() {
	if c.cfg.CloudHubURL == "" {
		utils.LogInfo("云端管理未启用 (未配置 cloud_hub_url)")
		return
	}

	utils.LogInfo("正在启动云端连接器 (ID: %s, URL: %s)", c.clientID, c.cfg.CloudHubURL)

	go c.connectLoop()
}

// Stop 停止连接器
func (c *Connector) Stop() {
	c.cancel()
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()
}

func (c *Connector) connectLoop() {
	c.retryCount = 0

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			metrics.ReconnectAttemptsTotal.Inc()
			err := c.connect()
			if err != nil {
				c.retryCount++
				delay := c.calculateBackoff()
				
				if c.maxRetries > 0 && c.retryCount >= c.maxRetries {
					utils.LogError("云端连接失败 (重试 %d/%d): %v", c.retryCount, c.maxRetries, err)
					utils.LogError("达到最大重试次数，停止重连")
					return
				}
				
				utils.LogWarn("云端连接失败 (重试 %d): %v, %v 后重试...", c.retryCount, err, delay)
				time.Sleep(delay)
				continue
			}

			// 连接成功，重置计数器
			c.retryCount = 0
			metrics.ReconnectSuccessTotal.Inc()
			metrics.WSConnectionsTotal.Inc()
			utils.LogInfo("✓ 已连接到云端 Hub")
			c.handleConnection()
			metrics.WSConnectionsTotal.Dec()
			utils.LogWarn("云端 Hub 连接已断开，尝试重新连接...")
			time.Sleep(1 * time.Second) // 短暂延迟后立即重连
		}
	}
}

// calculateBackoff 计算指数退避延迟
func (c *Connector) calculateBackoff() time.Duration {
	if c.retryCount <= 0 {
		return c.baseDelay
	}

	// 指数退避：1s, 2s, 4s, 8s, 16s, 32s, 64s, 120s (max)
	multiplier := 1 << uint(c.retryCount-1)
	delay := c.baseDelay * time.Duration(multiplier)
	
	if delay > c.maxDelay {
		delay = c.maxDelay
	}
	
	// 添加随机抖动 (0-25%)，避免雷鸣群效应
	jitter := time.Duration(rand.Int63n(int64(delay / 4)))
	return delay + jitter
}

func (c *Connector) connect() error {
	header := http.Header{}
	if c.cfg.CloudSecret != "" {
		header.Add("X-Cloud-Secret", c.cfg.CloudSecret)
	}
	header.Add("X-Client-ID", c.clientID)

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(c.cfg.CloudHubURL, header)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	return nil
}

func (c *Connector) handleConnection() {
	// 检查是否有绑定任务
	if c.cfg.BindToken != "" {
		utils.LogInfo("检测到绑定码，正在发送绑定请求...")
		payload := map[string]string{"token": c.cfg.BindToken}
		data, _ := json.Marshal(payload)

		msg := CloudMessage{
			ID:        fmt.Sprintf("bind-%d", time.Now().Unix()),
			Type:      MsgTypeBind,
			ClientID:  c.clientID,
			Payload:   data,
			Timestamp: time.Now().Unix(),
		}

		if err := c.send(msg); err != nil {
			utils.LogError("绑定请求发送失败: %v", err)
		}
	}

	// 启动心跳
	go c.heartbeatLoop()
	
	// 启动监控数据推送（如果启用了监控）
	if c.cfg.MetricsEnabled {
		go c.metricsLoop()
	}

	// 监听消息
	utils.LogInfo("开始监听云端消息...")
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			utils.LogError("读取消息失败: %v", err)
			return
		}

		var msg CloudMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			utils.LogError("云端消息解析失败: %v", err)
			continue
		}

		// 使用 goroutine 处理消息，但添加 panic 恢复
		go func(m CloudMessage) {
			defer func() {
				if r := recover(); r != nil {
					utils.LogError("消息处理 panic: %v", r)
				}
			}()
			c.processMessage(m)
		}(msg)
	}
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *Connector) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	missedHeartbeats := 0
	maxMissed := 3 // 连续 3 次心跳失败则重连

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.sendHeartbeat(); err != nil {
				missedHeartbeats++
				utils.LogWarn("心跳发送失败 (%d/%d): %v", missedHeartbeats, maxMissed, err)
				
				if missedHeartbeats >= maxMissed {
					utils.LogError("心跳连续失败，触发重连")
					c.mu.Lock()
					if c.conn != nil {
						c.conn.Close() // 触发 handleConnection 退出
					}
					c.mu.Unlock()
					return
				}
			} else {
				// 心跳成功，重置计数器
				if missedHeartbeats > 0 {
					utils.LogInfo("心跳恢复正常")
					missedHeartbeats = 0
				}
			}
		}
	}
}

// sendHeartbeat 发送心跳消息
func (c *Connector) sendHeartbeat() error {
	hostname, _ := os.Hostname()
	payload := HeartbeatPayload{
		Hostname: hostname,
		Version:  c.cfg.Version,
		Status:   "running",
	}
	payloadData, _ := json.Marshal(payload)

	msg := CloudMessage{
		ID:        fmt.Sprintf("hb-%d", time.Now().UnixNano()),
		Type:      MsgTypeHeartbeat,
		ClientID:  c.clientID,
		Payload:   payloadData,
		Timestamp: time.Now().Unix(),
	}

	// 设置写入超时
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.conn == nil {
		metrics.HeartbeatsFailed.Inc()
		return fmt.Errorf("connection closed")
	}

	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	defer c.conn.SetWriteDeadline(time.Time{})

	data, err := json.Marshal(msg)
	if err != nil {
		metrics.HeartbeatsFailed.Inc()
		return err
	}

	err = c.conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		metrics.HeartbeatsFailed.Inc()
		return err
	}

	metrics.HeartbeatsSent.Inc()
	metrics.WSMessagesSent.WithLabelValues("heartbeat").Inc()
	return nil
}

func (c *Connector) send(msg CloudMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("connection closed")
	}

	// 1. 序列化消息
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	originalSize := len(data)
	messageType := websocket.TextMessage

	// 2. 如果启用压缩且数据大于阈值，则压缩
	if c.cfg.CompressionEnabled && originalSize > c.cfg.CompressionThreshold {
		compressed, err := c.compressData(data)
		if err == nil && len(compressed) < originalSize {
			// 压缩成功且有效果
			data = compressed
			messageType = websocket.BinaryMessage // 使用二进制消息类型标识压缩数据
			
			// 记录压缩指标
			metrics.CompressionBytesIn.Add(float64(originalSize))
			metrics.CompressionBytesOut.Add(float64(len(compressed)))
			
			compressionRate := float64(originalSize-len(compressed)) / float64(originalSize) * 100
			utils.LogInfo("数据压缩: %d -> %d 字节 (压缩率: %.1f%%)", 
				originalSize, len(compressed), compressionRate)
		}
	}

	// 3. 发送数据
	metrics.WSMessagesSent.WithLabelValues(string(msg.Type)).Inc()
	return c.conn.WriteMessage(messageType, data)
}

// compressData 压缩数据
func (c *Connector) compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	
	if err := writer.Close(); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

func (c *Connector) processMessage(msg CloudMessage) {
	metrics.WSMessagesReceived.WithLabelValues(string(msg.Type)).Inc()
	
	if msg.Type != MsgTypeCommand {
		return
	}

	var cmd CommandPayload
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		utils.LogError("指令载荷解析失败: %v", err)
		c.sendError(msg.ID, "Invalid command payload")
		return
	}

	utils.LogInfo("收到云端指令: %s", cmd.Action)

	switch cmd.Action {
	case "api_call":
		c.handleAPICall(msg.ID, cmd.Data)
	default:
		utils.LogError("未知操作: %s", cmd.Action)
		c.sendError(msg.ID, fmt.Sprintf("Unknown action: %s", cmd.Action))
	}
}

func (c *Connector) handleAPICall(reqID string, data json.RawMessage) {
	var call struct {
		Key  string          `json:"key"`
		Body json.RawMessage `json:"body"`
	}

	if err := json.Unmarshal(data, &call); err != nil {
		utils.LogError("API 参数解析失败: %v", err)
		c.sendError(reqID, "Invalid API call parameters")
		return
	}

	// 检查本地 Hub 是否可用
	if c.local == nil {
		utils.LogError("本地 Hub 未初始化")
		c.sendError(reqID, "Local hub not initialized")
		return
	}

	// 调用本地 API
	respData, err := c.local.CallAPI(call.Key, call.Body, 30*time.Second)
	if err != nil {
		utils.LogError("API 调用失败: %v", err)
		c.sendError(reqID, err.Error())
		return
	}

	// 转换搜索接口返回数据为统一格式
	if call.Key == "key:channels:contact_list" {
		respData = c.transformSearchResponse(call.Body, respData)
	}
	
	// 返回结果
	c.sendResponse(reqID, true, respData, "")
}

// transformSearchResponse 转换搜索响应数据为统一格式
func (c *Connector) transformSearchResponse(requestBody, responseData json.RawMessage) json.RawMessage {
	// 解析请求参数，获取 type
	var reqBody struct {
		Type int `json:"type"`
	}
	if err := json.Unmarshal(requestBody, &reqBody); err != nil {
		utils.LogError("[ERROR] 解析请求参数失败: %v", err)
		return responseData // 返回原始数据
	}

	// 解析原始响应
	var rawResp struct {
		BaseResponse struct {
			Ret    int    `json:"Ret"`
			ErrMsg string `json:"ErrMsg"`
		} `json:"BaseResponse"`
		Data struct {
			InfoList   []interface{} `json:"infoList"`   // Type 1: 用户列表
			ObjectList []interface{} `json:"objectList"` // Type 3: 视频列表
			LastBuff   string        `json:"lastBuff"`
			Continue   int           `json:"continueFlag"`
		} `json:"data"`
	}

	if err := json.Unmarshal(responseData, &rawResp); err != nil {
		utils.LogError("[ERROR] 解析原始响应失败: %v", err)
		return responseData // 返回原始数据
	}

	// 根据 type 选择对应的列表
	var list []interface{}
	if reqBody.Type == 1 {
		// Type 1: 找人 - 使用 infoList
		list = rawResp.Data.InfoList
	} else {
		// Type 2/3: 找直播/找视频 - 使用 objectList
		list = rawResp.Data.ObjectList
	}

	if list == nil {
		list = make([]interface{}, 0)
	}

	// 构造统一的响应格式
	optimized := map[string]interface{}{
		"list":        list,
		"next_marker": rawResp.Data.LastBuff,
		"has_more":    rawResp.Data.Continue != 0,
	}

	optimizedBytes, err := json.Marshal(optimized)
	if err != nil {
		utils.LogError("[ERROR] 序列化优化响应失败: %v", err)
		return responseData // 返回原始数据
	}

	utils.LogInfo("[DEBUG] 数据转换成功: type=%d, listSize=%d, hasMore=%v", 
		reqBody.Type, len(list), rawResp.Data.Continue != 0)

	return optimizedBytes
}

func (c *Connector) sendResponse(reqID string, success bool, data json.RawMessage, errMsg string) {
	resp := ResponsePayload{
		RequestID: reqID,
		Success:   success,
		Data:      data,
		Error:     errMsg,
	}
	respData, err := json.Marshal(resp)
	if err != nil {
		utils.LogError("响应序列化失败: %v", err)
		return
	}

	msg := CloudMessage{
		ID:        fmt.Sprintf("resp-%s", reqID),
		Type:      MsgTypeResponse,
		ClientID:  c.clientID,
		Payload:   respData,
		Timestamp: time.Now().Unix(),
	}

	if err := c.send(msg); err != nil {
		utils.LogError("发送响应失败: %v", err)
	}
}

func (c *Connector) sendError(reqID string, errMsg string) {
	c.sendResponse(reqID, false, nil, errMsg)
}

// metricsLoop 定期推送监控数据到 Hub
func (c *Connector) metricsLoop() {
	ticker := time.NewTicker(15 * time.Second) // 每 15 秒推送一次
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.pushMetrics(); err != nil {
				utils.LogWarn("推送监控数据失败: %v", err)
			}
		}
	}
}

// pushMetrics 推送监控数据
func (c *Connector) pushMetrics() error {
	// 从本地 metrics 端点获取数据
	metricsURL := fmt.Sprintf("http://localhost:%d/metrics", c.cfg.MetricsPort)
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(metricsURL)
	if err != nil {
		return fmt.Errorf("获取监控数据失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取监控数据失败: %w", err)
	}

	// 构造监控数据消息
	payload := map[string]string{
		"metrics": string(body),
	}
	payloadData, _ := json.Marshal(payload)

	msg := CloudMessage{
		ID:        fmt.Sprintf("metrics-%d", time.Now().UnixNano()),
		Type:      "metrics", // 新的消息类型
		ClientID:  c.clientID,
		Payload:   payloadData,
		Timestamp: time.Now().Unix(),
	}

	return c.send(msg)
}
