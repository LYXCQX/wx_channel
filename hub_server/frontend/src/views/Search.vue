<template>
  <div class="min-h-screen bg-bg p-8 lg:p-12 font-sans text-text">
    <header class="flex justify-between items-center mb-12">
      <div v-if="client" class="px-4 py-2 rounded-xl bg-bg shadow-neu-sm border border-white/50 text-primary font-medium flex items-center gap-2">
        <span class="text-xs uppercase tracking-wider text-text-muted">Connected to</span>
        <strong>{{ client.hostname }}</strong>
      </div>
      <div v-else class="px-4 py-2 rounded-xl bg-yellow-50 border border-yellow-200 text-yellow-700 text-sm flex items-center gap-2">
        <Zap class="w-4 h-4" />
        <span>自动选择设备</span>
      </div>
    </header>

    <!-- Search Box & Type Selector -->
    <div class="flex flex-col gap-4 mb-12 p-6 bg-white rounded-[2rem] shadow-card">
      <!-- Type Selector -->
      <div class="flex gap-4 px-2">
          <label class="flex items-center gap-2 cursor-pointer">
              <input type="radio" v-model.number="searchType" :value="1" class="accent-primary">
              <span :class="{'text-primary font-bold': searchType===1, 'text-text-muted': searchType!==1}">找人</span>
          </label>
          <label class="flex items-center gap-2 cursor-pointer">
              <input type="radio" v-model.number="searchType" :value="3" class="accent-primary">
              <span :class="{'text-primary font-bold': searchType===3, 'text-text-muted': searchType!==3}">找视频</span>
          </label>
          <label class="flex items-center gap-2 cursor-pointer">
              <input type="radio" v-model.number="searchType" :value="2" class="accent-primary">
              <span :class="{'text-primary font-bold': searchType===2, 'text-text-muted': searchType!==2}">找直播</span>
          </label>
      </div>

      <div class="flex gap-4 items-center">
          <input 
          v-model="keyword" 
          type="text" 
          class="flex-1 bg-transparent border-none outline-none text-lg px-2 text-text placeholder-text-muted/50" 
          placeholder="输入关键词..."
          @keyup.enter="handleSearch(false)"
          >
          <button 
              class="px-8 py-3 rounded-full bg-primary text-white font-semibold shadow-lg shadow-primary/30 hover:bg-primary-dark transition-all transform hover:-translate-y-0.5 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              @click="handleSearch(false)" 
              :disabled="searching"
          >
          <Search v-if="!searching" class="w-5 h-5" />
          <div v-else class="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
          <span>{{ searching ? '搜索中...' : '开始搜索' }}</span>
          </button>
      </div>
    </div>

    <!-- Results Grid -->
    <div v-if="results.length > 0" class="grid grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-6">
        <div 
          v-for="(item, idx) in results" 
          :key="idx" 
          class="bg-white rounded-2xl overflow-hidden shadow-card border border-slate-100 cursor-pointer transition-all hover:-translate-y-1 hover:shadow-lg hover:border-primary/30 group relative"
          @click="openDetail(item)"
        >
          <!-- User Head (Type 1) -->
          <div v-if="searchType === 1" class="p-6 text-center">
              <div class="w-20 h-20 mx-auto rounded-full bg-bg shadow-neu-sm p-1 mb-4 group-hover:shadow-neu-pressed transition-shadow">
                 <img :src="getHeadUrl(item)" class="w-full h-full rounded-full object-cover" @error="onImgError">
              </div>
              <div class="font-bold text-lg mb-1 text-text group-hover:text-primary transition-colors line-clamp-1">{{ getNickname(item) }}</div>
              <div class="text-xs text-text-muted line-clamp-2 px-2 mb-3">{{ stripHtml(item.signature || item.contact?.signature || '暂无签名') }}</div>
              <!-- Subscribe Button -->
              <button 
                  @click.stop="toggleSubscribe(item)" 
                  :disabled="subscribing"
                  class="px-4 py-2 rounded-xl text-sm font-semibold shadow-neu-btn transition-all disabled:opacity-50"
                  :class="isSubscribed(item) ? 'bg-bg text-text-muted hover:text-red-500' : 'bg-primary text-white hover:bg-primary-dark'">
                  {{ subscribing ? '处理中...' : (isSubscribed(item) ? '已订阅' : '订阅') }}
              </button>
          </div>

          <!-- Video Cover (Type 3) -->
           <div v-else-if="searchType === 3" class="relative aspect-[9/16] w-full">
             <!-- Cover Image -->
             <img :src="getVideoCover(item)" class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105" @error="onImgError">
             
             <!-- Gradient Overlay -->
             <div class="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent opacity-80"></div>

             <!-- Play Icon Overlay -->
             <div class="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity duration-300">
                 <div class="bg-primary/90 text-white p-3 rounded-full backdrop-blur-sm shadow-xl transform scale-75 group-hover:scale-100 transition-transform">
                     <PlayCircle class="w-8 h-8 fill-current" />
                 </div>
             </div>

             <!-- Bottom Info -->
             <div class="absolute bottom-0 left-0 right-0 p-4 text-white">
                 <div class="font-bold text-sm line-clamp-2 mb-1 leading-snug" v-html="getVideoTitle(item)"></div>
                 <div class="flex items-center justify-between">
                     <div class="flex items-center gap-1 text-xs text-white/70">
                        <span>@{{ getNickname(item) }}</span>
                     </div>
                     <div class="flex items-center gap-1 text-xs text-white/90" v-if="getLikeCount(item) > 0">
                         <Heart class="w-3 h-3 fill-current" />
                         <span>{{ getLikeCount(item) }}</span>
                     </div>
                 </div>
             </div>
          </div>

           <!-- Live Cover (Type 2) -->
           <div v-else-if="searchType === 2" class="relative aspect-[9/16] w-full">
               <!-- Live Badge -->
               <div class="absolute top-2 left-2 bg-red-500 text-white text-[10px] font-bold px-2 py-0.5 rounded-sm z-10 flex items-center gap-1 shadow-sm animate-pulse">
                   <div class="w-1.5 h-1.5 bg-white rounded-full"></div>
                   LIVE
               </div>

             <img :src="getLiveCover(item)" class="w-full h-full object-cover" @error="onImgError">
             
             <div class="absolute inset-0 bg-black/10 group-hover:bg-black/0 transition-colors"></div>
             
             <div class="absolute bottom-0 left-0 right-0 p-3 bg-gradient-to-t from-black/80 to-transparent">
                 <div class="font-bold text-white text-sm line-clamp-1 mb-1">{{ getLiveTitle(item) }}</div>
                 <div class="flex items-center justify-between text-xs text-white/80">
                      <span>{{ getNickname(item) }}</span>
                      <span v-if="getLiveViewerCount(item)" class="flex items-center gap-1">
                          <Users class="w-3 h-3" />
                          {{ getLiveViewerCount(item) }}
                      </span>
                 </div>
             </div>
          </div>

      </div>
    </div>
    
    
    <!-- No More Results Notice for Live (Type 2) -->
    <div v-if="searchType === 2 && results.length > 0 && !searching" class="text-center mt-12 mb-8">
        <p class="text-text-muted text-sm">直播搜索不支持分页，已显示全部结果</p>
    </div>

    <!-- Load More Button (Hidden for Type 2) -->
    <div v-if="hasMoreSearch && searchType !== 2" class="text-center mt-12 mb-8">
        <button class="px-8 py-3 rounded-full bg-bg shadow-neu-btn text-text-muted font-medium hover:text-primary transition-all active:shadow-neu-btn-active disabled:opacity-50" @click="handleSearch(true)" :disabled="searching">
            {{ searching ? '加载中...' : '加载更多' }}
        </button>
    </div>
    
    <!-- Video Player Modal -->
    <div v-if="playerUrl" class="fixed inset-0 z-50 flex justify-center items-center bg-black/80 backdrop-blur-md p-4" @click="closePlayer">
      <div class="w-full max-w-5xl bg-white rounded-3xl shadow-card border border-slate-100 p-4" @click.stop>
        <div class="flex justify-between items-center mb-4 px-2">
          <h3 class="font-serif font-bold text-xl text-text">{{ playerTitle }}</h3>
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
import { ref, computed } from 'vue'
import { useClientStore } from '../store/client'
import { useRouter } from 'vue-router'
import { Search, PlayCircle, Heart, Users, Zap } from 'lucide-vue-next'

const clientStore = useClientStore()
const router = useRouter()
const client = computed(() => clientStore.currentClient)

const keyword = ref('')
const searchType = ref(3) // Default to Video
const searching = ref(false)
const results = ref([])
// 使用 data URI 避免外部请求和混合内容问题
const placeholderImg = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100"%3E%3Crect fill="%23f1f5f9" width="100" height="100"/%3E%3Ctext x="50" y="50" font-family="sans-serif" font-size="14" fill="%2394a3b8" text-anchor="middle" dominant-baseline="middle"%3E暂无图片%3C/text%3E%3C/svg%3E'

const lastSearchBuffer = ref('')
const hasMoreSearch = ref(false)
const searchSessionId = ref('')

// Video player state
const playerUrl = ref('')
const playerTitle = ref('')

// Subscription state
const subscriptions = ref([])
const subscribing = ref(false)

// Load subscriptions on mount
import { onMounted } from 'vue'
onMounted(() => {
    loadSubscriptions()
})

// Watch searchType to auto-trigger search
// When switching tabs, we must reset the list and search again with the new type
import { watch } from 'vue'
watch(searchType, (newVal) => {
    if (keyword.value) {
        handleSearch(false)
    } else {
        results.value = []
        hasMoreSearch.value = false
    }
})

const handleSearch = async (loadMore = false) => {
  if (!keyword.value) return
  searching.value = true
  if (!loadMore) {
      results.value = []
      lastSearchBuffer.value = ''
      hasMoreSearch.value = false
      // New search: Generate new session ID
      searchSessionId.value = String(new Date().valueOf())
  }
  
  console.log('Searching:', { 
      keyword: keyword.value, 
      type: searchType.value, 
      loadMore, 
      next_marker: loadMore ? lastSearchBuffer.value : '',
      request_id: searchSessionId.value 
  })

  try {
    const res = await clientStore.remoteCall('api_call', {
      key: 'key:channels:contact_list', // This key now handles all types based on 'type' param
      body: { 
          keyword: keyword.value,
          type: searchType.value,
          next_marker: loadMore ? lastSearchBuffer.value : '',
          request_id: searchSessionId.value
      }
    })
    console.log('Search response:', res)
    
    // Parse the new optimized response structure: { list: [], next_marker: "", has_more: bool }
    let data = res.data;
    // Hub wrapper might wrap it in 'data' again if the backend returned it wrapping the upstream response
    if (data.code === 0 && data.data) {
        data = data.data;
    }

    // Fallback logic if the response is still the Hub wrapper structure but inside data
    if (data.list === undefined && res.data.data?.list) {
         data = res.data.data;
    }

    const list = data.list || [];
    lastSearchBuffer.value = data.next_marker || '';
    // Type 2 (Live) does NOT support pagination - force disable
    hasMoreSearch.value = (searchType.value === 2) ? false : !!data.has_more;

    // Do NOT unwrap contact here. Handle it in the template helpers for flexibility.
    const newItems = list; 
    
    if (loadMore) {
        results.value = [...results.value, ...newItems]
    } else {
        results.value = newItems
    }

  } catch (err) {
    alert('搜索失败: ' + err.message)
    console.error(err)
  } finally {
    searching.value = false
  }
}

const openDetail = async (item) => {
    // Basic navigation logic based on type
    if (searchType.value === 1) {
        // User Profile
        router.push({
            path: '/profile',
            query: {
                username: item.username || item.contact?.username, 
                nickname: getNickname(item),
                headUrl: getHeadUrl(item), // Pass the resolved URL
                signature: stripHtml(item.signature || item.contact?.signature || '')
            }
        })
    } else if (searchType.value === 3) {
        // Video Playback (Type 3 only)
        try {
            playerTitle.value = stripHtml(getVideoTitle(item))
            const url = await resolveVideoUrl(item)
            playerUrl.value = url
        } catch (e) {
            alert('播放失败: ' + e.message)
            console.error('Video playback error:', e)
        }
    } else {
        // Type 2 (Live) - no playback for now
        console.log("Live item clicked (no playback):", item)
    }
}

const resolveVideoUrl = async (item) => {
    // Get video details via feed_profile API
    const objectId = item.objectId || item.id
    const nonceId = item.objectNonceId || item.nonceId
    
    if (!objectId) {
        throw new Error('缺少视频ID')
    }
    
    const res = await clientStore.remoteCall('api_call', {
        key: 'key:channels:feed_profile',
        body: { object_id: objectId, nonce_id: nonceId }
    })
    
    // Parse response structure (similar to UserProfile.vue)
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
    
    if (!media || !media.url) {
        throw new Error('无法获取视频地址')
    }
    
    let videoUrl = media.url + (media.urlToken || '')
    const decryptKey = media.decodeKey || ''
    
    // Select lowest quality spec if available (for faster loading)
    if (media.spec && media.spec.length > 0) {
        const lowestSpec = media.spec.reduce((prev, curr) => {
            return (curr.bitRate || 99999) < (prev.bitRate || 99999) ? curr : prev
        })
        if (lowestSpec.fileFormat) {
            videoUrl += `&X-snsvideoflag=${lowestSpec.fileFormat}`
        }
    }
    
    // Build proxy URL for decrypt playback
    let finalUrl = `/api/video/play?url=${encodeURIComponent(videoUrl)}`
    if (decryptKey) finalUrl += `&key=${decryptKey}`
    
    return finalUrl
}

const closePlayer = () => {
    playerUrl.value = ''
    playerTitle.value = ''
}

const onImgError = (e) => {
  e.target.src = placeholderImg
}

const stripHtml = (html) => {
    if (!html) return ''
    return html.replace(/<[^>]+>/g, '')
}

// 确保 URL 使用 HTTPS 协议
const ensureHttps = (url) => {
    if (!url || url === placeholderImg) return url
    return url.replace(/^http:\/\//i, 'https://')
}

// Helpers for template to cleaner access & robust fallbacks
const getHeadUrl = (item) => {
    // User JSON: item.contact.headUrl
    const url = item.headUrl || item.headImgUrl || item.contact?.headUrl || item.contact?.headImgUrl || placeholderImg
    return ensureHttps(url)
}

const getNickname = (item) => {
    // User JSON: item.contact.nickname (Clean) or item.highlightNickname (HTML)
    // Video JSON: item.objectDesc.nickname
    return stripHtml(item.nickname || item.contact?.nickname || item.objectDesc?.nickname || '未命名')
}

const getSignature = (item) => {
    return stripHtml(item.signature || item.contact?.signature || '')
}

const getVideoCover = (item) => {
    const url = item.objectDesc?.media?.[0]?.coverUrl || item.objectDesc?.media?.[0]?.url || placeholderImg
    return ensureHttps(url)
}

const getVideoTitle = (item) => {
    // Video title might be in description or highlightDescription
   return stripHtml(item.objectDesc?.description || item.description || '无标题视频')
}

const getLikeCount = (item) => {
    return item.likeCount || item.objectExtend?.favInfo?.fingerlikeFavCount || 0
}


// Subscription functions
const loadSubscriptions = async () => {
    try {
        const token = localStorage.getItem('token')
        if (!token) return
        
        const res = await fetch('/api/subscriptions', {
            headers: { 'Authorization': `Bearer ${token}` }
        })
        const data = await res.json()
        if (data.code === 0) {
            subscriptions.value = data.data || []
        }
    } catch (e) {
        console.error('Failed to load subscriptions:', e)
    }
}

const isSubscribed = (item) => {
    const username = item.username || item.contact?.username
    return subscriptions.value.some(sub => sub.wx_username === username)
}

const toggleSubscribe = async (item) => {
    subscribing.value = true
    try {
        const username = item.username || item.contact?.username
        const token = localStorage.getItem('token')
        
        if (isSubscribed(item)) {
            // Unsubscribe
            const subscription = subscriptions.value.find(sub => sub.wx_username === username)
            if (!subscription) return
            
            const res = await fetch(`/api/subscriptions/${subscription.id}`, {
                method: 'DELETE',
                headers: { 'Authorization': `Bearer ${token}` }
            })
            
            if (res.ok) {
                subscriptions.value = subscriptions.value.filter(sub => sub.id !== subscription.id)
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
                    wx_username: username,
                    wx_nickname: getNickname(item),
                    wx_head_url: getHeadUrl(item),
                    wx_signature: stripHtml(item.signature || item.contact?.signature || '')
                })
            })
            
            const data = await res.json()
            if (data.code === 0) {
                subscriptions.value.push(data.data)
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

const getLiveCover = (item) => {
    // Priority: LiveInfo Cover -> Contact Live Cover -> ObjectDesc Media (Type 9) -> Fallbacks
    if (item.liveInfo?.coverUrl) return item.liveInfo.coverUrl
    if (item.liveInfo?.liveCoverImgs?.length > 0) return item.liveInfo.liveCoverImgs[0].url
    
    // Check contact level live info
    if (item.contact?.liveInfo?.liveCoverImgs?.length > 0) return item.contact?.liveInfo?.liveCoverImgs[0].url
    if (item.contact?.liveCoverImgUrl) return item.contact.liveCoverImgUrl

    // Check media list (sometimes Live is Type 9 media)
    if (item.objectDesc?.media?.[0]?.coverUrl) return item.objectDesc.media[0].coverUrl

    return item.objectDesc?.liveInfo?.coverUrl || item.liveCoverImgUrl || placeholderImg
}

const getLiveTitle = (item) => {
    // Live title is often in objectDesc.description (stripped)
    // Fallback to nickname or specific live description fields
    const desc = item.objectDesc?.description || item.liveInfo?.description || item.objectDesc?.liveInfo?.description
    if (desc) return stripHtml(desc)
    
    return stripHtml(item.nickname || item.contact?.nickname || '直播中')
}

const getLiveViewerCount = (item) => {
    // "2.4万" or number
    // Check root liveInfo (User provided JSON) and objectDesc liveInfo (Fallback)
    const info = item.liveInfo || item.objectDesc?.liveInfo
    if (!info) return 0
    return info.liveSquareParticipantWording || info.participantCount || 0
}


</script>
