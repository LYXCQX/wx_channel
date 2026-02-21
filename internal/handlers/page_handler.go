package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wx_channel/internal/response"
	"wx_channel/internal/utils"

	"github.com/fatih/color"
	"github.com/qtgolang/SunnyNet/SunnyNet"
)

// HandlePageURL 处理页面URL请求
func (h *APIHandler) HandlePageURL(Conn *SunnyNet.HttpConn) bool {
	// 如果是页面请求，记录URL
	// 匹配 fetch_feed 等页面加载请求
	// 或者主页面 url
	path := Conn.Request.URL.Path
	if path == "/__wx_channels_api/page_url" {
		var data struct {
			URL string `json:"url"`
		}
		body, err := io.ReadAll(Conn.Request.Body)
		if err == nil {
			// 忽略错误，因为不仅仅依靠这个
			json.Unmarshal(body, &data)
		}
		Conn.Request.Body.Close()

		if data.URL != "" {
			h.SetCurrentURL(data.URL)
			utils.LogInfo("[页面访问] URL=%s", data.URL)
		}

		// 返回空响应
		h.sendEmptyResponse(Conn)
		return true
	}

	return false
}

// HandleSavePageContent 处理页面内容保存请求
func (h *APIHandler) HandleSavePageContent(Conn *SunnyNet.HttpConn) bool {
	path := Conn.Request.URL.Path
	if path != "/__wx_channels_api/save_page_content" {
		return false
	}

	// 提前检查配置，如果功能未启用则直接返回成功，避免不必要的处理
	cfg := h.getConfig()
	if cfg == nil || !cfg.SavePageSnapshot {
		// 功能未启用，直接返回成功，不做任何处理
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")
		headers.Set("__debug", "fake_resp")
		Conn.StopRequest(200, `{"code":0,"message":"页面快照功能未启用"}`, headers)
		return true
	}

	var contentData struct {
		URL       string `json:"url"`
		HTML      string `json:"html"`
		Timestamp int64  `json:"timestamp"`
	}
	body, err := io.ReadAll(Conn.Request.Body)
	if err != nil {
		utils.HandleError(err, "读取save_page_content请求体")
		return true
	}
	if err := Conn.Request.Body.Close(); err != nil {
		utils.HandleError(err, "关闭请求体")
	}

	// 检查请求体是否为空
	if len(body) == 0 {
		utils.Warn("save_page_content 请求体为空")
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")
		headers.Set("__debug", "fake_resp")
		Conn.StopRequest(400, `{"code":-1,"message":"请求体为空"}`, headers)
		return true
	}

	// 记录请求体大小（用于调试）
	utils.LogInfo("save_page_content 请求体大小: %d 字节", len(body))

	err = json.Unmarshal(body, &contentData)
	if err != nil {
		// 记录更详细的错误信息
		utils.LogError("解析页面内容数据失败: %v, 请求体前100字节: %s", err, string(body[:min(100, len(body))]))
		utils.HandleError(err, "解析页面内容数据")

		// 返回错误响应
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")
		headers.Set("__debug", "fake_resp")
		Conn.StopRequest(400, fmt.Sprintf(`{"code":-1,"message":"JSON解析失败: %s"}`, err.Error()), headers)
		return true
	}

	// 解析成功，保存页面内容
	parsedURL, err := url.Parse(contentData.URL)
	if err != nil {
		utils.HandleError(err, "解析页面内容URL")
	} else {
		h.saveDynamicHTML(contentData.HTML, parsedURL, contentData.URL, contentData.Timestamp)
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("__debug", "fake_resp")
	Conn.StopRequest(200, string(response.SuccessJSON(nil)), headers)
	return true
}

// saveDynamicHTML 保存动态页面的完整HTML内容
func (h *APIHandler) saveDynamicHTML(htmlContent string, parsedURL *url.URL, fullURL string, timestamp int64) {
	cfg := h.getConfig()
	if cfg == nil {
		utils.Warn("配置未初始化，无法保存页面内容: %s", fullURL)
		return
	}
	if !cfg.SavePageSnapshot {
		return
	}
	if htmlContent == "" || parsedURL == nil {
		return
	}

	if cfg.SaveDelay > 0 {
		time.Sleep(cfg.SaveDelay)
	}

	saveTime := time.Now()
	if timestamp > 0 {
		saveTime = time.Unix(0, timestamp*int64(time.Millisecond))
	}

	downloadsDir, err := utils.ResolveDownloadDir(cfg.DownloadsDir)
	if err != nil {
		utils.HandleError(err, "解析下载目录用于保存页面内容")
		return
	}

	if err := utils.EnsureDir(downloadsDir); err != nil {
		utils.HandleError(err, "创建下载目录用于保存页面内容")
		return
	}

	pagesRoot := filepath.Join(downloadsDir, "page_snapshots")
	if err := utils.EnsureDir(pagesRoot); err != nil {
		utils.HandleError(err, "创建页面保存根目录")
		return
	}

	dateDir := filepath.Join(pagesRoot, saveTime.Format("2006-01-02"))
	if err := utils.EnsureDir(dateDir); err != nil {
		utils.HandleError(err, "创建页面保存日期目录")
		return
	}

	var filenameParts []string
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		segments := strings.Split(parsedURL.Path, "/")
		for _, segment := range segments {
			segment = strings.TrimSpace(segment)
			if segment == "" || segment == "." {
				continue
			}
			filenameParts = append(filenameParts, utils.CleanFilename(segment))
		}
	}

	if parsedURL.RawQuery != "" {
		querySegment := strings.ReplaceAll(parsedURL.RawQuery, "&", "_")
		querySegment = strings.ReplaceAll(querySegment, "=", "-")
		querySegment = utils.CleanFilename(querySegment)
		if querySegment != "" {
			filenameParts = append(filenameParts, querySegment)
		}
	}

	if len(filenameParts) == 0 {
		filenameParts = append(filenameParts, "page")
	}

	baseName := strings.Join(filenameParts, "_")
	fileName := fmt.Sprintf("%s_%s.html", saveTime.Format("150405"), baseName)
	targetPath := utils.GenerateUniqueFilename(dateDir, fileName, 100)

	if err := os.WriteFile(targetPath, []byte(htmlContent), 0644); err != nil {
		utils.HandleError(err, "保存页面HTML内容")
		return
	}

	metaData := map[string]interface{}{
		"url":       fullURL,
		"host":      parsedURL.Host,
		"path":      parsedURL.Path,
		"query":     parsedURL.RawQuery,
		"saved_at":  saveTime.Format(time.RFC3339),
		"timestamp": timestamp,
	}

	metaBytes, err := json.MarshalIndent(metaData, "", "  ")
	if err == nil {
		metaPath := strings.TrimSuffix(targetPath, filepath.Ext(targetPath)) + ".meta.json"
		if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
			utils.HandleError(err, "保存页面元数据")
		}
	}

	utils.LogInfo("[页面快照] 已保存: %s", targetPath)

	utils.PrintSeparator()
	color.Blue("💾 页面快照已保存")
	utils.PrintSeparator()
	utils.PrintLabelValue("📁", "保存路径", targetPath)
	utils.PrintLabelValue("🔗", "页面链接", fullURL)
	utils.PrintSeparator()
	fmt.Println()
	fmt.Println()
}

// HandleStaticFiles 处理静态文件请求（jszip, FileSaver等）
func HandleStaticFiles(Conn *SunnyNet.HttpConn, zipJS, fileSaverJS []byte) bool {
	path := Conn.Request.URL.Path

	// 处理 jszip.min.js 请求
	if strings.HasSuffix(path, "/jszip.min.js") {
		headers := http.Header{}
		headers.Set("Content-Type", "application/javascript")
		headers.Set("Cache-Control", "public, max-age=31536000")
		headers.Set("Access-Control-Allow-Origin", "*") // 允许跨域

		Conn.StopRequest(200, string(zipJS), headers)
		return true
	}

	// 处理 FileSaver.min.js 请求
	if strings.HasSuffix(path, "/FileSaver.min.js") {
		headers := http.Header{}
		headers.Set("Content-Type", "application/javascript")
		headers.Set("Cache-Control", "public, max-age=31536000")
		headers.Set("Access-Control-Allow-Origin", "*") // 允许跨域

		Conn.StopRequest(200, string(fileSaverJS), headers)
		return true
	}

	return false
}
