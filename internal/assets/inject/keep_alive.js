/**
 * @file 保持页面活跃 - 防止页面休眠导致API调用超时
 * @version 3.1 - 禁用自动刷新，依赖 WebSocket 自动重连机制
 */
console.log('[keep_alive.js] 加载页面保活模块 v3.1 (自动刷新已禁用)');

window.__wx_keep_alive = {
    wakeLock: null,
    activityTimer: null,
    domActivityTimer: null,
    heartbeatTimer: null,
    refreshTimer: null,
    isActive: false,
    lastRefreshTime: Date.now(),
    stats: {
        startTime: Date.now(),
        heartbeats: 0,
        wakeLockRenewals: 0,
        visibilityChanges: 0,
        refreshCount: 0
    },

    // 初始化
    init: function () {
        if (this.isActive) {
            console.log('[页面保活] 已经在运行中');
            return;
        }

        console.log('[页面保活] 🚀 启动保活机制...');
        this.isActive = true;

        // 方法1: 使用 Wake Lock API 防止屏幕休眠
        this.requestWakeLock();

        // 方法2: 定期模拟用户活动（降低频率到60秒）
        this.startActivitySimulation();

        // 方法3: 监听页面可见性变化
        this.setupVisibilityMonitor();

        // 方法4: 定期执行轻量级DOM操作（降低频率到30秒）
        this.startDOMActivity();

        // 方法5: 定期发送心跳到后端（可选，用于监控）
        this.startHeartbeat();

        // 方法6: 定期刷新页面（已禁用 - 依赖 WebSocket 自动重连机制）
        // this.startAutoRefresh();

        // 添加全局访问方法
        window.getKeepAliveStats = () => this.getStats();
    },

    // 请求 Wake Lock（防止屏幕休眠）
    requestWakeLock: async function () {
        if (!('wakeLock' in navigator)) {
            console.log('[页面保活] ⚠️ 浏览器不支持 Wake Lock API');
            return;
        }

        try {
            this.wakeLock = await navigator.wakeLock.request('screen');
            console.log('[页面保活] ✅ Wake Lock 已激活');

            // 监听释放事件
            this.wakeLock.addEventListener('release', () => {
                console.log('[页面保活] ⚠️ Wake Lock 已释放');
                this.wakeLock = null;

                // 如果保活机制仍在运行，尝试重新获取
                if (this.isActive) {
                    setTimeout(() => {
                        this.stats.wakeLockRenewals++;
                        this.requestWakeLock();
                    }, 2000);
                }
            });
        } catch (err) {
            console.error('[页面保活] ❌ Wake Lock 请求失败:', err.message);
        }
    },

    // 模拟用户活动（持续运行，不受页面可见性影响）
    startActivitySimulation: function () {
        // 每45秒触发一次活动（确保在 WebSocket 90秒超时之前有足够的活动）
        this.activityTimer = setInterval(() => {
            // 移除页面隐藏检测，始终保持活动

            // 触发自定义事件
            const event = new CustomEvent('wx_keep_alive_ping', {
                detail: {
                    timestamp: Date.now(),
                    heartbeats: this.stats.heartbeats,
                    isVisible: !document.hidden
                }
            });
            document.dispatchEvent(event);

            // 读取DOM属性触发渲染引擎（最轻量的操作）
            void document.body.offsetHeight;

            // 触发一个微小的鼠标移动事件（模拟用户活动）
            try {
                const moveEvent = new MouseEvent('mousemove', {
                    bubbles: true,
                    cancelable: true,
                    view: window,
                    clientX: 1,
                    clientY: 1
                });
                document.dispatchEvent(moveEvent);
            } catch (e) {
                // 忽略错误
            }

            this.stats.heartbeats++;

            // 记录日志（降低频率，避免刷屏）
            if (this.stats.heartbeats % 10 === 0) {
                console.log(`[页面保活] 💓 活动 #${this.stats.heartbeats} (页面${document.hidden ? '隐藏' : '可见'})`);
            }
        }, 45000); // 45秒（确保在 WebSocket 超时前有活动）

        console.log('[页面保活] ✅ 活动模拟已启动 (45秒间隔，无视页面可见性)');
    },

    // 监听页面可见性（仅用于日志记录，不影响保活）
    setupVisibilityMonitor: function () {
        document.addEventListener('visibilitychange', () => {
            this.stats.visibilityChanges++;

            if (document.hidden) {
                console.warn('[页面保活] ⚠️ 页面已隐藏（保活机制继续运行）');
            } else {
                console.log('[页面保活] ✅ 页面已重新激活');

                // 页面重新可见时，尝试重新请求 Wake Lock
                if (this.isActive && !this.wakeLock) {
                    this.requestWakeLock();
                }
            }
        });

        console.log('[页面保活] ✅ 可见性监控已启动（仅记录，不影响保活）');
    },

    // 定期执行轻量级DOM操作（持续运行）
    startDOMActivity: function () {
        // 创建隐藏标记
        const keepAliveDiv = document.createElement('div');
        keepAliveDiv.id = '__wx_keep_alive_marker';
        keepAliveDiv.style.cssText = 'display:none;position:absolute;width:1px;height:1px;';
        keepAliveDiv.setAttribute('data-timestamp', Date.now());
        document.body.appendChild(keepAliveDiv);

        // 每30秒更新一次（无视页面可见性）
        this.domActivityTimer = setInterval(() => {
            // 移除页面隐藏检测，始终执行

            const marker = document.getElementById('__wx_keep_alive_marker');
            if (marker) {
                marker.setAttribute('data-timestamp', Date.now());
                marker.setAttribute('data-visible', !document.hidden);
                // 触发重绘（最轻量的方式）
                void marker.offsetHeight;
            }
        }, 30000); // 30秒

        console.log('[页面保活] ✅ DOM活动已启动 (30秒间隔，无视页面可见性)');
    },

    // 定期发送心跳到后端（用于监控页面状态）
    startHeartbeat: function () {
        // 每2分钟发送一次心跳
        this.heartbeatTimer = setInterval(() => {
            // 移除页面隐藏检测，始终发送心跳

            // 触发自定义事件，可以被其他模块监听
            const event = new CustomEvent('wx_keep_alive_heartbeat', {
                detail: this.getStats()
            });
            document.dispatchEvent(event);

            // 主动触发 WebSocket ping（如果存在）
            this.triggerWebSocketPing();

            // 可选：发送到后端（如果需要）
            // this.sendHeartbeatToBackend();

            console.log('[页面保活] 💗 心跳发送 (页面' + (document.hidden ? '隐藏' : '可见') + ')');
        }, 120000); // 2分钟

        console.log('[页面保活] ✅ 心跳监控已启动 (2分钟间隔，无视页面可见性)');
    },

    // 触发 WebSocket ping（保持 WebSocket 连接活跃）
    triggerWebSocketPing: function () {
        try {
            // 查找页面中的 WebSocket 连接并发送 ping
            if (window.__wsConnection) {
                const pingMsg = JSON.stringify({ type: 'ping', timestamp: Date.now() });
                window.__wsConnection.send(pingMsg);
                console.log('[页面保活] 📡 WebSocket ping 已发送');
            }
        } catch (e) {
            // 忽略错误（WebSocket 可能不存在或已关闭）
        }
    },

    // 发送心跳到后端（可选）
    sendHeartbeatToBackend: function () {
        // 使用 sendBeacon 发送，即使页面关闭也能发送
        if (navigator.sendBeacon) {
            const data = JSON.stringify({
                type: 'keep_alive_heartbeat',
                stats: this.getStats(),
                userAgent: navigator.userAgent,
                url: window.location.href
            });

            // 替换为实际的心跳接口
            // navigator.sendBeacon('/api/heartbeat', data);
        }
    },

    // 定期刷新页面（最后的保活手段）
    startAutoRefresh: function () {
        // 每 10 分钟刷新一次页面，确保连接不会超时
        const REFRESH_INTERVAL = 10 * 60 * 1000; // 10 分钟

        this.refreshTimer = setInterval(() => {
            const now = Date.now();
            const timeSinceLastRefresh = now - this.lastRefreshTime;

            // 只有在页面运行超过 5 分钟时才刷新
            if (timeSinceLastRefresh >= REFRESH_INTERVAL) {
                this.performRefresh('定期刷新');
            }
        }, REFRESH_INTERVAL);

        // 尝试恢复之前的统计信息
        try {
            const savedStats = sessionStorage.getItem('__wx_keep_alive_stats');
            if (savedStats) {
                const parsed = JSON.parse(savedStats);
                this.stats.refreshCount = (parsed.refreshCount || 0);
                this.lastRefreshTime = parsed.lastRefreshTime || Date.now();
                console.log(`[页面保活] ✅ 恢复统计信息: 已刷新 ${this.stats.refreshCount} 次`);
            }
        } catch (e) {
            console.error('[页面保活] 恢复状态失败:', e);
        }

        console.log('[页面保活] ✅ 自动刷新已启动 (10分钟间隔)');
    },

    // 执行页面刷新（可被外部调用）
    performRefresh: function (reason) {
        reason = reason || '手动刷新';
        const now = Date.now();

        this.stats.refreshCount++;
        console.warn(`[页面保活] 🔄 执行刷新: ${reason} (第 ${this.stats.refreshCount} 次)`);
        console.log('[页面保活] 刷新前统计:', this.getStats());

        // 保存当前状态到 sessionStorage
        try {
            sessionStorage.setItem('__wx_keep_alive_stats', JSON.stringify({
                ...this.stats,
                lastRefreshTime: now,
                lastRefreshReason: reason
            }));
        } catch (e) {
            console.error('[页面保活] 保存状态失败:', e);
        }

        // 刷新页面
        window.location.reload();
    },

    // 获取统计信息
    getStats: function () {
        return {
            ...this.stats,
            uptime: Date.now() - this.stats.startTime,
            uptimeMinutes: Math.floor((Date.now() - this.stats.startTime) / 60000),
            timeSinceLastRefresh: Date.now() - this.lastRefreshTime,
            isActive: this.isActive,
            isVisible: !document.hidden,
            hasWakeLock: !!this.wakeLock
        };
    },

    // 停止保活
    stop: function () {
        if (!this.isActive) {
            console.log('[页面保活] 未在运行');
            return;
        }

        console.log('[页面保活] 🛑 停止保活机制');
        this.isActive = false;

        // 释放 Wake Lock
        if (this.wakeLock) {
            this.wakeLock.release();
            this.wakeLock = null;
        }

        // 清除定时器
        if (this.activityTimer) {
            clearInterval(this.activityTimer);
            this.activityTimer = null;
        }

        if (this.domActivityTimer) {
            clearInterval(this.domActivityTimer);
            this.domActivityTimer = null;
        }

        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }

        if (this.refreshTimer) {
            clearInterval(this.refreshTimer);
            this.refreshTimer = null;
        }

        // 移除DOM标记
        const marker = document.getElementById('__wx_keep_alive_marker');
        if (marker) {
            marker.remove();
        }

        console.log('[页面保活] 最终统计:', this.getStats());
    }
};

// 自动启动
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        window.__wx_keep_alive.init();
    });
} else {
    window.__wx_keep_alive.init();
}

// 页面卸载时清理
window.addEventListener('beforeunload', () => {
    window.__wx_keep_alive.stop();
});

console.log('[keep_alive.js] 页面保活模块加载完成 v3.0 (自动刷新已禁用)');
console.log('[keep_alive.js] 使用 window.getKeepAliveStats() 查看统计信息');
console.log('[keep_alive.js] 依赖 WebSocket 自动重连机制保持连接稳定');
