<script setup lang="ts">
// Native <dialog>: focus is trapped and Escape closes it without extra code.
// With `origin` (a CSS selector), the dialog grows out of that element and shrinks back into it.
import { onMounted, ref, watch } from 'vue'
import Icon from './Icon.vue'
import { ease, flipFrom, reduced } from './motion'
const props = defineProps<{ open: boolean; title: string; wide?: boolean; side?: boolean; sheet?: boolean; tone?: string; origin?: string; hideHead?: boolean }>()
const emit = defineEmits<{ close: [] }>()
const el = ref<HTMLDialogElement>()
let hidden: HTMLElement | null = null
let closing: Animation | null = null

const originEl = () => (props.origin ? document.querySelector<HTMLElement>(props.origin) : null)
function restore() { if (hidden) { hidden.style.visibility = ''; hidden = null } }

function sync() {
  const d = el.value
  if (!d) return
  // Reopened while the close animation runs: stop it and stay open.
  if (props.open && closing) {
    closing.onfinish = null
    d.getAnimations().forEach(a => a.cancel())
    d.querySelector('.inner')?.getAnimations().forEach(a => a.cancel())
    closing = null
    return
  }
  if (props.open && !d.open) {
    d.showModal()
    const o = originEl()
    if (reduced() || !o) { d.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 160 }); return }
    const r = o.getBoundingClientRect(), m = d.getBoundingClientRect()
    hidden = o
    o.style.visibility = 'hidden'
    d.animate([flipFrom(r, m, 20), { transform: 'none', borderRadius: '28px' }], { duration: 280, easing: ease.inOut })
    d.querySelector('.inner')?.animate([{ opacity: 0 }, { opacity: 0, offset: 0.35 }, { opacity: 1 }], { duration: 280 })
  }
  if (!props.open && d.open) {
    const o = hidden && hidden.isConnected ? hidden : originEl()
    const done = () => { closing = null; d.close(); restore(); d.getAnimations().forEach(a => a.cancel()) }
    if (reduced()) { closing = d.animate([{ opacity: 1 }, { opacity: 0 }], { duration: 150, fill: 'forwards' }); closing.onfinish = done; return }
    const r = o?.getBoundingClientRect()
    if (!o || !r || r.bottom < 0 || r.top > innerHeight) {
      closing = d.animate([{ opacity: 1 }, { opacity: 0, transform: 'scale(.98)' }], { duration: 196, easing: ease.exit, fill: 'forwards' })
      closing.onfinish = done
      return
    }
    d.querySelector('.inner')?.animate([{ opacity: 1 }, { opacity: 0, offset: 0.45 }, { opacity: 0 }], { duration: 280, fill: 'forwards' })
    closing = d.animate([{ transform: 'none', borderRadius: '28px' }, flipFrom(r, d.getBoundingClientRect(), 20)], { duration: 280, easing: ease.inOut, fill: 'forwards' })
    closing.onfinish = () => {
      d.querySelector('.inner')?.getAnimations().forEach(a => a.cancel())
      done()
    }
  }
}
onMounted(sync)
watch(() => props.open, sync)
function backdrop(e: MouseEvent) { if (e.target === el.value) emit('close') }
// Escape: let the parent close it, so the closing animation runs.
function cancel(e: Event) { e.preventDefault(); emit('close') }
</script>

<template>
  <dialog ref="el" class="modal" :class="[tone, { wide, side, sheet }]" :aria-label="title" @cancel="cancel" @close="restore(); open && emit('close')" @click="backdrop">
    <div class="inner">
      <header v-if="!hideHead" class="mhead">
        <h2>{{ title }}</h2>
        <button class="btn ghost icon" aria-label="Закрыть" @click="emit('close')"><Icon name="x" /></button>
      </header>
      <div class="mbody"><slot /></div>
      <footer v-if="$slots.footer" class="mfoot"><slot name="footer" /></footer>
    </div>
  </dialog>
</template>

<style scoped>
.modal { border: 0; padding: 0; border-radius: var(--radius-xl); background: var(--color-surface); color: var(--color-text); box-shadow: var(--shadow-lg); width: min(520px, calc(100vw - 32px)); max-height: calc(100vh - 48px); transform-origin: 0 0; }
.modal.wide { width: min(760px, calc(100vw - 32px)); }
.modal.side { margin: 0 0 0 auto; height: 100vh; max-height: 100vh; border-radius: 0; width: min(640px, 100vw); }
.modal.sheet { width: min(640px, calc(100vw - 32px)); max-height: calc(100dvh - 48px); background: var(--fill); color: var(--ink); scrollbar-width: none; }
.modal.sheet.s-backlog { box-shadow: inset 0 0 0 1px var(--line), var(--shadow-lg); }
.modal:focus-visible { outline: none; }
.modal::backdrop { background: rgba(40, 28, 16, 0.26); -webkit-backdrop-filter: blur(10px); backdrop-filter: blur(10px); }
.inner { display: flex; flex-direction: column; max-height: inherit; height: 100%; }
.mhead { display: flex; align-items: center; justify-content: space-between; gap: var(--space-3); padding: var(--space-4) var(--space-5); border-bottom: 1px solid var(--color-border); }
.mhead h2 { font-size: var(--text-lg); }
.mbody { padding: var(--space-5); overflow-y: auto; flex: 1; }
.sheet .mbody { padding: 24px 28px 26px; scrollbar-width: none; }
.mfoot { display: flex; justify-content: flex-end; gap: var(--space-2); padding: var(--space-3) var(--space-5); border-top: 1px solid var(--color-border); flex-wrap: wrap; }
@media (max-width: 767px) {
  .modal.side { width: 100vw; }
  .mbody { padding: var(--space-4); }
  .modal.sheet { width: calc(100vw - 24px); border-radius: 26px; }
  .sheet .mbody { padding: 20px 20px 22px; }
}
</style>
