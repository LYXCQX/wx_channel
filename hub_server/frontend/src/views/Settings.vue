<template>
  <div class="min-h-screen bg-bg p-4 lg:p-12 font-sans text-text">
    <Toast />
    
    <header class="mb-6 lg:mb-8">
      <h2 class="font-serif font-bold text-xl lg:text-2xl text-text mb-1 lg:mb-2">个人设置</h2>
      <p class="text-text-muted text-xs lg:text-sm">管理您的账户信息和偏好设置</p>
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
            
            <!-- Profile Section -->
            <Panel header="个人资料" id="profile" v-if="activeSection === 'profile'">
                <div class="flex flex-col gap-6">
                    <div class="flex flex-col md:flex-row items-center md:items-start gap-4 md:gap-6 text-center md:text-left">
                        <div class="relative group">
                            <div class="w-20 h-20 lg:w-24 lg:h-24 rounded-full bg-surface-100 flex items-center justify-center text-surface-400 overflow-hidden ring-4 ring-surface-50">
                                <i v-if="!userProfile.avatar" class="pi pi-user text-3xl lg:text-4xl"></i>
                                <img v-else :src="userProfile.avatar" class="w-full h-full object-cover" />
                            </div>
                            <Button icon="pi pi-camera" rounded severity="secondary" size="small" class="absolute bottom-0 right-0 shadow-md transform scale-90 lg:scale-100" />
                        </div>
                        <div class="flex-1">
                            <h3 class="font-bold text-lg text-text">{{ userProfile.nickname || '用户' }}</h3>
                            <p class="text-text-muted text-xs lg:text-sm break-all">ID: {{ userProfile.id }}</p>
                            <div class="flex justify-center md:justify-start">
                                <Tag :value="userProfile.role" class="mt-2 !text-xs !px-2" :severity="userProfile.role === 'admin' ? 'warn' : 'info'"></Tag>
                            </div>
                        </div>
                    </div>

                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div class="flex flex-col gap-2">
                            <label class="font-semibold text-sm">昵称</label>
                            <InputText v-model="userProfile.nickname" class="!py-2.5" />
                        </div>
                        <div class="flex flex-col gap-2">
                            <label class="font-semibold text-sm">邮箱</label>
                            <InputText v-model="userProfile.email" disabled class="bg-surface-50 !py-2.5" />
                        </div>
                    </div>

                    <div class="flex justify-end pt-2">
                        <Button label="保存修改" icon="pi pi-check" @click="saveProfile" :loading="loading" class="w-full md:w-auto" />
                    </div>
                </div>
            </Panel>

            <!-- Security Section -->
            <Panel header="账号安全" id="security" v-if="activeSection === 'security'">
                <div class="flex flex-col gap-6 lg:gap-8">
                    <!-- Password Change -->
                    <div class="flex flex-col gap-4">
                        <h3 class="font-bold text-sm lg:text-base">修改密码</h3>
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div class="flex flex-col gap-2">
                                <label class="font-semibold text-sm">当前密码</label>
                                <Password v-model="security.currentPassword" toggleMask :feedback="false" inputClass="!w-full !py-2.5" class="w-full" />
                            </div>
                        </div>
                         <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div class="flex flex-col gap-2">
                                <label class="font-semibold text-sm">新密码</label>
                                <Password v-model="security.newPassword" toggleMask inputClass="!w-full !py-2.5" class="w-full" />
                            </div>
                            <div class="flex flex-col gap-2">
                                <label class="font-semibold text-sm">确认新密码</label>
                                <Password v-model="security.confirmPassword" toggleMask :feedback="false" inputClass="!w-full !py-2.5" class="w-full" />
                            </div>
                        </div>
                         <div class="flex justify-end pt-2">
                            <Button label="更新密码" severity="warn" icon="pi pi-lock" @click="updatePassword" :loading="loading" class="w-full md:w-auto" />
                        </div>
                    </div>

                    <hr class="border-surface-200" />

                    <!-- API Token -->
                    <div class="flex flex-col gap-4">
                         <div class="flex justify-between items-start">
                             <div>
                                <h3 class="font-bold text-sm lg:text-base">API 访问令牌</h3>
                                <p class="text-text-muted text-xs lg:text-sm mt-1">用于程序化访问 Hub API 的密钥。</p>
                             </div>
                             <Button label="重新生成" severity="danger" text icon="pi pi-refresh" @click="regenerateToken" size="small" />
                         </div>
                         
                         <div class="bg-surface-900 text-surface-50 p-3 lg:p-4 rounded-xl font-mono text-xs lg:text-sm break-all relative group">
                             {{ apiToken || '********************************' }}
                             <Button icon="pi pi-copy" class="absolute top-1 right-1 lg:top-2 lg:right-2 opacity-100 lg:opacity-0 group-hover:opacity-100 transition-opacity" size="small" text severity="secondary" @click="copyApiToken" />
                         </div>
                    </div>
                </div>
            </Panel>

            <!-- Preferences Section -->
            <Panel header="偏好设置" id="preferences" v-if="activeSection === 'preferences'">
                <div class="flex flex-col gap-4 lg:gap-6">
                     <div class="flex items-center justify-between p-2 lg:p-0">
                         <div>
                             <h3 class="font-bold text-sm lg:text-base">深色模式</h3>
                             <p class="text-text-muted text-xs lg:text-sm">切换应用程序的明暗主题</p>
                         </div>
                         <ToggleSwitch v-model="preferences.darkMode" @change="toggleTheme" />
                     </div>

                     <div class="flex items-center justify-between p-2 lg:p-0">
                         <div>
                             <h3 class="font-bold text-sm lg:text-base">邮件通知</h3>
                             <p class="text-text-muted text-xs lg:text-sm">接收关于任务完成和系统警告的邮件</p>
                         </div>
                         <ToggleSwitch v-model="preferences.notifications" />
                     </div>
                     
                     <div class="flex flex-col gap-2 pt-2">
                        <label class="font-bold text-sm lg:text-base">语言区域</label>
                        <SelectButton v-model="preferences.locale" :options="locales" optionLabel="name" optionValue="code" aria-labelledby="basic" class="w-full" />
                     </div>
                </div>
            </Panel>

        </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useToast } from 'primevue/usetoast'

// PrimeVue
import Panel from 'primevue/panel'
import Menu from 'primevue/menu'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import ToggleSwitch from 'primevue/toggleswitch'
import SelectButton from 'primevue/selectbutton'
import Tag from 'primevue/tag'
import Toast from 'primevue/toast'

const toast = useToast()
const loading = ref(false)
const activeSection = ref('profile')

const menuItems = ref([
    { label: '个人资料', icon: 'pi pi-user', command: () => activeSection.value = 'profile' },
    { label: '账号安全', icon: 'pi pi-shield', command: () => activeSection.value = 'security' },
    { label: '偏好设置', icon: 'pi pi-cog', command: () => activeSection.value = 'preferences' }
])

const userProfile = ref({
    id: 1,
    nickname: 'Admin User',
    email: 'admin@example.com',
    role: 'admin',
    avatar: ''
})

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

</script>
