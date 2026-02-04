<template>
  <div class="w-full space-y-8 p-8">
    <!-- Stats Cards -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
       <!-- Users -->
       <div class="bg-white rounded-3xl p-6 shadow-card border border-slate-100 flex items-center justify-between">
           <div>
               <p class="text-slate-500 text-sm font-medium mb-1">总用户数</p>
               <h3 class="text-3xl font-bold text-slate-800">{{ stats.users || 0 }}</h3>
           </div>
           <div class="w-12 h-12 rounded-2xl bg-blue-50 text-blue-500 flex items-center justify-center">
               <component :is="Users" class="w-6 h-6" />
           </div>
       </div>
       
       <!-- Devices -->
       <div class="bg-white rounded-3xl p-6 shadow-card border border-slate-100 flex items-center justify-between">
           <div>
               <p class="text-slate-500 text-sm font-medium mb-1">活跃设备</p>
               <h3 class="text-3xl font-bold text-slate-800">{{ stats.devices || 0 }}</h3>
           </div>
           <div class="w-12 h-12 rounded-2xl bg-purple-50 text-purple-500 flex items-center justify-center">
               <component :is="Monitor" class="w-6 h-6" />
           </div>
       </div>

       <!-- Transactions -->
       <div class="bg-white rounded-3xl p-6 shadow-card border border-slate-100 flex items-center justify-between">
           <div>
               <p class="text-slate-500 text-sm font-medium mb-1">交易记录</p>
               <h3 class="text-3xl font-bold text-slate-800">{{ stats.transactions || 0 }}</h3>
           </div>
           <div class="w-12 h-12 rounded-2xl bg-green-50 text-green-500 flex items-center justify-center">
               <component :is="Receipt" class="w-6 h-6" />
           </div>
       </div>

       <!-- Total Credits -->
       <div class="bg-white rounded-3xl p-6 shadow-card border border-slate-100 flex items-center justify-between">
           <div>
               <p class="text-slate-500 text-sm font-medium mb-1">积分流通量</p>
               <h3 class="text-3xl font-bold text-amber-500">{{ stats.total_credits || 0 }}</h3>
           </div>
           <div class="w-12 h-12 rounded-2xl bg-amber-50 text-amber-500 flex items-center justify-center">
               <component :is="Coins" class="w-6 h-6" />
           </div>
       </div>
    </div>

    <!-- User Table -->
    <div class="bg-white rounded-3xl p-8 shadow-card border border-slate-100">
      <div class="flex items-center justify-between mb-6">
        <div class="flex gap-2">
          <button
            @click="activeTab = 'users'"
            :class="activeTab === 'users' ? 'bg-primary text-white' : 'bg-bg text-slate-600'"
            class="px-4 py-2 rounded-xl shadow-neu hover:shadow-neu-sm transition-all font-medium text-sm"
          >
            <component :is="Users" class="w-4 h-4 inline mr-1" />
            用户
          </button>
          <button
            @click="activeTab = 'devices'"
            :class="activeTab === 'devices' ? 'bg-primary text-white' : 'bg-bg text-slate-600'"
            class="px-4 py-2 rounded-xl shadow-neu hover:shadow-neu-sm transition-all font-medium text-sm"
          >
            <component :is="Monitor" class="w-4 h-4 inline mr-1" />
            设备
          </button>
          <button
            @click="activeTab = 'tasks'"
            :class="activeTab === 'tasks' ? 'bg-primary text-white' : 'bg-bg text-slate-600'"
            class="px-4 py-2 rounded-xl shadow-neu hover:shadow-neu-sm transition-all font-medium text-sm"
          >
            <component :is="ListTodo" class="w-4 h-4 inline mr-1" />
            任务
          </button>
          <button
            @click="activeTab = 'subscriptions'"
            :class="activeTab === 'subscriptions' ? 'bg-primary text-white' : 'bg-bg text-slate-600'"
            class="px-4 py-2 rounded-xl shadow-neu hover:shadow-neu-sm transition-all font-medium text-sm"
          >
            <component :is="Rss" class="w-4 h-4 inline mr-1" />
            订阅
          </button>
        </div>
      </div>
      
      <div v-if="loading" class="text-center py-10 text-slate-400">
          加载中...
      </div>

      <!-- 用户列表 -->
      <div v-else-if="activeTab === 'users'" class="overflow-x-auto">
          <table class="w-full text-left border-collapse">
              <thead>
                  <tr>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">ID</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">用户邮箱</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">角色</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">当前积分</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">注册时间</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">操作</th>
                  </tr>
              </thead>
              <tbody>
                  <tr v-for="user in users" :key="user.id" class="group hover:bg-slate-50 transition-colors">
                      <td class="p-4 border-b border-slate-100 text-slate-500 font-mono text-xs">#{{ user.id }}</td>
                      <td class="p-4 border-b border-slate-100 font-medium text-slate-700">{{ user.email }}</td>
                      <td class="p-4 border-b border-slate-100">
                          <span :class="user.role === 'admin' ? 'bg-purple-100 text-purple-700' : 'bg-slate-100 text-slate-600'" class="px-2 py-1 rounded-md text-xs font-bold uppercase">{{ user.role }}</span>
                      </td>
                      <td class="p-4 border-b border-slate-100 font-mono font-bold text-amber-600">{{ user.credits }}</td>
                      <td class="p-4 border-b border-slate-100 text-slate-400 text-sm">{{ formatDate(user.created_at) }}</td>
                      <td class="p-4 border-b border-slate-100">
                        <div class="flex gap-2">
                          <button
                            @click="openEditCredits(user)"
                            class="px-3 py-1 bg-bg shadow-neu rounded-lg text-amber-600 hover:shadow-neu-sm transition-all flex items-center gap-1 text-sm"
                            title="编辑积分"
                          >
                            <component :is="Coins" class="w-3 h-3" />
                            积分
                          </button>
                          <button
                            @click="openEditRole(user)"
                            class="px-3 py-1 bg-bg shadow-neu rounded-lg text-purple-600 hover:shadow-neu-sm transition-all flex items-center gap-1 text-sm"
                            title="修改角色"
                          >
                            <component :is="Shield" class="w-3 h-3" />
                            角色
                          </button>
                          <button
                            @click="confirmDeleteUser(user)"
                            class="px-3 py-1 bg-bg shadow-neu rounded-lg text-red-600 hover:shadow-neu-sm transition-all flex items-center gap-1 text-sm"
                            title="删除用户"
                          >
                            <component :is="Trash2" class="w-3 h-3" />
                            删除
                          </button>
                        </div>
                      </td>
                  </tr>
              </tbody>
          </table>
      </div>

      <!-- 设备列表 -->
      <div v-else-if="activeTab === 'devices'" class="overflow-x-auto">
          <table class="w-full text-left border-collapse">
              <thead>
                  <tr>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">设备 ID</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">主机名</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">版本</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">状态</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">绑定用户</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">最后在线</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">操作</th>
                  </tr>
              </thead>
              <tbody>
                  <tr v-for="device in devices" :key="device.id" class="group hover:bg-slate-50 transition-colors">
                      <td class="p-4 border-b border-slate-100 text-slate-500 font-mono text-xs">{{ device.id }}</td>
                      <td class="p-4 border-b border-slate-100 font-medium text-slate-700">{{ device.hostname || 'N/A' }}</td>
                      <td class="p-4 border-b border-slate-100 text-slate-600 text-sm">{{ device.version || 'N/A' }}</td>
                      <td class="p-4 border-b border-slate-100">
                          <span :class="device.status === 'online' ? 'bg-green-100 text-green-700' : 'bg-slate-100 text-slate-600'" class="px-2 py-1 rounded-md text-xs font-bold uppercase">{{ device.status }}</span>
                      </td>
                      <td class="p-4 border-b border-slate-100 text-slate-600 text-sm">
                          <span v-if="device.user_id > 0" class="font-medium">用户 #{{ device.user_id }}</span>
                          <span v-else class="text-slate-400">未绑定</span>
                      </td>
                      <td class="p-4 border-b border-slate-100 text-slate-400 text-sm">{{ formatDate(device.last_seen) }}</td>
                      <td class="p-4 border-b border-slate-100">
                        <div class="flex gap-2">
                          <button
                            v-if="device.user_id > 0"
                            @click="confirmUnbindDevice(device)"
                            class="px-3 py-1 bg-bg shadow-neu rounded-lg text-orange-600 hover:shadow-neu-sm transition-all flex items-center gap-1 text-sm"
                            title="解绑设备"
                          >
                            <component :is="Unlink" class="w-3 h-3" />
                            解绑
                          </button>
                          <button
                            @click="confirmDeleteDevice(device)"
                            class="px-3 py-1 bg-bg shadow-neu rounded-lg text-red-600 hover:shadow-neu-sm transition-all flex items-center gap-1 text-sm"
                            title="删除设备"
                          >
                            <component :is="Trash2" class="w-3 h-3" />
                            删除
                          </button>
                        </div>
                      </td>
                  </tr>
              </tbody>
          </table>
      </div>

      <!-- 任务列表 -->
      <div v-else-if="activeTab === 'tasks'" class="overflow-x-auto">
          <table class="w-full text-left border-collapse">
              <thead>
                  <tr>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">ID</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">类型</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">用户</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">设备</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">状态</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">创建时间</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">操作</th>
                  </tr>
              </thead>
              <tbody>
                  <tr v-for="task in tasks" :key="task.id" class="group hover:bg-slate-50 transition-colors">
                      <td class="p-4 border-b border-slate-100 text-slate-500 font-mono text-xs">#{{ task.id }}</td>
                      <td class="p-4 border-b border-slate-100 font-medium text-slate-700">{{ task.type }}</td>
                      <td class="p-4 border-b border-slate-100 text-slate-600 text-sm">用户 #{{ task.user_id }}</td>
                      <td class="p-4 border-b border-slate-100 text-slate-600 text-sm font-mono">{{ task.node_id }}</td>
                      <td class="p-4 border-b border-slate-100">
                          <span :class="getTaskStatusClass(task.status)" class="px-2 py-1 rounded-md text-xs font-bold uppercase">{{ task.status }}</span>
                      </td>
                      <td class="p-4 border-b border-slate-100 text-slate-400 text-sm">{{ formatDate(task.created_at) }}</td>
                      <td class="p-4 border-b border-slate-100">
                        <button
                          @click="confirmDeleteTask(task)"
                          class="px-3 py-1 bg-bg shadow-neu rounded-lg text-red-600 hover:shadow-neu-sm transition-all flex items-center gap-1 text-sm"
                          title="删除任务"
                        >
                          <component :is="Trash2" class="w-3 h-3" />
                          删除
                        </button>
                      </td>
                  </tr>
              </tbody>
          </table>
      </div>

      <!-- 订阅列表 -->
      <div v-else-if="activeTab === 'subscriptions'" class="overflow-x-auto">
          <table class="w-full text-left border-collapse">
              <thead>
                  <tr>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">ID</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">昵称</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">Finder ID</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">用户</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">视频数</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">创建时间</th>
                      <th class="p-4 border-b border-slate-100 text-slate-400 font-medium text-sm">操作</th>
                  </tr>
              </thead>
              <tbody>
                  <tr v-for="sub in subscriptions" :key="sub.id" class="group hover:bg-slate-50 transition-colors">
                      <td class="p-4 border-b border-slate-100 text-slate-500 font-mono text-xs">#{{ sub.id }}</td>
                      <td class="p-4 border-b border-slate-100 font-medium text-slate-700">{{ sub.nickname }}</td>
                      <td class="p-4 border-b border-slate-100 text-slate-600 text-sm font-mono">{{ sub.finder_id }}</td>
                      <td class="p-4 border-b border-slate-100 text-slate-600 text-sm">用户 #{{ sub.user_id }}</td>
                      <td class="p-4 border-b border-slate-100 text-slate-600 text-sm">{{ sub.video_count || 0 }}</td>
                      <td class="p-4 border-b border-slate-100 text-slate-400 text-sm">{{ formatDate(sub.created_at) }}</td>
                      <td class="p-4 border-b border-slate-100">
                        <button
                          @click="confirmDeleteSubscription(sub)"
                          class="px-3 py-1 bg-bg shadow-neu rounded-lg text-red-600 hover:shadow-neu-sm transition-all flex items-center gap-1 text-sm"
                          title="删除订阅"
                        >
                          <component :is="Trash2" class="w-3 h-3" />
                          删除
                        </button>
                      </td>
                  </tr>
              </tbody>
          </table>
      </div>
    </div>

    <!-- 删除任务确认对话框 -->
    <div 
      v-if="showDeleteTask"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
      @click.self="showDeleteTask = false"
    >
      <div class="bg-white shadow-card rounded-2xl p-8 max-w-md w-full mx-4">
        <div class="flex items-center gap-4 mb-6">
          <div class="w-12 h-12 rounded-xl bg-red-100 flex items-center justify-center">
            <component :is="Trash2" class="w-6 h-6 text-red-600" />
          </div>
          <div>
            <h3 class="text-xl font-bold text-slate-800">删除任务</h3>
            <p class="text-sm text-slate-500">此操作不可恢复</p>
          </div>
        </div>

        <div class="mb-6">
          <p class="text-slate-700 mb-4">
            确定要删除任务 <span class="font-bold">#{{ selectedTask?.id }}</span> 吗？
          </p>
          <div class="bg-red-50 rounded-xl p-4">
            <p class="text-sm text-red-600">
              ⚠️ 删除后，该任务的所有记录都将被永久删除。
            </p>
          </div>
        </div>

        <div class="flex gap-3">
          <button
            @click="showDeleteTask = false"
            class="flex-1 px-4 py-3 bg-bg shadow-neu rounded-xl text-slate-600 hover:shadow-neu-sm transition-all"
          >
            取消
          </button>
          <button
            @click="deleteTask"
            :disabled="actionLoading"
            class="flex-1 px-4 py-3 rounded-xl bg-red-600 text-white hover:bg-red-700 transition-all"
          >
            {{ actionLoading ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 删除订阅确认对话框 -->
    <div 
      v-if="showDeleteSubscription"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
      @click.self="showDeleteSubscription = false"
    >
      <div class="bg-white shadow-card rounded-2xl p-8 max-w-md w-full mx-4">
        <div class="flex items-center gap-4 mb-6">
          <div class="w-12 h-12 rounded-xl bg-red-100 flex items-center justify-center">
            <component :is="Trash2" class="w-6 h-6 text-red-600" />
          </div>
          <div>
            <h3 class="text-xl font-bold text-slate-800">删除订阅</h3>
            <p class="text-sm text-slate-500">此操作不可恢复</p>
          </div>
        </div>

        <div class="mb-6">
          <p class="text-slate-700 mb-4">
            确定要删除订阅 <span class="font-bold">{{ selectedSubscription?.nickname }}</span> 吗？
          </p>
          <div class="bg-red-50 rounded-xl p-4">
            <p class="text-sm text-red-600">
              ⚠️ 删除后，该订阅及其所有视频记录都将被永久删除。
            </p>
          </div>
        </div>

        <div class="flex gap-3">
          <button
            @click="showDeleteSubscription = false"
            class="flex-1 px-4 py-3 bg-bg shadow-neu rounded-xl text-slate-600 hover:shadow-neu-sm transition-all"
          >
            取消
          </button>
          <button
            @click="deleteSubscription"
            :disabled="actionLoading"
            class="flex-1 px-4 py-3 rounded-xl bg-red-600 text-white hover:bg-red-700 transition-all"
          >
            {{ actionLoading ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 解绑设备确认对话框 -->
    <div 
      v-if="showUnbindDevice"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
      @click.self="showUnbindDevice = false"
    >
      <div class="bg-white shadow-card rounded-2xl p-8 max-w-md w-full mx-4">
        <div class="flex items-center gap-4 mb-6">
          <div class="w-12 h-12 rounded-xl bg-orange-100 flex items-center justify-center">
            <component :is="Unlink" class="w-6 h-6 text-orange-600" />
          </div>
          <div>
            <h3 class="text-xl font-bold text-slate-800">解绑设备</h3>
            <p class="text-sm text-slate-500">此操作将解除设备与用户的绑定</p>
          </div>
        </div>

        <div class="mb-6">
          <p class="text-slate-700 mb-4">
            确定要解绑设备 <span class="font-bold font-mono">{{ selectedDevice?.id }}</span> 吗？
          </p>
          <div class="bg-orange-50 rounded-xl p-4">
            <p class="text-sm text-orange-600">
              解绑后，设备将不再与用户 #{{ selectedDevice?.user_id }} 关联。
            </p>
          </div>
        </div>

        <div class="flex gap-3">
          <button
            @click="showUnbindDevice = false"
            class="flex-1 px-4 py-3 bg-bg shadow-neu rounded-xl text-slate-600 hover:shadow-neu-sm transition-all"
          >
            取消
          </button>
          <button
            @click="unbindDevice"
            :disabled="actionLoading"
            class="flex-1 px-4 py-3 rounded-xl bg-orange-600 text-white hover:bg-orange-700 transition-all"
          >
            {{ actionLoading ? '处理中...' : '确认解绑' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 删除设备确认对话框 -->
    <div 
      v-if="showDeleteDevice"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
      @click.self="showDeleteDevice = false"
    >
      <div class="bg-white shadow-card rounded-2xl p-8 max-w-md w-full mx-4">
        <div class="flex items-center gap-4 mb-6">
          <div class="w-12 h-12 rounded-xl bg-red-100 flex items-center justify-center">
            <component :is="Trash2" class="w-6 h-6 text-red-600" />
          </div>
          <div>
            <h3 class="text-xl font-bold text-slate-800">删除设备</h3>
            <p class="text-sm text-slate-500">此操作不可恢复</p>
          </div>
        </div>

        <div class="mb-6">
          <p class="text-slate-700 mb-4">
            确定要删除设备 <span class="font-bold font-mono">{{ selectedDevice?.id }}</span> 吗？
          </p>
          <div class="bg-red-50 rounded-xl p-4">
            <p class="text-sm text-red-600">
              ⚠️ 删除后，该设备的所有记录都将被永久删除。
            </p>
          </div>
        </div>

        <div class="flex gap-3">
          <button
            @click="showDeleteDevice = false"
            class="flex-1 px-4 py-3 bg-bg shadow-neu rounded-xl text-slate-600 hover:shadow-neu-sm transition-all"
          >
            取消
          </button>
          <button
            @click="deleteDevice"
            :disabled="actionLoading"
            class="flex-1 px-4 py-3 rounded-xl bg-red-600 text-white hover:bg-red-700 transition-all"
          >
            {{ actionLoading ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 编辑积分对话框 -->
    <div 
      v-if="showEditCredits"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
      @click.self="showEditCredits = false"
    >
      <div class="bg-white shadow-card rounded-2xl p-8 max-w-md w-full mx-4">
        <div class="flex items-center gap-4 mb-6">
          <div class="w-12 h-12 rounded-xl bg-amber-100 flex items-center justify-center">
            <component :is="Coins" class="w-6 h-6 text-amber-600" />
          </div>
          <div>
            <h3 class="text-xl font-bold text-slate-800">编辑积分</h3>
            <p class="text-sm text-slate-500">{{ selectedUser?.email }}</p>
          </div>
        </div>

        <div class="mb-6">
          <label class="block text-sm font-medium text-slate-700 mb-2">当前积分</label>
          <p class="text-2xl font-bold text-amber-600 mb-4">{{ selectedUser?.credits }}</p>
          
          <label class="block text-sm font-medium text-slate-700 mb-2">调整积分</label>
          <input
            v-model.number="creditsAdjustment"
            type="number"
            class="w-full px-4 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-primary"
            placeholder="输入正数增加，负数减少"
          />
          <p class="text-xs text-slate-500 mt-2">
            调整后积分：{{ (selectedUser?.credits || 0) + (creditsAdjustment || 0) }}
          </p>
        </div>

        <div class="flex gap-3">
          <button
            @click="showEditCredits = false"
            class="flex-1 px-4 py-3 bg-bg shadow-neu rounded-xl text-slate-600 hover:shadow-neu-sm transition-all"
          >
            取消
          </button>
          <button
            @click="updateCredits"
            :disabled="actionLoading"
            class="flex-1 px-4 py-3 rounded-xl bg-amber-600 text-white hover:bg-amber-700 transition-all"
          >
            {{ actionLoading ? '处理中...' : '确认' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 修改角色对话框 -->
    <div 
      v-if="showEditRole"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
      @click.self="showEditRole = false"
    >
      <div class="bg-white shadow-card rounded-2xl p-8 max-w-md w-full mx-4">
        <div class="flex items-center gap-4 mb-6">
          <div class="w-12 h-12 rounded-xl bg-purple-100 flex items-center justify-center">
            <component :is="Shield" class="w-6 h-6 text-purple-600" />
          </div>
          <div>
            <h3 class="text-xl font-bold text-slate-800">修改角色</h3>
            <p class="text-sm text-slate-500">{{ selectedUser?.email }}</p>
          </div>
        </div>

        <div class="mb-6">
          <label class="block text-sm font-medium text-slate-700 mb-2">当前角色</label>
          <p class="text-lg font-bold text-purple-600 mb-4 uppercase">{{ selectedUser?.role }}</p>
          
          <label class="block text-sm font-medium text-slate-700 mb-2">新角色</label>
          <select
            v-model="newRole"
            class="w-full px-4 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-primary"
          >
            <option value="user">User（普通用户）</option>
            <option value="admin">Admin（管理员）</option>
          </select>
        </div>

        <div class="flex gap-3">
          <button
            @click="showEditRole = false"
            class="flex-1 px-4 py-3 bg-bg shadow-neu rounded-xl text-slate-600 hover:shadow-neu-sm transition-all"
          >
            取消
          </button>
          <button
            @click="updateRole"
            :disabled="actionLoading"
            class="flex-1 px-4 py-3 rounded-xl bg-purple-600 text-white hover:bg-purple-700 transition-all"
          >
            {{ actionLoading ? '处理中...' : '确认' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 删除用户确认对话框 -->
    <div 
      v-if="showDeleteConfirm"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
      @click.self="showDeleteConfirm = false"
    >
      <div class="bg-white shadow-card rounded-2xl p-8 max-w-md w-full mx-4">
        <div class="flex items-center gap-4 mb-6">
          <div class="w-12 h-12 rounded-xl bg-red-100 flex items-center justify-center">
            <component :is="Trash2" class="w-6 h-6 text-red-600" />
          </div>
          <div>
            <h3 class="text-xl font-bold text-slate-800">删除用户</h3>
            <p class="text-sm text-slate-500">此操作不可恢复</p>
          </div>
        </div>

        <div class="mb-6">
          <p class="text-slate-700 mb-4">
            确定要删除用户 <span class="font-bold">{{ selectedUser?.email }}</span> 吗？
          </p>
          <div class="bg-red-50 rounded-xl p-4">
            <p class="text-sm text-red-600">
              ⚠️ 删除后，该用户的所有数据（设备、订阅、任务等）都将被永久删除。
            </p>
          </div>
        </div>

        <div class="flex gap-3">
          <button
            @click="showDeleteConfirm = false"
            class="flex-1 px-4 py-3 bg-bg shadow-neu rounded-xl text-slate-600 hover:shadow-neu-sm transition-all"
          >
            取消
          </button>
          <button
            @click="deleteUser"
            :disabled="actionLoading"
            class="flex-1 px-4 py-3 rounded-xl bg-red-600 text-white hover:bg-red-700 transition-all"
          >
            {{ actionLoading ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import axios from 'axios'
import { useRouter } from 'vue-router'
import { Users, Monitor, Receipt, Coins, RefreshCw, Shield, Trash2, Unlink, ListTodo, Rss } from 'lucide-vue-next'

const stats = ref({})
const users = ref([])
const devices = ref([])
const tasks = ref([])
const subscriptions = ref([])
const loading = ref(true)
const router = useRouter()
const activeTab = ref('users')

// 对话框状态
const showEditCredits = ref(false)
const showEditRole = ref(false)
const showDeleteConfirm = ref(false)
const showUnbindDevice = ref(false)
const showDeleteDevice = ref(false)
const showDeleteTask = ref(false)
const showDeleteSubscription = ref(false)
const selectedUser = ref(null)
const selectedDevice = ref(null)
const selectedTask = ref(null)
const selectedSubscription = ref(null)
const actionLoading = ref(false)

// 表单数据
const creditsAdjustment = ref(0)
const newRole = ref('user')

// 监听标签切换，自动刷新数据
watch(activeTab, () => {
    fetchData()
})

const fetchData = async () => {
    loading.value = true
    try {
        const requests = [
            axios.get('/api/admin/stats'),
            axios.get('/api/admin/users')
        ]
        
        // 根据当前标签页加载对应数据
        if (activeTab.value === 'devices') {
            requests.push(axios.get('/api/admin/devices'))
        } else if (activeTab.value === 'tasks') {
            requests.push(axios.get('/api/admin/tasks'))
        } else if (activeTab.value === 'subscriptions') {
            requests.push(axios.get('/api/admin/subscriptions'))
        }
        
        const responses = await Promise.all(requests)
        stats.value = responses[0].data
        users.value = responses[1].data.list
        
        if (responses[2]) {
            if (activeTab.value === 'devices') {
                devices.value = responses[2].data || []
            } else if (activeTab.value === 'tasks') {
                tasks.value = responses[2].data.list || []
            } else if (activeTab.value === 'subscriptions') {
                subscriptions.value = responses[2].data || []
            }
        }
    } catch (err) {
        if (err.response && err.response.status === 403) {
            alert("需要管理员权限")
            router.push('/dashboard')
        }
    } finally {
        loading.value = false
    }
}

const formatDate = (dateStr) => {
    return new Date(dateStr).toLocaleDateString()
}

// 打开编辑积分对话框
const openEditCredits = (user) => {
    selectedUser.value = user
    creditsAdjustment.value = 0
    showEditCredits.value = true
}

// 打开修改角色对话框
const openEditRole = (user) => {
    selectedUser.value = user
    newRole.value = user.role
    showEditRole.value = true
}

// 打开删除确认对话框
const confirmDeleteUser = (user) => {
    selectedUser.value = user
    showDeleteConfirm.value = true
}

// 更新积分
const updateCredits = async () => {
    if (!selectedUser.value || creditsAdjustment.value === 0) {
        alert('请输入调整金额')
        return
    }

    actionLoading.value = true
    try {
        await axios.post('/api/admin/user/credits', {
            user_id: selectedUser.value.id,
            adjustment: creditsAdjustment.value
        })
        
        showEditCredits.value = false
        await fetchData()
        alert('积分更新成功')
    } catch (error) {
        console.error('Update credits failed:', error)
        alert(error.response?.data || '更新积分失败')
    } finally {
        actionLoading.value = false
    }
}

// 更新角色
const updateRole = async () => {
    if (!selectedUser.value || !newRole.value) {
        alert('请选择角色')
        return
    }

    actionLoading.value = true
    try {
        await axios.post('/api/admin/user/role', {
            user_id: selectedUser.value.id,
            role: newRole.value
        })
        
        showEditRole.value = false
        await fetchData()
        alert('角色更新成功')
    } catch (error) {
        console.error('Update role failed:', error)
        alert(error.response?.data || '更新角色失败')
    } finally {
        actionLoading.value = false
    }
}

// 删除用户
const deleteUser = async () => {
    if (!selectedUser.value) return

    actionLoading.value = true
    try {
        await axios.delete(`/api/admin/user/${selectedUser.value.id}`)
        
        showDeleteConfirm.value = false
        await fetchData()
        alert('用户删除成功')
    } catch (error) {
        console.error('Delete user failed:', error)
        alert(error.response?.data || '删除用户失败')
    } finally {
        actionLoading.value = false
    }
}

// 打开解绑设备确认对话框
const confirmUnbindDevice = (device) => {
    selectedDevice.value = device
    showUnbindDevice.value = true
}

// 打开删除设备确认对话框
const confirmDeleteDevice = (device) => {
    selectedDevice.value = device
    showDeleteDevice.value = true
}

// 解绑设备
const unbindDevice = async () => {
    if (!selectedDevice.value) return

    actionLoading.value = true
    try {
        await axios.post('/api/admin/device/unbind', {
            device_id: selectedDevice.value.id
        })
        
        showUnbindDevice.value = false
        await fetchData()
        alert('设备解绑成功')
    } catch (error) {
        console.error('Unbind device failed:', error)
        alert(error.response?.data || '解绑设备失败')
    } finally {
        actionLoading.value = false
    }
}

// 删除设备
const deleteDevice = async () => {
    if (!selectedDevice.value) return

    actionLoading.value = true
    try {
        await axios.delete(`/api/admin/device/${selectedDevice.value.id}`)
        
        showDeleteDevice.value = false
        await fetchData()
        alert('设备删除成功')
    } catch (error) {
        console.error('Delete device failed:', error)
        alert(error.response?.data || '删除设备失败')
    } finally {
        actionLoading.value = false
    }
}

// 打开删除任务确认对话框
const confirmDeleteTask = (task) => {
    selectedTask.value = task
    showDeleteTask.value = true
}

// 删除任务
const deleteTask = async () => {
    if (!selectedTask.value) return

    actionLoading.value = true
    try {
        await axios.delete(`/api/admin/task/${selectedTask.value.id}`)
        
        showDeleteTask.value = false
        await fetchData()
        alert('任务删除成功')
    } catch (error) {
        console.error('Delete task failed:', error)
        alert(error.response?.data || '删除任务失败')
    } finally {
        actionLoading.value = false
    }
}

// 打开删除订阅确认对话框
const confirmDeleteSubscription = (sub) => {
    selectedSubscription.value = sub
    showDeleteSubscription.value = true
}

// 删除订阅
const deleteSubscription = async () => {
    if (!selectedSubscription.value) return

    actionLoading.value = true
    try {
        await axios.delete(`/api/admin/subscription/${selectedSubscription.value.id}`)
        
        showDeleteSubscription.value = false
        await fetchData()
        alert('订阅删除成功')
    } catch (error) {
        console.error('Delete subscription failed:', error)
        alert(error.response?.data || '删除订阅失败')
    } finally {
        actionLoading.value = false
    }
}

// 获取任务状态的 CSS 类
const getTaskStatusClass = (status) => {
    switch (status) {
        case 'pending':
            return 'bg-yellow-100 text-yellow-700'
        case 'running':
            return 'bg-blue-100 text-blue-700'
        case 'completed':
            return 'bg-green-100 text-green-700'
        case 'failed':
            return 'bg-red-100 text-red-700'
        default:
            return 'bg-slate-100 text-slate-600'
    }
}

onMounted(() => {
    fetchData()
})
</script>
