import { defineStore } from 'pinia'
import axios from 'axios'

export const useClientStore = defineStore('client', {
    state: () => ({
        clients: [], // 在线客户端列表
        loading: false,
        error: null,
        lastUpdated: null,
        currentClient: null, // 当前选中的客户端 (for remote ops)
    }),

    getters: {
        onlineCount: (state) => state.clients.length,
        getClientById: (state) => (id) => state.clients.find(c => c.id === id)
    },

    actions: {
        async fetchClients() {
            this.loading = true
            try {
                const res = await axios.get('/api/clients')
                this.clients = res.data
                this.lastUpdated = new Date()

                // Restore selection if exists
                const savedId = localStorage.getItem('hub_last_client_id')
                if (savedId && !this.currentClient) {
                    this.setCurrentClient(savedId)
                } else if (this.currentClient) {
                    // Update current client object with latest data
                    this.setCurrentClient(this.currentClient.id)
                }
            } catch (err) {
                this.error = err.message
                console.error('Failed to fetch clients:', err)
            } finally {
                this.loading = false
            }
        },

        setCurrentClient(clientId) {
            const client = this.getClientById(clientId) || null
            this.currentClient = client
            if (client) {
                localStorage.setItem('hub_last_client_id', clientId)
            }
        },

        // 远程调用通用方法
        async remoteCall(action, payload) {
            // 如果没有选中客户端，传递空 client_id，让后端自动选择
            const clientId = this.currentClient ? this.currentClient.id : ''

            const res = await axios.post('/api/call', {
                client_id: clientId,
                action: action,
                data: payload
            })

            // Hub Server returns { request_id, success, data, error }
            if (!res.data.success) {
                throw new Error(res.data.error || 'Remote call failed')
            }
            return res.data
        }
    }
})
