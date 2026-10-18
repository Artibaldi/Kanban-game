import { defineStore } from 'pinia'
import { api } from '../api'
import type { Task } from '../types'

// The review queue size, shown on the navigation for leads and kept fresh by the pages that change it.
export const useWork = defineStore('work', {
  state: () => ({ reviewCount: 0 }),
  actions: {
    async refreshReview() {
      try { this.reviewCount = (await api.get<{ review: Task[] }>('/review')).review.length } catch { this.reviewCount = 0 }
    },
  },
})
