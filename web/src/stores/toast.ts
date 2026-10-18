import { defineStore } from 'pinia'

export interface ToastAction { label: string; run: () => void }
export interface Toast { id: number; kind: 'success' | 'error' | 'info' | 'reward'; title: string; text?: string; action?: ToastAction; ms: number }
let seq = 0

export const useToasts = defineStore('toasts', {
  state: () => ({ items: [] as Toast[] }),
  actions: {
    push(t: Omit<Toast, 'id' | 'ms'>, ms = 4500) {
      const id = ++seq
      // An action needs time to be read and reached: never shorter than six seconds.
      if (t.action) ms = Math.max(ms, 6000)
      this.items = [...this.items.filter(x => !(x.action && t.action)), { ...t, id, ms }].slice(-3)
      setTimeout(() => this.dismiss(id), ms)
      return id
    },
    error(e: unknown) {
      this.push({ kind: 'error', title: e instanceof Error ? e.message : 'Не удалось выполнить действие.' }, 6500)
    },
    act(id: number) {
      const t = this.items.find(x => x.id === id)
      this.dismiss(id)
      t?.action?.run()
    },
    dismiss(id: number) { this.items = this.items.filter(t => t.id !== id) },
  },
})
