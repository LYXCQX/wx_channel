/**
 * @file Profile页面功能模块 - 事件监听和批量下载
 */

// ==================== Profile页面视频列表采集器 ====================
window.__wx_channels_profile_collector = {
  videos: [],
  isCollecting: false,
  batchDownloading: false,
  downloadProgress: { current: 0, total: 0 },
  _lastLogMessage: '',
  _lastTipVideoCount: 0,
  _lastTipLiveReplayCount: 0,
  _forceRedownload: false,
  _stopSignal: false, // 取消下载信号
  _selectedVideos: {}, // 选中的视频 {videoId: true}
  _currentPage: 1, // 当前页码
  _pageSize: 50, // 每页显示数量
  _maxVideos: 200, // 最多采集200个视频

  // 初始化
  init: function() {
    var self = this;
    // 延迟初始化UI
    setTimeout(function() {
      self.injectToolbarDownloadIcon();
    }, 2000);
  },

  // 在Profile页面工具栏注入下载图标
  injectToolbarDownloadIcon: function() {
    var self = this;
    
    // 查找工具栏图标容器
    var findIconContainer = function() {
      var container = document.querySelector('div[data-v-bf57a568].flex.items-center');
      if (container) return container;
      var parent = document.querySelector('div.flex-initial.flex-shrink-0.pl-6');
      if (parent) {
        container = parent.querySelector('.flex.items-center');
        if (container) return container;
      }
      return null;
    };
    
    var tryInject = function() {
      var container = findIconContainer();
      if (!container) return false;
      if (container.querySelector('#wx-profile-download-icon')) return true;
      
      // 创建下载图标 - 使用与原有图标一致的样式
      var iconWrapper = document.createElement('div');
      iconWrapper.id = 'wx-profile-download-icon';
      iconWrapper.className = 'mr-4 h-6 w-6 flex-initial flex-shrink-0 text-fg-0 cursor-pointer';
      iconWrapper.title = '批量下载';
      // 使用 fill 而非 stroke，与原有图标风格一致
      iconWrapper.innerHTML = '<svg class="h-full w-full" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none"><path fill-rule="evenodd" clip-rule="evenodd" d="M12 3C12.3314 3 12.6 3.26863 12.6 3.6V13.1515L15.5757 10.1757C15.8101 9.94142 16.1899 9.94142 16.4243 10.1757C16.6586 10.4101 16.6586 10.7899 16.4243 11.0243L12.4243 15.0243C12.1899 15.2586 11.8101 15.2586 11.5757 15.0243L7.57574 11.0243C7.34142 10.7899 7.34142 10.4101 7.57574 10.1757C7.81005 9.94142 8.18995 9.94142 8.42426 10.1757L11.4 13.1515V3.6C11.4 3.26863 11.6686 3 12 3ZM3.6 14.4C3.93137 14.4 4.2 14.6686 4.2 15V19.2C4.2 19.5314 4.46863 19.8 4.8 19.8H19.2C19.5314 19.8 19.8 19.5314 19.8 19.2V15C19.8 14.6686 20.0686 14.4 20.4 14.4C20.7314 14.4 21 14.6686 21 15V19.2C21 20.1941 20.1941 21 19.2 21H4.8C3.80589 21 3 20.1941 3 19.2V15C3 14.6686 3.26863 14.4 3.6 14.4Z" fill="currentColor"></path></svg>';
      
      // 点击事件 - 显示/隐藏批量下载面板
      iconWrapper.onclick = function() {
        // 使用通用批量下载组件
        if (window.__wx_batch_download_manager__ && window.__wx_batch_download_manager__.isVisible) {
          __close_batch_download_ui__();
        } else {
          // 显示批量下载UI（包含视频和直播回放）
          var filteredVideos = self.filterLivePictureVideos(self.videos).filter(function(v) {
            return v && (v.type === 'media' || v.type === 'live');
          });
          
          if (filteredVideos.length === 0) {
            __wx_log({ msg: '⚠️ 暂无视频数据' });
            return;
          }
          
          __show_batch_download_ui__(filteredVideos, 'Profile - 视频列表');
        }
      };
      
      container.insertBefore(iconWrapper, container.firstChild);
      console.log('[Profile] ✅ 下载图标已注入到工具栏');
      return true;
    };
    
    if (tryInject()) return;
    
    var observer = new MutationObserver(function(mutations, obs) {
      if (tryInject()) { obs.disconnect(); }
    });
    observer.observe(document.body, { childList: true, subtree: true });
    setTimeout(function() { observer.disconnect(); }, 5000);
  },

  // 过滤掉正在直播的图片类型数据
  filterLivePictureVideos: function(videos) {
    return (videos || []).filter(function(v) {
      if (v.type === 'picture' && v.contact && v.contact.liveStatus === 1) {
        return false;
      }
      return true;
    });
  },

  // 清理HTML标签
  cleanHtmlTags: function(text) {
    if (!text || typeof text !== 'string') return text || '';
    var tempDiv = document.createElement('div');
    tempDiv.innerHTML = text;
    var cleaned = tempDiv.textContent || tempDiv.innerText || '';
    return cleaned.trim();
  },

  // 从API添加单个视频
  addVideoFromAPI: function(videoData) {
    var self = this;
    if (!videoData || !videoData.id) return;

    // 过滤掉正在直播的图片类型数据
    if (videoData.type === 'picture' && videoData.contact && videoData.contact.liveStatus === 1) {
      return;
    }

    // 限制最多200个视频
    if (this.videos.length >= this._maxVideos) {
      if (this.videos.length === this._maxVideos) {
        __wx_log({ msg: '⚠️ [Profile] 已达到最大采集数量 ' + this._maxVideos + ' 个' });
      }
      return;
    }

    // 清理标题
    if (videoData.title) {
      videoData.title = this.cleanHtmlTags(videoData.title);
    }

    // 检查是否已存在
    var exists = this.videos.some(function(v) { return v.id === videoData.id; });
    if (!exists) {
      this.videos.push(videoData);
      // 默认选中新添加的视频
      this._selectedVideos[videoData.id] = true;
      console.log('[Profile] 新增视频:', (videoData.title || '').substring(0, 30));

      // 每10个视频发送一次日志
      var filteredVideos = this.filterLivePictureVideos(this.videos);
      var videoCount = filteredVideos.filter(function(v) { return v && v.type === 'media'; }).length;
      var liveReplayCount = filteredVideos.filter(function(v) { return v && v.type === 'live'; }).length;

      if (videoCount > 0 && videoCount % 10 === 0 && videoCount !== this._lastTipVideoCount) {
        this._lastTipVideoCount = videoCount;
        var msg = '📊 [Profile] 已采集 ' + videoCount + ' 个视频';
        if (liveReplayCount > 0) msg += ', ' + liveReplayCount + ' 个直播回放';
        __wx_log({ msg: msg });
      }

      // 更新UI（使用通用批量下载组件，包含视频和直播回放）
      if (window.__wx_batch_download_manager__ && window.__wx_batch_download_manager__.isVisible) {
        var filteredVideos = this.filterLivePictureVideos(this.videos).filter(function(v) {
          return v && (v.type === 'media' || v.type === 'live');
        });
        __update_batch_download_ui__(filteredVideos, 'Profile - 视频列表');
      }
    }
  },

  // 添加批量下载UI
  addBatchDownloadUI: function() {
    var self = this;
    var existingUI = document.getElementById('wx-channels-batch-download-ui');
    if (existingUI) existingUI.remove();

    var ui = document.createElement('div');
    ui.id = 'wx-channels-batch-download-ui';
    // 移除 flex 布局，使用普通布局
    ui.style.cssText = 'position:fixed;top:60px;right:20px;background:#2b2b2b;color:#e5e5e5;padding:0;border-radius:8px;z-index:99999;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif;font-size:14px;width:400px;max-height:80vh;box-shadow:0 8px 24px rgba(0,0,0,0.5);display:none;overflow:hidden;';

    ui.innerHTML = 
      // 标题栏
      '<div style="padding:16px 20px;border-bottom:1px solid rgba(255,255,255,0.08);display:flex;justify-content:space-between;align-items:center;">' +
        '<div style="font-size:15px;font-weight:500;color:#fff;">批量下载</div>' +
        '<div id="video-count" style="font-size:13px;color:#999;">0 个视频</div>' +
      '</div>' +
      
      // 视频列表区域（可滚动，固定高度）
      '<div id="video-list-container" style="overflow-y:auto;padding:12px 20px;max-height:200px;">' +
        '<div id="video-list" style="display:flex;flex-direction:column;gap:8px;"></div>' +
      '</div>' +
      
      // 分页控制
      '<div id="pagination-container" style="padding:12px 20px;border-top:1px solid rgba(255,255,255,0.08);border-bottom:1px solid rgba(255,255,255,0.08);display:flex;justify-content:space-between;align-items:center;">' +
        '<div style="font-size:13px;color:#999;">第 <span id="current-page">1</span> / <span id="total-pages">1</span> 页</div>' +
        '<div style="display:flex;gap:8px;">' +
          '<button id="prev-page-btn" style="background:rgba(255,255,255,0.08);color:#999;border:none;padding:4px 12px;border-radius:4px;cursor:pointer;font-size:13px;">上一页</button>' +
          '<button id="next-page-btn" style="background:rgba(255,255,255,0.08);color:#999;border:none;padding:4px 12px;border-radius:4px;cursor:pointer;font-size:13px;">下一页</button>' +
        '</div>' +
      '</div>' +
      
      // 主要操作区
      '<div style="padding:16px 20px;">' +
        // 全选/取消全选
        '<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:12px;">' +
          '<label style="display:flex;align-items:center;cursor:pointer;font-size:13px;color:#999;user-select:none;">' +
            '<input type="checkbox" id="select-all-checkbox" style="margin-right:8px;cursor:pointer;" />' +
            '<span>全选当前页</span>' +
          '</label>' +
          '<span id="selected-count" style="font-size:13px;color:#07c160;">已选 0 个</span>' +
        '</div>' +
        
        // 下载和取消按钮容器
        '<div style="display:flex;gap:8px;margin-bottom:12px;">' +
          '<button id="batch-download-btn" style="flex:1;background:#07c160;color:#fff;border:none;padding:8px 12px;border-radius:6px;cursor:pointer;font-size:14px;font-weight:500;transition:background 0.2s;">开始下载</button>' +
          '<button id="cancel-download-btn" style="flex:0 0 25%;background:#fa5151;color:#fff;border:none;padding:8px 12px;border-radius:6px;cursor:pointer;font-size:14px;font-weight:500;display:none;">取消</button>' +
        '</div>' +
        
        // 下载进度
        '<div id="download-progress" style="display:none;margin-bottom:12px;">' +
          '<div style="display:flex;justify-content:space-between;margin-bottom:8px;font-size:13px;color:#999;">' +
            '<span>下载进度</span>' +
            '<span id="progress-text">0/0</span>' +
          '</div>' +
          '<div style="background:rgba(255,255,255,0.08);height:6px;border-radius:3px;overflow:hidden;">' +
            '<div id="progress-bar" style="background:#07c160;height:100%;width:0%;border-radius:3px;transition:width 0.3s;"></div>' +
          '</div>' +
        '</div>' +
        
        // 强制重新下载选项
        '<label style="display:flex;align-items:center;cursor:pointer;font-size:13px;color:#999;user-select:none;">' +
          '<input type="checkbox" id="force-redownload-checkbox" style="margin-right:8px;cursor:pointer;" />' +
          '<span>强制重新下载</span>' +
        '</label>' +
      '</div>' +
      
      // 次要操作区
      '<div style="padding:12px 20px;border-top:1px solid rgba(255,255,255,0.08);display:flex;gap:8px;">' +
        '<button id="export-videos-btn" style="flex:1;background:transparent;color:#999;border:1px solid rgba(255,255,255,0.12);padding:8px 12px;border-radius:6px;cursor:pointer;font-size:13px;transition:all 0.2s;">导出列表</button>' +
        '<button id="clear-videos-btn" style="flex:1;background:transparent;color:#999;border:1px solid rgba(255,255,255,0.12);padding:8px 12px;border-radius:6px;cursor:pointer;font-size:13px;transition:all 0.2s;">清空列表</button>' +
      '</div>';

    document.body.appendChild(ui);

    // 绑定事件
    setTimeout(function() {
      var batchBtn = document.getElementById('batch-download-btn');
      var cancelBtn = document.getElementById('cancel-download-btn');
      var forceCheckbox = document.getElementById('force-redownload-checkbox');
      var exportBtn = document.getElementById('export-videos-btn');
      var clearBtn = document.getElementById('clear-videos-btn');
      var selectAllCheckbox = document.getElementById('select-all-checkbox');
      var prevPageBtn = document.getElementById('prev-page-btn');
      var nextPageBtn = document.getElementById('next-page-btn');

      // 按钮悬停效果
      if (batchBtn) {
        batchBtn.addEventListener('mouseenter', function() { this.style.background = '#06ad56'; });
        batchBtn.addEventListener('mouseleave', function() { this.style.background = '#07c160'; });
        batchBtn.addEventListener('click', function() { self.startBatchDownload(); });
      }

      if (cancelBtn) {
        cancelBtn.addEventListener('mouseenter', function() { this.style.background = '#e84545'; });
        cancelBtn.addEventListener('mouseleave', function() { this.style.background = '#fa5151'; });
        cancelBtn.addEventListener('click', function() { self.cancelDownload(); });
      }

      if (exportBtn) {
        exportBtn.addEventListener('mouseenter', function() { 
          this.style.background = 'rgba(255,255,255,0.08)'; 
          this.style.color = '#fff'; 
        });
        exportBtn.addEventListener('mouseleave', function() { 
          this.style.background = 'transparent'; 
          this.style.color = '#999'; 
        });
        exportBtn.addEventListener('click', function() { self.exportVideoList(); });
      }

      if (clearBtn) {
        clearBtn.addEventListener('mouseenter', function() { 
          this.style.background = 'rgba(255,255,255,0.08)'; 
          this.style.color = '#fff'; 
        });
        clearBtn.addEventListener('mouseleave', function() { 
          this.style.background = 'transparent'; 
          this.style.color = '#999'; 
        });
        clearBtn.addEventListener('click', function() { self.clearVideoList(); });
      }

      if (forceCheckbox) {
        forceCheckbox.addEventListener('change', function() {
          self._forceRedownload = this.checked;
        });
      }

      if (selectAllCheckbox) {
        selectAllCheckbox.addEventListener('change', function() {
          self.toggleSelectAll(this.checked);
        });
      }

      if (prevPageBtn) {
        prevPageBtn.addEventListener('click', function() { self.goToPrevPage(); });
      }

      if (nextPageBtn) {
        nextPageBtn.addEventListener('click', function() { self.goToNextPage(); });
      }
    }, 100);
  },

  // 更新批量下载UI
  updateBatchDownloadUI: function() {
    var countElement = document.getElementById('video-count');
    if (countElement) {
      var filteredVideos = this.filterLivePictureVideos(this.videos);
      var videoCount = filteredVideos.filter(function(v) { return v && v.type === 'media'; }).length;
      var liveReplayCount = filteredVideos.filter(function(v) { return v && v.type === 'live_replay'; }).length;
      var text = videoCount + ' 个视频';
      if (liveReplayCount > 0) text += ' + ' + liveReplayCount + ' 回放';
      countElement.textContent = text;
    }
  },

  // 取消下载
  cancelDownload: function() {
    if (this.batchDownloading) {
      this._stopSignal = true;
      __wx_log({ msg: '⏹️ [Profile] 正在取消下载...' });
      
      var cancelBtn = document.getElementById('cancel-download-btn');
      if (cancelBtn) {
        cancelBtn.textContent = '取消中...';
        cancelBtn.disabled = true;
      }
    }
  },

  // 开始批量下载
  startBatchDownload: async function() {
    var self = this;
    if (this.batchDownloading) {
      __wx_log({ msg: '⚠️ 正在下载中，请等待...' });
      return;
    }

    // 获取选中的视频
    var selectedIds = Object.keys(this._selectedVideos);
    if (selectedIds.length === 0) {
      __wx_log({ msg: '⚠️ 请先选择要下载的视频' });
      WXU.toast('请先选择要下载的视频');
      return;
    }

    var videosToDownload = this.filterLivePictureVideos(this.videos).filter(function(v) {
      return v && v.type === 'media' && v.url && self._selectedVideos[v.id] === true;
    });

    if (videosToDownload.length === 0) {
      __wx_log({ msg: '⚠️ 没有可下载的视频' });
      WXU.toast('没有可下载的视频');
      return;
    }

    this.batchDownloading = true;
    this._stopSignal = false;
    this.downloadProgress = { current: 0, total: videosToDownload.length };

    __wx_log({ msg: '🚀 [Profile] 开始批量下载 ' + videosToDownload.length + ' 个视频' });

    // 显示进度和取消按钮
    var progressDiv = document.getElementById('download-progress');
    var progressText = document.getElementById('progress-text');
    var progressBar = document.getElementById('progress-bar');
    var batchBtn = document.getElementById('batch-download-btn');
    var cancelBtn = document.getElementById('cancel-download-btn');

    if (progressDiv) progressDiv.style.display = 'block';
    if (batchBtn) {
      batchBtn.textContent = '下载中...';
      batchBtn.style.opacity = '0.7';
      batchBtn.style.cursor = 'not-allowed';
    }
    if (cancelBtn) {
      cancelBtn.style.display = 'block';
      cancelBtn.textContent = '取消';
      cancelBtn.disabled = false;
    }

    var successCount = 0;
    var skipCount = 0;
    var failCount = 0;

    for (var i = 0; i < videosToDownload.length; i++) {
      // 检查取消信号
      if (this._stopSignal) {
        __wx_log({ msg: '⏹️ [Profile] 下载已取消，已完成 ' + i + '/' + videosToDownload.length });
        break;
      }

      var video = videosToDownload[i];
      this.downloadProgress.current = i + 1;

      // 更新进度
      if (progressText) progressText.textContent = (i + 1) + '/' + videosToDownload.length;
      if (progressBar) progressBar.style.width = ((i + 1) / videosToDownload.length * 100) + '%';

      try {
        // 构建下载请求
        var authorName = video.nickname || (video.contact && video.contact.nickname) || '未知作者';
        var filename = video.title || video.id || String(Date.now());
        var resolution = '';
        var width = 0, height = 0, fileFormat = '';

        if (video.spec && video.spec.length > 0) {
          var firstSpec = video.spec[0];
          width = firstSpec.width || 0;
          height = firstSpec.height || 0;
          resolution = width && height ? (width + 'x' + height) : '';
          fileFormat = firstSpec.fileFormat || '';
        }

        var requestData = {
          videoUrl: video.url,
          videoId: video.id || '',
          title: filename,
          author: authorName,
          key: video.key || '',
          forceSave: this._forceRedownload,
          resolution: resolution,
          width: width,
          height: height,
          fileFormat: fileFormat
        };

        var response = await fetch('/__wx_channels_api/download_video', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(requestData)
        });

        var result = await response.json();

        if (result.success) {
          if (result.skipped) {
            skipCount++;
          } else {
            successCount++;
          }
        } else {
          failCount++;
          console.error('[Profile] 下载失败:', video.title, result.error);
        }

        // 添加延迟避免请求过快
        await WXU.sleep(300);

      } catch (err) {
        failCount++;
        console.error('[Profile] 下载出错:', video.title, err);
      }
    }

    // 下载完成，重置状态
    this.batchDownloading = false;
    this._stopSignal = false;

    if (batchBtn) {
      batchBtn.textContent = '开始下载';
      batchBtn.style.opacity = '1';
      batchBtn.style.cursor = 'pointer';
    }
    if (cancelBtn) {
      cancelBtn.style.display = 'none';
    }

    var summaryMsg = '✅ [Profile] 批量下载完成: 成功 ' + successCount + ' 个';
    if (skipCount > 0) summaryMsg += ', 跳过 ' + skipCount + ' 个';
    if (failCount > 0) summaryMsg += ', 失败 ' + failCount + ' 个';

    __wx_log({ msg: summaryMsg });
    WXU.toast(summaryMsg);
  },

  // 导出视频列表
  exportVideoList: function() {
    var filteredVideos = this.filterLivePictureVideos(this.videos).filter(function(v) {
      return v && v.type === 'media';
    });

    if (filteredVideos.length === 0) {
      WXU.toast('没有可导出的视频');
      return;
    }

    var exportData = filteredVideos.map(function(v) {
      return {
        id: v.id,
        title: v.title,
        url: v.url,
        coverUrl: v.coverUrl || v.thumbUrl,
        duration: v.duration,
        size: v.size,
        nickname: v.nickname || (v.contact && v.contact.nickname) || '',
        createtime: v.createtime
      };
    });

    var blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
    var url = URL.createObjectURL(blob);
    var a = document.createElement('a');
    a.href = url;
    a.download = 'profile_videos_' + new Date().toISOString().slice(0, 10) + '.json';
    a.click();
    URL.revokeObjectURL(url);

    __wx_log({ msg: '📤 [Profile] 已导出 ' + exportData.length + ' 个视频' });
  },

  // 清空视频列表
  clearVideoList: function() {
    if (this.batchDownloading) {
      WXU.toast('下载中，无法清空');
      return;
    }

    var count = this.videos.length;
    this.videos = [];
    this._selectedVideos = {};
    this._currentPage = 1;
    this._lastTipVideoCount = 0;
    this._lastTipLiveReplayCount = 0;
    this.updateBatchDownloadUI();
    this.renderVideoList();

    __wx_log({ msg: '🗑️ [Profile] 已清空 ' + count + ' 个视频' });
    WXU.toast('已清空视频列表');
  },

  // 渲染视频列表
  renderVideoList: function() {
    var listContainer = document.getElementById('video-list');
    if (!listContainer) return;

    var filteredVideos = this.filterLivePictureVideos(this.videos);
    var totalPages = Math.ceil(filteredVideos.length / this._pageSize);
    var startIndex = (this._currentPage - 1) * this._pageSize;
    var endIndex = Math.min(startIndex + this._pageSize, filteredVideos.length);
    var pageVideos = filteredVideos.slice(startIndex, endIndex);

    // 清空列表
    listContainer.innerHTML = '';

    // 渲染当前页的视频
    var self = this;
    pageVideos.forEach(function(video) {
      var isSelected = self._selectedVideos[video.id] === true;
      var isLiveReplay = video.type === 'live_replay';
      
      // 格式化时长
      var duration = '';
      if (video.duration) {
        var seconds = Math.floor(video.duration / 1000);
        var minutes = Math.floor(seconds / 60);
        seconds = seconds % 60;
        duration = minutes + ':' + (seconds < 10 ? '0' : '') + seconds;
      }

      // 格式化文件大小
      var fileSize = '';
      if (video.size) {
        var mb = video.size / (1024 * 1024);
        fileSize = mb.toFixed(1) + ' MB';
      }

      // 格式化发布时间
      var publishTime = '';
      if (video.createtime) {
        var date = new Date(video.createtime * 1000);
        var month = date.getMonth() + 1;
        var day = date.getDate();
        publishTime = month + '月' + day + '日';
      }

      // 获取封面图
      var coverUrl = video.thumbUrl || video.coverUrl || video.fullThumbUrl || '';

      var item = document.createElement('div');
      item.style.cssText = 'display:flex;align-items:flex-start;padding:8px;background:rgba(255,255,255,0.05);border-radius:6px;cursor:pointer;transition:background 0.2s;gap:10px;';
      item.innerHTML = 
        '<input type="checkbox" ' + (isSelected ? 'checked' : '') + ' style="margin-top:4px;cursor:pointer;flex-shrink:0;" data-video-id="' + video.id + '" />' +
        // 封面图
        '<div style="width:60px;height:40px;border-radius:4px;overflow:hidden;background:#1a1a1a;flex-shrink:0;position:relative;">' +
          (coverUrl ? '<img src="' + coverUrl + '" style="width:100%;height:100%;object-fit:cover;" />' : '<div style="width:100%;height:100%;display:flex;align-items:center;justify-content:center;color:#666;font-size:12px;">无封面</div>') +
          // 时长标签
          (duration ? '<div style="position:absolute;bottom:4px;right:4px;background:rgba(0,0,0,0.8);color:#fff;font-size:11px;padding:2px 4px;border-radius:2px;">' + duration + '</div>' : '') +
        '</div>' +
        // 视频信息
        '<div style="flex:1;min-width:0;display:flex;flex-direction:column;gap:4px;">' +
          // 标题（带直播回放标签）
          '<div style="font-size:13px;color:#fff;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;line-height:1.4;">' + 
            (video.title || '无标题') + 
            (isLiveReplay ? '<span style="display:inline-block;margin-left:6px;background:#fa5151;color:#fff;font-size:10px;padding:2px 4px;border-radius:2px;vertical-align:middle;">回放</span>' : '') +
          '</div>' +
          // 详细信息
          '<div style="display:flex;gap:8px;font-size:11px;color:#999;flex-wrap:wrap;">' +
            (fileSize ? '<span>' + fileSize + '</span>' : '') +
            (publishTime ? '<span>' + publishTime + '</span>' : '') +
            (video.nickname ? '<span style="overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:100px;">@' + video.nickname + '</span>' : '') +
          '</div>' +
        '</div>';

      // 悬停效果
      item.onmouseenter = function() { this.style.background = 'rgba(255,255,255,0.08)'; };
      item.onmouseleave = function() { this.style.background = 'rgba(255,255,255,0.05)'; };

      // 点击切换选中状态
      item.onclick = function(e) {
        if (e.target.tagName !== 'INPUT' && e.target.tagName !== 'IMG') {
          var checkbox = this.querySelector('input[type="checkbox"]');
          checkbox.checked = !checkbox.checked;
          self.toggleVideoSelection(video.id, checkbox.checked);
        }
      };

      // 复选框变化事件
      var checkbox = item.querySelector('input[type="checkbox"]');
      checkbox.onchange = function(e) {
        e.stopPropagation();
        self.toggleVideoSelection(video.id, this.checked);
      };

      listContainer.appendChild(item);
    });

    // 更新分页信息
    this.updatePagination(totalPages);
    this.updateSelectedCount();
  },

  // 切换视频选中状态
  toggleVideoSelection: function(videoId, selected) {
    if (selected) {
      this._selectedVideos[videoId] = true;
    } else {
      delete this._selectedVideos[videoId];
    }
    this.updateSelectedCount();
    this.updateSelectAllCheckbox();
  },

  // 全选/取消全选当前页
  toggleSelectAll: function(selectAll) {
    var filteredVideos = this.filterLivePictureVideos(this.videos);
    var startIndex = (this._currentPage - 1) * this._pageSize;
    var endIndex = Math.min(startIndex + this._pageSize, filteredVideos.length);
    var pageVideos = filteredVideos.slice(startIndex, endIndex);

    var self = this;
    pageVideos.forEach(function(video) {
      if (selectAll) {
        self._selectedVideos[video.id] = true;
      } else {
        delete self._selectedVideos[video.id];
      }
    });

    this.renderVideoList();
  },

  // 更新选中数量显示
  updateSelectedCount: function() {
    var selectedCountEl = document.getElementById('selected-count');
    if (selectedCountEl) {
      var count = Object.keys(this._selectedVideos).length;
      selectedCountEl.textContent = '已选 ' + count + ' 个';
    }
  },

  // 更新全选复选框状态
  updateSelectAllCheckbox: function() {
    var selectAllCheckbox = document.getElementById('select-all-checkbox');
    if (!selectAllCheckbox) return;

    var filteredVideos = this.filterLivePictureVideos(this.videos);
    var startIndex = (this._currentPage - 1) * this._pageSize;
    var endIndex = Math.min(startIndex + this._pageSize, filteredVideos.length);
    var pageVideos = filteredVideos.slice(startIndex, endIndex);

    var allSelected = pageVideos.length > 0 && pageVideos.every(function(video) {
      return this._selectedVideos[video.id] === true;
    }, this);

    selectAllCheckbox.checked = allSelected;
  },

  // 更新分页信息
  updatePagination: function(totalPages) {
    var currentPageEl = document.getElementById('current-page');
    var totalPagesEl = document.getElementById('total-pages');
    var prevBtn = document.getElementById('prev-page-btn');
    var nextBtn = document.getElementById('next-page-btn');

    if (currentPageEl) currentPageEl.textContent = this._currentPage;
    if (totalPagesEl) totalPagesEl.textContent = totalPages;

    if (prevBtn) {
      prevBtn.disabled = this._currentPage <= 1;
      prevBtn.style.opacity = this._currentPage <= 1 ? '0.5' : '1';
      prevBtn.style.cursor = this._currentPage <= 1 ? 'not-allowed' : 'pointer';
    }

    if (nextBtn) {
      nextBtn.disabled = this._currentPage >= totalPages;
      nextBtn.style.opacity = this._currentPage >= totalPages ? '0.5' : '1';
      nextBtn.style.cursor = this._currentPage >= totalPages ? 'not-allowed' : 'pointer';
    }
  },

  // 上一页
  goToPrevPage: function() {
    if (this._currentPage > 1) {
      this._currentPage--;
      this.renderVideoList();
    }
  },

  // 下一页
  goToNextPage: function() {
    var filteredVideos = this.filterLivePictureVideos(this.videos);
    var totalPages = Math.ceil(filteredVideos.length / this._pageSize);
    if (this._currentPage < totalPages) {
      this._currentPage++;
      this.renderVideoList();
    }
  }
};

// ==================== 事件监听 ====================

// 监听用户视频列表加载
WXE.onUserFeedsLoaded(function(feeds) {
  console.log('[Profile] onUserFeedsLoaded 事件触发，feeds:', feeds);
  
  if (!feeds || !Array.isArray(feeds)) {
    console.warn('[Profile] feeds 不是数组或为空');
    return;
  }

  // 检查是否是Profile页面
  var isProfilePage = window.location.pathname.includes('/pages/profile');
  console.log('[Profile] 是否是Profile页面:', isProfilePage, '当前路径:', window.location.pathname);
  if (!isProfilePage) return;

  console.log('[Profile] 开始处理', feeds.length, '个视频');
  
  var processedCount = 0;
  feeds.forEach(function(item) {
    if (!item || !item.objectDesc) {
      console.warn('[Profile] 跳过无效项:', item);
      return;
    }

    var media = item.objectDesc.media && item.objectDesc.media[0];
    if (!media) {
      console.warn('[Profile] 跳过无media的项:', item);
      return;
    }

    // 使用 WXU.format_feed 格式化数据
    var profile = WXU.format_feed(item);
    if (!profile) {
      console.warn('[Profile] format_feed 返回 null:', item);
      return;
    }

    // 传递给 collector
    window.__wx_channels_profile_collector.addVideoFromAPI(profile);
    processedCount++;
  });
  
  console.log('[Profile] 成功处理', processedCount, '个视频');
});

// 监听直播回放列表加载
WXE.onUserLiveReplayLoaded(function(feeds) {
  if (!feeds || !Array.isArray(feeds)) return;

  // 检查是否是Profile页面
  var isProfilePage = window.location.pathname.includes('/pages/profile');
  if (!isProfilePage) return;

  __wx_log({ msg: '📺 [Profile] 获取到直播回放列表，数量: ' + feeds.length });

  feeds.forEach(function(item) {
    if (!item || !item.objectDesc) return;

    var media = item.objectDesc.media && item.objectDesc.media[0];
    var liveInfo = item.liveInfo || {};

    // 获取时长
    var duration = 0;
    if (media && media.spec && media.spec.length > 0 && media.spec[0].durationMs) {
      duration = media.spec[0].durationMs;
    } else if (liveInfo.duration) {
      duration = liveInfo.duration;
    }

    // 构建直播回放数据
    var profile = {
      type: "live_replay",
      id: item.id,
      nonce_id: item.objectNonceId,
      title: window.__wx_channels_profile_collector.cleanHtmlTags(item.objectDesc.description || ''),
      coverUrl: media ? (media.thumbUrl || media.coverUrl || '') : '',
      thumbUrl: media ? (media.thumbUrl || '') : '',
      url: media ? (media.url + (media.urlToken || '')) : '',
      size: media ? (media.fileSize || 0) : 0,
      key: media ? (media.decodeKey || '') : '',
      duration: duration,
      spec: media ? media.spec : [],
      nickname: item.contact ? item.contact.nickname : '',
      contact: item.contact || {},
      createtime: item.createtime || 0,
      liveInfo: liveInfo
    };

    // 传递给 collector
    window.__wx_channels_profile_collector.addVideoFromAPI(profile);
  });

  __wx_log({ msg: '✅ [Profile] 直播回放列表采集完成，共 ' + feeds.length + ' 个' });
});

// ==================== 初始化 ====================

// 检查是否是Profile页面
function is_profile_page() {
  return window.location.pathname.includes('/pages/profile');
}

// 页面加载后初始化
if (is_profile_page()) {
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function() {
      window.__wx_channels_profile_collector.init();
    });
  } else {
    window.__wx_channels_profile_collector.init();
  }
}
