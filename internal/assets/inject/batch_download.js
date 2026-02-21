/**
 * @file 通用批量下载组件
 * 提供统一的视频列表弹窗和批量下载功能
 */
console.log('[batch_download.js] 加载通用批量下载模块');

// ==================== 通用批量下载管理器 ====================
window.__wx_batch_download_manager__ = {
  videos: [], // 当前视频列表
  selectedItems: {}, // 选中的项目 {id: true}
  currentPage: 1,
  pageSize: 50,
  maxItems: 100000, // Gopeed接管后取消限制 (原300)
  isVisible: false,
  title: '视频列表',
  isDownloading: false, // 是否正在下载
  isDownloading: false, // 是否正在下载
  stopSignal: false, // 取消下载信号
  forceRedownload: false, // 强制重新下载
  abortController: null, // 当前请求的 AbortController

  // 设置视频数据
  setVideos: function (videos, title) {
    this.videos = videos.slice(0, this.maxItems); // 限制最多300个
    this.selectedItems = {};
    this.currentPage = 1;
    if (title) this.title = title;
    console.log('[批量下载] 设置视频数据，共', this.videos.length, '个');
  },

  // 追加视频数据（去重）
  appendVideos: function (videos) {
    var existingIds = {};
    this.videos.forEach(function (v) {
      existingIds[v.id] = true;
    });

    var newCount = 0;
    for (var i = 0; i < videos.length && this.videos.length < this.maxItems; i++) {
      var video = videos[i];
      if (video.id && !existingIds[video.id]) {
        this.videos.push(video);
        existingIds[video.id] = true;
        newCount++;
      }
    }

    console.log('[批量下载] 追加', newCount, '个视频，总计:', this.videos.length);
    return newCount;
  },

  // 获取当前页的视频
  getCurrentPageVideos: function () {
    var start = (this.currentPage - 1) * this.pageSize;
    var end = start + this.pageSize;
    return this.videos.slice(start, end);
  },

  // 获取总页数
  getTotalPages: function () {
    return Math.ceil(this.videos.length / this.pageSize);
  },

  // 获取选中的视频
  getSelectedVideos: function () {
    var self = this;
    return this.videos.filter(function (video) {
      return self.selectedItems[video.id];
    });
  },

  // 切换选中状态
  toggleSelect: function (videoId, selected) {
    if (selected) {
      this.selectedItems[videoId] = true;
    } else {
      delete this.selectedItems[videoId];
    }
  },

  // 全选当前页
  selectAllCurrentPage: function (selected) {
    var pageVideos = this.getCurrentPageVideos();
    for (var i = 0; i < pageVideos.length; i++) {
      this.toggleSelect(pageVideos[i].id, selected);
    }
  }
};

// ==================== 显示批量下载弹窗 ====================
function __show_batch_download_ui__(videos, title) {
  if (!videos || videos.length === 0) {
    __wx_log({ msg: '❌ 暂无视频数据' });
    return;
  }

  // 设置数据
  __wx_batch_download_manager__.setVideos(videos, title || '视频列表');

  // 移除已存在的弹窗
  var existingUI = document.getElementById('wx-batch-download-ui');
  if (existingUI) existingUI.remove();

  // 创建弹窗
  var ui = document.createElement('div');
  ui.id = 'wx-batch-download-ui';
  ui.style.cssText = 'position:fixed;top:60px;right:20px;background:#2b2b2b;color:#e5e5e5;padding:0;border-radius:8px;z-index:99999;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif;font-size:14px;width:450px;max-height:80vh;box-shadow:0 8px 24px rgba(0,0,0,0.5);overflow:hidden;';

  // 统计视频和直播数量
  var videoCount = 0;
  var liveCount = 0;
  videos.forEach(function (v) {
    if (v.type === 'live' || v.type === 'live_replay') {
      liveCount++;
    } else if (v.type === 'media' || !v.type) {
      videoCount++;
    }
  });

  // 根据页面类型构建统计文本
  var statsText = '';
  var currentPath = window.location.pathname;

  if (currentPath.includes('/pages/home')) {
    // Home页：显示"X 个视频"
    statsText = videoCount + ' 个视频';
    if (liveCount > 0) {
      statsText += ', ' + liveCount + ' 个直播';
    }
  } else if (currentPath.includes('/pages/s')) {
    // 搜索页：显示"X 个动态, Y 个直播"
    if (liveCount > 0) {
      statsText = videoCount + ' 个动态, ' + liveCount + ' 个直播';
    } else {
      statsText = videoCount + ' 个动态';
    }
  } else if (currentPath.includes('/pages/profile')) {
    // Profile页：显示"X 个视频, Y 个直播回放"
    if (liveCount > 0) {
      statsText = videoCount + ' 个视频, ' + liveCount + ' 个直播回放';
    } else {
      statsText = videoCount + ' 个视频';
    }
  } else {
    // 其他页面：默认显示
    if (liveCount > 0) {
      statsText = videoCount + ' 个视频, ' + liveCount + ' 个直播';
    } else {
      statsText = videos.length + ' 个';
    }
  }

  ui.innerHTML =
    // 标题栏
    '<div style="padding:16px 20px;border-bottom:1px solid rgba(255,255,255,0.08);display:flex;justify-content:space-between;align-items:center;">' +
    '<div style="font-size:15px;font-weight:500;color:#fff;">' + __wx_batch_download_manager__.title + '</div>' +
    '<div style="display:flex;align-items:center;gap:12px;">' +
    '<div id="batch-total-count" style="font-size:13px;color:#999;">' + statsText + '</div>' +
    '<div id="batch-close-icon" style="cursor:pointer;color:#999;font-size:20px;line-height:1;padding:4px;" title="关闭">×</div>' +
    '</div>' +
    '</div>' +

    // 列表区域
    '<div id="batch-list-container" style="overflow-y:auto;padding:12px 20px;max-height:200px;">' +
    '<div id="batch-list" style="display:flex;flex-direction:column;gap:8px;"></div>' +
    '</div>' +

    // 分页
    '<div id="batch-pagination" style="padding:12px 20px;border-top:1px solid rgba(255,255,255,0.08);border-bottom:1px solid rgba(255,255,255,0.08);display:flex;justify-content:space-between;align-items:center;">' +
    '<div style="font-size:13px;color:#999;">第 <span id="batch-current-page">1</span> / <span id="batch-total-pages">1</span> 页</div>' +
    '<div style="display:flex;gap:8px;">' +
    '<button id="batch-prev-page" style="background:rgba(255,255,255,0.08);color:#999;border:none;padding:4px 12px;border-radius:4px;cursor:pointer;font-size:13px;">上一页</button>' +
    '<button id="batch-next-page" style="background:rgba(255,255,255,0.08);color:#999;border:none;padding:4px 12px;border-radius:4px;cursor:pointer;font-size:13px;">下一页</button>' +
    '</div>' +
    '</div>' +

    // 操作区
    '<div style="padding:16px 20px;">' +
    '<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:12px;">' +
    '<label style="display:flex;align-items:center;cursor:pointer;font-size:13px;color:#999;user-select:none;">' +
    '<input type="checkbox" id="batch-select-all" style="margin-right:8px;cursor:pointer;" />' +
    '<span>全选当前页</span>' +
    '</label>' +
    '<span id="batch-selected-count" style="font-size:13px;color:#07c160;">已选 0 个</span>' +
    '</div>' +

    // 下载和取消按钮容器
    '<div style="display:flex;gap:8px;margin-bottom:12px;">' +
    '<button id="batch-download-btn" style="flex:1;background:#07c160;color:#fff;border:none;padding:8px 12px;border-radius:6px;cursor:pointer;font-size:14px;font-weight:500;transition:background 0.2s;">开始下载</button>' +
    '<button id="batch-cancel-btn" style="flex:0 0 25%;background:#fa5151;color:#fff;border:none;padding:8px 12px;border-radius:6px;cursor:pointer;font-size:14px;font-weight:500;display:none;">取消</button>' +
    '</div>' +

    // 下载进度
    '<div id="batch-download-progress" style="display:none;margin-bottom:12px;">' +
    '<div style="display:flex;justify-content:space-between;margin-bottom:8px;font-size:13px;color:#999;">' +
    '<span>下载进度</span>' +
    '<span id="batch-progress-text">0/0</span>' +
    '</div>' +
    '<div style="background:rgba(255,255,255,0.08);height:6px;border-radius:3px;overflow:hidden;">' +
    '<div id="batch-progress-bar" style="background:#07c160;height:100%;width:0%;border-radius:3px;transition:width 0.3s;"></div>' +
    '</div>' +
    '</div>' +

    // 强制重新下载选项
    '<label style="display:flex;align-items:center;cursor:pointer;font-size:13px;color:#999;user-select:none;">' +
    '<input type="checkbox" id="batch-force-redownload" style="margin-right:8px;cursor:pointer;" />' +
    '<span>强制重新下载</span>' +
    '</label>' +
    '</div>' +

    // 次要操作区
    '<div style="padding:12px 20px;border-top:1px solid rgba(255,255,255,0.08);display:flex;gap:8px;">' +
    '<button id="batch-export-btn" style="flex:1;background:transparent;color:#999;border:1px solid rgba(255,255,255,0.12);padding:8px 12px;border-radius:6px;cursor:pointer;font-size:13px;transition:all 0.2s;">导出列表</button>' +
    '<button id="batch-clear-btn" style="flex:1;background:transparent;color:#999;border:1px solid rgba(255,255,255,0.12);padding:8px 12px;border-radius:6px;cursor:pointer;font-size:13px;transition:all 0.2s;">清空列表</button>' +
    '</div>';

  document.body.appendChild(ui);

  __wx_batch_download_manager__.isVisible = true;

  // 渲染列表
  __render_batch_video_list__();

  // 绑定事件
  setTimeout(function () {
    // 分页
    document.getElementById('batch-prev-page').onclick = function () {
      if (__wx_batch_download_manager__.currentPage > 1) {
        __wx_batch_download_manager__.currentPage--;
        __render_batch_video_list__();
      }
    };

    document.getElementById('batch-next-page').onclick = function () {
      if (__wx_batch_download_manager__.currentPage < __wx_batch_download_manager__.getTotalPages()) {
        __wx_batch_download_manager__.currentPage++;
        __render_batch_video_list__();
      }
    };

    // 全选
    document.getElementById('batch-select-all').onchange = function () {
      __wx_batch_download_manager__.selectAllCurrentPage(this.checked);
      __render_batch_video_list__();
    };

    // 下载
    document.getElementById('batch-download-btn').onclick = function () {
      __batch_download_selected__();
    };

    // 取消下载
    document.getElementById('batch-cancel-btn').onclick = function () {
      __cancel_batch_download__();
    };

    // 强制重新下载
    document.getElementById('batch-force-redownload').onchange = function () {
      __wx_batch_download_manager__.forceRedownload = this.checked;
    };

    // 导出列表
    var exportBtn = document.getElementById('batch-export-btn');
    if (exportBtn) {
      exportBtn.addEventListener('mouseenter', function () {
        this.style.background = 'rgba(255,255,255,0.08)';
        this.style.color = '#fff';
      });
      exportBtn.addEventListener('mouseleave', function () {
        this.style.background = 'transparent';
        this.style.color = '#999';
      });
      exportBtn.addEventListener('click', function () {
        __export_batch_video_list__();
      });
    }

    // 清空列表
    var clearBtn = document.getElementById('batch-clear-btn');
    if (clearBtn) {
      clearBtn.addEventListener('mouseenter', function () {
        this.style.background = 'rgba(255,255,255,0.08)';
        this.style.color = '#fff';
      });
      clearBtn.addEventListener('mouseleave', function () {
        this.style.background = 'transparent';
        this.style.color = '#999';
      });
      clearBtn.addEventListener('click', function () {
        __clear_batch_video_list__();
      });
    }

    // 关闭
    document.getElementById('batch-close-icon').onclick = function () {
      __close_batch_download_ui__();
    };

    // 监听实时进度更新
    document.removeEventListener('wx_download_progress', __handle_download_progress__); // 防止重复绑定
    document.addEventListener('wx_download_progress', __handle_download_progress__);
  }, 100);
}

// ==================== 处理进度更新 ====================
function __handle_download_progress__(e) {
  var data = e.detail;
  if (!data) return;

  // 仅在批量下载UI显示时更新
  if (!__wx_batch_download_manager__.isVisible || !__wx_batch_download_manager__.isDownloading) return;

  var progressText = document.getElementById('batch-progress-text');
  var progressBar = document.getElementById('batch-progress-bar');

  if (progressText && progressBar && data.percentage > 0) {
    // 获取当前处理索引（从文本解析或通过其他方式）
    // 这里简单地在当前文本后追加百分比
    // data.total 是单个文件的总大小，不是批量任务的总数
    // 我们可以显示 "1/5 (45%)"

    // 尝试读取当前的进度文本 "1/5"
    var currentText = progressText.textContent.split(' ')[0]; // 取第一部分 n/m
    if (currentText && currentText.includes('/')) {
      var details = data.percentage.toFixed(1) + '%';
      if (data.total > 0) {
        var downMB = (data.downloaded / (1024 * 1024)).toFixed(1);
        var totalMB = (data.total / (1024 * 1024)).toFixed(1);
        details += ' ' + downMB + '/' + totalMB + ' MB';
      }
      progressText.textContent = currentText + ' (' + details + ')';
    }

    // 更新进度条宽度
    progressBar.style.width = data.percentage + '%';
  }
}

// ==================== 关闭弹窗 ====================
function __close_batch_download_ui__() {
  var ui = document.getElementById('wx-batch-download-ui');
  if (ui) ui.remove();
  __wx_batch_download_manager__.isVisible = false;
}

// ==================== 取消下载 ====================
function __cancel_batch_download__() {
  if (__wx_batch_download_manager__.isDownloading) {
    __wx_batch_download_manager__.stopSignal = true;
    __wx_log({ msg: '⏹️ 正在取消下载...' });

    var cancelBtn = document.getElementById('batch-cancel-btn');
    if (cancelBtn) {
      cancelBtn.textContent = '取消中...';
      cancelBtn.disabled = true;
    }

    // 立即终止当前请求
    if (__wx_batch_download_manager__.abortController) {
      try {
        __wx_batch_download_manager__.abortController.abort();
        console.log('[批量下载] 已触发 HTTP 请求中断');
      } catch (e) {
        console.warn('[批量下载] 中断请求失败:', e);
      }
    }
  }
}

// ==================== 导出视频列表 ====================
function __export_batch_video_list__() {
  var videos = __wx_batch_download_manager__.videos;

  if (videos.length === 0) {
    __wx_log({ msg: '⚠️ 没有可导出的视频' });
    return;
  }

  // 格式化导出数据
  var exportData = videos.map(function (v) {
    var media = v.objectDesc && v.objectDesc.media && v.objectDesc.media[0];
    var spec = v.spec || (media && media.spec) || [];

    // 解析 bypass 获取更多信息 (如 cgi_id)
    var cgiId = '';
    var sourceType = '';

    try {
      if (spec && spec.length > 0 && spec[0].bypass) {
        var bypassStr = spec[0].bypass;
        // 简单提取 cgi_id (兼容 "key":val 和 key:val 格式)
        var cgiMatch = bypassStr.match(/"cgi_id":(\d+)/) || bypassStr.match(/cgi_id:(\d+)/);

        if (cgiMatch) {
          cgiId = cgiMatch[1];
          // 6638 = 首页 (FinderGetRecommend)
          if (cgiId === '6638') {
            sourceType = 'Home';
          }
          // 8060 = 其他 (未分类)
          else if (cgiId === '8060') {
            sourceType = 'Other';
          }
          else {
            sourceType = 'Unknown_' + cgiId;
          }
        }
      }
    } catch (e) {
      console.error('[batch_download.js] 解析 bypass 失败', e);
    }

    return {
      id: v.id,
      title: v.title || (v.objectDesc && v.objectDesc.description) || '无标题',
      sourceType: sourceType, // [新增] 数据来源类型
      cgiId: cgiId,           // [新增] 接口ID
      url: v.url || (media && (media.url + (media.urlToken || ''))),
      key: v.key || (media && (media.decodeKey || media.decryptKey)) || '',
      coverUrl: v.coverUrl || v.thumbUrl || (media && media.thumbUrl),
      duration: v.duration || (media && (media.videoPlayLen * 1000 || media.durationMs)),
      size: v.size || (media && media.fileSize),
      nickname: v.nickname || (v.contact && v.contact.nickname) || '',
      createtime: v.createtime,
      // 额外信息
      spec: spec,
      width: (spec[0] && spec[0].width) || (media && media.width) || 0,
      height: (spec[0] && spec[0].height) || (media && media.height) || 0
    };
  });

  var blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
  var url = URL.createObjectURL(blob);
  var a = document.createElement('a');
  a.href = url;
  a.download = 'batch_videos_' + new Date().toISOString().slice(0, 10) + '.json';
  a.click();
  URL.revokeObjectURL(url);

  __wx_log({ msg: '📤 已导出 ' + exportData.length + ' 个视频（含来源标记）' });
}

// ==================== 清空视频列表 ====================
function __clear_batch_video_list__() {
  if (__wx_batch_download_manager__.isDownloading) {
    __wx_log({ msg: '⚠️ 下载中，无法清空' });
    return;
  }

  var count = __wx_batch_download_manager__.videos.length;

  if (count === 0) {
    __wx_log({ msg: '⚠️ 列表已经是空的' });
    return;
  }

  // 确认清空
  if (!confirm('确定要清空 ' + count + ' 个视频吗？')) {
    return;
  }

  __wx_batch_download_manager__.videos = [];
  __wx_batch_download_manager__.selectedItems = {};
  __wx_batch_download_manager__.currentPage = 1;

  // 更新UI
  var countElement = document.getElementById('batch-total-count');
  if (countElement) {
    countElement.textContent = '0 个';
  }

  __render_batch_video_list__();

  __wx_log({ msg: '🗑️ 已清空 ' + count + ' 个视频' });
}

// ==================== 更新弹窗 ====================
function __update_batch_download_ui__(videos, title) {
  if (!__wx_batch_download_manager__.isVisible) return;

  // 追加新视频
  var newCount = __wx_batch_download_manager__.appendVideos(videos);

  if (title) {
    __wx_batch_download_manager__.title = title;
  }

  // 统计视频和直播数量
  var allVideos = __wx_batch_download_manager__.videos;
  var videoCount = 0;
  var liveCount = 0;
  allVideos.forEach(function (v) {
    if (v.type === 'live' || v.type === 'live_replay') {
      liveCount++;
    } else if (v.type === 'media' || !v.type) {
      videoCount++;
    }
  });

  // 更新总数
  var countElement = document.getElementById('batch-total-count');
  if (countElement) {
    var statsText = '';
    var currentPath = window.location.pathname;

    if (currentPath.includes('/pages/home')) {
      // Home页：显示"X 个视频"
      statsText = videoCount + ' 个视频';
      if (liveCount > 0) {
        statsText += ', ' + liveCount + ' 个直播';
      }
    } else if (currentPath.includes('/pages/s')) {
      // 搜索页：显示"X 个动态, Y 个直播"
      if (liveCount > 0) {
        statsText = videoCount + ' 个动态, ' + liveCount + ' 个直播';
      } else {
        statsText = videoCount + ' 个动态';
      }
    } else if (currentPath.includes('/pages/profile')) {
      // Profile页：显示"X 个视频, Y 个直播回放"
      if (liveCount > 0) {
        statsText = videoCount + ' 个视频, ' + liveCount + ' 个直播回放';
      } else {
        statsText = videoCount + ' 个视频';
      }
    } else {
      // 其他页面：默认显示
      if (liveCount > 0) {
        statsText = videoCount + ' 个视频, ' + liveCount + ' 个直播';
      } else {
        statsText = allVideos.length + ' 个';
      }
    }

    countElement.textContent = statsText;
  }

  // 重新渲染列表
  __render_batch_video_list__();

  if (newCount > 0) {
    console.log('[批量下载] UI已更新，新增', newCount, '个视频');
  }
}

// ==================== 渲染视频列表 ====================
function __render_batch_video_list__() {
  var pageVideos = __wx_batch_download_manager__.getCurrentPageVideos();
  var listContainer = document.getElementById('batch-list');
  if (!listContainer) return;

  listContainer.innerHTML = '';

  for (var i = 0; i < pageVideos.length; i++) {
    var video = pageVideos[i];
    var isSelected = __wx_batch_download_manager__.selectedItems[video.id];

    // 调试：打印视频类型和下载状态
    if (i === 0) {
      console.log('[批量下载] 第一个视频调试信息:', {
        id: video.id,
        title: (video.title || '').substring(0, 30),
        type: video.type,
        canDownload: video.canDownload,
        hasUrl: !!video.url,
        hasKey: video.key !== undefined
      });
    }

    var item = document.createElement('div');
    item.style.cssText = 'display:flex;align-items:flex-start;padding:8px;background:rgba(255,255,255,0.05);border-radius:6px;cursor:pointer;transition:background 0.2s;gap:10px;';
    item.onmouseover = function () { this.style.background = 'rgba(255,255,255,0.08)'; };
    item.onmouseout = function () { this.style.background = 'rgba(255,255,255,0.05)'; };

    // 提取视频信息（兼容多种数据格式）
    var media = video.objectDesc && video.objectDesc.media && video.objectDesc.media[0];

    // 判断是否是直播（不能下载）- 必须在使用前定义
    var isLive = video.type === 'live';
    // 只有明确标记为 false 才不能下载，其他情况（undefined、true）都可以下载
    var canDownload = video.canDownload !== false && video.type !== 'live';

    // 复选框
    var checkbox = document.createElement('input');
    checkbox.type = 'checkbox';
    checkbox.checked = isSelected;
    checkbox.style.cssText = 'margin-top:4px;cursor:pointer;flex-shrink:0;';
    checkbox.dataset.videoId = video.id;
    // 如果是直播或不能下载，禁用复选框
    if (isLive || !canDownload) {
      checkbox.disabled = true;
      checkbox.style.opacity = '0.5';
      checkbox.style.cursor = 'not-allowed';
    }
    checkbox.onclick = function (e) {
      e.stopPropagation();
      if (!this.disabled) {
        __wx_batch_download_manager__.toggleSelect(this.dataset.videoId, this.checked);
        __update_batch_ui__();
      }
    };

    // 封面URL
    var coverUrl = video.thumbUrl || video.coverUrl || video.fullThumbUrl ||
      (media && media.thumbUrl) || '';

    // 标题
    var title = video.title ||
      (video.objectDesc && video.objectDesc.description) ||
      '无标题';

    // 时长（毫秒）
    var duration = video.duration ||
      (media && (media.videoPlayLen * 1000 || media.durationMs)) || 0;

    // 文件大小（字节）
    var size = video.size ||
      (media && (media.fileSize || media.cdnFileSize)) || 0;

    // 作者
    var nickname = video.nickname ||
      (video.contact && video.contact.nickname) || '';

    // 创建时间
    var createtime = video.createtime || 0;

    // 格式化时长
    var durationStr = '';
    if (duration) {
      var seconds = Math.floor(duration / 1000);
      var minutes = Math.floor(seconds / 60);
      seconds = seconds % 60;
      durationStr = minutes + ':' + (seconds < 10 ? '0' : '') + seconds;
    }

    // 格式化文件大小
    var sizeStr = '';
    if (size) {
      var mb = size / (1024 * 1024);
      sizeStr = mb.toFixed(1) + ' MB';
    }

    // 格式化发布时间
    var publishTime = '';
    if (createtime) {
      var date = new Date(createtime * 1000);
      var month = date.getMonth() + 1;
      var day = date.getDate();
      publishTime = month + '月' + day + '日';
    }

    // 封面容器（带时长标签）
    var thumbContainer = document.createElement('div');
    thumbContainer.style.cssText = 'width:60px;height:40px;border-radius:4px;overflow:hidden;background:#1a1a1a;flex-shrink:0;position:relative;';

    if (coverUrl) {
      var thumbImg = document.createElement('img');
      thumbImg.src = coverUrl;
      thumbImg.style.cssText = 'width:100%;height:100%;object-fit:cover;';
      thumbContainer.appendChild(thumbImg);
    } else {
      var noThumb = document.createElement('div');
      noThumb.style.cssText = 'width:100%;height:100%;display:flex;align-items:center;justify-content:center;color:#666;font-size:12px;';
      noThumb.textContent = '无封面';
      thumbContainer.appendChild(noThumb);
    }

    // 直播标签（左上角）
    if (isLive) {
      var liveLabel = document.createElement('div');
      liveLabel.style.cssText = 'position:absolute;top:4px;left:4px;background:#fa5151;color:#fff;font-size:10px;padding:2px 4px;border-radius:2px;font-weight:500;';
      liveLabel.textContent = '直播';
      thumbContainer.appendChild(liveLabel);
    }

    // 时长标签（右下角）
    if (durationStr && !isLive) {
      var durationLabel = document.createElement('div');
      durationLabel.style.cssText = 'position:absolute;bottom:4px;right:4px;background:rgba(0,0,0,0.8);color:#fff;font-size:11px;padding:2px 4px;border-radius:2px;';
      durationLabel.textContent = durationStr;
      thumbContainer.appendChild(durationLabel);
    }

    // 信息容器
    var info = document.createElement('div');
    info.style.cssText = 'flex:1;min-width:0;display:flex;flex-direction:column;gap:4px;';

    // 标题
    var titleDiv = document.createElement('div');
    titleDiv.style.cssText = 'font-size:13px;color:#fff;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;line-height:1.4;';
    titleDiv.textContent = title;

    // 如果是直播回放，添加回放标签
    if (video.type === 'live_replay') {
      var replayBadge = document.createElement('span');
      replayBadge.style.cssText = 'display:inline-block;margin-left:6px;background:#fa5151;color:#fff;font-size:10px;padding:2px 4px;border-radius:2px;vertical-align:middle;';
      replayBadge.textContent = '回放';
      titleDiv.appendChild(replayBadge);
    }
    // 如果是直播且不能下载，添加提示
    else if (isLive || !canDownload) {
      titleDiv.style.color = '#999';
      var tipSpan = document.createElement('span');
      tipSpan.style.cssText = 'color:#fa5151;font-size:11px;margin-left:6px;';
      tipSpan.textContent = '(暂不支持下载)';
      titleDiv.appendChild(tipSpan);
    }
    info.appendChild(titleDiv);

    // 详细信息（大小、日期、作者）
    var detailDiv = document.createElement('div');
    detailDiv.style.cssText = 'display:flex;gap:8px;font-size:11px;color:#999;flex-wrap:wrap;';

    var details = [];
    if (sizeStr) details.push('<span>' + sizeStr + '</span>');
    if (publishTime) details.push('<span>' + publishTime + '</span>');
    if (nickname) details.push('<span style="overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:100px;">@' + nickname + '</span>');

    detailDiv.innerHTML = details.join('');
    info.appendChild(detailDiv);

    // 组装列表项
    item.appendChild(checkbox);
    item.appendChild(thumbContainer);
    item.appendChild(info);

    item.onclick = function () {
      // 如果是直播或不能下载，不响应点击
      if (isLive || !canDownload) return;

      var cb = this.querySelector('input[type="checkbox"]');
      cb.checked = !cb.checked;
      __wx_batch_download_manager__.toggleSelect(cb.dataset.videoId, cb.checked);
      __update_batch_ui__();
    };

    listContainer.appendChild(item);
  }

  __update_batch_ui__();
}

function __update_batch_ui__() {
  // 更新页码
  document.getElementById('batch-current-page').textContent = __wx_batch_download_manager__.currentPage;
  document.getElementById('batch-total-pages').textContent = __wx_batch_download_manager__.getTotalPages();

  // 更新选中数量
  var selectedCount = __wx_batch_download_manager__.getSelectedVideos().length;
  document.getElementById('batch-selected-count').textContent = '已选 ' + selectedCount + ' 个';

  // 更新全选状态
  var pageVideos = __wx_batch_download_manager__.getCurrentPageVideos();
  var allSelected = pageVideos.length > 0 && pageVideos.every(function (video) {
    return __wx_batch_download_manager__.selectedItems[video.id];
  });
  var selectAllCheckbox = document.getElementById('batch-select-all');
  if (selectAllCheckbox) {
    selectAllCheckbox.checked = allSelected;
  }
}

// ==================== 批量下载 ====================
async function __batch_download_selected__() {
  var selectedVideos = __wx_batch_download_manager__.getSelectedVideos();

  if (selectedVideos.length === 0) {
    __wx_log({ msg: '❌ 请先选择要下载的视频' });
    return;
  }

  if (__wx_batch_download_manager__.isDownloading) {
    __wx_log({ msg: '⚠️ 正在下载中，请等待...' });
    return;
  }

  // 格式化视频数据（使用 WXU.format_feed 统一格式）
  var formattedVideos = [];
  for (var i = 0; i < selectedVideos.length; i++) {
    var video = selectedVideos[i];

    // 跳过不能下载的项目（直播等）
    if (video.canDownload === false || video.type === 'live') {
      continue;
    }

    // 如果已经格式化过（有 url 和 key 字段），直接使用
    if (video.url && video.key !== undefined) {
      formattedVideos.push(video);
    } else if (video.objectDesc) {
      // 否则使用 format_feed 格式化
      var formatted = WXU.format_feed(video);
      if (formatted && formatted.type === 'media' && formatted.canDownload !== false) {
        formattedVideos.push(formatted);
      }
    }
  }

  if (formattedVideos.length === 0) {
    __wx_log({ msg: '❌ 没有可下载的视频' });
    return;
  }

  // 设置下载状态
  __wx_batch_download_manager__.isDownloading = true;
  __wx_batch_download_manager__.stopSignal = false;

  __wx_log({ msg: '🚀 开始批量下载 ' + formattedVideos.length + ' 个视频（后端并发）...' });

  // 显示进度和取消按钮
  var progressDiv = document.getElementById('batch-download-progress');
  var progressText = document.getElementById('batch-progress-text');
  var progressBar = document.getElementById('batch-progress-bar');
  var downloadBtn = document.getElementById('batch-download-btn');
  var cancelBtn = document.getElementById('batch-cancel-btn');

  if (progressDiv) progressDiv.style.display = 'block';
  if (downloadBtn) {
    downloadBtn.textContent = '下载中...';
    downloadBtn.style.opacity = '0.7';
    downloadBtn.style.cursor = 'not-allowed';
  }
  if (cancelBtn) {
    cancelBtn.style.display = 'block';
    cancelBtn.textContent = '取消';
    cancelBtn.disabled = false;
  }

  try {
    // 构建批量下载请求数据
    var batchVideos = formattedVideos.map(function(video) {
      var authorName = video.nickname || (video.contact && video.contact.nickname) || '未知作者';
      var resolution = '';
      var width = 0, height = 0;

      if (video.spec && video.spec.length > 0) {
        var firstSpec = video.spec[0];
        width = firstSpec.width || 0;
        height = firstSpec.height || 0;
        resolution = width && height ? (width + 'x' + height) : '';
      }

      return {
        id: video.id || '',
        url: video.url || '',
        title: video.title || video.id || String(Date.now()),
        author: authorName,
        key: video.key || '',
        resolution: resolution
      };
    });

    // 调用后端批量下载接口
    var response = await fetch('/__wx_channels_api/batch_start', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        videos: batchVideos,
        forceRedownload: __wx_batch_download_manager__.forceRedownload
      })
    });

    if (!response.ok) {
      throw new Error('HTTP ' + response.status + ': ' + response.statusText);
    }

    var result = await response.json();
    
    // 检查响应格式（兼容两种格式）
    var data = result.data || result;
    if (!result.success && result.code !== 0) {
      throw new Error(result.error || result.message || '启动批量下载失败');
    }

    __wx_log({ msg: '✅ 批量下载已启动，并发数: ' + (data.concurrency || 5) });

    // 等待100ms后立即查询一次进度（避免错过快速完成的下载）
    await new Promise(function(resolve) { setTimeout(resolve, 100); });
    
    // 立即查询一次进度
    try {
      var progressRes = await fetch('/__wx_channels_api/batch_progress', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      });
      if (progressRes.ok) {
        var progressData = await progressRes.json();
        var data = progressData.data || progressData;
        if (progressData.success || progressData.code === 0 || data.total !== undefined) {
          var total = data.total || 0;
          var done = data.done || 0;
          var failed = data.failed || 0;
          
          // 如果已经完成，直接显示结果并返回
          if (total > 0 && done + failed >= total) {
            __wx_log({ msg: '✅ 批量下载完成: 成功 ' + done + ' 个, 失败 ' + failed + ' 个' });
            __reset_batch_download_ui__();
            return;
          }
        }
      }
    } catch (e) {
      console.error('[批量下载] 初始进度查询失败:', e);
    }

    // 开始轮询进度
    var pollInterval = setInterval(async function() {
      // 检查取消信号
      if (__wx_batch_download_manager__.stopSignal) {
        clearInterval(pollInterval);
        // 调用取消接口
        try {
          await fetch('/__wx_channels_api/batch_cancel', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
          });
          __wx_log({ msg: '⏹️ 批量下载已取消' });
        } catch (e) {
          console.error('[批量下载] 取消失败:', e);
        }
        __reset_batch_download_ui__();
        return;
      }

      try {
        var progressRes = await fetch('/__wx_channels_api/batch_progress', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' }
        });

        if (progressRes.ok) {
          var progressData = await progressRes.json();
          console.log('[批量下载] 进度数据:', progressData);
          
          // 兼容两种响应格式
          var data = progressData.data || progressData;
          if (progressData.success || progressData.code === 0 || data.total !== undefined) {
            var total = data.total || 0;
            var done = data.done || 0;
            var failed = data.failed || 0;
            var running = data.running || 0;

            console.log('[批量下载] 解析后:', { total: total, done: done, failed: failed, running: running });

            // 更新进度显示
            if (progressText) {
              progressText.textContent = done + '/' + total;
              if (running > 0) {
                progressText.textContent += ' (并发: ' + running + ')';
              }
              
              // 显示当前正在下载的任务的详细进度
              if (data.currentTasks && data.currentTasks.length > 0) {
                var currentTask = data.currentTasks[0];
                if (currentTask.progress > 0) {
                  progressText.textContent += ' - ' + currentTask.progress.toFixed(1) + '%';
                }
              }
            }
            if (progressBar) {
              // 计算总体进度：已完成的 + 正在下载的平均进度
              var overallProgress = 0;
              if (total > 0) {
                // 已完成的视频占比
                overallProgress = (done / total) * 100;
                
                // 加上正在下载的视频的平均进度
                if (data.currentTasks && data.currentTasks.length > 0) {
                  var downloadingProgress = 0;
                  for (var i = 0; i < data.currentTasks.length; i++) {
                    downloadingProgress += (data.currentTasks[i].progress || 0);
                  }
                  // 平均进度
                  var avgProgress = downloadingProgress / data.currentTasks.length;
                  // 正在下载的视频占总数的比例
                  var downloadingRatio = data.currentTasks.length / total;
                  // 加到总进度中
                  overallProgress += (avgProgress * downloadingRatio);
                }
              }
              
              progressBar.style.width = overallProgress + '%';
              console.log('[批量下载] 进度条宽度:', overallProgress.toFixed(1) + '%');
            }

            // 检查是否完成
            if (total > 0 && done + failed >= total && running === 0) {
              clearInterval(pollInterval);
              __wx_log({ msg: '✅ 批量下载完成: 成功 ' + done + ' 个, 失败 ' + failed + ' 个' });
              __reset_batch_download_ui__();
            }
          } else {
            console.warn('[批量下载] 无效的进度数据格式:', progressData);
          }
        } else {
          console.error('[批量下载] 进度查询失败:', progressRes.status);
        }
      } catch (e) {
        console.error('[批量下载] 轮询进度失败:', e);
      }
    }, 2000); // 每2秒轮询一次

  } catch (err) {
    __wx_log({ msg: '❌ 批量下载失败: ' + (err.message || err) });
    console.error('[批量下载] 错误:', err);
    __reset_batch_download_ui__();
  }
}

// 重置批量下载UI状态
function __reset_batch_download_ui__() {
  __wx_batch_download_manager__.isDownloading = false;
  __wx_batch_download_manager__.stopSignal = false;

  var downloadBtn = document.getElementById('batch-download-btn');
  var cancelBtn = document.getElementById('batch-cancel-btn');
  var progressDiv = document.getElementById('batch-download-progress');
  var progressBar = document.getElementById('batch-progress-bar');

  if (downloadBtn) {
    downloadBtn.textContent = '开始下载';
    downloadBtn.style.opacity = '1';
    downloadBtn.style.cursor = 'pointer';
  }
  if (cancelBtn) {
    cancelBtn.style.display = 'none';
  }

  // 延迟隐藏进度条
  setTimeout(function () {
    if (progressDiv) progressDiv.style.display = 'none';
    if (progressBar) progressBar.style.width = '0%';
  }, 3000);
}

console.log('[batch_download.js] 通用批量下载模块加载完成');
