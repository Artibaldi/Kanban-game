<script setup lang="ts">
// «⋯» menu of a card: open it, or move it to another column. Unavailable moves stay visible with the reason.
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { columns, moveBlock } from '../lib/board'
import { ease, reduced } from '../ui/motion'
import type { Task, TaskState } from '../types'
import type { Viewer } from '../lib/board'

const props = defineProps<{ task: Task; anchor: HTMLElement; viewer: Viewer; canOpen?: boolean }>()
const emit = defineEmits<{ close: [refocus: boolean]; open: []; move: [to: TaskState] }>()
const el = ref<HTMLElement>()
const pos = ref({ left: 0, top: 0 })

const items = computed(() => columns.map(c => {
  const here = c.state === props.task.state
  const block = here ? '' : moveBlock(props.task, c.state, props.viewer)
  const label = c.state === 'done' && props.task.state === 'review' && !block ? 'Подтвердить' : c.name
  return { state: c.state, label, disabled: here || !!block, note: here ? 'сейчас здесь' : block }
}))

const enabled = () => [...(el.value?.querySelectorAll<HTMLButtonElement>('button:not([disabled])') || [])]
function onKey(e: KeyboardEvent) {
  const it = enabled(), i = it.indexOf(document.activeElement as HTMLButtonElement)
  if (e.key === 'ArrowDown') { e.preventDefault(); it[(i + 1) % it.length]?.focus() }
  else if (e.key === 'ArrowUp') { e.preventDefault(); it[(i - 1 + it.length) % it.length]?.focus() }
  else if (e.key === 'Home') { e.preventDefault(); it[0]?.focus() }
  else if (e.key === 'End') { e.preventDefault(); it[it.length - 1]?.focus() }
  else if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); emit('close', true) }
  else if (e.key === 'Tab') emit('close', false)
}
function outside(e: PointerEvent) {
  if (!el.value?.contains(e.target as Node) && !props.anchor.contains(e.target as Node)) emit('close', false)
}
onMounted(async () => {
  await nextTick()
  const r = props.anchor.getBoundingClientRect(), m = el.value!
  const mw = m.offsetWidth, mh = m.offsetHeight
  let top = r.bottom + 8
  if (top + mh > innerHeight - 12) top = Math.max(12, r.top - mh - 8)
  pos.value = { left: Math.min(innerWidth - mw - 12, Math.max(12, r.right - mw)), top }
  m.animate(reduced() ? [{ opacity: 0 }, { opacity: 1 }] : [{ opacity: 0, transform: 'translateY(-4px)' }, { opacity: 1, transform: 'none' }], { duration: reduced() ? 150 : 200, easing: ease.out })
  enabled()[0]?.focus()
  document.addEventListener('pointerdown', outside, true)
})
onUnmounted(() => document.removeEventListener('pointerdown', outside, true))
</script>

<template>
  <Teleport to="body">
    <div ref="el" class="menu" role="menu" :aria-label="`Действия с ${task.number}`" :style="{ left: pos.left + 'px', top: pos.top + 'px' }" @keydown="onKey">
      <template v-if="canOpen">
        <button role="menuitem" @click="emit('open')">Открыть карточку</button>
        <hr />
      </template>
      <div class="mh" aria-hidden="true">Переместить в…</div>
      <button v-for="i in items" :key="i.state" role="menuitem" :class="'s-' + i.state" :disabled="i.disabled" :aria-disabled="i.disabled || undefined"
        @click="emit('move', i.state)">
        <span class="sw" aria-hidden="true" />{{ i.label }}<small v-if="i.note === 'сейчас здесь'">{{ i.note }}</small>
        <span v-else-if="i.note" class="reason">{{ i.note }}</span>
      </button>
    </div>
  </Teleport>
</template>

<style scoped>
.menu { position: fixed; z-index: 250; min-width: 236px; max-width: 300px; padding: 6px; border-radius: 16px; background: #fffdf8; color: #1e1e1e; box-shadow: 0 22px 50px rgba(40, 25, 10, 0.24), 0 0 0 1px rgba(30, 30, 30, 0.06); }
.mh { font-size: 11px; color: #6b6560; padding: 8px 10px 4px; text-transform: uppercase; letter-spacing: 0.06em; font-weight: 500; }
button { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; width: 100%; padding: 9px 10px; border: 0; border-radius: 10px; background: none; font-size: 14px; text-align: left; cursor: pointer; color: inherit; }
button:hover:not(:disabled) { background: #f5eadb; }
button:focus-visible { outline-offset: -3px; }
button:disabled { color: #8e8780; cursor: default; }
small { margin-left: auto; font-size: 11px; color: #6b6560; }
.sw { width: 14px; height: 14px; border-radius: 5px; background: var(--fill); box-shadow: inset 0 0 0 1px rgba(30, 30, 30, 0.18); flex: none; }
hr { border: 0; height: 1px; background: rgba(30, 30, 30, 0.08); margin: 4px 6px; }
.reason { flex-basis: 100%; padding-left: 24px; font-size: 11px; line-height: 1.35; color: #8e8780; }
</style>
