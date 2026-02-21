<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AuthLayout from './layouts/AuthLayout.vue'
import SidebarLayout from './layouts/SidebarLayout.vue'

const route = useRoute()
const layout = computed(() => {
  // 只有登录/注册页面使用 Auth 布局，其他所有页面都使用 Sidebar 布局
  if (route.meta.layout === 'Auth') return AuthLayout
  return SidebarLayout
})
</script>

<template>
  <component :is="layout">
    <router-view />
  </component>
</template>
