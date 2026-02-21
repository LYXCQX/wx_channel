<template>
    <div class="min-h-screen bg-bg p-4 lg:p-10 font-sans text-text">
    <!-- Header Controls -->
    <header class="flex flex-col md:flex-row gap-4 justify-between items-start md:items-center mb-6">
      <div class="flex items-center gap-1 bg-surface-0 rounded-xl border border-surface-100 p-1 overflow-x-auto max-w-full scrollbar-hide">
        <!-- Custom Tab Filter -->
        <button
            v-for="opt in statusOptions" :key="opt.value"
            @click="statusFilter = opt.value"
            class="px-3 py-1.5 lg:px-4 lg:py-2 text-xs font-medium rounded-lg transition-all duration-200 cursor-pointer whitespace-nowrap"
            :class="statusFilter === opt.value 
                ? 'bg-primary text-primary-contrast shadow-sm' 
                : 'text-text-muted hover:text-text hover:bg-surface-50'"
        >
            {{ opt.label }}
        </button>
      </div>
      <div class="flex items-center gap-3 w-full md:w-auto justify-between md:justify-end">
        <!-- Auto-refresh Toggle -->
        <div class="flex items-center gap-2.5 px-3 py-2 lg:px-4 lg:py-2.5 bg-surface-0 rounded-xl border border-surface-100 select-none hover:border-surface-200 transition-colors">
            <i class="pi pi-sync text-xs lg:text-sm transition-all" :class="autoRefresh ? 'pi-spin text-primary' : 'text-text-muted'"></i>
            <span class="text-xs font-medium" :class="autoRefresh ? 'text-primary' : 'text-text-muted'">自动刷新</span>
            <ToggleSwitch v-model="autoRefresh" class="!w-8 !h-4" />
        </div>
        <Button 
            :label="taskStore.loading ? '加载中...' : '刷新'" 
            icon="pi pi-refresh" 
            :loading="taskStore.loading"
            rounded
            outlined
            size="small"
            class="!text-xs lg:!text-sm"
            @click="taskStore.fetchTasks(taskStore.page)"
        />
      </div>
    </header>

    <!-- Stats Cards -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-3 lg:gap-4 mb-6">
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-6 border border-surface-100 shadow-sm flex items-center justify-between hover:shadow-md transition-shadow">
            <div>
                <p class="text-text-muted text-xs lg:text-sm font-medium uppercase tracking-wider mb-1">全部任务</p>
                <div class="text-2xl lg:text-3xl font-bold text-text">{{ taskStore.total }}</div>
            </div>
            <div class="w-10 h-10 lg:w-12 lg:h-12 rounded-full bg-blue-50 text-blue-500 flex items-center justify-center">
                <i class="pi pi-list-check text-lg lg:text-xl"></i>
            </div>
        </div>
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-6 border border-surface-100 shadow-sm flex items-center justify-between hover:shadow-md transition-shadow">
             <div>
                <p class="text-text-muted text-xs lg:text-sm font-medium uppercase tracking-wider mb-1">成功</p>
                <div class="text-2xl lg:text-3xl font-bold text-green-500">{{ successCount }}</div>
            </div>
            <div class="w-10 h-10 lg:w-12 lg:h-12 rounded-full bg-green-50 text-green-500 flex items-center justify-center">
                <i class="pi pi-check-circle text-lg lg:text-xl"></i>
            </div>
        </div>
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-6 border border-surface-100 shadow-sm flex items-center justify-between hover:shadow-md transition-shadow">
             <div>
                <p class="text-text-muted text-xs lg:text-sm font-medium uppercase tracking-wider mb-1">失败</p>
                <div class="text-2xl lg:text-3xl font-bold text-red-500">{{ failedCount }}</div>
            </div>
            <div class="w-10 h-10 lg:w-12 lg:h-12 rounded-full bg-red-50 text-red-500 flex items-center justify-center">
                <i class="pi pi-times-circle text-lg lg:text-xl"></i>
            </div>
        </div>
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-6 border border-surface-100 shadow-sm flex items-center justify-between hover:shadow-md transition-shadow">
             <div>
                <p class="text-text-muted text-xs lg:text-sm font-medium uppercase tracking-wider mb-1">等待 / 超时</p>
                <div class="text-2xl lg:text-3xl font-bold text-amber-500">{{ pendingCount }}</div>
            </div>
            <div class="w-10 h-10 lg:w-12 lg:h-12 rounded-full bg-amber-50 text-amber-500 flex items-center justify-center">
                <i class="pi pi-clock text-lg lg:text-xl"></i>
            </div>
        </div>
    </div>

    <!-- Table Card -->
    <div class="bg-surface-0 rounded-2xl border border-surface-100 shadow-sm overflow-hidden">
        <DataTable 
            :value="filteredTasks" 
            :loading="taskStore.loading"
            stripedRows 
            tableStyle="min-width: 50rem"
            class="p-datatable-sm text-xs lg:text-sm"
            :rowHover="true"
            responsiveLayout="scroll"
        >
            <template #empty>
                <div class="flex flex-col items-center justify-center p-12 text-text-muted">
                    <i class="pi pi-inbox text-4xl mb-3 text-surface-300"></i>
                    <p class="font-medium">暂无任务数据</p>
                    <p class="text-sm mt-1">任务将在系统操作后自动出现</p>
                </div>
            </template>
            
            <Column field="id" header="ID" style="width: 90px">
                <template #body="slotProps">
                    <span class="font-mono text-xs px-2 py-1 bg-surface-100 rounded-lg text-text-muted">#{{ slotProps.data.id }}</span>
                </template>
            </Column>
            
            <Column field="type" header="类型" style="width: 140px">
                <template #body="slotProps">
                    <div class="flex items-center gap-2">
                        <i :class="getTypeIcon(slotProps.data.type)" class="text-primary text-sm"></i>
                        <Tag :value="getTypeLabel(slotProps.data.type)" severity="secondary" rounded class="!text-xs"></Tag>
                    </div>
                </template>
            </Column>
            
            <Column field="node_id" header="执行节点">
                <template #body="slotProps">
                    <div class="flex items-center gap-2">
                        <div class="w-2 h-2 rounded-full bg-green-400 shrink-0"></div>
                        <span class="font-mono text-xs truncate max-w-48" :title="slotProps.data.node_id">{{ slotProps.data.node_id }}</span>
                    </div>
                </template>
            </Column>
            
            <Column field="status" header="状态" style="width: 120px">
                <template #body="slotProps">
                    <div class="flex items-center gap-2">
                        <i :class="getStatusIcon(slotProps.data.status)" class="text-sm"></i>
                        <Tag 
                            :value="getStatusLabel(slotProps.data.status)" 
                            :severity="getStatusSeverity(slotProps.data.status)" 
                            rounded
                            class="!text-xs"
                        ></Tag>
                    </div>
                </template>
            </Column>
            
            <Column field="created_at" header="创建时间" style="width: 180px">
                <template #body="slotProps">
                    <div class="flex flex-col">
                        <span class="text-xs text-text">{{ formatDate(slotProps.data.created_at) }}</span>
                        <span class="text-[10px] text-text-muted">{{ formatTimeAgo(slotProps.data.created_at) }}</span>
                    </div>
                </template>
            </Column>
            
            <Column header="操作" style="width: 100px" frozen alignFrozen="right">
                <template #body="slotProps">
                    <div class="flex gap-1">
                        <Button 
                            icon="pi pi-eye" 
                            text 
                            rounded 
                            severity="secondary" 
                            size="small"
                            aria-label="View" 
                            v-tooltip.top="'查看详情'"
                            @click="showDetail(slotProps.data)" 
                        />
                        <Button 
                            icon="pi pi-copy" 
                            text 
                            rounded 
                            severity="secondary" 
                            size="small"
                            aria-label="Copy ID" 
                            v-tooltip.top="'复制ID'"
                            @click="copyTaskId(slotProps.data.id)" 
                        />
                    </div>
                </template>
            </Column>
        </DataTable>

        <!-- Paginator -->
        <div class="border-t border-surface-100 px-4 py-2 flex items-center justify-between">
            <span class="text-xs text-text-muted">
                共 {{ taskStore.total }} 条记录，第 {{ taskStore.page }} / {{ Math.ceil(taskStore.total / taskStore.pageSize) || 1 }} 页
            </span>
            <Paginator 
                :rows="taskStore.pageSize" 
                :totalRecords="taskStore.total" 
                :first="(taskStore.page - 1) * taskStore.pageSize"
                @page="onPageChange"
                template="PrevPageLink CurrentPageReport NextPageLink"
                class="!p-0 !bg-transparent"
            ></Paginator>
        </div>
    </div>

    <!-- Task Detail Modal -->
    <Dialog 
        v-model:visible="visible" 
        modal 
        :header="''" 
        :style="{ width: '44rem' }"
        :breakpoints="{ '1199px': '75vw', '575px': '90vw' }"
        :pt="{
            root: { class: '!rounded-2xl overflow-hidden !border-0' },
            header: { class: '!p-0 !min-h-0' },
            content: { class: '!p-0' },
            closeButton: { class: '!absolute !right-4 !top-4 !z-20 !text-white/70 hover:!text-white !bg-white/10 !rounded-full !w-8 !h-8' }
        }"
    >
        <!-- Loading -->
        <div v-if="detailLoading" class="flex flex-col items-center justify-center p-16">
            <div class="w-12 h-12 rounded-full border-3 border-surface-200 border-t-primary animate-spin mb-4"></div>
            <p class="text-sm text-text-muted">正在加载任务详情...</p>
        </div>

        <div v-else-if="selectedTask">
            <!-- Colored Header Banner -->
            <div class="relative px-6 pt-6 pb-5" :class="getStatusBannerClass(selectedTask.status)">
                <div class="flex items-start justify-between">
                    <div class="flex items-center gap-3">
                        <div class="w-10 h-10 rounded-xl bg-white/20 flex items-center justify-center">
                            <i :class="getStatusIcon(selectedTask.status)" class="text-lg !text-white"></i>
                        </div>
                        <div>
                            <div class="flex items-center gap-2">
                                <span class="text-white font-bold text-lg">任务 #{{ selectedTask.id }}</span>
                                <span class="text-white/80 text-xs px-2 py-0.5 bg-white/15 rounded-full">{{ getTypeLabel(selectedTask.type) }}</span>
                            </div>
                            <p class="text-white/70 text-xs mt-0.5">{{ getStatusLabel(selectedTask.status) }} · {{ formatTimeAgo(selectedTask.created_at) }}</p>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Body Content -->
            <div class="p-6 space-y-5">
                <!-- Info Grid -->
                <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
                    <div class="bg-surface-50 rounded-xl px-4 py-3 border border-surface-100">
                        <div class="flex items-center gap-1.5 mb-1">
                            <i class="pi pi-server text-[10px] text-text-muted"></i>
                            <p class="text-[10px] text-text-muted uppercase tracking-wider font-semibold">执行节点</p>
                        </div>
                        <p class="font-mono text-xs text-text break-all leading-relaxed">{{ selectedTask.node_id }}</p>
                    </div>
                    <div class="bg-surface-50 rounded-xl px-4 py-3 border border-surface-100">
                        <div class="flex items-center gap-1.5 mb-1">
                            <i class="pi pi-calendar text-[10px] text-text-muted"></i>
                            <p class="text-[10px] text-text-muted uppercase tracking-wider font-semibold">创建时间</p>
                        </div>
                        <p class="text-xs text-text">{{ formatDate(selectedTask.created_at) }}</p>
                    </div>
                    <div class="bg-surface-50 rounded-xl px-4 py-3 border border-surface-100">
                        <div class="flex items-center gap-1.5 mb-1">
                            <i class="pi pi-tag text-[10px] text-text-muted"></i>
                            <p class="text-[10px] text-text-muted uppercase tracking-wider font-semibold">任务类型</p>
                        </div>
                        <p class="text-xs text-text font-medium">{{ getTypeLabel(selectedTask.type) }}</p>
                    </div>
                </div>

                <!-- Payload Section -->
                <div class="rounded-xl border border-surface-100 overflow-hidden">
                    <div class="flex items-center justify-between px-4 py-2.5 bg-surface-50 border-b border-surface-100">
                        <div class="flex items-center gap-2">
                            <i class="pi pi-send text-xs text-primary"></i>
                            <span class="text-xs font-semibold text-text">Payload</span>
                            <span class="text-[10px] text-text-muted">输入参数</span>
                        </div>
                        <button 
                            @click="copyText(formatJson(selectedTask.payload))" 
                            class="flex items-center gap-1 px-2 py-1 text-[10px] text-text-muted hover:text-primary hover:bg-primary/5 rounded-md transition-colors cursor-pointer"
                        >
                            <i class="pi pi-copy text-[10px]"></i>
                            复制
                        </button>
                    </div>
                    <div class="bg-surface-900 text-surface-100 p-4 font-mono text-xs overflow-auto max-h-52 whitespace-pre-wrap leading-relaxed scrollbar-hide">{{ formatJson(selectedTask.payload) }}</div>
                </div>
                
                <!-- Result Section -->
                <div class="rounded-xl border border-surface-100 overflow-hidden">
                    <div class="flex items-center justify-between px-4 py-2.5 bg-surface-50 border-b border-surface-100">
                        <div class="flex items-center gap-2">
                            <i class="pi pi-check-circle text-xs text-green-500"></i>
                            <span class="text-xs font-semibold text-text">Result</span>
                            <span class="text-[10px] text-text-muted">执行结果</span>
                        </div>
                        <button 
                            @click="copyText(formatJson(selectedTask.result))" 
                            class="flex items-center gap-1 px-2 py-1 text-[10px] text-text-muted hover:text-primary hover:bg-primary/5 rounded-md transition-colors cursor-pointer"
                        >
                            <i class="pi pi-copy text-[10px]"></i>
                            复制
                        </button>
                    </div>
                    <div class="bg-surface-900 text-surface-100 p-4 font-mono text-xs overflow-auto max-h-52 whitespace-pre-wrap leading-relaxed scrollbar-hide">{{ formatJson(selectedTask.result) }}</div>
                </div>

                <!-- Error Section -->
                <div v-if="selectedTask.error" class="rounded-xl border border-red-200 overflow-hidden">
                    <div class="flex items-center gap-2 px-4 py-2.5 bg-red-50 border-b border-red-200">
                        <i class="pi pi-exclamation-triangle text-xs text-red-500"></i>
                        <span class="text-xs font-semibold text-red-600">Error</span>
                        <span class="text-[10px] text-red-400">错误信息</span>
                    </div>
                    <div class="bg-red-50/50 text-red-600 p-4 text-xs font-mono leading-relaxed whitespace-pre-wrap">{{ selectedTask.error }}</div>
                </div>
            </div>
        </div>
    </Dialog>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { useTaskStore } from '../store/task'
import { formatTime } from '../utils/format'
import { useToast } from 'primevue/usetoast'
import axios from 'axios'



const taskStore = useTaskStore()
const toast = useToast()
const selectedTask = ref(null)
const detailLoading = ref(false)
const visible = ref(false)
const statusFilter = ref(null)
const autoRefresh = ref(false)
let refreshInterval = null

const statusOptions = [
    { label: '全部', value: null },
    { label: '成功', value: 'success' },
    { label: '失败', value: 'failed' },
    { label: '等待', value: 'pending' },
    { label: '超时', value: 'timeout' },
]

// Computed stats
const successCount = computed(() => taskStore.tasks.filter(t => t.status === 'success').length)
const failedCount = computed(() => taskStore.tasks.filter(t => t.status === 'failed').length)
const pendingCount = computed(() => taskStore.tasks.filter(t => t.status === 'pending' || t.status === 'timeout').length)

const filteredTasks = computed(() => {
    if (!statusFilter.value) return taskStore.tasks
    return taskStore.tasks.filter(t => t.status === statusFilter.value)
})

onMounted(() => {
  taskStore.fetchTasks()
})

// Auto-refresh watcher
const startAutoRefresh = () => {
    refreshInterval = setInterval(() => {
        taskStore.fetchTasks(taskStore.page)
    }, 5000)
}
const stopAutoRefresh = () => {
    if (refreshInterval) {
        clearInterval(refreshInterval)
        refreshInterval = null
    }
}

// Watch autoRefresh toggle
import { watch } from 'vue'
watch(autoRefresh, (val) => {
    if (val) startAutoRefresh()
    else stopAutoRefresh()
})

onUnmounted(() => {
    stopAutoRefresh()
})

const onPageChange = (event) => {
    const newPage = event.page + 1
    taskStore.fetchTasks(newPage)
}

const getStatusSeverity = (status) => {
    switch (status) {
        case 'success': return 'success'
        case 'pending': return 'warn'
        case 'failed': return 'danger'
        case 'running': return 'info'
        case 'timeout': return 'warn'
        default: return 'secondary'
    }
}

const getStatusIcon = (status) => {
    switch (status) {
        case 'success': return 'pi pi-check-circle text-green-500'
        case 'pending': return 'pi pi-clock text-amber-500'
        case 'failed': return 'pi pi-times-circle text-red-500'
        case 'running': return 'pi pi-spin pi-spinner text-blue-500'
        case 'timeout': return 'pi pi-exclamation-circle text-amber-500'
        default: return 'pi pi-question-circle text-surface-400'
    }
}

const getStatusLabel = (status) => {
    switch (status) {
        case 'success': return '成功'
        case 'pending': return '等待中'
        case 'failed': return '失败'
        case 'running': return '运行中'
        case 'timeout': return '超时'
        default: return status
    }
}

const getTypeIcon = (type) => {
    switch (type) {
        case 'api_call': return 'pi pi-cloud'
        case 'download': return 'pi pi-download'
        case 'upload': return 'pi pi-upload'
        default: return 'pi pi-cog'
    }
}

const getTypeLabel = (type) => {
    switch (type) {
        case 'api_call': return 'API 调用'
        case 'download': return '下载'
        case 'upload': return '上传'
        default: return type
    }
}

const showDetail = async (task) => {
    visible.value = true
    detailLoading.value = true
    selectedTask.value = null
    try {
        const res = await axios.get(`/api/tasks/detail?id=${task.id}`)
        selectedTask.value = res.data
    } catch (err) {
        console.error("Fetch detail failed: ", err)
    } finally {
        detailLoading.value = false
    }
}

const formatJson = (str) => {
    if (!str) return '-'
    try {
        return JSON.stringify(JSON.parse(str), null, 2)
    } catch (e) {
        return str
    }
}

const formatDate = (dateStr) => {
    if (!dateStr) return '-'
    const d = new Date(dateStr)
    return d.toLocaleString('zh-CN', { 
        year: 'numeric', month: '2-digit', day: '2-digit',
        hour: '2-digit', minute: '2-digit', second: '2-digit'
    })
}

const formatTimeAgo = (dateStr) => {
    if (!dateStr) return ''
    const d = new Date(dateStr)
    const now = new Date()
    const diff = Math.floor((now - d) / 1000)
    
    if (diff < 60) return `${diff} 秒前`
    if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
    if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`
    return `${Math.floor(diff / 86400)} 天前`
}

const copyTaskId = (id) => {
    navigator.clipboard.writeText(String(id))
    toast.add({ severity: 'success', summary: '已复制', detail: `任务 #${id} 已复制到剪贴板`, life: 2000 })
}

const copyText = (text) => {
    if (!text || text === '-') return
    navigator.clipboard.writeText(text)
    toast.add({ severity: 'success', summary: '已复制', detail: '内容已复制到剪贴板', life: 2000 })
}

const getStatusBannerClass = (status) => {
    switch (status) {
        case 'success': return 'bg-gradient-to-r from-green-600 to-emerald-500'
        case 'failed': return 'bg-gradient-to-r from-red-600 to-rose-500'
        case 'pending': return 'bg-gradient-to-r from-amber-600 to-yellow-500'
        case 'running': return 'bg-gradient-to-r from-blue-600 to-cyan-500'
        case 'timeout': return 'bg-gradient-to-r from-orange-600 to-amber-500'
        default: return 'bg-gradient-to-r from-surface-600 to-surface-500'
    }
}
</script>

<style scoped>
/* DataTable Styling Overrides */
:deep(.p-datatable) {
    border-radius: 0;
}
:deep(.p-datatable-thead > tr > th) {
    background-color: var(--color-surface-50);
    color: var(--p-text-muted-color);
    font-weight: 600;
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    padding: 0.875rem 1rem;
    border-bottom: 1px solid var(--p-surface-200);
    border-top: none;
}
:deep(.p-datatable-tbody > tr > td) {
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--p-surface-100);
    font-size: 0.875rem;
}
:deep(.p-datatable-tbody > tr:last-child > td) {
    border-bottom: none;
}
:deep(.p-datatable-tbody > tr:hover > td) {
    background-color: var(--color-surface-50) !important;
}
:deep(.p-datatable .p-datatable-tbody > tr.p-datatable-row-odd) {
    background-color: transparent;
}
:deep(.p-datatable .p-datatable-tbody > tr.p-datatable-row-odd:hover > td) {
    background-color: var(--color-surface-50) !important;
}

/* Paginator Overrides */
:deep(.p-paginator) {
    border: none;
    background: transparent;
    padding: 0;
}


/* ToggleSwitch Overrides */
:deep(.p-toggleswitch) {
    --p-toggleswitch-border-width: 0px;
    --p-toggleswitch-border-color: transparent;
    --p-toggleswitch-background: var(--p-surface-300);
    --p-toggleswitch-transition-duration: 0.2s;
    --p-toggleswitch-border-radius: 30px;
    --p-toggleswitch-shadow: none;
    
    --p-toggleswitch-handle-background: var(--p-surface-0);
    --p-toggleswitch-handle-color: var(--p-text-color);
    --p-toggleswitch-handle-size: 0.75rem; 
    --p-toggleswitch-gap: 2px;
    --p-toggleswitch-handle-border-radius: 50%;
    --p-toggleswitch-slide-duration: 0.2s;
    
    position: relative;
    display: inline-block;
}

:deep(.p-toggleswitch.p-toggleswitch-checked) {
    --p-toggleswitch-background: var(--p-primary-color);
}

:deep(.p-toggleswitch-slider) {
    cursor: pointer;
    width: 100%;
    height: 100%;
    border-width: var(--p-toggleswitch-border-width);
    border-style: solid;
    border-color: var(--p-toggleswitch-border-color);
    background: var(--p-toggleswitch-background);
    transition: background var(--p-toggleswitch-transition-duration), color var(--p-toggleswitch-transition-duration), border-color var(--p-toggleswitch-transition-duration), outline-color var(--p-toggleswitch-transition-duration), box-shadow var(--p-toggleswitch-transition-duration);
    border-radius: var(--p-toggleswitch-border-radius);
    outline-color: transparent;
    box-shadow: var(--p-toggleswitch-shadow);
    position: relative;
}

:deep(.p-toggleswitch-handle) {
    position: absolute;
    top: 50%;
    display: flex;
    justify-content: center;
    align-items: center;
    background: var(--p-toggleswitch-handle-background);
    color: var(--p-toggleswitch-handle-color);
    width: var(--p-toggleswitch-handle-size);
    height: var(--p-toggleswitch-handle-size);
    inset-inline-start: var(--p-toggleswitch-gap);
    margin-block-start: calc(-1 * calc(var(--p-toggleswitch-handle-size) / 2));
    border-radius: var(--p-toggleswitch-handle-border-radius);
    transition: background var(--p-toggleswitch-transition-duration), color var(--p-toggleswitch-transition-duration), inset-inline-start var(--p-toggleswitch-slide-duration), box-shadow var(--p-toggleswitch-slide-duration);
}

:deep(.p-toggleswitch.p-toggleswitch-checked .p-toggleswitch-handle) {
    inset-inline-start: calc(100% - var(--p-toggleswitch-handle-size) - var(--p-toggleswitch-gap));
}

.dark :deep(.p-toggleswitch) {
     --p-toggleswitch-background: var(--p-surface-700);
}
</style>

