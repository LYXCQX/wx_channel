/**
 * @file Profile页面功能模块 - 事件监听和数据采集
 */

// ==================== Profile页面视频列表采集器 ====================
window.__wx_channels_profile_collector = {
  videos: [],
  isCollecting: false,
  _lastLogMessage: '',
  _lastTipVideoCount: 0,
  _lastTipLiveReplayCount: 0,
  _maxVideos: 300, // 最多采集300个视频

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
          // 显示批量下载UI（包含视频和直播回放，排除正在直播）
          var filteredVideos = self.filterLivePictureVideos(self.videos).filter(function(v) {
            return v && (v.type === 'media' || v.type === 'live_replay');
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

    // 限制最多300个视频
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
      console.log('[Profile] 新增视频:', (videoData.title || '').substring(0, 30));

      // 每10个视频发送一次日志
      var filteredVideos = this.filterLivePictureVideos(this.videos);
      var videoCount = filteredVideos.filter(function(v) { return v && v.type === 'media'; }).length;
      var liveReplayCount = filteredVideos.filter(function(v) { return v && v.type === 'live_replay'; }).length;

      if (videoCount > 0 && videoCount % 10 === 0 && videoCount !== this._lastTipVideoCount) {
        this._lastTipVideoCount = videoCount;
        var msg = '📊 [Profile] 已采集 ' + videoCount + ' 个视频';
        if (liveReplayCount > 0) msg += ', ' + liveReplayCount + ' 个直播回放';
        __wx_log({ msg: msg });
      }

      // 更新UI（使用通用批量下载组件，包含视频和直播回放）
      if (window.__wx_batch_download_manager__ && window.__wx_batch_download_manager__.isVisible) {
        var filteredVideos = this.filterLivePictureVideos(this.videos).filter(function(v) {
          return v && (v.type === 'media' || v.type === 'live_replay');
        });
        __update_batch_download_ui__(filteredVideos, 'Profile - 视频列表');
      }
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
