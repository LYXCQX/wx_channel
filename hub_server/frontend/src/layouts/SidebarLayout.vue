<template>
  <div class="min-h-screen w-full bg-bg flex font-sans">
    <!-- Sidebar -->
    <SidebarComponent :visible="sidebarVisible" />
    
    <!-- Mobile Mask -->
    <div v-if="sidebarVisible" class="fixed inset-0 bg-black/50 z-40 lg:hidden transition-opacity duration-300" @click="sidebarVisible = false"></div>

    <!-- Main Content -->
    <div class="flex-1 flex flex-col overflow-hidden transition-all duration-300" :class="[sidebarVisible ? 'lg:ml-64' : 'ml-0']">
      <!-- Topbar -->
      <header class="bg-surface-0 border-b border-surface-200 px-8 py-4 flex items-center justify-between sticky top-0 z-40 shadow-sm h-20">
        <div class="flex items-center gap-4">
            <Button icon="pi pi-bars" text rounded @click="toggleSidebar" aria-label="Toggle Sidebar" />
            <div>
                <h1 class="text-2xl font-serif font-bold text-text">{{ pageTitle }}</h1>
                <p class="text-sm font-sans text-text-muted">{{ pageDescription }}</p>
            </div>
        </div>
        
        <!-- User Profile -->
        <div class="flex items-center gap-4">
          <div class="text-right hidden md:block">
            <p class="text-sm font-medium text-text">{{ user?.username || user?.email || 'Guest' }}</p>
            <Tag :value="user?.role" :severity="user?.role === 'admin' ? 'warn' : 'info'" class="scale-90" rounded></Tag>
          </div>

          <!-- Dark Mode Toggle -->
          <DarkModeToggle />
          
           <!-- User Profile -->
           <!-- User Menu Backdrop -->
           <div v-if="userMenuVisible" class="fixed inset-0 z-40" @click="userMenuVisible = false"></div>

           <div class="relative z-50">
                <Avatar :label="userInitial" shape="circle" size="large" class="bg-primary text-primary-contrast cursor-pointer hover:shadow-md transition-shadow" @click="toggleUserMenu" />
                
                 <!-- Dropdown -->
                 <div v-if="userMenuVisible" class="absolute right-0 top-full mt-2 w-48 bg-surface-0 border border-surface-200 rounded-xl shadow-lg p-2 animate-fade-in origin-top-right">
                    <div class="px-3 py-2 border-b border-surface-100 mb-2 md:hidden">
                         <p class="text-sm font-bold truncate">{{ user?.email }}</p>
                    </div>
                    <router-link to="/settings" class="flex items-center gap-2 px-3 py-2 rounded-lg hover:bg-surface-50 text-text transition-colors" @click="userMenuVisible = false">
                        <i class="pi pi-user"></i>
                        <span>个人中心</span>
                    </router-link>
                     <button @click="handleLogout" class="w-full flex items-center gap-2 px-3 py-2 rounded-lg hover:bg-red-50 text-red-600 transition-colors text-left">
                        <i class="pi pi-sign-out"></i>
                        <span>退出登录</span>
                    </button>
                 </div>
           </div>
        </div>
      </header>

      <!-- Main Content Area -->
      <main class="flex-1 overflow-y-auto bg-pattern p-0">
        <slot></slot>
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import SidebarComponent from '../components/Sidebar.vue'
import DarkModeToggle from '../components/DarkModeToggle.vue'
import { useUserStore } from '../store/user'
import Avatar from 'primevue/avatar'
import Tag from 'primevue/tag'
import Button from 'primevue/button'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

// Default to visible on larger screens, hidden on mobile
const sidebarVisible = ref(window.innerWidth >= 1024)

const toggleSidebar = () => {
    sidebarVisible.value = !sidebarVisible.value
}

// Close sidebar on route change if on mobile
router.afterEach(() => {
    if (window.innerWidth < 1024) {
        sidebarVisible.value = false
    }
})

const user = computed(() => userStore.user)

const userInitial = computed(() => {
  const name = user.value?.username || user.value?.email
  if (!name) return '?'
  return name.charAt(0).toUpperCase()
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
    '/settings': '个人中心',
    '/admin': '系统管理'
  }
  if (route.path.includes('/subscriptions/') && route.path.includes('/videos')) {
    return '订阅视频'
  }
  if (route.path.includes('/nodes/')) {
      return '终端详情'
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
    '/settings': '管理您的账户信息、安全设置及偏好',
    '/admin': '管理用户和系统资源'
  }
  if (route.path.includes('/subscriptions/') && route.path.includes('/videos')) {
    return '查看订阅的视频内容'
  }
  return descriptions[route.path] || '欢迎使用 Hub Control'
})


// User Menu Logic
const userMenuVisible = ref(false)
const toggleUserMenu = () => {
    userMenuVisible.value = !userMenuVisible.value
}
</script>

<style scoped>
@keyframes fadeIn {
    from { opacity: 0; transform: translateY(-5px); }
    to { opacity: 1; transform: translateY(0); }
}
.animate-fade-in {
    animation: fadeIn 0.15s ease-out;
}
</style>
