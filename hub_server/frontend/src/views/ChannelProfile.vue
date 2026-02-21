<template>
    <div class="min-h-screen bg-bg p-4 lg:p-12 font-sans text-text">
    <header class="flex flex-col md:flex-row justify-between items-start md:items-center mb-6 lg:mb-12 gap-4">
      <div class="flex flex-col md:flex-row items-start md:items-center gap-4 lg:gap-6 flex-1 w-full">
          <div class="flex items-center gap-2 w-full md:w-auto">
            <Button icon="pi pi-arrow-left" rounded variant="text" aria-label="Back" @click="goBack" class="!w-10 !h-10 lg:!w-12 lg:!h-12 !text-text hover:!bg-surface-100 shrink-0" />
            
            <!-- Mobile Device Badge (Visible on mobile, next to back button) -->
            <div v-if="client" class="md:hidden ml-auto px-2 py-1 rounded-lg bg-surface-0 shadow-sm border border-surface-200 text-primary font-medium flex items-center gap-1.5 overflow-hidden max-w-[150px]">
                <span class="text-[10px] uppercase tracking-wider text-text-muted whitespace-nowrap">Connected</span>
                <strong class="text-xs truncate">{{ client.hostname }}</strong>
            </div>
             <div v-else class="md:hidden ml-auto px-2 py-1 rounded-lg bg-yellow-50 border border-yellow-200 text-yellow-700 text-xs flex items-center gap-1.5">
                <i class="pi pi-bolt text-[10px]"></i>
                <span>自动选择</span>
            </div>
          </div>
          
          <!-- Loading Author Info -->
          <div class="flex items-center gap-3 lg:gap-4 flex-1 w-full" v-if="loadingVideos && !author.nickname">
             <Skeleton shape="circle" size="4rem" class="shadow-neu-sm lg:w-20 lg:h-20"></Skeleton>
             <div class="flex-1">
                <Skeleton width="50%" height="1.5rem" class="mb-2 lg:h-2rem lg:w-12rem"></Skeleton>
                <Skeleton width="70%" height="1rem" class="mb-2 lg:w-18rem"></Skeleton>
                <div class="flex gap-2 mt-2">
                     <Skeleton width="4rem" height="1.2rem" class="lg:w-6rem lg:h-1.5rem"></Skeleton>
                </div>
             </div>
          </div>

          <!-- Author Info -->
          <div class="flex items-center gap-3 lg:gap-4 flex-1 w-full" v-else>
             <div class="w-16 h-16 lg:w-20 lg:h-20 rounded-full shadow-neu-sm p-0.5 lg:p-1 bg-surface-0 overflow-hidden shrink-0">
                <img :src="author.headUrl || placeholderImg" class="w-full h-full rounded-full object-cover" @error="onImgError">
             </div>
             <div class="flex-1 min-w-0">
                <div class="flex flex-col md:flex-row md:items-center gap-1 md:gap-4">
                    <h2 class="font-serif font-bold text-xl lg:text-2xl text-text truncate">{{ author.nickname }}</h2>
                </div>
                <p class="text-text-muted text-xs lg:text-sm max-w-md line-clamp-2 mt-0.5 lg:mt-1">{{ author.signature || '暂无签名' }}</p>
                <div class="flex items-center gap-4 mt-1 lg:mt-2">
                    <Tag :value="`${videos.length} 个视频`" icon="pi pi-video" severity="secondary" rounded class="!text-[10px] lg:!text-xs scale-90 origin-left"></Tag>
                </div>
             </div>
             <!-- Subscribe Button (Desktop: right side, Mobile: handled below or separate?) -->
             <!-- Let's keep it here but styled flexible -->
             <Button 
                 :label="subscribing ? '处理中...' : (isSubscribed ? '已订阅' : '订阅')" 
                 :icon="isSubscribed ? 'pi pi-check' : 'pi pi-plus'"
                 :disabled="subscribing"
                 :severity="isSubscribed ? 'secondary' : 'primary'"
                 rounded
                 size="small"
                 class="whitespace-nowrap md:ml-4 !text-xs lg:!text-base !px-3 lg:!px-4"
                 @click="toggleSubscribe" 
             />
          </div>
      </div>
      
      <!-- Desktop Device Badge -->
      <div v-if="client" class="hidden md:flex px-4 py-2 rounded-xl bg-surface-0 shadow-sm border border-surface-200 text-primary font-medium items-center gap-2">
        <span class="text-xs uppercase tracking-wider text-text-muted">Connected to</span>
        <strong>{{ client.hostname }}</strong>
      </div>
      <div v-else class="hidden md:flex px-4 py-2 rounded-xl bg-yellow-50 border border-yellow-200 text-yellow-700 text-sm items-center gap-2">
        <i class="pi pi-bolt"></i>
        <span>自动选择设备</span>
      </div>
    </header>

    <div class="max-w-[1600px] mx-auto">
        <!-- Grid Skeleton Loading -->
        <div v-if="loadingVideos && videos.length === 0" class="grid grid-cols-2 md:grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-3 lg:gap-6">
            <div v-for="i in 8" :key="i" class="bg-surface-0 rounded-xl lg:rounded-2xl overflow-hidden shadow-sm aspect-[9/16] w-full p-2 flex flex-col gap-2">
                <Skeleton width="100%" height="100%" class="rounded-lg"></Skeleton>
            </div>
        </div>

        <!-- Video Grid using DataView -->
        <DataView v-else-if="videos.length > 0" :value="videos" layout="grid">
            <template #grid="{ items }">
                <div class="grid grid-cols-2 md:grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-3 lg:gap-6">
                    <div 
                      v-for="video in items" 
                      :key="video.id" 
                      class="bg-surface-0 rounded-xl lg:rounded-2xl overflow-hidden shadow-sm border border-surface-100 cursor-pointer transition-all hover:-translate-y-1 hover:shadow-md hover:border-primary/30 group relative aspect-[9/16] w-full"
                      @click="playVideo(video)"
                    >
                        <!-- Cover Image -->
                        <img :src="video.coverUrl" class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105" @error="onImgError">
                        
                        <!-- Gradient Overlay -->
                        <div class="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent opacity-80"></div>

                        <!-- Play Icon Overlay -->
                        <div class="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity duration-300">
                            <div class="bg-primary/90 text-white p-2 lg:p-3 rounded-full backdrop-blur-sm shadow-xl transform scale-75 group-hover:scale-100 transition-transform">
                                <i class="pi pi-play text-xl lg:text-2xl"></i>
                            </div>
                        </div>

                        <!-- Duration Badge -->
                        <div class="absolute top-1.5 right-1.5 lg:top-2 lg:right-2 bg-black/60 backdrop-blur-sm text-white text-[10px] px-1.5 py-0.5 rounded-md font-medium flex items-center gap-1">
                           <i class="pi pi-clock text-[10px]"></i>
                           {{ video.duration }}
                        </div>

                        <!-- Bottom Info -->
                        <div class="absolute bottom-0 left-0 right-0 p-2 lg:p-4 text-white">
                            <div class="font-bold text-xs lg:text-sm line-clamp-2 mb-1 lg:mb-2 leading-snug">{{ video.title || '无标题视频' }}</div>
                            <div class="flex items-center justify-between text-[10px] lg:text-xs text-white/70">
                                 <div class="flex items-center gap-1" v-if="video.width && video.height">
                                    <i class="pi pi-desktop text-[8px] lg:text-[10px]"></i>
                                    <span>{{ video.width }}x{{ video.height }}</span>
                                 </div>
                                 <div class="ml-auto">
                                     {{ formatTime(video.createTime * 1000) }}
                                 </div>
                            </div>
                        </div>
                    </div>
                </div>
            </template>
        </DataView>
          
        <!-- Load More Button -->
        <div v-if="hasMoreVideos" class="text-center mt-8 lg:mt-12 mb-8 lg:mb-12">
            <Button 
                :label="loadingVideos ? '加载中...' : '加载更多视频'" 
                :icon="loadingVideos ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'"
                rounded 
                outlined
                size="small"
                severity="secondary" 
                @click="fetchVideos(true)"
                :disabled="loadingVideos"
            />
        </div>
        
        <!-- Empty State -->
        <div v-if="!loadingVideos && videos.length === 0" class="text-center p-8 lg:p-16 text-text-muted bg-surface-0 rounded-2xl lg:rounded-[2rem] shadow-sm">
            <i class="pi pi-video text-4xl lg:text-6xl text-surface-200 mb-4 block"></i>
            <p class="text-base lg:text-lg font-medium mb-2">暂无视频动态</p>
            <p class="text-xs lg:text-sm">该用户还没有发布任何视频</p>
        </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useClientStore } from '../store/client'
import { formatTime, formatDuration } from '../utils/format'

// PrimeVue Components
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import DataView from 'primevue/dataview'
import Skeleton from 'primevue/skeleton'

const router = useRouter()
const route = useRoute()
const clientStore = useClientStore()
const client = computed(() => clientStore.currentClient)

const placeholderImg = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100"%3E%3Crect fill="%23f1f5f9" width="100" height="100"/%3E%3Ctext x="50" y="50" font-family="sans-serif" font-size="14" fill="%2394a3b8" text-anchor="middle" dominant-baseline="middle"%3E暂无图片%3C/text%3E%3C/svg%3E'

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
const hasMoreVideos = ref(false)
const lastVideoMarker = ref('')

// Subscription state
const isSubscribed = ref(false)
const subscribing = ref(false)
const subscriptionId = ref(null)

onMounted(() => {
    const q = route.query
    if (route.params.username) {
        author.value = {
            username: route.params.username,
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
    router.go(-1)
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
    
    // Robust extraction logic from UserProfile.vue
    let objects = []
    const findObjects = (obj) => {
        if (!obj) return null
        if (Array.isArray(obj.object)) return obj.object
        if (Array.isArray(obj.list)) return obj.list
        return null
    }
    
    // 1. Try res.data
    if (res.data) {
        objects = findObjects(res.data)
        const payload = res.data.payload || {}
        if (res.data.continueFlag || payload.lastBuffer) {
             lastVideoMarker.value = payload.lastBuffer || res.data.lastBuffer || ''
             hasMoreVideos.value = !!lastVideoMarker.value
        }

        // 2. Try res.data.data
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
    console.error('Failed to fetch videos:', err)
  } finally {
    loadingVideos.value = false
  }
}

const playVideo = (video) => {
    if (video.id) {
        router.push({
            name: 'VideoDetail',
            params: { id: video.id },
            query: { nonceId: video.nonceId }
        })
    }
}

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

const onImgError = (e) => {
  e.target.src = placeholderImg
}
</script>
