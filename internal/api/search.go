package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"wx_channel/internal/response"
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

// SearchContactRequest 搜索账号请求参数
type SearchContactRequest struct {
	Keyword  string `json:"keyword"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

// SearchContact 搜索账号
func (s *SearchService) SearchContact(w http.ResponseWriter, r *http.Request) {
	var req SearchContactRequest

	// 支持 GET 和 POST
	if r.Method == http.MethodGet {
		req.Keyword = r.URL.Query().Get("keyword")
		req.Page, _ = strconv.Atoi(r.URL.Query().Get("page"))
		req.PageSize, _ = strconv.Atoi(r.URL.Query().Get("page_size"))
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

	// 默认分页
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 50 {
		req.PageSize = 20
	}

	// 调用前端 API
	body := websocket.SearchContactBody{
		Keyword: req.Keyword,
	}

	data, err := s.hub.CallAPI("key:channels:contact_list", body, 60*time.Second)
	if err != nil {
		if strings.Contains(err.Error(), "no available client") {
			response.ErrorWithStatus(w, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "WeChat client not connected. Please open the target page.")
			return
		}
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 解析返回数据以支持分页
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		response.Success(w, json.RawMessage(data))
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
