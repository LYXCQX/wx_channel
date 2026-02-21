<template>
  <button 
    @click="toggleDarkMode" 
    class="relative w-10 h-10 rounded-xl flex items-center justify-center transition-all duration-300 hover:bg-surface-100 dark:hover:bg-surface-800 text-text-muted hover:text-primary focus:outline-none group"
    :title="isDark ? '切换亮色模式' : '切换暗色模式'"
  >
    <div class="relative w-5 h-5 overflow-hidden">
      <!-- Sun Icon -->
      <i class="pi pi-sun absolute inset-0 transition-all duration-500 transform" 
         :class="isDark ? 'translate-y-full opacity-0 rotate-90' : 'translate-y-0 opacity-100 rotate-0'"></i>
      
      <!-- Moon Icon -->
      <i class="pi pi-moon absolute inset-0 transition-all duration-500 transform" 
         :class="isDark ? 'translate-y-0 opacity-100 rotate-0' : '-translate-y-full opacity-0 -rotate-90'"></i>
    </div>
  </button>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const isDark = ref(false)

onMounted(() => {
  // Check local storage or system preference
  const savedTheme = localStorage.getItem('hub_theme')
  const systemDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  
  if (savedTheme === 'dark' || (!savedTheme && systemDark)) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  } else {
    isDark.value = false
    document.documentElement.classList.remove('dark')
  }
})

const toggleDarkMode = () => {
  isDark.value = !isDark.value
  
  if (isDark.value) {
    document.documentElement.classList.add('dark')
    localStorage.setItem('hub_theme', 'dark')
  } else {
    document.documentElement.classList.remove('dark')
    localStorage.setItem('hub_theme', 'light')
  }
}
</script>
