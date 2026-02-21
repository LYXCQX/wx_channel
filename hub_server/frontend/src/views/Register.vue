<template>
  <div class="min-h-screen bg-gradient-to-br from-emerald-400 via-teal-500 to-emerald-600 flex items-center justify-center p-6 relative overflow-hidden font-sans">
    <!-- Decorative Background Shapes -->
    <div class="absolute top-0 left-0 w-full h-full overflow-hidden z-0 pointer-events-none">
        <div class="absolute -top-[10%] -left-[10%] w-[60vh] h-[60vh] rounded-full bg-white/20 blur-3xl opacity-60"></div>
        <div class="absolute top-[20%] right-[10%] w-[40vh] h-[40vh] rounded-full bg-teal-300/20 blur-2xl opacity-60"></div>
        <div class="absolute -bottom-[10%] -right-[5%] w-[70vh] h-[70vh] rounded-full bg-emerald-800/20 blur-3xl opacity-60"></div>
    </div>

    <Toast />

    <div class="w-full max-w-[460px] relative z-10 animate-fade-in-up">
        <div class="bg-white rounded-[2rem] shadow-[0_20px_50px_-12px_rgba(0,0,0,0.15)] p-10 border border-white/50">
             <div class="mb-8">
                <h1 class="text-4xl font-extrabold text-gray-900 mb-2 text-left tracking-tight">创建账号</h1>
                <p class="text-gray-400 text-base text-left font-medium">加入 WX Channel Hub 开始您的分布式任务</p>
            </div>

            <form @submit.prevent="handleRegister" class="flex flex-col gap-6">
                <!-- Email -->
                <div class="space-y-1">
                    <label class="text-sm font-semibold text-gray-700 ml-1">邮箱</label>
                    <IconField iconPosition="left">
                        <InputIcon class="pi pi-envelope text-gray-400" />
                        <InputText 
                            id="email" 
                            v-model="email" 
                            class="w-full !pl-10 !bg-gray-50 !border-gray-200 focus:!bg-white focus:!border-emerald-500 hover:!border-emerald-400 text-gray-800 h-12 transition-all duration-200 rounded-xl" 
                            type="email" 
                            placeholder="name@example.com" 
                            size="large" 
                        />
                    </IconField>
                </div>

                <!-- Password -->
                <div class="space-y-1">
                    <label class="text-sm font-semibold text-gray-700 ml-1">密码</label>
                     <IconField iconPosition="left">
                        <InputIcon class="pi pi-lock z-10 text-gray-400" />
                        <Password 
                            id="password" 
                            v-model="password" 
                            class="w-full" 
                            toggleMask 
                            fluid 
                            :feedback="true" 
                            promptLabel="至少6位字符" 
                            weakLabel="太简单" 
                            mediumLabel="不错" 
                            strongLabel="完美" 
                            size="large"
                            placeholder="设置密码"
                             inputClass="!pl-10 w-full !bg-gray-50 !border-gray-200 focus:!bg-white focus:!border-emerald-500 hover:!border-emerald-400 text-gray-800 h-12 transition-all duration-200 rounded-xl"
                            :toggleMaskIcon="false"
                        >
                            <template #header>
                                <div class="font-semibold text-xs mb-2">安全建议</div>
                            </template>
                            <template #footer>
                                <ul class="pl-2 ml-2 mt-0 list-disc line-height-2 text-xs text-secondary">
                                    <li>使用大小写字母混合</li>
                                    <li>包含数字或符号</li>
                                </ul>
                            </template>
                        </Password>
                    </IconField>
                </div>

                <!-- Confirm Password -->
                <div class="space-y-1">
                    <label class="text-sm font-semibold text-gray-700 ml-1">确认密码</label>
                    <IconField iconPosition="left">
                        <InputIcon class="pi pi-lock-open z-10 text-gray-400" />
                        <Password 
                            id="confirmPassword" 
                            v-model="confirmPassword" 
                            class="w-full" 
                            :feedback="false" 
                            toggleMask 
                            fluid 
                            size="large" 
                            placeholder="再次输入密码"
                             inputClass="!pl-10 w-full !bg-gray-50 !border-gray-200 focus:!bg-white focus:!border-emerald-500 hover:!border-emerald-400 text-gray-800 h-12 transition-all duration-200 rounded-xl"
                            :toggleMaskIcon="false"
                        />
                    </IconField>
                </div>

                <div v-if="error || userStore.error" class="w-full">
                     <Message severity="error" :closable="false" class="w-full justify-start rounded-xl">{{ error || userStore.error }}</Message>
                </div>

                <Button 
                    type="submit" 
                    label="立即加入" 
                    :loading="userStore.loading" 
                    size="large" 
                    class="w-full h-12 font-bold text-lg !bg-gradient-to-r from-emerald-500 to-teal-500 hover:from-emerald-600 hover:to-teal-600 !border-0 shadow-lg shadow-emerald-500/30 rounded-xl transition-all hover:scale-[1.02] active:scale-[0.98]" 
                />
            </form>

            <div class="text-center text-sm text-gray-500 mt-8 font-medium">
                已有账号? 
                <router-link to="/login" class="font-bold text-emerald-600 ml-1 hover:text-emerald-700 transition-colors">
                     直接登录
                </router-link>
            </div>
        </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../store/user'
import { useToast } from 'primevue/usetoast'

const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const error = ref('')
const userStore = useUserStore()
const router = useRouter()
const toast = useToast()

const handleRegister = async () => {
  error.value = ''
  if (!email.value || !password.value) {
      error.value = '请输入邮箱和密码'
      return
  }
  if (password.value !== confirmPassword.value) {
    error.value = '两次输入的密码不一致'
    return
  }
  
  try {
     const success = await userStore.register(email.value, password.value)
      if (success) {
        toast.add({ severity: 'success', summary: 'Welcome', detail: '注册成功', life: 2000 })
        router.push('/')
      }
  } catch (e) {
      toast.add({ severity: 'error', summary: 'Error', detail: '注册失败', life: 3000 })
  }
}
</script>
