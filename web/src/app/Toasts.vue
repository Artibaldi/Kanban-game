<script setup lang="ts">
import Icon from '../ui/Icon.vue'
import { useToasts } from '../stores/toast'
const toasts = useToasts()
const icon = { success: 'check', error: 'alert', info: 'info', reward: 'sparkles' } as const
</script>

<template>
  <div class="toasts" role="status" aria-live="polite">
    <TransitionGroup name="toast">
      <div v-for="t in toasts.items" :key="t.id" class="toast" :class="t.kind">
        <Icon :name="icon[t.kind]" :size="18" />
        <div class="grow">
          <strong>{{ t.title }}</strong>
          <p v-if="t.text" class="small">{{ t.text }}</p>
        </div>
        <button class="btn ghost icon sm" aria-label="Закрыть уведомление" @click="toasts.dismiss(t.id)"><Icon name="x" :size="14" /></button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toasts { position: fixed; right: var(--space-4); bottom: var(--space-4); z-index: 60; display: flex; flex-direction: column; gap: var(--space-2); width: min(380px, calc(100vw - 32px)); }
.toast { display: flex; gap: var(--space-3); align-items: flex-start; padding: var(--space-3) var(--space-3) var(--space-3) var(--space-4); border-radius: var(--radius-md); background: var(--color-surface); border: 1px solid var(--color-border); box-shadow: var(--shadow-lg); border-left: 4px solid var(--color-info); }
.toast.success { border-left-color: var(--color-success); }
.toast.error { border-left-color: var(--color-danger); }
.toast.reward { border-left-color: var(--color-gold); }
.toast.reward > .icon { color: var(--color-gold); }
.toast.error > .icon { color: var(--color-danger); }
.toast p { margin-top: 2px; color: var(--color-text-muted); }
.toast-enter-active, .toast-leave-active { transition: all var(--duration-base) var(--ease-out); }
.toast-enter-from, .toast-leave-to { opacity: 0; transform: translateY(8px); }
@media (max-width: 767px) { .toasts { bottom: 84px; left: 16px; right: 16px; width: auto; } }
</style>
