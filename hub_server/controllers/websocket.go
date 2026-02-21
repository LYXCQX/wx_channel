package controllers

import (
	"encoding/json"
	"net/http"
	"wx_channel/hub_server/ws"
)

// GetWSStats 获取 WebSocket 连接统计信息
func GetWSStats(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clients := hub.GetAllClientsStats()

		totalPings := int64(0)
		totalPongs := int64(0)
		totalMessages := int64(0)

		for _, client := range clients {
			if pings, ok := client["ping_count"].(int64); ok {
				totalPings += pings
			}
			if pongs, ok := client["pong_count"].(int64); ok {
				totalPongs += pongs
			}
			if sent, ok := client["messages_sent"].(int64); ok {
				totalMessages += sent
			}
			if recv, ok := client["messages_recv"].(int64); ok {
				totalMessages += recv
			}
		}

		response := map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"total_connections": len(clients),
				"total_pings":       totalPings,
				"total_pongs":       totalPongs,
				"total_messages":    totalMessages,
				"clients":           clients,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
