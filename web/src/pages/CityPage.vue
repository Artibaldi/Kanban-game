<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import Icon from '../ui/Icon.vue'
import CityScene from '../components/CityScene.vue'
import GameOff from '../components/GameOff.vue'
import BuildingIcon from '../components/BuildingIcon.vue'
import { api } from '../api'
import { placeBlock, specLines, upgradeBlock } from '../lib/city'
import { duration, num } from '../lib/format'
import { useGame } from '../stores/game'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import type { BuildingType, City, CityBuilding } from '../types'

const game = useGame()
const session = useSession()
const toasts = useToasts()
const city = ref<City | null>(null)
const selectedId = ref<string | null>(null)
const placing = ref<BuildingType | null>(null)
const busy = ref(false)
const now = ref(Date.now())
let offset = 0
let tick = 0
let reloading = false

async function load() {
  try {
    const c = await api.get<City>('/city')
    if (c.server_time) offset = new Date(c.server_time).getTime() - Date.now()
    city.value = c
    game.setPlayer(c.player)
  } catch (e) { toasts.error(e) }
}
onMounted(async () => {
  await game.loadConfig()
  await load()
  tick = window.setInterval(async () => {
    now.value = Date.now() + offset
    const due = city.value?.buildings.some(b => b.busy_until && new Date(b.busy_until).getTime() <= now.value)
    if (due && !reloading) {
      reloading = true
      const before = city.value?.buildings.filter(b => b.busy_until).length || 0
      await load()
      const after = city.value?.buildings.filter(b => b.busy_until).length || 0
      if (after < before) toasts.push({ kind: 'success', title: 'Строительство завершено' }, 2500)
      reloading = false
    }
  }, 1000)
})
onUnmounted(() => clearInterval(tick))

const selected = computed(() => city.value?.buildings.find(b => b.id === selectedId.value) || null)
const player = computed(() => game.player)
const defs = computed(() => game.config?.buildings.filter(d => d.type !== 'townhall') || [])
const builders = computed(() => game.config?.builders || 2)
const busyList = computed(() => city.value?.buildings.filter(b => b.busy_until) || [])

function select(b: CityBuilding) { placing.value = null; selectedId.value = b.id }
function startPlacing(t: BuildingType) { selectedId.value = null; placing.value = placing.value === t ? null : t }

async function place(x: number, y: number) {
  if (!placing.value || busy.value) return
  busy.value = true
  try {
    const c = await api.post<City>('/city/buildings', { type: placing.value, x, y })
    city.value = c
    game.setPlayer(c.player)
    toasts.push({ kind: 'success', title: `${game.building(placing.value)?.name}: стройка началась` }, 2500)
    placing.value = null
  } catch (e) { toasts.error(e) } finally { busy.value = false }
}
async function upgrade(b: CityBuilding) {
  busy.value = true
  try {
    const c = await api.post<City>(`/city/buildings/${b.id}/upgrade`, {})
    city.value = c
    game.setPlayer(c.player)
    toasts.push({ kind: 'success', title: `${b.name}: улучшение до уровня ${b.level + 1}` }, 2500)
  } catch (e) { toasts.error(e) } finally { busy.value = false }
}
const left = (b: CityBuilding) => (b.busy_until ? new Date(b.busy_until).getTime() - now.value : 0)
</script>

<template>
  <GameOff v-if="!session.gameOn" />
  <div v-else class="page city-page">
    <div class="page-head">
      <div>
        <p class="eyebrow">Ратуша {{ city?.townhall || '—' }} уровня</p>
        <h1>Город {{ session.me?.display_name.split(' ')[0] }}</h1>
        <p class="muted">Золото приходит за подтверждённые задачи. Стройте, улучшайте ратушу и укрепляйте оборону.</p>
      </div>
      <div v-if="player" class="stats">
        <span class="stat"><Icon name="hammer" :size="16" />Строители {{ player.builders_busy }}/{{ player.builders }}</span>
        <span class="stat"><Icon name="shield" :size="16" />Армия {{ player.army_housing }}/{{ player.army_capacity }}</span>
      </div>
    </div>
    <p v-if="player?.debt" class="notice danger" style="margin-bottom: 16px">После отмены награды баланс отрицательный. Строительство откроется, когда следующие награды погасят долг.</p>

    <div class="layout">
      <section class="scene-wrap card" aria-label="Поле города">
        <div v-if="placing" class="placing-bar"><Icon name="info" :size="16" />Выберите свободную клетку для «{{ game.building(placing)?.name }}»<button class="btn sm" @click="placing = null">Отмена</button></div>
        <CityScene v-if="city" :buildings="city.buildings" :grid="game.config?.grid" :selected-id="selectedId" :placing="placing" :now="now"
          :label="`Город: ${city.buildings.length} зданий`" @select="select" @cell="place" />
        <div class="legend xs muted">Нажмите на здание, чтобы увидеть характеристики и улучшить его.</div>
      </section>

      <aside class="side">
        <section v-if="selected" class="card panel stack" aria-live="polite">
          <div class="row"><h2 class="grow">{{ selected.name }}</h2><span class="badge info">Уровень {{ selected.level }}</span></div>
          <p class="small muted">{{ game.building(selected.type)?.description }}</p>
          <ul class="specs small"><li v-for="l in specLines(selected.type, selected.spec)" :key="l">{{ l }}</li></ul>
          <div v-if="selected.busy_until" class="stack">
            <div class="row small"><Icon name="clock" :size="16" />Строится уровень {{ selected.target_level }}: осталось {{ duration(left(selected)) }}</div>
            <div class="bar"><span :style="{ width: Math.min(100, 100 - (left(selected) / Math.max(1, new Date(selected.busy_until).getTime() - new Date(selected.started_at || selected.busy_until).getTime())) * 100) + '%' }" /></div>
          </div>
          <template v-else-if="selected.next && player && city">
            <h3>Уровень {{ selected.next.level }}</h3>
            <ul class="specs small"><li v-for="l in specLines(selected.type, selected.next)" :key="l">{{ l }}</li></ul>
            <div class="row small wrap"><span class="cost"><Icon name="coins" :size="14" />{{ num(selected.next.cost) }}</span><span class="muted"><Icon name="clock" :size="14" />{{ duration(selected.next.seconds * (game.config?.settings.time_scale || 1) * 1000) }}</span>
              <span v-if="selected.next.requires_level" class="muted">уровень игрока {{ selected.next.requires_level }}</span></div>
            <button class="btn primary" :disabled="!!upgradeBlock(selected, city, player, builders) || busy" :aria-busy="busy" @click="upgrade(selected)">
              <Icon name="hammer" :size="16" />Улучшить
            </button>
            <p v-if="upgradeBlock(selected, city, player, builders)" class="small reason">{{ upgradeBlock(selected, city, player, builders) }}</p>
          </template>
          <p v-else class="small muted">Максимальный уровень.</p>
        </section>

        <section class="card panel stack">
          <h2>Построить</h2>
          <ul v-if="city && player" class="build-list">
            <li v-for="d in defs" :key="d.type">
              <button class="build" :class="{ on: placing === d.type }" :disabled="!!placeBlock(d, city, player, builders)" :aria-pressed="placing === d.type" @click="startPlacing(d.type)">
                <BuildingIcon :type="d.type" />
                <span class="grow">
                  <strong>{{ d.name }}</strong>
                  <span class="xs muted">{{ city.limits[d.type]?.built }}/{{ city.limits[d.type]?.allowed }} · {{ specLines(d.type, d.levels[0]).slice(0, 2).join(' · ') }}</span>
                  <span v-if="placeBlock(d, city, player, builders)" class="xs reason">{{ placeBlock(d, city, player, builders) }}</span>
                </span>
                <span class="cost"><Icon name="coins" :size="14" />{{ d.levels[0].cost }}</span>
              </button>
            </li>
          </ul>
        </section>

        <section v-if="busyList.length" class="card panel stack">
          <h2>Стройка</h2>
          <div v-for="b in busyList" :key="b.id" class="stack" style="gap: 4px">
            <div class="row small"><strong class="grow">{{ b.name }} → {{ b.target_level }}</strong><span class="num">{{ duration(left(b)) }}</span></div>
            <div class="bar"><span :style="{ width: Math.max(0, Math.min(100, 100 - left(b) / Math.max(1, new Date(b.busy_until!).getTime() - new Date(b.started_at || b.busy_until!).getTime()) * 100)) + '%' }" /></div>
          </div>
        </section>
      </aside>
    </div>
  </div>
</template>


<style scoped>
.stats { display: flex; gap: var(--space-2); flex-wrap: wrap; }
.stat { display: inline-flex; align-items: center; gap: 6px; padding: 6px 12px; background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-pill); font-size: var(--text-sm); font-weight: 650; }
.layout { display: grid; grid-template-columns: minmax(0, 1fr) 360px; gap: var(--space-5); align-items: start; }
.scene-wrap { position: relative; padding: var(--space-3); background: linear-gradient(180deg, var(--art-sky), var(--color-surface) 70%); overflow: hidden; }
.placing-bar { position: absolute; top: var(--space-3); left: var(--space-3); right: var(--space-3); z-index: 2; display: flex; align-items: center; gap: var(--space-2); padding: var(--space-2) var(--space-3); background: var(--color-surface); border-radius: var(--radius-sm); box-shadow: var(--shadow-md); font-size: var(--text-sm); font-weight: 600; }
.placing-bar .btn { margin-left: auto; }
.legend { text-align: center; padding-top: var(--space-2); }
.side { display: flex; flex-direction: column; gap: var(--space-4); }
.specs { margin: 0; padding-left: 18px; color: var(--color-text-muted); }
.cost { display: inline-flex; align-items: center; gap: 4px; font-weight: 750; color: var(--color-gold); }
.reason { color: var(--color-warning); }
.build-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: var(--space-2); }
.build { display: flex; align-items: center; gap: var(--space-3); width: 100%; text-align: left; padding: var(--space-2) var(--space-3); border: 1px solid var(--color-border); border-radius: var(--radius-md); background: var(--color-surface); cursor: pointer; }
.build:hover:not(:disabled) { border-color: var(--color-action-primary); }
.build.on { border-color: var(--color-action-primary); background: var(--color-selected); }
.build:disabled { cursor: not-allowed; opacity: 0.75; }
.build:disabled svg { filter: grayscale(0.8); }
.build .grow { display: flex; flex-direction: column; }
@media (max-width: 1024px) { .layout { grid-template-columns: 1fr; } }
</style>
