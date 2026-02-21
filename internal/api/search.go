package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"wx_channel/internal/cloud"
	"wx_channel/internal/response"
	"wx_channel/internal/utils"
	"wx_channel/internal/websocket"
)

// SearchService 搜索服务
type SearchService struct {
	hub *websocket.Hub
}

// NewSearchService 创建搜索服务
func NewSearchService(hub *websocket.Hub) *SearchService {
	return &SearchService{hub: hub}
}

// SearchContactRequest 搜索请求参数
type SearchContactRequest struct {
	Keyword    string `json:"keyword"`
	Type       int    `json:"type"` // 搜索类型：1=账号, 2=直播, 3=视频
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	NextMarker string `json:"next_marker"` // 分页标记
}

// SearchContact 统一搜索接口（支持账号、直播、视频）
func (s *SearchService) SearchContact(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr

	var req SearchContactRequest

	// 支持 GET 和 POST
	if r.Method == http.MethodGet {
		req.Keyword = r.URL.Query().Get("keyword")
		req.Type, _ = strconv.Atoi(r.URL.Query().Get("type"))
		req.Page, _ = strconv.Atoi(r.URL.Query().Get("page"))
		req.PageSize, _ = strconv.Atoi(r.URL.Query().Get("page_size"))
		req.NextMarker = r.URL.Query().Get("next_marker")
	} else if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, 400, "Invalid request body")
			return
		}
	}

	// 参数校验
	if req.Keyword == "" {
		response.Error(w, 400, "keyword is required")
		return
	}
	if len(req.Keyword) > 100 {
		response.Error(w, 400, "keyword too long (max 100 characters)")
		return
	}

	// 默认值和范围检查
	originalType := req.Type
	if req.Type == 0 {
		req.Type = 1 // 默认搜索账号（向后兼容）
	}
	if req.Type < 1 || req.Type > 3 {
		utils.LogWarn("API 请求参数错误: %s from %s, type值无效:%d", r.URL.Path, clientIP, originalType)
		response.Error(w, 400, "type must be 1 (account), 2 (live), or 3 (video)")
		return
	}

	// 默认分页
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 50 {
		req.PageSize = 20
	}

	// 调用前端 API
	body := websocket.SearchContactBody{
		Keyword:    req.Keyword,
		Type:       req.Type, // 传递搜索类型
		NextMarker: req.NextMarker, // 传递分页标记
	}

	apiCallStart := time.Now()

	// 使用与 api/call 相同的超时时间（3分钟）
	data, err := s.hub.CallAPI("key:channels:contact_list", body, 3*time.Minute)

	apiCallDuration := time.Since(apiCallStart)

	if err != nil {
		utils.LogError("WebSocket API 调用失败: %s, keyword=%s, type=%d, duration=%v, error=%v",
			r.URL.Path, req.Keyword, req.Type, apiCallDuration, err)

		if strings.Contains(err.Error(), "no available client") {
			utils.LogWarn("WebSocket 客户端未连接: %s", r.URL.Path)
			response.ErrorWithStatus(w, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "WeChat client not connected. Please open the target page.")
			return
		}
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 转换响应数据为统一格式（与 api/call 使用完全相同的逻辑）
	bodyJSON, _ := json.Marshal(body)
	transformedData := cloud.TransformSearchResponse(bodyJSON, data)
	
	utils.LogInfo("数据转换完成，转换后数据大小: %d bytes", len(transformedData))
	
	// 直接返回转换后的数据（与 api/call 保持一致）
	var result interface{}
	if err := json.Unmarshal(transformedData, &result); err != nil {
		response.Success(w, json.RawMessage(transformedData))
		return
	}

	response.Success(w, result)
}

// GetFeedListRequest 获取视频列表请求参数
type GetFeedListRequest struct {
	Username   string `json:"username"`
	NextMarker string `json:"next_marker"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
}

// GetFeedList 获取账号的视频列表
func (s *SearchService) GetFeedList(w http.ResponseWriter, r *http.Request) {
	var req GetFeedListRequest

	if r.Method == http.MethodGet {
		req.Username = r.URL.Query().Get("username")
		req.NextMarker = r.URL.Query().Get("next_marker")
		req.Page, _ = strconv.Atoi(r.URL.Query().Get("page"))
		req.PageSize, _ = strconv.Atoi(r.URL.Query().Get("page_size"))
	} else if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, 400, "Invalid request body")
			return
		}
	}

	if req.Username == "" {
		response.Error(w, 400, "username is required")
		return
	}

	// 默认分页
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 50 {
		req.PageSize = 20
	}

	// 调用前端 API
	body := websocket.FeedListBody{
		Username:   req.Username,
		NextMarker: req.NextMarker,
	}

	data, err := s.hub.CallAPI("key:channels:feed_list", body, 60*time.Second)
	if err != nil {
		if strings.Contains(err.Error(), "no available client") {
			response.ErrorWithStatus(w, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "WeChat client not connected")
			return
		}
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		response.Success(w, json.RawMessage(data))
		return
	}

	response.Success(w, result)
}

// GetFeedProfileRequest 获取视频详情请求参数
type GetFeedProfileRequest struct {
	ObjectID string `json:"object_id"`
	NonceID  string `json:"nonce_id"`
	URL      string `json:"url"`
}

// GetFeedProfile 获取视频详情
func (s *SearchService) GetFeedProfile(w http.ResponseWriter, r *http.Request) {
	var req GetFeedProfileRequest

	if r.Method == http.MethodGet {
		req.ObjectID = r.URL.Query().Get("object_id")
		req.NonceID = r.URL.Query().Get("nonce_id")
		req.URL = r.URL.Query().Get("url")
	} else if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, 400, "Invalid request body")
			return
		}
	}

	if req.ObjectID == "" && req.URL == "" {
		response.Error(w, 400, "object_id or url is required")
		return
	}

	// 调用前端 API
	body := websocket.FeedProfileBody{
		ObjectID: req.ObjectID,
		NonceID:  req.NonceID,
		URL:      req.URL,
	}

	data, err := s.hub.CallAPI("key:channels:feed_profile", body, 60*time.Second)
	if err != nil {
		if strings.Contains(err.Error(), "no available client") {
			response.ErrorWithStatus(w, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "WeChat client not connected")
			return
		}
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		response.Success(w, json.RawMessage(data))
		return
	}

	response.Success(w, result)
}

// GetStatus 获取 WebSocket 连接状态
func (s *SearchService) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"connected": s.hub.ClientCount() > 0,
		"clients":   s.hub.ClientCount(),
	}
	response.Success(w, status)
}

// RegisterRoutes 注册搜索相关路由
func (s *SearchService) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/search/contact", s.SearchContact)
	mux.HandleFunc("/api/v1/search/feed", s.GetFeedList)
	mux.HandleFunc("/api/v1/search/feed/profile", s.GetFeedProfile)
	mux.HandleFunc("/api/v1/status", s.GetStatus)

	// 兼容旧路由
	mux.HandleFunc("/api/search/contact", s.SearchContact)
	mux.HandleFunc("/api/search/feed", s.GetFeedList)
	mux.HandleFunc("/api/search/feed/profile", s.GetFeedProfile)
	mux.HandleFunc("/api/status", s.GetStatus)

	// 兼容 /api/channels 路由 (WebSocket服务器原有的路由)
	mux.HandleFunc("/api/channels/contact/search", s.SearchContact)
	mux.HandleFunc("/api/channels/contact/feed/list", s.GetFeedList)
	mux.HandleFunc("/api/channels/feed/profile", s.GetFeedProfile)
	mux.HandleFunc("/api/channels/status", s.GetStatus)
}
