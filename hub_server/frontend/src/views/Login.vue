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
                <h1 class="text-4xl font-extrabold text-gray-900 mb-2 text-left tracking-tight">登录</h1>
                <p class="text-gray-400 text-base text-left font-medium">欢迎回来，请输入您的账号详情</p>
            </div>

            <form @submit.prevent="handleLogin" class="flex flex-col gap-6">
                <!-- Email -->
                <div class="space-y-1">
                    <label class="text-sm font-semibold text-gray-700 ml-1">邮箱</label>
                    <IconField iconPosition="left">
                        <InputIcon class="pi pi-envelope text-gray-400 !absolute !top-1/2 !-translate-y-1/2" />
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
                        <InputIcon class="pi pi-lock z-10 text-gray-400 !absolute !top-1/2 !-translate-y-1/2" />
                        <Password 
                            id="password" 
                            v-model="password" 
                            class="w-full" 
                            :feedback="false" 
                            toggleMask 
                            placeholder="••••••••" 
                            size="large"
                            inputClass="!pl-10 w-full !bg-gray-50 !border-gray-200 focus:!bg-white focus:!border-emerald-500 hover:!border-emerald-400 text-gray-800 h-12 transition-all duration-200 rounded-xl"
                            :toggleMaskIcon="false"
                        />
                    </IconField>
                </div>

                <div class="flex items-center">
                    <div class="flex items-center gap-2">
                            <Checkbox v-model="rememberMe" binary inputId="remember" class="!border-emerald-500" />
                            <label for="remember" class="text-sm text-gray-600 cursor-pointer select-none font-medium">记住我</label>
                    </div>
                </div>

                <div v-if="userStore.error" class="w-full">
                        <Message severity="error" :closable="false" class="w-full justify-start rounded-xl">{{ userStore.error }}</Message>
                </div>

                <Button 
                    type="submit" 
                    label="立即登录" 
                    :loading="userStore.loading" 
                    size="large" 
                    class="w-full h-12 font-bold text-lg !bg-gradient-to-r from-emerald-500 to-teal-500 hover:from-emerald-600 hover:to-teal-600 !border-0 shadow-lg shadow-emerald-500/30 rounded-xl transition-all hover:scale-[1.02] active:scale-[0.98]" 
                />
            </form>

            <div class="text-center text-sm text-gray-500 mt-8 font-medium">
                还没有账号? 
                <router-link to="/register" class="font-bold text-emerald-600 ml-1 hover:text-emerald-700 transition-colors">
                    免费注册
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
const rememberMe = ref(false)
const userStore = useUserStore()
const router = useRouter()
const toast = useToast()

const handleLogin = async () => {
  try {
    const success = await userStore.login(email.value, password.value)
    if (success) {
        toast.add({ severity: 'success', summary: 'Welcome', detail: '登录成功', life: 2000 })
        router.push('/')
    }
  } catch (e) {
      toast.add({ severity: 'error', summary: 'Error', detail: '登录失败', life: 3000 })
  }
}
</script>
