package controllers

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	hubutils "wx_channel/hub_server/utils"
)

// mediaClient 复用的 HTTP 客户端（流式代理使用无超时，但有连接超时）
var mediaClient = &http.Client{
	Timeout: 0, // 流式传输不设整体超时
	Transport: &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     90 * time.Second,
	},
}

// isAllowedURL 检查 URL 是否为允许的外部地址（防止 SSRF）
func isAllowedURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// 只允许 http/https
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	host := u.Hostname()

	// 阻止访问内网地址
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return false
		}
	}

	// 阻止 localhost
	if strings.EqualFold(host, "localhost") {
		return false
	}

	return true
}

func PlayVideo(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	fmt.Printf("[PlayVideo] Target URL: %s\n", targetURL)
	if targetURL == "" {
		fmt.Println("[PlayVideo] Error: url parameter required")
		http.Error(w, "url parameter required", http.StatusBadRequest)
		return
	}

	// SSRF 防护：检查 URL 合法性
	if !isAllowedURL(targetURL) {
		http.Error(w, "forbidden URL", http.StatusForbidden)
		return
	}

	// 获取可选的解密密钥
	decryptKeyStr := r.URL.Query().Get("key")
	fmt.Printf("[PlayVideo] Decrypt key string: %s\n", decryptKeyStr)
	var decryptKey uint64
	var needsDecryption bool

	if decryptKeyStr != "" {
		var err error
		decryptKey, err = strconv.ParseUint(decryptKeyStr, 10, 64)
		if err != nil {
			fmt.Printf("[PlayVideo] Error: failed to parse decrypt key: %v\n", err)
			http.Error(w, "invalid decryption key", http.StatusBadRequest)
			return
		}
		needsDecryption = true
		fmt.Printf("[PlayVideo] Decrypt key parsed: %d\n", decryptKey)
	}

	// 创建上游请求
	req, err := http.NewRequest(r.Method, targetURL, nil)
	if err != nil {
		fmt.Printf("[PlayVideo] Error: failed to create request: %v\n", err)
		http.Error(w, "invalid URL", http.StatusBadRequest)
		return
	}

	// 复制 Range 头（支持视频拖动）
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	// 使用复用的 http.Client
	resp, err := mediaClient.Do(req)
	if err != nil {
		http.Error(w, "failed to fetch video", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range")

	// 复制响应头
	for k, v := range resp.Header {
		w.Header()[k] = v
	}

	// 确保设置 Accept-Ranges
	if w.Header().Get("Accept-Ranges") == "" {
		w.Header().Set("Accept-Ranges", "bytes")
	}

	// 如果需要解密
	if needsDecryption {
		// 解析 Content-Range 头以获取起始偏移
		var startOffset uint64 = 0
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			// Content-Range 格式: "bytes start-end/total"
			parts := strings.Split(cr, " ")
			if len(parts) == 2 {
				rangePart := parts[1]
				dashIdx := strings.Index(rangePart, "-")
				if dashIdx > 0 {
					if v, err := strconv.ParseUint(rangePart[:dashIdx], 10, 64); err == nil {
						startOffset = v
					}
				}
			}
		}

		// 创建解密读取器
		// 加密区域大小为 131072 字节（128KB）
		decryptReader := hubutils.NewDecryptReader(resp.Body, decryptKey, startOffset, 131072)

		// 写入状态码
		w.WriteHeader(resp.StatusCode)

		// 如果是 HEAD 请求，不传输内容
		if r.Method == "HEAD" {
			return
		}

		// 流式复制解密后的数据到客户端
		io.Copy(w, decryptReader)
	} else {
		// 无需解密，直接代理
		w.WriteHeader(resp.StatusCode)

		// 如果是 HEAD 请求，不传输内容
		if r.Method == "HEAD" {
			return
		}

		// 流式复制数据到客户端
		io.Copy(w, resp.Body)
	}
}
