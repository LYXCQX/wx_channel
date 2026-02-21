<template>
  <div class="min-h-screen bg-bg font-sans">
    <Toast />
    <ConfirmDialog />

    <!-- Main Content -->
    <div class="max-w-7xl mx-auto px-4 py-8 lg:px-8 lg:py-12 relative z-10">
        <div class="grid grid-cols-1 lg:grid-cols-12 gap-8">
            
            <!-- Left Column: User Profile & Stats (4 cols) -->
            <div class="lg:col-span-4 space-y-6">
                <!-- User Card -->
                <div class="bg-surface-0 rounded-2xl lg:rounded-3xl shadow-xl border border-surface-100 overflow-hidden flex flex-col animate-fade-in-up">
                    <div class="p-6 lg:p-8 flex flex-col items-center text-center relative">
                        <div class="absolute top-0 left-0 right-0 h-24 lg:h-28 bg-gradient-to-b from-surface-50 to-transparent"></div>
                        
                        <div class="relative mb-3 lg:mb-4 group z-10">
                             <Avatar :label="userInitial" size="xlarge" shape="circle" class="!w-24 !h-24 lg:!w-32 lg:!h-32 !text-3xl lg:!text-4xl shadow-card border-4 border-surface-0 bg-primary text-primary-contrast transition-transform group-hover:scale-105" />
                             <div class="absolute bottom-1 lg:bottom-2 right-1 lg:right-2 w-5 h-5 lg:w-6 lg:h-6 bg-green-500 rounded-full border-2 border-surface-0 shadow-sm" title="在线"></div>
                        </div>

                        <h2 class="text-xl lg:text-2xl font-bold text-text mb-1 break-all relative z-10">{{ user?.email }}</h2>
                        <div class="flex items-center justify-center gap-2 mb-4 lg:mb-6 relative z-10">
                            <Tag :value="roleText" :severity="user?.role === 'admin' ? 'warn' : 'info'" rounded class="!px-2 lg:!px-3 !text-xs lg:!text-sm"></Tag>
                            <span class="text-xs text-text-muted bg-surface-100 px-2 lg:px-3 py-0.5 lg:py-1 rounded-full">
                                ID: {{ user?.id }}
                            </span>
                        </div>
                        
                        <div class="text-xs lg:text-sm text-text-muted flex items-center gap-2 relative z-10">
                             <i class="pi pi-calendar"></i>
                             <span>加入于 {{ formatDate(user?.created_at) }}</span>
                        </div>
                    </div>

                    <!-- Stats -->
                    <div class="mt-auto bg-surface-50/50 p-4 lg:p-6 border-t border-surface-200/60 backdrop-blur-sm">
                         <div class="grid grid-cols-3 gap-2 text-center divide-x divide-surface-200">
                             <div class="px-1 lg:px-2">
                                 <div class="text-[10px] lg:text-xs text-text-muted mb-1 font-medium transform px-1 uppercase tracking-wider">积分</div>
                                 <div class="text-lg lg:text-xl font-bold text-primary">{{ user?.credits || 0 }}</div>
                             </div>
                             <div class="px-1 lg:px-2">
                                 <div class="text-[10px] lg:text-xs text-text-muted mb-1 font-medium transform px-1 uppercase tracking-wider">设备</div>
                                 <div class="text-lg lg:text-xl font-bold text-text">{{ deviceCount }}</div>
                             </div>
                             <div class="px-1 lg:px-2">
                                 <div class="text-[10px] lg:text-xs text-text-muted mb-1 font-medium transform px-1 uppercase tracking-wider">订阅</div>
                                 <div class="text-lg lg:text-xl font-bold text-text">{{ subscriptionCount }}</div>
                             </div>
                         </div>
                    </div>
                </div>

                <!-- Info Card -->
                <div class="bg-surface-0 rounded-2xl lg:rounded-3xl shadow-sm border border-surface-100 p-4 lg:p-6 animate-fade-in-up" style="animation-delay: 0.1s">
                    <h3 class="font-bold text-base lg:text-lg mb-3 lg:mb-4 flex items-center gap-2">
                        <i class="pi pi-id-card text-primary"></i>
                        账户详情
                    </h3>
                    <div class="space-y-2 lg:space-y-3">
                        <div class="flex items-center justify-between p-3 lg:p-3.5 rounded-xl bg-surface-50 border border-transparent hover:border-surface-200 transition-all">
                             <span class="text-text-muted text-xs lg:text-sm">用户名</span>
                             <span class="font-medium text-text text-sm lg:text-base">{{ user?.username || '未设置' }}</span>
                        </div>
                        <div class="flex items-center justify-between p-3 lg:p-3.5 rounded-xl bg-surface-50 border border-transparent hover:border-surface-200 transition-all">
                             <span class="text-text-muted text-xs lg:text-sm">绑定邮箱</span>
                             <span class="font-medium text-text text-sm lg:text-base truncate max-w-[150px] lg:max-w-[180px]">{{ user?.email }}</span>
                        </div>
                        <div class="flex items-center justify-between p-3 lg:p-3.5 rounded-xl bg-surface-50 border border-transparent hover:border-surface-200 transition-all">
                             <span class="text-text-muted text-xs lg:text-sm">最后登录</span>
                             <span class="font-medium text-text text-xs lg:text-sm">{{ formatDate(new Date()) }}</span>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Right Column: Actions & Security (8 cols) -->
            <div class="lg:col-span-8 space-y-4 lg:space-y-6">
                 
                 <!-- Quick Actions -->
                 <div class="grid grid-cols-2 lg:grid-cols-2 gap-3 lg:gap-4 animate-fade-in-up" style="animation-delay: 0.2s">
                      <div class="bg-surface-0 p-4 lg:p-6 rounded-2xl lg:rounded-3xl shadow-sm border border-surface-100 hover:shadow-card hover:border-primary/30 transition-all cursor-pointer group relative overflow-hidden" @click="$router.push('/devices')">
                          <div class="absolute right-0 top-0 p-6 opacity-0 group-hover:opacity-10 group-hover:scale-125 transition-all duration-500 text-6xl text-blue-500">
                              <i class="pi pi-desktop"></i>
                          </div>
                          <div class="w-10 h-10 lg:w-14 lg:h-14 rounded-xl lg:rounded-2xl bg-blue-50 text-blue-600 flex items-center justify-center text-xl lg:text-2xl mb-3 lg:mb-4 group-hover:bg-blue-500 group-hover:text-white transition-all shadow-sm">
                              <i class="pi pi-desktop"></i>
                          </div>
                          <h3 class="text-sm lg:text-lg font-bold mb-0.5 lg:mb-1 group-hover:text-blue-600 transition-colors">设备管理</h3>
                          <p class="text-text-muted text-[10px] lg:text-sm line-clamp-1">查看和管理您的在线终端设备</p>
                      </div>

                      <div class="bg-surface-0 p-4 lg:p-6 rounded-2xl lg:rounded-3xl shadow-sm border border-surface-100 hover:shadow-card hover:border-green-500/30 transition-all cursor-pointer group relative overflow-hidden" @click="$router.push('/subscriptions')">
                          <div class="absolute right-0 top-0 p-6 opacity-0 group-hover:opacity-10 group-hover:scale-125 transition-all duration-500 text-6xl text-green-500">
                              <i class="pi pi-bookmark"></i>
                          </div>
                          <div class="w-10 h-10 lg:w-14 lg:h-14 rounded-xl lg:rounded-2xl bg-green-50 text-green-600 flex items-center justify-center text-xl lg:text-2xl mb-3 lg:mb-4 group-hover:bg-green-500 group-hover:text-white transition-all shadow-sm">
                              <i class="pi pi-bookmark"></i>
                          </div>
                          <h3 class="text-sm lg:text-lg font-bold mb-0.5 lg:mb-1 group-hover:text-green-600 transition-colors">订阅管理</h3>
                          <p class="text-text-muted text-[10px] lg:text-sm line-clamp-1">管理您的视频号订阅内容</p>
                      </div>

                      <div class="bg-surface-0 p-4 lg:p-6 rounded-2xl lg:rounded-3xl shadow-sm border border-surface-100 hover:shadow-card hover:border-purple-500/30 transition-all cursor-pointer group relative overflow-hidden" @click="$router.push('/tasks')">
                          <div class="absolute right-0 top-0 p-6 opacity-0 group-hover:opacity-10 group-hover:scale-125 transition-all duration-500 text-6xl text-purple-500">
                              <i class="pi pi-list"></i>
                          </div>
                          <div class="w-10 h-10 lg:w-14 lg:h-14 rounded-xl lg:rounded-2xl bg-purple-50 text-purple-600 flex items-center justify-center text-xl lg:text-2xl mb-3 lg:mb-4 group-hover:bg-purple-500 group-hover:text-white transition-all shadow-sm">
                              <i class="pi pi-list"></i>
                          </div>
                          <h3 class="text-sm lg:text-lg font-bold mb-0.5 lg:mb-1 group-hover:text-purple-600 transition-colors">任务记录</h3>
                          <p class="text-text-muted text-[10px] lg:text-sm line-clamp-1">查看系统后台任务执行状态</p>
                      </div>

                      <div v-if="user?.role === 'admin'" class="bg-surface-0 p-4 lg:p-6 rounded-2xl lg:rounded-3xl shadow-sm border border-surface-100 hover:shadow-card hover:border-orange-500/30 transition-all cursor-pointer group relative overflow-hidden" @click="$router.push('/monitoring')">
                          <div class="absolute right-0 top-0 p-6 opacity-0 group-hover:opacity-10 group-hover:scale-125 transition-all duration-500 text-6xl text-orange-500">
                              <i class="pi pi-chart-line"></i>
                          </div>
                          <div class="w-10 h-10 lg:w-14 lg:h-14 rounded-xl lg:rounded-2xl bg-orange-50 text-orange-600 flex items-center justify-center text-xl lg:text-2xl mb-3 lg:mb-4 group-hover:bg-orange-500 group-hover:text-white transition-all shadow-sm">
                              <i class="pi pi-chart-line"></i>
                          </div>
                          <h3 class="text-sm lg:text-lg font-bold mb-0.5 lg:mb-1 group-hover:text-orange-600 transition-colors">系统监控</h3>
                          <p class="text-text-muted text-[10px] lg:text-sm line-clamp-1">实时监控服务器资源使用情况</p>
                      </div>
                 </div>

                 <!-- Security Settings -->
                 <div class="bg-surface-0 rounded-2xl lg:rounded-3xl shadow-sm border border-surface-100 p-5 lg:p-8 animate-fade-in-up" style="animation-delay: 0.3s">
                      <div class="flex items-center gap-3 lg:gap-4 mb-5 lg:mb-8 border-b border-surface-100 pb-4">
                          <div class="w-10 h-10 lg:w-12 lg:h-12 rounded-xl lg:rounded-2xl bg-surface-100 text-text-muted flex items-center justify-center text-lg lg:text-xl">
                              <i class="pi pi-shield"></i>
                          </div>
                          <div>
                              <h3 class="font-bold text-lg lg:text-xl text-text">安全中心</h3>
                              <p class="text-xs lg:text-sm text-text-muted">保护您的账户安全与隐私</p>
                          </div>
                      </div>

                      <div class="grid grid-cols-1 md:grid-cols-2 gap-3 lg:gap-4">
                          <Button 
                            label="修改登录密码" 
                            icon="pi pi-lock" 
                            severity="secondary" 
                            outlined 
                            class="!h-12 lg:!h-14 !justify-between !px-4 lg:!px-6 !rounded-lg lg:!rounded-xl !border-surface-200 hover:!border-primary hover:!text-primary hover:!bg-primary/5 transition-all !text-sm lg:!text-base"
                            @click="showChangePasswordDialog = true"
                        />
                          <Button 
                            label="退出当前账户" 
                            icon="pi pi-power-off" 
                            severity="danger" 
                            outlined 
                            class="!h-12 lg:!h-14 !justify-between !px-4 lg:!px-6 !rounded-lg lg:!rounded-xl opacity-80 hover:opacity-100 hover:!bg-red-50 transition-all !text-sm lg:!text-base"
                            @click="handleLogout"
                        />
                      </div>
                 </div>
            </div>
        </div>
    </div>
    
    <!-- Change Password Dialog -->
    <Dialog 
        v-model:visible="showChangePasswordDialog" 
        header="修改密码" 
        modal 
        class="w-full max-w-lg" 
        :pt="{ 
            root: { class: '!rounded-2xl !border-0 !shadow-2xl overflow-hidden' },
            header: { class: '!bg-surface-50 !py-4 !px-6 !border-b !border-surface-100' },
            content: { class: '!p-0' },
            footer: { class: '!p-6 !border-t !border-surface-100' },
            closeButton: { class: '!w-8 !h-8 !rounded-full hover:!bg-surface-200' }
        }"
    >
        <template #header>
            <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center text-primary">
                    <i class="pi pi-lock text-lg"></i>
                </div>
                <div>
                    <span class="font-bold text-lg block text-text">修改密码</span>
                    <span class="text-xs text-text-muted font-normal">为了您的账户安全，建议定期更换密码</span>
                </div>
            </div>
        </template>
        
        <div class="flex flex-col gap-6 p-8">
            <div class="flex flex-col gap-2">
                 <label for="oldPassword" class="font-medium text-sm text-text">当前密码</label>
                 <InputText type="password" id="oldPassword" v-model="passwordForm.oldPassword" class="!rounded-lg !py-3" placeholder="请输入当前使用的密码" toggleMask />
            </div>
             <div class="flex flex-col gap-2">
                 <label for="newPassword" class="font-medium text-sm text-text">新密码</label>
                 <Password id="newPassword" v-model="passwordForm.newPassword" toggleMask fluid promptLabel="请输入新密码" weakLabel="安全强度：弱" mediumLabel="安全强度：中" strongLabel="安全强度：强" :inputStyle="{borderRadius: '0.5rem', padding: '0.75rem'}">
                    <template #header>
                        <div class="font-semibold text-xs mb-2 text-text-muted uppercase tracking-wider">密码要求</div>
                    </template>
                    <template #footer>
                        <ul class="pl-4 mt-1 list-disc text-xs text-text-muted space-y-1">
                            <li>长度至少 6 个字符</li>
                            <li>不能与旧密码相同</li>
                        </ul>
                    </template>
                 </Password>
            </div>
             <div class="flex flex-col gap-2">
                 <label for="confirmPassword" class="font-medium text-sm text-text">确认新密码</label>
                 <InputText type="password" id="confirmPassword" v-model="passwordForm.confirmPassword" class="!rounded-lg !py-3" placeholder="请再次输入新密码" toggleMask />
            </div>
        </div>
        <template #footer>
            <div class="flex gap-3 w-full">
                <Button label="取消" text severity="secondary" @click="closePasswordDialog" class="flex-1 !rounded-xl !h-12 !font-medium" />
                <Button label="确认修改" icon="pi pi-check" @click="handleChangePassword" :loading="changingPassword" class="flex-1 !rounded-xl !h-12 !font-bold !bg-primary !border-primary" />
            </div>
        </template>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../store/user'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'

// PrimeVue
import Card from 'primevue/card'
import Avatar from 'primevue/avatar'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import Password from 'primevue/password'
import InputText from 'primevue/inputtext'
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'

const router = useRouter()
const userStore = useUserStore()
const toast = useToast()
const confirm = useConfirm()

const user = computed(() => userStore.user)
const deviceCount = ref(0)
const subscriptionCount = ref(0)

// Change password dialog
const showChangePasswordDialog = ref(false)
const changingPassword = ref(false)
const passwordForm = ref({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const userInitial = computed(() => {
  if (!user.value?.email) return '?'
  return user.value.email.charAt(0).toUpperCase()
})

const roleText = computed(() => {
  if (user.value?.role === 'admin') return '管理员'
  return '普通用户'
})

onMounted(async () => {
  await loadStats()
})

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

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })
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

const closePasswordDialog = () => {
  showChangePasswordDialog.value = false
  passwordForm.value = {
    oldPassword: '',
    newPassword: '',
    confirmPassword: ''
  }
}

const handleChangePassword = async () => {
  const { oldPassword, newPassword, confirmPassword } = passwordForm.value
  
  // Validate
  if (newPassword !== confirmPassword) {
    toast.add({ severity: 'error', summary: 'Error', detail: '两次输入的新密码不一致', life: 3000 })
    return
  }
  
  if (newPassword.length < 6) {
    toast.add({ severity: 'error', summary: 'Error', detail: '新密码长度至少为 6 位', life: 3000 })
    return
  }
  
  if (oldPassword === newPassword) {
    toast.add({ severity: 'error', summary: 'Error', detail: '新密码不能与旧密码相同', life: 3000 })
    return
  }
  
  changingPassword.value = true
  
  try {
    const token = localStorage.getItem('token')
    const res = await fetch('/api/user/change-password', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({
        old_password: oldPassword,
        new_password: newPassword
      })
    })
    
    const data = await res.json()
    
    if (data.code === 0) {
      toast.add({ severity: 'success', summary: 'Success', detail: '密码修改成功！请重新登录', life: 2000 })
      closePasswordDialog()
      setTimeout(() => {
          userStore.logout()
          router.push('/login')
      }, 2000)
    } else {
      toast.add({ severity: 'error', summary: 'Error', detail: data.message || '密码修改失败', life: 3000 })
    }
  } catch (e) {
    toast.add({ severity: 'error', summary: 'Error', detail: '网络错误，请稍后重试', life: 3000 })
  } finally {
    changingPassword.value = false
  }
}
</script>

<style scoped>
.animate-fade-in-up {
    animation: fadeInUp 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards;
    opacity: 0;
    transform: translateY(20px);
}

@keyframes fadeInUp {
    to {
        opacity: 1;
        transform: translateY(0);
    }
}
</style>
