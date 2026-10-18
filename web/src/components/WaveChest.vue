<script setup lang="ts">
// The team chest of a wave. What is inside is shown before it opens: no lottery. The server has already paid it.
import { nextTick, onMounted, onUnmounted, ref, shallowRef } from 'vue'
import Icon from '../ui/Icon.vue'
import { reduced } from '../ui/motion'
import type { ChestStage } from '../game3d/chestStage'

const props = defineProps<{ gold: number; energy: number; repair: number; crowns: number }>()
const emit = defineEmits<{ close: [] }>()
const canvas = ref<HTMLCanvasElement>()
const root = ref<HTMLElement>()
const stage = shallowRef<ChestStage | null>(null)
const phase = ref<'closed' | 'opening' | 'open'>('closed')
const shown = ref([0, 0, 0])
const flat = ref(false)
/** The chest can be opened once it has landed (or the flat fallback is chosen). */
const ready = ref(false)
let unmounted = false
const timers: number[] = []

const cards = [
  { cls: 'gold', hd: 'Золото', lbl: 'каждому участнику' },
  { cls: 'energy', hd: 'Энергия', lbl: 'каждому участнику' },
  { cls: 'repair', hd: 'Ремонт', lbl: 'стенам крепости' },
]
const amounts = () => [props.gold, props.energy, props.repair]

onMounted(async () => {
  addEventListener('keydown', onKey)
  nextTick(() => root.value?.querySelector<HTMLElement>('.open-btn')?.focus())
  try {
    const mod = await import('../game3d/chestStage')
    const ok = (await import('../game3d/fortressStage')).webglAvailable()
    if (unmounted) return
    if (!ok || !canvas.value) throw new Error('no webgl')
    const s = new mod.ChestStage(canvas.value, reduced())
    s.onLanded = () => { ready.value = true }
    s.onLost = () => { flat.value = true; ready.value = true }
    stage.value = s
  } catch { if (!unmounted) { flat.value = true; ready.value = true } }
})
onUnmounted(() => { unmounted = true; removeEventListener('keydown', onKey); timers.forEach(clearTimeout); stage.value?.dispose() })
function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') { emit('close'); return }
  // Keep Tab inside the chest dialog.
  if (e.key === 'Tab') {
    const f = [...(root.value?.querySelectorAll<HTMLElement>('button:not([disabled])') || [])]
    if (!f.length) { e.preventDefault(); return }
    const i = f.indexOf(document.activeElement as HTMLElement)
    if (e.shiftKey && i <= 0) { e.preventDefault(); f[f.length - 1].focus() }
    else if (!e.shiftKey && (i === -1 || i === f.length - 1)) { e.preventDefault(); f[0].focus() }
  }
}

function open() {
  if (phase.value !== 'closed' || !ready.value) return
  phase.value = 'opening'
  const burst = () => {
    phase.value = 'open'
    // Each card counts up as it lands.
    amounts().forEach((to, i) => {
      if (reduced()) { shown.value[i] = to; return }
      timers.push(window.setTimeout(() => {
        const t0 = performance.now()
        const step = (now: number) => { const q = Math.min(1, (now - t0) / 500); shown.value[i] = Math.round(to * (1 - Math.pow(1 - q, 3))); if (q < 1 && !unmounted) requestAnimationFrame(step) }
        requestAnimationFrame(step)
        // Land on the exact amount even if frames were dropped.
        timers.push(window.setTimeout(() => { shown.value[i] = to }, 520))
      }, 320 + i * 420))
    })
    nextTick(() => root.value?.querySelector<HTMLElement>('.take')?.focus())
  }
  if (stage.value) stage.value.open(burst)
  else burst()
}
</script>

<template>
  <Teleport to="body">
    <div ref="root" class="chest" role="dialog" aria-modal="true" aria-label="Сундук волны">
      <div class="rays" aria-hidden="true" />
      <canvas v-show="!flat" ref="canvas" aria-hidden="true" />
      <div v-if="flat" class="flat" aria-hidden="true"><Icon name="sparkles" :size="96" /></div>
      <div class="top">
        <span class="g-ribbon gold"><b>Сундук волны</b></span>
        <p class="known">Внутри, без сюрпризов: <b>{{ gold }} золота</b> · <b>{{ energy }} энергии</b> · <b>{{ repair >= 100 ? 'полный ремонт' : `+${repair}% стенам` }}</b></p>
      </div>
      <div v-if="phase === 'open'" class="cards" :class="{ still: reduced() }">
        <div v-for="(c, i) in cards" :key="c.cls" class="card" :class="c.cls" :style="{ animationDelay: i * 420 + 'ms' }">
          <div class="hd g-stroke s2">{{ c.hd }}</div>
          <div class="art">
            <span v-if="c.cls === 'gold'" class="g-coin big" />
            <svg v-else-if="c.cls === 'energy'" viewBox="0 0 32 40" width="52" height="64"><path d="M19 2 4 23h10l-3 15 17-23H17z" fill="#5FD8FF" stroke="#1E1E1E" stroke-width="3" stroke-linejoin="round" /></svg>
            <svg v-else viewBox="0 0 36 40" width="54" height="60"><path d="M18 2 4 7v11c0 9 6 16 14 20 8-4 14-11 14-20V7z" fill="#3E8EF0" stroke="#1E1E1E" stroke-width="3" stroke-linejoin="round" /><path d="M18 8v24c5-3 8-8 8-14v-6z" fill="#7DB8FF" /></svg>
          </div>
          <div class="amt g-stroke">{{ c.cls === 'repair' ? (repair >= 100 ? '100%' : `+${shown[i]}%`) : `+${shown[i]}` }}</div>
          <div class="lbl">{{ c.lbl }}</div>
        </div>
      </div>
      <div class="bottom">
        <button v-if="phase !== 'open'" class="g-btn yellow big open-btn" :disabled="!ready || phase === 'opening'" :aria-busy="phase === 'opening'" @click="open"><span>Открыть сундук</span></button>
        <template v-else-if="phase === 'open'">
          <p class="paid">Награда уже в казне у каждого участника команды.</p>
          <button class="g-btn big take" @click="emit('close')"><span>Забрать</span></button>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.chest { position: fixed; inset: 0; z-index: 200; background: radial-gradient(ellipse at 50% 45%, #5c4fc4 0%, #3a2f8e 45%, #1d1850 100%); color: #fff; overflow: hidden; }
.rays { position: absolute; left: 50%; top: 48%; width: 1400px; height: 1400px; margin: -700px 0 0 -700px; pointer-events: none; background: repeating-conic-gradient(from 0deg, rgba(255, 236, 160, 0.16) 0 9deg, transparent 9deg 22.5deg); -webkit-mask: radial-gradient(circle, #000 0, transparent 60%); mask: radial-gradient(circle, #000 0, transparent 60%); animation: spin 8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
canvas { position: absolute; inset: 0; width: 100%; height: 100%; display: block; }
.flat { position: absolute; inset: 0; display: grid; place-items: center; color: var(--g-gold); }
.top { position: absolute; top: 48px; left: 0; right: 0; text-align: center; padding: 0 16px; }
.known { display: block; width: fit-content; margin: 18px auto 0; padding: 8px 16px; border: 2px solid var(--g-ink); border-radius: 999px; background: rgba(255, 255, 255, 0.92); font: 600 14px var(--font-sans); color: #2b2466; }
.cards { position: absolute; left: 0; right: 0; bottom: 190px; display: flex; justify-content: center; gap: 22px; padding: 0 12px; }
.card { width: 150px; height: 196px; border: 3px solid var(--g-ink); border-radius: 16px; background: var(--g-panel); box-shadow: inset 0 0 0 3px #fff6dd, 0 6px 0 #3e2914, 0 6px 0 3px var(--g-ink); overflow: hidden; display: flex; flex-direction: column; align-items: center; animation: fly 0.52s cubic-bezier(0.34, 1.56, 0.64, 1) both; }
.still .card { animation: none; }
@keyframes fly { from { transform: translateY(-180px) scale(0.3) rotateY(180deg); opacity: 0; } 60% { transform: translateY(-60px) scale(1.1) rotateY(70deg); opacity: 1; } to { transform: none; opacity: 1; } }
.hd { width: 100%; padding: 6px 0 8px; text-align: center; font-size: 15px; background: linear-gradient(#ffec8a, #ffd323 55%, #eca800); border-bottom: 3px solid var(--g-ink); }
.energy .hd { background: linear-gradient(#9be2ff, #45b8f5 55%, #2b93d8); }
.repair .hd { background: linear-gradient(#b7f39a, #5cc02e 55%, #3f9a1f); }
.art { flex: 1; display: grid; place-items: center; }
.g-coin.big { width: 64px; height: 64px; }
.g-coin.big::after { width: 18px; height: 18px; border-width: 2px; }
.amt { font-size: 28px; margin-bottom: 2px; }
.lbl { font: 700 12px var(--font-sans); color: #5a3a1e; margin-bottom: 10px; text-align: center; padding: 0 6px; }
.bottom { position: absolute; left: 0; right: 0; bottom: 40px; text-align: center; padding: 0 16px; }
.paid { margin-bottom: 14px; font: 600 14px var(--font-sans); color: #e6e1ff; }
@media (max-width: 600px) {
  .cards { gap: 8px; bottom: 170px; }
  .card { width: 104px; height: 144px; }
  .hd { font-size: 12px; }
  .amt { font-size: 21px; }
  .lbl { display: none; }
  .art { transform: scale(0.8); }
  .known { font-size: 12px; }
}
@media (prefers-reduced-motion: reduce) { .rays { animation: none; } }
</style>
