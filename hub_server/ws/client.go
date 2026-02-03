package ws

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"sync"
	"time"
	"wx_channel/hub_server/cache"
	"wx_channel/hub_server/database"
	"wx_channel/hub_server/models"
	"wx_channel/hub_server/services"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID       string
	Hostname string
	Version  string
	LastSeen time.Time
	Conn     *websocket.Conn
	mu       sync.Mutex

	respChannels map[string]chan ResponsePayload
	respMu       sync.RWMutex
	Hub          *Hub
}

func NewClient(id string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:           id,
		LastSeen:     time.Now(),
		Conn:         conn,
		respChannels: make(map[string]chan ResponsePayload),
		Hub:          hub,
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	// 设置最大消息大小为 10MB
	c.Conn.SetReadLimit(10 * 1024 * 1024)

	for {
		messageType, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		// 如果是二进制消息，说明是压缩数据，先解压
		if messageType == websocket.BinaryMessage {
			decompressed, err := c.decompressData(message)
			if err != nil {
				log.Printf("解压失败: %v", err)
				continue
			}
			message = decompressed
		}

		var msg CloudMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		c.handleMessage(msg)
	}
}

func (c *Client) handleMessage(msg CloudMessage) {
	c.mu.Lock()
	c.LastSeen = time.Now()
	c.mu.Unlock()

	switch msg.Type {
	case MsgTypeHeartbeat:
		var p HeartbeatPayload
		json.Unmarshal(msg.Payload, &p)
		c.mu.Lock()
		c.Hostname = p.Hostname
		c.Version = p.Version
		c.mu.Unlock()

		database.UpsertNode(&models.Node{
			ID:       c.ID,
			Hostname: p.Hostname,
			Version:  p.Version,
			Status:   "online",
			LastSeen: time.Now(),
		})

	case "metrics":
		var payload struct {
			Metrics string `json:"metrics"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err == nil {
			cache.UpdateClientMetrics(c.ID, payload.Metrics)
		}

	case MsgTypeResponse:
		var resp ResponsePayload
		if err := json.Unmarshal(msg.Payload, &resp); err != nil {
			log.Printf("解析响应失败: %v", err)
			return
		}
		
		c.respMu.RLock()
		ch, ok := c.respChannels[resp.RequestID]
		c.respMu.RUnlock()
		
		if ok {
			select {
			case ch <- resp:
			default:
				log.Printf("响应通道已满: RequestID=%s", resp.RequestID)
			}
		}

	case MsgTypeBind:
		var payload struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err == nil {
			err := services.ProcessBindRequest(c.ID, payload.Token)

			response := map[string]interface{}{
				"type":    "bind_result",
				"success": err == nil,
			}
			if err != nil {
				response["error"] = err.Error()
			}

			respBytes, _ := json.Marshal(response)
			c.WriteMessage(respBytes)
		}
	}
}

// decompressData 解压数据
func (c *Client) decompressData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func (c *Client) WriteMessage(msg []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteMessage(websocket.TextMessage, msg)
}
