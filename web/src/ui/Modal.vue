<script setup lang="ts">
// Native <dialog>: focus is trapped and Escape closes it without extra code.
import { onMounted, ref, watch } from 'vue'
import Icon from './Icon.vue'
const props = defineProps<{ open: boolean; title: string; wide?: boolean; side?: boolean }>()
const emit = defineEmits<{ close: [] }>()
const el = ref<HTMLDialogElement>()
function sync() {
  if (!el.value) return
  if (props.open && !el.value.open) el.value.showModal()
  if (!props.open && el.value.open) el.value.close()
}
onMounted(sync)
watch(() => props.open, sync)
function backdrop(e: MouseEvent) { if (e.target === el.value) emit('close') }
</script>

<template>
  <dialog ref="el" class="modal" :class="{ wide, side }" :aria-label="title" @close="emit('close')" @click="backdrop">
    <div class="inner">
      <header class="mhead">
        <h2>{{ title }}</h2>
        <button class="btn ghost icon" aria-label="Закрыть" @click="emit('close')"><Icon name="x" /></button>
      </header>
      <div class="mbody"><slot /></div>
      <footer v-if="$slots.footer" class="mfoot"><slot name="footer" /></footer>
    </div>
  </dialog>
</template>

<style scoped>
.modal { border: 0; padding: 0; border-radius: var(--radius-lg); background: var(--color-surface); color: var(--color-text); box-shadow: var(--shadow-lg); width: min(520px, calc(100vw - 32px)); max-height: calc(100vh - 48px); }
.modal.wide { width: min(760px, calc(100vw - 32px)); }
.modal.side { margin: 0 0 0 auto; height: 100vh; max-height: 100vh; border-radius: 0; width: min(640px, 100vw); }
.modal::backdrop { background: rgba(10, 14, 12, 0.5); }
.inner { display: flex; flex-direction: column; max-height: inherit; height: 100%; }
.mhead { display: flex; align-items: center; justify-content: space-between; gap: var(--space-3); padding: var(--space-4) var(--space-5); border-bottom: 1px solid var(--color-border); }
.mhead h2 { font-size: var(--text-lg); }
.mbody { padding: var(--space-5); overflow-y: auto; flex: 1; }
.mfoot { display: flex; justify-content: flex-end; gap: var(--space-2); padding: var(--space-3) var(--space-5); border-top: 1px solid var(--color-border); flex-wrap: wrap; }
@media (max-width: 767px) { .modal.side { width: 100vw; } .mbody { padding: var(--space-4); } }
</style>
