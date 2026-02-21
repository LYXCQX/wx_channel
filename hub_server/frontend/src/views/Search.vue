<template>
  <div class="h-full flex flex-col bg-bg relative overflow-hidden">
    <!-- Header / Search Area -->
    <!-- Header / Search Area -->
    <div class="px-3 md:px-4 pt-3 md:pt-6 pb-2 shrink-0 z-10 w-full flex justify-center">
       <div class="w-full max-w-5xl bg-surface-0/80 backdrop-blur-md rounded-2xl md:rounded-3xl shadow-lg border border-surface-100 p-3 md:p-6 flex flex-col gap-3 md:gap-5 transition-all hover:shadow-xl">
           <div class="flex items-center justify-between px-1 md:px-2">
               <h1 class="text-lg md:text-2xl font-serif font-bold text-text bg-gradient-to-r from-primary to-primary-600 bg-clip-text text-transparent">穿透搜索</h1>
               <div v-if="client" class="px-2 md:px-3 py-0.5 md:py-1 rounded-full bg-primary/10 text-primary text-[10px] md:text-xs font-bold flex items-center gap-1 md:gap-2 max-w-[45%] truncate">
                 <i class="pi pi-desktop text-[10px] md:text-xs shrink-0"></i>
                 <span class="truncate">{{ client.hostname }}</span>
               </div>
               <div v-else class="px-2 md:px-3 py-0.5 md:py-1 rounded-full bg-yellow-50 text-yellow-700 text-[10px] md:text-xs font-bold flex items-center gap-1 md:gap-2 border border-yellow-200 cursor-pointer hover:bg-yellow-100 transition-colors" @click="handleSearch(false)">
                 <i class="pi pi-bolt text-[10px] md:text-xs"></i>
                 <span>自动选择</span>
               </div>
           </div>

           <!-- Search Input -->
           <div class="relative group">
                <div class="absolute inset-y-0 left-0 pl-3 md:pl-5 flex items-center pointer-events-none">
                    <i class="pi pi-search text-surface-400 text-base md:text-xl group-focus-within:text-primary transition-colors"></i>
                </div>
                <input 
                    v-model="keyword"
                    type="text"
                    class="block w-full pl-9 md:pl-14 pr-16 md:pr-24 py-2.5 md:py-4 bg-surface-50 border-2 border-surface-100 rounded-xl md:rounded-2xl text-sm md:text-lg transition-all focus:bg-surface-0 focus:border-primary focus:ring-4 focus:ring-primary/10 outline-none placeholder:text-surface-400 shadow-inner hover:border-surface-300"
                    placeholder="搜索全网视频号内容..."
                    @keyup.enter="handleSearch(false)"
                />
                <div class="absolute inset-y-0 right-1.5 md:right-2 flex items-center">
                    <Button 
                        label="搜索" 
                        class="!px-3 md:!px-6 !py-1 md:!py-2 !rounded-lg md:!rounded-xl !font-bold shadow-md !text-xs md:!text-base" 
                        :loading="searching" 
                        @click="handleSearch(false)"
                    />
                </div>
           </div>

           <!-- Filter Tabs -->
           <div class="flex justify-center -mb-1 md:-mb-2">
               <div class="inline-flex bg-surface-100/50 p-1 md:p-1.5 rounded-full border border-surface-200/50 backdrop-blur-sm">
                    <button 
                        v-for="type in searchTypes" 
                        :key="type.value"
                        @click="searchType = type.value"
                        class="px-3 md:px-6 py-1.5 md:py-2 rounded-full text-xs md:text-sm font-bold transition-all flex items-center gap-1 md:gap-2 whitespace-nowrap"
                        :class="searchType === type.value ? 'bg-primary text-white shadow-lg shadow-primary/20 scale-105' : 'text-text-muted hover:text-text hover:bg-surface-200/50'"
                    >
                        <i :class="type.icon" class="text-[10px] md:text-xs"></i>
                        {{ type.label }}
                    </button>
               </div>
           </div>
       </div>
    </div>

    <!-- Results Area -->
    <div class="flex-1 overflow-y-auto p-3 md:p-6 custom-scrollbar w-full">
        <div class="w-full">
            <!-- Loading State -->
            <div v-if="searching && !results.length" class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-3 md:gap-6">
                <div v-for="i in 12" :key="i" class="bg-surface-0 rounded-2xl p-4 shimmer h-72">
                    <Skeleton height="100%" class="rounded-xl" />
                </div>
            </div>

            <!-- Results Grid -->
            <div v-else-if="results.length > 0" class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-3 md:gap-6 pb-12">
                <div 
                  v-for="(item, idx) in results" 
                  :key="idx" 
                  class="bg-surface-0 rounded-xl md:rounded-2xl overflow-hidden shadow-sm border border-surface-100 cursor-pointer transition-all hover:-translate-y-1 hover:shadow-xl hover:shadow-primary/5 hover:border-primary/30 group relative flex flex-col"
                  @click="openDetail(item)"
                >
                  <!-- 1. Video Card Style -->
                  <template v-if="searchType === 3">
                      <div class="relative aspect-[9/16] w-full bg-surface-100 overflow-hidden">
                         <img :src="getVideoCover(item)" class="w-full h-full object-cover transition-transform duration-700 group-hover:scale-105" @error="onImgError">
                         <div class="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent opacity-60"></div>
                         
                         <!-- Play Icon Overlay -->
                         <div class="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-all duration-300 backdrop-blur-[2px] bg-black/10">
                             <div class="bg-surface-0 text-primary p-4 rounded-full shadow-lg transform scale-90 group-hover:scale-100 transition-transform">
                                 <i class="pi pi-play text-2xl fill-current"></i>
                             </div>
                         </div>
                         
                         <!-- Duration/Stats Badges -->
                         <div class="absolute top-2 right-2 md:top-3 md:right-3 flex flex-col gap-1 items-end">
                            <Tag v-if="item.objectDesc?.media?.[0]?.duration" :value="formatDuration(item.objectDesc.media[0].duration)" severity="secondary" class="!bg-black/40 !text-white !backdrop-blur-md !border-none !rounded-md !text-[10px] md:!text-xs"></Tag>
                         </div>
                      </div>
                      
                      <div class="p-3 md:p-4 flex flex-col gap-1.5 md:gap-2 flex-1">
                         <h3 class="font-bold text-text text-xs md:text-sm line-clamp-2 leading-relaxed h-[2.2rem] md:h-[2.5rem]" v-html="getVideoTitle(item)"></h3>
                         
                         <div class="mt-auto flex items-center justify-between text-[10px] md:text-xs text-text-muted pt-2 md:pt-3 border-t border-surface-100">
                             <div class="flex items-center gap-1 md:gap-1.5 truncate max-w-[65%]">
                                <i class="pi pi-user text-[10px]"></i>
                                <span class="truncate">{{ getNickname(item) }}</span>
                             </div>
                             <div class="flex items-center gap-2 md:gap-3">
                                <span v-if="getLikeCount(item) > 0" class="flex items-center gap-1 font-medium">
                                     <i class="pi pi-heart text-[10px]"></i> {{ formatNumber(getLikeCount(item)) }}
                                </span>
                             </div>
                         </div>
                      </div>
                  </template>

                  <!-- 2. User Card Style -->
                  <template v-else-if="searchType === 1">
                      <div class="p-4 md:p-6 flex flex-col items-center text-center h-full gap-3 md:gap-4 relative">
                          <div class="absolute top-0 left-0 w-full h-16 md:h-24 bg-gradient-to-b from-primary/5 to-transparent"></div>
                          
                          <div class="w-16 h-16 md:w-20 md:h-20 rounded-full p-1 bg-surface-0 shadow-lg border-2 border-surface-50 group-hover:border-primary/20 group-hover:scale-105 transition-all overflow-hidden relative z-10 shrink-0">
                              <img :src="getHeadUrl(item)" class="w-full h-full rounded-full object-cover" @error="onImgError">
                          </div>
                          
                          <div class="w-full relative z-10">
                              <h3 class="font-bold text-sm md:text-lg text-text truncate group-hover:text-primary transition-colors">{{ getNickname(item) }}</h3>
                              <p class="text-[10px] md:text-xs text-text-muted line-clamp-2 mt-1 md:mt-2 px-1 md:px-2 min-h-[2em] md:min-h-[2.5em] leading-relaxed">{{ stripHtml(item.signature || item.contact?.signature || '暂无签名') }}</p>
                          </div>

                          <div class="mt-auto w-full pt-1 md:pt-2">
                              <Button 
                                  :label="isSubscribed(item) ? '已订阅' : '订阅'" 
                                  :icon="subscribing ? 'pi pi-spin pi-spinner' : (isSubscribed(item) ? 'pi pi-check' : 'pi pi-plus')"
                                  :severity="isSubscribed(item) ? 'secondary' : 'primary'"
                                  rounded
                                  size="small"
                                  class="w-full font-bold shadow-sm !text-xs md:!text-sm !py-1.5 md:!py-2"
                                  :disabled="subscribing"
                                  @click.stop="toggleSubscribe(item)" 
                              />
                          </div>
                      </div>
                  </template>

                  <!-- 3. Live Card Style -->
                  <template v-else-if="searchType === 2">
                       <div class="relative aspect-[3/4] w-full bg-surface-900">
                           <img :src="getLiveCover(item)" class="w-full h-full object-cover opacity-80 group-hover:opacity-60 transition-opacity" @error="onImgError">
                           <div class="absolute top-2 left-2 md:top-3 md:left-3">
                               <Tag value="LIVE" severity="danger" icon="pi pi-circle-fill" class="animate-pulse shadow-lg !text-[10px] md:!text-xs !px-2"></Tag>
                           </div>
                           
                           <div class="absolute bottom-0 left-0 right-0 p-3 md:p-4 bg-gradient-to-t from-black/90 via-black/50 to-transparent">
                               <h3 class="text-white font-bold text-xs md:text-sm line-clamp-2 mb-1 md:mb-2 leading-relaxed">{{ getLiveTitle(item) }}</h3>
                               <div class="flex items-center justify-between text-white/80 text-[10px] md:text-xs">
                                   <div class="flex items-center gap-1 md:gap-1.5 truncate">
                                       <img :src="getHeadUrl(item)" class="w-4 h-4 md:w-5 md:h-5 rounded-full border border-white/20">
                                       <span>{{ getNickname(item) }}</span>
                                   </div>
                                    <span v-if="getLiveViewerCount(item)" class="flex items-center gap-1 bg-black/40 px-1.5 py-0.5 md:px-2 md:py-1 rounded-lg backdrop-blur-md border border-white/10">
                                        <i class="pi pi-eye"></i>
                                        {{ getLiveViewerCount(item) }}
                                    </span>
                               </div>
                           </div>
                       </div>
                  </template>
                </div>
            </div>

            <!-- Empty States -->
            <div v-else-if="searched" class="flex flex-col items-center justify-center p-20 text-center animate-fade-in">
                 <div class="w-32 h-32 bg-surface-50 rounded-full flex items-center justify-center mb-6 shadow-neu-pressed">
                     <i class="pi pi-search text-5xl text-surface-300"></i>
                 </div>
                 <h3 class="text-xl font-bold text-text mb-2">未找到相关结果</h3>
                 <p class="text-text-muted max-w-xs mx-auto">尝试更换关键词或切换搜索类型</p>
            </div>
            
            <div v-else class="flex flex-col items-center justify-center p-32 text-center opacity-40 animate-fade-in">
                 <div class="relative mb-8">
                     <div class="absolute inset-0 bg-primary/20 blur-3xl rounded-full"></div>
                     <i class="pi pi-compass text-8xl text-surface-200 relative z-10 transition-transform hover:rotate-45 duration-700"></i>
                 </div>
                 <p class="text-2xl font-serif text-text-muted">探索全网精彩内容</p>
            </div>
        
            <!-- Load More -->
             <div v-if="hasMoreSearch && searchType !== 2 && results.length > 0" class="text-center mt-8 mb-16">
                <Button 
                    label="加载更多内容" 
                    icon="pi pi-angle-down" 
                    rounded 
                    outlined 
                    severity="secondary" 
                    class="px-8 !border-surface-200 hover:!border-primary hover:!text-primary"
                    @click="handleSearch(true)" 
                    :loading="searching"
                />
            </div>
        </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useClientStore } from '../store/client'
import { useRouter } from 'vue-router'

// PrimeVue
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import Skeleton from 'primevue/skeleton'

const clientStore = useClientStore()
const router = useRouter()
const client = computed(() => clientStore.currentClient)

const keyword = ref('')
const searchType = ref(3) // Default to Video
const searching = ref(false)
const results = ref([])
const searched = ref(false)

const searchTypes = [
    { label: '找视频', value: 3, icon: 'pi pi-video' },
    { label: '找用户', value: 1, icon: 'pi pi-user' },
    { label: '找直播', value: 2, icon: 'pi pi-users' }
]

const placeholderImg = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100"%3E%3Crect fill="%23f1f5f9" width="100" height="100"/%3E%3Ctext x="50" y="50" font-family="sans-serif" font-size="14" fill="%2394a3b8" text-anchor="middle" dominant-baseline="middle"%3E暂无图片%3C/text%3E%3C/svg%3E'

const lastSearchBuffer = ref('')
const hasMoreSearch = ref(false)
const searchSessionId = ref('')

// Subscription state
const subscriptions = ref([])
const subscribing = ref(false)

onMounted(() => {
    loadSubscriptions()
})

watch(searchType, () => {
    if (keyword.value) {
        handleSearch(false)
    } else {
        results.value = []
        hasMoreSearch.value = false
        searched.value = false
    }
})

const handleSearch = async (loadMore = false) => {
  if (!keyword.value) return
  
  if (!client.value) {
       await clientStore.fetchClients() 
       if (!clientStore.currentClient && clientStore.clients.length > 0) {
           clientStore.setCurrentClient(clientStore.clients[0].id)
       }
  }
  
  searching.value = true
  if (!loadMore) {
      results.value = []
      lastSearchBuffer.value = ''
      hasMoreSearch.value = false
      searchSessionId.value = String(new Date().valueOf())
      searched.value = true
  }
  
  try {
    const res = await clientStore.remoteCall('api_call', {
      key: 'key:channels:contact_list',
      body: { 
          keyword: keyword.value,
          type: searchType.value,
          next_marker: loadMore ? lastSearchBuffer.value : '',
          request_id: searchSessionId.value
      }
    })
    
    let data = res.data;
    if (data && data.code === 0 && data.data) {
        data = data.data;
    }
    if ((!data || data.list === undefined) && res.data?.data?.list) {
         data = res.data.data;
    }

    const list = data?.list || [];
    lastSearchBuffer.value = data?.next_marker || '';
    hasMoreSearch.value = (searchType.value === 2) ? false : !!data?.has_more;

    const newItems = list; 
    
    if (loadMore) {
        results.value = [...results.value, ...newItems]
    } else {
        results.value = newItems
    }

  } catch (err) {
    console.error(err)
  } finally {
    searching.value = false
  }
}

const openDetail = async (item) => {
    if (searchType.value === 1) {
        router.push({
            path: '/channel/' + (item.username || item.contact?.username),
            query: {
                username: item.username || item.contact?.username, 
                nickname: getNickname(item),
                headUrl: getHeadUrl(item),
                signature: stripHtml(item.signature || item.contact?.signature || '')
            }
        })
    } else if (searchType.value === 3) {
        const objectId = item.objectId || item.id
        const nonceId = item.objectNonceId || item.nonceId
        
        if (objectId) {
            router.push({
                name: 'VideoDetail',
                params: { id: objectId },
                query: { nonceId: nonceId }
            })
        }
    }
}

const onImgError = (e) => {
  e.target.src = placeholderImg
}

const stripHtml = (html) => {
    if (!html) return ''
    return html.replace(/<[^>]+>/g, '')
}

const ensureHttps = (url) => {
    if (!url || url === placeholderImg) return url
        .replace(/^http:\/\//i, 'https://')
        .replace("http://wx.qlogo.cn", "https://wx.qlogo.cn")
    return url.replace(/^http:\/\//i, 'https://')
}

// Helpers
const getHeadUrl = (item) => {
    const url = item.headUrl || item.headImgUrl || item.contact?.headUrl || item.contact?.headImgUrl || placeholderImg
    return ensureHttps(url)
}

const getNickname = (item) => {
    return stripHtml(item.nickname || item.contact?.nickname || item.objectDesc?.nickname || '未命名')
}

const getVideoCover = (item) => {
    const url = item.objectDesc?.media?.[0]?.coverUrl || item.objectDesc?.media?.[0]?.url || placeholderImg
    return ensureHttps(url)
}

const getVideoTitle = (item) => {
   return stripHtml(item.objectDesc?.description || item.description || '无标题视频')
}

const getLikeCount = (item) => {
    return item.likeCount || item.objectExtend?.favInfo?.fingerlikeFavCount || 0
}

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
        alert('操作失败: ' + e.message)
    } finally {
        subscribing.value = false
    }
}

const getLiveCover = (item) => {
    if (item.liveInfo?.coverUrl) return item.liveInfo.coverUrl
    if (item.liveInfo?.liveCoverImgs?.length > 0) return item.liveInfo.liveCoverImgs[0].url
    if (item.contact?.liveInfo?.liveCoverImgs?.length > 0) return item.contact?.liveInfo?.liveCoverImgs[0].url
    if (item.contact?.liveCoverImgUrl) return item.contact.liveCoverImgUrl
    if (item.objectDesc?.media?.[0]?.coverUrl) return item.objectDesc.media[0].coverUrl
    return item.objectDesc?.liveInfo?.coverUrl || item.liveCoverImgUrl || placeholderImg
}

const getLiveTitle = (item) => {
    const desc = item.objectDesc?.description || item.liveInfo?.description || item.objectDesc?.liveInfo?.description
    if (desc) return stripHtml(desc)
    return stripHtml(item.nickname || item.contact?.nickname || '直播中')
}

const getLiveViewerCount = (item) => {
    const info = item.liveInfo || item.objectDesc?.liveInfo
    if (!info) return 0
    return info.liveSquareParticipantWording || info.participantCount || 0
}

const formatNumber = (num) => {
    if (num > 10000) return (num / 10000).toFixed(1) + 'w'
    if (num > 1000) return (num / 1000).toFixed(1) + 'k'
    return num
}

const formatDuration = (seconds) => {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return `${m}:${s.toString().padStart(2, '0')}`
}
</script>

<style scoped>
.shimmer {
    background: linear-gradient(90deg, #f0f2f5 25%, #e1e4e8 50%, #f0f2f5 75%);
    background-size: 200% 100%;
    animation: shimmer 1.5s infinite;
}
@keyframes shimmer {
    0% { background-position: 200% 0; }
    100% { background-position: -200% 0; }
}
.animate-fade-in {
    animation: fadeIn 0.5s ease-out;
}
@keyframes fadeIn {
    from { opacity: 0; transform: translateY(10px); }
    to { opacity: 1; transform: translateY(0); }
}
/* Custom Scrollbar */
.custom-scrollbar::-webkit-scrollbar {
    width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-track {
    background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
    background-color: rgba(156, 163, 175, 0.5);
    border-radius: 20px;
}
</style>
