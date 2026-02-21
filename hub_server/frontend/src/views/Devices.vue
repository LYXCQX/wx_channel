<template>
    <div class="min-h-screen bg-bg p-4 lg:p-12 font-sans text-text">
    <Toast />
    <ConfirmDialog />
    
    <header class="flex justify-end items-center mb-4 lg:mb-8">
      <div>
        <Button 
            :label="loading ? '刷新中...' : '刷新列表'" 
            icon="pi pi-refresh" 
            :loading="loading"
            rounded
            size="small"
            class="!text-sm lg:!text-base"
            @click="refreshDevices"
        />
      </div>
    </header>

    <!-- Stats Cards -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-3 lg:gap-6 mb-6 lg:mb-8">
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-6 shadow-sm border border-surface-100 flex items-center justify-between">
            <div>
                <p class="text-text-muted text-xs lg:text-sm font-medium uppercase tracking-wider mb-1">总设备</p>
                <div class="text-2xl lg:text-3xl font-bold text-text">{{ devices.length }}</div>
            </div>
            <div class="w-10 h-10 lg:w-12 lg:h-12 rounded-full bg-blue-50 text-blue-500 flex items-center justify-center">
                <i class="pi pi-desktop text-lg lg:text-xl"></i>
            </div>
        </div>
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-6 shadow-sm border border-surface-100 flex items-center justify-between">
            <div>
                <p class="text-text-muted text-xs lg:text-sm font-medium uppercase tracking-wider mb-1">在线</p>
                <div class="text-2xl lg:text-3xl font-bold text-green-500">{{ onlineCount }}</div>
            </div>
            <div class="w-10 h-10 lg:w-12 lg:h-12 rounded-full bg-green-50 text-green-500 flex items-center justify-center">
                <i class="pi pi-wifi text-lg lg:text-xl"></i>
            </div>
        </div>
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-6 shadow-sm border border-surface-100 flex items-center justify-between">
            <div>
                <p class="text-text-muted text-xs lg:text-sm font-medium uppercase tracking-wider mb-1">离线</p>
                <div class="text-2xl lg:text-3xl font-bold text-text-muted">{{ offlineCount }}</div>
            </div>
            <div class="w-10 h-10 lg:w-12 lg:h-12 rounded-full flex items-center justify-center transition-colors bg-surface-100 text-text-muted">
                <i class="pi pi-power-off text-lg lg:text-xl"></i>
            </div>
        </div>
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-6 shadow-sm border border-surface-100 flex items-center justify-between">
            <div>
                <p class="text-text-muted text-xs lg:text-sm font-medium uppercase tracking-wider mb-1">锁定</p>
                <div class="text-2xl lg:text-3xl font-bold" :class="lockedCount > 0 ? 'text-amber-500' : 'text-text-muted'">{{ lockedCount }}</div>
            </div>
            <div class="w-10 h-10 lg:w-12 lg:h-12 rounded-full flex items-center justify-center transition-colors"
                 :class="lockedCount > 0 ? 'bg-amber-50 text-amber-500' : 'bg-surface-100 text-text-muted'">
                <i class="pi pi-lock text-lg lg:text-xl"></i>
            </div>
        </div>
    </div>

    <!-- Add Device & Filter -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 lg:gap-8 mb-6 lg:mb-8">
        <!-- Add Device Panel -->
        <div class="lg:col-span-1 bg-surface-0 rounded-2xl p-4 lg:p-6 shadow-sm border border-surface-100">
            <h3 class="font-bold text-base lg:text-lg text-text mb-2">添加新设备</h3>
            <p class="text-text-muted text-xs mb-4">在客户端运行绑定命令以连接。</p>
            
            <div v-if="bindToken" class="flex flex-col gap-3">
                <div class="bg-surface-50 p-2 lg:p-3 rounded-xl border border-surface-200 flex items-center justify-between">
                    <span class="font-mono text-lg lg:text-xl font-bold text-primary tracking-widest">{{ bindToken }}</span>
                     <Button icon="pi pi-copy" text rounded aria-label="Copy" @click="copyToken" />
                </div>
                <div class="text-[10px] lg:text-xs text-text-muted font-mono bg-surface-900 text-surface-50 p-2 rounded break-all">
                    > client bind {{ bindToken }}
                </div>
            </div>
            <Button v-else label="生成绑定码" icon="pi pi-key" class="w-full" size="small" @click="generateToken" />
        </div>

        <!-- Filter Panel -->
        <div class="lg:col-span-2 bg-surface-0 rounded-2xl p-4 lg:p-6 shadow-sm border border-surface-100 flex flex-col justify-center">
             <div class="flex flex-col md:flex-row gap-3 lg:gap-4">
                <IconField class="flex-1">
                    <InputIcon class="pi pi-search" />
                    <InputText v-model="filters['global'].value" placeholder="搜索 ID、主机名、IP..." class="w-full" size="small" />
                </IconField>
                <Select v-model="filterStatus" :options="statusOptions" optionLabel="label" optionValue="value" placeholder="全部状态" class="w-full md:w-48 !text-sm" showClear size="small" />
                <Select v-model="filterGroup" :options="deviceGroups" placeholder="全部分组" class="w-full md:w-48 !text-sm" showClear size="small" />
             </div>
        </div>
    </div>

    <!-- Device DataTable -->
    <div class="bg-surface-0 rounded-2xl p-4 lg:p-6 shadow-sm border border-surface-100">
        <DataTable 
            v-model:filters="filters"
            :value="devices" 
            :loading="loading"
            paginator 
            :rows="10" 
            :rowsPerPageOptions="[5, 10, 20, 50]"
            stripedRows 
            removableSort
            tableStyle="min-width: 60rem"
            :globalFilterFields="['id', 'hostname', 'display_name', 'ip']"
        >
            <template #empty>
                <div class="text-center p-8 text-text-muted">无设备数据</div>
            </template>

            <Column field="status" header="状态" sortable style="width: 100px">
                 <template #body="slotProps">
                    <Tag 
                        :value="slotProps.data.status === 'online' ? '在线' : '离线'" 
                        :severity="slotProps.data.status === 'online' ? 'success' : 'secondary'"
                        :icon="slotProps.data.status === 'online' ? 'pi pi-check-circle' : 'pi pi-times-circle'"
                        rounded
                    ></Tag>
                 </template>
            </Column>

            <Column field="display_name" header="名称/ID" sortable style="min-width: 200px">
                <template #body="{ data }">
                    <div class="flex flex-col">
                        <span class="font-bold text-text">{{ data.display_name || data.hostname || '未命名' }}</span>
                        <span class="text-xs font-mono text-text-muted">{{ data.id }}</span>
                    </div>
                </template>
            </Column>

            <Column field="ip" header="网络/版本" sortable style="min-width: 150px">
                 <template #body="{ data }">
                    <div class="flex flex-col">
                        <span class="text-sm font-mono">{{ data.ip || 'Unknown' }}</span>
                        <span class="text-xs text-text-muted">v{{ data.version || '...' }}</span>
                    </div>
                </template>
            </Column>

            <Column field="device_group" header="分组" sortable style="min-width: 120px">
                <template #body="{ data }">
                    <Tag v-if="data.device_group" :value="data.device_group" severity="info" rounded></Tag>
                    <span v-else class="text-text-muted text-xs">-</span>
                </template>
            </Column>

            <Column field="last_seen" header="最后在线" sortable style="min-width: 150px">
                <template #body="{ data }">
                    <span class="text-sm">{{ formatTime(data.last_seen) }}</span>
                </template>
            </Column>
            
            <Column header="操作" style="width: 160px">
                <template #body="{ data }">
                    <div class="flex gap-2">
                        <Button icon="pi pi-pencil" text rounded severity="secondary" size="small" @click="openRename(data)" v-tooltip="'重命名'" />
                        <Button icon="pi pi-folder" text rounded severity="secondary" size="small" @click="openGroup(data)" v-tooltip="'分组'" />
                        <Button icon="pi pi-info-circle" text rounded severity="info" size="small" @click="showHardwareInfo(data)" v-tooltip="'详情'" />
                        <Button icon="pi pi-cog" text rounded severity="secondary" size="small" @click="menu.toggle($event); selectedDevice = data" />
                    </div>
                </template>
            </Column>
        </DataTable>
    </div>

    <!-- Actions Menu -->
    <Menu ref="menu" :model="menuItems" :popup="true" />

    <!-- Dialogs -->
    <!-- Rename Dialog -->
    <Dialog v-model:visible="dialogs.rename" header="重命名设备" modal :style="{ width: '25rem' }">
        <span class="text-text-muted block mb-4">输入新的显示名称。</span>
        <div class="flex flex-col gap-4">
            <InputText v-model="inputs.rename" placeholder="名称" />
            <div class="flex justify-end gap-2">
                <Button label="取消" text severity="secondary" @click="dialogs.rename = false" />
                <Button label="保存" @click="executeRename" :loading="actionLoading" />
            </div>
        </div>
    </Dialog>

    <!-- Group Dialog -->
    <Dialog v-model:visible="dialogs.group" header="设备分组" modal :style="{ width: '25rem' }">
        <span class="text-text-muted block mb-4">设置设备分组以便于管理。</span>
        <div class="flex flex-col gap-4">
             <div class="flex flex-col gap-2">
                <AutoComplete v-model="inputs.group" :suggestions="groupSuggestions" @complete="searchGroup" dropdown forceSelection={false} placeholder="输入或选择分组" />
             </div>
            <div class="flex justify-end gap-2">
                <Button label="取消" text severity="secondary" @click="dialogs.group = false" />
                <Button label="保存" @click="executeGroup" :loading="actionLoading" />
            </div>
        </div>
    </Dialog>

    <!-- Transfer Dialog -->
    <Dialog v-model:visible="dialogs.transfer" header="转移设备" modal :style="{ width: '25rem' }">
        <div class="bg-yellow-50 text-yellow-700 p-3 rounded-lg text-sm mb-4">
            <i class="pi pi-exclamation-triangle mr-2"></i>警告：转移后您将失去对此设备的控制权。
        </div>
        <div class="flex flex-col gap-4">
            <InputNumber v-model="inputs.transferId" placeholder="目标用户 ID" class="w-full" />
            <div class="flex justify-end gap-2">
                <Button label="取消" text severity="secondary" @click="dialogs.transfer = false" />
                <Button label="确认转移" severity="warn" @click="executeTransfer" :loading="actionLoading" :disabled="!inputs.transferId" />
            </div>
        </div>
    </Dialog>

    <!-- Hardware Info Dialog -->
    <Dialog v-model:visible="dialogs.hardware" header="设备详情" modal :style="{ width: '40rem' }" :breakpoints="{ '960px': '75vw', '640px': '90vw' }">
         <div v-if="selectedDevice" class="space-y-6">
            <div class="bg-surface-50 p-4 rounded-xl space-y-2 text-sm">
                <div class="flex justify-between border-b border-surface-200 pb-2">
                    <span class="text-text-muted">ID</span>
                    <span class="font-mono">{{ selectedDevice.id }}</span>
                </div>
                <div class="flex justify-between border-b border-surface-200 pb-2">
                    <span class="text-text-muted">IP</span>
                    <span class="font-mono">{{ selectedDevice.ip }}</span>
                </div>
                <div class="flex justify-between border-b border-surface-200 pb-2">
                    <span class="text-text-muted">Hostname</span>
                    <span>{{ selectedDevice.hostname }}</span>
                </div>
                <div class="flex justify-between border-b border-surface-200 pb-2">
                    <span class="text-text-muted">OS</span>
                    <span>{{ hardwareFingerprint?.os || '-' }}</span>
                </div>
                <div class="flex justify-between pt-2">
                    <span class="text-text-muted">CPU</span>
                    <span class="font-mono text-xs">{{ hardwareFingerprint?.cpu_info || '-' }}</span>
                </div>
            </div>
            
            <div v-if="hardwareFingerprint?.mac_addresses" class="space-y-2">
                <h4 class="font-bold text-sm">网络接口 (MAC)</h4>
                <div class="flex flex-wrap gap-2">
                    <Tag v-for="mac in hardwareFingerprint.mac_addresses" :key="mac" :value="mac" severity="secondary" class="font-mono"></Tag>
                </div>
            </div>
         </div>
    </Dialog>

  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { FilterMatchMode } from '@primevue/core/api';
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import axios from 'axios'

const toast = useToast()
const confirm = useConfirm()

const devices = ref([])
const loading = ref(false)
const bindToken = ref('')
const selectedDevice = ref(null)
const actionLoading = ref(false)
const hardwareFingerprint = ref(null)

const menu = ref();
const menuItems = ref([
    {
        label: '操作',
        items: [
            {
                label: '锁定/解锁',
                icon: 'pi pi-lock',
                command: () => toggleLock(selectedDevice.value)
            },
            {
                label: '转移设备',
                icon: 'pi pi-arrow-right-arrow-left',
                command: () => { dialogs.value.transfer = true }
            },
            {
                separator: true
            },
            {
                label: '解绑设备',
                icon: 'pi pi-link',
                class: 'text-red-500',
                command: () => confirmUnbind(selectedDevice.value)
            },
            {
                label: '删除设备',
                icon: 'pi pi-trash',
                class: 'text-red-500',
                command: () => confirmDelete(selectedDevice.value)
            }
        ]
    }
]);

const filters = ref({
    global: { value: null, matchMode: FilterMatchMode.CONTAINS },
});

const filterStatus = ref(null)
const filterGroup = ref(null)

const statusOptions = [
    { label: '在线', value: 'online'},
    { label: '离线', value: 'offline'}
]

// Dialogs state
const dialogs = ref({
    rename: false,
    group: false,
    transfer: false,
    hardware: false
})
const inputs = ref({
    rename: '',
    group: '',
    transferId: null
})

const onlineCount = computed(() => devices.value.filter(d => d.status === 'online').length)
const offlineCount = computed(() => devices.value.filter(d => d.status !== 'online').length)
const lockedCount = computed(() => devices.value.filter(d => d.is_locked).length)

const deviceGroups = computed(() => {
    const groups = new Set()
    devices.value.forEach(d => { if (d.device_group) groups.add(d.device_group) })
    return Array.from(groups).sort()
})

const groupSuggestions = ref([])
const searchGroup = (event) => {
    groupSuggestions.value = deviceGroups.value.filter(g => g.toLowerCase().includes(event.query.toLowerCase()))
}


onMounted(() => {
    refreshDevices()
})

const refreshDevices = async () => {
  loading.value = true
  try {
    const response = await axios.get('/api/device/list')
    if (response.data.code === 0) {
      devices.value = response.data.devices || []
    } else {
      devices.value = response.data || [] // Legacy support if needed
    }
  } catch (error) {
    console.error('Failed to load devices:', error)
    toast.add({ severity: 'error', summary: 'Error', detail: '加载设备列表失败', life: 3000 })
  } finally {
    loading.value = false
  }
}

const generateToken = async () => {
  try {
    const res = await axios.post('/api/device/bind_token')
    bindToken.value = res.data.token
    toast.add({ severity: 'success', summary: 'Success', detail: '绑定码生成成功', life: 3000 })
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Error', detail: '生成绑定码失败', life: 3000 })
  }
}

const copyToken = () => {
    if (bindToken.value) {
        navigator.clipboard.writeText(bindToken.value)
        toast.add({ severity: 'success', summary: 'Copied', detail: '已复制绑定码', life: 2000 })
    }
}

// Actions
const openRename = (device) => {
    selectedDevice.value = device
    inputs.value.rename = device.display_name || ''
    dialogs.value.rename = true
}

const executeRename = async () => {
    actionLoading.value = true
    try {
        await axios.post('/api/device/update', {
            id: selectedDevice.value.id,
            display_name: inputs.value.rename
        })
        selectedDevice.value.display_name = inputs.value.rename
        dialogs.value.rename = false
        toast.add({ severity: 'success', summary: 'Saved', detail: '已重命名', life: 2000 })
    } catch (e) {
        toast.add({ severity: 'error', summary: 'Error', detail: '重命名失败', life: 3000 })
    } finally {
        actionLoading.value = false
    }
}

const openGroup = (device) => {
    selectedDevice.value = device
    inputs.value.group = device.device_group || ''
    dialogs.value.group = true
}

const executeGroup = async () => {
    actionLoading.value = true
    try {
        await axios.post('/api/device/update', {
            id: selectedDevice.value.id,
            device_group: inputs.value.group
        })
        selectedDevice.value.device_group = inputs.value.group
        dialogs.value.group = false
        toast.add({ severity: 'success', summary: 'Saved', detail: '分组已更新', life: 2000 })
    } catch (e) {
        toast.add({ severity: 'error', summary: 'Error', detail: '更新分组失败', life: 3000 })
    } finally {
        actionLoading.value = false
    }
}

const executeTransfer = async () => {
    if (!inputs.value.transferId) return
    actionLoading.value = true
    try {
        await axios.post('/api/device/transfer', {
            device_id: selectedDevice.value.id,
            user_id: inputs.value.transferId
        })
        devices.value = devices.value.filter(d => d.id !== selectedDevice.value.id)
        dialogs.value.transfer = false
        toast.add({ severity: 'success', summary: 'Transferred', detail: '设备已转移', life: 3000 })
    } catch (e) {
        toast.add({ severity: 'error', summary: 'Error', detail: '转移失败: ' + (e.response?.data?.message || e.message), life: 3000 })
    } finally {
        actionLoading.value = false
    }
}

const toggleLock = async (device) => {
    try {
        const action = device.is_locked ? 'unlock' : 'lock'
        await axios.post(`/api/device/${action}`, { id: device.id })
        device.is_locked = !device.is_locked
        toast.add({ severity: 'info', summary: 'Status', detail: device.is_locked ? '设备已锁定' : '设备已解锁', life: 2000 })
    } catch (e) {
        toast.add({ severity: 'error', summary: 'Error', detail: '操作失败', life: 3000 })
    }
}

const confirmUnbind = (device) => {
    confirm.require({
        message: '确定要解绑此设备吗？解绑后设备将无法继续连接。',
        header: '解绑确认',
        icon: 'pi pi-exclamation-triangle',
        acceptClass: 'p-button-danger',
        accept: async () => {
            try {
                await axios.post('/api/device/unbind', { id: device.id })
                devices.value = devices.value.filter(d => d.id !== device.id)
                toast.add({ severity: 'success', summary: 'Unbound', detail: '设备已解绑', life: 2000 })
            } catch (e) {
                toast.add({ severity: 'error', summary: 'Error', detail: '解绑失败', life: 3000 })
            }
        }
    })
}

const confirmDelete = (device) => {
    confirm.require({
        message: '确定要永久删除此设备吗？此操作不可恢复。',
        header: '删除确认',
        icon: 'pi pi-trash',
        acceptClass: 'p-button-danger',
        accept: async () => {
            try {
                await axios.post('/api/device/delete', { id: device.id })
                devices.value = devices.value.filter(d => d.id !== device.id)
                toast.add({ severity: 'success', summary: 'Deleted', detail: '设备已删除', life: 2000 })
            } catch (e) {
                toast.add({ severity: 'error', summary: 'Error', detail: '删除失败', life: 3000 })
            }
        }
    })
}

const showHardwareInfo = async (device) => {
    selectedDevice.value = device
    dialogs.value.hardware = true
    hardwareFingerprint.value = null
    try {
        // Mock query or existing data, actual implementation might need API
        if (device.hardware_fingerprint) {
            hardwareFingerprint.value = JSON.parse(device.hardware_fingerprint)
        }
    } catch (e) {
        console.error(e)
    }
}


const formatTime = (time) => {
  if (!time) return 'N/A'
  const date = new Date(time)
  return date.toLocaleString('zh-CN')
}
</script>
