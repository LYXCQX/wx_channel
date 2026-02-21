/**
 * @file API 客户端 - 通过 WebSocket 与后端通信
 */
console.log('[api_client.js] 加载 API 客户端模块');

window.__wx_api_client = {
  ws: null,
  connected: false,
  reconnectTimer: null,
  reconnectDelay: 3000,
  requests: {},
  heartbeatTimer: null,
  lastHeartbeatTime: 0,
  missedHeartbeats: 0,

  // 初始化
  init: function () {
    this.connect();
    this.setupVisibilityHandler();
    this.setupBeforeUnloadHandler();
  },

  // 设置页面可见性监听
  setupVisibilityHandler: function () {
    var self = this;

    document.addEventListener('visibilitychange', function () {
      if (!document.hidden) {
        // 页面变为可见
        console.log('[API客户端] 📱 页面激活，检查连接状态...');

        if (!self.connected) {
          console.log('[API客户端] 连接已断开，立即重连...');
          // 清除现有的重连定时器
          if (self.reconnectTimer) {
            clearTimeout(self.reconnectTimer);
            self.reconnectTimer = null;
          }
          // 立即重连
          self.connect();
        } else {
          // 连接还在，发送一个心跳测试
          self.sendHeartbeat();
        }
      } else {
        // 页面变为隐藏
        console.log('[API客户端] 📴 页面进入后台');
      }
    });

    console.log('[API客户端] ✅ 页面可见性监听已启动');
  },

  // 设置页面关闭前的处理
  setupBeforeUnloadHandler: function () {
    var self = this;

    window.addEventListener('beforeunload', function () {
      // 页面即将关闭，清理资源
      if (self.ws && self.connected) {
        self.ws.close(1000, 'Page unloading');
      }

      if (self.heartbeatTimer) {
        clearInterval(self.heartbeatTimer);
      }

      if (self.reconnectTimer) {
        clearTimeout(self.reconnectTimer);
      }
    });
  },

  // 连接 WebSocket
  connect: function () {
    var self = this;

    // 检测代理端口
    // 方法1: 尝试从 /__wx_channels_api 端点获取端口信息
    // 方法2: 使用默认端口 2026
    var wsPort = 2026; // 默认端口

    // 尝试多个可能的端口
    var possiblePorts = [2026, 9527, 8081, 3001];

    // 从 localStorage 获取上次成功的端口
    try {
      var lastPort = localStorage.getItem('__wx_api_ws_port');
      if (lastPort) {
        possiblePorts.unshift(parseInt(lastPort));
      }
    } catch (e) {
      // ignore
    }

    // 尝试连接
    this.tryConnect(possiblePorts, 0);
  },

  // 尝试连接到指定端口
  tryConnect: function (ports, index) {
    var self = this;

    if (index >= ports.length) {
      console.error('[API客户端] 所有端口都连接失败，3秒后重试...');
      this.reconnectTimer = setTimeout(function () {
        self.connect();
      }, this.reconnectDelay);
      return;
    }

    var wsPort = ports[index];
    var wsUrl = 'ws://127.0.0.1:' + wsPort + '/ws/api';
    if (window.__WX_LOCAL_TOKEN__) {
      wsUrl += '?token=' + encodeURIComponent(window.__WX_LOCAL_TOKEN__);
    }

    console.log('[API客户端] 尝试连接:', wsUrl);

    // 标记当前尝试的端口索引
    this.currentPortIndex = index;
    this.currentPorts = ports;

    try {
      this.ws = new WebSocket(wsUrl);

      // 设置连接超时（5秒）
      var connectTimeout = setTimeout(function () {
        if (!self.connected && self.ws && self.ws.readyState !== WebSocket.OPEN) {
          console.log('[API客户端] 连接超时，尝试下一个端口...');
          self.ws.close();
          self.tryConnect(ports, index + 1);
        }
      }, 5000);

      this.ws.onopen = function () {
        clearTimeout(connectTimeout);
        self.connected = true;
        console.log('[API客户端] ✅ 已连接到后端: ws://127.0.0.1:' + wsPort + '/ws/api');

        // 保存成功的端口
        try {
          localStorage.setItem('__wx_api_ws_port', wsPort);
        } catch (e) {
          // ignore
        }

        // 清除重连定时器
        if (self.reconnectTimer) {
          clearTimeout(self.reconnectTimer);
          self.reconnectTimer = null;
        }

        // 启动心跳
        self.startHeartbeat();
      };

      this.ws.onmessage = function (event) {
        try {
          var msg = JSON.parse(event.data);
          self.handleMessage(msg);
        } catch (err) {
          console.error('[API客户端] 解析消息失败:', err);
        }
      };

      this.ws.onerror = function (error) {
        clearTimeout(connectTimeout);
        console.error('[API客户端] ❌ WebSocket 错误:', error);
        // 如果还没有连接成功，尝试下一个端口
        if (!self.connected) {
          self.tryConnect(ports, index + 1);
        }
      };

      this.ws.onclose = function (event) {
        clearTimeout(connectTimeout);
        console.log('[API客户端] 🔌 连接关闭:', event.code, event.reason);

        // 停止心跳
        self.stopHeartbeat();

        if (self.connected) {
          // 之前连接成功过，现在断开了，需要重连
          self.connected = false;
          console.log('[API客户端] 连接已关闭，3秒后重连...');

          // 自动重连（使用之前成功的端口）
          self.reconnectTimer = setTimeout(function () {
            self.connect();
          }, self.reconnectDelay);
        } else {
          // 连接从未成功，尝试下一个端口
          self.tryConnect(ports, index + 1);
        }
      };
    } catch (err) {
      console.error('[API客户端] ❌ 连接失败:', err);
      // 尝试下一个端口
      this.tryConnect(ports, index + 1);
    }
  },

  // 处理消息
  handleMessage: function (msg) {
    if (msg.type === 'api_call') {
      this.handleAPICall(msg.data);
    } else if (msg.type === 'cmd') {
      this.handleCommand(msg.data);
    }
  },

  // 处理指令
  handleCommand: function (data) {
    console.log('[API客户端] 收到指令:', data);

    if (data.action === 'start_comment_collection') {
      if (typeof window.__wx_channels_start_comment_collection === 'function') {
        console.log('[API客户端] 执行评论采集指令...');
        window.__wx_channels_start_comment_collection();
      } else {
        console.warn('[API客户端] 评论采集函数未就绪');
      }
    }

    if (data.action === 'download_progress') {
      // 派发自定义事件，供 UI 组件消费
      var event = new CustomEvent('wx_download_progress', { detail: data.payload });
      document.dispatchEvent(event);
    }
  },

  // 处理 API 调用请求
  handleAPICall: async function (data) {
    var id = data.id;
    var key = data.key;
    var body = data.body;

    // 响应函数
    var self = this;
    function resp(responseData) {
      self.sendResponse(id, responseData);
    }

    try {
      // 等待 WXU.API 和 WXU.API2 初始化
      var maxWait = 10000; // 最多等待10秒
      var startTime = Date.now();

      while ((!window.WXU || !window.WXU.API || !window.WXU.API2) && (Date.now() - startTime < maxWait)) {
        console.log('[API客户端] 等待 WXU.API 初始化...');
        await new Promise(function (resolve) { setTimeout(resolve, 500); });
      }

      if (!window.WXU || !window.WXU.API || !window.WXU.API2) {
        resp({
          errCode: 1,
          errMsg: 'WXU.API 未初始化，请刷新页面重试'
        });
        return;
      }

      if (key === 'key:channels:contact_list') {
        // Correct Scene Mapping:
        // Type 1 (User): Scene 13 → infoList (supports pagination)
        // Type 2 (Live): Scene 13 → objectList (NO pagination support)
        // Type 3 (Video): Scene 19 → objectList (supports pagination)
        var scene = 13; // Default to Scene 13 for Type 1 and Type 2
        if (body.type == 3) {
          scene = 19; // Only Type 3 (Video) uses Scene 19
        }

        var payload = {
          query: body.keyword,
          scene: scene,
          requestId: String(new Date().valueOf()), // Unique request ID for every page
          lastBuffer: body.next_marker ? decodeURIComponent(body.next_marker) : '',
          lastBuff: body.next_marker ? decodeURIComponent(body.next_marker) : '', // Try alias
        };
        var r = await window.WXU.API2.finderSearch(payload);
        console.log('[API客户端] finderSearch 结果:', r);
        resp({
          ...r,
          payload: payload
        });
        return;
      }

      // 获取账号视频列表
      if (key === 'key:channels:feed_list') {
        var payload = {
          username: body.username,
          finderUsername: window.__wx_username || '',
          lastBuffer: body.next_marker ? decodeURIComponent(body.next_marker) : '',
          needFansCount: 0,
          objectId: '0'
        };
        var r = await window.WXU.API.finderUserPage(payload);
        console.log('[API客户端] finderUserPage 结果:', r);
        resp({
          ...r,
          payload: payload
        });
        return;
      }

      // 获取视频详情
      if (key === 'key:channels:feed_profile') {
        console.log('[API客户端] 获取视频详情:', body);

        try {
          var oid = body.objectId || body.object_id || body.oid || '';
          var nid = body.nonceId || body.nonce_id || body.nid || '';

          // 如果提供了 URL，从 URL 中解析 oid 和 nid
          if (body.url) {
            var u = new URL(decodeURIComponent(body.url));
            oid = window.WXU.API.decodeBase64ToUint64String(u.searchParams.get('oid'));
            nid = window.WXU.API.decodeBase64ToUint64String(u.searchParams.get('nid'));
          }

          if (!oid || !nid) {
            throw new Error('缺失 object_id 或 nonce_id');
          }

          var payload = {
            needObject: 1,
            lastBuffer: '',
            scene: 146,
            direction: 2,
            identityScene: 2,
            pullScene: 6,
            objectid: String(oid).includes('_') ? String(oid).split('_')[0] : String(oid),
            objectNonceId: nid,
            encrypted_objectid: ''
          };

          var r = await window.WXU.API.finderGetCommentDetail(payload);
          console.log('[API客户端] finderGetCommentDetail 结果:', r);
          resp({
            ...r,
            payload: payload
          });
          return;
        } catch (err) {
          console.error('[API客户端] 获取视频详情失败:', err);
          resp({
            errCode: 1011,
            errMsg: err.message,
            payload: body
          });
          return;
        }
      }

      // 未匹配的 key
      resp({
        errCode: 1000,
        errMsg: '未匹配的key: ' + key,
        payload: data
      });

    } catch (err) {
      console.error('[API客户端] API 调用失败:', err);
      resp({
        errCode: 1,
        errMsg: err.message || 'API 调用失败',
        payload: data
      });
    }
  },

  // 发送响应
  sendResponse: function (id, responseData) {
    if (!this.connected || !this.ws) {
      console.error('[API客户端] WebSocket 未连接');
      return;
    }

    // 构建响应消息
    // 后端期望的格式: {type: "api_response", data: {id: "xxx", data: {...}, errCode: 0, errMsg: "ok"}}
    var msg = {
      type: 'api_response',
      data: {
        id: id,
        data: responseData,  // 整个 responseData 作为 data 字段
        errCode: responseData.errCode || 0,
        errMsg: responseData.errMsg || 'ok'
      }
    };

    try {
      var msgStr = JSON.stringify(msg);
      this.ws.send(msgStr);
    } catch (err) {
      console.error('[API客户端] 发送响应失败:', err);
    }
  },

  // 启动心跳
  startHeartbeat: function () {
    var self = this;

    // 清除旧的心跳定时器
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
    }

    // 重置心跳计数
    this.missedHeartbeats = 0;
    this.lastHeartbeatTime = Date.now();

    // 每 30 秒发送一次心跳
    this.heartbeatTimer = setInterval(function () {
      self.sendHeartbeat();
    }, 30000);

    console.log('[API客户端] ✅ 心跳已启动 (30秒间隔)');
  },

  // 停止心跳
  stopHeartbeat: function () {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
      console.log('[API客户端] ⏹️ 心跳已停止');
    }
  },

  // 发送心跳
  sendHeartbeat: function () {
    if (!this.connected || !this.ws) {
      console.warn('[API客户端] 无法发送心跳：未连接');
      this.missedHeartbeats++;

      // 连续 3 次心跳失败，触发重连
      if (this.missedHeartbeats >= 3) {
        console.error('[API客户端] 心跳连续失败，触发重连...');
        this.stopHeartbeat();

        // 关闭当前连接
        if (this.ws) {
          try {
            this.ws.close();
          } catch (e) {
            // ignore
          }
        }

        // 立即重连
        this.connected = false;
        this.connect();
      }
      return;
    }

    try {
      var heartbeat = {
        type: 'ping',
        timestamp: Date.now()
      };

      this.ws.send(JSON.stringify(heartbeat));
      this.lastHeartbeatTime = Date.now();
      this.missedHeartbeats = 0;

      console.log('[API客户端] 💓 心跳已发送');
    } catch (err) {
      console.error('[API客户端] 发送心跳失败:', err);
      this.missedHeartbeats++;
    }
  }
};

// 自动初始化
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', function () {
    window.__wx_api_client.init();
  });
} else {
  window.__wx_api_client.init();
}

// 监听初始化事件，获取用户名
if (window.WXE && window.WXE.onInit) {
  window.WXE.onInit(function (data) {
    if (data && data.mainFinderUsername) {
      window.__wx_username = data.mainFinderUsername;
      console.log('[API客户端] 已获取用户名:', window.__wx_username);
    }
  });
}

console.log('[api_client.js] API 客户端模块加载完成');
