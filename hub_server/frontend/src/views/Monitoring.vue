<template>
    <div class="min-h-screen bg-bg p-4 lg:p-10 font-sans text-text">
    <!-- Header Controls -->
    <header class="flex flex-col md:flex-row gap-4 justify-between items-start md:items-center mb-6 lg:mb-8">
      <div class="flex items-center gap-1 bg-surface-0 rounded-xl border border-surface-100 p-1 overflow-x-auto max-w-full scrollbar-hide">
        <button
            v-for="opt in timeOptions" :key="opt.value"
            @click="timeRange = opt.value"
            class="px-3 py-1.5 lg:px-3.5 lg:py-2 text-xs font-medium rounded-lg transition-all duration-200 cursor-pointer whitespace-nowrap"
            :class="timeRange === opt.value
                ? 'bg-primary text-primary-contrast shadow-sm'
                : 'text-text-muted hover:text-text hover:bg-surface-50'"
        >
            {{ opt.label }}
        </button>
      </div>
      <div class="w-full md:w-auto flex justify-end">
          <Button
              :label="loading ? '刷新中...' : '刷新'"
              icon="pi pi-refresh"
              :loading="loading"
              rounded
              outlined
              size="small"
              class="!text-xs lg:!text-sm"
              @click="refreshData"
          />
      </div>
    </header>

    <!-- Key Metrics Cards -->
    <div class="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-3 lg:gap-4 mb-6 lg:mb-8">
        <!-- WebSocket Connections -->
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-5 border border-surface-100 shadow-sm hover:shadow-md transition-shadow">
            <div class="flex items-center justify-between mb-2 lg:mb-3">
                <div class="w-8 h-8 lg:w-9 lg:h-9 rounded-xl bg-blue-500/10 text-blue-500 flex items-center justify-center">
                    <i class="pi pi-server text-xs lg:text-sm"></i>
                </div>
                <div class="flex items-center gap-1 text-[10px] lg:text-xs font-medium" :class="getTrendClass(metrics.connectionsTrend)">
                    <i :class="getTrendIcon(metrics.connectionsTrend)" class="text-[8px] lg:text-[10px]"></i>
                    {{ Math.abs(metrics.connectionsTrend).toFixed(2) }}%
                </div>
            </div>
            <p class="text-xl lg:text-2xl font-bold text-text mb-0.5">{{ metrics.connections }}</p>
            <p class="text-[10px] text-text-muted font-medium uppercase tracking-wider">WebSocket 连接</p>
        </div>

        <!-- API Calls -->
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-5 border border-surface-100 shadow-sm hover:shadow-md transition-shadow">
            <div class="flex items-center justify-between mb-2 lg:mb-3">
                <div class="w-8 h-8 lg:w-9 lg:h-9 rounded-xl bg-purple-500/10 text-purple-500 flex items-center justify-center">
                    <i class="pi pi-globe text-xs lg:text-sm"></i>
                </div>
                <div class="flex items-center gap-1 text-[10px] lg:text-xs font-medium" :class="getTrendClass(metrics.apiCallsTrend)">
                    <i :class="getTrendIcon(metrics.apiCallsTrend)" class="text-[8px] lg:text-[10px]"></i>
                    {{ Math.abs(metrics.apiCallsTrend).toFixed(2) }}%
                </div>
            </div>
            <p class="text-xl lg:text-2xl font-bold text-text mb-0.5">{{ formatNumber(metrics.apiCalls) }}</p>
            <p class="text-[10px] text-text-muted font-medium uppercase tracking-wider">API 调用总数</p>
        </div>

        <!-- API Success Rate -->
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-5 border border-surface-100 shadow-sm hover:shadow-md transition-shadow">
            <div class="flex items-center justify-between mb-2 lg:mb-3">
                <div class="w-8 h-8 lg:w-9 lg:h-9 rounded-xl bg-green-500/10 text-green-500 flex items-center justify-center">
                    <i class="pi pi-check-circle text-xs lg:text-sm"></i>
                </div>
                <div class="flex items-center gap-1 text-[10px] lg:text-xs font-medium" :class="getStatusClass(metrics.successRate)">
                    <i :class="getStatusIcon(metrics.successRate)" class="text-[8px] lg:text-[10px]"></i>
                    {{ getStatusText(metrics.successRate) }}
                </div>
            </div>
            <p class="text-xl lg:text-2xl font-bold text-text mb-0.5">{{ Number(metrics.successRate).toFixed(2) }}%</p>
            <p class="text-[10px] text-text-muted font-medium uppercase tracking-wider">API 成功率</p>
        </div>

        <!-- Avg Response Time -->
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-5 border border-surface-100 shadow-sm hover:shadow-md transition-shadow">
            <div class="flex items-center justify-between mb-2 lg:mb-3">
                <div class="w-8 h-8 lg:w-9 lg:h-9 rounded-xl bg-indigo-500/10 text-indigo-500 flex items-center justify-center">
                    <i class="pi pi-clock text-xs lg:text-sm"></i>
                </div>
                <div class="flex items-center gap-1 text-[10px] lg:text-xs font-medium" :class="getTrendClass(-metrics.responseTimeTrend)">
                    <i :class="getTrendIcon(-metrics.responseTimeTrend)" class="text-[8px] lg:text-[10px]"></i>
                    {{ Math.abs(metrics.responseTimeTrend).toFixed(2) }}%
                </div>
            </div>
            <p class="text-xl lg:text-2xl font-bold text-text mb-0.5">{{ Number(metrics.avgResponseTime).toFixed(2) }}<span class="text-xs lg:text-sm font-normal text-text-muted">ms</span></p>
            <p class="text-[10px] text-text-muted font-medium uppercase tracking-wider">平均响应时间</p>
        </div>

        <!-- Heartbeat -->
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-5 border border-surface-100 shadow-sm hover:shadow-md transition-shadow">
            <div class="flex items-center justify-between mb-2 lg:mb-3">
                <div class="w-8 h-8 lg:w-9 lg:h-9 rounded-xl bg-red-500/10 text-red-500 flex items-center justify-center">
                    <i class="pi pi-heart text-xs lg:text-sm"></i>
                </div>
                <span class="text-[10px] lg:text-xs font-medium" :class="metrics.heartbeatsFailed === 0 ? 'text-green-500' : 'text-red-500'">
                    失败: {{ metrics.heartbeatsFailed }}
                </span>
            </div>
            <p class="text-xl lg:text-2xl font-bold text-text mb-0.5">{{ metrics.heartbeatsSent }}</p>
            <p class="text-[10px] text-text-muted font-medium uppercase tracking-wider">心跳状态</p>
        </div>

        <!-- Compression -->
        <div class="bg-surface-0 rounded-2xl p-4 lg:p-5 border border-surface-100 shadow-sm hover:shadow-md transition-shadow">
            <div class="flex items-center justify-between mb-2 lg:mb-3">
                <div class="w-8 h-8 lg:w-9 lg:h-9 rounded-xl bg-orange-500/10 text-orange-500 flex items-center justify-center">
                    <i class="pi pi-box text-xs lg:text-sm"></i>
                </div>
                <span class="text-[10px] lg:text-xs font-medium text-green-500">{{ formatBytes(metrics.bytesSaved) }}</span>
            </div>
            <p class="text-xl lg:text-2xl font-bold text-text mb-0.5">{{ Number(metrics.compressionRate).toFixed(2) }}<span class="text-xs lg:text-sm font-normal text-text-muted">%</span></p>
            <p class="text-[10px] text-text-muted font-medium uppercase tracking-wider">压缩节省</p>
        </div>
    </div>

    <!-- Charts Area -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-3 lg:gap-4 mb-6 lg:mb-8">
        <div class="bg-surface-0 rounded-2xl border border-surface-100 shadow-sm overflow-hidden">
            <div class="flex items-center gap-2 px-4 py-3 lg:px-5 lg:py-3.5 border-b border-surface-100">
                <i class="pi pi-chart-line text-xs text-blue-500"></i>
                <h3 class="text-xs lg:text-sm font-bold text-text">WebSocket 连接趋势</h3>
            </div>
            <div class="h-[220px] lg:h-[280px] p-2 lg:p-4">
                <canvas ref="connectionsChart"></canvas>
            </div>
        </div>
        <div class="bg-surface-0 rounded-2xl border border-surface-100 shadow-sm overflow-hidden">
            <div class="flex items-center gap-2 px-4 py-3 lg:px-5 lg:py-3.5 border-b border-surface-100">
                <i class="pi pi-chart-line text-xs text-green-500"></i>
                <h3 class="text-xs lg:text-sm font-bold text-text">API 调用趋势</h3>
            </div>
            <div class="h-[220px] lg:h-[280px] p-2 lg:p-4">
                <canvas ref="apiCallsChart"></canvas>
            </div>
        </div>
        <div class="bg-surface-0 rounded-2xl border border-surface-100 shadow-sm overflow-hidden">
            <div class="flex items-center gap-2 px-4 py-3 lg:px-5 lg:py-3.5 border-b border-surface-100">
                <i class="pi pi-clock text-xs text-indigo-500"></i>
                <h3 class="text-xs lg:text-sm font-bold text-text">响应时间分布</h3>
            </div>
            <div class="h-[220px] lg:h-[280px] p-2 lg:p-4">
                <canvas ref="responseTimeChart"></canvas>
            </div>
        </div>
        <div class="bg-surface-0 rounded-2xl border border-surface-100 shadow-sm overflow-hidden">
            <div class="flex items-center gap-2 px-4 py-3 lg:px-5 lg:py-3.5 border-b border-surface-100">
                <i class="pi pi-chart-bar text-xs text-purple-500"></i>
                <h3 class="text-xs lg:text-sm font-bold text-text">API 端点调用</h3>
            </div>
            <div class="h-[220px] lg:h-[280px] p-2 lg:p-4">
                <canvas ref="endpointsChart"></canvas>
            </div>
        </div>
    </div>

    <!-- WebSocket Details -->
    <div class="bg-surface-0 rounded-2xl border border-surface-100 shadow-sm overflow-hidden">
        <!-- Section Header -->
        <div class="flex flex-col md:flex-row justify-between md:items-center px-4 py-3 lg:px-6 lg:py-4 border-b border-surface-100 gap-3">
            <div class="flex items-center gap-2">
                <i class="pi pi-wifi text-sm text-primary"></i>
                <h3 class="text-xs lg:text-sm font-bold text-text">WebSocket 连接详情</h3>
            </div>
            <div class="flex flex-wrap items-center gap-3 lg:gap-4 text-[10px] lg:text-xs">
                <div class="flex items-center gap-1.5">
                    <div class="w-1.5 h-1.5 lg:w-2 lg:h-2 rounded-full bg-primary"></div>
                    <span class="text-text-muted">总连接:</span>
                    <span class="text-text font-bold">{{ wsStats.total_connections }}</span>
                </div>
                <div class="flex items-center gap-1.5">
                    <div class="w-1.5 h-1.5 lg:w-2 lg:h-2 rounded-full" :class="parseFloat(wsStats.ping_success_rate) >= 95 ? 'bg-green-400' : 'bg-red-400'"></div>
                    <span class="text-text-muted">Ping 成功率:</span>
                    <span class="font-bold" :class="getPingSuccessRateClass(wsStats.ping_success_rate)">{{ wsStats.ping_success_rate }}%</span>
                </div>
            </div>
        </div>

        <!-- WS Stats Mini Cards -->
        <div class="grid grid-cols-2 md:grid-cols-4 gap-2 lg:gap-3 p-3 lg:p-5">
            <div class="bg-blue-500/5 p-3 lg:p-3.5 rounded-xl border border-blue-100/50">
                <div class="flex items-center gap-1.5 mb-1 lg:mb-1.5">
                    <i class="pi pi-arrow-right-arrow-left text-[9px] lg:text-[10px] text-blue-500"></i>
                    <span class="text-[9px] lg:text-[10px] text-blue-600 font-semibold uppercase tracking-wider">总 Ping</span>
                </div>
                <p class="text-lg lg:text-xl font-bold text-text">{{ formatNumber(wsStats.total_pings) }}</p>
            </div>
            <div class="bg-green-500/5 p-3 lg:p-3.5 rounded-xl border border-green-100/50">
                <div class="flex items-center gap-1.5 mb-1 lg:mb-1.5">
                    <i class="pi pi-check text-[9px] lg:text-[10px] text-green-500"></i>
                    <span class="text-[9px] lg:text-[10px] text-green-600 font-semibold uppercase tracking-wider">总 Pong</span>
                </div>
                <p class="text-lg lg:text-xl font-bold text-text">{{ formatNumber(wsStats.total_pongs) }}</p>
            </div>
            <div class="bg-purple-500/5 p-3 lg:p-3.5 rounded-xl border border-purple-100/50">
                <div class="flex items-center gap-1.5 mb-1 lg:mb-1.5">
                    <i class="pi pi-comments text-[9px] lg:text-[10px] text-purple-500"></i>
                    <span class="text-[9px] lg:text-[10px] text-purple-600 font-semibold uppercase tracking-wider">消息数</span>
                </div>
                <p class="text-lg lg:text-xl font-bold text-text">{{ formatNumber(wsStats.total_messages) }}</p>
            </div>
            <div class="bg-orange-500/5 p-3 lg:p-3.5 rounded-xl border border-orange-100/50">
                <div class="flex items-center gap-1.5 mb-1 lg:mb-1.5">
                    <i class="pi pi-clock text-[9px] lg:text-[10px] text-orange-500"></i>
                    <span class="text-[9px] lg:text-[10px] text-orange-600 font-semibold uppercase tracking-wider">平均延迟</span>
                </div>
                <p class="text-lg lg:text-xl font-bold text-text">{{ wsStats.avg_latency }}</p>
            </div>
        </div>

        <!-- WS Clients Table -->
        <div class="px-0 pb-0 lg:px-5 lg:pb-5">
            <DataTable
                :value="wsStats.clients"
                :rowHover="true"
                class="ws-table"
                responsiveLayout="scroll"
            >
                <template #empty>
                    <div class="flex flex-col items-center justify-center p-10 text-text-muted">
                        <i class="pi pi-wifi text-3xl mb-2 text-surface-300"></i>
                        <p class="text-sm">暂无 WebSocket 连接</p>
                    </div>
                </template>

                <Column field="id" header="客户端 ID" style="width: 200px">
                    <template #body="{ data }">
                        <span class="font-mono text-xs text-text-muted px-2 py-1 bg-surface-100 rounded-lg" :title="data.id">{{ data.id.substring(0, 16) }}...</span>
                    </template>
                </Column>

                <Column field="hostname" header="主机名">
                    <template #body="{ data }">
                        <div class="flex items-center gap-2">
                            <i class="pi pi-desktop text-xs text-text-muted"></i>
                            <span class="text-sm font-medium">{{ data.hostname }}</span>
                        </div>
                    </template>
                </Column>

                <Column field="ip" header="IP 地址" style="width: 130px">
                    <template #body="{ data }">
                        <span class="font-mono text-xs">{{ data.ip }}</span>
                    </template>
                </Column>

                <Column field="uptime" header="运行时长" style="width: 110px">
                    <template #body="{ data }">
                        <span class="text-xs text-text-muted">{{ data.uptime }}</span>
                    </template>
                </Column>

                <Column header="Ping / Pong" style="width: 110px">
                    <template #body="{ data }">
                        <div class="flex items-center gap-1 font-mono text-xs">
                            <span class="text-green-600 font-medium">{{ data.ping_count }}</span>
                            <span class="text-surface-300">/</span>
                            <span class="text-blue-600 font-medium">{{ data.pong_count }}</span>
                        </div>
                    </template>
                </Column>

                <Column header="延迟" field="avg_latency" style="width: 90px">
                    <template #body="{ data }">
                        <span class="font-mono text-xs font-medium" :class="getLatencyClass(data.avg_latency)">{{ data.avg_latency }}</span>
                    </template>
                </Column>

                <Column header="消息 (发/收)" style="width: 120px">
                    <template #body="{ data }">
                        <div class="flex items-center gap-1 font-mono text-xs">
                            <span class="text-purple-600 font-medium">↑{{ data.messages_sent }}</span>
                            <span class="text-surface-300">/</span>
                            <span class="text-blue-600 font-medium">↓{{ data.messages_recv }}</span>
                        </div>
                    </template>
                </Column>

                <Column header="状态" style="width: 90px">
                    <template #body="{ data }">
                        <div class="flex items-center gap-1.5">
                            <div class="w-2 h-2 rounded-full" :class="data.failure_count === 0 ? 'bg-green-400' : 'bg-red-400'"></div>
                            <span class="text-xs font-medium" :class="data.failure_count === 0 ? 'text-green-600' : 'text-red-600'">
                                {{ data.failure_count === 0 ? '正常' : `失败 ${data.failure_count}` }}
                            </span>
                        </div>
                    </template>
                </Column>
            </DataTable>
        </div>
    </div>

  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import Chart from 'chart.js/auto'

// PrimeVue
import Button from 'primevue/button'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'

const loading = ref(false)
const timeRange = ref('15m')
const timeOptions = [
    { label: '5分钟', value: '5m' },
    { label: '15分钟', value: '15m' },
    { label: '1小时', value: '1h' },
    { label: '6小时', value: '6h' },
    { label: '24小时', value: '24h' }
]

const metrics = ref({
  connections: 0,
  connectionsTrend: 0,
  apiCalls: 0,
  apiCallsTrend: 0,
  successRate: 0,
  avgResponseTime: 0,
  responseTimeTrend: 0,
  heartbeatsSent: 0,
  heartbeatsFailed: 0,
  compressionRate: 0,
  bytesSaved: 0
})

const wsStats = ref({
  total_connections: 0,
  total_pings: 0,
  total_pongs: 0,
  total_messages: 0,
  ping_success_rate: 0,
  avg_latency: '-',
  clients: []
})

const connectionsChart = ref(null)
const apiCallsChart = ref(null)
const responseTimeChart = ref(null)
const endpointsChart = ref(null)

let charts = {}
let refreshInterval = null

// Watch timeRange change to auto-refresh
watch(timeRange, () => {
  refreshData()
})

async function fetchMetrics() {
  try {
    const token = localStorage.getItem('token')
    const response = await fetch('/api/metrics/summary', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    const data = await response.json()
    metrics.value = { ...metrics.value, ...data }
    return data
  } catch (error) {
    console.error('Fetch Metrics Failed:', error)
    return null
  }
}

async function fetchTimeSeriesData() {
  try {
    const token = localStorage.getItem('token')
    const response = await fetch(`/api/metrics/timeseries?range=${timeRange.value}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    return await response.json()
  } catch (error) {
    console.error('Fetch TS Data Failed:', error)
    return null
  }
}

async function fetchWSStats() {
  try {
    const token = localStorage.getItem('token')
    const response = await fetch('/api/ws/stats', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    const result = await response.json()
    if (result.code === 0 && result.data) {
        const data = result.data
        const pingSuccessRate = data.total_pings > 0 ? ((data.total_pongs / data.total_pings) * 100).toFixed(2) : 0
        let avgLatency = '-'
        if (data.clients && data.clients.length > 0) {
            const latencies = data.clients.map(c => c.avg_latency).filter(l => l && l !== '-')
            if (latencies.length > 0) avgLatency = latencies[0]
        }
        wsStats.value = {
            total_connections: data.total_connections || 0,
            total_pings: data.total_pings || 0,
            total_pongs: data.total_pongs || 0,
            total_messages: data.total_messages || 0,
            ping_success_rate: pingSuccessRate,
            avg_latency: avgLatency,
            clients: data.clients || []
        }
    }
  } catch (error) {
      console.error('Fetch WS Stats Failed:', error)
  }
}

async function refreshData() {
  loading.value = true
  try {
    await fetchMetrics()
    await fetchWSStats()
    const timeSeriesData = await fetchTimeSeriesData()
    if (timeSeriesData) {
      updateCharts(timeSeriesData)
    }
  } finally {
    loading.value = false
  }
}

const chartBaseOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false,
      labels: { font: { size: 11, family: 'Inter, system-ui, sans-serif' }, usePointStyle: true, padding: 16 }
    }
  },
  scales: {
    x: {
      grid: { display: false },
      ticks: { font: { size: 10 }, color: '#94a3b8' }
    },
    y: {
      beginAtZero: true,
      grid: { color: 'rgba(0,0,0,0.04)', drawBorder: false },
      ticks: { font: { size: 10 }, color: '#94a3b8' }
    }
  }
}

function initCharts() {
  if (connectionsChart.value) {
    charts.connections = new Chart(connectionsChart.value, {
      type: 'line',
      data: { labels: [], datasets: [{
        label: '连接数', data: [],
        borderColor: '#3b82f6', backgroundColor: 'rgba(59, 130, 246, 0.08)',
        tension: 0.4, fill: true, borderWidth: 2, pointRadius: 0, pointHoverRadius: 4
      }]},
      options: { ...chartBaseOptions }
    })
  }

  if (apiCallsChart.value) {
    charts.apiCalls = new Chart(apiCallsChart.value, {
      type: 'line',
      data: { labels: [], datasets: [
          { label: '成功', data: [], borderColor: '#10b981', backgroundColor: 'rgba(16, 185, 129, 0.08)', tension: 0.4, fill: true, borderWidth: 2, pointRadius: 0, pointHoverRadius: 4 },
          { label: '失败', data: [], borderColor: '#ef4444', backgroundColor: 'rgba(239, 68, 68, 0.08)', tension: 0.4, fill: true, borderWidth: 2, pointRadius: 0, pointHoverRadius: 4 }
      ]},
      options: { ...chartBaseOptions, plugins: { ...chartBaseOptions.plugins, legend: { ...chartBaseOptions.plugins.legend, display: true } } }
    })
  }

  if (responseTimeChart.value) {
    charts.responseTime = new Chart(responseTimeChart.value, {
      type: 'line',
      data: { labels: [], datasets: [
          { label: 'P50', data: [], borderColor: '#3b82f6', tension: 0.4, borderWidth: 2, pointRadius: 0, pointHoverRadius: 4 },
          { label: 'P95', data: [], borderColor: '#f59e0b', tension: 0.4, borderWidth: 2, pointRadius: 0, pointHoverRadius: 4 },
          { label: 'P99', data: [], borderColor: '#ef4444', tension: 0.4, borderWidth: 2, pointRadius: 0, pointHoverRadius: 4 }
      ]},
      options: { ...chartBaseOptions, plugins: { ...chartBaseOptions.plugins, legend: { ...chartBaseOptions.plugins.legend, display: true } } }
    })
  }

  if (endpointsChart.value) {
    charts.endpoints = new Chart(endpointsChart.value, {
      type: 'bar',
      data: { labels: [], datasets: [{
        label: '调用次数', data: [],
        backgroundColor: [
          'rgba(59, 130, 246, 0.7)', 'rgba(16, 185, 129, 0.7)',
          'rgba(245, 158, 11, 0.7)', 'rgba(139, 92, 246, 0.7)',
          'rgba(236, 72, 153, 0.7)', 'rgba(20, 184, 166, 0.7)'
        ],
        borderRadius: 8, borderSkipped: false
      }]},
      options: { ...chartBaseOptions, indexAxis: 'y' }
    })
  }
}

function updateCharts(data) {
  if (charts.connections && data.connections) {
    charts.connections.data.labels = data.connections.labels
    charts.connections.data.datasets[0].data = data.connections.values
    charts.connections.update()
  }
  if (charts.apiCalls && data.apiCalls) {
    charts.apiCalls.data.labels = data.apiCalls.labels
    charts.apiCalls.data.datasets[0].data = data.apiCalls.success
    charts.apiCalls.data.datasets[1].data = data.apiCalls.failed
    charts.apiCalls.update()
  }
  if (charts.responseTime && data.responseTime) {
    charts.responseTime.data.labels = data.responseTime.labels
    charts.responseTime.data.datasets[0].data = data.responseTime.p50
    charts.responseTime.data.datasets[1].data = data.responseTime.p95
    charts.responseTime.data.datasets[2].data = data.responseTime.p99
    charts.responseTime.update()
  }
  if (charts.endpoints && data.endpoints) {
    charts.endpoints.data.labels = data.endpoints.labels
    charts.endpoints.data.datasets[0].data = data.endpoints.values
    charts.endpoints.update()
  }
}

function formatNumber(num) {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num || 0
}

function formatBytes(bytes) {
  if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(2) + ' GB'
  if (bytes >= 1048576) return (bytes / 1048576).toFixed(2) + ' MB'
  if (bytes >= 1024) return (bytes / 1024).toFixed(2) + ' KB'
  return (bytes || 0) + ' B'
}

function getTrendClass(trend) {
  if (trend > 0) return 'text-green-600'
  if (trend < 0) return 'text-red-500'
  return 'text-text-muted'
}

function getTrendIcon(trend) {
    if (trend > 0) return 'pi pi-arrow-up'
    if (trend < 0) return 'pi pi-arrow-down'
    return 'pi pi-minus'
}

function getStatusClass(rate) {
  if (rate >= 95) return 'text-green-600'
  if (rate >= 90) return 'text-orange-500'
  return 'text-red-500'
}

function getStatusIcon(rate) {
    if (rate >= 95) return 'pi pi-check-circle'
    if (rate >= 90) return 'pi pi-exclamation-circle'
    return 'pi pi-times-circle'
}

function getStatusText(rate) {
  if (rate >= 95) return '优秀'
  if (rate >= 90) return '良好'
  return '需关注'
}

function getPingSuccessRateClass(rate) {
  const numRate = parseFloat(rate)
  if (numRate >= 95) return 'text-green-600'
  if (numRate >= 90) return 'text-orange-500'
  return 'text-red-500'
}

function getLatencyClass(latency) {
  if (!latency || latency === '-') return 'text-text-muted'
  const ms = parseInt(latency)
  if (ms < 100) return 'text-green-600'
  if (ms < 500) return 'text-orange-500'
  return 'text-red-500'
}

onMounted(async () => {
  await refreshData()
  initCharts()
  refreshInterval = setInterval(refreshData, 10000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
  Object.values(charts).forEach(chart => { if (chart) chart.destroy() })
})
</script>

<style scoped>
/* WebSocket DataTable Styling */
:deep(.ws-table .p-datatable-thead > tr > th) {
    background-color: var(--color-surface-50);
    color: var(--p-text-muted-color);
    font-weight: 600;
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    padding: 0.625rem 0.875rem;
    border-bottom: 1px solid var(--p-surface-200);
    border-top: none;
}
:deep(.ws-table .p-datatable-tbody > tr > td) {
    padding: 0.5rem 0.875rem;
    border-bottom: 1px solid var(--p-surface-100);
    font-size: 0.8125rem;
}
:deep(.ws-table .p-datatable-tbody > tr:last-child > td) {
    border-bottom: none;
}
:deep(.ws-table .p-datatable-tbody > tr:hover > td) {
    background-color: var(--color-surface-50) !important;
}
</style>
