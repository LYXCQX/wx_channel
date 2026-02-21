import { defineStore } from 'pinia'
import axios from 'axios'

export const useTaskStore = defineStore('task', {
    state: () => ({
        tasks: [],
        total: 0,
        loading: false,
        page: 1,
        pageSize: 20,
        error: null
    }),

    actions: {
        async fetchTasks(page = 1) {
            this.loading = true
            this.page = page
            try {
                const offset = (this.page - 1) * this.pageSize
                const res = await axios.get(`/api/tasks?offset=${offset}&limit=${this.pageSize}`)
                this.tasks = res.data.list || []
                this.total = res.data.total || 0
            } catch (err) {
                this.error = err.message
                console.error('Failed to fetch tasks:', err)
            } finally {
                this.loading = false
            }
        }
    }
})
