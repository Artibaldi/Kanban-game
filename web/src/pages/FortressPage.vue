<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import Icon from '../ui/Icon.vue'
import GameOff from '../components/GameOff.vue'
import GameCrown from '../components/GameCrown.vue'
import FortressScene from '../components/FortressScene.vue'
import FortressScene3D from '../components/FortressScene3D.vue'
import { api, type ApiError } from '../api'
import { crownsText, daysLeftText, enemyName, enemyRows, waveProgress } from '../lib/fortress'
import { date, dateTime, plural } from '../lib/format'
import { moveTask } from '../lib/taskActions'
import { floatText, gameCoins } from '../ui/gameFx'
import { reduced } from '../ui/motion'
import { useGame } from '../stores/game'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import type { Enemy, FortressState, Task, TaskDetail, WaveSummary } from '../types'

const session = useSession()
const game = useGame()
const toasts = useToasts()
const router = useRouter()
const state = ref<FortressState | null>(null)
const waves = ref<WaveSummary[]>([])
const noTeam = ref(false)
const selectedId = ref<string | null>(null)
const flat = ref(false)
const busy = ref(false)
const scene = ref<InstanceType<typeof FortressScene3D>>()
const waveBar = ref<HTMLElement>()
const glint = ref(false)

// Only the newest response counts: a slow periodic refresh must not bring back an enemy that just fell.
let loadSeq = 0
async function load() {
  const seq = ++loadSeq
  try {
    const [f, w] = await Promise.all([api.get<FortressState>('/fortress'), api.get<{ items: WaveSummary[] }>('/waves')])
    if (seq !== loadSeq) return
    state.value = f
    waves.value = w.items
    if (selectedId.value && !f.enemies.some(e => e.task_id === selectedId.value)) selectedId.value = null
  } catch (e) {
    if ((e as ApiError).code === 'not_a_player') noTeam.value = true
    else toasts.error(e)
  }
}
let timer = 0
onMounted(() => {
  if (!session.gameOn) return
  load()
  game.loadConfig().catch(() => {})
  // The horde changes as the team works: refresh quietly.
  timer = window.setInterval(() => { if (document.visibilityState === 'visible' && !battle.value) load() }, 20000)
})
onUnmounted(() => clearInterval(timer))

const enemies = computed(() => state.value?.enemies || [])
const selected = computed(() => enemies.value.find(e => e.task_id === selectedId.value) || null)
const progress = computed(() => (state.value ? Math.round(waveProgress(state.value.wave) * 100) : 0))
const memoryKey = computed(() => `kg-horde-${session.me?.team_id || 'none'}`)
const canApprove = (e: Enemy) => session.isLead && e.state === 'review' && !e.mine
const stageName: Record<Enemy['state'], string> = { ready: 'К выполнению', in_progress: 'В работе', review: 'На проверке' }

// A defeated enemy: its points fly into the wave bar — the team's progress, not anyone's purse.
function onGone(e: Enemy, x: number, y: number) {
  floatText(x, y - 30, `+${e.points}`)
  if (waveBar.value) gameCoins(x, y, waveBar.value, 3 + Math.min(3, Math.floor(e.points / 3)), () => { glint.value = false; nextTick(() => (glint.value = true)) })
}

async function approve(e: Enemy) {
  busy.value = true
  let task: Task
  try { task = (await api.get<TaskDetail>(`/tasks/${e.task_id}`)).task } catch (err) { toasts.error(err); busy.value = false; return }
  const r = await moveTask(task, 'done', { undoable: true, after: load, anonymous: true })
  busy.value = false
  if (!r) return
  selectedId.value = null
  await load()
}

// —— the wave replay ——
const lastWave = computed(() => waves.value[0] || null)
const seenKey = computed(() => `kg-wave-seen-${session.me?.team_id || 'none'}`)
const unseen = computed(() => {
  if (!lastWave.value) return false
  try { return localStorage.getItem(seenKey.value) !== lastWave.value.id } catch { return false }
})
const battle = ref<null | { wave: WaveSummary; points: number; crowns: number; speed: number; done: boolean; shownCrowns: number; dropped: number[] }>(null)
const thresholds = computed(() => game.config?.settings.crown_thresholds || [0.5, 0.8])

function startBattle(w: WaveSummary) {
  selectedId.value = null
  scene.value?.endWave()
  battle.value = { wave: w, points: 0, crowns: 0, speed: 1, done: false, shownCrowns: 0, dropped: [] }
  // The reactive proxy, so identity checks against battle.value hold.
  const b = battle.value
  try { localStorage.setItem(seenKey.value, w.id) } catch { /* fine */ }
  const started = !flat.value && scene.value?.playWave({ planned: w.planned || 0, closed: w.closed || 0, overdue: w.overdue || 0, crowns: w.crowns, thresholds: thresholds.value }, {
    onProgress: p => { if (battle.value) battle.value.points = p },
    onCrown: n => { if (battle.value) battle.value.crowns = n },
    onWallHit: () => {},
    onDone: () => {
      if (battle.value !== b) return
      showResult(b)
    },
  })
  // No 3D scene (flat fallback, or still loading): go straight to the result with the wave's numbers.
  if (!started) { b.points = w.closed || 0; b.crowns = w.crowns; showResult(b) }
}
function showResult(b: NonNullable<typeof battle.value>) {
  b.done = true
  // The result panel drops the crowns one by one; a newer run stops the old timers.
  const drop = (i: number) => { if (battle.value !== b || i >= b.wave.crowns) return; b.shownCrowns = i + 1; setTimeout(() => drop(i + 1), reduced() ? 0 : 420) }
  setTimeout(() => drop(0), reduced() ? 0 : 600)
  nextTick(() => document.querySelector<HTMLElement>('.result .g-btn:last-child')?.focus())
}
function speed() { const b = battle.value; if (!b) return; b.speed = b.speed === 1 ? 2 : 1; scene.value?.setSpeed(b.speed) }
function skip() { scene.value?.finishWave() }
function closeBattle() { scene.value?.endWave(); battle.value = null }
// Escape closes the topmost game panel; focus moves into a panel when it opens and back when it closes.
let opener: HTMLElement | null = null
function select(id: string | null) {
  if (id && !selectedId.value) opener = document.activeElement as HTMLElement
  selectedId.value = id
  if (id) nextTick(() => document.querySelector<HTMLElement>('.enemy-panel .g-x')?.focus())
}
function closePanel() { selectedId.value = null; opener?.focus?.(); opener = null }
function onKey(e: KeyboardEvent) {
  if (e.key !== 'Escape') return
  if (battle.value?.done) closeBattle()
  else if (selectedId.value) closePanel()
}
onMounted(() => addEventListener('keydown', onKey))
onUnmounted(() => removeEventListener('keydown', onKey))
const chest = computed(() => {
  const s = game.config?.settings, c = battle.value?.wave.crowns ?? 0
  if (!s) return null
  return { gold: s.wave_chest_gold[c] ?? 0, energy: s.wave_chest_energy[c] ?? 0, repair: c >= 3 ? 100 : s.repair_by_crowns[c] ?? 0 }
})
const resultTitle = computed(() => {
  const c = battle.value?.wave.crowns ?? 0
  return c === 3 ? 'Чистая победа!' : c > 0 ? 'Крепость выстояла' : 'Волна прорвалась'
})
</script>

<template>
  <GameOff v-if="!session.gameOn" />
  <div v-else class="page fortress-page">
    <p v-if="noTeam" class="card panel muted">Крепость есть у команд. Попросите администратора добавить вас в команду.</p>

    <template v-else-if="state">
      <section class="arena" :class="{ battle: !!battle }" aria-label="Крепость команды">
        <FortressScene3D v-if="!flat" ref="scene" :hp="state.fortress.hp" :fog="state.fog" :enemies="enemies" :selected-id="selectedId" :memory-key="memoryKey"
          @select="select" @gone="onGone" @unavailable="flat = true" />
        <div v-else class="flat"><FortressScene :fortress="state.fortress" :rows="enemyRows(enemies)" :fog="state.fog" :selected-id="selectedId" @select="e => select(e.task_id)" /></div>

        <div class="hud">
          <div class="g-box hp-box" :title="`Прочность крепости: ${state.fortress.hp}%`">
            <svg viewBox="0 0 36 40" width="30" height="34" aria-hidden="true"><path d="M18 2 4 7v11c0 9 6 16 14 20 8-4 14-11 14-20V7z" fill="#3E8EF0" stroke="#1E1E1E" stroke-width="3" stroke-linejoin="round" /><path d="M18 8v24c5-3 8-8 8-14v-6z" fill="#7DB8FF" /></svg>
            <div>
              <div class="g-stroke s2 t"><span class="w">Прочность </span>{{ state.fortress.hp }}%</div>
              <div class="g-bar hp" :class="{ low: state.fortress.siege }"><i :style="{ width: state.fortress.hp + '%' }" /></div>
            </div>
            <span v-if="state.fortress.siege" class="siege g-stroke s2">Осада</span>
          </div>

          <div class="wave">
            <div class="g-box wave-box">
              <div class="grow">
                <div class="g-stroke s2 t">
                  <template v-if="battle">Бой волны: отбито {{ battle.points }} из {{ battle.wave.planned }}</template>
                  <template v-else>Волна: {{ state.wave.closed }} из {{ state.wave.planned }} {{ plural(state.wave.planned, 'очка', 'очков', 'очков') }}</template>
                </div>
                <div ref="waveBar" class="g-bar" :class="{ glint }" role="progressbar" :aria-valuenow="battle ? battle.points : state.wave.closed" aria-valuemin="0" :aria-valuemax="battle ? battle.wave.planned : state.wave.planned" aria-label="Закрыто от плана волны">
                  <i :style="{ width: (battle ? (battle.wave.planned ? (battle.points / battle.wave.planned) * 100 : 0) : progress) + '%' }" />
                </div>
              </div>
              <div class="crowns" :aria-label="`Сейчас: ${crownsText(battle ? battle.crowns : state.wave.crowns_now)}`">
                <GameCrown v-for="i in 3" :key="i" :on="i <= (battle ? battle.crowns : state.wave.crowns_now)" :mid="i === 2" :drop="!!battle" />
              </div>
            </div>
            <span v-if="!battle" class="timer"><Icon name="clock" :size="14" />{{ state.wave.days_left <= 0 ? 'последний день волны' : `до конца волны ${daysLeftText(state.wave.days_left)}` }}</span>
          </div>

          <div class="actions">
            <button v-if="lastWave && !battle" class="g-btn yellow small" :class="{ pulse: unseen }" @click="startBattle(lastWave)"><span>{{ unseen ? 'Итог волны!' : 'Бой волны' }}</span></button>
            <RouterLink v-if="!battle" to="/fortress/duels" class="g-btn blue small"><span>Дружеские бои</span></RouterLink>
          </div>
        </div>

        <p v-if="!battle && !selected" class="hint g-stroke s2">{{ enemies.length ? 'Нажмите на врага — это задача команды' : 'Тишина на границе. Возьмите задачи из бэклога в план.' }}</p>

        <div v-if="battle && !battle.done" class="bctl">
          <button class="g-btn blue small" :class="{ on: battle.speed === 2 }" :aria-pressed="battle.speed === 2" @click="speed"><span>×2</span></button>
          <button class="g-btn blue small" @click="skip"><span>Пропустить</span></button>
        </div>

        <div v-if="battle?.done" class="result">
          <div class="g-panel pop">
            <span class="g-ribbon" :class="{ gold: battle.wave.crowns === 3, red: battle.wave.crowns === 0 }"><b>{{ resultTitle }}</b></span>
            <button class="g-x" aria-label="Закрыть итог" @click="closeBattle"><Icon name="x" :size="18" /></button>
            <div class="big-crowns" :aria-label="crownsText(battle.wave.crowns)">
              <GameCrown v-for="i in 3" :key="i" :on="i <= battle.shownCrowns" :mid="i === 2" drop />
            </div>
            <p class="sum-text">{{ battle.wave.summary }}</p>
            <div v-if="chest" class="sums">
              <div><span class="g-coin" aria-hidden="true" />+{{ chest.gold }}</div>
              <div><Icon name="zap" :size="16" />+{{ chest.energy }}</div>
              <div><Icon name="shield" :size="16" />{{ chest.repair >= 100 ? 'стены как новые' : `+${chest.repair}% стенам` }}</div>
            </div>
            <p class="note">Сундук команды получает каждый участник; бонус обороны добавляет до 20%. {{ battle.wave.crowns === 0 ? 'Без короны стены не чинятся — следующая волна начнётся с трещинами.' : '' }}</p>
            <div class="row-btns">
              <button class="g-btn blue" @click="startBattle(battle.wave)"><span>Ещё раз</span></button>
              <button class="g-btn" @click="closeBattle"><span>Готово</span></button>
            </div>
          </div>
        </div>

        <div v-if="selected && !battle" class="enemy-panel" role="dialog" aria-modal="false" :aria-label="`${enemyName(selected.kind)}: ${selected.title}`">
          <div class="g-panel pop">
            <span class="g-ribbon" :class="{ red: selected.overdue }"><b>{{ enemyName(selected.kind) }}</b></span>
            <button class="g-x" aria-label="Закрыть" @click="closePanel"><Icon name="x" :size="18" /></button>
            <div class="task">
              <div class="pts" :class="{ due: selected.overdue }"><b class="g-stroke">{{ selected.points }}</b><small class="g-stroke s2">{{ plural(selected.points, 'очко', 'очка', 'очков') }}</small></div>
              <div class="grow">
                <h3>{{ selected.title }}</h3>
                <p class="kind">{{ selected.number }} · {{ stageName[selected.state] }}</p>
              </div>
            </div>
            <div class="chips">
              <span class="g-chip">{{ selected.priority }} · ряд {{ selected.row }}</span>
              <span v-if="selected.due_date" class="g-chip" :class="{ red: selected.overdue }">{{ selected.overdue ? 'просрочена с' : 'срок' }} {{ date(selected.due_date) }}</span>
              <span v-if="selected.overdue" class="g-chip red">бьёт по стене</span>
              <span v-else-if="selected.at_gates" class="g-chip">у ворот: срок скоро</span>
            </div>
            <p class="reward">Победа над ним: <b>+{{ selected.points }}</b> к волне команды</p>
            <button v-if="canApprove(selected)" class="g-btn wide" :disabled="busy" :aria-busy="busy" @click="approve(selected)"><span>Подтвердить задачу</span></button>
            <button class="g-btn blue wide" @click="router.push(`/board?task=${selected.task_id}`)"><span>Открыть на доске</span></button>
            <p class="note">{{ selected.state === 'review' ? (selected.mine ? 'Это ваша задача — её подтвердит другой руководитель.' : session.isLead ? 'Задача сдана: подтвердите её, и враг будет повержен.' : 'Задача ждёт проверки руководителя.') : 'Враг падает, когда руководитель подтверждает задачу.' }}</p>
          </div>
        </div>
      </section>

      <!-- The same horde for screen readers and keyboards: the 3D scene is decorative. -->
      <section class="sr-only" aria-label="Враги на поле">
        <ul><li v-for="e in enemies" :key="e.task_id"><button @click="select(e.task_id)">{{ enemyName(e.kind) }}, {{ e.points }} очк.: {{ e.number }} {{ e.title }}{{ e.overdue ? ', просрочена' : '' }}</button></li></ul>
      </section>

      <section class="card panel stack waves-card">
        <h2>Волны</h2>
        <ol v-if="waves.length" class="waves">
          <li v-for="w in waves" :key="w.id">
            <span class="crowns-sm" :aria-label="crownsText(w.crowns)"><GameCrown v-for="i in 3" :key="i" :on="i <= w.crowns" /></span>
            <span class="grow small">{{ w.summary }}<span class="xs muted"> · {{ dateTime(w.ends_at) }}</span></span>
            <button class="btn sm" @click="startBattle(w)">Смотреть бой</button>
          </li>
        </ol>
        <p v-else class="small muted">Первая волна завершится в конце недели.</p>
      </section>
    </template>
  </div>
</template>

<style scoped>
.fortress-page { max-width: none; padding-top: var(--space-4); }
.arena { position: relative; height: calc(100vh - 150px); min-height: 560px; border-radius: var(--radius-xl); overflow: hidden; border: 3px solid var(--g-ink); box-shadow: 0 8px 0 rgba(30, 30, 30, 0.2), var(--shadow-md); isolation: isolate; user-select: none; }
.flat { position: absolute; inset: 0; display: grid; place-items: center; background: linear-gradient(#77c6f2, #bde6fa 60%, #d9eef5); padding: 90px 12px 12px; }
.hud { position: absolute; top: 12px; left: 12px; right: 12px; z-index: 5; display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; pointer-events: none; }
.hud > * { pointer-events: auto; }
.hp-box { display: flex; align-items: center; gap: 10px; padding: 8px 12px; }
.t { font-size: 16px; margin-bottom: 5px; white-space: nowrap; }
.hp-box .g-bar { width: 150px; height: 16px; }
.siege { padding: 2px 8px; border: 2px solid var(--g-ink); border-radius: 8px; background: var(--g-red); font-size: 13px; }
.wave { display: flex; flex-direction: column; align-items: center; gap: 8px; }
.wave-box { display: flex; align-items: center; gap: 12px; padding: 8px 12px 10px 16px; }
.wave-box .g-bar { width: 270px; }
.crowns { display: flex; gap: 4px; align-items: flex-end; --crown: 40px; }
.timer { display: inline-flex; align-items: center; gap: 6px; padding: 4px 12px; border: 2px solid var(--g-ink); border-radius: 999px; background: linear-gradient(#fff7de, #f5e7c6); font: 800 14px var(--font-game); color: #5a3a1e; box-shadow: 0 3px 0 rgba(0, 0, 0, 0.3); }
.actions { display: flex; flex-direction: column; align-items: flex-end; gap: 4px; }
.pulse { animation: pulse 1.2s ease-in-out infinite; }
@keyframes pulse { 50% { transform: scale(1.06); } }
.hint { position: absolute; left: 50%; bottom: 18px; transform: translateX(-50%); z-index: 5; padding: 8px 18px; border: 2px solid var(--g-ink); border-radius: 999px; background: rgba(24, 30, 72, 0.85); font-size: 15px; white-space: nowrap; pointer-events: none; }
.bctl { position: absolute; left: 50%; bottom: 18px; transform: translateX(-50%); z-index: 6; display: flex; gap: 12px; }
.enemy-panel, .result { position: absolute; z-index: 8; left: 50%; bottom: 24px; transform: translateX(-50%); width: min(440px, calc(100% - 24px)); }
.result { bottom: 32px; text-align: center; }
@media (min-width: 1000px) { .enemy-panel { left: auto; right: 24px; transform: none; bottom: 28px; } }
.result::before { content: ""; position: fixed; inset: 0; z-index: -1; background: rgba(10, 12, 34, 0.35); pointer-events: none; }
.task { display: flex; gap: 14px; align-items: center; }
.pts { flex: none; width: 78px; height: 78px; border: 3px solid var(--g-ink); border-radius: 18px; background: linear-gradient(#ff8a8a, #f04848 55%, #c8302f); box-shadow: inset 0 3px 0 rgba(255, 255, 255, 0.4), 0 4px 0 var(--g-ink); display: flex; flex-direction: column; align-items: center; justify-content: center; line-height: 1; }
.pts.due { background: linear-gradient(#ffe07a, #ffb21e 55%, #e58a00); }
.pts b { font-size: 34px; }
.pts small { font-size: 12px; margin-top: 2px; }
.task h3 { margin: 0 0 4px; font: 800 19px/1.2 var(--font-game); color: #3a2410; }
.kind { font: 700 14px var(--font-sans); color: #7a5530; }
.chips { display: flex; flex-wrap: wrap; gap: 8px; margin: 14px 0 6px; }
.reward { margin: 12px 0 14px; padding: 10px; border-radius: 14px; background: rgba(107, 70, 38, 0.1); font: 800 16px var(--font-game); color: #4a3018; text-align: center; }
.wide { width: 100%; }
.note { margin-top: 6px; text-align: center; font: 600 13px var(--font-sans); color: #8a6844; }
.big-crowns { display: flex; justify-content: center; align-items: flex-end; gap: 6px; margin: 4px 0 6px; --crown: 78px; }
.sum-text { margin: 4px 0 10px; font: 700 15px var(--font-sans); color: #5a3a1e; }
.sums { display: flex; justify-content: center; gap: 10px; margin-bottom: 8px; flex-wrap: wrap; }
.sums div { display: flex; align-items: center; gap: 6px; padding: 8px 12px; border-radius: 12px; background: rgba(107, 70, 38, 0.12); font: 800 16px var(--font-game); color: #3a2410; }
.row-btns { display: flex; gap: 12px; justify-content: center; margin-top: 10px; }
.waves-card { margin-top: var(--space-5); }
.waves { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: var(--space-3); }
.waves li { display: flex; gap: var(--space-3); align-items: center; }
.crowns-sm { display: inline-flex; gap: 2px; --crown: 22px; flex: none; }
@media (max-width: 900px) {
  .hud { flex-wrap: wrap; gap: 8px; top: 8px; left: 8px; right: 8px; }
  .wave { order: 3; width: 100%; gap: 6px; }
  .wave-box { width: 100%; padding: 6px 8px 8px 10px; }
  .wave-box .g-bar { width: 100%; height: 16px; }
  .actions { gap: 2px; }
  .actions .g-btn { font-size: 13px; padding: 5px 10px; margin-bottom: 6px; }
  .arena :deep(.g3-fog) { display: none; }
}
@media (max-width: 767px) {
  .arena { height: calc(100vh - 210px); min-height: 520px; border-radius: 22px; margin: 0 -4px; }
  .hp-box { padding: 5px 8px; gap: 6px; }
  .hp-box svg { width: 24px; height: 28px; }
  .hp-box .g-bar { width: 64px; height: 12px; }
  .t .w { display: none; }
  .timer { font-size: 12px; padding: 2px 10px; }
  .t { font-size: 14px; }
  .crowns { --crown: 30px; }
  .big-crowns { --crown: 56px; }
  .hint { font-size: 12px; bottom: 12px; white-space: normal; text-align: center; width: max-content; max-width: calc(100% - 24px); }
}
</style>
