package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"wx_channel/hub_server/database"
	"wx_channel/hub_server/middleware"
	"wx_channel/hub_server/models"
	"wx_channel/hub_server/ws"
)

func GetTasks(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	nodeID := r.URL.Query().Get("node_id")

	userID := r.Context().Value(middleware.ContextKeyUserID).(uint)

	if limit <= 0 {
		limit = 20
	}

	tasks, count, err := database.GetTasks(userID, nodeID, offset, limit)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total": count,
		"list":  tasks,
	})
}

func GetTaskDetail(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", 400)
		return
	}

	userID := r.Context().Value(middleware.ContextKeyUserID).(uint)
	task, err := database.GetTaskByID(uint(id), userID)
	if err != nil {
		http.Error(w, "Task not found", 404)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func RemoteCall(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ClientID string          `json:"client_id"`
			Action   string          `json:"action"`
			Data     json.RawMessage `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":    -1,
				"message": err.Error(),
			})
			return
		}

		// Check Credits
		userID := r.Context().Value(middleware.ContextKeyUserID).(uint)
		cost := int64(0)

		switch req.Action {
		case "search_channels", "search_videos":
			cost = 1
		case "download_video":
			cost = 10
		case "api_call":
			// Check specific API calls for browsing cost
			var apiData struct {
				Key string `json:"key"`
			}
			if err := json.Unmarshal(req.Data, &apiData); err == nil {
				switch apiData.Key {
				case "key:channels:feed_profile": // Video Detail
					cost = 1
				case "key:channels:feed_list": // User Profile / Channel Feed
					cost = 1
				}
			}
		}

		if cost > 0 {
			user, err := database.GetUserByID(userID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    -1,
					"message": "User not found",
				})
				return
			}

			if user.Credits < cost {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    -1,
					"message": "Insufficient credits",
				})
				return
			}

			// Deduct credits
			if err := database.AddCredits(userID, -cost); err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    -1,
					"message": "Transaction failed",
				})
				return
			}

			// Record Transaction
			database.RecordTransaction(&models.Transaction{
				UserID:      userID,
				Amount:      -cost,
				Type:        req.Action,
				Description: "API Call: " + req.Action,
				RelatedID:   req.ClientID,
				CreatedAt:   time.Now(),
			})
		}

		// Auto-detect online client if not provided
		clientID := req.ClientID
		if clientID == "" {
			user, err := database.GetUserByID(userID)
			if err != nil || len(user.Devices) == 0 {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    -1,
					"message": "No device found",
				})
				return
			}

			// Find first online device
			for _, device := range user.Devices {
				if device.Status == "online" {
					clientID = device.ID
					break
				}
			}

			if clientID == "" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    -1,
					"message": "No online device found",
				})
				return
			}
		}

		// 根据不同的操作设置不同的超时时间
		timeout := 2 * time.Minute // 默认 2 分钟（从 30 秒增加）
		switch req.Action {
		case "search_channels", "search_videos":
			timeout = 3 * time.Minute // 搜索操作最多 3 分钟
		case "download_video":
			timeout = 10 * time.Minute // 下载操作最多 10 分钟
		case "api_call":
			// api_call 使用 2 分钟超时
			timeout = 2 * time.Minute
		case "get_profile", "get_channel_info", "get_video_info":
			// 获取信息类操作使用 1 分钟超时
			timeout = 1 * time.Minute
		}

		resp, err := hub.Call(userID, clientID, req.Action, req.Data, timeout)
		if err != nil {
			// 调用失败，退还已扣积分
			if cost > 0 {
				if refundErr := database.AddCredits(userID, cost); refundErr != nil {
					log.Printf("[RemoteCall] 退还积分失败 userID=%d cost=%d: %v", userID, cost, refundErr)
				}
			}
			// Return JSON error instead of plain text
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":    -1,
				"message": err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
