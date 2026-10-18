<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import Icon from '../ui/Icon.vue'
import { useToasts } from '../stores/toast'
const toasts = useToasts()
// Ctrl/Cmd+Z undoes the newest undoable action, unless the user is typing.
function onKey(e: KeyboardEvent) {
  if (!(e.ctrlKey || e.metaKey) || e.shiftKey || e.key.toLowerCase() !== 'z') return
  const el = e.target as HTMLElement
  if (el.closest('input, textarea, select, [contenteditable="true"]')) return
  const t = toasts.lastAction
  if (!t) return
  e.preventDefault()
  toasts.act(t.id)
}
onMounted(() => addEventListener('keydown', onKey))
onUnmounted(() => removeEventListener('keydown', onKey))
const icon = { success: 'check', error: 'alert', info: 'info', reward: 'coins' } as const
</script>

<template>
  <div class="toasts" role="status" aria-live="polite">
    <TransitionGroup name="toast">
      <div v-for="t in toasts.items" :key="t.id" class="toast" :class="t.kind" @mouseenter="toasts.pause(t.id)" @mouseleave="toasts.resume(t.id)" @focusin="toasts.pause(t.id)" @focusout="toasts.resume(t.id)">
        <span class="ic"><Icon :name="icon[t.kind]" :size="18" /></span>
        <div class="grow msg">
          <strong>{{ t.title }}</strong>
          <span v-if="t.text" class="text">{{ t.text }}</span>
        </div>
        <button v-if="t.action" class="act" :aria-keyshortcuts="'Control+Z Meta+Z'" title="Или Ctrl+Z (⌘Z)" @click="toasts.act(t.id)">{{ t.action.label }}</button>
        <button v-else class="x" aria-label="Закрыть уведомление" @click="toasts.dismiss(t.id)"><Icon name="x" :size="14" /></button>
        <span v-if="t.action" class="timer" :style="{ animationDuration: t.ms + 'ms', animationPlayState: t.paused ? 'paused' : 'running' }" aria-hidden="true" />
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toasts { position: fixed; left: 50%; bottom: 28px; transform: translateX(-50%); z-index: 300; display: flex; flex-direction: column; align-items: center; gap: var(--space-2); width: max-content; max-width: calc(100vw - 32px); pointer-events: none; }
.toast { position: relative; overflow: hidden; pointer-events: auto; display: flex; align-items: center; gap: var(--space-3); max-width: 100%; padding: 8px 8px 8px 10px; border-radius: var(--radius-xl); background: #1e1e1e; color: #fff; box-shadow: 0 20px 44px rgba(30, 20, 10, 0.32); font-size: var(--text-sm); }
.ic { display: grid; place-items: center; width: 32px; height: 32px; border-radius: 50%; background: rgba(255, 255, 255, 0.12); flex: none; }
.reward .ic { background: var(--gold-fill); color: var(--gold-ink); }
.error .ic { background: #c24a20; }
.success .ic { color: #7fe0b0; }
.msg { display: flex; flex-direction: column; min-width: 0; padding-right: 4px; }
.msg strong { font-weight: 600; }
.text { color: #c9c4bd; font-size: var(--text-xs); line-height: 1.4; }
.act { flex: none; height: 34px; padding: 0 16px; border: 0; border-radius: var(--radius-pill); background: var(--gold-fill); color: #1e1e1e; font-weight: 700; font-size: var(--text-sm); cursor: pointer; transition: transform 80ms var(--ease-emphasized); }
.act:active { transform: scale(0.98); }
.act:focus-visible, .x:focus-visible { outline-color: #fff; }
.x { flex: none; display: grid; place-items: center; width: 32px; height: 32px; border: 0; border-radius: 50%; background: transparent; color: #c9c4bd; cursor: pointer; }
.x:hover { background: rgba(255, 255, 255, 0.1); color: #fff; }
.timer { position: absolute; left: 0; right: 0; bottom: 0; height: 3px; background: var(--gold-fill); transform-origin: left; animation: drain linear forwards; }
@keyframes drain { from { transform: scaleX(1); } to { transform: scaleX(0); } }
.toast-enter-active { transition: transform var(--duration-base) var(--ease-emphasized), opacity var(--duration-base) var(--ease-emphasized); }
.toast-leave-active { transition: opacity 150ms var(--ease-exit); }
.toast-enter-from { opacity: 0; transform: translateY(16px); }
.toast-leave-to { opacity: 0; }
@media (max-width: 767px) { .toasts { bottom: calc(96px + env(safe-area-inset-bottom)); } .toast { width: calc(100vw - 32px); } }
</style>
