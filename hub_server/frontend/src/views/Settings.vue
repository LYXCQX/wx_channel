<template>
  <div class="min-h-screen bg-bg p-4 lg:p-12 font-sans text-text">
    <Toast />
    <ConfirmDialog />
    
    <header class="mb-6 lg:mb-8">
      <h2 class="font-serif font-bold text-xl lg:text-2xl text-text mb-1 lg:mb-2">个人中心</h2>
      <p class="text-text-muted text-xs lg:text-sm">管理您的账户信息、安全设置及偏好</p>
    </header>

    <div class="grid grid-cols-1 lg:grid-cols-4 gap-6 lg:gap-8">
        <!-- Sidebar Menu -->
        <div class="lg:col-span-1">
             <div class="bg-surface-0 rounded-2xl shadow-sm border border-surface-100 overflow-hidden sticky top-4">
                <Menu :model="menuItems" class="w-full border-none" />
             </div>
        </div>

        <!-- Content Area -->
        <div class="lg:col-span-3 space-y-6">
            
            <!-- Overview Section (Migrated from Profile.vue) -->
            <div v-if="activeSection === 'overview'" class="space-y-8">
                <!-- User Card & Stats -->
                <div class="bg-surface-0 rounded-2xl shadow-sm border border-surface-100 overflow-hidden flex flex-col md:flex-row transition-all hover:shadow-md">
                    <div class="p-6 lg:p-8 flex flex-col items-center text-center md:items-start md:text-left relative flex-1 gap-6">
                         <div class="flex flex-col md:flex-row items-center gap-6">
                            <div class="relative group">
                                <Avatar :label="userInitial" size="xlarge" shape="circle" class="!w-24 !h-24 !text-3xl shadow-lg border-4 border-surface-0 bg-primary text-primary-contrast" />
                                <div class="absolute bottom-1 right-1 w-5 h-5 bg-green-500 rounded-full border-4 border-surface-0 shadow-sm" title="在线"></div>
                            </div>
                            <div class="space-y-2">
                                <h2 class="text-2xl font-bold text-text break-all">{{ userProfile.email }}</h2>
                                <div class="flex items-center justify-center md:justify-start gap-3">
                                    <Tag :value="roleText" :severity="userProfile.role === 'admin' ? 'warn' : 'info'" rounded class="!px-3 !py-1"></Tag>
                                    <span class="text-sm text-text-muted bg-surface-100 px-3 py-0.5 rounded-full border border-surface-200">ID: {{ userProfile.id }}</span>
                                </div>
                                <div class="text-sm text-text-muted flex items-center justify-center md:justify-start gap-2">
                                     <i class="pi pi-calendar"></i>
                                     <span>加入于 {{ formatDate(userStore.user?.created_at) }}</span>
                                </div>
                            </div>
                         </div>
                    </div>
                    
                    <!-- Stats -->
                    <div class="bg-surface-50/80 p-6 lg:p-8 border-t md:border-t-0 md:border-l border-surface-200 py-8 flex items-center justify-center md:min-w-[350px]">
                         <div class="grid grid-cols-3 gap-8 text-center text-text-muted">
                             <div class="cursor-pointer group" @click="activeSection = 'credits'">
                                 <div class="text-xs font-bold uppercase tracking-widest mb-2 group-hover:text-primary transition-colors">积分</div>
                                 <div class="text-3xl font-black text-primary">{{ userStore.user?.credits || 0 }}</div>
                             </div>
                             <div>
                                 <div class="text-xs font-bold uppercase tracking-widest mb-2">设备</div>
                                 <div class="text-3xl font-black text-text">{{ deviceCount }}</div>
                             </div>
                             <div>
                                 <div class="text-xs font-bold uppercase tracking-widest mb-2">订阅</div>
                                 <div class="text-3xl font-black text-text">{{ subscriptionCount }}</div>
                             </div>
                         </div>
                    </div>
                </div>

                <!-- Quick Actions -->
                 <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
                      <div class="bg-surface-0 p-6 rounded-2xl shadow-sm border border-surface-100 hover:shadow-card hover:border-blue-500/30 transition-all cursor-pointer group flex items-center gap-5" @click="$router.push('/devices')">
                          <div class="w-14 h-14 rounded-2xl bg-blue-50 text-blue-600 flex items-center justify-center text-2xl group-hover:scale-110 transition-transform duration-300">
                              <i class="pi pi-desktop"></i>
                          </div>
                          <div>
                              <h3 class="font-extrabold text-base text-text mb-1">设备管理</h3>
                              <p class="text-sm text-text-muted">管理 {{ deviceCount }} 台在线终端</p>
                          </div>
                      </div>

                      <div class="bg-surface-0 p-6 rounded-2xl shadow-sm border border-surface-100 hover:shadow-card hover:border-green-500/30 transition-all cursor-pointer group flex items-center gap-5" @click="$router.push('/subscriptions')">
                          <div class="w-14 h-14 rounded-2xl bg-green-50 text-green-600 flex items-center justify-center text-2xl group-hover:scale-110 transition-transform duration-300">
                              <i class="pi pi-bookmark"></i>
                          </div>
                          <div>
                              <h3 class="font-extrabold text-base text-text mb-1">订阅管理</h3>
                              <p class="text-sm text-text-muted">管理 {{ subscriptionCount }} 个视频订阅</p>
                          </div>
                      </div>

                      <div class="bg-surface-0 p-6 rounded-2xl shadow-sm border border-surface-100 hover:shadow-card hover:border-purple-500/30 transition-all cursor-pointer group flex items-center gap-5" @click="$router.push('/tasks')">
                          <div class="w-14 h-14 rounded-2xl bg-purple-50 text-purple-600 flex items-center justify-center text-2xl group-hover:scale-110 transition-transform duration-300">
                              <i class="pi pi-list"></i>
                          </div>
                          <div>
                              <h3 class="font-extrabold text-base text-text mb-1">任务记录</h3>
                              <p class="text-sm text-text-muted">查看系统后台任务</p>
                          </div>
                      </div>
                 </div>
            </div>

            <!-- Profile Edit Section -->
            <Panel header="资料修改" id="profile" v-if="activeSection === 'profile'" key="profile" class="!rounded-2xl overflow-hidden !border-surface-100 !shadow-sm" :pt="{ content: { class: '!p-6 lg:!p-8' } }">
                <template #header>
                    <div class="flex items-center gap-3 py-1">
                        <div class="w-8 h-8 rounded-full bg-blue-50 text-blue-500 flex items-center justify-center">
                            <i class="pi pi-user-edit text-sm"></i>
                        </div>
                        <span class="font-bold text-lg text-text">资料修改</span>
                    </div>
                </template>
                <div class="flex flex-col gap-10">
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
                        <div class="flex flex-col gap-3">
                            <label class="font-medium text-sm text-text-muted ml-1">昵称</label>
                            <IconField iconPosition="left" class="w-full">
                                <InputIcon class="pi pi-user" />
                                <InputText v-model="userProfile.nickname" class="!w-full !rounded-xl !py-3 !pl-10" placeholder="设置您的昵称" />
                            </IconField>
                        </div>
                        <div class="flex flex-col gap-3">
                            <label class="font-medium text-sm text-text-muted ml-1">邮箱</label>
                            <IconField iconPosition="left" class="w-full">
                                <InputIcon class="pi pi-envelope" />
                                <InputText v-model="userProfile.email" disabled class="!w-full bg-surface-50 !rounded-xl !py-3 !pl-10 text-text-muted" />
                            </IconField>
                        </div>
                    </div>

                    <div class="flex justify-end border-t border-surface-100 pt-8">
                        <Button label="保存修改" icon="pi pi-check" @click="saveProfile" :loading="loading" class="w-full md:w-auto !rounded-xl !px-8 !py-2.5" />
                    </div>
                </div>
            </Panel>

            <!-- Security Section -->
            <Panel header="账号安全" id="security" v-if="activeSection === 'security'" key="security" class="!rounded-2xl overflow-hidden !border-surface-100 !shadow-sm" :pt="{ content: { class: '!p-6 lg:!p-8' } }">
                 <template #header>
                    <div class="flex items-center gap-3 py-1">
                        <div class="w-8 h-8 rounded-full bg-primary-50 text-primary flex items-center justify-center">
                            <i class="pi pi-shield text-sm"></i>
                        </div>
                        <span class="font-bold text-lg text-text">账号安全</span>
                    </div>
                </template>
                <div class="flex flex-col gap-10">
                    <!-- Password Change -->
                    <div class="flex flex-col gap-6">
                        <div class="flex items-center gap-2 pb-2 border-b border-surface-100">
                            <i class="pi pi-lock text-text-muted"></i>
                            <h3 class="font-bold text-base text-text">修改密码</h3>
                        </div>
                        
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
                            <div class="flex flex-col gap-3">
                                <label class="font-medium text-sm text-text-muted ml-1">当前密码</label>
                                <Password v-model="security.currentPassword" toggleMask :feedback="false" inputClass="!w-full !rounded-xl !py-3 !px-4" class="w-full" placeholder="验证当前密码">
                                    <template #header>
                                        <i class="pi pi-lock" />
                                    </template>
                                </Password>
                            </div>
                        </div>
                         <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
                            <div class="flex flex-col gap-3">
                                <label class="font-medium text-sm text-text-muted ml-1">新密码</label>
                                <Password v-model="security.newPassword" toggleMask inputClass="!w-full !rounded-xl !py-3 !px-4" class="w-full" placeholder="输入新密码" />
                            </div>
                            <div class="flex flex-col gap-3">
                                <label class="font-medium text-sm text-text-muted ml-1">确认新密码</label>
                                <Password v-model="security.confirmPassword" toggleMask :feedback="false" inputClass="!w-full !rounded-xl !py-3 !px-4" class="w-full" placeholder="再次输入新密码" />
                            </div>
                        </div>
                         <div class="flex justify-end pt-4">
                            <Button label="更新密码" severity="primary" icon="pi pi-check-circle" @click="updatePassword" :loading="loading" class="w-full md:w-auto !rounded-xl !px-8 !py-2.5" />
                        </div>
                    </div>

                    <!-- API Token -->
                    <div class="flex flex-col gap-6">
                         <div class="flex justify-between items-center pb-2 border-b border-surface-100">
                             <div class="flex items-center gap-2">
                                <i class="pi pi-key text-text-muted"></i>
                                <div>
                                    <h3 class="font-bold text-base text-text">API 访问令牌</h3>
                                </div>
                             </div>
                             <Button label="重新生成" severity="secondary" text icon="pi pi-refresh" @click="regenerateToken" size="small" class="!rounded-lg !text-xs" />
                         </div>
                         <p class="text-text-muted text-sm -mt-4">用于程序化访问 Hub API 的密钥</p>
                         
                         <div class="bg-surface-50 border border-surface-200 p-5 rounded-xl flex items-center justify-between group h-16 relative transition-colors hover:border-surface-300">
                             <code class="font-mono text-sm text-text-muted truncate select-all">{{ apiToken || 'sk_live_********************************' }}</code>
                             <Button icon="pi pi-copy" class="!w-9 !h-9 !rounded-lg" size="small" text severity="secondary" @click="copyApiToken" />
                         </div>
                    </div>

                    <!-- Logout -->
                    <div class="flex justify-between items-center py-4 bg-red-50/50 rounded-2xl px-6 border border-red-100/50">
                        <div class="flex items-center gap-4">
                            <div class="w-12 h-12 rounded-full bg-red-100 text-red-600 flex items-center justify-center shadow-sm">
                                <i class="pi pi-power-off text-lg"></i>
                            </div>
                            <div>
                                <h3 class="font-bold text-lg text-text">退出登录</h3>
                                <p class="text-text-muted text-sm mt-0.5">注销当前账户的登录状态</p>
                            </div>
                        </div>
                        <Button label="退出账户" severity="danger" outlined icon="pi pi-sign-out" @click="handleLogout" class="!rounded-xl hover:!bg-red-100 !px-6 !py-2.5" />
                    </div>
                </div>
            </Panel>

            <!-- Preferences Section -->
            <Panel header="偏好设置" id="preferences" v-if="activeSection === 'preferences'" key="preferences" class="!rounded-2xl overflow-hidden !border-surface-100 !shadow-sm" :pt="{ content: { class: '!p-6 lg:!p-8' } }">
                 <template #header>
                    <div class="flex items-center gap-3 py-1">
                        <div class="w-8 h-8 rounded-full bg-surface-100 text-text-muted flex items-center justify-center">
                            <i class="pi pi-cog text-sm"></i>
                        </div>
                        <span class="font-bold text-lg text-text">偏好设置</span>
                    </div>
                </template>
                <div class="flex flex-col gap-6">
                     <div class="flex items-center justify-between p-5 rounded-2xl hover:bg-surface-50 transition-colors border border-transparent hover:border-surface-100">
                         <div class="flex items-center gap-5">
                             <div class="w-12 h-12 rounded-full bg-surface-100 flex items-center justify-center text-text-muted text-xl">
                                 <i class="pi pi-moon"></i>
                             </div>
                             <div>
                                 <h3 class="font-bold text-lg text-text">深色模式</h3>
                                 <p class="text-text-muted text-sm mt-1">切换应用程序的明暗主题</p>
                             </div>
                         </div>
                         <ToggleSwitch v-model="preferences.darkMode" @change="toggleTheme" />
                     </div>

                     <div class="flex items-center justify-between p-5 rounded-2xl hover:bg-surface-50 transition-colors border border-transparent hover:border-surface-100">
                         <div class="flex items-center gap-5">
                             <div class="w-12 h-12 rounded-full bg-surface-100 flex items-center justify-center text-text-muted text-xl">
                                 <i class="pi pi-envelope"></i>
                             </div>
                             <div>
                                 <h3 class="font-bold text-lg text-text">邮件通知</h3>
                                 <p class="text-text-muted text-sm mt-1">接收关于任务完成和系统警告的邮件</p>
                             </div>
                         </div>
                         <ToggleSwitch v-model="preferences.notifications" @change="showNotImplemented('邮件通知')" />
                     </div>
                     
                     <div class="flex flex-col gap-4 p-5 rounded-2xl hover:bg-surface-50 transition-colors border border-transparent hover:border-surface-100">
                        <label class="font-bold text-lg flex items-center gap-4">
                             <div class="w-12 h-12 rounded-full bg-surface-100 flex items-center justify-center text-text-muted text-xl">
                                 <i class="pi pi-globe"></i>
                             </div>
                             <span>语言区域</span>
                        </label>
                        <SelectButton v-model="preferences.locale" :options="locales" optionLabel="name" optionValue="code" aria-labelledby="basic" class="w-full mt-2" :pt="{ button: { class: '!flex-1 !py-3' } }" @change="showNotImplemented('多语言')" />
                     </div>
                </div>
            </Panel>

            <!-- Credits Section -->
            <Panel header="积分记录" id="credits" v-if="activeSection === 'credits'" key="credits" class="!rounded-2xl overflow-hidden !border-surface-100 !shadow-sm" :pt="{ content: { class: '!p-6 lg:!p-8' } }">
                 <template #header>
                    <div class="flex items-center gap-3 py-1">
                        <div class="w-8 h-8 rounded-full bg-yellow-50 text-yellow-500 flex items-center justify-center">
                            <i class="pi pi-history text-sm"></i>
                        </div>
                        <span class="font-bold text-lg text-text">积分记录</span>
                    </div>
                </template>
                <div class="pt-2">
                    <DataTable :value="transactions" :loading="loadingTransactions" tableStyle="min-width: 100%" 
                        paginator :rows="rows" :totalRecords="totalRecords" lazy @page="onPage"
                        class="p-datatable-sm"
                        stripedRows
                        :showGridlines="false"
                        :pt="{
                            headerRow: { class: '!bg-surface-50 !text-text-muted !text-sm' },
                            bodyRow: { class: '!hover:bg-surface-50 transition-colors' },
                            headerCell: { class: '!py-4 !px-4' },
                            bodyCell: { class: '!py-4 !px-4' }
                        }"
                    >
                        <template #empty>
                            <div class="flex flex-col items-center justify-center py-12 text-text-muted">
                                <i class="pi pi-inbox text-5xl mb-4 opacity-30"></i>
                                <p class="text-lg">暂无积分记录</p>
                            </div>
                        </template>
                        <Column field="created_at" header="时间" style="width: 25%" sortable>
                            <template #body="slotProps">
                                <span class="text-text-muted text-sm font-mono">{{ formatTime(slotProps.data.created_at) }}</span>
                            </template>
                        </Column>
                        <Column field="type" header="类型" style="width: 20%" sortable>
                             <template #body="slotProps">
                                <Tag :value="formatType(slotProps.data.type)" :severity="getTypeSeverity(slotProps.data.type)" rounded class="!px-3 !py-1" />
                            </template>
                        </Column>
                        <Column field="amount" header="变动" style="width: 15%" sortable>
                            <template #body="slotProps">
                                <span :class="getAmountClass(slotProps.data.amount)" class="font-bold font-mono text-base">
                                    {{ slotProps.data.amount > 0 ? '+' : '' }}{{ slotProps.data.amount }}
                                </span>
                            </template>
                        </Column>
                        <Column field="description" header="详情" style="width: 40%">
                            <template #body="slotProps">
                                <span class="text-text-muted text-sm truncate block max-w-[200px] lg:max-w-md" :title="slotProps.data.description">{{ slotProps.data.description }}</span>
                            </template>
                        </Column>
                    </DataTable>
                </div>
            </Panel>

        </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, computed } from 'vue'
import { useToast } from 'primevue/usetoast'
import axios from 'axios'
import { formatTime } from '../utils/format'
import { useRouter } from 'vue-router'
import { useUserStore } from '../store/user'
import { useConfirm } from 'primevue/useconfirm'

// PrimeVue
import Panel from 'primevue/panel'
import Menu from 'primevue/menu'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import IconField from 'primevue/iconfield'
import InputIcon from 'primevue/inputicon'
import Password from 'primevue/password'
import ToggleSwitch from 'primevue/toggleswitch'
import SelectButton from 'primevue/selectbutton'
import Tag from 'primevue/tag'
import Toast from 'primevue/toast'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Avatar from 'primevue/avatar'
import ConfirmDialog from 'primevue/confirmdialog'

const router = useRouter()
const userStore = useUserStore()
const toast = useToast()
const confirm = useConfirm()
const loading = ref(false)
const activeSection = ref('overview')

const menuItems = ref([
    { label: '概览', icon: 'pi pi-home', command: () => activeSection.value = 'overview' },
    { label: '资料修改', icon: 'pi pi-user-edit', command: () => activeSection.value = 'profile' },
    { label: '账号安全', icon: 'pi pi-shield', command: () => activeSection.value = 'security' },
    { label: '积分记录', icon: 'pi pi-history', command: () => activeSection.value = 'credits' },
    { label: '偏好设置', icon: 'pi pi-cog', command: () => activeSection.value = 'preferences' }
])

const userProfile = ref({
    id: 1,
    nickname: 'Admin User',
    email: 'admin@example.com',
    role: 'admin',
    avatar: ''
})

const deviceCount = ref(0)
const subscriptionCount = ref(0)

const security = ref({
    currentPassword: '',
    newPassword: '',
    confirmPassword: ''
})

const apiToken = ref('sk_live_51J9...')

const preferences = ref({
    darkMode: document.documentElement.classList.contains('dark'),
    notifications: true,
    locale: 'zh-CN'
})

const locales = [
    { name: '简体中文', code: 'zh-CN' },
    { name: 'English', code: 'en-US' }
]

const userInitial = computed(() => {
  if (!userStore.user?.email) return '?'
  return userStore.user.email.charAt(0).toUpperCase()
})

const roleText = computed(() => {
  if (userStore.user?.role === 'admin') return '管理员'
  return '普通用户'
})

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })
}

const loadStats = async () => {
  try {
    const token = localStorage.getItem('token')
    
    // Load user stats (device count, subscription count, credits)
    const statsRes = await fetch('/api/user/stats', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    const statsData = await statsRes.json()
    
    if (statsData.code === 0 && statsData.data) {
      deviceCount.value = statsData.data.device_count || 0
      subscriptionCount.value = statsData.data.subscription_count || 0
    }
  } catch (e) {
    console.error('Failed to load stats:', e)
  }
}

const handleLogout = () => {
  confirm.require({
    message: '确定要退出登录吗？',
    header: '退出登录',
    icon: 'pi pi-exclamation-triangle',
    rejectProps: { label: '取消', severity: 'secondary', outlined: true, class: '!rounded-lg' },
    acceptProps: { label: '退出', severity: 'danger', class: '!rounded-lg' },
    accept: () => {
        userStore.logout()
        router.push('/login')
        toast.add({ severity: 'info', summary: 'Goodbye', detail: '您已退出登录', life: 2000 })
    }
  })
}

const saveProfile = async () => {
    loading.value = true
    // Simulate API call
    await new Promise(r => setTimeout(r, 1000))
    loading.value = false
    toast.add({ severity: 'success', summary: 'Success', detail: '个人资料已更新', life: 3000 })
}

const updatePassword = async () => {
    if (security.value.newPassword !== security.value.confirmPassword) {
        toast.add({ severity: 'error', summary: 'Error', detail: '两次输入的密码不一致', life: 3000 })
        return
    }
    loading.value = true
    await new Promise(r => setTimeout(r, 1500))
    loading.value = false
    security.value = { currentPassword: '', newPassword: '', confirmPassword: '' }
    toast.add({ severity: 'success', summary: 'Success', detail: '密码已修改', life: 3000 })
}

const regenerateToken = () => {
    apiToken.value = 'sk_live_' + Math.random().toString(36).substring(7)
    toast.add({ severity: 'info', summary: 'Token Refresh', detail: 'API Token 已重新生成', life: 3000 })
}

const copyApiToken = () => {
    navigator.clipboard.writeText(apiToken.value)
    toast.add({ severity: 'success', summary: 'Copied', detail: '已复制到剪贴板', life: 2000 })
}

const toggleTheme = () => {
    document.documentElement.classList.toggle('dark')
}

// Credits Logic
// Credits Logic
const transactions = ref([])
const loadingTransactions = ref(false)
const totalRecords = ref(0)
const rows = ref(20)

const fetchTransactions = async (page = 1) => {
    loadingTransactions.value = true
    try {
        const res = await axios.get(`/api/user/transactions?page=${page}`)
        if (res.data.code === 0) {
            transactions.value = res.data.data.list
            totalRecords.value = res.data.data.total
        }
    } catch (e) {
        toast.add({ severity: 'error', summary: 'Error', detail: '加载积分记录失败' })
    } finally {
        loadingTransactions.value = false
    }
}

const onPage = (event) => {
    fetchTransactions(event.page + 1)
}

onMounted(() => {
    fetchTransactions()
})

watch(activeSection, (newVal) => {
    if (newVal === 'credits') {
        fetchTransactions()
    }
})

const formatType = (type) => {
    const map = {
        'mining': '在线挖矿',
        'search_channels': '搜索',
        'search_videos': '搜索',
        'download_video': '下载',
        'feed_profile': '查看详情',
        'feed_list': '浏览主页'
    }
    return map[type] || type
}

const getTypeSeverity = (type) => {
    if (type === 'mining') return 'success'
    if (type === 'download_video') return 'info'
    return 'secondary'
}

const getAmountClass = (amount) => {
    return amount > 0 ? 'text-green-500' : 'text-red-500'
}

const showNotImplemented = (feature) => {
    toast.add({ severity: 'info', summary: 'Coming Soon', detail: `${feature} 功能暂未开放`, life: 2000 })
}


import { useRoute } from 'vue-router'
const route = useRoute()

onMounted(async () => {
    if (userStore.user) {
        userProfile.value = {
            id: userStore.user.id,
            nickname: userStore.user.email.split('@')[0], // Fallback nickname
            email: userStore.user.email,
            role: userStore.user.role,
            avatar: ''
        }
    }
    
    await loadStats()
    
    // Check for tab query param
    if (route.query.tab) {
        // Fix: check against valid sections including overview
        if (['overview', 'profile', 'security', 'preferences', 'credits'].includes(route.query.tab)) {
            activeSection.value = route.query.tab
        }
    }

    // Also fetch transactions if starting on credits tab (unlikely but possible)
    if (activeSection.value === 'credits') {
        fetchTransactions()
    }
})

</script>
