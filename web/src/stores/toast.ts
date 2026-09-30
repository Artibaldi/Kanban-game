import { defineStore } from 'pinia'

export interface Toast { id: number; kind: 'success' | 'error' | 'info' | 'reward'; title: string; text?: string }
let seq = 0

export const useToasts = defineStore('toasts', {
  state: () => ({ items: [] as Toast[] }),
  actions: {
    push(t: Omit<Toast, 'id'>, ms = 4500) {
      const id = ++seq
      this.items.push({ ...t, id })
      setTimeout(() => this.dismiss(id), ms)
    },
    error(e: unknown) {
      this.push({ kind: 'error', title: e instanceof Error ? e.message : 'Не удалось выполнить действие.' }, 6500)
    },
    dismiss(id: number) { this.items = this.items.filter(t => t.id !== id) },
  },
})
