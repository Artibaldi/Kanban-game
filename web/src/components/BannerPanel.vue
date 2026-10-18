<script setup lang="ts">
// The player's own team banner: who lent which troops, a release button, and the team's wave crowns.
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'
import type { BannerState } from '../lib/banner'
import { useGame } from '../stores/game'
import { useToasts } from '../stores/toast'

const game = useGame()
const toasts = useToasts()
const state = ref<BannerState | null>(null)
const busy = ref(false)

onMounted(async () => { try { state.value = await api.get<BannerState>('/banner') } catch (e) { toasts.error(e) } })
// The last waves, newest last: the banner shows the team's crowns, never anyone's share.
const recent = computed(() => (state.value?.crowns || []).slice(-8))

async function release() {
  busy.value = true
  try {
    state.value = await api.post<BannerState>('/banner/release', {})
    toasts.push({ kind: 'info', title: 'Подкрепления вернулись к коллегам' }, 2500)
  } catch (e) { toasts.error(e) } finally { busy.value = false }
}
</script>

<template>
  <div v-if="state" class="banner-panel">
    <h3>Подкрепления: {{ state.held }} из {{ state.capacity }} мест</h3>
    <p v-if="!state.reinforcements.length" class="muted">Пока пусто. Коллеги отправляют войска со страницы «Казарма».</p>
    <ul v-else class="lent">
      <li v-for="r in state.reinforcements" :key="r.donor_id + r.type">{{ game.troop(r.type)?.name }} × {{ r.count }} <span class="muted">от {{ r.donor_name }}</span></li>
    </ul>
    <button v-if="state.reinforcements.length" class="g-btn red small" :disabled="busy" @click="release"><span>Отпустить всех</span></button>
    <div class="crowns" :aria-label="`Короны последних волн команды: ${recent.join(', ') || 'волн ещё не было'}`">
      <span class="muted">Короны волн команды:</span>
      <span v-for="(c, i) in recent" :key="i" class="wave" :title="`${c} из 3`"><i v-for="n in 3" :key="n" :class="{ on: n <= c }">♛</i></span>
      <span v-if="!recent.length" class="muted">волн ещё не было</span>
    </div>
  </div>
</template>

<style scoped>
.banner-panel { display: flex; flex-direction: column; gap: 6px; margin-top: 8px; padding: 10px; border-radius: 14px; background: #fff8e6; border: 2px solid var(--g-panel-b); color: #3a2410; }
h3 { margin: 0; font: 800 15px var(--font-game); }
.lent { margin: 0; padding-left: 18px; font: 600 13px var(--font-sans); }
.muted { font: 600 12px var(--font-sans); color: #7a5530; }
.crowns { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; }
.wave { display: inline-flex; padding: 1px 4px; border-radius: 8px; background: #fff; border: 1px solid var(--g-panel-b); }
.wave i { font-style: normal; color: #d8cdb5; font-size: 13px; }
.wave i.on { color: #e6a800; }
</style>
