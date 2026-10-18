import { defineStore } from 'pinia'

export interface ToastAction { label: string; run: () => void }
export interface Toast { id: number; kind: 'success' | 'error' | 'info' | 'reward'; title: string; text?: string; action?: ToastAction; ms: number; paused: boolean }
let seq = 0
const timers = new Map<number, { handle: number; started: number; left: number }>()

export const useToasts = defineStore('toasts', {
  state: () => ({ items: [] as Toast[] }),
  getters: {
    /** The newest notice that can still be undone (for Ctrl/Cmd+Z). */
    lastAction: s => [...s.items].reverse().find(t => t.action),
  },
  actions: {
    push(t: Omit<Toast, 'id' | 'ms' | 'paused'>, ms = 4500) {
      const id = ++seq
      // An action needs time to be read and reached: never shorter than eight seconds, and it pauses on hover or focus.
      if (t.action) ms = Math.max(ms, 8000)
      this.items.push({ ...t, id, ms, paused: false })
      // Plain notices make room; a notice with an action is never pushed out by them.
      const plain = this.items.filter(x => !x.action)
      if (this.items.length > 4 && plain.length) this.dismiss(plain[0].id)
      this.schedule(id, ms)
      return id
    },
    schedule(id: number, left: number) {
      timers.set(id, { handle: window.setTimeout(() => this.dismiss(id), left), started: Date.now(), left })
    },
    pause(id: number) {
      const tm = timers.get(id), t = this.items.find(x => x.id === id)
      if (!tm || !t || t.paused) return
      clearTimeout(tm.handle)
      tm.left -= Date.now() - tm.started
      t.paused = true
    },
    resume(id: number) {
      const tm = timers.get(id), t = this.items.find(x => x.id === id)
      if (!tm || !t || !t.paused) return
      t.paused = false
      this.schedule(id, Math.max(1500, tm.left))
    },
    error(e: unknown) {
      this.push({ kind: 'error', title: e instanceof Error ? e.message : 'Не удалось выполнить действие.' }, 6500)
    },
    act(id: number) {
      const t = this.items.find(x => x.id === id)
      this.dismiss(id)
      t?.action?.run()
    },
    dismiss(id: number) {
      const tm = timers.get(id)
      if (tm) clearTimeout(tm.handle)
      timers.delete(id)
      this.items = this.items.filter(t => t.id !== id)
    },
  },
})
