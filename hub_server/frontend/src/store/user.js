import { defineStore } from 'pinia'
import axios from 'axios'

export const useUserStore = defineStore('user', {
    state: () => ({
        user: null,
        token: localStorage.getItem('token') || null,
        loading: false,
        error: null
    }),

    getters: {
        isAuthenticated: (state) => !!state.token,
    },

    actions: {
        async register(email, password) {
            this.loading = true
            this.error = null
            try {
                const res = await axios.post('/api/auth/register', { email, password })
                this.setAuth(res.data)
                return true
            } catch (err) {
                this.error = err.response?.data || err.message
                return false
            } finally {
                this.loading = false
            }
        },

        async login(email, password) {
            this.loading = true
            this.error = null
            try {
                const res = await axios.post('/api/auth/login', { email, password })
                this.setAuth(res.data)
                return true
            } catch (err) {
                this.error = err.response?.data || err.message
                return false
            } finally {
                this.loading = false
            }
        },

        async fetchProfile() {
            if (!this.token) return
            try {
                const res = await axios.get('/api/auth/profile')
                this.user = res.data
            } catch (err) {
                // Token invalid
                this.logout()
            }
        },

        setAuth(data) {
            this.token = data.token
            this.user = data.user
            localStorage.setItem('token', data.token)
            axios.defaults.headers.common['Authorization'] = `Bearer ${data.token}`
        },

        logout() {
            this.token = null
            this.user = null
            localStorage.removeItem('token')
            delete axios.defaults.headers.common['Authorization']
        },

        // Initialize from local storage
        async initAuth() {
            if (this.token) {
                axios.defaults.headers.common['Authorization'] = `Bearer ${this.token}`
                await this.fetchProfile()
            }
        }
    }
})
