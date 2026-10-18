<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from '../ui/Icon.vue'
import Toasts from './Toasts.vue'
import GameCounters from '../components/GameCounters.vue'
import GameRulesDialog from '../components/GameRulesDialog.vue'
import { initials } from '../lib/work'
import { useSession } from '../stores/session'
import { useGame } from '../stores/game'
import { useWork } from '../stores/work'
import { useToasts } from '../stores/toast'

const session = useSession()
const game = useGame()
const work = useWork()
const toasts = useToasts()
const route = useRoute()
const router = useRouter()

interface NavItem { to: string; label: string; icon: string; game?: boolean; lead?: boolean; admin?: boolean }
const all: NavItem[] = [
  { to: '/', label: 'Главная', icon: 'home' },
  { to: '/board', label: 'Доска', icon: 'board' },
  { to: '/review', label: 'Проверка', icon: 'review', lead: true },
  { to: '/progress', label: 'Прогресс', icon: 'chart' },
  { to: '/fortress', label: 'Крепость', icon: 'flag', game: true },
  { to: '/city', label: 'Город', icon: 'castle', game: true },
  { to: '/army', label: 'Казарма', icon: 'shield', game: true },
  { to: '/admin', label: 'Админ', icon: 'users', admin: true },
]
const nav = computed(() => all.filter(i => (!i.game || session.gameOn) && (!i.lead || session.isLead) && (!i.admin || session.isAdmin)))
const mobileNav = computed(() => {
  const want = ['/', '/board', session.isLead ? '/review' : '/progress', session.gameOn ? '/fortress' : '/progress', session.gameOn ? '/city' : '']
  return [...new Set(want)].map(to => nav.value.find(i => i.to === to)).filter((i): i is NavItem => !!i)
})
const active = (to: string) => (to === '/' ? route.path === '/' : route.path.startsWith(to))

// The dark pill slides under the current section.
const navEl = ref<HTMLElement>()
const ind = ref({ x: 0, w: 0, on: false })
function slide() {
  const el = navEl.value?.querySelector<HTMLElement>('[aria-current="page"]')
  ind.value = el ? { x: el.offsetLeft, w: el.offsetWidth, on: true } : { ...ind.value, on: false }
}
watch([() => route.path, nav, () => work.reviewCount], () => nextTick(slide))

let timer = 0
onMounted(() => {
  timer = window.setInterval(() => {
    if (document.visibilityState !== 'visible') return
    if (session.gameOn) game.refresh()
    if (session.isLead) work.refreshReview()
  }, 15000)
  addEventListener('resize', slide)
  document.fonts?.ready.then(slide)
})
onUnmounted(() => { clearInterval(timer); removeEventListener('resize', slide) })
watch(() => session.me?.id, id => {
  if (id && session.plays) { game.refresh(); game.loadConfig() }
  if (id && session.isLead) work.refreshReview()
}, { immediate: true })

async function setMode(on: boolean) {
  if (!session.me || session.me.game_mode === on) return
  try {
    await session.setGameMode(on)
    toasts.push({ kind: 'info', title: on ? 'Игровой режим включён' : 'Рабочий режим: только задачи, без игры и анимаций' })
    if (!session.gameOn && route.meta.game) router.push('/board')
  } catch (e) { toasts.error(e) }
}
async function logout() { await session.logout(); router.push('/login') }
</script>

<template>
  <a class="skip" href="#main">Перейти к содержимому</a>
  <template v-if="session.me && route.path !== '/login'">
    <header class="top">
      <RouterLink to="/" class="brand" aria-label="Бастион задач — главная">
        <span class="logo" aria-hidden="true"><svg viewBox="0 0 32 32" width="22" height="22"><path d="M4 28V12l4-3V4h4v3l4-3 4 3V4h4v5l4 3v16h-9v-6a3 3 0 0 0-6 0v6z" fill="currentColor"/></svg></span>
        <span class="brand-text">Бастион задач<small>{{ session.me.team_name ? `Команда «${session.me.team_name}»` : 'Все команды' }}</small></span>
      </RouterLink>

      <nav ref="navEl" class="nav glass" aria-label="Разделы">
        <span class="ind" :class="{ on: ind.on }" :style="{ transform: `translateX(${ind.x}px)`, width: ind.w + 'px' }" aria-hidden="true" />
        <RouterLink v-for="i in nav" :key="i.to" :to="i.to" class="nav-link" :class="{ game: i.game }" :aria-current="active(i.to) ? 'page' : undefined">
          {{ i.label }}<span v-if="i.to === '/review' && work.reviewCount" class="nb" :aria-label="`, ждут ${work.reviewCount}`">{{ work.reviewCount }}</span>
        </RouterLink>
      </nav>

      <div class="right">
        <GameCounters v-if="session.gameOn && game.player" :player="game.player" class="game-only counters-top" />
        <div v-if="session.plays" class="mode glass" role="radiogroup" aria-label="Режим">
          <span class="mode-ind" :class="{ game: session.me.game_mode }" aria-hidden="true" />
          <button role="radio" :aria-checked="!session.me.game_mode" title="Только канбан, без игры и анимаций" @click="setMode(false)">Работа</button>
          <button role="radio" :aria-checked="session.me.game_mode" title="Награды, крепость и город" @click="setMode(true)">Игра</button>
        </div>
        <RouterLink to="/settings" class="me" :title="`${session.me.display_name} — настройки`" :aria-label="`${session.me.display_name}, настройки`">{{ initials(session.me.display_name) }}</RouterLink>
        <button class="btn ghost icon logout" aria-label="Выйти" title="Выйти" @click="logout"><Icon name="logout" /></button>
      </div>
    </header>
    <div v-if="session.gameOn && game.player" class="counters-mobile game-only"><GameCounters :player="game.player" /></div>
    <main id="main"><RouterView /></main>
    <nav class="tabbar" aria-label="Разделы">
      <RouterLink v-for="i in mobileNav" :key="i.to" :to="i.to" :aria-current="active(i.to) ? 'page' : undefined">
        <Icon :name="i.icon" :size="20" /><span>{{ i.label }}</span>
        <span v-if="i.to === '/review' && work.reviewCount" class="nb">{{ work.reviewCount }}</span>
      </RouterLink>
    </nav>
  </template>
  <main v-else id="main"><RouterView /></main>
  <GameRulesDialog v-if="session.gameOn && session.me && !session.me.game_rules_ack" />
  <Toasts />
</template>

<style scoped>
.skip { position: absolute; left: -999px; top: 8px; z-index: 100; background: var(--color-surface); padding: 8px 12px; border-radius: var(--radius-sm); }
.skip:focus { left: 8px; }
.top { max-width: 1560px; margin: 0 auto; padding: 22px var(--space-8) 0; display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: var(--space-4); }
.brand { display: flex; align-items: center; gap: 10px; text-decoration: none; color: var(--color-text); min-width: 0; }
.logo { display: grid; place-items: center; width: 40px; height: 40px; border-radius: 12px; background: #1e1e1e; color: var(--gold-fill); flex: none; }
.brand-text { font-weight: 700; font-size: 16px; letter-spacing: -0.01em; line-height: 1.25; min-width: 0; }
.brand-text small { display: block; font-size: 12px; font-weight: 400; color: var(--color-text-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

.nav { justify-self: center; position: relative; display: flex; gap: 2px; padding: 4px; border-radius: var(--radius-pill); max-width: 100%; overflow-x: auto; scrollbar-width: none; }
.nav::-webkit-scrollbar { display: none; }
.ind { position: absolute; top: 4px; left: 0; height: 38px; border-radius: var(--radius-pill); background: var(--color-action-primary); opacity: 0; transition: transform 0.28s var(--ease-in-out), width 0.28s var(--ease-in-out), opacity 0.2s; }
.ind.on { opacity: 1; }
.nav-link { position: relative; z-index: 1; display: inline-flex; align-items: center; gap: 6px; height: 38px; padding: 0 16px; border-radius: var(--radius-pill); color: var(--color-text); text-decoration: none; font-weight: 500; font-size: 14px; white-space: nowrap; transition: color 0.3s; }
.nav-link:hover:not([aria-current="page"]) { background: var(--color-action-secondary-hover); }
.nav-link[aria-current="page"] { color: var(--color-action-primary-text); }
.nav-link.game::before { content: ""; width: 6px; height: 6px; border-radius: 50%; background: var(--gold-fill); box-shadow: 0 0 0 1px var(--gold-edge); }
.nb { display: inline-grid; place-items: center; min-width: 20px; height: 20px; padding: 0 6px; border-radius: var(--radius-pill); background: var(--gold-fill); color: #1e1e1e; font-size: 11px; font-weight: 700; }

.right { justify-self: end; display: flex; align-items: center; gap: 10px; }
.mode { position: relative; display: flex; padding: 4px; border-radius: var(--radius-pill); }
.mode button { position: relative; z-index: 1; height: 34px; padding: 0 14px; border: 0; background: none; border-radius: var(--radius-pill); font-weight: 500; font-size: 14px; cursor: pointer; transition: color 0.3s; }
.mode button[aria-checked="true"] { color: var(--color-action-primary-text); }
.mode-ind { position: absolute; top: 4px; left: 4px; width: calc(50% - 4px); height: 34px; border-radius: var(--radius-pill); background: var(--color-action-primary); transition: transform 0.28s var(--ease-in-out), background 0.2s; }
.mode-ind.game { transform: translateX(100%); background: linear-gradient(180deg, var(--gold-top), var(--gold-bottom)); box-shadow: inset 0 -2px 0 rgba(122, 74, 0, 0.35); }
.mode-ind.game ~ button[aria-checked="true"] { color: var(--gold-ink); font-weight: 700; }
.me { display: grid; place-items: center; width: 40px; height: 40px; border-radius: 50%; background: #f7e39a; color: #1e1e1e; font-weight: 700; font-size: 13px; text-decoration: none; box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.8); flex: none; }
.counters-mobile { display: none; }
.tabbar { display: none; }

@media (max-width: 1360px) {
  .top { grid-template-columns: minmax(0, 1fr) auto; }
  .nav { grid-column: 1 / -1; grid-row: 2; justify-self: start; }
}
@media (max-width: 960px) { .counters-top { display: none !important; } .counters-mobile { display: flex; justify-content: center; padding: 14px 16px 0; } }
@media (max-width: 767px) {
  .top { padding: 14px var(--space-4) 0; gap: var(--space-3); }
  .nav, .logout { display: none; }
  .mode button { height: 32px; padding: 0 12px; }
  .mode-ind { height: 32px; }
  .tabbar { display: flex; position: fixed; left: 12px; right: 12px; bottom: max(12px, env(safe-area-inset-bottom)); z-index: 44; height: 66px; padding: 6px; border-radius: 26px; background: #1e1e1e; box-shadow: 0 18px 40px rgba(30, 20, 10, 0.3); }
  .tabbar a { position: relative; flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 3px; border-radius: 20px; color: #b8b8b8; font-size: 11px; font-weight: 500; text-decoration: none; min-width: 0; transition: color 0.25s, background 0.25s; }
  .tabbar a[aria-current="page"] { color: #fff; background: rgba(255, 255, 255, 0.12); }
  .tabbar .nb { position: absolute; top: 4px; left: calc(50% + 6px); min-width: 18px; height: 18px; font-size: 10px; }
}
@media (max-width: 420px) { .brand-text small { display: none; } .brand-text { font-size: 15px; } }
</style>
