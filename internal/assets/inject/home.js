/**
 * @file Home页面功能模块 - 下载按钮注入和幻灯片监听
 */
console.log('[home.js] 加载Home页面模块');

// ==================== 全局变量 ====================
var __last_slide_index__ = -1;
var __home_slide_observer__ = null;
var __home_first_load__ = true;
var __current_tab__ = 'unknown';
var __current_tab_type__ = 'unknown'; // video-player, video-list, live-list
var __category_feeds_cache__ = {}; // 缓存各分类的完整视频数据

// ==================== 分类视频列表弹窗 ====================
// 使用通用批量下载组件
function __show_category_video_list__() {
  var currentTabName = __get_tab_display_name(__current_tab__);
  var feeds = __category_feeds_cache__[currentTabName];

  if (!feeds || feeds.length === 0) {
    __wx_log({ msg: '❌ 当前分类暂无视频数据' });
    return;
  }

  // 调用通用批量下载UI
  __show_batch_download_ui__(feeds, currentTabName + ' - 视频列表');
}

// ==================== Tab检测 ====================
function __detect_current_tab() {
  // 查找所有 role="tab" 的元素
  var tabs = document.querySelectorAll('[role="tab"]');

  for (var i = 0; i < tabs.length; i++) {
    var tab = tabs[i];
    var isSelected = tab.getAttribute('aria-selected') === 'true';

    if (isSelected) {
      var text = tab.textContent.trim();
      console.log('[home.js] 找到选中的tab:', text);

      if (text === '首页') return 'home';
      if (text === '推荐') return 'recommend';
      if (text === '关注') return 'follow';
      if (text === '朋友') return 'friend';
      if (text === '直播') return 'live';

      // 其他分类tab
      return 'category_' + text;
    }
  }

  console.log('[home.js] 无法检测当前tab');
  return 'unknown';
}

function __get_tab_type(tab) {
  // 推荐、关注、朋友 = 视频播放页（可上下滑动切换）
  if (tab === 'recommend' || tab === 'follow' || tab === 'friend') {
    return 'video-player';
  }
  // 直播 = 直播列表
  if (tab === 'live') {
    return 'live-list';
  }
  // 首页和其他分类 = 视频列表
  return 'video-list';
}

function __get_tab_display_name(tab) {
  if (tab.startsWith('category_')) {
    return tab.replace('category_', '');
  }

  var tabNames = {
    'home': '首页',
    'recommend': '推荐',
    'follow': '关注',
    'friend': '朋友',
    'live': '直播',
    'unknown': '未知'
  };
  return tabNames[tab] || tab;
}

function __update_tab_display() {
  var newTab = __detect_current_tab();
  var newTabType = __get_tab_type(newTab);

  if (newTab !== __current_tab__) {
    __current_tab__ = newTab;
    __current_tab_type__ = newTabType;

    var displayName = __get_tab_display_name(newTab);
    var typeDesc = newTabType === 'video-player' ? '视频播放' :
      newTabType === 'live-list' ? '直播列表' : '视频列表';

    console.log('[home.js] 当前tab切换为:', displayName, '类型:', typeDesc);

    // 根据tab类型更新下载按钮状态
    __update_download_button_state();
  }
}

function __try_collect_page_data() {
  // 分类视频列表的数据通过API拦截获取，不需要从DOM采集
  // 数据会通过 CategoryFeedsLoaded 事件传递
}

function __update_download_button_state() {
  var downloadBtn = document.getElementById('wx-home-download-icon');
  if (!downloadBtn) return;

  // 视频播放页和视频列表页都启用下载按钮
  if (__current_tab_type__ === 'video-player' || __current_tab_type__ === 'video-list') {
    downloadBtn.style.opacity = '1';
    downloadBtn.style.cursor = 'pointer';
    downloadBtn.style.pointerEvents = 'auto';

    if (__current_tab_type__ === 'video-player') {
      downloadBtn.title = '下载视频';
    } else {
      downloadBtn.title = '批量下载视频列表';
    }
  } else {
    downloadBtn.style.opacity = '0.3';
    downloadBtn.style.cursor = 'not-allowed';
    downloadBtn.style.pointerEvents = 'none';

    if (__current_tab_type__ === 'live-list') {
      downloadBtn.title = '直播列表页暂不支持下载';
    } else {
      downloadBtn.title = '当前页面不支持下载';
    }
  }
}

// ==================== 下载按钮注入 ====================
async function __insert_download_btn_to_home_page() {
  console.log('[home.js] 开始注入下载按钮到顶部工具栏...');

  // 查找顶部工具栏容器
  var findToolbarContainer = function () {
    // 尝试多种选择器
    var container = document.querySelector('div[data-v-bf57a568].flex.items-center');
    if (container) return container;

    var parent = document.querySelector('div.flex-initial.flex-shrink-0.pl-6');
    if (parent) {
      container = parent.querySelector('.flex.items-center');
      if (container) return container;
    }

    // 尝试查找包含相机图标的容器
    var cameraIcon = document.querySelector('svg[data-v-bf57a568]');
    if (cameraIcon) {
      var current = cameraIcon;
      while (current && current.parentElement) {
        current = current.parentElement;
        if (current.classList && current.classList.contains('flex') && current.classList.contains('items-center')) {
          return current;
        }
      }
    }

    return null;
  };

  var tryInject = function () {
    var container = findToolbarContainer();
    if (!container) return false;

    // 检查是否已存在
    if (container.querySelector('#wx-home-download-icon')) {
      console.log('[home.js] 工具栏下载按钮已存在');
      return true;
    }

    // 创建下载图标
    var downloadIconWrapper = document.createElement('div');
    downloadIconWrapper.id = 'wx-home-download-icon';
    downloadIconWrapper.className = 'mr-4 h-6 w-6 flex-initial flex-shrink-0 text-fg-0 cursor-pointer';
    downloadIconWrapper.title = '下载视频';
    downloadIconWrapper.innerHTML = '<svg class="h-full w-full" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none"><path fill-rule="evenodd" clip-rule="evenodd" d="M12 3C12.3314 3 12.6 3.26863 12.6 3.6V13.1515L15.5757 10.1757C15.8101 9.94142 16.1899 9.94142 16.4243 10.1757C16.6586 10.4101 16.6586 10.7899 16.4243 11.0243L12.4243 15.0243C12.1899 15.2586 11.8101 15.2586 11.5757 15.0243L7.57574 11.0243C7.34142 10.7899 7.34142 10.4101 7.57574 10.1757C7.81005 9.94142 8.18995 9.94142 8.42426 10.1757L11.4 13.1515V3.6C11.4 3.26863 11.6686 3 12 3ZM3.6 14.4C3.93137 14.4 4.2 14.6686 4.2 15V19.2C4.2 19.5314 4.46863 19.8 4.8 19.8H19.2C19.5314 19.8 19.8 19.5314 19.8 19.2V15C19.8 14.6686 20.0686 14.4 20.4 14.4C20.7314 14.4 21 14.6686 21 15V19.2C21 20.1941 20.1941 21 19.2 21H4.8C3.80589 21 3 20.1941 3 19.2V15C3 14.6686 3.26863 14.4 3.6 14.4Z" fill="currentColor"></path></svg>';

    downloadIconWrapper.onclick = function () {
      // 检查当前tab类型
      if (__current_tab_type__ === 'video-player') {
        // 视频播放页：显示单个视频的下载选项
        var checkCount = 0;
        var maxChecks = 30;

        var checkData = function () {
          if (window.__wx_channels_store__ && window.__wx_channels_store__.profile) {
            __show_home_download_options(window.__wx_channels_store__.profile);
          } else {
            checkCount++;
            if (checkCount < maxChecks) {
              setTimeout(checkData, 100);
              if (checkCount === 1) {
                __wx_log({ msg: '⏳ 正在获取视频数据，请稍候...' });
              }
            } else {
              __wx_log({ msg: '❌ 获取视频数据超时\n请重新滑动视频或刷新页面' });
            }
          }
        };

        checkData();
      } else if (__current_tab_type__ === 'video-list') {
        // 视频列表页：显示批量下载弹窗
        __show_category_video_list__();
      } else {
        // 其他页面
        var tabName = __get_tab_display_name(__current_tab__);
        var message = '当前在 "' + tabName + '" 页面';
        if (__current_tab_type__ === 'live-list') {
          message += '，这是直播列表页，暂不支持下载';
        } else {
          message += '，暂不支持下载';
        }
        __wx_log({ msg: '⚠️ ' + message });
      }
    };

    // 插入到容器最前面
    container.insertBefore(downloadIconWrapper, container.firstChild);

    console.log('[home.js] ✅ 工具栏下载按钮注入成功');
    __wx_log({ msg: "注入下载按钮成功!" });

    // 检测并显示当前tab
    setTimeout(function () {
      __update_tab_display();
    }, 500);

    return true;
  };

  // 立即尝试注入
  if (tryInject()) return true;

  // 如果失败，使用 MutationObserver 监听 DOM 变化
  return new Promise(function (resolve) {
    var observer = new MutationObserver(function (_mutations, obs) {
      if (tryInject()) {
        obs.disconnect();
        resolve(true);
      }
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true
    });

    // 5秒后超时
    setTimeout(function () {
      observer.disconnect();
      console.log('[home.js] 工具栏按钮注入超时');
      resolve(false);
    }, 5000);
  });
}

// ==================== 幻灯片切换监听 ====================
// Home页面改为顶部工具栏按钮后，不再需要监听幻灯片切换来重新注入按钮
// 保留此函数以防需要监听其他事件
function __start_home_slide_monitor() {
  console.log("[home.js] Home页面使用顶部工具栏按钮，无需监听幻灯片切换");
}

// ==================== Tab切换监听 ====================
function __start_tab_monitor() {
  console.log('[home.js] 启动tab切换监听器');

  // 初始检测
  setTimeout(function () {
    __update_tab_display();
  }, 1000);

  // 监听点击事件 - 只监听tab元素的点击
  document.addEventListener('click', function (e) {
    var target = e.target;

    // 检查是否点击了tab元素
    var isTabClick = false;
    var current = target;
    for (var i = 0; i < 5; i++) {
      if (!current) break;
      if (current.getAttribute && current.getAttribute('role') === 'tab') {
        isTabClick = true;
        break;
      }
      current = current.parentElement;
    }

    // 只有点击tab时才检测
    if (isTabClick) {
      setTimeout(function () {
        __update_tab_display();
      }, 500);
    }
  });

  console.log('[home.js] ✅ Tab监听器已启动');
}

// ==================== 下载选项菜单 ====================
function __show_home_download_options(profile) {
  console.log('[home.js] 显示下载选项菜单', profile);

  // 移除已存在的菜单
  var existingMenu = document.getElementById('wx-download-menu');
  if (existingMenu) existingMenu.remove();
  var existingOverlay = document.getElementById('wx-download-overlay');
  if (existingOverlay) existingOverlay.remove();

  var menu = document.createElement('div');
  menu.id = 'wx-download-menu';
  menu.style.cssText = 'position:fixed;top:60px;right:20px;z-index:99999;background:#2b2b2b;color:#e5e5e5;border-radius:8px;padding:0;width:280px;box-shadow:0 8px 24px rgba(0,0,0,0.5);font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif;font-size:14px;';

  var title = profile.title || '未知视频';
  var shortTitle = title.length > 30 ? title.substring(0, 30) + '...' : title;

  var html = '';

  // 标题栏
  html += '<div style="padding:16px 20px;border-bottom:1px solid rgba(255,255,255,0.08);">';
  html += '<div style="font-size:15px;font-weight:500;color:#fff;margin-bottom:8px;">下载选项</div>';
  html += '<div style="font-size:13px;color:#999;line-height:1.4;">' + shortTitle + '</div>';
  html += '</div>';

  // 选项区域
  html += '<div style="padding:16px 20px;">';

  // 视频下载选项
  if (profile.spec && profile.spec.length > 0) {
    html += '<div style="margin-bottom:12px;font-size:12px;color:#999;">选择画质:</div>';
    profile.spec.forEach(function (spec, index) {
      var label = spec.fileFormat || ('画质' + (index + 1));
      if (spec.width && spec.height) {
        label += ' (' + spec.width + 'x' + spec.height + ')';
      }
      html += '<div class="download-option" data-index="' + index + '" style="padding:10px 16px;margin:8px 0;background:rgba(255,255,255,0.08);border-radius:6px;cursor:pointer;text-align:center;transition:background 0.2s;font-size:13px;">' + label + '</div>';
    });
  } else {
    html += '<div class="download-option" data-index="-1" style="padding:10px 16px;margin:8px 0;background:rgba(255,255,255,0.08);border-radius:6px;cursor:pointer;text-align:center;font-size:13px;">下载视频</div>';
  }

  // 封面下载
  html += '<div class="download-cover" style="padding:10px 16px;margin:8px 0;background:rgba(7,193,96,0.15);color:#07c160;border-radius:6px;cursor:pointer;text-align:center;font-size:13px;font-weight:500;">下载封面</div>';

  html += '</div>';

  // 底部按钮
  html += '<div style="padding:12px 20px;border-top:1px solid rgba(255,255,255,0.08);">';
  html += '<div class="close-menu" style="padding:8px;text-align:center;cursor:pointer;color:#999;font-size:13px;">取消</div>';
  html += '</div>';

  menu.innerHTML = html;
  document.body.appendChild(menu);

  // 添加遮罩
  var overlay = document.createElement('div');
  overlay.id = 'wx-download-overlay';
  overlay.style.cssText = 'position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.5);z-index:99998;';
  document.body.appendChild(overlay);

  function closeMenu() {
    menu.remove();
    overlay.remove();
  }

  // 绑定事件
  menu.querySelectorAll('.download-option').forEach(function (el) {
    el.onmouseover = function () { this.style.background = 'rgba(255,255,255,0.15)'; };
    el.onmouseout = function () { this.style.background = 'rgba(255,255,255,0.08)'; };
    el.onclick = function () {
      var index = parseInt(this.getAttribute('data-index'));
      var spec = index >= 0 && profile.spec ? profile.spec[index] : null;
      closeMenu();
      __wx_channels_handle_click_download__(spec);
    };
  });

  var coverBtn = menu.querySelector('.download-cover');
  coverBtn.onmouseover = function () { this.style.background = 'rgba(7,193,96,0.25)'; };
  coverBtn.onmouseout = function () { this.style.background = 'rgba(7,193,96,0.15)'; };
  coverBtn.onclick = function () {
    closeMenu();
    __wx_channels_handle_download_cover();
  };

  menu.querySelector('.close-menu').onclick = closeMenu;
  overlay.onclick = closeMenu;
}

// ==================== 统一按钮插入入口 ====================
async function insert_download_btn() {
  __wx_log({ msg: "等待注入下载按钮" });

  var pathname = window.location.pathname;
  console.log('[home.js] 当前页面路径:', pathname);

  // 搜索页面由 search.js 处理
  if (pathname.includes('/pages/s')) {
    console.log('[home.js] 搜索页面由 search.js 处理');
    return;
  }

  // Feed页面（视频详情页）
  if (pathname.includes('/pages/feed')) {
    console.log('[home.js] 检测到Feed页面');
    if (typeof __insert_download_btn_to_feed_page === 'function') {
      var success = await __insert_download_btn_to_feed_page();
      if (success) return;
    } else {
      console.error('[home.js] __insert_download_btn_to_feed_page 函数未定义');
    }
  }

  // Home页面
  if (pathname.includes('/pages/home')) {
    console.log('[home.js] 检测到Home页面');
    var success = await __insert_download_btn_to_home_page();
    if (success) {
      setTimeout(function () {
        __start_home_slide_monitor();
        // 启动tab监听
        __start_tab_monitor();
      }, 500);
      return;
    }
  }

  // 其他页面尝试通用注入
  __wx_log({ msg: "没有找到操作栏，注入下载按钮失败" });
}

console.log('[home.js] Home页面模块加载完成');

// ==================== 事件监听 ====================

// 监听首页推荐视频列表加载
WXE.onPCFlowLoaded(function (data) {
  // 兼容旧格式 (直接返回数组) 和新格式 ({feeds: [], params: {}})
  var feeds = Array.isArray(data) ? data : (data.feeds || []);
  var params = (data && !Array.isArray(data)) ? (data.params || {}) : {};

  // console.log('[home.js] onPCFlowLoaded 事件触发，feeds数量:', feeds ? feeds.length : 0);
  // console.log('[home.js] onPCFlowLoaded 参数:', JSON.stringify(params));

  // 过滤非首页数据
  // [新增] 排除 displayTabType: 3 (通常是相关推荐/非首页流)
  var isHomeData = false;
  if ((params.scene == 1 || params.scene == 2) || (!params.scene && params.displayTabType != 3)) {
    isHomeData = true;
  } else {
    // console.warn('[home.js] 忽略非首页数据 (scene:', params.scene, 'displayTabType:', params.displayTabType, ')');
  }

  if (isHomeData && feeds && feeds.length > 0) {
    // 同时也作为 "首页" 分类的缓存
    var tagName = "首页";
    if (!__category_feeds_cache__[tagName]) {
      __category_feeds_cache__[tagName] = [];
      console.log('[home.js] 初始化首页缓存');
    }

    // 追加新视频（去重 + 严格过滤 cgi_id=6638）
    var existingIds = {};
    __category_feeds_cache__[tagName].forEach(function (f) {
      existingIds[f.id] = true;
    });

    var newCount = 0;
    var ignoredCount = 0;

    feeds.forEach(function (feed) {
      if (feed.id && !existingIds[feed.id]) {
        __category_feeds_cache__[tagName].push(feed);
        existingIds[feed.id] = true;
        newCount++;
      }
    });

    var totalCount = __category_feeds_cache__[tagName].length;
    console.log('[home.js] "首页" (PCFlow) 新增', newCount, '个视频 (忽略 ' + ignoredCount + ' 个非首页数据)，总计:', totalCount);

    // 如果当前选中的是首页，显示提示
    var currentTabName = __get_tab_display_name(__current_tab__);
    if (currentTabName === '首页') {
      if (ignoredCount > 0) {
        __wx_log({ msg: '✅ "首页" 加载 ' + totalCount + ' 个视频 (已过滤 ' + ignoredCount + ' 个杂项)' });
      } else {
        __wx_log({ msg: '✅ "首页" 已加载 ' + totalCount + ' 个视频' });
      }
    }

    // 设置第一个视频为当前视频（兼容旧逻辑）
    // 注意：如果全部被过滤了，feeds[0] 可能是不合法的，但 set_feed 应该能处理
    if (feeds.length > 0) {
      WXU.set_feed(feeds[0]);
    }
  }
});

// 监听切换到下一个视频
WXE.onGotoNextFeed(function (feed) {
  console.log('[home.js] onGotoNextFeed 事件触发');
  WXU.set_cur_video();
  WXU.set_feed(feed);
});

// 监听切换到上一个视频
WXE.onGotoPrevFeed(function (feed) {
  console.log('[home.js] onGotoPrevFeed 事件触发');
  WXU.set_cur_video();
  WXU.set_feed(feed);
});

// 监听视频详情加载
WXE.onFetchFeedProfile(function (feed) {
  console.log('[home.js] onFetchFeedProfile 事件触发');
  WXU.set_cur_video();
  WXU.set_feed(feed);
});

// 监听 Feed 事件（统一处理）
WXE.onFeed(function (feed) {
  console.log('[home.js] onFeed 事件触发');
  WXU.set_feed(feed);
});

// 新增：监听搜索结果加载（如果有的话）
if (WXE.onSearchResultLoaded) {
  WXE.onSearchResultLoaded(function (data) {
    console.log('[home.js] onSearchResultLoaded 事件触发');
    console.log('[home.js] 搜索结果数据:', data);
  });
}

// 新增：监听分类视频列表加载（首页、美食、生活等分类tab）
if (WXE.onCategoryFeedsLoaded) {
  WXE.onCategoryFeedsLoaded(function (data) {
    // data 包含 {feeds: [], params: {}}
    var feeds = data.feeds || data; // 兼容旧格式
    var params = data.params || {};

    console.log('[home.js] CategoryFeedsLoaded 触发, 参数:', JSON.stringify(params));
    console.log('[home.js] 提取到视频数:', feeds ? feeds.length : 0);

    // 提取分类名称
    var apiTagName = '';

    // 情况1: 显式指定了分类名称 (如：美食、旅行)
    if (params.tagItem && params.tagItem.topTag && params.tagItem.topTag.tagName) {
      apiTagName = params.tagItem.topTag.tagName;
    }
    // 情况2: 首页场景
    // 通常 scene 为 1，且没有 tagItem
    else if (params.scene == 1 || !params.scene) {
      apiTagName = '首页';
      console.log('[home.js] 未检测到tagName，判定为 "首页" 数据 (scene:', params.scene, ')');
    }

    // 初始化该分类的缓存
    if (!__category_feeds_cache__[apiTagName]) {
      __category_feeds_cache__[apiTagName] = [];
      console.log('[home.js] 初始化分类缓存:', apiTagName);
    }

    // 追加新视频（去重）
    var existingIds = {};
    __category_feeds_cache__[apiTagName].forEach(function (f) {
      existingIds[f.id] = true;
    });

    var newCount = 0;
    var ignoredCount = 0;

    feeds.forEach(function (feed) {
      if (feed.id && !existingIds[feed.id]) {
        __category_feeds_cache__[apiTagName].push(feed);
        existingIds[feed.id] = true;
        newCount++;
      }
    });

    var totalCount = __category_feeds_cache__[apiTagName].length;

    console.log('[home.js] "' + apiTagName + '" 新增', newCount, '个视频 (忽略 ' + ignoredCount + ' 个)，总计:', totalCount);

    // 如果弹窗已打开且是当前分类，实时更新UI（使用通用组件）
    var currentTabName = __get_tab_display_name(__current_tab__);
    if (window.__wx_batch_download_manager__ &&
      window.__wx_batch_download_manager__.isVisible &&
      apiTagName === currentTabName) {
      __update_batch_download_ui__(feeds, apiTagName + ' - 视频列表');
    }

    // 始终显示提示（特别是首页）
    if (apiTagName === currentTabName) {
      __wx_log({ msg: '✅ "' + currentTabName + '" 已加载 ' + totalCount + ' 个视频' });
    }
  });

  // 监听tab切换，显示缓存的视频数量
  var __original_update_tab_display__ = __update_tab_display;
  __update_tab_display = function () {
    var oldTab = __current_tab__;
    __original_update_tab_display__();

    // 如果tab发生了变化
    if (oldTab !== __current_tab__) {
      var newTabName = __get_tab_display_name(__current_tab__);
      console.log('[home.js] Tab 已切换:', oldTab, '->', __current_tab__, '(', newTabName, ')');

      // 不再在切换时立即清除缓存，允许用户切换回来查看
      // if (oldTabName && __category_feeds_cache__[oldTabName]) {
      //   console.log('[home.js] 清空旧tab缓存:', oldTabName);
      //   delete __category_feeds_cache__[oldTabName];
      // }

      // 如果是视频列表类型，显示提示
      if (__current_tab_type__ === 'video-list' || __current_tab__ === 'home') {
        var currentTabName = __get_tab_display_name(__current_tab__);
        var cachedFeeds = __category_feeds_cache__[currentTabName];

        if (cachedFeeds !== undefined && cachedFeeds.length > 0) {
          __wx_log({ msg: '📍 "' + currentTabName + '" - 已加载 ' + cachedFeeds.length + ' 个视频' });
        } else {
          __wx_log({ msg: '📍 "' + currentTabName + '" - 等待加载数据...' });
        }
      }
    }
  };
}
