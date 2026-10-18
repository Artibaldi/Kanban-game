<script setup lang="ts">
// «Подкрепления коллегам»: send ready troops to a teammate's banner. They defend that city in friendly duels,
// never die there, and come home when the owner releases them. No tallies of who gave what.
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { donateBlock, type BannerState } from '../lib/banner'
import { useGame } from '../stores/game'
import { useToasts } from '../stores/toast'
import Portrait3D from './Portrait3D.vue'

const emit = defineEmits<{ changed: [] }>()
const game = useGame()
const toasts = useToasts()
const state = ref<BannerState | null>(null)
const pick = ref<Record<string, { type: string; count: number }>>({})
const busy = ref('')

async function load() {
  try {
    state.value = await api.get<BannerState>('/banner')
    for (const m of state.value.team) pick.value[m.user_id!] ??= { type: firstReady(), count: 1 }
  } catch (e) { toasts.error(e) }
}
const firstReady = () => Object.entries(state.value?.troops || {}).find(([, n]) => n > 0)?.[0] || 'guard'
onMounted(load)
defineExpose({ load })

const housing = (type: string) => game.troop(type)?.housing || 1
const block = (id: string) => {
  const m = state.value?.team.find(x => x.user_id === id), p = pick.value[id]
  if (!m || !p) return 'Загрузка…'
  return donateBlock(m, housing(p.type), p.count, state.value?.troops[p.type] || 0)
}
async function send(id: string, name: string) {
  const p = pick.value[id]
  busy.value = id
  try {
    state.value = await api.post<BannerState>('/banner/donate', { to_id: id, type: p.type, count: p.count })
    toasts.push({ kind: 'success', title: `${name}: подкрепление на знамени` }, 2500)
    emit('changed')
  } catch (e) { toasts.error(e) } finally { busy.value = '' }
}
</script>

<template>
  <section class="g-panel reinforce" aria-labelledby="rf-title">
    <span class="g-ribbon"><b id="rf-title">Подкрепления коллегам</b></span>
    <p class="empty">Готовые войска можно отправить на знамя команды коллеги: они защищают его город в дружеских боях, не гибнут и вернутся к вам, когда он их отпустит.</p>
    <p v-if="state && !state.team.length" class="empty">В команде пока никого, кроме вас.</p>
    <ul v-if="state" class="mates">
      <li v-for="m in state.team" :key="m.user_id" class="mate">
        <div class="who"><strong>{{ m.display_name }}</strong>
          <span class="muted">{{ m.capacity ? `знамя: занято ${m.held} из ${m.capacity}` : 'знамени пока нет' }}</span></div>
        <template v-if="m.capacity && pick[m.user_id!]">
          <label class="sr-only" :for="`rf-t-${m.user_id}`">Отряд для {{ m.display_name }}</label>
          <select :id="`rf-t-${m.user_id}`" v-model="pick[m.user_id!].type" class="sel">
            <option v-for="t in game.config?.troops || []" :key="t.type" :value="t.type" :disabled="!(state.troops[t.type] > 0)">{{ t.name }} (готово {{ state.troops[t.type] || 0 }})</option>
          </select>
          <label class="sr-only" :for="`rf-n-${m.user_id}`">Количество</label>
          <input :id="`rf-n-${m.user_id}`" v-model.number="pick[m.user_id!].count" type="number" min="1" step="1" class="count" />
          <Portrait3D :troop="pick[m.user_id!].type" :size="36" />
          <button class="g-btn small" :disabled="!!block(m.user_id!) || !!busy" :aria-busy="busy === m.user_id" @click="send(m.user_id!, m.display_name!)"><span>Отправить</span></button>
          <p v-if="block(m.user_id!)" class="reason">{{ block(m.user_id!) }}</p>
        </template>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.reinforce { display: flex; flex-direction: column; gap: 8px; }
.empty { text-align: center; font: 600 14px var(--font-sans); color: #7a5530; margin: 0; }
.mates { list-style: none; margin: 0; padding: 0; display: grid; gap: 8px; }
.mate { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; padding: 8px 10px; border: 2px solid var(--g-panel-b); border-radius: 12px; background: #fff8e6; color: #3a2410; }
.who { display: flex; flex-direction: column; flex: 1 1 180px; min-width: 0; }
.muted { font: 600 12px var(--font-sans); color: #7a5530; }
.sel, .count { padding: 5px 8px; border: 2px solid var(--g-panel-b); border-radius: 10px; font: 700 13px var(--font-sans); background: #fff; }
.count { width: 64px; }
.reason { flex-basis: 100%; margin: 0; font: 700 12px var(--font-sans); color: #9b1f1f; }
</style>
