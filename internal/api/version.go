package api

import (
	"net/http"

	"wx_channel/internal/response"
	"wx_channel/internal/services"
)

// VersionAPI 处理版本相关请求
type VersionAPI struct {
	service *services.VersionService
}

// NewVersionAPI 创建一个新的 VersionAPI
func NewVersionAPI() *VersionAPI {
	return &VersionAPI{
		service: services.NewVersionService(),
	}
}

// CheckUpdate 检查应用更新
func (api *VersionAPI) CheckUpdate(w http.ResponseWriter, r *http.Request) {
	result, err := api.service.CheckUpdate()
	if err != nil {
		response.Error(w, 500, err.Error())
		return
	}

	response.Success(w, result)
}

// RegisterRoutes 注册版本路由
func (api *VersionAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/system/version/check", api.CheckUpdate)
}
