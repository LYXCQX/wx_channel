<template>
    <div class="min-h-screen bg-bg p-4 lg:p-12 font-sans text-text">
    <header class="flex flex-col md:flex-row justify-end items-stretch md:items-center mb-6 lg:mb-12 gap-4">
      <div class="flex flex-col md:flex-row gap-3 lg:gap-4 w-full md:w-auto">
          <Button 
              label="去搜索添加" 
              icon="pi pi-search"
              outlined
              rounded
              class="w-full md:w-auto text-sm lg:text-base"
              @click="$router.push('/search')"
          />
          <Button 
              :label="updatingAll ? '更新中...' : '一键更新全部'" 
              :icon="updatingAll ? 'pi pi-spin pi-spinner' : 'pi pi-sync'"
              :disabled="updatingAll || subscriptions.length === 0"
              rounded
              class="w-full md:w-auto text-sm lg:text-base"
              @click="updateAllSubscriptions" 
          />
      </div>
    </header>

    <!-- Loading State -->
    <div v-if="loading && subscriptions.length === 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 lg:gap-6">
        <div v-for="i in 6" :key="i" class="bg-surface-0 rounded-2xl p-4 lg:p-6 shadow-sm border border-surface-100 flex flex-col gap-4">
             <div class="flex items-center gap-4">
                <Skeleton shape="circle" size="3rem" class="lg:w-16 lg:h-16"></Skeleton>
                <div class="flex-1">
                    <Skeleton width="60%" height="1.5rem" class="mb-2"></Skeleton>
                    <Skeleton width="40%" height="1rem"></Skeleton>
                </div>
             </div>
             <Skeleton width="100%" height="2rem" class="rounded-lg"></Skeleton>
             <div class="flex gap-2">
                 <Skeleton class="flex-1 h-10 rounded-xl"></Skeleton>
                 <Skeleton class="flex-1 h-10 rounded-xl"></Skeleton>
             </div>
        </div>
    </div>

    <!-- Empty State -->
    <div v-else-if="subscriptions.length === 0" class="text-center p-8 lg:p-16 bg-surface-0 rounded-[2rem] shadow-sm">
       <i class="pi pi-users text-4xl lg:text-6xl text-surface-200 mb-4 block"></i>
       <p class="text-text-muted text-base lg:text-lg mb-6">暂无订阅</p>
       <Button label="去搜索用户" icon="pi pi-search" rounded @click="$router.push('/search')" />
    </div>

    <!-- Subscription List -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 lg:gap-6">
      <div 
          v-for="sub in subscriptions" 
          :key="sub.id"
          class="bg-surface-0 rounded-2xl p-4 lg:p-6 shadow-sm border border-surface-100 hover:-translate-y-1 hover:shadow-md transition-all group relative overflow-hidden">
        
        <!-- User Info -->
        <div class="flex items-center gap-3 lg:gap-4 mb-3 lg:mb-4">
          <div class="w-12 h-12 lg:w-16 lg:h-16 rounded-full bg-surface-50 shadow-neu-sm p-0.5 lg:p-1 cursor-pointer shrink-0" @click="viewVideos(sub)">
            <img :src="sub.wx_head_url || placeholderImg" class="w-full h-full rounded-full object-cover" @error="onImgError">
          </div>
          <div class="flex-1 min-w-0">
            <h3 class="font-bold text-base lg:text-lg text-text mb-0.5 lg:mb-1 truncate cursor-pointer hover:text-primary transition-colors" @click="viewVideos(sub)">{{ sub.wx_nickname }}</h3>
            <p class="text-[10px] lg:text-xs text-text-muted line-clamp-2 min-h-[2.5em]">{{ sub.wx_signature || '暂无签名' }}</p>
          </div>
        </div>

        <!-- Stats -->
        <div class="flex items-center justify-between mb-4 lg:mb-6 px-3 py-2 lg:px-4 lg:py-3 bg-surface-50 rounded-xl">
          <div class="flex flex-col">
              <span class="text-[10px] lg:text-xs text-text-muted uppercase tracking-wider">视频数</span>
              <span class="font-bold text-base lg:text-lg text-primary">{{ sub.video_count }}</span>
          </div>
          <div class="flex flex-col text-right">
              <span class="text-[10px] lg:text-xs text-text-muted uppercase tracking-wider">上次更新</span>
              <span class="font-medium text-xs lg:text-sm text-text">{{ formatDate(sub.last_fetched_at) }}</span>
          </div>
        </div>

        <!-- Actions -->
        <div class="grid grid-cols-2 gap-2 lg:gap-3">
          <Button 
              label="查看" 
              icon="pi pi-eye" 
              outlined 
              size="small"
              class="w-full !text-xs lg:!text-sm"
              @click="viewVideos(sub)" 
          />
          <Button 
              :label="updating[sub.id] ? '更新中' : '更新'" 
              :icon="updating[sub.id] ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'"
              :disabled="updating[sub.id]"
              size="small"
              :severity="updating[sub.id] ? 'secondary' : 'primary'"
              class="w-full !text-xs lg:!text-sm"
              @click="fetchVideos(sub)" 
          />
          <Button 
              label="取消订阅" 
              icon="pi pi-trash" 
              severity="danger" 
              variant="text" 
              size="small"
              class="col-span-2 !py-1 lg:!py-2 opacity-60 hover:opacity-100 !text-xs lg:!text-sm h-8 lg:h-auto"
              @click="unsubscribe(sub)" 
          />
        </div>
      </div>
    </div>
    <ConfirmDialog />
    <Toast />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import axios from 'axios'

const router = useRouter()
const toast = useToast()
const confirm = useConfirm()

const subscriptions = ref([])
const loading = ref(false)
const updatingAll = ref(false)
const updating = ref({})

// 使用 data URI 避免外部请求和混合内容问题
const placeholderImg = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100"%3E%3Crect fill="%23f1f5f9" width="100" height="100"/%3E%3Ctext x="50" y="50" font-family="sans-serif" font-size="14" fill="%2394a3b8" text-anchor="middle" dominant-baseline="middle"%3E暂无图片%3C/text%3E%3C/svg%3E'

onMounted(() => {
  loadSubscriptions()
})

const loadSubscriptions = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/subscriptions')
    if (res.data.code === 0) {
      subscriptions.value = res.data.data || []
    }
  } catch (e) {
    console.error('Failed to load subscriptions:', e)
    toast.add({ severity: 'error', summary: 'Error', detail: '加载订阅失败', life: 3000 })
  } finally {
    loading.value = false
  }
}

const fetchVideos = async (sub) => {
  updating.value[sub.id] = true
  try {
    const res = await axios.post(`/api/subscriptions/${sub.id}/fetch`)
    if (res.data.code === 0) {
      toast.add({ severity: 'success', summary: 'Success', detail: `更新成功: ${res.data.data.new_videos} 个新视频`, life: 3000 })
      loadSubscriptions() // Reload to update counts
    } else {
      toast.add({ severity: 'error', summary: 'Error', detail: `更新失败: ${res.data.message || '未知错误'}`, life: 3000 })
    }
  } catch (e) {
    console.error('Failed to fetch videos:', e)
    toast.add({ severity: 'error', summary: 'Error', detail: '更新失败: ' + e.message, life: 3000 })
  } finally {
    updating.value[sub.id] = false
  }
}

const updateAllSubscriptions = async () => {
    confirm.require({
        message: `确定要更新所有 ${subscriptions.value.length} 个订阅吗？`,
        header: '更新确认',
        icon: 'pi pi-sync',
        accept: async () => {
            updatingAll.value = true
            let totalNew = 0
            let successCount = 0
            let failedList = []
            
            for (const sub of subscriptions.value) {
                try {
                    // Update individual subscription status for UI feedback
                    updating.value[sub.id] = true
                    const res = await axios.post(`/api/subscriptions/${sub.id}/fetch`)
                    if (res.data.code === 0) {
                        totalNew += res.data.data.new_videos
                        successCount++
                    } else {
                        failedList.push(`${sub.wx_nickname}: ${res.data.message}`)
                    }
                } catch (e) {
                    console.error(`Failed to update ${sub.wx_nickname}:`, e)
                    failedList.push(`${sub.wx_nickname}: ${e.message}`)
                } finally {
                    updating.value[sub.id] = false
                }
            }
            
            updatingAll.value = false
            loadSubscriptions()
            
            if (failedList.length > 0) {
                toast.add({ 
                    severity: 'warn', 
                    summary: '部分更新完成', 
                    detail: `成功: ${successCount}，失败: ${failedList.length}。新视频: ${totalNew}`, 
                    life: 5000 
                })
            } else {
                toast.add({ 
                    severity: 'success', 
                    summary: '全部更新完成', 
                    detail: `成功更新 ${successCount} 个订阅，发现 ${totalNew} 个新视频`, 
                    life: 3000 
                })
            }
        }
    })
}

const viewVideos = (sub) => {
  router.push({
    name: 'SubscriptionVideos',
    params: { id: sub.id },
    query: {
      nickname: sub.wx_nickname,
      headUrl: sub.wx_head_url
    }
  })
}

const unsubscribe = (sub) => {
    confirm.require({
        message: `确定要取消订阅 ${sub.wx_nickname} 吗？`,
        header: '取消订阅',
        icon: 'pi pi-trash',
        acceptClass: 'p-button-danger',
        accept: async () => {
            try {
                await axios.delete(`/api/subscriptions/${sub.id}`)
                subscriptions.value = subscriptions.value.filter(s => s.id !== sub.id)
                toast.add({ severity: 'success', summary: 'Unbound', detail: '已取消订阅', life: 2000 })
            } catch (e) {
                console.error('Failed to unsubscribe:', e)
                toast.add({ severity: 'error', summary: 'Error', detail: '操作失败', life: 3000 })
            }
        }
    })
}

const formatDate = (dateStr) => {
  if (!dateStr || dateStr === '0001-01-01T00:00:00Z') return '未更新'
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now - date
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)
  
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`
  return date.toLocaleDateString('zh-CN')
}

const onImgError = (e) => {
  e.target.src = placeholderImg
}
</script>
