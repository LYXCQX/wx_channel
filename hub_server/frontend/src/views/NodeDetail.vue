<template>
  <div class="min-h-screen bg-bg p-4 lg:p-12 font-sans text-text">
    <!-- Loading State -->
    <div v-if="loading && !node" class="flex justify-center p-12">
      <div class="w-8 h-8 border-4 border-primary/30 border-t-primary rounded-full animate-spin"></div>
    </div>

    <!-- Empty State -->
    <div v-else-if="!node" class="p-8 lg:p-12 text-center bg-surface-0 rounded-[2rem] shadow-card">
      <p class="text-text-muted mb-4">节点不存在或已删除</p>
      <button 
        class="px-6 py-3 rounded-xl bg-bg shadow-neu-btn text-text font-semibold hover:text-primary active:shadow-neu-btn-active transition-all"
        @click="router.push('/dashboard')">
        返回概览
      </button>
    </div>

    <!-- Node Detail -->
    <div v-else>
      <!-- Header -->
      <header class="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-6 lg:mb-12">
        <div class="flex items-center gap-4 lg:gap-6">
          <button 
            class="w-10 h-10 lg:w-12 lg:h-12 rounded-full bg-bg shadow-neu-btn flex items-center justify-center text-text hover:text-primary active:shadow-neu-btn-active transition-all" 
            @click="router.back()">
            <ArrowLeft class="w-4 h-4 lg:w-5 lg:h-5" />
          </button>
          <div>
            <h1 class="font-serif font-bold text-xl lg:text-3xl text-text mb-0.5 lg:mb-1">{{ node.hostname }}</h1>
            <p class="font-mono text-xs lg:text-sm text-text-muted break-all">{{ node.id }}</p>
          </div>
        </div>
        <div class="self-end md:self-auto">
          <span 
            class="px-3 py-1 lg:px-4 lg:py-2 rounded-lg lg:rounded-xl font-semibold text-xs lg:text-sm uppercase tracking-wide"
            :class="node.status === 'online' ? 'bg-green-100 text-green-700' : 'bg-slate-100 text-slate-500'">
            {{ node.status }}
          </span>
        </div>
      </header>

      <!-- Stats Grid -->
      <div class="grid grid-cols-2 md:grid-cols-2 lg:grid-cols-4 gap-3 lg:gap-6 mb-8 lg:mb-12">
        <div class="bg-surface-0 rounded-xl lg:rounded-2xl p-4 lg:p-6 shadow-card border border-surface-100">
          <div class="text-[10px] lg:text-xs uppercase tracking-wider text-text-muted mb-1 lg:mb-2">IP 地址</div>
          <div class="text-sm lg:text-lg font-semibold text-text truncate" :title="node.ip">{{ node.ip || 'Unknown' }}</div>
        </div>
        <div class="bg-surface-0 rounded-xl lg:rounded-2xl p-4 lg:p-6 shadow-card border border-surface-100">
          <div class="text-[10px] lg:text-xs uppercase tracking-wider text-text-muted mb-1 lg:mb-2">客户端版本</div>
          <div class="text-sm lg:text-lg font-semibold text-text truncate">{{ node.version }}</div>
        </div>
        <div class="bg-surface-0 rounded-xl lg:rounded-2xl p-4 lg:p-6 shadow-card border border-surface-100">
          <div class="text-[10px] lg:text-xs uppercase tracking-wider text-text-muted mb-1 lg:mb-2">首次发现</div>
          <div class="text-xs lg:text-sm font-semibold text-text truncate">{{ formatTime(node.created_at) }}</div>
        </div>
        <div class="bg-surface-0 rounded-xl lg:rounded-2xl p-4 lg:p-6 shadow-card border border-surface-100">
          <div class="text-[10px] lg:text-xs uppercase tracking-wider text-text-muted mb-1 lg:mb-2">最近心跳</div>
          <div class="text-xs lg:text-sm font-semibold text-text truncate">{{ formatTime(node.last_seen) }}</div>
        </div>
      </div>

      <!-- Task History Section -->
      <div class="mb-4 lg:mb-6">
        <h3 class="font-serif font-bold text-lg lg:text-2xl text-text">执行历史</h3>
      </div>

      <!-- Task Table -->
      <div class="bg-surface-0 rounded-xl lg:rounded-2xl shadow-card border border-surface-100 overflow-hidden">
        <div class="overflow-x-auto custom-scrollbar">
          <table class="w-full whitespace-nowrap">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-100">
                <th class="px-4 py-3 lg:px-6 lg:py-4 text-left text-[10px] lg:text-xs font-semibold text-text-muted uppercase tracking-wider">ID</th>
                <th class="px-4 py-3 lg:px-6 lg:py-4 text-left text-[10px] lg:text-xs font-semibold text-text-muted uppercase tracking-wider">类型</th>
                <th class="px-4 py-3 lg:px-6 lg:py-4 text-left text-[10px] lg:text-xs font-semibold text-text-muted uppercase tracking-wider">状态</th>
                <th class="px-4 py-3 lg:px-6 lg:py-4 text-left text-[10px] lg:text-xs font-semibold text-text-muted uppercase tracking-wider">时间</th>
                <th class="px-4 py-3 lg:px-6 lg:py-4 text-left text-[10px] lg:text-xs font-semibold text-text-muted uppercase tracking-wider">详情</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="task in tasks" :key="task.id" class="border-b border-slate-100 hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 lg:px-6 lg:py-4 text-xs lg:text-sm text-text">#{{ task.id }}</td>
                <td class="px-4 py-3 lg:px-6 lg:py-4">
                  <span class="px-2 py-0.5 lg:px-3 lg:py-1 rounded-md lg:rounded-lg bg-primary/10 text-primary text-[10px] lg:text-xs font-semibold">
                    {{ task.type }}
                  </span>
                </td>
                <td class="px-4 py-3 lg:px-6 lg:py-4">
                  <span 
                    class="px-2 py-0.5 lg:px-3 lg:py-1 rounded-md lg:rounded-lg text-[10px] lg:text-xs font-semibold capitalize"
                    :class="{
                      'bg-green-100 text-green-700': task.status === 'success',
                      'bg-red-100 text-red-700': task.status === 'failed',
                      'bg-yellow-100 text-yellow-700': task.status === 'pending',
                      'bg-slate-100 text-slate-700': task.status === 'timeout'
                    }">
                    {{ task.status }}
                  </span>
                </td>
                <td class="px-4 py-3 lg:px-6 lg:py-4 text-xs lg:text-sm text-text-muted">{{ formatTime(task.created_at) }}</td>
                <td class="px-4 py-3 lg:px-6 lg:py-4">
                  <button 
                    class="px-3 py-1.5 lg:px-4 lg:py-2 rounded-lg lg:rounded-xl bg-bg shadow-neu-btn text-text text-[10px] lg:text-xs font-semibold hover:text-primary active:shadow-neu-btn-active transition-all"
                    @click="showTaskDetail(task)">
                    查看
                  </button>
                </td>
              </tr>
              <tr v-if="tasks.length === 0">
                <td colspan="5" class="px-6 py-8 lg:py-12 text-center text-xs lg:text-sm text-text-muted">
                  暂无历史记录
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useClientStore } from '../store/client'
import { ArrowLeft } from 'lucide-vue-next'
import { formatTime } from '../utils/format'
import axios from 'axios'

const route = useRoute()
const router = useRouter()
const clientStore = useClientStore()

const node = ref(null)
const tasks = ref([])
const loading = ref(true)

onMounted(async () => {
    const id = route.params.id
    // Try to find in store first
    node.value = clientStore.getClientById(id)
    
    // If not found (offline node?), fetch clients
    if (!node.value) {
        await clientStore.fetchClients()
        node.value = clientStore.getClientById(id)
    }

    if (node.value) {
        loadTasks(node.value.id)
    } else {
        loading.value = false
    }
})

const loadTasks = async (nodeId) => {
    try {
        const res = await axios.get(`/api/tasks?node_id=${nodeId}&limit=50`)
        tasks.value = res.data.list || []
    } catch (e) {
        console.error(e)
    } finally {
        loading.value = false
    }
}

const showTaskDetail = (task) => {
    alert(JSON.stringify(task, null, 2))
}
</script>
