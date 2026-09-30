import { defineStore } from 'pinia'
import { api } from '../api'
import type { GameConfig, Player } from '../types'

export const useGame = defineStore('game', {
  state: () => ({ config: null as GameConfig | null, player: null as Player | null }),
  actions: {
    async loadConfig() {
      if (!this.config) this.config = await api.get<GameConfig>('/game/config')
      return this.config
    },
    async refresh() {
      try { this.player = await api.get<Player>('/player') } catch { this.player = null }
    },
    setPlayer(p: Player | undefined) { if (p) this.player = p },
    building(type: string) { return this.config?.buildings.find(b => b.type === type) },
    troop(type: string) { return this.config?.troops.find(t => t.type === type) },
  },
})
