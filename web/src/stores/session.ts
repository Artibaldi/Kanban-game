import { defineStore } from 'pinia'
import { api, setCsrf } from '../api'
import type { Me } from '../types'

export const useSession = defineStore('session', {
  state: () => ({ me: null as Me | null, loaded: false }),
  getters: {
    isLead: s => s.me?.role === 'lead' || s.me?.role === 'admin',
    isAdmin: s => s.me?.role === 'admin',
    plays: s => !!s.me?.plays,
    gameOn: s => !!s.me?.plays && !!s.me?.game_mode,
  },
  actions: {
    async load() {
      try { this.me = await api.get<Me>('/me') } catch { this.me = null }
      this.loaded = true
      this.applyMode()
    },
    async login(login: string) {
      const r = await api.post<{ csrf_token: string }>('/sessions', { login })
      setCsrf(r.csrf_token)
      await this.load()
    },
    async logout() {
      try { await api.post('/sessions/logout') } finally { this.me = null; this.applyMode() }
    },
    async setGameMode(on: boolean) {
      this.me = await api.patch<Me>('/me/settings', { game_mode: on })
      this.applyMode()
    },
    applyMode() {
      document.body.classList.toggle('work-mode', !this.gameOn)
    },
  },
})
