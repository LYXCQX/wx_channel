<template>
  <div class="min-h-screen bg-bg p-4 lg:p-12 font-sans text-text">
    <header class="flex flex-col lg:flex-row justify-between items-start gap-4 lg:gap-0 mb-6 lg:mb-12">
      <div class="flex items-start gap-4 lg:gap-6 flex-1 w-full">
          <button class="w-10 h-10 lg:w-12 lg:h-12 rounded-full bg-bg shadow-neu-btn flex items-center justify-center text-text hover:text-primary active:shadow-neu-btn-active transition-all shrink-0" @click="goBack">
            ←
          </button>
          <div class="flex flex-col md:flex-row items-start md:items-center gap-4 flex-1 w-full" v-if="author">
             <div class="w-16 h-16 lg:w-20 lg:h-20 rounded-full bg-bg shadow-neu-sm p-1 shrink-0">
                <img :src="author.headUrl || placeholderImg" class="w-full h-full rounded-full object-cover" @error="onImgError">
             </div>
             <div class="flex-1 w-full">
                <div class="flex flex-wrap items-center justify-between gap-2">
                    <h2 class="font-serif font-bold text-xl lg:text-2xl text-text mb-1">{{ author.nickname }}</h2>
                     <!-- Subscribe Button (Mobile: Top Right) -->
                     <button 
                         @click="toggleSubscribe" 
                         :disabled="subscribing"
                         class="md:hidden px-4 py-2 rounded-xl font-semibold text-xs shadow-neu-btn transition-all disabled:opacity-50 whitespace-nowrap"
                         :class="isSubscribed ? 'bg-bg text-text-muted hover:text-red-500' : 'bg-primary text-white hover:bg-primary-dark'">
                         {{ subscribing ? '...' : (isSubscribed ? '已订阅' : '订阅') }}
                     </button>
                </div>
                <p class="text-text-muted text-xs lg:text-sm max-w-md line-clamp-2 break-all">{{ author.signature || '暂无签名' }}</p>
                <div class="flex items-center gap-4 mt-2 text-xs text-text-muted">
                    <span class="flex items-center gap-1">
                        <Video class="w-3 h-3" />
                        {{ videos.length }} 个视频
                    </span>
                </div>
             </div>
             <!-- Subscribe Button (Desktop) -->
             <button 
                 @click="toggleSubscribe" 
                 :disabled="subscribing"
                 class="hidden md:block px-6 py-3 rounded-xl font-semibold shadow-neu-btn transition-all disabled:opacity-50 whitespace-nowrap"
                 :class="isSubscribed ? 'bg-bg text-text-muted hover:text-red-500' : 'bg-primary text-white hover:bg-primary-dark'">
                 {{ subscribing ? '处理中...' : (isSubscribed ? '已订阅' : '订阅') }}
             </button>
          </div>
      </div>
      <div v-if="client" class="self-end lg:self-auto px-3 py-1.5 lg:px-4 lg:py-2 rounded-xl bg-bg shadow-neu-sm border border-white/50 text-primary font-medium flex items-center gap-2 mt-2 lg:mt-0 max-w-full">
        <span class="text-[10px] lg:text-xs uppercase tracking-wider text-text-muted truncate">Connected to</span>
        <strong class="text-xs lg:text-sm truncate">{{ client.hostname }}</strong>
      </div>
      <div v-else class="self-end lg:self-auto px-3 py-1.5 lg:px-4 lg:py-2 rounded-xl bg-yellow-50 border border-yellow-200 text-yellow-700 text-xs lg:text-sm flex items-center gap-2 mt-2 lg:mt-0">
        <Zap class="w-3 h-3 lg:w-4 lg:h-4" />
        <span>自动选择设备</span>
      </div>
    </header>

    <div class="max-w-5xl mx-auto">
        <!-- Loading State -->
        <div v-if="loadingVideos && videos.length === 0" class="flex flex-col items-center justify-center p-12">
          <div class="w-12 h-12 border-4 border-primary/30 border-t-primary rounded-full animate-spin mb-4"></div>
          <p class="text-text-muted">加载视频中...</p>
        </div>

        <!-- Client Inactive State -->
        <div v-else-if="isClientInactive" class="flex flex-col items-center justify-center p-8 lg:p-12 text-center max-w-2xl mx-auto">
            <div class="w-16 h-16 lg:w-20 lg:h-20 bg-orange-50 rounded-full flex items-center justify-center mb-6">
                <i class="pi pi-lock text-orange-500 text-2xl lg:text-3xl"></i>
            </div>
            <h3 class="text-xl lg:text-2xl font-bold text-text mb-3">无法连接到视频号页面</h3>
            <p class="text-text-muted mb-8 lg:text-lg">请确保您的微信视频号页面处于打开且激活状态。</p>
            
            <div class="bg-white border border-slate-100 rounded-3xl p-6 lg:p-8 text-left w-full mb-8 shadow-card">
                <h4 class="font-bold text-text mb-4">解决方案：</h4>
                <ol class="list-decimal list-inside space-y-3 text-text-muted text-sm lg:text-base">
                    <li>在浏览器中打开 <strong class="text-text">微信视频号</strong> 页面。</li>
                    <li>确保该页面没有被最小化，且处于当前浏览器的 <strong class="text-text">可见标签页</strong>。</li>
                    <li>如果页面已打开，请尝试刷新该页面。</li>
                    <li>完成后点击下方的 <strong class="text-primary">重试</strong> 按钮。</li>
                </ol>
            </div>

            <button class="px-8 py-3 rounded-xl bg-primary text-white font-semibold shadow-neu-btn hover:bg-primary-dark transition-all active:shadow-neu-btn-active flex items-center gap-2 mx-auto" @click="fetchVideos(false)">
                <span>已解决，重试</span>
            </button>
        </div>
        
        <!-- Video Grid -->
        <div v-else-if="videos.length > 0" class="flex flex-col gap-4 lg:gap-6">
          <div v-for="video in videos" :key="video.id" class="p-4 lg:p-6 rounded-2xl lg:rounded-3xl bg-white shadow-card border border-slate-100 flex flex-col md:flex-row gap-4 lg:gap-6 transition-all hover:shadow-lg hover:-translate-y-0.5 group">
            <!-- Video Thumbnail -->
            <div class="relative w-full md:w-56 aspect-video shrink-0 rounded-xl lg:rounded-2xl overflow-hidden shadow-inner bg-slate-100 cursor-pointer" @click="playVideo(video)">
               <img :src="video.coverUrl" class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500" @error="onImgError">
               <!-- Play Overlay -->
               <div class="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-colors flex items-center justify-center">
                   <div class="opacity-0 group-hover:opacity-100 transition-opacity bg-primary/90 text-white p-2 lg:p-3 rounded-full backdrop-blur-sm shadow-xl">
                       <PlayCircle class="w-6 h-6 lg:w-8 lg:h-8" />
                   </div>
               </div>
               <!-- Duration Badge -->
               <div class="absolute bottom-2 right-2 bg-black/70 backdrop-blur-sm text-white text-[10px] lg:text-xs px-1.5 py-0.5 lg:px-2 lg:py-1 rounded-md font-medium">
                   {{ video.duration }}
               </div>
            </div>
            
            <!-- Video Info -->
            <div class="flex-1 flex flex-col justify-between py-1 lg:py-2">
              <div>
                 <h3 class="font-bold text-base lg:text-lg text-text mb-1 lg:mb-2 line-clamp-2 leading-snug">{{ video.title || '无标题视频' }}</h3>
                 <div class="flex flex-wrap gap-2 lg:gap-3 text-xs text-text-muted font-medium mb-3">
                     <span class="flex items-center gap-1">
                         <Clock class="w-3 h-3" />
                         {{ formatTime(video.createTime * 1000) }}
                     </span>
                     <span class="flex items-center gap-1 px-2 py-0.5 rounded-md bg-slate-50 border border-slate-200">
                         <Monitor class="w-3 h-3" />
                         {{ video.width }}x{{ video.height }}
                     </span>
                 </div>
              </div>
              
              <!-- Action Buttons -->
              <div class="flex gap-3 mt-2 md:mt-0">
                <button class="flex-1 md:flex-none px-4 py-2 lg:px-6 lg:py-2.5 rounded-xl bg-primary text-white text-xs lg:text-sm font-semibold shadow-neu-btn hover:bg-primary-dark active:shadow-neu-btn-active transition-all flex items-center justify-center gap-2" @click="playVideo(video)">
                    <PlayCircle class="w-3.5 h-3.5 lg:w-4 lg:h-4" />
                    <span>播放</span>
                </button>
                <button class="flex-1 md:flex-none px-4 py-2 lg:px-6 lg:py-2.5 rounded-xl bg-bg text-text-muted text-xs lg:text-sm font-semibold shadow-neu-btn hover:text-primary active:shadow-neu-btn-active transition-all flex items-center justify-center gap-2" @click="downloadVideo(video)">
                    <Download class="w-3.5 h-3.5 lg:w-4 lg:h-4" />
                    <span>下载</span>
                </button>
              </div>
            </div>
          </div>
          
          <!-- Load More Button -->
          <div v-if="hasMoreVideos" class="text-center mt-4 lg:mt-8 pb-8 lg:pb-12">
              <button class="px-6 py-2.5 lg:px-8 lg:py-3 rounded-full bg-bg shadow-neu-btn text-text-muted text-sm font-medium hover:text-primary transition-all active:shadow-neu-btn-active disabled:opacity-50 flex items-center gap-2 mx-auto" @click="fetchVideos(true)" :disabled="loadingVideos">
                  <div v-if="loadingVideos" class="w-4 h-4 border-2 border-text-muted/30 border-t-text-muted rounded-full animate-spin"></div>
                  <span>{{ loadingVideos ? '加载中...' : '加载更多视频' }}</span>
              </button>
          </div>
          
          <!-- No More Videos -->
          <div v-else class="text-center p-6 text-text-muted text-xs lg:text-sm">
              已显示全部视频
          </div>
        </div>
        
        <!-- Empty State -->
        <div v-else class="text-center p-8 lg:p-16 text-text-muted bg-white rounded-[2rem] shadow-card">
            <Video class="w-12 h-12 lg:w-16 lg:h-16 mx-auto mb-4 text-text-muted/30" />
            <p class="text-base lg:text-lg font-medium mb-2">暂无视频动态</p>
            <p class="text-xs lg:text-sm">该用户还没有发布任何视频</p>
        </div>
    </div>
    
    <!-- Video Player Modal -->
    <div v-if="playerUrl" class="fixed inset-0 z-50 flex justify-center items-center bg-black/80 backdrop-blur-md p-4" @click="closePlayer">
      <div class="w-full max-w-5xl bg-white rounded-2xl lg:rounded-3xl shadow-card border border-slate-100 p-4 lg:p-6" @click.stop>
        <div class="flex justify-between items-center mb-4">
          <h3 class="font-serif font-bold text-lg lg:text-xl text-text truncate pr-4">{{ currentVideo?.title || '视频预览' }}</h3>
          <button class="w-8 h-8 lg:w-10 lg:h-10 rounded-full bg-bg shadow-neu-btn flex items-center justify-center text-text hover:text-red-500 active:shadow-neu-btn-active transition-all text-xl lg:text-2xl leading-none" @click="closePlayer">×</button>
        </div>
        <div class="rounded-xl lg:rounded-2xl overflow-hidden shadow-inner bg-black aspect-video">
           <video :src="playerUrl" controls autoplay class="w-full h-full"></video>
        </div>
        <!-- Video Info -->
        <div v-if="currentVideo" class="mt-4 p-3 lg:p-4 bg-slate-50 rounded-xl">
            <div class="flex items-center gap-3 text-xs lg:text-sm text-text-muted">
                <span class="flex items-center gap-1">
                    <Clock class="w-3.5 h-3.5 lg:w-4 lg:h-4" />
                    {{ formatTime(currentVideo.createTime * 1000) }}
                </span>
                <span class="flex items-center gap-1">
                    <Monitor class="w-3.5 h-3.5 lg:w-4 lg:h-4" />
                    {{ currentVideo.width }}x{{ currentVideo.height }}
                </span>
                <span class="flex items-center gap-1">
                    <Video class="w-3.5 h-3.5 lg:w-4 lg:h-4" />
                    {{ currentVideo.duration }}
                </span>
            </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useClientStore } from '../store/client'
import { useRouter, useRoute } from 'vue-router'
import { formatTime, formatDuration } from '../utils/format'
import { Zap, PlayCircle, Download, Clock, Monitor, Video } from 'lucide-vue-next'

const clientStore = useClientStore()
const router = useRouter()
const route = useRoute()
const client = computed(() => clientStore.currentClient)

// 使用 data URI 避免外部请求和混合内容问题
const placeholderImg = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100"%3E%3Crect fill="%23f1f5f9" width="100" height="100"/%3E%3Ctext x="50" y="50" font-family="sans-serif" font-size="14" fill="%2394a3b8" text-anchor="middle" dominant-baseline="middle"%3E暂无图片%3C/text%3E%3C/svg%3E'

// 确保 URL 使用 HTTPS 协议
const ensureHttps = (url) => {
    if (!url || url === placeholderImg) return url
    return url.replace(/^http:\/\//i, 'https://')
}

const author = ref({
    username: '',
    nickname: '',
    headUrl: '',
    signature: ''
})

const loadingVideos = ref(false)
const videos = ref([])
const playerUrl = ref('')
const currentVideo = ref(null)
const error = ref(null)
const isClientInactive = ref(false)

// Subscription state
const isSubscribed = ref(false)
const subscribing = ref(false)
const subscriptionId = ref(null)

const lastVideoMarker = ref('')
const hasMoreVideos = ref(false)

onMounted(() => {
    // Restore author info from query
    const q = route.query
    if (q.username) {
        author.value = {
            username: q.username,
            nickname: q.nickname || '未知用户',
            headUrl: ensureHttps(q.headUrl || ''),
            signature: q.signature || ''
        }
        fetchVideos(false)
        checkSubscriptionStatus()
    } else {
        alert("无效的用户参数")
        router.push('/search')
    }
})

const goBack = () => {
    router.push('/search')
}

const fetchVideos = async (loadMore = false) => {
  if (!loadMore) {
      loadingVideos.value = true
      videos.value = []
      lastVideoMarker.value = ''
      hasMoreVideos.value = false
      error.value = null
      isClientInactive.value = false
  } else {
      loadingVideos.value = true
  }
  
  try {
    const res = await clientStore.remoteCall('api_call', {
      key: 'key:channels:feed_list',
      body: { 
          username: author.value.username, 
          next_marker: loadMore ? lastVideoMarker.value : '' 
      }
    })
    
    // Config adapter: robustly find the video list
    let objects = []
    const findObjects = (obj) => {
        if (!obj) return null
        if (Array.isArray(obj.object)) return obj.object
        if (Array.isArray(obj.list)) return obj.list
        return null
    }
    
    // 1. Try res.data (Hub payload -> data)
    if (res.data) {
        objects = findObjects(res.data)
        const payload = res.data.payload || {}
        if (res.data.continueFlag || payload.lastBuffer) {
             lastVideoMarker.value = payload.lastBuffer || res.data.lastBuffer || ''
             hasMoreVideos.value = !!lastVideoMarker.value
        }

        // 2. Try res.data.data (Hub payload -> data -> business payload)
        if (!objects && res.data.data) {
            objects = findObjects(res.data.data)
            const payload = res.data.data.payload || {}
            if (res.data.data.continueFlag || payload.lastBuffer) {
                lastVideoMarker.value = payload.lastBuffer || res.data.data.lastBuffer || ''
                hasMoreVideos.value = !!lastVideoMarker.value
            }
        }
    }
    // 3. Try root
    if (!objects) {
        objects = findObjects(res) || []
    }

    if (!Array.isArray(objects)) objects = [] 
    
    const newVideos = objects.map(item => {
        const v = item.object || item
        const desc = v.objectDesc || v.desc || {}
        const media = (desc.media && desc.media[0]) || {}
        return {
            id: v.id || v.objectId || v.displayid,
            nonceId: v.nonceId || v.objectNonceId,
            title: desc.description,
            coverUrl: ensureHttps(v.coverUrl || media.thumbUrl || media.coverUrl),
            createTime: v.createtime || v.createTime,
            width: media.width || 0,
            height: media.height || 0,
            duration: formatDuration(v.videoPlayLen || media.videoPlayLen || 0),
            authorName: author.value.nickname
        }
    })

    if (loadMore) {
        videos.value = [...videos.value, ...newVideos]
    } else {
        videos.value = newVideos
    }
  } catch (err) {
    console.error('获取视频失败:', err)
    if (err.message && err.message.includes('客户端页面未激活')) {
        isClientInactive.value = true
        error.value = '无法连接到视频号页面'
    } else {
        alert('获取视频失败: ' + err.message)
    }
  } finally {
    loadingVideos.value = false
  }
}

const resolveVideoUrl = async (video) => {
    const res = await clientStore.remoteCall('api_call', {
        key: 'key:channels:feed_profile',
        body: { object_id: video.id, nonce_id: video.nonceId }
    })
    
    let actual = {}
    if (res.data && res.data.object) {
        actual = res.data.object
    } else if (res.data && res.data.data && res.data.data.object) {
        actual = res.data.data.object
    } else {
        actual = (res.data || {})
    }

    const mediaArray = (actual.objectDesc && actual.objectDesc.media) || actual.media || []
    const media = mediaArray[0]
    
    if (!media || !media.url) throw new Error("无法获取视频地址")
    
    let videoUrl = media.url + (media.urlToken || '')
    const decryptKey = media.decodeKey || ''
    
    if (media.spec && media.spec.length > 0) {
        const lowestSpec = media.spec.reduce((prev, curr) => {
            return (curr.bitRate || 99999) < (prev.bitRate || 99999) ? curr : prev
        })
        if (lowestSpec.fileFormat) {
            videoUrl += `&X-snsvideoflag=${lowestSpec.fileFormat}`
        }
    }
    
    let finalUrl = `/api/video/play?url=${encodeURIComponent(videoUrl)}`
    if (decryptKey) finalUrl += `&key=${decryptKey}`
    
    return finalUrl
}

const playVideo = async (video) => {
    try {
        currentVideo.value = video
        const url = await resolveVideoUrl(video)
        playerUrl.value = url
    } catch (e) {
        console.error('播放视频失败:', e)
        alert('播放失败: ' + e.message)
    }
}

const downloadVideo = async (video) => {
    try {
        const url = await resolveVideoUrl(video)
        const a = document.createElement('a')
        a.href = url
        a.download = (video.title || 'video') + '.mp4'
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
    } catch (e) {
        console.error('下载视频失败:', e)
        alert('下载失败: ' + e.message)
    }
}

const closePlayer = () => {
    playerUrl.value = ''
    currentVideo.value = null
}

const onImgError = (e) => {
  e.target.src = placeholderImg
}

// Subscription functions
const checkSubscriptionStatus = async () => {
    try {
        const token = localStorage.getItem('token')
        if (!token) return
        
        const res = await fetch('/api/subscriptions', {
            headers: { 'Authorization': `Bearer ${token}` }
        })
        const data = await res.json()
        if (data.code === 0) {
            const subscription = (data.data || []).find(sub => sub.wx_username === author.value.username)
            if (subscription) {
                isSubscribed.value = true
                subscriptionId.value = subscription.id
            }
        }
    } catch (e) {
        console.error('Failed to check subscription status:', e)
    }
}

const toggleSubscribe = async () => {
    subscribing.value = true
    try {
        const token = localStorage.getItem('token')
        
        if (isSubscribed.value) {
            // Unsubscribe
            if (!subscriptionId.value) return
            
            const res = await fetch(`/api/subscriptions/${subscriptionId.value}`, {
                method: 'DELETE',
                headers: { 'Authorization': `Bearer ${token}` }
            })
            
            if (res.ok) {
                isSubscribed.value = false
                subscriptionId.value = null
            } else {
                alert('取消订阅失败')
            }
        } else {
            // Subscribe
            const res = await fetch('/api/subscriptions', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify({
                    wx_username: author.value.username,
                    wx_nickname: author.value.nickname,
                    wx_head_url: author.value.headUrl,
                    wx_signature: author.value.signature
                })
            })
            
            const data = await res.json()
            if (data.code === 0) {
                isSubscribed.value = true
                subscriptionId.value = data.data.id
            } else {
                alert('订阅失败: ' + (data.message || ''))
            }
        }
    } catch (e) {
        console.error('Subscription error:', e)
        alert('操作失败: ' + e.message)
    } finally {
        subscribing.value = false
    }
}
</script>
