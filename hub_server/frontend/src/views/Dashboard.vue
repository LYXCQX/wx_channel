```
<template>
  <div class="min-h-screen bg-bg p-4 lg:p-12 font-sans text-text">
    <header class="flex justify-end items-center mb-6 lg:mb-12">
      <div class="flex gap-4">
        <Button 
            :label="clientStore.loading ? '刷新中...' : '刷新状态'" 
            icon="pi pi-refresh" 
            :loading="clientStore.loading"
            rounded
            size="small"
            class="!text-sm lg:!text-base"
            @click="clientStore.fetchClients"
        />
      </div>
    </header>

    <!-- Stats Cards -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3 lg:gap-6 mb-6 lg:mb-12">
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-6 shadow-sm border border-surface-100 flex items-center justify-between">
            <div>
                <p class="text-text-muted text-xs lg:text-sm font-medium uppercase tracking-wider mb-1">总设备数</p>
                <div class="text-2xl lg:text-3xl font-bold text-text">{{ clientStore.clients.length }}</div>
            </div>
            <div class="w-10 h-10 lg:w-12 lg:h-12 rounded-full bg-blue-50 text-blue-500 flex items-center justify-center">
                <i class="pi pi-server text-lg lg:text-xl"></i>
            </div>
        </div>
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-6 shadow-sm border border-surface-100 flex items-center justify-between">
            <div>
                <p class="text-text-muted text-xs lg:text-sm font-medium uppercase tracking-wider mb-1">在线</p>
                <div class="text-2xl lg:text-3xl font-bold text-green-500">{{ onlineCount }}</div>
            </div>
            <div class="w-10 h-10 lg:w-12 lg:h-12 rounded-full bg-green-50 text-green-500 flex items-center justify-center">
                <i class="pi pi-check-circle text-lg lg:text-xl"></i>
            </div>
        </div>
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-6 shadow-sm border border-surface-100 flex items-center justify-between">
            <div>
                <p class="text-text-muted text-xs lg:text-sm font-medium uppercase tracking-wider mb-1">离线</p>
                <div class="text-2xl lg:text-3xl font-bold text-text-muted">{{ offlineCount }}</div>
            </div>
            <div class="w-10 h-10 lg:w-12 lg:h-12 rounded-full flex items-center justify-center transition-colors bg-surface-100 text-text-muted">
                <i class="pi pi-power-off text-lg lg:text-xl"></i>
            </div>
        </div>
    </div>

    <!-- Loading State -->
    <div v-if="clientStore.loading && !clientStore.clients.length" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3 gap-4 lg:gap-8">
        <div v-for="i in 3" :key="i" class="bg-surface-0 rounded-[2rem] p-6 lg:p-8 shadow-sm border border-surface-100 h-64 flex flex-col justify-between">
            <div class="flex justify-between">
                <Skeleton shape="circle" size="3.5rem"></Skeleton>
                <div class="flex flex-col items-end gap-2">
                     <Skeleton width="4rem" height="1.5rem"></Skeleton>
                     <Skeleton width="2rem" height="1rem"></Skeleton>
                </div>
            </div>
            <div class="space-y-3">
                 <Skeleton width="100%" height="1rem"></Skeleton>
                 <Skeleton width="80%" height="1rem"></Skeleton>
            </div>
            <div class="flex gap-4 mt-4">
                 <Skeleton class="flex-1 h-10 rounded-xl"></Skeleton>
                 <Skeleton class="flex-1 h-10 rounded-xl"></Skeleton>
            </div>
        </div>
    </div>

    <!-- Empty State -->
    <div v-else-if="clientStore.clients.length === 0" class="flex flex-col items-center justify-center py-12 lg:py-20 bg-surface-0 rounded-3xl shadow-sm border border-surface-100">
        <i class="pi pi-desktop text-4xl lg:text-6xl text-surface-200 mb-4 lg:mb-6"></i>
        <h3 class="text-xl lg:text-2xl font-bold text-text mb-2">暂无在线终端</h3>
        <p class="text-sm lg:text-base text-text-muted max-w-md text-center mb-4 lg:mb-6">请在目标机器上启动客户端应用程序并配置 Hub URL。</p>
        <Button label="刷新列表" icon="pi pi-refresh" rounded outlined @click="clientStore.fetchClients" />
    </div>

    <!-- Client Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3 gap-4 lg:gap-8">
        <div 
          v-for="client in clientStore.clients" 
          :key="client.id" 
          class="bg-surface-0 rounded-2xl lg:rounded-[2rem] p-4 lg:p-8 shadow-sm border border-surface-100 transition-all hover:-translate-y-1 hover:shadow-card flex flex-col"
        >
          <!-- Header -->
          <div class="flex items-start justify-between mb-4 lg:mb-6">
             <div class="flex items-center gap-3 lg:gap-4 overflow-hidden">
                 <div class="w-12 h-12 lg:w-14 lg:h-14 rounded-xl lg:rounded-2xl flex items-center justify-center transition-colors shrink-0"
                      :class="client.status === 'online' ? 'bg-primary-50 text-primary animate-pulse' : 'bg-surface-100 text-surface-400'">
                     <i class="pi pi-desktop text-xl lg:text-2xl"></i>
                 </div>
                 <div class="min-w-0 flex-1">
                     <h3 class="font-bold text-lg lg:text-xl text-text leading-tight mb-1 truncate" :title="client.hostname">{{ client.hostname }}</h3>
                     <Tag 
                        :severity="client.status === 'online' ? 'success' : 'secondary'" 
                        :value="client.status === 'online' ? '在线' : '离线'"
                        :icon="client.status === 'online' ? 'pi pi-check-circle' : 'pi pi-times-circle'"
                        rounded
                        class="!text-xs"
                     ></Tag>
                 </div>
             </div>
          </div>
          
          <!-- Metrics -->
          <div class="space-y-2 lg:space-y-4 mb-4 lg:mb-8 flex-1">
             <div class="flex items-center justify-between p-2 lg:p-3 rounded-lg lg:rounded-xl bg-surface-50">
                 <span class="text-[10px] lg:text-xs font-bold text-text-muted uppercase tracking-wider">版本</span>
                 <span class="text-xs lg:text-sm font-semibold text-text">v{{ client.version || '1.0.0' }}</span>
             </div>
             <div class="flex items-center justify-between p-2 lg:p-3 rounded-lg lg:rounded-xl bg-surface-50">
                 <span class="text-[10px] lg:text-xs font-bold text-text-muted uppercase tracking-wider">设备 ID</span>
                 <span class="text-[10px] lg:text-xs font-mono font-medium text-text-muted truncate max-w-[100px] lg:max-w-[120px]" :title="client.id">
                    {{ client.id }}
                 </span>
             </div>
             <div class="flex items-center justify-between p-2 lg:p-3 rounded-lg lg:rounded-xl bg-surface-50">
                 <span class="text-[10px] lg:text-xs font-bold text-text-muted uppercase tracking-wider">最近心跳</span>
                 <span class="text-xs lg:text-sm font-semibold text-text">{{ timeAgo(client.last_seen) }}</span>
             </div>
          </div>

          <!-- Actions -->
          <div class="grid grid-cols-2 gap-3 lg:gap-4">
             <Button 
                 label="详情" 
                 icon="pi pi-info-circle" 
                 outlined 
                 class="w-full !text-xs lg:!text-base"
                 size="small"
                 @click="router.push('/nodes/' + client.id)"
             />
             <Button 
                 label="控制台" 
                 icon="pi pi-terminal" 
                 :disabled="client.status !== 'online'"
                 :severity="client.status === 'online' ? 'primary' : 'secondary'"
                 class="w-full !text-xs lg:!text-base"
                 size="small"
                 @click="enterConsole(client)"
             />
          </div>
        </div>
    </div>

  </div>
</template>

<script setup>
import { onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useClientStore } from '../store/client'
import { timeAgo } from '../utils/format'

// PrimeVue
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import Skeleton from 'primevue/skeleton'

const clientStore = useClientStore()
const router = useRouter()
let timer = null

const onlineCount = computed(() => clientStore.clients.filter(c => c.status === 'online').length)
const offlineCount = computed(() => clientStore.clients.filter(c => c.status !== 'online').length)

onMounted(() => {
  clientStore.fetchClients()
  timer = setInterval(() => {
    clientStore.fetchClients()
  }, 5000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const enterConsole = (client) => {
  clientStore.setCurrentClient(client.id)
  router.push('/search')
}
</script>
