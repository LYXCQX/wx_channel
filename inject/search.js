/**
 * @file 搜索页面功能模块 - 事件监听和数据采集
 */
console.log('[search.js] 加载搜索页面模块');

// ==================== 搜索页面数据采集器 ====================
window.__wx_channels_search_collector = {
  feeds: [], // 动态（视频）
  _selectedItems: {}, // 选中的项目 {id: true}
  _currentPage: 1,
  _pageSize: 50,
  _maxItems: 200,
  _processing: false, // 防止并发处理
  _lastProcessTime: 0, // 上次处理时间
  _processDelay: 100, // 处理延迟（毫秒）

  // 初始化
  init: function() {
    var self = this;
    setTimeout(function() {
      self.injectToolbarIcon();
    }, 2000);
  },

  // 从API添加搜索结果
  addSearchResult: function(data) {
    if (!data) return;

    // 防抖：如果正在处理或距离上次处理时间太短，则延迟处理
    var now = Date.now();
    if (this._processing || (now - this._lastProcessTime) < this._processDelay) {
      return;
    }

    this._processing = true;
    this._lastProcessTime = now;

    var startTime = Date.now();
    var initialCount = this.feeds.length;

    // 处理动态（视频）- objectList
    if (data.feeds && Array.isArray(data.feeds)) {
      data.feeds.forEach(function(feed) {
        var feedId = feed.id;
        if (feed && feedId && !this.feeds.find(function(f) { return f.id === feedId; })) {
          if (this.feeds.length < this._maxItems) {
            // 使用 WXU.format_feed 格式化数据（与其他页面统一）
            var formatted = WXU.format_feed(feed);
            
            // 添加视频和直播数据（保留直播数据显示，但暂时不能下载）
            if (formatted && (formatted.type === 'media' || formatted.type === 'live')) {
              this.feeds.push(formatted);
              // 只有视频类型才默认选中
              if (formatted.type === 'media') {
                this._selectedItems[feedId] = true;
              }
            }
          }
        }
      }, this);
    }

    var newCount = this.feeds.length;
    var addedCount = newCount - initialCount;
    
    // 只在有新数据时才更新UI和打印日志
    if (addedCount > 0) {
      // 如果通用批量下载UI已打开，更新它（包含所有数据：视频和直播）
      if (window.__wx_batch_download_manager__ && window.__wx_batch_download_manager__.isVisible) {
        __update_batch_download_ui__(this.feeds, '搜索结果');
      }
      
      var elapsed = Date.now() - startTime;
      
      // 统计视频和直播数量
      var videoCount = this.feeds.filter(function(f) { return f.type === 'media'; }).length;
      var liveCount = this.feeds.filter(function(f) { return f.type === 'live'; }).length;
      
      console.log('[搜索] 新增 ' + addedCount + ' 条数据，总计: ' + videoCount + ' 个视频' + (liveCount > 0 ? ', ' + liveCount + ' 个直播' : '') + ' (耗时: ' + elapsed + 'ms)');
      
      // 只在整十数时打印到后台日志
      if (newCount % 50 === 0) {
        var msg = '📊 [搜索] 已采集 ' + videoCount + ' 个视频';
        if (liveCount > 0) msg += ', ' + liveCount + ' 个直播';
        __wx_log({ msg: msg });
      }
    }

    this._processing = false;
  },

  // 在工具栏注入图标
  injectToolbarIcon: function() {
    var self = this;
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
      if (container.querySelector('#wx-search-download-icon')) return true;

      var iconWrapper = document.createElement('div');
      iconWrapper.id = 'wx-search-download-icon';
      iconWrapper.className = 'mr-4 h-6 w-6 flex-initial flex-shrink-0 text-fg-0 cursor-pointer';
      iconWrapper.title = '搜索结果采集';
      iconWrapper.innerHTML = '<svg class="h-full w-full" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none"><path fill-rule="evenodd" clip-rule="evenodd" d="M12 3C12.3314 3 12.6 3.26863 12.6 3.6V13.1515L15.5757 10.1757C15.8101 9.94142 16.1899 9.94142 16.4243 10.1757C16.6586 10.4101 16.6586 10.7899 16.4243 11.0243L12.4243 15.0243C12.1899 15.2586 11.8101 15.2586 11.5757 15.0243L7.57574 11.0243C7.34142 10.7899 7.34142 10.4101 7.57574 10.1757C7.81005 9.94142 8.18995 9.94142 8.42426 10.1757L11.4 13.1515V3.6C11.4 3.26863 11.6686 3 12 3ZM3.6 14.4C3.93137 14.4 4.2 14.6686 4.2 15V19.2C4.2 19.5314 4.46863 19.8 4.8 19.8H19.2C19.5314 19.8 19.8 19.5314 19.8 19.2V15C19.8 14.6686 20.0686 14.4 20.4 14.4C20.7314 14.4 21 14.6686 21 15V19.2C21 20.1941 20.1941 21 19.2 21H4.8C3.80589 21 3 20.1941 3 19.2V15C3 14.6686 3.26863 14.4 3.6 14.4Z" fill="currentColor"></path></svg>';

      iconWrapper.onclick = function() {
        // 使用通用批量下载组件
        if (window.__wx_batch_download_manager__ && window.__wx_batch_download_manager__.isVisible) {
          __close_batch_download_ui__();
        } else {
          // 显示批量下载UI（包含所有数据：视频和直播）
          if (self.feeds.length === 0) {
            __wx_log({ msg: '⚠️ 暂无搜索结果' });
            return;
          }
          
          __show_batch_download_ui__(self.feeds, '搜索结果');
        }
      };

      container.insertBefore(iconWrapper, container.firstChild);
      console.log('[搜索] ✅ 下载图标已注入到工具栏');
      return true;
    };

    if (tryInject()) return;
    var observer = new MutationObserver(function(_mutations, obs) {
      if (tryInject()) { obs.disconnect(); }
    });
    observer.observe(document.body, { childList: true, subtree: true });
    setTimeout(function() { observer.disconnect(); }, 5000);
  },

  // 添加搜索UI
  addSearchUI: function() {
    var self = this;
    var existingUI = document.getElementById('wx-channels-search-ui');
    if (existingUI) existingUI.remove();

    var ui = document.createElement('div');
    ui.id = 'wx-channels-search-ui';
    ui.style.cssText = 'position:fixed;top:60px;right:20px;background:#2b2b2b;color:#e5e5e5;padding:0;border-radius:8px;z-index:99999;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif;font-size:14px;width:450px;max-height:80vh;box-shadow:0 8px 24px rgba(0,0,0,0.5);display:none;overflow:hidden;';

    ui.innerHTML = 
      // 标题栏
      '<div style="padding:16px 20px;border-bottom:1px solid rgba(255,255,255,0.08);display:flex;justify-content:space-between;align-items:center;">' +
        '<div style="font-size:15px;font-weight:500;color:#fff;">搜索结果 - 动态</div>' +
        '<div id="search-total-count" style="font-size:13px;color:#999;">0 个</div>' +
      '</div>' +
      
      // 列表区域
      '<div id="search-list-container" style="overflow-y:auto;padding:12px 20px;max-height:200px;">' +
        '<div id="search-list" style="display:flex;flex-direction:column;gap:8px;"></div>' +
      '</div>' +
      
      // 分页
      '<div id="search-pagination" style="padding:12px 20px;border-top:1px solid rgba(255,255,255,0.08);border-bottom:1px solid rgba(255,255,255,0.08);display:flex;justify-content:space-between;align-items:center;">' +
        '<div style="font-size:13px;color:#999;">第 <span id="search-current-page">1</span> / <span id="search-total-pages">1</span> 页</div>' +
        '<div style="display:flex;gap:8px;">' +
          '<button id="search-prev-page" style="background:rgba(255,255,255,0.08);color:#999;border:none;padding:4px 12px;border-radius:4px;cursor:pointer;font-size:13px;">上一页</button>' +
          '<button id="search-next-page" style="background:rgba(255,255,255,0.08);color:#999;border:none;padding:4px 12px;border-radius:4px;cursor:pointer;font-size:13px;">下一页</button>' +
        '</div>' +
      '</div>' +
      
      // 操作区
      '<div style="padding:16px 20px;">' +
        '<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:12px;">' +
          '<label style="display:flex;align-items:center;cursor:pointer;font-size:13px;color:#999;user-select:none;">' +
            '<input type="checkbox" id="search-select-all" style="margin-right:8px;cursor:pointer;" />' +
            '<span>全选当前页</span>' +
          '</label>' +
          '<span id="search-selected-count" style="font-size:13px;color:#07c160;">已选 0 个</span>' +
        '</div>' +
        '<div style="display:flex;gap:8px;">' +
          '<button id="search-download-btn" style="flex:1;background:#07c160;color:#fff;border:none;padding:8px 12px;border-radius:6px;cursor:pointer;font-size:14px;font-weight:500;">下载选中</button>' +
          '<button id="search-export-btn" style="flex:1;background:transparent;color:#999;border:1px solid rgba(255,255,255,0.12);padding:8px 12px;border-radius:6px;cursor:pointer;font-size:13px;">导出数据</button>' +
        '</div>' +
      '</div>';

    document.body.appendChild(ui);

    // 绑定事件
    setTimeout(function() {
      // 分页
      document.getElementById('search-prev-page').addEventListener('click', function() { self.goToPrevPage(); });
      document.getElementById('search-next-page').addEventListener('click', function() { self.goToNextPage(); });

      // 全选
      document.getElementById('search-select-all').addEventListener('change', function() { self.toggleSelectAll(this.checked); });

      // 下载和导出
      document.getElementById('search-download-btn').addEventListener('click', function() { self.downloadSelected(); });
      document.getElementById('search-export-btn').addEventListener('click', function() { self.exportData(); });
    }, 100);
  },

  // 更新UI统计
  updateSearchUI: function() {
    var totalCountEl = document.getElementById('search-total-count');
    if (totalCountEl) totalCountEl.textContent = this.feeds.length + ' 个';
  },

  // 渲染列表
  renderItemList: function() {
    var listContainer = document.getElementById('search-list');
    if (!listContainer) return;

    var items = this.getCurrentTabItems();
    var totalPages = Math.ceil(items.length / this._pageSize);
    var startIndex = (this._currentPage - 1) * this._pageSize;
    var endIndex = Math.min(startIndex + this._pageSize, items.length);
    var pageItems = items.slice(startIndex, endIndex);

    listContainer.innerHTML = '';

    var self = this;
    pageItems.forEach(function(item) {
      var isSelected = self._selectedItems[item.id] === true;
      var itemEl = self.createItemElement(item, isSelected);
      listContainer.appendChild(itemEl);
    });

    this.updatePagination(totalPages);
    this.updateSelectedCount();
  },

  // 获取当前标签页的数据
  getCurrentTabItems: function() {
    return this.feeds;
  },

  // 创建列表项元素
  createItemElement: function(item, isSelected) {
    var self = this;
    
    var el = document.createElement('div');
    el.style.cssText = 'display:flex;align-items:flex-start;padding:8px;background:rgba(255,255,255,0.05);border-radius:6px;transition:background 0.2s;gap:10px;cursor:pointer;';
    
    // 动态（视频）
    el.innerHTML = this.createFeedItemHTML(item, isSelected);

    el.onmouseenter = function() { this.style.background = 'rgba(255,255,255,0.08)'; };
    el.onmouseleave = function() { this.style.background = 'rgba(255,255,255,0.05)'; };

    el.onclick = function(e) {
      if (e.target.tagName !== 'INPUT' && e.target.tagName !== 'IMG') {
        var checkbox = this.querySelector('input[type="checkbox"]');
        if (checkbox) {
          checkbox.checked = !checkbox.checked;
          self.toggleItemSelection(item.id, checkbox.checked);
        }
      }
    };

    var checkbox = el.querySelector('input[type="checkbox"]');
    if (checkbox) {
      checkbox.onchange = function(e) {
        e.stopPropagation();
        self.toggleItemSelection(item.id, this.checked);
      };
    }

    return el;
  },

  // 创建动态项HTML
  createFeedItemHTML: function(item, isSelected) {
    var coverUrl = item.thumbUrl || item.coverUrl || '';
    
    // 格式化时长
    var duration = '';
    if (item.duration) {
      var seconds = Math.floor(item.duration / 1000);
      var minutes = Math.floor(seconds / 60);
      seconds = seconds % 60;
      duration = minutes + ':' + (seconds < 10 ? '0' : '') + seconds;
    }

    // 格式化文件大小
    var fileSize = '';
    if (item.size) {
      var mb = item.size / (1024 * 1024);
      fileSize = mb.toFixed(1) + ' MB';
    }

    // 格式化发布时间
    var publishTime = '';
    if (item.createtime) {
      var date = new Date(item.createtime * 1000);
      var month = date.getMonth() + 1;
      var day = date.getDate();
      publishTime = month + '月' + day + '日';
    }

    return '<input type="checkbox" ' + (isSelected ? 'checked' : '') + ' style="margin-top:4px;cursor:pointer;flex-shrink:0;" />' +
      '<div style="width:60px;height:40px;border-radius:4px;overflow:hidden;background:#1a1a1a;flex-shrink:0;position:relative;">' +
        (coverUrl ? '<img src="' + coverUrl + '" style="width:100%;height:100%;object-fit:cover;" />' : '<div style="width:100%;height:100%;display:flex;align-items:center;justify-content:center;color:#666;font-size:12px;">无封面</div>') +
        (duration ? '<div style="position:absolute;bottom:4px;right:4px;background:rgba(0,0,0,0.8);color:#fff;font-size:11px;padding:2px 4px;border-radius:2px;">' + duration + '</div>' : '') +
      '</div>' +
      '<div style="flex:1;min-width:0;display:flex;flex-direction:column;gap:4px;">' +
        '<div style="font-size:13px;color:#fff;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;line-height:1.4;">' + (item.title || '无标题') + '</div>' +
        '<div style="display:flex;gap:8px;font-size:11px;color:#999;flex-wrap:wrap;">' +
          (fileSize ? '<span>' + fileSize + '</span>' : '') +
          (publishTime ? '<span>' + publishTime + '</span>' : '') +
          (item.nickname ? '<span style="overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:100px;">@' + item.nickname + '</span>' : '') +
        '</div>' +
      '</div>';
  },

  // 创建动态项HTML
  createFeedItemHTML: function(item, isSelected) {
    var coverUrl = item.thumbUrl || item.coverUrl || '';
    
    // 格式化时长
    var duration = '';
    if (item.duration) {
      var seconds = Math.floor(item.duration / 1000);
      var minutes = Math.floor(seconds / 60);
      seconds = seconds % 60;
      duration = minutes + ':' + (seconds < 10 ? '0' : '') + seconds;
    }

    // 格式化文件大小
    var fileSize = '';
    if (item.size) {
      var mb = item.size / (1024 * 1024);
      fileSize = mb.toFixed(1) + ' MB';
    }

    // 格式化发布时间
    var publishTime = '';
    if (item.createtime) {
      var date = new Date(item.createtime * 1000);
      var month = date.getMonth() + 1;
      var day = date.getDate();
      publishTime = month + '月' + day + '日';
    }

    return '<input type="checkbox" ' + (isSelected ? 'checked' : '') + ' style="margin-top:4px;cursor:pointer;flex-shrink:0;" />' +
      '<div style="width:60px;height:40px;border-radius:4px;overflow:hidden;background:#1a1a1a;flex-shrink:0;position:relative;">' +
        (coverUrl ? '<img src="' + coverUrl + '" style="width:100%;height:100%;object-fit:cover;" />' : '<div style="width:100%;height:100%;display:flex;align-items:center;justify-content:center;color:#666;font-size:12px;">无封面</div>') +
        (duration ? '<div style="position:absolute;bottom:4px;right:4px;background:rgba(0,0,0,0.8);color:#fff;font-size:11px;padding:2px 4px;border-radius:2px;">' + duration + '</div>' : '') +
      '</div>' +
      '<div style="flex:1;min-width:0;display:flex;flex-direction:column;gap:4px;">' +
        '<div style="font-size:13px;color:#fff;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;line-height:1.4;">' + (item.title || '无标题') + '</div>' +
        '<div style="display:flex;gap:8px;font-size:11px;color:#999;flex-wrap:wrap;">' +
          (fileSize ? '<span>' + fileSize + '</span>' : '') +
          (publishTime ? '<span>' + publishTime + '</span>' : '') +
          (item.nickname ? '<span style="overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:100px;">@' + item.nickname + '</span>' : '') +
        '</div>' +
      '</div>';
  },

  // 切换选中状态
  toggleItemSelection: function(id, selected) {
    if (selected) {
      this._selectedItems[id] = true;
    } else {
      delete this._selectedItems[id];
    }
    this.updateSelectedCount();
  },

  // 全选/取消全选
  toggleSelectAll: function(selectAll) {
    var items = this.getCurrentTabItems();
    var startIndex = (this._currentPage - 1) * this._pageSize;
    var endIndex = Math.min(startIndex + this._pageSize, items.length);
    var pageItems = items.slice(startIndex, endIndex);

    var self = this;
    pageItems.forEach(function(item) {
      if (selectAll) {
        self._selectedItems[item.id] = true;
      } else {
        delete self._selectedItems[item.id];
      }
    });

    this.renderItemList();
  },

  // 更新选中数量
  updateSelectedCount: function() {
    var countEl = document.getElementById('search-selected-count');
    if (countEl) {
      var count = 0;
      for (var id in this._selectedItems) {
        if (this._selectedItems[id] === true) {
          count++;
        }
      }
      countEl.textContent = '已选 ' + count + ' 个';
    }
  },

  // 更新分页
  updatePagination: function(totalPages) {
    var currentPageEl = document.getElementById('search-current-page');
    var totalPagesEl = document.getElementById('search-total-pages');
    var prevBtn = document.getElementById('search-prev-page');
    var nextBtn = document.getElementById('search-next-page');

    if (currentPageEl) currentPageEl.textContent = this._currentPage;
    if (totalPagesEl) totalPagesEl.textContent = totalPages;

    if (prevBtn) {
      prevBtn.disabled = this._currentPage <= 1;
      prevBtn.style.opacity = this._currentPage <= 1 ? '0.5' : '1';
    }

    if (nextBtn) {
      nextBtn.disabled = this._currentPage >= totalPages;
      nextBtn.style.opacity = this._currentPage >= totalPages ? '0.5' : '1';
    }
  },

  goToPrevPage: function() {
    if (this._currentPage > 1) {
      this._currentPage--;
      this.renderItemList();
    }
  },

  goToNextPage: function() {
    var items = this.getCurrentTabItems();
    var totalPages = Math.ceil(items.length / this._pageSize);
    if (this._currentPage < totalPages) {
      this._currentPage++;
      this.renderItemList();
    }
  },

  // 下载选中的视频
  downloadSelected: function() {
    // 获取选中的动态（视频）
    var selectedFeeds = this.feeds.filter(function(f) {
      return this._selectedItems[f.id] === true && f.url;
    }, this);

    if (selectedFeeds.length === 0) {
      WXU.toast('没有选中可下载的内容');
      return;
    }

    __wx_log({ msg: '🚀 [搜索] 开始下载 ' + selectedFeeds.length + ' 个视频' });
    // TODO: 实现视频下载逻辑
    WXU.toast('开始下载 ' + selectedFeeds.length + ' 个视频');
  },

  // 导出数据
  exportData: function() {
    var data = {
      feeds: this.feeds,
      timestamp: Date.now()
    };

    var blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    var url = URL.createObjectURL(blob);
    var a = document.createElement('a');
    a.href = url;
    a.download = 'search_results_' + new Date().toISOString().slice(0, 10) + '.json';
    a.click();
    URL.revokeObjectURL(url);

    __wx_log({ msg: '📤 [搜索] 已导出搜索结果' });
  }
};

// ==================== 事件监听 ====================

// 确保事件监听器只注册一次
if (!window.__wx_search_event_registered) {
  window.__wx_search_event_registered = true;

  // 监听搜索结果加载
  WXE.onSearchResultLoaded(function(data) {
    // 检查是否是搜索页面
    var isSearchPage = window.location.pathname.includes('/pages/s');
    if (!isSearchPage) {
      return;
    }

    if (!data) {
      console.warn('[搜索] 数据为空');
      return;
    }

    console.log('[搜索] 收到搜索结果 - feeds:', data.feeds ? data.feeds.length : 0);
    
    window.__wx_channels_search_collector.addSearchResult(data);
  });
}

// ==================== 初始化 ====================

function is_search_page() {
  return window.location.pathname.includes('/pages/s');
}

if (is_search_page()) {
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function() {
      window.__wx_channels_search_collector.init();
    });
  } else {
    window.__wx_channels_search_collector.init();
  }
}

console.log('[search.js] 搜索页面模块加载完成');
