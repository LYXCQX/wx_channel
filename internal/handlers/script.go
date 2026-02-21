package handlers

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wx_channel/internal/config"
	"wx_channel/internal/utils"

	"wx_channel/pkg/util"

	"github.com/qtgolang/SunnyNet/SunnyNet"
	sunnyPublic "github.com/qtgolang/SunnyNet/public"
)

// ScriptHandler JavaScript注入处理器
type ScriptHandler struct {
	coreJS          []byte
	decryptJS       []byte
	downloadJS      []byte
	homeJS          []byte
	feedJS          []byte
	profileJS       []byte
	searchJS        []byte
	batchDownloadJS []byte
	zipJS           []byte
	fileSaverJS     []byte
	mittJS          []byte
	eventbusJS      []byte
	utilsJS         []byte
	apiClientJS     []byte
	keepAliveJS     []byte
	version         string
}

// NewScriptHandler 创建脚本处理器
func NewScriptHandler(cfg *config.Config, coreJS, decryptJS, downloadJS, homeJS, feedJS, profileJS, searchJS, batchDownloadJS, zipJS, fileSaverJS, mittJS, eventbusJS, utilsJS, apiClientJS, keepAliveJS []byte, version string) *ScriptHandler {
	return &ScriptHandler{
		coreJS:          coreJS,
		decryptJS:       decryptJS,
		downloadJS:      downloadJS,
		homeJS:          homeJS,
		feedJS:          feedJS,
		profileJS:       profileJS,
		searchJS:        searchJS,
		batchDownloadJS: batchDownloadJS,
		zipJS:           zipJS,
		fileSaverJS:     fileSaverJS,
		mittJS:          mittJS,
		eventbusJS:      eventbusJS,
		utilsJS:         utilsJS,
		apiClientJS:     apiClientJS,
		keepAliveJS:     keepAliveJS,
		version:         version,
	}
}

// getConfig 获取当前配置（动态获取最新配置）
func (h *ScriptHandler) getConfig() *config.Config {
	return config.Get()
}

// Handle implements router.Interceptor
func (h *ScriptHandler) Handle(Conn *SunnyNet.HttpConn) bool {

	if Conn.Type != sunnyPublic.HttpResponseOK {
		return false
	}

	// 防御性检查
	if Conn.Request == nil || Conn.Request.URL == nil {
		return false
	}

	// 只有响应成功且有内容才处理
	if Conn.Response == nil || Conn.Response.Body == nil {
		return false
	}

	// 读取响应体
	// 注意：这里读取了Body，如果未被修改，需要重新赋值回去
	body, err := io.ReadAll(Conn.Response.Body)
	if err != nil {
		return false
	}
	_ = Conn.Response.Body.Close()

	host := Conn.Request.URL.Hostname()
	path := Conn.Request.URL.Path

	// 记录所有JS文件的加载（简略日志）
	if strings.HasSuffix(path, ".js") {
		contentType := strings.ToLower(Conn.Response.Header.Get("content-type"))
		utils.LogFileInfo("[响应] Path=%s | ContentType=%s", path, contentType)
	}

	if h.HandleHTMLResponse(Conn, host, path, body) {
		return true
	}

	if h.HandleJavaScriptResponse(Conn, host, path, body) {
		return true
	}

	// 如果没有处理，恢复Body
	Conn.Response.Body = io.NopCloser(bytes.NewBuffer(body))
	return false
}

// HandleHTMLResponse 处理HTML响应，注入JavaScript代码
func (h *ScriptHandler) HandleHTMLResponse(Conn *SunnyNet.HttpConn, host, path string, body []byte) bool {
	contentType := strings.ToLower(Conn.Response.Header.Get("content-type"))
	if contentType != "text/html; charset=utf-8" {
		return false
	}

	html := string(body)

	// 添加版本号到JS引用
	scriptReg1 := regexp.MustCompile(`src="([^"]{1,})\.js"`)
	html = scriptReg1.ReplaceAllString(html, `src="$1.js`+h.version+`"`)
	scriptReg2 := regexp.MustCompile(`href="([^"]{1,})\.js"`)
	html = scriptReg2.ReplaceAllString(html, `href="$1.js`+h.version+`"`)
	Conn.Response.Header.Set("__debug", "append_script")

	if host == "channels.weixin.qq.com" && (path == "/web/pages/feed" || path == "/web/pages/home" || path == "/web/pages/profile" || path == "/web/pages/s") {
		// 根据页面路径注入不同的脚本
		injectedScripts := h.buildInjectedScripts(path)
		html = strings.Replace(html, "<head>", "<head>\n"+injectedScripts, 1)
		utils.LogFileInfo("页面已成功加载！")
		utils.LogFileInfo("已添加视频缓存监控和提醒功能")
		utils.LogFileInfo("[页面加载] 视频号页面已加载 | Host=%s | Path=%s", host, path)
		Conn.Response.Body = io.NopCloser(bytes.NewBuffer([]byte(html)))
		return true
	}

	Conn.Response.Body = io.NopCloser(bytes.NewBuffer([]byte(html)))
	return true
}

// HandleJavaScriptResponse 处理JavaScript响应，修改JavaScript代码
func (h *ScriptHandler) HandleJavaScriptResponse(Conn *SunnyNet.HttpConn, host, path string, body []byte) bool {
	contentType := strings.ToLower(Conn.Response.Header.Get("content-type"))
	if contentType != "application/javascript" {
		return false
	}

	// 记录所有JS文件的加载（用于调试）
	utils.LogFileInfo("[JS文件] %s", path)

	// 保存关键的 JS 文件到本地以便分析
	h.saveJavaScriptFile(path, body)

	content := string(body)

	// 添加版本号到JS引用
	depReg := regexp.MustCompile(`"js/([^"]{1,})\.js"`)
	fromReg := regexp.MustCompile(`from {0,1}"([^"]{1,})\.js"`)
	lazyImportReg := regexp.MustCompile(`import\("([^"]{1,})\.js"\)`)
	importReg := regexp.MustCompile(`import {0,1}"([^"]{1,})\.js"`)
	content = fromReg.ReplaceAllString(content, `from"$1.js`+h.version+`"`)
	content = depReg.ReplaceAllString(content, `"js/$1.js`+h.version+`"`)
	content = lazyImportReg.ReplaceAllString(content, `import("$1.js`+h.version+`")`)
	content = importReg.ReplaceAllString(content, `import"$1.js`+h.version+`"`)
	Conn.Response.Header.Set("__debug", "replace_script")

	// 处理不同的JS文件
	content, handled := h.handleIndexPublish(path, content)
	if handled {
		Conn.Response.Body = io.NopCloser(bytes.NewBuffer([]byte(content)))
		return true
	}
	content, handled = h.handleVirtualSvgIcons(path, content)
	if handled {
		Conn.Response.Body = io.NopCloser(bytes.NewBuffer([]byte(content)))
		return true
	}

	content, handled = h.handleWorkerRelease(path, content)
	if handled {
		Conn.Response.Body = io.NopCloser(bytes.NewBuffer([]byte(content)))
		return true
	}
	content, handled = h.handleConnectPublish(Conn, path, content)
	if handled {
		return true
	}

	Conn.Response.Body = io.NopCloser(bytes.NewBuffer([]byte(content)))
	return true
}

// buildInjectedScripts 构建所有需要注入的脚本（根据页面路径注入不同脚本）
func (h *ScriptHandler) buildInjectedScripts(path string) string {
	// 日志面板脚本（必须在最前面，以便拦截所有console输出）- 所有页面都需要
	logPanelScript := h.getLogPanelScript()

	// 事件系统脚本（mitt + eventbus + utils）- 必须在主脚本之前加载
	mittScript := fmt.Sprintf(`<script>%s</script>`, string(h.mittJS))
	eventbusScript := fmt.Sprintf(`<script>%s</script>`, string(h.eventbusJS))
	utilsScript := fmt.Sprintf(`<script>%s</script>`, string(h.utilsJS))

	// API 客户端脚本 - 必须在其他脚本之前加载
	apiClientScript := fmt.Sprintf(`<script>%s</script>`, string(h.apiClientJS))

	// 页面保活脚本 - 防止页面休眠
	keepAliveScript := fmt.Sprintf(`<script>%s</script>`, string(h.keepAliveJS))

	// 模块化脚本 - 按依赖顺序加载
	coreScript := fmt.Sprintf(`<script>%s</script>`, string(h.coreJS))
	decryptScript := fmt.Sprintf(`<script>%s</script>`, string(h.decryptJS))
	downloadScript := fmt.Sprintf(`<script>%s</script>`, string(h.downloadJS))
	batchDownloadScript := fmt.Sprintf(`<script>%s</script>`, string(h.batchDownloadJS))
	feedScript := fmt.Sprintf(`<script>%s</script>`, string(h.feedJS))
	profileScript := fmt.Sprintf(`<script>%s</script>`, string(h.profileJS))
	searchScript := fmt.Sprintf(`<script>%s</script>`, string(h.searchJS))
	homeScript := fmt.Sprintf(`<script>%s</script>`, string(h.homeJS))

	// 预加载FileSaver.js库 - 所有页面都需要
	preloadScript := h.getPreloadScript()

	// 下载记录功能 - 所有页面都需要
	downloadTrackerScript := h.getDownloadTrackerScript()

	// 捕获URL脚本 - 所有页面都需要
	captureUrlScript := h.getCaptureUrlScript()

	// 保存页面内容脚本 - 所有页面都需要（用于保存快照）
	savePageContentScript := h.getSavePageContentScript()

	// 基础脚本（所有页面都需要）
	baseScripts := logPanelScript + mittScript + eventbusScript + utilsScript + apiClientScript + keepAliveScript + coreScript + decryptScript + downloadScript + batchDownloadScript + feedScript + profileScript + searchScript + homeScript + preloadScript + downloadTrackerScript + captureUrlScript + savePageContentScript

	// 根据页面路径决定是否注入特定脚本
	var pageSpecificScripts string

	switch path {
	case "/web/pages/home":
		// Home页面：注入视频缓存监控脚本
		pageSpecificScripts = h.getVideoCacheNotificationScript()
		utils.LogFileInfo("[脚本注入] Home页面 - 注入事件系统和视频缓存监控脚本")

	case "/web/pages/profile":
		// Profile页面（视频列表）：不需要特定脚本
		pageSpecificScripts = ""
		utils.LogFileInfo("[脚本注入] Profile页面 - 仅注入基础脚本")

	case "/web/pages/feed":
		// Feed页面（视频详情）：注入视频缓存监控和评论采集脚本
		pageSpecificScripts = h.getVideoCacheNotificationScript() + h.getCommentCaptureScript()
		utils.LogFileInfo("[脚本注入] Feed页面 - 注入视频缓存监控和评论采集脚本")

	case "/web/pages/s":
		// 搜索页面：注入搜索模块
		pageSpecificScripts = searchScript
		utils.LogInfo("[脚本注入] 搜索页面 - 注入搜索模块（事件系统）")

	default:
		// 其他页面：不注入页面特定脚本
		pageSpecificScripts = ""
		utils.LogInfo("[脚本注入] 其他页面 - 仅注入基础脚本")
	}

	// 初始化脚本（延迟执行）
	initScript := `<script>
console.log('[init] 开始初始化...');
setTimeout(function() {
	console.log('[init] 执行 insert_download_btn');
	if (typeof insert_download_btn === 'function') {
		insert_download_btn();
	} else {
		console.error('[init] insert_download_btn 函数未定义');
	}
}, 800);
</script>`

	return baseScripts + pageSpecificScripts + initScript
}

// getPreloadScript 获取预加载FileSaver.js库的脚本
func (h *ScriptHandler) getPreloadScript() string {
	return `<script>
	// 预加载FileSaver.js库
	(function() {
		const script = document.createElement('script');
		script.src = '/FileSaver.min.js';
		document.head.appendChild(script);
	})();
	</script>`
}

// getDownloadTrackerScript 获取下载记录功能的脚本
func (h *ScriptHandler) getDownloadTrackerScript() string {
	return `<script>
	// 确保FileSaver.js库已加载
	if (typeof saveAs === 'undefined') {
		console.log('加载FileSaver.js库');
		const script = document.createElement('script');
		script.src = '/FileSaver.min.js';
		script.onload = function() {
			console.log('FileSaver.js库加载成功');
		};
		document.head.appendChild(script);
	}

	// 跟踪已记录的下载，防止重复记录
	window.__wx_channels_recorded_downloads = {};

	// 添加下载记录功能
	window.__wx_channels_record_download = function(data) {
		// 检查是否已经记录过这个下载
		const recordKey = data.id;
		if (window.__wx_channels_recorded_downloads[recordKey]) {
			console.log("已经记录过此下载，跳过记录");
			return;
		}
		
		// 标记为已记录
		window.__wx_channels_recorded_downloads[recordKey] = true;
		
		// 发送到记录API
		fetch("/__wx_channels_api/record_download", {
			method: "POST",
			headers: {
				"Content-Type": "application/json"
			},
			body: JSON.stringify(data)
		});
	};
	
	// 暂停视频的辅助函数（只暂停，不阻止自动切换）
	window.__wx_channels_pause_video__ = function() {
		console.log('[视频助手] 暂停视频（下载期间）...');
		try {
			let pausedCount = 0;
			const pausedVideos = [];
			
			// 方法1: 使用 Video.js API
			if (typeof videojs !== 'undefined') {
				const players = videojs.getAllPlayers?.() || [];
				players.forEach((player, index) => {
					if (player && typeof player.pause === 'function' && !player.paused()) {
						player.pause();
						pausedVideos.push({ type: 'videojs', player, index });
						pausedCount++;
						console.log('[视频助手] Video.js 播放器', index, '已暂停');
					}
				});
			}
			
			// 方法2: 查找所有 video 元素
			const videos = document.querySelectorAll('video');
			videos.forEach((video, index) => {
				// 尝试通过 Video.js 获取播放器实例
				let player = null;
				if (typeof videojs !== 'undefined') {
					try {
						player = videojs(video);
					} catch (e) {
						// 不是 Video.js 播放器
					}
				}
				
				if (player && typeof player.pause === 'function') {
					if (!player.paused()) {
						player.pause();
						pausedVideos.push({ type: 'videojs', player, index });
						pausedCount++;
						console.log('[视频助手] Video.js 播放器', index, '已暂停');
					}
				} else {
					if (!video.paused) {
						video.pause();
						pausedVideos.push({ type: 'native', video, index });
						pausedCount++;
						console.log('[视频助手] 原生视频', index, '已暂停');
					}
				}
			});
			
			console.log('[视频助手] 共暂停', pausedCount, '个视频');
			
			// 返回暂停的视频列表，用于后续恢复
			return pausedVideos;
		} catch (e) {
			console.error('[视频助手] 暂停视频失败:', e);
			return [];
		}
	};
	
	// 恢复视频播放的辅助函数
	window.__wx_channels_resume_video__ = function(pausedVideos) {
		if (!pausedVideos || pausedVideos.length === 0) return;
		
		console.log('[视频助手] 恢复视频播放...');
		try {
			pausedVideos.forEach(item => {
				if (item.type === 'videojs' && item.player) {
					item.player.play();
					console.log('[视频助手] Video.js 播放器', item.index, '已恢复');
				} else if (item.type === 'native' && item.video) {
					item.video.play();
					console.log('[视频助手] 原生视频', item.index, '已恢复');
				}
			});
		} catch (e) {
			console.error('[视频助手] 恢复视频失败:', e);
		}
	};
	
	// 覆盖原有的下载处理函数
	const originalHandleClick = window.__wx_channels_handle_click_download__;
	if (originalHandleClick) {
		window.__wx_channels_handle_click_download__ = function(sp) {
			// 暂停视频
			const pausedVideos = window.__wx_channels_pause_video__();
			
			// 调用原始函数进行下载
			originalHandleClick(sp);
			
			// 注意：不再手动记录下载，因为后端API已经处理了记录保存
			// 移除重复的记录调用以避免CSV中出现重复记录
			
			// 3秒后恢复播放（给下载一些时间开始）
			setTimeout(() => {
				window.__wx_channels_resume_video__(pausedVideos);
			}, 5000);
		};
	}
	
	// 覆盖当前视频下载函数
	const originalDownloadCur = window.__wx_channels_download_cur__;
	if (originalDownloadCur) {
		window.__wx_channels_download_cur__ = function() {
			// 暂停视频
			const pausedVideos = window.__wx_channels_pause_video__();
			
			// 调用原始函数进行下载
			originalDownloadCur();
			
			// 注意：不再手动记录下载，因为后端API已经处理了记录保存
			// 移除重复的记录调用以避免CSV中出现重复记录
			
			// 3秒后恢复播放（给下载一些时间开始）
			setTimeout(() => {
				window.__wx_channels_resume_video__(pausedVideos);
			}, 3000);
		};
	}
	
	// 优化封面下载函数：使用后端API保存到服务器
	window.__wx_channels_handle_download_cover = function() {
		if (window.__wx_channels_store__ && window.__wx_channels_store__.profile) {
			const profile = window.__wx_channels_store__.profile;
			// 优先使用thumbUrl，然后是fullThumbUrl，最后才是coverUrl
			const coverUrl = profile.thumbUrl || profile.fullThumbUrl || profile.coverUrl;
			
			if (!coverUrl) {
				alert("未找到封面图片");
				return;
			}
			
			// 记录日志
			if (window.__wx_log) {
				window.__wx_log({
					msg: '正在保存封面到服务器...\n' + coverUrl
				});
			}
			
			// 构建请求数据
			const requestData = {
				coverUrl: coverUrl,
				videoId: profile.id || '',
				title: profile.title || '',
				author: profile.nickname || (profile.contact && profile.contact.nickname) || '未知作者',
				forceSave: false
			};
			
			// 添加授权头
			const headers = {
				'Content-Type': 'application/json'
			};
			if (window.__WX_LOCAL_TOKEN__) {
				headers['X-Local-Auth'] = window.__WX_LOCAL_TOKEN__;
			}
			
			// 发送到后端API保存封面
			fetch('/__wx_channels_api/save_cover', {
				method: 'POST',
				headers: headers,
				body: JSON.stringify(requestData)
			})
			.then(response => response.json())
			.then(data => {
				if (data.success) {
					const msg = data.message || '封面已保存';
					const path = data.relativePath || data.path || '';
					if (window.__wx_log) {
						window.__wx_log({
							msg: '✓ ' + msg
						});
					}
					console.log('✓ [封面下载] 封面已保存:', path);
				} else {
					const errorMsg = data.error || '保存封面失败';
					if (window.__wx_log) {
						window.__wx_log({
							msg: '❌ ' + errorMsg
						});
					}
					alert('保存封面失败: ' + errorMsg);
				}
			})
			.catch(error => {
				console.error("保存封面失败:", error);
				if (window.__wx_log) {
					window.__wx_log({
						msg: '❌ 保存封面失败: ' + error.message
					});
				}
				alert("保存封面失败: " + error.message);
			});
		} else {
			alert("未找到视频信息");
		}
	};
	</script>`
}

// getCaptureUrlScript 获取捕获完整URL的脚本
func (h *ScriptHandler) getCaptureUrlScript() string {
	return `<script>
	setTimeout(function() {
		// 获取完整的URL
		var fullUrl = window.location.href;
		// 发送到我们的API端点
		fetch("/__wx_channels_api/page_url", {
			method: "POST",
			headers: {
				"Content-Type": "application/json"
			},
			body: JSON.stringify({
				url: fullUrl
			})
		});
	}, 2000); // 延迟2秒执行，确保页面完全加载
	</script>`
}

// getSavePageContentScript 获取保存页面内容的脚本
func (h *ScriptHandler) getSavePageContentScript() string {
	return `<script>
	// 简单的字符串哈希函数 (djb2算法)
	function computeHash(str) {
		var hash = 5381;
		var i = str.length;
		while(i) {
			hash = (hash * 33) ^ str.charCodeAt(--i);
		}
		return hash >>> 0; // 强制转换为无符号32位整数
	}

	// 状态变量
	window.__wx_last_saved_hash = 0;
	window.__wx_save_timer = null;

	// 保存当前页面完整内容的函数 (带去重和防抖)
	window.__wx_channels_save_page_content = function(force) {
		try {
			// 清除之前的定时器
			if (window.__wx_save_timer) {
				clearTimeout(window.__wx_save_timer);
				window.__wx_save_timer = null;
			}

			// 获取当前完整的HTML内容
			var fullHtml = document.documentElement.outerHTML;
			
			// 计算哈希
			var currentHash = computeHash(fullHtml);

			// 如果不是强制保存，且哈希值与上次相同，则跳过
			if (!force && currentHash === window.__wx_last_saved_hash) {
				// console.log("[PageSave] 内容未变化，跳过保存");
				return;
			}

			var currentUrl = window.location.href;
			
			// 发送到保存API
			fetch("/__wx_channels_api/save_page_content", {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify({
					url: currentUrl,
					html: fullHtml,
					timestamp: new Date().getTime()
				})
			}).then(response => {
				if (response.ok) {
					console.log("[PageSave] 页面内容已保存");
					window.__wx_last_saved_hash = currentHash;
				}
			}).catch(error => {
				console.error("[PageSave] 保存页面内容失败:", error);
			});
		} catch (error) {
			console.error("[PageSave] 获取页面内容失败:", error);
		}
	};
	
	// 触发带防抖的保存 (默认延迟2秒)
	window.__wx_trigger_save_page = function(delay) {
		if (typeof delay === 'undefined') delay = 2000;
		
		if (window.__wx_save_timer) {
			clearTimeout(window.__wx_save_timer);
		}
		
		window.__wx_save_timer = setTimeout(function() {
			window.__wx_channels_save_page_content(false);
		}, delay);
	};

	// 监听URL变化，自动保存页面内容
	let currentPageUrl = window.location.href;
	const checkUrlChange = () => {
		if (window.location.href !== currentPageUrl) {
			currentPageUrl = window.location.href;
			// URL变化后延迟保存，等待内容加载
			window.__wx_trigger_save_page(5000);
		}
	};
	
	// 定期检查URL变化（适用于SPA）
	setInterval(checkUrlChange, 1000);
	
	// 监听历史记录变化
	window.addEventListener('popstate', () => {
		window.__wx_trigger_save_page(3000);
	});
	
	// 在页面加载完成后也保存一次
	setTimeout(() => {
		window.__wx_trigger_save_page(2000);
	}, 8000);
	</script>`
}

// getVideoCacheNotificationScript 获取视频缓存监控脚本
func (h *ScriptHandler) getVideoCacheNotificationScript() string {
	return `<script>
	// 初始化视频缓存监控
	window.__wx_channels_video_cache_monitor = {
		isBuffering: false,
		lastBufferTime: 0,
		totalBufferSize: 0,
		videoSize: 0,
		completeThreshold: 0.98, // 认为98%缓冲完成时视频已缓存完成
		checkInterval: null,
		notificationShown: false, // 防止重复显示通知
		
		// 开始监控缓存
		startMonitoring: function(expectedSize) {
			console.log('=== 开始启动视频缓存监控 ===');
			
			// 检查播放器状态
			const vjsPlayer = document.querySelector('.video-js');
			const video = vjsPlayer ? vjsPlayer.querySelector('video') : document.querySelector('video');
			
			if (!video) {
				console.error('未找到视频元素，无法启动监控');
				return;
			}
			
			console.log('视频元素状态:');
			console.log('- readyState:', video.readyState);
			console.log('- duration:', video.duration);
			console.log('- buffered.length:', video.buffered ? video.buffered.length : 0);
			
			if (this.checkInterval) {
				clearInterval(this.checkInterval);
			}
			
			this.isBuffering = true;
			this.lastBufferTime = Date.now();
			this.totalBufferSize = 0;
			this.videoSize = expectedSize || 0;
			this.notificationShown = false; // 重置通知状态
			
			console.log('视频缓存监控已启动');
			console.log('- 视频大小:', (this.videoSize / (1024 * 1024)).toFixed(2) + 'MB');
			console.log('- 监控间隔: 2秒');
			
			// 定期检查缓冲状态 - 增加检查频率
			this.checkInterval = setInterval(() => this.checkBufferStatus(), 2000);
			
			// 添加可见的缓存状态指示器
			this.addStatusIndicator();
			
			// 监听视频播放完成事件
			this.setupVideoEndedListener();
			
			// 延迟开始监控，让播放器有时间初始化
			setTimeout(() =>{
				this.monitorNativeBuffering();
			}, 1000);
		},
		
		// 监控Video.js播放器和原生视频元素的缓冲状态
		monitorNativeBuffering: function() {
			let firstCheck = true; // 标记是否是第一次检查
			const checkBufferedProgress = () => {
				// 优先检查Video.js播放器
				const vjsPlayer = document.querySelector('.video-js');
				let video = null;
				
				if (vjsPlayer) {
					// 从Video.js播放器中获取video元素
					video = vjsPlayer.querySelector('video');
					if (firstCheck) {
						console.log('找到Video.js播放器，开始监控');
						firstCheck = false;
					}
				} else {
					// 回退到查找普通video元素
					const videoElements = document.querySelectorAll('video');
					if (videoElements.length > 0) {
						video = videoElements[0];
						if (firstCheck) {
							console.log('使用普通video元素监控');
							firstCheck = false;
						}
					}
				}
				
				if (video) {
					// 获取预加载进度条数据
					if (video.buffered && video.buffered.length > 0 && video.duration) {
						// 获取最后缓冲时间范围的结束位置
						const bufferedEnd = video.buffered.end(video.buffered.length - 1);
						// 计算缓冲百分比
						const bufferedPercent = (bufferedEnd / video.duration) * 100;
						
						// 更新页面指示器
						const indicator = document.getElementById('video-cache-indicator');
						if (indicator) {
							indicator.innerHTML = '<div>视频缓存中: ' + bufferedPercent.toFixed(1) + '% (Video.js播放器)</div>';
							
							// 高亮显示接近完成的状态
							if (bufferedPercent >= 95) {
								indicator.style.backgroundColor = 'rgba(0,128,0,0.8)';
							}
						}
						
						// 检查Video.js播放器的就绪状态（只在第一次检查时输出）
						if (vjsPlayer && typeof vjsPlayer.readyState !== 'undefined' && firstCheck) {
							console.log('Video.js播放器就绪状态:', vjsPlayer.readyState);
						}
						
						// 检查是否缓冲完成
						if (bufferedPercent >= 98) {
							console.log('根据Video.js播放器数据，视频已缓存完成 (' + bufferedPercent.toFixed(1) + '%)');
							this.showNotification();
							this.stopMonitoring();
							return true; // 缓存完成，停止监控
						}
					}
				}
				return false; // 继续监控
			};
			
			// 立即检查一次
			if (!checkBufferedProgress()) {
				// 每秒检查一次预加载进度
				const bufferCheckInterval = setInterval(() => {
					if (checkBufferedProgress() || !this.isBuffering) {
						clearInterval(bufferCheckInterval);
					}
				}, 1000);
			}
		},
		
		// 设置Video.js播放器和视频播放结束监听
		setupVideoEndedListener: function() {
			// 尝试查找Video.js播放器和视频元素
			setTimeout(() => {
				const vjsPlayer = document.querySelector('.video-js');
				let video = null;
				
				if (vjsPlayer) {
					// 从Video.js播放器中获取video元素
					video = vjsPlayer.querySelector('video');
					console.log('为Video.js播放器设置事件监听');
					
					// 尝试监听Video.js特有的事件
					if (vjsPlayer.addEventListener) {
						vjsPlayer.addEventListener('ended', () => {
							console.log('Video.js播放器播放结束，标记为缓存完成');
							this.showNotification();
							this.stopMonitoring();
						});
						
						vjsPlayer.addEventListener('loadeddata', () => {
							console.log('Video.js播放器数据加载完成');
						});
					}
				} else {
					// 回退到查找普通video元素
					const videoElements = document.querySelectorAll('video');
					if (videoElements.length > 0) {
						video = videoElements[0];
						console.log('为普通video元素设置事件监听');
					}
				}
				
				if (video) {
					// 监听视频播放结束事件
					video.addEventListener('ended', () => {
						console.log('视频播放已结束，标记为缓存完成');
						this.showNotification();
						this.stopMonitoring();
					});
					
					// 如果视频已在播放中，添加定期检查播放状态
					if (!video.paused) {
						const playStateInterval = setInterval(() => {
							// 如果视频已经播放完或接近结束（剩余小于2秒）
							if (video.ended || (video.duration && video.currentTime > 0 && video.duration - video.currentTime < 2)) {
								console.log('视频接近或已播放完成，标记为缓存完成');
								this.showNotification();
								this.stopMonitoring();
								clearInterval(playStateInterval);
							}
						}, 1000);
					}
				}
			}, 3000); // 延迟3秒再查找视频元素，确保Video.js播放器完全初始化
		},
		
		// 添加缓冲状态指示器
		addStatusIndicator: function() {
			console.log('正在创建缓存状态指示器...');
			
			// 移除现有指示器
			const existingIndicator = document.getElementById('video-cache-indicator');
			if (existingIndicator) {
				console.log('移除现有指示器');
				existingIndicator.remove();
			}
			
			// 创建新指示器
			const indicator = document.createElement('div');
			indicator.id = 'video-cache-indicator';
			indicator.style.cssText = "position:fixed;bottom:20px;left:20px;background-color:rgba(0,0,0,0.8);color:white;padding:10px 15px;border-radius:6px;z-index:99999;font-size:14px;font-family:Arial,sans-serif;border:2px solid rgba(255,255,255,0.3);";
			indicator.innerHTML = '<div>🔄 视频缓存中: 0%</div>';
			document.body.appendChild(indicator);
			
			console.log('缓存状态指示器已创建并添加到页面');
			
			// 初始化进度跟踪变量
			this.lastLoggedProgress = 0;
			this.stuckCheckCount = 0;
			this.maxStuckCount = 30; // 30秒不变则认为停滞
			
			// 每秒更新进度
			const updateInterval = setInterval(() => {
				if (!this.isBuffering) {
					clearInterval(updateInterval);
					indicator.remove();
					return;
				}
				
				let progress = 0;
				let progressSource = 'unknown';
				
				// 优先方案：从video元素实时读取（最准确）
				const vjsPlayer = document.querySelector('.video-js');
				let video = vjsPlayer ? vjsPlayer.querySelector('video') : null;
				
				if (!video) {
					const videoElements = document.querySelectorAll('video');
					if (videoElements.length > 0) {
						video = videoElements[0];
					}
				}
				
				if (video && video.buffered && video.buffered.length > 0) {
					try {
						const bufferedEnd = video.buffered.end(video.buffered.length - 1);
						const duration = video.duration;
						if (duration > 0 && !isNaN(duration) && isFinite(duration)) {
							progress = (bufferedEnd / duration) * 100;
							progressSource = 'video.buffered';
						}
					} catch (e) {
						// 忽略读取错误
					}
				}
				
				// 备用方案：使用 totalBufferSize
				if (progress === 0 && this.videoSize > 0 && this.totalBufferSize > 0) {
					progress = (this.totalBufferSize / this.videoSize) * 100;
					progressSource = 'totalBufferSize';
				}
				
				// 限制进度范围
				progress = Math.min(Math.max(progress, 0), 100);
				
				// 检测进度是否停滞
				const progressChanged = Math.abs(progress - this.lastLoggedProgress) >= 0.1;
				
				if (!progressChanged) {
					this.stuckCheckCount++;
				} else {
					this.stuckCheckCount = 0;
				}
				
				// 更新指示器
				if (progress > 0) {
					// 根据停滞状态显示不同的图标
					let icon = '🔄';
					let statusText = '视频缓存中';
					
					if (this.stuckCheckCount >= this.maxStuckCount) {
						icon = '⏸️';
						statusText = '缓存暂停';
						indicator.style.backgroundColor = 'rgba(128,128,128,0.8)';
					} else if (progress >= 95) {
						icon = '✅';
						statusText = '缓存接近完成';
						indicator.style.backgroundColor = 'rgba(0,128,0,0.8)';
					} else if (progress >= 50) {
						indicator.style.backgroundColor = 'rgba(255,165,0,0.8)';
					} else {
						indicator.style.backgroundColor = 'rgba(0,0,0,0.8)';
					}
					
					indicator.innerHTML = '<div>' + icon + ' ' + statusText + ': ' + progress.toFixed(1) + '%</div>';
					
					// 只在进度变化≥1%时输出日志
					if (Math.abs(progress - this.lastLoggedProgress) >= 1) {
						console.log('缓存进度更新:', progress.toFixed(1) + '% (来源:' + progressSource + ')');
						this.lastLoggedProgress = progress;
					}
					
					// 停滞提示（只输出一次）
					if (this.stuckCheckCount === this.maxStuckCount) {
						console.log('⏸️ 缓存进度长时间未变化 (' + progress.toFixed(1) + '%)，可能原因：');
						console.log('  - 视频已暂停播放');
						console.log('  - 网络速度慢或连接中断');
						console.log('  - 浏览器缓存策略限制');
						console.log('  提示：继续播放视频可能会恢复缓存');
					}
				} else {
					indicator.innerHTML = '<div>⏳ 等待视频数据...</div>';
				}
				
				// 如果进度达到98%以上，检查是否完成
				if (progress >= 98) {
					this.checkCompletion();
				}
			}, 1000);
		},
		
		// 添加缓冲块
		addBuffer: function(buffer) {
			if (!this.isBuffering) return;
			
			// 更新最后缓冲时间
			this.lastBufferTime = Date.now();
			
			// 累计缓冲大小
			if (buffer && buffer.byteLength) {
				this.totalBufferSize += buffer.byteLength;
				
				// 输出调试信息到控制台
				if (this.videoSize > 0) {
					const percent = ((this.totalBufferSize / this.videoSize) * 100).toFixed(1);
					console.log('视频缓存进度: ' + percent + '% (' + (this.totalBufferSize / (1024 * 1024)).toFixed(2) + 'MB/' + (this.videoSize / (1024 * 1024)).toFixed(2) + 'MB)');
				}
			}
			
			// 检查是否接近完成
			this.checkCompletion();
		},
		
		// 检查Video.js播放器和原生视频的缓冲状态
		checkBufferStatus: function() {
			if (!this.isBuffering) return;
			
			// 优先检查Video.js播放器
			const vjsPlayer = document.querySelector('.video-js');
			let video = null;
			
			if (vjsPlayer) {
				// 从Video.js播放器中获取video元素
				video = vjsPlayer.querySelector('video');
				
				// 检查Video.js播放器特有的状态（只在状态变化时输出日志）
				if (vjsPlayer.classList.contains('vjs-has-started')) {
					if (!this._vjsStartedLogged) {
						console.log('Video.js播放器已开始播放');
						this._vjsStartedLogged = true;
					}
				}
				
				if (vjsPlayer.classList.contains('vjs-waiting')) {
					if (!this._vjsWaitingLogged) {
						console.log('Video.js播放器正在等待数据');
						this._vjsWaitingLogged = true;
					}
				} else {
					this._vjsWaitingLogged = false; // 重置标记，以便下次等待时再次输出
				}
				
				if (vjsPlayer.classList.contains('vjs-ended')) {
					console.log('Video.js播放器播放结束，标记为缓存完成');
					this.checkCompletion(true);
					return;
				}
			} else {
				// 回退到查找普通video元素
				const videoElements = document.querySelectorAll('video');
				if (videoElements.length > 0) {
					video = videoElements[0];
				}
			}
			
			if (video) {
				if (video.buffered && video.buffered.length > 0 && video.duration) {
					// 获取最后缓冲时间范围的结束位置
					const bufferedEnd = video.buffered.end(video.buffered.length - 1);
					// 计算缓冲百分比
					const bufferedPercent = (bufferedEnd / video.duration) * 100;
					
					// 如果预加载接近完成，触发完成检测（只输出一次日志）
					if (bufferedPercent >= 95 && !this._preloadNearCompleteLogged) {
						console.log('检测到视频预加载接近完成 (' + bufferedPercent.toFixed(1) + '%)');
						this._preloadNearCompleteLogged = true;
						this.checkCompletion(true);
					}
				}
				
				// 只在readyState为4且缓冲百分比较高时才认为完成
				if (video.readyState >= 4 && video.buffered && video.buffered.length > 0 && video.duration) {
					const bufferedEnd = video.buffered.end(video.buffered.length - 1);
					const bufferedPercent = (bufferedEnd / video.duration) * 100;
					if (bufferedPercent >= 98 && !this._readyStateCompleteLogged) {
						console.log('视频readyState为4且缓冲98%以上，标记为缓存完成');
						this._readyStateCompleteLogged = true;
						this.checkCompletion(true);
					}
				}
			}
			
			// 如果超过10秒没有新的缓冲数据且已经缓冲了部分数据，可能表示视频已暂停或缓冲完成
			const timeSinceLastBuffer = Date.now() - this.lastBufferTime;
			if (timeSinceLastBuffer > 10000 && this.totalBufferSize > 0) {
				this.checkCompletion(true);
			}
		},
		
		// 检查是否完成
		checkCompletion: function(forcedCheck) {
			if (!this.isBuffering) return;
			
			let isComplete = false;
			
			// 优先检查Video.js播放器是否已播放完成
			const vjsPlayer = document.querySelector('.video-js');
			let video = null;
			
			if (vjsPlayer) {
				video = vjsPlayer.querySelector('video');
				
				// 检查Video.js播放器的完成状态
				if (vjsPlayer.classList.contains('vjs-ended')) {
					console.log('Video.js播放器已播放完毕，认为缓存完成');
					isComplete = true;
				}
			} else {
				// 回退到查找普通video元素
				const videoElements = document.querySelectorAll('video');
				if (videoElements.length > 0) {
					video = videoElements[0];
				}
			}
			
			if (video && !isComplete) {
				// 如果视频已经播放完毕或接近结束，直接认为完成
				if (video.ended || (video.duration && video.currentTime > 0 && video.duration - video.currentTime < 2)) {
					console.log('视频已播放完毕或接近结束，认为缓存完成');
					isComplete = true;
				}
				
				// 只在readyState为4且缓冲百分比较高时才认为完成
				if (video.readyState >= 4 && video.buffered && video.buffered.length > 0 && video.duration) {
					const bufferedEnd = video.buffered.end(video.buffered.length - 1);
					const bufferedPercent = (bufferedEnd / video.duration) * 100;
					if (bufferedPercent >= 98) {
						console.log('视频readyState为4且缓冲98%以上，认为缓存完成');
						isComplete = true;
					}
				}
			}
			
			// 如果未通过播放状态判断完成，再检查缓冲大小
			if (!isComplete) {
				// 如果知道视频大小，则根据百分比判断
				if (this.videoSize > 0) {
					const ratio = this.totalBufferSize / this.videoSize;
					// 对短视频降低阈值要求
					const threshold = this.videoSize < 5 * 1024 * 1024 ? 0.9 : this.completeThreshold; // 5MB以下视频降低阈值到90%
					isComplete = ratio >= threshold;
				} 
				// 强制检查：如果长时间没有新数据且视频元素可以播放到最后，也认为已完成
				else if (forcedCheck && video) {
					if (video.readyState >= 3 && video.buffered.length > 0) {
						const bufferedEnd = video.buffered.end(video.buffered.length - 1);
						const duration = video.duration;
						isComplete = duration > 0 && (bufferedEnd / duration) >= 0.95; // 降低阈值到95%
						
						if (isComplete) {
							console.log('强制检查：根据缓冲数据判断视频缓存完成');
						}
					}
				}
			}
			
			// 如果完成，显示通知
			if (isComplete) {
				this.showNotification();
				this.stopMonitoring();
			}
		},
		
		// 显示通知
		showNotification: function() {
			// 防止重复显示通知
			if (this.notificationShown) {
				console.log('通知已经显示过，跳过重复显示');
				return;
			}
			
			console.log('显示缓存完成通知');
			this.notificationShown = true;
			
			// 移除进度指示器
			const indicator = document.getElementById('video-cache-indicator');
			if (indicator) {
				indicator.remove();
			}
			
			// 创建桌面通知
			if ("Notification" in window && Notification.permission === "granted") {
				new Notification("视频缓存完成", {
					body: "视频已缓存完成，可以进行下载操作",
					icon: window.__wx_channels_store__?.profile?.coverUrl
				});
			}
			
			// 在页面上显示通知
			const notification = document.createElement('div');
			notification.style.cssText = "position:fixed;bottom:20px;right:20px;background-color:rgba(0,128,0,0.9);color:white;padding:15px 25px;border-radius:8px;z-index:99999;animation:fadeInOut 12s forwards;box-shadow:0 4px 12px rgba(0,0,0,0.3);font-size:16px;font-weight:bold;";
			notification.innerHTML = '<div style="display:flex;align-items:center;"><span style="font-size:24px;margin-right:12px;">🎉</span> <span>视频缓存完成，可以下载了！</span></div>';
			
			// 添加动画样式 - 延长显示时间到12秒
			const style = document.createElement('style');
			style.textContent = '@keyframes fadeInOut {0% {opacity:0;transform:translateY(20px);} 8% {opacity:1;transform:translateY(0);} 85% {opacity:1;} 100% {opacity:0;}}';
			document.head.appendChild(style);
			
			document.body.appendChild(notification);
			
			// 12秒后移除通知
			setTimeout(() => {
				notification.remove();
			}, 12000);
			
			// 发送通知事件
			fetch("/__wx_channels_api/tip", {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify({
					msg: "视频缓存完成，可以下载了！"
				})
			});
			
			console.log("视频缓存完成通知已显示");
		},
		
		// 停止监控
		stopMonitoring: function() {
			console.log('停止视频缓存监控');
			if (this.checkInterval) {
				clearInterval(this.checkInterval);
				this.checkInterval = null;
			}
			this.isBuffering = false;
			// 注意：不重置notificationShown，保持通知状态直到下次startMonitoring
		}
	};
	
	// 请求通知权限
	if ("Notification" in window && Notification.permission !== "granted" && Notification.permission !== "denied") {
		// 用户操作后再请求权限
		document.addEventListener('click', function requestPermission() {
			Notification.requestPermission();
			document.removeEventListener('click', requestPermission);
		}, {once: true});
	}
	</script>`
}

// handleIndexPublish 处理index.publish JS文件
func (h *ScriptHandler) handleIndexPublish(path string, content string) (string, bool) {
	if !util.Includes(path, "/t/wx_fed/finder/web/web-finder/res/js/index.publish") {
		return content, false
	}

	utils.LogFileInfo("[Home数据采集] 正在处理 index.publish 文件")

	regexp1 := regexp.MustCompile(`this.sourceBuffer.appendBuffer\(h\),`)
	replaceStr1 := `(() => {
if (window.__wx_channels_store__) {
window.__wx_channels_store__.buffers.push(h);
// 添加缓存监控
if (window.__wx_channels_video_cache_monitor) {
    window.__wx_channels_video_cache_monitor.addBuffer(h);
}
}
})(),this.sourceBuffer.appendBuffer(h),`
	if regexp1.MatchString(content) {
		utils.LogFileInfo("视频播放已成功加载！")
		utils.LogFileInfo("视频缓冲将被监控，完成时会有提醒")
		utils.LogFileInfo("[视频播放] 视频播放器已加载 | Path=%s", path)
	}
	content = regexp1.ReplaceAllString(content, replaceStr1)
	regexp2 := regexp.MustCompile(`if\(f.cmd===re.MAIN_THREAD_CMD.AUTO_CUT`)
	replaceStr2 := `if(f.cmd==="CUT"){
	if (window.__wx_channels_store__) {
	// console.log("CUT", f, __wx_channels_store__.profile.key);
	window.__wx_channels_store__.keys[__wx_channels_store__.profile.key]=f.decryptor_array;
	}
}
if(f.cmd===re.MAIN_THREAD_CMD.AUTO_CUT`
	content = regexp2.ReplaceAllString(content, replaceStr2)

	return content, true
}

// handleVirtualSvgIcons 处理virtual_svg-icons-register JS文件
func (h *ScriptHandler) handleVirtualSvgIcons(path string, content string) (string, bool) {
	if !util.Includes(path, "/t/wx_fed/finder/web/web-finder/res/js/virtual_svg-icons-register") {
		return content, false
	}

	// 拦截 finderPcFlow - 首页推荐视频列表（参考 wx_channels_download 项目）
	pcFlowRegex := regexp.MustCompile(`(?s)async\s+finderPcFlow\s*\(([^)]+)\)\s*\{(.*?)\}\s*async`)
	if pcFlowRegex.MatchString(content) {
		utils.LogFileInfo("[API拦截] ✅ 在virtual_svg-icons-register中成功拦截 finderPcFlow 函数")
		pcFlowReplace := `async finderPcFlow($1){var result=await(async()=>{$2})();if(result&&result.data&&result.data.object){var feeds=result.data.object;console.log("[API拦截] finderPcFlow 触发 PCFlowLoaded",feeds.length);WXU.emit(WXU.Events.PCFlowLoaded,{feeds:feeds,params:$1});}return result;}async`
		content = pcFlowRegex.ReplaceAllString(content, pcFlowReplace)
	}

	// 拦截 finderStream - 另一种首页推荐列表
	streamRegex := regexp.MustCompile(`(?s)async\s+finderStream\s*\(([^)]+)\)\s*\{(.*?)\}\s*async`)
	if streamRegex.MatchString(content) {
		utils.LogFileInfo("[API拦截] ✅ 在virtual_svg-icons-register中成功拦截 finderStream 函数")
		streamReplace := `async finderStream($1){var result=await(async()=>{$2})();if(result&&result.data&&result.data.object){var feeds=result.data.object;console.log("[API拦截] finderStream 触发 PCFlowLoaded",feeds.length);WXU.emit(WXU.Events.PCFlowLoaded,{feeds:feeds,params:$1});}return result;}async`
		content = streamRegex.ReplaceAllString(content, streamReplace)
	}

	// 拦截 finderGetCommentDetail - 视频详情（参考 wx_channels_download 项目）
	feedProfileRegex := regexp.MustCompile(`(?s)async\s+finderGetCommentDetail\s*\(([^)]+)\)\s*\{(.*?)\}\s*async`)
	if feedProfileRegex.MatchString(content) {
		utils.LogFileInfo("[API拦截] ✅ 在virtual_svg-icons-register中成功拦截 finderGetCommentDetail 函数")
		feedProfileReplace := `async finderGetCommentDetail($1){var result=await(async()=>{$2})();var feed=result.data.object;console.log("[API拦截] finderGetCommentDetail 触发 FeedProfileLoaded");WXU.emit(WXU.Events.FeedProfileLoaded,feed);return result;}async`
		content = feedProfileRegex.ReplaceAllString(content, feedProfileReplace)
	}

	// 拦截 Profile 页面的视频列表数据 - 使用事件系统（参考 wx_channels_download 项目）
	profileListRegex := regexp.MustCompile(`(?s)async\s+finderUserPage\s*\(([^)]+)\)\s*\{return(.*?)\}\s*async`)
	if profileListRegex.MatchString(content) {
		utils.LogFileInfo("[API拦截] ✅ 在virtual_svg-icons-register中成功拦截 finderUserPage 函数")
		profileListReplace := `async finderUserPage($1){console.log("[Profile API] finderUserPage 调用参数:",$1);var result=await(async()=>{return$2})();console.log("[Profile API] finderUserPage 原始结果:",result);if(result&&result.data&&result.data.object){var feeds=result.data.object;console.log("[Profile API] 提取到",feeds.length,"个视频");WXU.emit(WXU.Events.UserFeedsLoaded,feeds);}else{console.warn("[Profile API] result.data.object 为空",result);}return result;}async`
		content = profileListRegex.ReplaceAllString(content, profileListReplace)
	}

	// 拦截 Profile 页面的直播回放列表数据 - 使用事件系统
	liveListRegex := regexp.MustCompile(`(?s)async\s+finderLiveUserPage\s*\(([^)]+)\)\s*\{return(.*?)\}\s*async`)
	if liveListRegex.MatchString(content) {
		utils.LogFileInfo("[API拦截] ✅ 在virtual_svg-icons-register中成功拦截 finderLiveUserPage 函数")
		liveListReplace := `async finderLiveUserPage($1){console.log("[Profile API] finderLiveUserPage 调用参数:",$1);var result=await(async()=>{return$2})();console.log("[Profile API] finderLiveUserPage 原始结果:",result);if(result&&result.data&&result.data.object){var feeds=result.data.object;console.log("[Profile API] 提取到",feeds.length,"个直播回放");WXU.emit(WXU.Events.UserLiveReplayLoaded,feeds);}else{console.warn("[Profile API] result.data.object 为空",result);}return result;}async`
		content = liveListRegex.ReplaceAllString(content, liveListReplace)
	}

	// 拦截分类视频列表API - finderGetRecommend（首页、美食、生活等分类tab）
	categoryFeedsRegex := regexp.MustCompile(`(?s)async\s+finderGetRecommend\s*\(([^)]+)\)\s*\{(.*?)\}\s*async`)
	if categoryFeedsRegex.MatchString(content) {
		utils.LogFileInfo("[API拦截] ✅ 在virtual_svg-icons-register中成功拦截 finderGetRecommend 函数")
		categoryFeedsReplace := `async finderGetRecommend($1){var result=await(async()=>{$2})();if(result&&result.data&&result.data.object){var feeds=result.data.object;WXU.emit(WXU.Events.CategoryFeedsLoaded,{feeds:feeds,params:$1});}return result;}async`
		content = categoryFeedsRegex.ReplaceAllString(content, categoryFeedsReplace)
	}

	// 拦截搜索API - finderPCSearch（PC端搜索）
	// 函数格式: async finderPCSearch(n){...return(...),t}async
	// 在最后的 return 之前插入代码，然后保持 ,t}async 不变
	searchPCRegex := regexp.MustCompile(`(async finderPCSearch\([^)]+\)\{.*?)(,t\}async)`)

	if searchPCRegex.MatchString(content) {
		utils.LogFileInfo("[API拦截] ✅ 在virtual_svg-icons-register中成功拦截 finderPCSearch 函数")
		// 在 ,t 之前插入代码，保持 ,t}async 完整
		// 从 acctList 中提取正在直播的账号，添加调试日志
		searchPCReplace := `$1,t&&t.data&&(function(){var lives=t.data.liveObjectList||[];var accounts=[];var liveCount=0;if(t.data.acctList){t.data.acctList.forEach(function(info){if(info.liveStatus===1){liveCount++;console.log("[搜索API] 发现直播账号:",info.contact?info.contact.nickname:"未知",info.liveStatus,info.liveInfo);}if(info.liveStatus===1&&info.liveInfo){lives.push({id:info.contact.username,objectId:info.contact.username,nickname:info.contact.nickname,username:info.contact.username,description:info.liveInfo.description||"",streamUrl:info.liveInfo.streamUrl,coverUrl:info.liveInfo.media&&info.liveInfo.media[0]?info.liveInfo.media[0].thumbUrl:"",thumbUrl:info.liveInfo.media&&info.liveInfo.media[0]?info.liveInfo.media[0].thumbUrl:"",liveInfo:info.liveInfo,type:"live"});}accounts.push(info);});}if(liveCount>0){console.log("[搜索API] 共发现",liveCount,"个直播账号，成功提取",lives.length,"个");}var searchData={feeds:t.data.objectList||[],accounts:accounts,lives:lives};WXU.emit("SearchResultLoaded",searchData);})()$2`
		content = searchPCRegex.ReplaceAllString(content, searchPCReplace)
	} else {
		utils.LogFileInfo("[API拦截] ❌ 在virtual_svg-icons-register中未找到 finderPCSearch 函数")
	}

	// 拦截搜索API - finderSearch（移动端搜索）
	// 使用非贪婪匹配，匹配到最后的 ,t}async 模式
	searchRegex := regexp.MustCompile(`(async finderSearch\([^)]+\)\{.*?)(,t\}async)`)

	if searchRegex.MatchString(content) {
		utils.LogFileInfo("[API拦截] ✅ 在virtual_svg-icons-register中成功拦截 finderSearch 函数")
		// 从 infoList 中提取正在直播的账号，添加调试日志
		searchReplace := `$1,t&&t.data&&(function(){var lives=[];var accounts=[];var liveCount=0;if(t.data.infoList){t.data.infoList.forEach(function(info){if(info.liveStatus===1){liveCount++;console.log("[搜索API] 发现直播账号:",info.contact?info.contact.nickname:"未知",info.liveStatus,info.liveInfo);}if(info.liveStatus===1&&info.liveInfo){lives.push({id:info.contact.username,objectId:info.contact.username,nickname:info.contact.nickname,username:info.contact.username,description:info.liveInfo.description||"",streamUrl:info.liveInfo.streamUrl,coverUrl:info.liveInfo.media&&info.liveInfo.media[0]?info.liveInfo.media[0].thumbUrl:"",thumbUrl:info.liveInfo.media&&info.liveInfo.media[0]?info.liveInfo.media[0].thumbUrl:"",liveInfo:info.liveInfo,type:"live"});}accounts.push(info);});}if(liveCount>0){console.log("[搜索API] 共发现",liveCount,"个直播账号，成功提取",lives.length,"个");}var searchData={feeds:t.data.objectList||[],accounts:accounts,lives:lives};WXU.emit("SearchResultLoaded",searchData);})()$2`
		content = searchRegex.ReplaceAllString(content, searchReplace)
	} else {
		utils.LogFileInfo("[API拦截] ❌ 在virtual_svg-icons-register中未找到 finderSearch 函数")
	}

	// 拦截 export 语句，提取所有导出的 API 函数
	// 格式: export{xxx as yyy,zzz as www,...}
	exportBlockRegex := regexp.MustCompile(`export\s*\{([^}]+)\}`)
	exportRegex := regexp.MustCompile(`export\s*\{`)

	if exportBlockRegex.MatchString(content) {
		utils.LogFileInfo("[API拦截] ✅ 在virtual_svg-icons-register中找到 export 语句")

		// 提取 export 块中的内容
		matches := exportBlockRegex.FindStringSubmatch(content)
		if len(matches) >= 2 {
			exportContent := matches[1]
			utils.LogFileInfo("[API拦截] Export 内容: %s", exportContent[:min(100, len(exportContent))])

			// 解析导出的函数名
			items := strings.Split(exportContent, ",")
			var locals []string
			for _, item := range items {
				p := strings.TrimSpace(item)
				if p == "" {
					continue
				}
				// 处理 "xxx as yyy" 格式
				idx := strings.Index(p, " as ")
				local := p
				if idx != -1 {
					local = strings.TrimSpace(p[:idx])
				}
				if local != "" && local != " " {
					locals = append(locals, local)
				}
			}

			if len(locals) > 0 {
				utils.LogFileInfo("[API拦截] 提取到 %d 个导出函数", len(locals))
				apiMethods := "{" + strings.Join(locals, ",") + "}"
				// 转义 $ 符号
				apiMethodsEscaped := strings.ReplaceAll(apiMethods, "$", "$$")

				// 在 export 之前插入 API 加载事件
				jsWXAPI := ";WXU.emit(WXU.Events.APILoaded," + apiMethodsEscaped + ");export{"
				content = exportRegex.ReplaceAllString(content, jsWXAPI)
				utils.LogFileInfo("[API拦截] ✅ 已注入 APILoaded 事件")
			}
		}
	} else {
		utils.LogFileInfo("[API拦截] ❌ 在virtual_svg-icons-register中未找到 export 语句")
	}

	return content, true
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleWorkerRelease 处理worker_release JS文件
func (h *ScriptHandler) handleWorkerRelease(path string, content string) (string, bool) {
	if !util.Includes(path, "worker_release") {
		return content, false
	}

	regex := regexp.MustCompile(`fmp4Index:p.fmp4Index`)
	replaceStr := `decryptor_array:p.decryptor_array,fmp4Index:p.fmp4Index`
	content = regex.ReplaceAllString(content, replaceStr)
	return content, true
}

// handleConnectPublish 处理connect.publish JS文件（参考 wx_channels_download 项目的实现）
func (h *ScriptHandler) handleConnectPublish(Conn *SunnyNet.HttpConn, path string, content string) (string, bool) {
	if !util.Includes(path, "connect.publish") {
		return content, false
	}

	utils.LogFileInfo("[Home数据采集] ✅ 正在处理 connect.publish 文件")

	// 首先找到 flowTab 对应的变量名（可能是 yt, nn 或其他）
	// 格式: flowTab:变量名,flowTabId:
	flowTabReg := regexp.MustCompile(`flowTab:([a-zA-Z]{1,}),flowTabId:`)
	flowTabVar := "yt" // 默认值
	if matches := flowTabReg.FindStringSubmatch(content); len(matches) > 1 {
		flowTabVar = matches[1]
		utils.LogFileInfo("[Home数据采集] ✅ 找到 flowTab 变量名: %s", flowTabVar)
	} else {
		utils.LogFileInfo("[Home数据采集] ⚠️ 未找到 flowTab 变量名，使用默认值: %s", flowTabVar)
	}

	// 参考 wx_channels_download 项目的正则表达式，匹配函数定义而不是函数调用
	// 原始代码格式: goToNextFlowFeed:函数名 或 goToPrevFlowFeed:函数名
	goToNextFlowReg := regexp.MustCompile(`goToNextFlowFeed:([a-zA-Z]{1,})`)
	goToPrevFlowReg := regexp.MustCompile(`goToPrevFlowFeed:([a-zA-Z]{1,})`)

	// 替换 goToNextFlowFeed 函数定义 - 使用 WXU.emit 发送事件（与 wx_channels_download 完全一致）
	if goToNextFlowReg.MatchString(content) {
		utils.LogFileInfo("[Home数据采集] ✅ 在connect.publish中成功拦截 goToNextFlowFeed 函数定义")
		// 使用动态获取的 flowTab 变量名
		jsGoNextFeed := fmt.Sprintf("goToNextFlowFeed:async function(v){await $1(v);console.log('goToNextFlowFeed',%s);if(!%s||!%s.value.feeds){return;}var feed=%s.value.feeds[%s.value.currentFeedIndex];console.log('before GotoNextFeed',%s,feed);WXU.emit(WXU.Events.GotoNextFeed,feed);}", flowTabVar, flowTabVar, flowTabVar, flowTabVar, flowTabVar, flowTabVar)
		content = goToNextFlowReg.ReplaceAllString(content, jsGoNextFeed)
	} else {
		utils.LogFileInfo("[Home数据采集] ❌ 在connect.publish中未找到 goToNextFlowFeed 函数定义")
	}

	// 替换 goToPrevFlowFeed 函数定义 - 使用 WXU.emit 发送事件
	if goToPrevFlowReg.MatchString(content) {
		utils.LogFileInfo("[Home数据采集] ✅ 在connect.publish中成功拦截 goToPrevFlowFeed 函数定义")
		// 使用动态获取的 flowTab 变量名
		jsGoPrevFeed := fmt.Sprintf("goToPrevFlowFeed:async function(v){await $1(v);console.log('goToPrevFlowFeed',%s);if(!%s||!%s.value.feeds){return;}var feed=%s.value.feeds[%s.value.currentFeedIndex];console.log('before GotoPrevFeed',%s,feed);WXU.emit(WXU.Events.GotoPrevFeed,feed);}", flowTabVar, flowTabVar, flowTabVar, flowTabVar, flowTabVar, flowTabVar)
		content = goToPrevFlowReg.ReplaceAllString(content, jsGoPrevFeed)
	} else {
		utils.LogFileInfo("[Home数据采集] ❌ 在connect.publish中未找到 goToPrevFlowFeed 函数定义")
	}

	// 禁用浏览器缓存，确保每次都能拦截到最新的代码
	Conn.Response.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	Conn.Response.Header.Set("Pragma", "no-cache")
	Conn.Response.Header.Set("Expires", "0")

	Conn.Response.Body = io.NopCloser(bytes.NewBuffer([]byte(content)))
	return content, true
}

// getCommentCaptureScript 获取评论采集脚本 (优化版 - 基于 Pinia 订阅)
func (h *ScriptHandler) getCommentCaptureScript() string {
	return `<script>
(function() {
	'use strict';
	
	console.log('[评论采集] 初始化新版采集系统 (Pinia Store订阅模式 v3)...');
	
	// 状态变量
	var autoScrollTimer = null;
	var lastCommentCount = 0;
	var noChangeCount = 0;
	var isCollecting = false;
	var currentFeedId = '';
	var saveDebounceTimer = null;
	var stableDataCount = 0;
	
	// 工具函数：查找 Vue/Pinia Store
	function findFeedStore() {
		try {
			var app = document.querySelector('[data-v-app]') || document.getElementById('app');
			if (!app) return null;
			
			var vue = app.__vue__ || app.__vueParentComponent || (app._vnode && app._vnode.component);
			if (!vue) return null;
			
			var appContext = vue.appContext || (vue.ctx && vue.ctx.appContext);
			if (!appContext || !appContext.config || !appContext.config.globalProperties) return null;
			
			var pinia = appContext.config.globalProperties.$pinia;
			if (!pinia) return null;
			
			// 尝试从不同路径获取 feed store
			if (pinia._s && pinia._s.feed) return pinia._s.feed;
			if (pinia.state && pinia.state._value && pinia.state._value.feed) return pinia.state._value.feed;
			
			return null;
		} catch (e) {
			console.error('[评论采集] 查找Store失败:', e);
			return null;
		}
	}

	// 格式化评论数据，使其符合后端 API 要求
	function formatComments(items) {
		if (!items || !Array.isArray(items)) return [];
		
		function formatItem(item) {
			// 递归处理子回复
			var levelTwo = [];
			if (item.levelTwoComment && Array.isArray(item.levelTwoComment)) {
				levelTwo = item.levelTwoComment.map(formatItem);
			}
			
			return {
				id: item.id || item.commentId,
				content: item.content,
				createTime: item.createtime || item.createTime,
				likeCount: item.likeCount,
				nickname: item.nickname || (item.author && item.author.nickname),
				headUrl: item.headUrl || (item.author && item.author.headUrl),
				ipLocation: item.ipLocation || '',
				// 新增：回复引用信息 (支持三级回复)
				replyCommentId: item.replyCommentId || (item.replyComment && item.replyComment.id) || '', 
				replyNickname: item.replyNickname || (item.replyComment && item.replyComment.nickname) || '',
				// 递归包含子回复
				levelTwoComment: levelTwo,
				expandCommentCount: item.expandCommentCount || 0
			};
		}
		
		return items.map(formatItem);
	}

	// 获取视频信息
	function getVideoInfo(store) {
		var info = { id: '', title: '' };
		
		// 1. 尝试从 store.feed 获取 (根据 Log Keys: [ "feed", ... ])
		if (store && store.feed) {
			info.id = store.feed.id || store.feed.objectId || store.feed.exportId || '';
			info.title = store.feed.description || store.feed.desc || '';
		}
		
		// 2. 尝试从 store.currentFeed 获取
		if (!info.id && store && store.currentFeed) {
			info.id = store.currentFeed.id || store.currentFeed.objectId || '';
			info.title = store.currentFeed.description || store.currentFeed.desc || '';
		}
		
		// 3. 尝试从 store.profile 获取
		if (!info.id && store && store.profile) {
			info.id = store.profile.id || store.profile.objectId || '';
			info.title = store.profile.description || store.profile.desc || '';
		}
		
		// 4. 尝试从 URL 获取
		if (!info.id) {
			// 匹配 /feed/export/ID 或 /feed/ID
			var match = window.location.pathname.match(/\/feed\/([^/?]+)/);
			if (match) info.id = match[1];
		}
		
		// 5. 尝试从 document title 获取
		if (!info.title) {
			info.title = document.title || '';
		}
		
		return info;
	}

	// 保存评论数据到后端
	function saveComments(comments, totalExpected) {
		if (!comments || comments.length === 0) return;
		
		var store = findFeedStore();
		var videoInfo = getVideoInfo(store);

		// 如果 ID 变了，说明切换了视频，重置计数
		if (videoInfo.id && currentFeedId && videoInfo.id !== currentFeedId) {
			console.log('[评论采集] 检测到视频切换: ' + currentFeedId + ' -> ' + videoInfo.id);
			lastCommentCount = 0;
		}
		currentFeedId = videoInfo.id;

		console.log('[评论采集] 发送数据: ' + comments.length + ' 条评论 (期望总数: ' + totalExpected + ') | ID: ' + videoInfo.id);

		fetch('/__wx_channels_api/save_comment_data', {
			method: 'POST',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({
				comments: comments,
				videoId: videoInfo.id,
				videoTitle: videoInfo.title,
				originalCommentCount: totalExpected || 0,
				timestamp: Date.now(),
				isFullUpdate: true
			})
		}).catch(function(err) {
			console.error('[评论采集] 保存失败:', err);
		});
	}

	// 触发保存（带防抖）
	function triggerSave(comments, totalCount) {
		if (saveDebounceTimer) clearTimeout(saveDebounceTimer);
		
		// 检查是否"完成"
		var isComplete = totalCount > 0 && comments.length >= totalCount;
		
		// 如果已完成，快速保存
		// 如果未完成，等待较长时间以合并更新，减少文件生成
		var delay = isComplete ? 1000 : 5000;
		
		saveDebounceTimer = setTimeout(function() {
			saveComments(comments, totalCount);
		}, delay);
	}

	// 尝试调用 Store 的加载更多方法 (增强版: 遍历所有Store查找)
	function tryTriggerStoreLoadMore(store) {
		// 常见的加载更多方法名
		var candidates = ['loadMoreComment', 'fetchComment', 'getCommentHere', 'nextPage', 'loadMore', 'fetchMore', 'loadNext', 'loadMoreData'];
		
		// 1. 如果传入的 explicit store 有效，先尝试它
		if (store) {
			if (checkAndCall(store, candidates, 'CurrentStore')) return true;
		}

		// 2. 扫描 Pinia 所有 Stores
		try {
			var app = document.querySelector('[data-v-app]') || document.getElementById('app');
			if (app) {
				var vue = app.__vue__ || app.__vueParentComponent || (app._vnode && app._vnode.component);
				var pinia = vue && vue.appContext && vue.appContext.config && vue.appContext.config.globalProperties && vue.appContext.config.globalProperties.$pinia;
				
				if (pinia && pinia._s) {
					// 遍历 Map
					var stores = pinia._s;
					var iterator = stores.keys();
					var result = iterator.next();
					while (!result.done) {
						var id = result.value;
						var s = stores.get(id);
						// console.log('[评论采集] 扫描Store: ' + id);
						
						// 检查是否包含 comment 相关数据，如果是，大概率是目标 store
						if (s.commentList || s.comments || (s.$state && s.$state.commentList)) {
							// console.log('[评论采集] 发现疑似目标Store: ' + id);
							if (checkAndCall(s, candidates, 'Store(' + id + ')')) return true;
						}
						
						result = iterator.next();
					}
				}
			}
		} catch (e) {
			// console.error('[评论采集] 扫描Store失败:', e);
		}
		
		return false;
	}

	// 辅助函数: 检查并调用方法
	function checkAndCall(obj, methods, contextName) {
		// 1. 直接查方法
		for (var i = 0; i < methods.length; i++) {
			var name = methods[i];
			if (typeof obj[name] === 'function') {
				// console.log('[评论采集] 调用 ' + contextName + ' 方法: ' + name);
				try {
					obj[name]();
					return true;
				} catch (e) {
					// console.error('[评论采集] 调用失败:', e);
				}
			}
		}
		
		// 2. 查 Actions (Pinia)
		if (obj._a || obj.$actions) {
			var actions = obj._a || obj.$actions;
			for (var i = 0; i < methods.length; i++) {
				var name = methods[i];
				if (typeof actions[name] === 'function') {
					// console.log('[评论采集] 调用 ' + contextName + ' Action: ' + name);
					try {
						actions[name]();
						return true;
					} catch (e) {
						// console.error('[评论采集] 调用失败:', e);
					}
				}
			}
		}
		
		return false;
	}

	// 查找并滚动评论容器
	function scrollCommentList() {
		// 1. 尝试找到包含评论的滚动容器
		var walkers = document.createTreeWalker(document.body, NodeFilter.SHOW_ELEMENT, {
			acceptNode: function(node) {
				// 忽略日志面板本身
				if (node.id === 'log-content' || node.classList.contains('log-window')) return NodeFilter.FILTER_SKIP;
				// 检查是否有滚动条
				var style = window.getComputedStyle(node);
				var isScrollable = (style.overflowY === 'auto' || style.overflowY === 'scroll') && node.scrollHeight > node.clientHeight;
				return isScrollable ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_SKIP;
			}
		});

		var node;
		var scrollableContainers = [];
		while(node = walkers.nextNode()) {
			scrollableContainers.push(node);
		}
		
		// 倒序遍历（通常我们需要最内层的滚动容器，或者根据包含内容判断）
		var found = false;
		for (var i = scrollableContainers.length - 1; i >= 0; i--) {
			var container = scrollableContainers[i];
			// 简单的判断：容器高度大于一定值，且包含一些文本
			if (container.scrollHeight > 300 && container.innerText.length > 50) {
				// 滚动它
				// console.log('[评论采集] 📜 滚动容器:', container.className || container.tagName);
				container.scrollTop = container.scrollHeight;
				found = true;
				// 不break，可能由多个嵌套容器需要滚动，或者我们不确定是哪一个，都滚一下
			}
		}
		
		if (!found) {
			window.scrollTo(0, document.body.scrollHeight);
		}
	}

	// 主逻辑：订阅 Store 变化
	function initObserver() {
		var store = findFeedStore();
		if (!store) {
			setTimeout(initObserver, 1000);
			return;
		}
		
		// 暴露 store 以便调试
		window.__wx_feed_store = store;
		console.log('[评论采集] Store 连接成功!');
		// if (store.commentList) {
		// 	console.log('[评论采集] commentList Keys:', Object.keys(store.commentList));
		// }
		
		// 订阅变化
		store.$subscribe(function(mutation, state) {
			if (state.commentList && state.commentList.dataList) {
				var items = state.commentList.dataList.items;
				
				// 尝试多处获取总数
				var total = 0;
				// 注意：在subscribe回调中，state可能会只有部分变动，所以尽量去读 store (proxy) 或者做好空值判断
				// 但 state.commentList 应该是完整的
				if (state.commentList.totalCount !== undefined) total = state.commentList.totalCount;
				else if (state.commentList.total !== undefined) total = state.commentList.total;
				// 回退到 store 实例获取 (state 可能不包含 feed)
				else if (store.feed && store.feed.commentCount !== undefined) total = store.feed.commentCount;
				
				// 只要数量变化，就触发（防抖）保存
				if (items.length !== lastCommentCount) {
					var stats = getCommentStats(items);
					console.log('[评论采集] 评论更新: ' + stats.total + ' (总数: ' + total + ')');
					lastCommentCount = items.length;
					noChangeCount = 0;
					
					var formatted = formatComments(items);
					triggerSave(formatted, total);
				} else {
				    noChangeCount++;
				}
			}
		}, { detached: true });
		
		isCollecting = true;
		startAutoScroll();
	}

	// 增强版自动滚动
	function startAutoScroll() {
		if (autoScrollTimer) clearInterval(autoScrollTimer);
		
		autoScrollTimer = setInterval(function() {
			var store = window.__wx_feed_store;
			
			// 1. 优先尝试调用 Store 的加载方法
			var calledStore = tryTriggerStoreLoadMore(store);
			
			// 2. 查找并滚动所有可能的容器
			var scrollableFound = false;
			var containers = document.querySelectorAll('.comment-list, .recycle-list, [class*="comments"], [class*="container"]');
			
			for (var i = 0; i < containers.length; i++) {
				var el = containers[i];
				// 检查是否可滚动
				if (el.scrollHeight > el.clientHeight) {
					// 滚动到底部
					el.scrollTop = el.scrollHeight;
					scrollableFound = true;
				}
			}
			
			if (!scrollableFound) {
				window.scrollTo(0, document.body.scrollHeight);
			}
			
			// 3. 点击"更多"按钮
			var buttons = document.querySelectorAll('div, span, p, button');
			for (var i = 0; i < buttons.length; i++) {
				var btn = buttons[i];
				var text = btn.innerText || '';
				if (text.includes('查看更多') || text.includes('展开更多') || text === '更多评论') {
					// console.log('[评论采集] 点击 "更多" 按钮');
					btn.click();
					break;
				}
			}
			
		}, 1000); // 1秒一次
	}

	// 辅助函数：获取评论统计信息
	function getCommentStats(items) {
		var total = 0;
		var topLevel = 0;
		var replies = 0;
		var missingReplies = 0; // 未展开的二级回复数量
		
		if (!items || !Array.isArray(items)) {
			return { total: 0, topLevel: 0, replies: 0, missingReplies: 0 };
		}

		items.forEach(function(item) {
			topLevel++;
			total++;
			if (item.levelTwoComment && Array.isArray(item.levelTwoComment)) {
				replies += item.levelTwoComment.length;
				total += item.levelTwoComment.length;
			}
			// 如果有 expandCommentCount 但 levelTwoComment 数量不匹配，说明有未展开的
			if (item.expandCommentCount > 0 && (!item.levelTwoComment || item.levelTwoComment.length < item.expandCommentCount)) {
				missingReplies += (item.expandCommentCount - (item.levelTwoComment ? item.levelTwoComment.length : 0));
			}
		});
		return { total: total, topLevel: topLevel, replies: replies, missingReplies: missingReplies };
	}




	// 校验二级评论展开情况
	function verifyCommentAllExpanded(items) {
		console.log('=== 二级评论展开校验报告 ===');
		var totalWithReplies = 0;
		var notFullyExpanded = 0;
		var totalExpected = 0;
		var totalActual = 0;

		items.forEach(function(item) {
			// 修正: 原始数据的字段通常是 expandCommentCount 或 replyCount
			var expected = item.expandCommentCount || item.replyCount || item.commentCount || 0;
			
			if (expected > 0) { // 预期有回复
				totalWithReplies++;
				totalExpected += expected;
				
				var actualReplies = 0;
				if (item.levelTwoComment && item.levelTwoComment.length > 0) {
					actualReplies = item.levelTwoComment.length;
				}
				totalActual += actualReplies;

				// 允许少量误差
				if (expected > actualReplies) { 
					console.warn('❌ [未完全展开] 用户: ' + (item.nickname||'') + ' | 预期: ' + expected + ' | 实际: ' + actualReplies + ' | 内容: ' + (item.content||'').substring(0, 30) + '...');
					notFullyExpanded++;
				} else {
					console.log('✅ [已展开] 用户: ' + (item.nickname||'') + ' | 回复数: ' + actualReplies);
				}
			}
		});

		console.log('--------------------------------');
		console.log('总计发现含回复评论: ' + totalWithReplies);
		console.log('未完全展开数: ' + notFullyExpanded);
		if (totalExpected > 0) {
			console.log('回复总完成率: ' + totalActual + '/' + totalExpected + ' (' + ((totalActual/totalExpected)*100).toFixed(1) + '%)');
		}
		console.log('================================');
		
		return {
			missingReplies: notFullyExpanded,
			totalWithReplies: totalWithReplies
		};
	}

	// 尝试展开二级评论
	function expandSecondaryComments() {
		var count = 0;

		// 辅助点击函数
		var clickNode = function(node, actionName) {
			try {
				if (actionName) console.log('[评论采集] ' + actionName + ':', node.innerText.trim().substring(0, 30));
				node.scrollIntoView({block: 'center', inline: 'nearest'});
				var eventTypes = ['mouseover', 'mousedown', 'mouseup', 'click'];
				for (var k = 0; k < eventTypes.length; k++) {
					var event = new MouseEvent(eventTypes[k], { 'view': window, 'bubbles': true, 'cancelable': true });
					node.dispatchEvent(event);
				}
				return true;
			} catch(e) {
				console.error('[评论采集] 点击失败:', e);
				return false;
			}
		};

		// 策略1: 精确查找 .load-more__btn (最准确)
		// 结构: .comment-item__extra -> .comment-reply-list + .load-more -> .click-box.load-more__btn
		var preciseCandidates = document.querySelectorAll('.load-more__btn, .click-box, .comment-item__extra .load-more');
		for (var i = 0; i < preciseCandidates.length; i++) {
			var node = preciseCandidates[i];
			if (node.offsetParent === null) continue; // 不可见
			var text = node.innerText || '';
			
			// 检查是否已处理过且文字未变 (防止DOM复用导致的漏点)
			if (node.classList.contains('expanded-handled') && node.getAttribute('data-handled-text') === text) {
				continue;
			}

			if (text.includes('回复') || text.includes('展开') || text.includes('更多')) {
				clickNode(node, '🎯 精确点击');
				node.classList.add('expanded-handled');
				node.setAttribute('data-handled-text', text); // 记录处理时的文字
				count++;
			}
		}

		// 策略2: 文本模糊查找 (防止DOM结构变化)
		if (count === 0) {
			var candidates = document.querySelectorAll('div, span, p, a'); 
			for (var i = 0; i < candidates.length; i++) {
				var node = candidates[i];
				var text = node.innerText || '';
				
				// 避免匹配过多无关元素
				if (text.length > 50 || text.length < 2) continue;

				// 宽松匹配：包含 "展开"、"回复"、"更多"
				if (text.includes('展开') || text.includes('回复') || text.includes('更多')) {
					// 排除无效元素
					if (node.offsetParent === null) continue;
					
					// 检查是否已处理过且文字未变
					if (node.classList.contains('expanded-handled') && node.getAttribute('data-handled-text') === text) {
						continue;
					}
					
					if (node.closest('#__wx_channels_log_panel')) continue; // 排除日志面板
					// if (node.closest('.load-more__btn')) continue; // REMOVED: 让策略2覆盖漏网之鱼

					// 尝试定位到最佳点击容器
					var clickTarget = node.closest('.click-box') || node.closest('.load-more') || node;
					
					clickNode(clickTarget, '🔍 模糊点击');
					
					node.classList.add('expanded-handled');
					node.setAttribute('data-handled-text', text);
					
					if (clickTarget !== node) {
						clickTarget.classList.add('expanded-handled');
						clickTarget.setAttribute('data-handled-text', text);
					}
					count++;
				}
			}
		}

		if (count > 0) {
			console.log('[评论采集] 本轮触发展开: ' + count + ' 个');
		}
		return count;
	}
	
	// 暴露手动启动函数 (供按钮调用)
	window.__wx_channels_start_comment_collection = function() {
		console.log('[评论采集] 初始化采集...');
		
		var store = findFeedStore();
		if (!store) {
			console.warn('[评论采集] 未找到Store');
			initObserver();
			return;
		}
		
		// 强制触发一次加载更多
		tryTriggerStoreLoadMore(store);
		
		if (store.commentList && store.commentList.dataList) {
			var items = store.commentList.dataList.items;
			
			// 尝试多处获取总数
			var total = 0;
			if (store.commentList.totalCount !== undefined) total = store.commentList.totalCount;
			else if (store.commentList.total !== undefined) total = store.commentList.total;
			else if (store.feed && store.feed.commentCount !== undefined) total = store.feed.commentCount;
			else if (store.profile && store.profile.commentCount !== undefined) total = store.profile.commentCount;
			
			var stats = getCommentStats(items);
			
			// 获取分页标记
			var lastBuffer = '';
			if (store.commentList.lastBuffer) {
				lastBuffer = store.commentList.lastBuffer;
			} else if (store.commentList.dataList && store.commentList.dataList.buffers && store.commentList.dataList.buffers.lastBuffer) {
				// 命中！根据日志分析，这是正确路径
				lastBuffer = store.commentList.dataList.buffers.lastBuffer;
			}
			
			var hasMore = !!lastBuffer;
			console.log('[评论采集] 采集概况: 已加载' + stats.total + '/' + total + ' (一级:' + stats.topLevel + ', 二级:' + stats.replies + ') | hasMore:' + hasMore);
			
			var formatted = formatComments(items);
			// 只有在没有更多或者用户取消采集时才保存
			// saveComments(formatted, total); 
			
			if (hasMore || stats.total < total) {
			    // 如果还有更多，询问是否继续加载
			    if (confirm('已发现 ' + stats.total + ' 条评论 (目标: ' + total + ')。\n检测到还有更多内容，是否自动采集全部？\n(包含自动点击"展开回复")')) {
			        var sameCountRetries = 0;
			        var loadLoop = setInterval(function() {
			            // 更新 buffer 获取逻辑
			            var currentBuffer = '';
			            if (store.commentList.lastBuffer) currentBuffer = store.commentList.lastBuffer;
			            else if (store.commentList.dataList && store.commentList.dataList.buffers && store.commentList.dataList.buffers.lastBuffer) {
			                currentBuffer = store.commentList.dataList.buffers.lastBuffer;
			            }
			            
			            var currentStats = getCommentStats(store.commentList.dataList.items);
			            
			            // 终止条件1: 已加载数达到或超过总数 (无论是否有 Buffer)
			            if (currentStats.total >= total) {
			                console.log('[评论采集] 数量已达标，采集完成');
			                clearInterval(loadLoop);
							
							// 强制保存最终结果
							var finalItems = store.commentList.dataList.items;
							var finalFormatted = formatComments(finalItems);
							saveComments(finalFormatted, total);

							// 输出详细的二级回复报告
							verifyCommentAllExpanded(finalItems);

			                alert('采集完成！\n总计: ' + currentStats.total + '\n一级: ' + currentStats.topLevel + '\n二级: ' + currentStats.replies);
			                return;
			            }
			            
			            // 检查是否卡死 (增加重试次数到 10)
			            if (currentStats.total === stats.total) { // stats.total 是上一次循环的值
			                sameCountRetries++;
			                
							// 在重试期间尝试展开二级评论
							expandSecondaryComments();

			                if (sameCountRetries > 10) {
			                    clearInterval(loadLoop);
								
								// 强制保存最终结果 (即使是不完整的)
								var finalItems = store.commentList.dataList.items;
								var finalFormatted = formatComments(finalItems);
								saveComments(finalFormatted, total);

								// 输出详细的二级回复报告
								verifyCommentAllExpanded(finalItems);

								var msg = '采集停止：多次重试无新增数据。\n' +
								          '当前: ' + currentStats.total + '/' + total + '\n';
								
								if (currentStats.missingReplies > 0) {
									msg += '\n⚠️ 仍有约 ' + currentStats.missingReplies + ' 条二级回复可能未展开。';
								}
								msg += '\n(已尝试自动保存当前数据)';
								
			                    alert(msg);
			                    return;
			                }
			            } else {
			                stats = currentStats; // 更新基准
			                sameCountRetries = 0; // 重置计数
							
							// 策略调整：优先处理二级评论展开，必须等所有展开点完再滚动
							// 这样可以防止滚动过快导致"查看更多"按钮消失或未被点击
							var expandedCount = expandSecondaryComments();
							if (expandedCount > 0) {
								console.log('[评论采集] 正在展开 ' + expandedCount + ' 个回复，暂停主列表滚动...');
								return; // 跳过本次循环的后续步骤 (Scroll/LoadMore)
							}
			            }
			            
			            console.log('[评论采集] 采集中... ' + currentStats.total + '/' + total);
			            
			            // 触发加载 (API + 滚动)
			            tryTriggerStoreLoadMore(store);
			            scrollCommentList();
						
			        }, 1500 + Math.random() * 1000); // 1.5-2.5秒间隔
			    } else {
					// 用户选择不继续，保存当前数据
					saveComments(formatted, total);
					alert('已保存当前采集的 ' + stats.total + ' 条评论。');
				}
			} else {
				saveComments(formatted, total);
			    alert('正在保存评论...\n已加载: ' + stats.total + '\n总数: ' + total + '\n(已全部加载完成)');
			}
		} else {
			console.warn('[评论采集] Store中没有评论数据');
			alert('未检测到评论数据，请确保已打开评论区');
		}
	};

	if (document.readyState === 'complete') {
		initObserver();
	} else {
		window.addEventListener('load', initObserver);
	}
	setTimeout(initObserver, 5000);

})();
</script>`
}

// getLogPanelScript 获取日志面板脚本，用于在页面上显示日志（替代控制台）
func (h *ScriptHandler) getLogPanelScript() string {
	// 根据配置决定是否显示日志按钮
	showLogButton := "false"
	if h.getConfig().ShowLogButton {
		showLogButton = "true"
	}

	return `<script>
// 日志按钮显示配置
window.__wx_channels_show_log_button__ = ` + showLogButton + `;
</script>
<script>
(function() {
	'use strict';
	
	// 防止重复初始化
	if (window.__wx_channels_log_panel_initialized__) {
		return;
	}
	window.__wx_channels_log_panel_initialized__ = true;
	
	// 日志存储
	const logStore = {
		logs: [],
		maxLogs: 500, // 最多保存500条日志
		addLog: function(level, args) {
			const timestamp = new Date().toLocaleTimeString('zh-CN', { hour12: false });
			const message = Array.from(args).map(arg => {
				if (typeof arg === 'object') {
					try {
						return JSON.stringify(arg, null, 2);
					} catch (e) {
						return String(arg);
					}
				}
				return String(arg);
			}).join(' ');
			
			this.logs.push({
				level: level,
				message: message,
				timestamp: timestamp
			});
			
			// 限制日志数量
			if (this.logs.length > this.maxLogs) {
				this.logs.shift();
			}
			
			// 更新面板显示
			if (window.__wx_channels_log_panel) {
				window.__wx_channels_log_panel.updateDisplay();
			}
		},
		clear: function() {
			this.logs = [];
			if (window.__wx_channels_log_panel) {
				window.__wx_channels_log_panel.updateDisplay();
			}
		}
	};
	
	// 创建日志面板
	function createLogPanel() {
		const panel = document.createElement('div');
		panel.id = '__wx_channels_log_panel';
		// 检测是否为移动设备
		const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent) || window.innerWidth < 768;
		
		// 面板位置：在按钮旁边，向上展开
		const btnBottom = isMobile ? 80 : 20;
		const btnLeft = isMobile ? 15 : 20;
		const btnSize = isMobile ? 56 : 50;
		const panelWidth = isMobile ? 'calc(100% - 30px)' : '400px';
		const panelMaxWidth = isMobile ? '100%' : '500px';
		const panelMaxHeight = isMobile ? 'calc(100vh - ' + (btnBottom + btnSize + 20) + 'px)' : '500px';
		const panelFontSize = isMobile ? '11px' : '12px';
		const panelBottom = btnBottom + btnSize + 10; // 按钮上方10px
		
		panel.style.cssText = 'position: fixed;' +
			'bottom: ' + panelBottom + 'px;' +
			'left: ' + btnLeft + 'px;' +
			'width: ' + panelWidth + ';' +
			'max-width: ' + panelMaxWidth + ';' +
			'max-height: ' + panelMaxHeight + ';' +
			'height: 0;' +
			'background: rgba(0, 0, 0, 0.95);' +
			'border: 1px solid #333;' +
			'border-radius: 8px 8px 0 0;' +
			'box-shadow: 0 -4px 12px rgba(0, 0, 0, 0.5);' +
			'z-index: 999999;' +
			'font-family: "Consolas", "Monaco", "Courier New", monospace;' +
			'font-size: ' + panelFontSize + ';' +
			'color: #fff;' +
			'display: none;' +
			'flex-direction: column;' +
			'overflow: hidden;' +
			'transition: height 0.3s ease, opacity 0.3s ease;' +
			'opacity: 0;';
		
		// 标题栏
		const header = document.createElement('div');
		header.style.cssText = 'background: #1a1a1a;' +
			'padding: 8px 12px;' +
			'border-bottom: 1px solid #333;' +
			'display: flex;' +
			'justify-content: space-between;' +
			'align-items: center;' +
			'cursor: move;' +
			'user-select: none;';
		
		const title = document.createElement('span');
		title.textContent = '📋 日志面板';
		title.style.cssText = 'font-weight: bold; color: #4CAF50;';
		
		const controls = document.createElement('div');
		controls.style.cssText = 'display: flex; gap: 8px;';
		
		// 清空按钮
		const clearBtn = document.createElement('button');
		clearBtn.textContent = '清空';
		clearBtn.style.cssText = 'background: #f44336;' +
			'color: white;' +
			'border: none;' +
			'padding: 4px 12px;' +
			'border-radius: 4px;' +
			'cursor: pointer;' +
			'font-size: 11px;';
		clearBtn.onclick = function(e) {
			e.stopPropagation();
			logStore.clear();
		};
		
		// 复制日志按钮
		const copyBtn = document.createElement('button');
		copyBtn.textContent = '复制';
		copyBtn.style.cssText = 'background: #4CAF50;' +
			'color: white;' +
			'border: none;' +
			'padding: 4px 12px;' +
			'border-radius: 4px;' +
			'cursor: pointer;' +
			'font-size: 11px;';
		copyBtn.onclick = function(e) {
			e.stopPropagation();
			try {
				// 构建日志文本
				var logText = '';
				logStore.logs.forEach(function(log) {
					var levelPrefix = '';
					switch(log.level) {
						case 'log': levelPrefix = '[LOG]'; break;
						case 'info': levelPrefix = '[INFO]'; break;
						case 'warn': levelPrefix = '[WARN]'; break;
						case 'error': levelPrefix = '[ERROR]'; break;
						default: levelPrefix = '[LOG]';
					}
					logText += '[' + log.timestamp + '] ' + levelPrefix + ' ' + log.message + '\n';
				});
				
				if (logText === '') {
					alert('日志为空，无需复制');
					return;
				}
				
				// 使用 Clipboard API 复制
				if (navigator.clipboard && navigator.clipboard.writeText) {
					navigator.clipboard.writeText(logText).then(function() {
						copyBtn.textContent = '已复制';
						setTimeout(function() {
							copyBtn.textContent = '复制';
						}, 2000);
					}).catch(function(err) {
						console.error('复制失败:', err);
						// 降级方案：使用传统方法
						copyToClipboardFallback(logText);
					});
				} else {
					// 降级方案：使用传统方法
					copyToClipboardFallback(logText);
				}
			} catch (error) {
				console.error('复制日志失败:', error);
				alert('复制失败: ' + error.message);
			}
		};
		
		// 复制到剪贴板的降级方案
		function copyToClipboardFallback(text) {
			var textArea = document.createElement('textarea');
			textArea.value = text;
			textArea.style.position = 'fixed';
			textArea.style.top = '-999px';
			textArea.style.left = '-999px';
			document.body.appendChild(textArea);
			textArea.select();
			try {
				var successful = document.execCommand('copy');
				if (successful) {
					copyBtn.textContent = '已复制';
					setTimeout(function() {
						copyBtn.textContent = '复制';
					}, 2000);
				} else {
					alert('复制失败，请手动选择文本复制');
				}
			} catch (err) {
				console.error('复制失败:', err);
				alert('复制失败: ' + err.message);
			}
			document.body.removeChild(textArea);
		}
		
		// 导出日志按钮
		const exportBtn = document.createElement('button');
		exportBtn.textContent = '导出';
		exportBtn.style.cssText = 'background: #FF9800;' +
			'color: white;' +
			'border: none;' +
			'padding: 4px 12px;' +
			'border-radius: 4px;' +
			'cursor: pointer;' +
			'font-size: 11px;';
		exportBtn.onclick = function(e) {
			e.stopPropagation();
			try {
				// 构建日志文本
				var logText = '';
				logStore.logs.forEach(function(log) {
					var levelPrefix = '';
					switch(log.level) {
						case 'log': levelPrefix = '[LOG]'; break;
						case 'info': levelPrefix = '[INFO]'; break;
						case 'warn': levelPrefix = '[WARN]'; break;
						case 'error': levelPrefix = '[ERROR]'; break;
						default: levelPrefix = '[LOG]';
					}
					logText += '[' + log.timestamp + '] ' + levelPrefix + ' ' + log.message + '\n';
				});
				
				if (logText === '') {
					alert('日志为空，无需导出');
					return;
				}
				
				// 创建 Blob 并下载
				var blob = new Blob([logText], { type: 'text/plain;charset=utf-8' });
				var url = URL.createObjectURL(blob);
				var a = document.createElement('a');
				var timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, -5);
				a.href = url;
				a.download = 'wx_channels_logs_' + timestamp + '.txt';
				document.body.appendChild(a);
				a.click();
				document.body.removeChild(a);
				URL.revokeObjectURL(url);
				
				exportBtn.textContent = '已导出';
				setTimeout(function() {
					exportBtn.textContent = '导出';
				}, 2000);
			} catch (error) {
				console.error('导出日志失败:', error);
				alert('导出失败: ' + error.message);
			}
		};
		
		// 最小化/最大化按钮
		const toggleBtn = document.createElement('button');
		toggleBtn.textContent = '−';
		toggleBtn.style.cssText = 'background: #2196F3;' +
			'color: white;' +
			'border: none;' +
			'padding: 4px 12px;' +
			'border-radius: 4px;' +
			'cursor: pointer;' +
			'font-size: 11px;';
		toggleBtn.onclick = function(e) {
			e.stopPropagation();
			const content = panel.querySelector('.log-content');
			if (content.style.display === 'none') {
				content.style.display = 'flex';
				toggleBtn.textContent = '−';
			} else {
				content.style.display = 'none';
				toggleBtn.textContent = '+';
			}
		};
		
		// 关闭按钮
		const closeBtn = document.createElement('button');
		closeBtn.textContent = '×';
		closeBtn.style.cssText = 'background: #666;' +
			'color: white;' +
			'border: none;' +
			'padding: 4px 12px;' +
			'border-radius: 4px;' +
			'cursor: pointer;' +
			'font-size: 14px;' +
			'line-height: 1;';
		closeBtn.onclick = function(e) {
			e.stopPropagation();
			panel.style.display = 'none';
		};
		
		controls.appendChild(clearBtn);
		controls.appendChild(copyBtn);
		controls.appendChild(exportBtn);
		controls.appendChild(toggleBtn);
		controls.appendChild(closeBtn);
		header.appendChild(title);
		header.appendChild(controls);
		
		// 日志内容区域
		const content = document.createElement('div');
		content.className = 'log-content';
		content.style.cssText = 'flex: 1;' +
			'overflow-y: auto;' +
			'padding: 8px;' +
			'display: flex;' +
			'flex-direction: column;' +
			'gap: 2px;';
		
		// 滚动条样式
		content.style.scrollbarWidth = 'thin';
		content.style.scrollbarColor = '#555 #222';
		
		// 更新显示
		function updateDisplay() {
			content.innerHTML = '';
			logStore.logs.forEach(log => {
				const logItem = document.createElement('div');
				logItem.style.cssText = 'padding: 4px 8px;' +
					'border-radius: 4px;' +
					'word-break: break-all;' +
					'line-height: 1.4;' +
					'background: rgba(255, 255, 255, 0.05);';
				
				// 根据日志级别设置颜色
				let levelColor = '#fff';
				let levelPrefix = '';
				switch(log.level) {
					case 'log':
						levelColor = '#4CAF50';
						levelPrefix = '[LOG]';
						break;
					case 'info':
						levelColor = '#2196F3';
						levelPrefix = '[INFO]';
						break;
					case 'warn':
						levelColor = '#FF9800';
						levelPrefix = '[WARN]';
						break;
					case 'error':
						levelColor = '#f44336';
						levelPrefix = '[ERROR]';
						logItem.style.background = 'rgba(244, 67, 54, 0.2)';
						break;
					default:
						levelPrefix = '[LOG]';
				}
				
				logItem.innerHTML = '<span style="color: #888; font-size: 10px;">[' + log.timestamp + ']</span>' +
					'<span style="color: ' + levelColor + '; font-weight: bold; margin: 0 4px;">' + levelPrefix + '</span>' +
					'<span style="color: #fff;">' + escapeHtml(log.message) + '</span>';
				
				content.appendChild(logItem);
			});
			
			// 自动滚动到底部
			content.scrollTop = content.scrollHeight;
		}
		
		// HTML转义
		function escapeHtml(text) {
			const div = document.createElement('div');
			div.textContent = text;
			return div.innerHTML;
		}
		
		panel.appendChild(header);
		panel.appendChild(content);
		document.body.appendChild(panel);
		
		// 移除拖拽功能，面板位置固定在按钮旁边
		
		// 计算面板高度
		function getPanelHeight() {
			// 临时显示以计算高度
			const wasHidden = panel.style.display === 'none';
			if (wasHidden) {
				panel.style.display = 'flex';
				panel.style.height = 'auto';
				panel.style.opacity = '0';
			}
			
			const maxHeight = parseInt(panel.style.maxHeight) || 500;
			const headerHeight = header.offsetHeight || 40;
			const contentHeight = content.scrollHeight || 0;
			const totalHeight = headerHeight + contentHeight + 16; // 16px padding
			const finalHeight = Math.min(maxHeight, totalHeight);
			
			if (wasHidden) {
				panel.style.display = 'none';
				panel.style.height = '0';
			}
			
			return finalHeight;
		}
		
		// 暴露更新方法
		window.__wx_channels_log_panel = {
			panel: panel,
			updateDisplay: updateDisplay,
			show: function() {
				panel.style.display = 'flex';
				// 使用requestAnimationFrame确保DOM已更新
				requestAnimationFrame(function() {
					const targetHeight = getPanelHeight();
					panel.style.height = targetHeight + 'px';
					panel.style.opacity = '1';
				});
			},
			hide: function() {
				panel.style.height = '0';
				panel.style.opacity = '0';
				// 动画结束后隐藏
				setTimeout(function() {
					if (panel.style.opacity === '0') {
						panel.style.display = 'none';
					}
				}, 300);
			},
			toggle: function() {
				if (panel.style.display === 'none' || panel.style.opacity === '0') {
					this.show();
				} else {
					this.hide();
				}
			}
		};
	}
	
	// 保存原始的console方法
	const originalConsole = {
		log: console.log.bind(console),
		info: console.info.bind(console),
		warn: console.warn.bind(console),
		error: console.error.bind(console),
		debug: console.debug.bind(console)
	};
	
	// 重写console方法
	console.log = function(...args) {
		originalConsole.log.apply(console, args);
		logStore.addLog('log', args);
	};
	
	console.info = function(...args) {
		originalConsole.info.apply(console, args);
		logStore.addLog('info', args);
	};
	
	console.warn = function(...args) {
		originalConsole.warn.apply(console, args);
		logStore.addLog('warn', args);
	};
	
	console.error = function(...args) {
		originalConsole.error.apply(console, args);
		logStore.addLog('error', args);
	};
	
	console.debug = function(...args) {
		originalConsole.debug.apply(console, args);
		logStore.addLog('log', args);
	};
	
	// 创建浮动触发按钮（用于微信浏览器等无法使用快捷键的场景）
	function createToggleButton() {
		const btn = document.createElement('div');
		btn.id = '__wx_channels_log_toggle_btn';
		btn.innerHTML = '📋';
		// 检测是否为移动设备
		const isMobileBtn = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent) || window.innerWidth < 768;
		
		const btnBottom = isMobileBtn ? '80px' : '20px';
		const btnLeft = isMobileBtn ? '15px' : '20px';
		const btnWidth = isMobileBtn ? '56px' : '50px';
		const btnHeight = isMobileBtn ? '56px' : '50px';
		const btnFontSize = isMobileBtn ? '28px' : '24px';
		
		btn.style.cssText = 'position: fixed;' +
			'bottom: ' + btnBottom + ';' +
			'left: ' + btnLeft + ';' +
			'width: ' + btnWidth + ';' +
			'height: ' + btnHeight + ';' +
			'background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);' +
			'border-radius: 50%;' +
			'box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);' +
			'z-index: 999998;' +
			'cursor: pointer;' +
			'display: flex;' +
			'align-items: center;' +
			'justify-content: center;' +
			'font-size: ' + btnFontSize + ';' +
			'user-select: none;' +
			'transition: all 0.3s ease;' +
			'border: 2px solid rgba(255, 255, 255, 0.3);' +
			'touch-action: manipulation;' +
			'-webkit-tap-highlight-color: transparent;';
		
		btn.addEventListener('mouseenter', function() {
			btn.style.transform = 'scale(1.1)';
			btn.style.boxShadow = '0 6px 16px rgba(0, 0, 0, 0.4)';
		});
		
		btn.addEventListener('mouseleave', function() {
			btn.style.transform = 'scale(1)';
			btn.style.boxShadow = '0 4px 12px rgba(0, 0, 0, 0.3)';
		});
		
		// 切换面板显示的函数
		function togglePanel() {
			if (window.__wx_channels_log_panel) {
				const isVisible = window.__wx_channels_log_panel.panel.style.display !== 'none' && 
				                  window.__wx_channels_log_panel.panel.style.opacity !== '0';
				window.__wx_channels_log_panel.toggle();
				// 延迟更新按钮状态，等待动画完成
				setTimeout(function() {
					const nowVisible = window.__wx_channels_log_panel.panel.style.display !== 'none' && 
					                  window.__wx_channels_log_panel.panel.style.opacity !== '0';
					if (nowVisible) {
						btn.style.opacity = '1';
						btn.title = '点击隐藏日志面板';
					} else {
						btn.style.opacity = '0.6';
						btn.title = '点击显示日志面板';
					}
				}, 100);
			}
		}
		
		// 支持点击和触摸事件
		btn.addEventListener('click', togglePanel);
		btn.addEventListener('touchend', function(e) {
			e.preventDefault();
			togglePanel();
		});
		
		btn.title = '点击显示/隐藏日志面板';
		document.body.appendChild(btn);
		
		// 初始状态：面板默认不显示，按钮半透明
		btn.style.opacity = '0.6';
	}
	
	// 页面加载完成后创建面板和按钮
	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', function() {
			createLogPanel();
			// 根据配置决定是否创建日志按钮
			if (window.__wx_channels_show_log_button__) {
				createToggleButton();
			}
		});
	} else {
		createLogPanel();
		// 根据配置决定是否创建日志按钮
		if (window.__wx_channels_show_log_button__) {
			createToggleButton();
		}
	}
	
	// 添加快捷键：Ctrl+Shift+L 显示/隐藏日志面板（桌面浏览器可用）
	document.addEventListener('keydown', function(e) {
		if (e.ctrlKey && e.shiftKey && e.key === 'L') {
			e.preventDefault();
			if (window.__wx_channels_log_panel) {
				window.__wx_channels_log_panel.toggle();
				// 同步更新按钮状态
				const btn = document.getElementById('__wx_channels_log_toggle_btn');
				if (btn) {
					setTimeout(function() {
						const isVisible = window.__wx_channels_log_panel.panel.style.display !== 'none' && 
						                  window.__wx_channels_log_panel.panel.style.opacity !== '0';
						if (isVisible) {
							btn.style.opacity = '1';
						} else {
							btn.style.opacity = '0.6';
						}
					}, 100);
				}
			}
		}
	});
	
	// 面板默认不显示，需要点击按钮才会显示
})();
</script>`
}

// saveJavaScriptFile 保存页面加载的 JavaScript 文件到本地以便分析
func (h *ScriptHandler) saveJavaScriptFile(path string, content []byte) {
	// 检查是否启用JS文件保存
	if h.getConfig() != nil && !h.getConfig().SavePageJS {
		return
	}

	// 只保存 .js 文件
	if !strings.HasSuffix(strings.Split(path, "?")[0], ".js") {
		return
	}

	// 获取基础目录
	baseDir, err := utils.GetBaseDir()
	if err != nil {
		return
	}

	// 根据JS文件路径识别页面类型
	pageType := "common"
	pathLower := strings.ToLower(path)
	if strings.Contains(pathLower, "home") || strings.Contains(pathLower, "finderhome") {
		pageType = "home"
	} else if strings.Contains(pathLower, "profile") {
		pageType = "profile"
	} else if strings.Contains(pathLower, "feed") {
		pageType = "feed"
	} else if strings.Contains(pathLower, "search") {
		pageType = "search"
	} else if strings.Contains(pathLower, "live") {
		pageType = "live"
	}

	// 创建按页面类型分类的保存目录
	jsDir := filepath.Join(baseDir, h.getConfig().DownloadsDir, "cached_js", pageType)
	if err := utils.EnsureDir(jsDir); err != nil {
		return
	}

	// 从路径中提取文件名
	fileName := filepath.Base(path)
	if fileName == "" || fileName == "." || fileName == "/" {
		fileName = strings.ReplaceAll(path, "/", "_")
		fileName = strings.ReplaceAll(fileName, "\\", "_")
	}

	// 移除版本号后缀（如 .js?v=xxx）
	fileName = strings.Split(fileName, "?")[0]

	// 检查文件是否已存在（避免重复保存相同内容）
	filePath := filepath.Join(jsDir, fileName)
	if _, err := os.Stat(filePath); err == nil {
		// 文件已存在，跳过
		return
	}

	// 保存文件
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		utils.LogInfo("[JS保存] 保存失败: %s - %v", fileName, err)
		return
	}

	utils.LogInfo("[JS保存] ✅ 已保存: %s/%s", pageType, fileName)
}
