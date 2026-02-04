<template>
  <div class="min-h-screen bg-bg p-8 lg:p-12 font-sans text-text">
    <header class="flex justify-between items-start mb-12">
      <div class="flex items-center gap-6 flex-1">
          <button class="w-12 h-12 rounded-full bg-bg shadow-neu-btn flex items-center justify-center text-text hover:text-primary active:shadow-neu-btn-active transition-all" @click="goBack">
            ←
          </button>
          <div class="flex items-center gap-4 flex-1" v-if="author">
             <div class="w-16 h-16 rounded-full bg-bg shadow-neu-sm p-1">
                <img :src="author.headUrl || placeholderImg" class="w-full h-full rounded-full object-cover" @error="onImgError">
             </div>
             <div class="flex-1">
                <h2 class="font-serif font-bold text-2xl text-text mb-1">{{ author.nickname }} 的动态</h2>
                <p class="text-text-muted text-sm max-w-md">{{ author.signature || '暂无签名' }}</p>
             </div>
             <!-- Subscribe Button -->
             <button 
                 @click="toggleSubscribe" 
                 :disabled="subscribing"
                 class="px-6 py-3 rounded-xl font-semibold shadow-neu-btn transition-all disabled:opacity-50 whitespace-nowrap"
                 :class="isSubscribed ? 'bg-bg text-text-muted hover:text-red-500' : 'bg-primary text-white hover:bg-primary-dark'">
                 {{ subscribing ? '处理中...' : (isSubscribed ? '已订阅' : '订阅') }}
             </button>
          </div>
      </div>
      <div v-if="client" class="px-4 py-2 rounded-xl bg-bg shadow-neu-sm border border-white/50 text-primary font-medium flex items-center gap-2">
        <span class="text-xs uppercase tracking-wider text-text-muted">Connected to</span>
        <strong>{{ client.hostname }}</strong>
      </div>
      <div v-else class="px-4 py-2 rounded-xl bg-yellow-50 border border-yellow-200 text-yellow-700 text-sm flex items-center gap-2">
        <Zap class="w-4 h-4" />
        <span>自动选择设备</span>
      </div>
    </header>

    <div class="max-w-4xl mx-auto">
        <div v-if="loadingVideos && !hasMoreVideos && videos.length === 0" class="flex justify-center p-12">
          <div class="w-8 h-8 border-4 border-primary/30 border-t-primary rounded-full animate-spin"></div>
        </div>
        
        <div v-else class="flex flex-col gap-6">
          <div v-for="video in videos" :key="video.id" class="p-4 rounded-3xl bg-white shadow-card border border-slate-100 flex flex-col md:flex-row gap-6 transition-all hover:shadow-lg group">
            <div class="relative w-full md:w-48 aspect-video shrink-0 rounded-2xl overflow-hidden shadow-inner">
               <img :src="video.coverUrl" class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500">
               <div class="absolute bottom-2 right-2 bg-black/60 backdrop-blur-sm text-white text-xs px-2 py-1 rounded-md font-medium">{{ video.duration }}</div>
            </div>
            <div class="flex-1 flex flex-col justify-between py-2">
              <div>
                 <div class="font-bold text-lg text-text mb-2 line-clamp-2">{{ video.title || '无标题' }}</div>
                 <div class="flex gap-4 text-xs text-text-muted font-medium mb-4">
                     <span>{{ formatTime(video.createTime * 1000) }}</span>
                     <span class="px-2 py-0.5 rounded-md bg-white/50 border border-white/50">{{ video.authorName }} @ {{ video.width }}x{{ video.height }}</span>
                 </div>
              </div>
              <div class="flex gap-3">
                <button class="px-6 py-2 rounded-xl bg-primary text-white text-sm font-semibold shadow-neu-btn hover:bg-primary-dark active:shadow-neu-btn-active transition-all" @click="playVideo(video)">在线播放</button>
                <button class="px-6 py-2 rounded-xl bg-bg text-text-muted text-sm font-semibold shadow-neu-btn hover:text-primary active:shadow-neu-btn-active transition-all" @click="downloadVideo(video)">下载</button>
              </div>
            </div>
          </div>
          
          <div v-if="hasMoreVideos" class="text-center mt-8 pb-12">
              <button class="px-8 py-3 rounded-full bg-bg shadow-neu-btn text-text-muted font-medium hover:text-primary transition-all active:shadow-neu-btn-active disabled:opacity-50" @click="fetchVideos(true)" :disabled="loadingVideos">
                  {{ loadingVideos ? '加载中...' : '加载更多视频' }}
              </button>
          </div>
          
          <div v-if="!loadingVideos && videos.length === 0" class="text-center p-12 text-text-muted bg-white rounded-[2rem] shadow-card">
              暂无视频动态
          </div>
        </div>
    </div>
    
    <!-- Video Player Modal -->
    <div v-if="playerUrl" class="fixed inset-0 z-50 flex justify-center items-center bg-black/60 backdrop-blur-md p-4" @click="closePlayer">
      <div class="w-full max-w-5xl bg-white rounded-3xl shadow-card border border-slate-100 p-4" @click.stop>
        <div class="flex justify-between items-center mb-4 px-2">
          <h3 class="font-serif font-bold text-xl text-text">视频预览</h3>
          <button class="w-10 h-10 rounded-full bg-bg shadow-neu-btn flex items-center justify-center text-text hover:text-red-500 active:shadow-neu-btn-active transition-all" @click="closePlayer">×</button>
        </div>
        <div class="rounded-2xl overflow-hidden shadow-inner bg-black aspect-video">
           <video :src="playerUrl" controls autoplay class="w-full h-full"></video>
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
import { Zap } from 'lucide-vue-next'

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
            coverUrl: ensureHttps(v.coverUrl || media.thumbUrl),
            createTime: v.createtime || v.createTime,
            width: media.width,
            height: media.height,
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
    alert('获取视频失败: ' + err.message)
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
        const url = await resolveVideoUrl(video)
        playerUrl.value = url
    } catch (e) {
        alert(e.message)
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
        alert(e.message)
    }
}

const closePlayer = () => {
    playerUrl.value = ''
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
