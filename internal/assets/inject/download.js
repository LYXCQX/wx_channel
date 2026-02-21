/**
 * @file 下载功能模块
 */
console.log('[download.js] 加载下载模块');

// ==================== 进度条显示 ====================
async function show_progress_or_loaded_size(response) {
  var content_length = response.headers.get("Content-Length");
  var chunks = [];
  var total_size = content_length ? parseInt(content_length, 10) : 0;

  var progressBarId = 'progress-' + Date.now();
  var progressBarHTML = '<div id="' + progressBarId + '" style="position: fixed; top: 20px; left: 50%; transform: translateX(-50%); z-index: 10000; background: rgba(0,0,0,0.7); border-radius: 8px; padding: 15px; box-shadow: 0 4px 12px rgba(0,0,0,0.15); color: white; font-size: 14px; min-width: 280px; text-align: center;">' +
    '<div style="margin-bottom: 12px; font-weight: bold;">视频下载中</div>' +
    '<div class="progress-container" style="background: rgba(255,255,255,0.2); height: 10px; border-radius: 5px; overflow: hidden; margin-bottom: 10px;">' +
    '<div class="progress-bar" style="height: 100%; width: 0%; background: #07c160; transition: width 0.3s;"></div></div>' +
    '<div class="progress-details" style="display: flex; justify-content: space-between; font-size: 12px; opacity: 0.8;">' +
    '<span class="progress-size">准备下载...</span><span class="progress-speed"></span></div></div>';

  var progressBarContainer = document.createElement('div');
  progressBarContainer.innerHTML = progressBarHTML;
  document.body.appendChild(progressBarContainer.firstElementChild);

  var progressBar = document.querySelector('#' + progressBarId + ' .progress-bar');
  var progressSize = document.querySelector('#' + progressBarId + ' .progress-size');
  var progressSpeed = document.querySelector('#' + progressBarId + ' .progress-speed');

  var loaded_size = 0;
  var reader = response.body.getReader();
  var lastUpdate = Date.now();
  var lastLoaded = 0;

  while (true) {
    var result = await reader.read();
    if (result.done) break;

    chunks.push(result.value);
    loaded_size += result.value.length;

    var currentTime = Date.now();
    if (currentTime - lastUpdate > 200) {
      var percent = total_size ? (loaded_size / total_size * 100) : 0;
      if (progressBar) progressBar.style.width = percent + '%';

      if (total_size) {
        progressSize.textContent = formatFileSize(loaded_size) + ' / ' + formatFileSize(total_size);
      } else {
        progressSize.textContent = '已下载: ' + formatFileSize(loaded_size);
      }

      var timeElapsed = (currentTime - lastUpdate) / 1000;
      if (timeElapsed > 0) {
        var currentSpeed = (loaded_size - lastLoaded) / timeElapsed;
        progressSpeed.textContent = formatFileSize(currentSpeed) + '/s';
      }

      lastLoaded = loaded_size;
      lastUpdate = currentTime;
    }
  }

  var progressElement = document.getElementById(progressBarId);
  if (progressElement) {
    setTimeout(function () {
      progressElement.style.opacity = '0';
      progressElement.style.transition = 'opacity 0.5s';
      setTimeout(function () { progressElement.remove(); }, 500);
    }, 1000);
  }

  __wx_log({ msg: '下载完成，文件总大小<' + formatFileSize(loaded_size) + '>' });

  return new Blob(chunks);
}

// ==================== 下载函数 ====================

/** 下载非加密视频 */
async function __wx_channels_download2(profile, filename) {
  console.log("__wx_channels_download2");
  await __wx_load_script("https://res.wx.qq.com/t/wx_fed/cdn_libs/res/FileSaver.min.js");
  var response = await fetch(profile.url);
  var blob = await show_progress_or_loaded_size(response);
  saveAs(blob, filename + ".mp4");
}

/** 下载图片 */
async function __wx_channels_download3(profile, filename) {
  console.log("__wx_channels_download3");
  await __wx_load_script("https://res.wx.qq.com/t/wx_fed/cdn_libs/res/FileSaver.min.js");
  await __wx_load_script("https://res.wx.qq.com/t/wx_fed/cdn_libs/res/jszip.min.js");

  var zip = new JSZip();
  zip.file("contact.txt", JSON.stringify(profile.contact, null, 2));
  var folder = zip.folder("images");

  var fetchPromises = profile.files.map(function (f, index) {
    return fetch(f.url).then(function (response) {
      return response.blob();
    }).then(function (blob) {
      folder.file((index + 1) + ".png", blob);
    });
  });

  try {
    await Promise.all(fetchPromises);
    var content = await zip.generateAsync({ type: "blob" });
    saveAs(content, filename + ".zip");
  } catch (err) {
    __wx_log({ msg: "下载失败\n" + err.message });
  }
}

/** 下载加密视频 */
async function __wx_channels_download4(profile, filename) {
  console.log("__wx_channels_download4");
  await __wx_load_script("https://res.wx.qq.com/t/wx_fed/cdn_libs/res/FileSaver.min.js");

  if (profile.key && !profile.decryptor_array) {
    console.log('🔑 检测到加密key，正在生成解密数组...');
    profile.decryptor_array = await __wx_channels_decrypt(profile.key);
  }

  var response = await fetch(profile.url);
  var blob = await show_progress_or_loaded_size(response);

  var array = new Uint8Array(await blob.arrayBuffer());
  if (profile.decryptor_array) {
    console.log('🔐 开始解密视频');
    array = __wx_channels_video_decrypt(array, 0, profile);
    console.log('✓ 视频解密完成');
  }

  var result = new Blob([array], { type: "video/mp4" });
  saveAs(result, filename + ".mp4");
}

// ==================== 点击下载处理 ====================
async function __wx_channels_handle_click_download__(spec) {
  var profile = __wx_channels_store__.profile;
  if (!profile) {
    alert("检测不到视频，请将本工具更新到最新版");
    return;
  }

  var filename = profile.title || profile.id || String(new Date().valueOf());
  var _profile = Object.assign({}, profile);

  if (spec) {
    _profile.url = profile.url + "&X-snsvideoflag=" + spec.fileFormat;
    var qualityInfo = spec.fileFormat;
    if (spec.width && spec.height) {
      qualityInfo += '_' + spec.width + 'x' + spec.height;
    }
    filename = filename + "_" + qualityInfo;
  }

  __wx_log({ msg: '下载文件名<' + filename + '>' });
  __wx_log({ msg: '视频链接<' + _profile.url + '>' });

  if (_profile.type === "picture") {
    __wx_channels_download3(_profile, filename);
    return;
  }

  if (!_profile.url) {
    alert("视频URL为空，无法下载");
    return;
  }

  var authorName = _profile.nickname || (_profile.contact && _profile.contact.nickname) || '未知作者';
  var hasKey = !!(_profile.key && _profile.key.length > 0);

  // 获取分辨率信息
  var resolution = '';
  var width = 0, height = 0, fileFormat = '';

  if (spec && spec.width && spec.height) {
    width = spec.width;
    height = spec.height;
    resolution = spec.width + 'x' + spec.height;
    fileFormat = spec.fileFormat || '';
  } else if (_profile.spec && _profile.spec.length > 0) {
    var firstSpec = _profile.spec[0];
    width = firstSpec.width || 0;
    height = firstSpec.height || 0;
    resolution = width && height ? (width + 'x' + height) : '';
    fileFormat = firstSpec.fileFormat || '';
  }

  var requestData = {
    videoUrl: _profile.url,
    videoId: _profile.id || '',
    title: filename,
    author: authorName,
    key: _profile.key || '',
    forceSave: false,
    resolution: resolution,
    width: width,
    height: height,
    fileFormat: fileFormat,
    likeCount: _profile.likeCount || 0,
    commentCount: _profile.commentCount || 0,
    forwardCount: _profile.forwardCount || 0,
    favCount: _profile.favCount || 0
  };

  var headers = { 'Content-Type': 'application/json' };
  if (window.__WX_LOCAL_TOKEN__) {
    headers['X-Local-Auth'] = window.__WX_LOCAL_TOKEN__;
  }

  __wx_log({ msg: '📥 开始下载: ' + filename.substring(0, 30) + '...' });

  fetch('/__wx_channels_api/download_video', {
    method: 'POST',
    headers: headers,
    body: JSON.stringify(requestData)
  })
    .then(function (response) { return response.json(); })
    .then(function (data) {
      if (data.success) {
        var msg = data.skipped ? '⏭️ 文件已存在，跳过下载' : (hasKey ? '✓ 视频已下载并解密' : '✓ 视频已下载');
        var path = data.relativePath || data.path || '';
        __wx_log({ msg: msg });
      } else {
        __wx_log({ msg: '❌ ' + (data.error || '下载视频失败') });
        alert('下载失败: ' + (data.error || '下载视频失败'));
      }
    })
    .catch(function (error) {
      __wx_log({ msg: '❌ 下载视频失败: ' + error.message });
      alert('下载失败: ' + error.message);
    });
}

// ==================== 封面下载 ====================
async function __wx_channels_handle_download_cover() {
  var profile = __wx_channels_store__.profile;
  if (!profile) {
    alert("未找到视频信息");
    return;
  }

  var coverUrl = profile.thumbUrl || profile.fullThumbUrl || profile.coverUrl;
  if (!coverUrl) {
    alert("未找到封面图片");
    return;
  }

  __wx_log({ msg: '正在保存封面到服务器...' });

  var requestData = {
    coverUrl: coverUrl,
    videoId: profile.id || '',
    title: profile.title || '',
    author: profile.nickname || (profile.contact && profile.contact.nickname) || '未知作者',
    forceSave: false
  };

  var headers = { 'Content-Type': 'application/json' };
  if (window.__WX_LOCAL_TOKEN__) {
    headers['X-Local-Auth'] = window.__WX_LOCAL_TOKEN__;
  }

  fetch('/__wx_channels_api/save_cover', {
    method: 'POST',
    headers: headers,
    body: JSON.stringify(requestData)
  })
    .then(function (response) { return response.json(); })
    .then(function (data) {
      if (data.success) {
        __wx_log({ msg: '✓ ' + (data.message || '封面已保存') });
      } else {
        __wx_log({ msg: '❌ ' + (data.error || '保存封面失败') });
        alert('保存封面失败: ' + (data.error || '未知错误'));
      }
    })
    .catch(function (error) {
      __wx_log({ msg: '❌ 保存封面失败: ' + error.message });
      alert("保存封面失败: " + error.message);
    });
}

console.log('[download.js] 下载模块加载完成');
