package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/qtgolang/SunnyNet/SunnyNet"
	"github.com/qtgolang/SunnyNet/public"

	"wx_channel/internal/api"
	"wx_channel/internal/assets"
	"wx_channel/internal/config"
	"wx_channel/internal/database"
	"wx_channel/internal/handlers"
	"wx_channel/internal/router"
	"wx_channel/internal/services"
	"wx_channel/internal/storage"
	"wx_channel/internal/utils"
	"wx_channel/internal/websocket"
	"wx_channel/pkg/certificate"
	"wx_channel/pkg/proxy"
)

// App 结构体，用于保存依赖项和状态
type App struct {
	Sunny          *SunnyNet.Sunny
	Cfg            *config.Config
	Version        string
	Port           int
	CurrentPageURL string
	LogInitMsg     string

	// 管理器
	FileManager *storage.FileManager

	// 处理器
	APIHandler        *handlers.APIHandler
	UploadHandler     *handlers.UploadHandler
	RecordHandler     *handlers.RecordHandler
	ScriptHandler     *handlers.ScriptHandler
	BatchHandler      *handlers.BatchHandler
	CommentHandler    *handlers.CommentHandler
	ConsoleAPIHandler *handlers.ConsoleAPIHandler
	WebSocketHandler  *handlers.WebSocketHandler
	StaticFileHandler *handlers.StaticFileHandler

	// 服务
	WSHub         *websocket.Hub
	SearchService *api.SearchService
	GopeedService *services.GopeedService // Add GopeedService

	// 路由器
	APIRouter *router.APIRouter

	// 拦截器
	requestInterceptors  []router.Interceptor
	responseInterceptors []router.Interceptor
}

// 全局变量，用于将 SunnyNet C 风格回调桥接到 App 方法
var globalApp *App

// NewApp 创建并初始化一个新的 App 实例
func NewApp(cfgParam *config.Config) *App {
	app := &App{
		Sunny:   SunnyNet.NewSunny(),
		Cfg:     cfgParam,
		Version: "?t=" + cfgParam.Version,
		Port:    cfgParam.Port,
	}

	// 设置全局实例用于回调桥接
	globalApp = app

	// 初始化日志
	app.printTitle()
	utils.LogConfigLoad("config.yaml", true)
	if app.Cfg.LogFile != "" {
		_ = utils.InitLoggerWithRotation(utils.INFO, app.Cfg.LogFile, app.Cfg.MaxLogSizeMB)
		app.LogInitMsg = fmt.Sprintf("日志已初始化: %s (最大 %dMB)", app.Cfg.LogFile, app.Cfg.MaxLogSizeMB)
	}

	// 尽早初始化 WebSocket Hub，以确保它对 APIRouter 可用
	app.WSHub = websocket.NewHub()

	return app
}

// initDownloadRecords 初始化下载记录系统
func (app *App) initDownloadRecords() error {
	downloadsDir, err := utils.ResolveDownloadDir(app.Cfg.DownloadsDir)
	if err != nil {
		return fmt.Errorf("解析下载目录失败: %v", err)
	}

	app.FileManager, err = storage.NewFileManager(downloadsDir)
	if err != nil {
		return fmt.Errorf("创建文件管理器失败: %v", err)
	}

	// Initialize Database
	dbPath := filepath.Join(downloadsDir, "records.db")
	if err := database.Initialize(&database.Config{DBPath: dbPath}); err != nil {
		return fmt.Errorf("初始化数据库失败: %v", err)
	}

	// Initialize Gopeed Service
	app.GopeedService = services.NewGopeedService(downloadsDir)
	// app.GopeedService.Start() // Removed

	return nil
}

// Run 启动应用
func (app *App) Run() {
	os_env := runtime.GOOS

	// 确保端口设置正确
	app.Sunny.SetPort(app.Port)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signalChan
		color.Red("\n正在关闭服务...%v\n\n", sig)
		utils.LogSystemShutdown(fmt.Sprintf("收到信号: %v", sig))
		database.Close()
		if os_env == "darwin" {
			proxy.DisableProxyInMacOS(proxy.ProxySettings{
				Device:   "",
				Hostname: "127.0.0.1",
				Port:     strconv.Itoa(app.Port),
			})
		}
		os.Exit(0)
	}()

	// 启动时检查更新 (移到这里以确保尽早执行)
	go func() {
		time.Sleep(2 * time.Second) // 缩短等待时间
		utils.Info("正在检查更新...")
		vService := services.NewVersionService()
		result, err := vService.CheckUpdate()
		if err != nil {
			utils.Warn("检查更新失败: %v", err)
			return
		}

		if result.HasUpdate {
			utils.PrintSeparator()
			color.Green("🚀 发现新版本 available: v%s", result.LatestVersion)
			color.Green("⬇️ 下载地址: %s", result.DownloadURL)
			utils.PrintSeparator()
		} else {
			utils.PrintSeparator()
			color.Green("✅ 当前已是最新版本: v%s", result.CurrentVersion)
			utils.PrintSeparator()
		}
	}()

	if err := app.initDownloadRecords(); err != nil {
		utils.HandleError(err, "初始化下载记录系统")
	} else {
		if app.LogInitMsg != "" {
			utils.Info(app.LogInitMsg)
			app.LogInitMsg = ""
		}
	}

	app.printEnvConfig()

	app.ConsoleAPIHandler = handlers.NewConsoleAPIHandler(app.Cfg, app.WSHub)
	app.WebSocketHandler = handlers.NewWebSocketHandler()

	// 初始化新的 API 路由器
	app.APIRouter = router.NewAPIRouter(app.Cfg, app.WSHub, app.Sunny)

	// 初始化静态文件处理器
	app.StaticFileHandler = handlers.NewStaticFileHandler()

	// 初始化业务处理器
	app.APIHandler = handlers.NewAPIHandler(app.Cfg)
	app.UploadHandler = handlers.NewUploadHandler(app.Cfg, app.WSHub, app.GopeedService)
	app.RecordHandler = handlers.NewRecordHandler(app.Cfg)
	app.CommentHandler = handlers.NewCommentHandler(app.Cfg)

	// BatchHandler (Injecting GopeedService)
	app.BatchHandler = handlers.NewBatchHandler(app.Cfg, app.GopeedService)

	// ScriptHandler
	app.ScriptHandler = handlers.NewScriptHandler(
		app.Cfg,
		assets.CoreJS,
		assets.DecryptJS,
		assets.DownloadJS,
		assets.HomeJS,
		assets.FeedJS,
		assets.ProfileJS,
		assets.SearchJS,
		assets.BatchDownloadJS,
		assets.ZipJS,
		assets.FileSaverJS,
		assets.MittJS,
		assets.EventbusJS,
		assets.UtilsJS,
		assets.APIClientJS,
		app.Version,
	)

	// 初始化拦截器
	app.requestInterceptors = []router.Interceptor{
		app.StaticFileHandler,
		app.APIRouter,
		app.APIHandler,
		app.UploadHandler,
		app.RecordHandler,
		app.BatchHandler,
		app.CommentHandler,
	}
	app.responseInterceptors = []router.Interceptor{
		app.ScriptHandler,
	}

	existing, err1 := certificate.CheckCertificate("SunnyNet")
	if err1 != nil {
		utils.HandleError(err1, "检查证书")
		utils.Warn("程序将继续运行，但HTTPS功能可能受限...")
		existing = false
	} else if !existing {
		utils.Info("正在安装证书...")
		err := certificate.InstallCertificate(assets.CertData)
		time.Sleep(app.Cfg.CertInstallDelay)
		if err != nil {
			utils.HandleError(err, "证书安装")
			utils.Warn("如需完整功能，请手动安装证书或以管理员身份运行程序。")

			if app.FileManager != nil {
				downloadsDir, err := utils.ResolveDownloadDir(app.Cfg.DownloadsDir)
				if err == nil {
					certPath := filepath.Join(downloadsDir, app.Cfg.CertFile)
					if err := utils.EnsureDir(downloadsDir); err == nil {
						if err := os.WriteFile(certPath, assets.CertData, 0644); err == nil {
							utils.Info("证书文件已保存到: %s", certPath)
						}
					}
				}
			}
		} else {
			utils.Info("✓ 证书安装成功！")
		}
	} else {
		utils.Info("✓ 证书已存在，无需重新安装。")
	}

	app.Sunny.SetGoCallback(GlobalHttpCallback, nil, nil, nil)
	sunnyErr := app.Sunny.Start().Error
	if sunnyErr != nil {
		utils.HandleError(sunnyErr, "启动代理服务")
		utils.Warn("按 Ctrl+C 退出...")
		select {}
	}

	proxy_server := fmt.Sprintf("127.0.0.1:%v", app.Port)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: "http",
				Host:   proxy_server,
			}),
		},
		Timeout: 5 * time.Second, // 设置超时防止阻塞
	}
	_, err3 := client.Get("https://sunny.io/")
	if err3 == nil {
		if os_env == "windows" {
			ok := app.Sunny.StartProcess()
			if !ok {
				color.Red("\nERROR 启动进程代理失败，检查是否以管理员身份运行\n")
				color.Yellow("按 Ctrl+C 退出...\n")
				select {}
			}
			app.Sunny.ProcessAddName("WeChatAppEx.exe")
		}

		utils.PrintSeparator()
		color.Blue("📡 服务状态信息")
		utils.PrintSeparator()
		utils.PrintLabelValue("⏳", "服务状态", "已启动")
		utils.PrintLabelValue("🔌", "代理端口", app.Port)
		utils.PrintLabelValue("📱", "支持平台", "微信视频号")

		proxyMode := "进程代理"
		if os_env != "windows" {
			proxyMode = "系统代理"
		}
		utils.LogSystemStart(app.Port, proxyMode)

		// Start WebSocket Hub (Now initialized earlier)
		go app.WSHub.Run()
		utils.Info("✓ WebSocket Hub 已启动")

		wsPort := app.Port + 1
		go app.startWebSocketServer(wsPort)

		utils.Info("🔍 请打开需要下载的视频号页面进行下载")
	} else {
		utils.PrintSeparator()
		utils.Warn("⚠️ 您还未安装证书，请在浏览器打开 http://%v 并根据说明安装证书", proxy_server)
		utils.Warn("⚠️ 在安装完成后重新启动此程序即可")
		utils.PrintSeparator()
	}
	utils.Info("💡 服务正在运行，按 Ctrl+C 退出...")

	// 启动时检查更新 - 已移动到 Run 函数开头

	select {}
}

// GlobalHttpCallback 桥接到单例 app 实例
func GlobalHttpCallback(Conn *SunnyNet.HttpConn) {
	if globalApp != nil {
		globalApp.HandleRequest(Conn)
	}
}

// HandleRequest 处理 HTTP 回调
func (app *App) HandleRequest(Conn *SunnyNet.HttpConn) {
	// 恐慌恢复
	defer func() {
		if r := recover(); r != nil {
			utils.Error("HandleRequest panic: %v", r)
		}
	}()

	if Conn.Type == public.HttpSendRequest {
		Conn.Request.Header.Del("Accept-Encoding")

		for _, interceptor := range app.requestInterceptors {
			if interceptor != nil && interceptor.Handle(Conn) {
				return
			}
		}
	} else if Conn.Type == public.HttpResponseOK {
		for _, interceptor := range app.responseInterceptors {
			if interceptor != nil && interceptor.Handle(Conn) {
				return
			}
		}
	}
}

func (app *App) printEnvConfig() {
	hasAnyConfig := os.Getenv("WX_CHANNEL_TOKEN") != "" ||
		os.Getenv("WX_CHANNEL_ALLOWED_ORIGINS") != "" ||
		os.Getenv("WX_CHANNEL_LOG_FILE") != "" ||
		os.Getenv("WX_CHANNEL_LOG_MAX_MB") != "" ||
		os.Getenv("WX_CHANNEL_SAVE_PAGE_SNAPSHOT") != "" ||
		os.Getenv("WX_CHANNEL_SAVE_SEARCH_DATA") != "" ||
		os.Getenv("WX_CHANNEL_SAVE_PAGE_JS") != "" ||
		os.Getenv("WX_CHANNEL_SHOW_LOG_BUTTON") != "" ||
		os.Getenv("WX_CHANNEL_UPLOAD_CHUNK_CONCURRENCY") != "" ||
		os.Getenv("WX_CHANNEL_UPLOAD_MERGE_CONCURRENCY") != "" ||
		os.Getenv("WX_CHANNEL_DOWNLOAD_CONCURRENCY") != ""

	if hasAnyConfig {
		utils.PrintSeparator()
		color.Blue("⚙️  环境变量配置信息")
		utils.PrintSeparator()

		if app.Cfg.SecretToken != "" {
			utils.PrintLabelValue("🔐", "安全令牌", "已设置")
		}
		if len(app.Cfg.AllowedOrigins) > 0 {
			utils.PrintLabelValue("🌐", "允许的Origin", strings.Join(app.Cfg.AllowedOrigins, ", "))
		}
		if app.Cfg.LogFile != "" {
			utils.PrintLabelValue("📝", "日志文件", app.Cfg.LogFile)
		}
		if app.Cfg.MaxLogSizeMB > 0 {
			utils.PrintLabelValue("📊", "日志最大大小", fmt.Sprintf("%d MB", app.Cfg.MaxLogSizeMB))
		}
		utils.PrintLabelValue("💾", "保存页面快照", fmt.Sprintf("%v", app.Cfg.SavePageSnapshot))
		utils.PrintLabelValue("🔍", "保存搜索数据", fmt.Sprintf("%v", app.Cfg.SaveSearchData))
		utils.PrintLabelValue("📄", "保存JS文件", fmt.Sprintf("%v", app.Cfg.SavePageJS))
		utils.PrintLabelValue("🖼️", "显示日志按钮", fmt.Sprintf("%v", app.Cfg.ShowLogButton))
		utils.PrintLabelValue("📤", "分片上传并发", app.Cfg.UploadChunkConcurrency)
		utils.PrintLabelValue("🔀", "分片合并并发", app.Cfg.UploadMergeConcurrency)
		utils.PrintLabelValue("📥", "批量下载并发", app.Cfg.DownloadConcurrency)
		utils.PrintSeparator()
	}
}

func (app *App) printTitle() {
	color.Set(color.FgCyan)
	fmt.Println("")
	fmt.Println(" ██╗    ██╗██╗  ██╗     ██████╗██╗  ██╗ █████╗ ███╗   ██╗███╗   ██╗███████╗██╗     ")
	fmt.Println(" ██║    ██║╚██╗██╔╝    ██╔════╝██║  ██║██╔══██╗████╗  ██║████╗  ██║██╔════╝██║     ")
	fmt.Println(" ██║ █╗ ██║ ╚███╔╝     ██║     ███████║███████║██╔██╗ ██║██╔██╗ ██║█████╗  ██║     ")
	fmt.Println(" ██║███╗██║ ██╔██╗     ██║     ██╔══██║██╔══██║██║╚██╗██║██║╚██╗██║██╔══╝  ██║     ")
	fmt.Println(" ╚███╔███╔╝██╔╝ ██╗    ╚██████╗██║  ██║██║  ██║██║ ╚████║██║ ╚████║███████╗███████╗")
	fmt.Println("  ╚══╝╚══╝ ╚═╝  ╚═╝     ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝  ╚═══╝╚══════╝╚══════╝")
	color.Unset()

	color.Yellow("    微信视频号下载助手 v%s", app.Cfg.Version)
	color.Yellow("    项目地址：https://github.com/nobiyou/wx_channel")
	color.Green("    v%s 更新要点：", app.Cfg.Version)
	color.Green("    • 核心升级 - 集成 Gopeed 引擎，支持断点续传/高并发")
	color.Green("    • 限制移除 - 批量下载解除数量限制 (支持10万+)")
	color.Green("    • 体验增强 - 实时进度显示 (45.2%%)，支持立即取消")
	color.Green("    • 问题修复 - 取消直接保存csv，优化数据库写入")
	fmt.Println()
}

// 隐式需要的辅助函数

func (app *App) startWebSocketServer(wsPort int) {
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		handlers.ServeWs(w, r)
	})

	// 挂载主 API Router，允许通过 WS 端口 (2026) 直接访问管理 API
	if app.APIRouter != nil {
		mux.Handle("/api/", app.APIRouter)
	}

	wsHandler := websocket.NewHandler(app.WSHub)
	mux.HandleFunc("/ws/api", wsHandler.ServeHTTP)

	mux.HandleFunc("/ws/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		hub := handlers.GetWebSocketHub()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"clients": hub.ClientCount(),
		})
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", wsPort),
		Handler: mux,
	}

	utils.Info("🔌 WebSocket服务已启动，端口: %d", wsPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		utils.Warn("WebSocket服务启动失败: %v", err)
	}
}
