<template>
  <div class="min-h-screen w-full bg-bg flex font-sans">
    <!-- 侧边栏 -->
    <Sidebar />
    
    <!-- 主内容区域 - 添加左边距为侧边栏宽度 -->
    <div class="flex-1 flex flex-col overflow-hidden ml-64">
      <!-- 顶部栏 - 固定在顶部 -->
      <header class="bg-white shadow-sm px-8 py-4 flex items-center justify-between sticky top-0 z-40">
        <div>
          <h1 class="text-2xl font-serif font-bold text-slate-800">{{ pageTitle }}</h1>
          <p class="text-sm font-sans text-slate-500">{{ pageDescription }}</p>
        </div>
        
        <!-- 用户信息 -->
        <div class="flex items-center gap-4">
          <div class="text-right">
            <p class="text-sm font-medium text-slate-800">{{ user?.username || 'Guest' }}</p>
            <p class="text-xs text-slate-500">{{ user?.email || '' }}</p>
          </div>
          <div class="w-10 h-10 rounded-xl bg-bg shadow-neu flex items-center justify-center text-primary font-bold">
            {{ userInitial }}
          </div>
          <button
            @click="handleLogout"
            class="px-4 py-2 rounded-xl bg-bg shadow-neu hover:shadow-neu-sm text-slate-600 hover:text-red-600 transition-all flex items-center gap-2"
            title="退出登录"
          >
            <component :is="LogOut" class="w-4 h-4" />
            <span class="text-sm font-medium">退出</span>
          </button>
        </div>
      </header>

      <!-- 主内容 -->
      <main class="flex-1 overflow-y-auto bg-green-50/30">
        <slot></slot>
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Sidebar from '../components/Sidebar.vue'
import { useUserStore } from '../store/user'
import { LogOut } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const user = computed(() => userStore.user)

const userInitial = computed(() => {
  if (!user.value?.username) return 'G'
  return user.value.username.charAt(0).toUpperCase()
})

const handleLogout = () => {
  userStore.logout()
  router.push('/login')
}

const pageTitle = computed(() => {
  const titles = {
    '/dashboard': '在线终端',
    '/search': '穿透搜索',
    '/subscriptions': '订阅管理',
    '/devices': '设备管理',
    '/tasks': '任务追踪',
    '/monitoring': '系统监控',
    '/profile': '个人资料',
    '/admin': '系统管理'
  }
  // 处理动态路由
  if (route.path.includes('/subscriptions/') && route.path.includes('/videos')) {
    return '订阅视频'
  }
  return titles[route.path] || '控制面板'
})

const pageDescription = computed(() => {
  const descriptions = {
    '/dashboard': '查看所有在线的客户端终端',
    '/search': '远程搜索视频号内容',
    '/subscriptions': '管理您的视频号订阅',
    '/devices': '管理您绑定的所有设备',
    '/tasks': '查看和管理任务执行状态',
    '/monitoring': '实时监控系统运行状态',
    '/profile': '查看和编辑个人信息',
    '/admin': '管理用户和系统资源'
  }
  // 处理动态路由
  if (route.path.includes('/subscriptions/') && route.path.includes('/videos')) {
    return '查看订阅的视频内容'
  }
  return descriptions[route.path] || '欢迎使用 Hub Control'
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
