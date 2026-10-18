<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import Icon from '../ui/Icon.vue'
import CityScene from '../components/CityScene.vue'
import CityScene3D from '../components/CityScene3D.vue'
import GameOff from '../components/GameOff.vue'
import Portrait3D from '../components/Portrait3D.vue'
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
/** The building being moved («Переставить»), if any. */
const moving = ref<CityBuilding | null>(null)
const busy = ref(false)
const now = ref(Date.now())
const flat = ref(false)
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
const defs = computed(() => game.config?.buildings.filter(d => d.type !== 'townhall' && !d.decor) || [])
const decorDefs = computed(() => game.config?.buildings.filter(d => d.decor) || [])
const isDecor = (t: BuildingType) => !!game.building(t)?.decor
const builders = computed(() => game.config?.builders || 2)
const busyList = computed(() => city.value?.buildings.filter(b => b.busy_until) || [])

function select(b: CityBuilding) {
  placing.value = null
  moving.value = null
  selectedId.value = b.id
  // One column (tablet, phone): the panel is below the field — bring it into view.
  if (innerWidth <= 1100) nextTick(() => document.querySelector('.sel')?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
}
function onKey(e: KeyboardEvent) { if (e.key === 'Escape') { if (moving.value) moving.value = null; else if (placing.value) placing.value = null; else selectedId.value = null } }
onMounted(() => addEventListener('keydown', onKey))
onUnmounted(() => removeEventListener('keydown', onKey))
function startPlacing(t: BuildingType) { selectedId.value = null; moving.value = null; placing.value = placing.value === t ? null : t }
function startMoving(b: CityBuilding) { placing.value = null; moving.value = b }
async function onCell(x: number, y: number) { if (moving.value) await move(x, y); else await place(x, y) }
async function move(x: number, y: number) {
  const b = moving.value
  if (!b || busy.value) return
  busy.value = true
  try {
    const c = await api.post<City>(`/city/buildings/${b.id}/move`, { x, y })
    city.value = c
    toasts.push({ kind: 'success', title: `${b.name}: на новом месте` }, 2500)
    moving.value = null
    selectedId.value = b.id
  } catch (e) { toasts.error(e) } finally { busy.value = false }
}
async function removeDecor(b: CityBuilding) {
  busy.value = true
  try {
    const c = await api.del<City>(`/city/buildings/${b.id}`)
    city.value = c
    game.setPlayer(c.player)
    selectedId.value = null
    toasts.push({ kind: 'info', title: `${b.name}: убрано` }, 2500)
  } catch (e) { toasts.error(e) } finally { busy.value = false }
}

async function place(x: number, y: number) {
  if (!placing.value || busy.value) return
  busy.value = true
  try {
    const c = await api.post<City>('/city/buildings', { type: placing.value, x, y })
    city.value = c
    game.setPlayer(c.player)
    toasts.push({ kind: 'success', title: `${game.building(placing.value)?.name}: стройка началась` }, 2500)
    placing.value = null
    // Back to the list the build started from.
    nextTick(() => document.querySelector<HTMLElement>('.build:not(:disabled)')?.focus())
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
    <p v-if="player?.debt" class="notice danger" style="margin-bottom: 16px">После отмены награды баланс отрицательный. Строительство откроется, когда следующие награды погасят долг.</p>

    <div class="layout">
      <section class="arena" aria-label="Поле города">
        <CityScene3D v-if="city && !flat" :buildings="city.buildings" :grid="city.grid || game.config?.grid || 16" :land="city.land" :selected-id="selectedId" :placing="moving ? 'move' : placing" :now="now"
          @select="select" @deselect="selectedId = null" @cell="onCell" @unavailable="flat = true" />
        <div v-else-if="city" class="flat">
          <!-- While moving, the flat view only needs some type to light up the free cells. -->
          <CityScene :buildings="city.buildings" :grid="city.grid || game.config?.grid" :selected-id="selectedId" :placing="moving ? 'wall' : placing" :now="now"
            :label="`Город: ${city.buildings.length} зданий`" @select="select" @cell="onCell" />
        </div>
        <div class="hud">
          <div class="g-box title-box">
            <div class="g-stroke s2 name">Город {{ session.me?.display_name.split(' ')[0] }}</div>
            <div class="sub">Ратуша {{ city?.townhall || '—' }} уровня</div>
          </div>
          <div v-if="player" class="chips">
            <span class="g-box chip"><Icon name="hammer" :size="16" />Строители {{ player.builders_busy }}/{{ player.builders }}</span>
            <span class="g-box chip"><Icon name="shield" :size="16" />Армия {{ player.army_housing }}/{{ player.army_capacity }}</span>
          </div>
        </div>
        <div v-if="moving" class="placing g-box">
          <span class="g-stroke s2">Куда переставить «{{ moving.name }}»?</span>
          <span class="hint">Свободная клетка вашей земли{{ flat ? '' : ' — мышью или стрелками и Enter' }}. Бесплатно.</span>
          <button class="g-btn red small" @click="moving = null"><span>Отмена</span></button>
        </div>
        <div v-else-if="placing" class="placing g-box">
          <span class="g-stroke s2">Где построить «{{ game.building(placing)?.name }}»?</span>
          <span class="hint">Нажмите на свободную клетку{{ flat ? '' : ' или выберите её стрелками и Enter' }}</span>
          <button class="g-btn red small" @click="placing = null"><span>Отмена</span></button>
        </div>
        <p v-else class="tip g-stroke s2">Золото приходит за подтверждённые задачи. Нажмите на здание, чтобы улучшить его.</p>
      </section>

      <aside class="side">
        <section v-if="selected" class="g-panel pop sel" aria-live="polite">
          <span class="g-ribbon"><b>{{ selected.name }}</b></span>
          <button class="g-x" aria-label="Закрыть" @click="selectedId = null"><Icon name="x" :size="18" /></button>
          <div v-if="!isDecor(selected.type)" class="lvl"><span class="g-stroke">Ур. {{ selected.level }}</span><span v-if="selected.next" class="arrow">→ {{ selected.next.level }}</span></div>
          <p class="desc">{{ game.building(selected.type)?.description }}</p>
          <ul v-if="!isDecor(selected.type)" class="specs"><li v-for="l in specLines(selected.type, selected.spec)" :key="l">{{ l }}</li></ul>
          <div class="tools">
            <button class="g-btn blue small" :disabled="busy" @click="startMoving(selected)"><span>Переставить</span></button>
            <button v-if="isDecor(selected.type)" class="g-btn red small" :disabled="busy" @click="removeDecor(selected)"><span>Убрать</span></button>
          </div>
          <p v-if="isDecor(selected.type)" class="desc">Украшение: на оборону не влияет. Если убрать, золото не вернётся.</p>
          <div v-if="isDecor(selected.type)" />
          <div v-else-if="selected.busy_until" class="building">
            <div class="row"><Icon name="clock" :size="16" />Строится уровень {{ selected.target_level }}: осталось {{ duration(left(selected)) }}</div>
            <div class="g-bar"><i :style="{ width: Math.min(100, 100 - (left(selected) / Math.max(1, new Date(selected.busy_until).getTime() - new Date(selected.started_at || selected.busy_until).getTime())) * 100) + '%' }" /></div>
          </div>
          <template v-else-if="selected.next && player && city">
            <div class="next">
              <h3>Уровень {{ selected.next.level }}</h3>
              <ul class="specs"><li v-for="l in specLines(selected.type, selected.next)" :key="l">{{ l }}</li></ul>
              <div class="cost-row"><span class="price"><span class="g-coin" aria-hidden="true" />{{ num(selected.next.cost) }}</span><span><Icon name="clock" :size="14" />{{ duration(selected.next.seconds * (game.config?.settings.time_scale || 1) * 1000) }}</span>
                <span v-if="selected.next.requires_level">уровень игрока {{ selected.next.requires_level }}</span></div>
            </div>
            <button class="g-btn wide" :disabled="!!upgradeBlock(selected, city, player, builders) || busy" :aria-busy="busy" @click="upgrade(selected)"><span>Улучшить</span></button>
            <p v-if="upgradeBlock(selected, city, player, builders)" class="reason">{{ upgradeBlock(selected, city, player, builders) }}</p>
          </template>
          <p v-else-if="!isDecor(selected.type)" class="desc">Максимальный уровень.</p>
        </section>

        <section class="g-panel build-panel">
          <span class="g-ribbon gold"><b>Построить</b></span>
          <ul v-if="city && player" class="build-list">
            <li v-for="d in defs" :key="d.type">
              <button class="build" :class="{ on: placing === d.type }" :disabled="!!placeBlock(d, city, player, builders)" :aria-pressed="placing === d.type" @click="startPlacing(d.type)">
                <Portrait3D :building="d.type" :size="52" :faded="!!placeBlock(d, city, player, builders)" />
                <span class="grow">
                  <strong>{{ d.name }}</strong>
                  <span class="meta">{{ city.limits[d.type]?.built }}/{{ city.limits[d.type]?.allowed }} · {{ specLines(d.type, d.levels[0]).slice(0, 2).join(' · ') }}</span>
                  <span v-if="placeBlock(d, city, player, builders)" class="reason">{{ placeBlock(d, city, player, builders) }}</span>
                </span>
                <span class="price"><span class="g-coin" aria-hidden="true" />{{ d.levels[0].cost }}</span>
              </button>
            </li>
          </ul>
        </section>

        <section class="g-panel build-panel">
          <span class="g-ribbon"><b>Облик</b></span>
          <p class="decor-note">Мгновенно, без строителя. На силу города не влияет — только на его вид.</p>
          <ul v-if="city && player" class="decor-list">
            <li v-for="d in decorDefs" :key="d.type">
              <button class="decor" :class="{ on: placing === d.type }" :disabled="!!placeBlock(d, city, player, builders)" :aria-pressed="placing === d.type"
                :title="placeBlock(d, city, player, builders) || d.description" @click="startPlacing(d.type)">
                <Portrait3D :building="d.type" :size="44" :faded="!!placeBlock(d, city, player, builders)" />
                <strong>{{ d.name }}</strong>
                <span class="price"><span class="g-coin" aria-hidden="true" />{{ d.levels[0].cost }}</span>
                <span class="meta">{{ city.limits[d.type]?.built }}/{{ city.limits[d.type]?.allowed }}</span>
              </button>
            </li>
          </ul>
        </section>

        <section v-if="busyList.length" class="g-panel build-panel">
          <span class="g-ribbon"><b>Стройка</b></span>
          <div v-for="b in busyList" :key="b.id" class="busy">
            <div class="row"><strong class="grow">{{ b.name }} → {{ b.target_level }}</strong><span class="num">{{ duration(left(b)) }}</span></div>
            <div class="g-bar"><i :style="{ width: Math.max(0, Math.min(100, 100 - left(b) / Math.max(1, new Date(b.busy_until!).getTime() - new Date(b.started_at || b.busy_until!).getTime()) * 100)) + '%' }" /></div>
          </div>
        </section>

        <!-- Buildings for the keyboard and screen readers: the 3D field is pointer-first. -->
        <section v-if="city && !flat" class="sr-only" aria-label="Здания города">
          <ul><li v-for="b in city.buildings" :key="b.id"><button @click="select(b)">{{ b.name }}, уровень {{ b.level }}{{ b.busy_until ? ', строится' : '' }}, клетка {{ b.x + 1 }}:{{ b.y + 1 }}</button></li></ul>
        </section>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.city-page { max-width: none; padding-top: var(--space-4); }
.layout { display: grid; grid-template-columns: minmax(0, 1fr) 380px; gap: var(--space-5); align-items: start; }
.arena { position: relative; height: calc(100vh - 150px); min-height: 560px; border-radius: var(--radius-xl); overflow: hidden; border: 3px solid var(--g-ink); box-shadow: 0 8px 0 rgba(30, 30, 30, 0.2), var(--shadow-md); isolation: isolate; user-select: none; }
.flat { position: absolute; inset: 0; display: grid; place-items: center; padding: 90px 12px 60px; background: linear-gradient(#69bdf0, #b9e4fa 70%, #e3f4fb); }
.hud { position: absolute; top: 12px; left: 12px; right: 12px; z-index: 5; display: flex; justify-content: space-between; align-items: flex-start; gap: 10px; pointer-events: none; }
.title-box { padding: 8px 14px; }
.name { font-size: 20px; }
.sub { font: 700 13px var(--font-sans); color: #dde6ff; }
.chips { display: flex; gap: 8px; flex-wrap: wrap; justify-content: flex-end; }
.chip { display: inline-flex; align-items: center; gap: 6px; padding: 6px 12px; border-radius: 999px; font: 800 14px var(--font-game); }
.placing { position: absolute; left: 50%; bottom: 16px; transform: translateX(-50%); z-index: 6; display: flex; align-items: center; gap: 12px; padding: 10px 12px 4px 16px; flex-wrap: wrap; justify-content: center; max-width: calc(100% - 24px); }
.placing .hint { font: 600 13px var(--font-sans); color: #dde6ff; }
.tip { position: absolute; left: 50%; bottom: 16px; transform: translateX(-50%); z-index: 5; padding: 8px 18px; border: 2px solid var(--g-ink); border-radius: 999px; background: rgba(24, 30, 72, 0.85); font-size: 14px; white-space: nowrap; pointer-events: none; }
.side { display: flex; flex-direction: column; gap: 34px; padding-top: 26px; }
.sel .lvl { display: flex; align-items: baseline; justify-content: center; gap: 8px; margin-top: 4px; }
.sel .lvl .g-stroke { font-size: 30px; color: var(--g-gold); }
.arrow { font: 800 18px var(--font-game); color: #7a5530; }
.desc { font: 600 14px var(--font-sans); color: #5a3a1e; margin: 8px 0; text-align: center; }
.specs { margin: 0 0 8px; padding-left: 18px; font: 600 13px var(--font-sans); color: #7a5530; }
.next { padding: 10px 12px; border-radius: 14px; background: rgba(107, 70, 38, 0.1); margin-bottom: 12px; }
.next h3 { font: 800 16px var(--font-game); color: #3a2410; margin-bottom: 4px; }
.cost-row { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; font: 700 13px var(--font-sans); color: #5a3a1e; }
.cost-row span { display: inline-flex; align-items: center; gap: 4px; }
.price { display: inline-flex; align-items: center; gap: 6px; font: 900 16px var(--font-game); color: #3a2410; }
.price .g-coin { width: 22px; height: 22px; }
.price .g-coin::after { width: 6px; height: 6px; }
.wide { width: 100%; }
.reason { font: 700 12px var(--font-sans); color: #9b1f1f; margin-top: 4px; }
.building { display: flex; flex-direction: column; gap: 8px; font: 700 14px var(--font-sans); color: #5a3a1e; }
.build-panel { padding-top: 40px; }
.build-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.build { display: flex; align-items: center; gap: 12px; width: 100%; text-align: left; padding: 8px 10px; border: 2px solid var(--g-panel-b); border-radius: 14px; background: #fff8e6; cursor: pointer; color: #3a2410; box-shadow: 0 3px 0 rgba(107, 70, 38, 0.35); }
.build:hover:not(:disabled) { background: #fffdf3; transform: translateY(-1px); }
.build.on { border-color: var(--g-ink); background: linear-gradient(#ffec8a, #ffd323); }
.build:disabled { cursor: not-allowed; opacity: 0.72; }
.build:disabled svg { filter: grayscale(0.8); }
.build .grow { display: flex; flex-direction: column; }
.build strong { font: 800 15px var(--font-game); }
.meta { font: 600 12px var(--font-sans); color: #7a5530; }
.tools { display: flex; gap: 8px; justify-content: center; flex-wrap: wrap; margin: 4px 0 6px; }
.decor-note { font: 600 13px var(--font-sans); color: #7a5530; text-align: center; margin-bottom: 8px; }
.decor-list { list-style: none; margin: 0; padding: 0; display: grid; grid-template-columns: repeat(auto-fill, minmax(96px, 1fr)); gap: 8px; }
.decor { display: flex; flex-direction: column; align-items: center; gap: 2px; width: 100%; padding: 6px 4px; border: 2px solid var(--g-panel-b); border-radius: 14px; background: #fff8e6; color: #3a2410; cursor: pointer; box-shadow: 0 3px 0 rgba(107, 70, 38, 0.35); }
.decor strong { font: 800 13px var(--font-game); }
.decor.on { border-color: var(--g-ink); background: linear-gradient(#ffec8a, #ffd323); }
.decor:disabled { opacity: 0.6; cursor: not-allowed; }
.busy { display: flex; flex-direction: column; gap: 6px; font: 700 14px var(--font-sans); color: #3a2410; margin-bottom: 10px; }
@media (max-width: 1100px) { .layout { grid-template-columns: 1fr; } .tip { display: none; } }
@media (max-width: 767px) {
  .arena { height: calc(100vh - 230px); min-height: 480px; border-radius: 22px; }
  .tip { display: none; }
  .chips .chip { font-size: 12px; padding: 4px 9px; }
  .name { font-size: 15px; white-space: nowrap; }
  .title-box { padding: 6px 10px; }
  .sub { font-size: 12px; }
}
</style>
