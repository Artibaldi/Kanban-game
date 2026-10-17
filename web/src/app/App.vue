<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from '../ui/Icon.vue'
import Toasts from './Toasts.vue'
import ResourceBar from '../components/ResourceBar.vue'
import { useSession } from '../stores/session'
import { useGame } from '../stores/game'
import { useToasts } from '../stores/toast'

const session = useSession()
const game = useGame()
const toasts = useToasts()
const route = useRoute()
const router = useRouter()

interface NavItem { to: string; label: string; icon: string; game?: boolean; lead?: boolean; admin?: boolean }
const all: NavItem[] = [
  { to: '/', label: 'Главная', icon: 'home' },
  { to: '/board', label: 'Доска', icon: 'board' },
  { to: '/review', label: 'Проверка', icon: 'review', lead: true },
  { to: '/fortress', label: 'Крепость', icon: 'flag', game: true },
  { to: '/city', label: 'Город', icon: 'castle', game: true },
  { to: '/army', label: 'Казарма', icon: 'shield', game: true },
  { to: '/ratings', label: 'Рейтинги', icon: 'trophy' },
  { to: '/admin', label: 'Админ', icon: 'users', admin: true },
]
const nav = computed(() => all.filter(i => (!i.game || session.gameOn) && (!i.lead || session.isLead) && (!i.admin || session.isAdmin)))
const mobileNav = computed(() => nav.value.filter(i => i.to !== '/army' && i.to !== '/admin' && i.to !== '/review').slice(0, 5))
const active = (to: string) => (to === '/' ? route.path === '/' : route.path.startsWith(to))

let timer = 0
onMounted(() => {
  timer = window.setInterval(() => { if (session.gameOn && document.visibilityState === 'visible') game.refresh() }, 15000)
})
onUnmounted(() => clearInterval(timer))
watch(() => session.me?.id, id => { if (id && session.plays) { game.refresh(); game.loadConfig() } }, { immediate: true })

async function toggleMode() {
  try {
    await session.setGameMode(!session.me?.game_mode)
    toasts.push({ kind: 'info', title: session.me?.game_mode ? 'Игровой режим включён' : 'Рабочий режим: только задачи, без игры и анимаций' })
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
        <svg viewBox="0 0 32 32" width="28" height="28" aria-hidden="true"><path d="M4 28V12l4-3V4h4v3l4-3 4 3V4h4v5l4 3v16h-9v-6a3 3 0 0 0-6 0v6z" fill="var(--art-gold)" stroke="var(--art-gold-dark)" stroke-width="1.5"/></svg>
        <span class="brand-name">Бастион задач</span>
      </RouterLink>
      <nav class="nav" aria-label="Разделы">
        <RouterLink v-for="i in nav" :key="i.to" :to="i.to" class="nav-link" :aria-current="active(i.to) ? 'page' : undefined">
          <Icon :name="i.icon" :size="16" />{{ i.label }}
        </RouterLink>
      </nav>
      <div class="right">
        <ResourceBar v-if="session.gameOn && game.player" :player="game.player" compact class="game-only res-top" />
        <button v-if="session.plays" class="mode" :aria-pressed="session.me.game_mode" @click="toggleMode"
          :title="session.me.game_mode ? 'Переключить в рабочий режим: только канбан' : 'Включить игровой режим'">
          <Icon :name="session.me.game_mode ? 'gamepad' : 'briefcase'" :size="16" />
          <span class="mode-text">{{ session.me.game_mode ? 'Игра' : 'Работа' }}</span>
        </button>
        <RouterLink to="/settings" class="me" :title="`${session.me.display_name}, ${session.me.team_name || 'все команды'} — настройки`">
          <span class="avatar" aria-hidden="true">{{ session.me.display_name.split(' ').map(p => p[0]).join('').slice(0, 2) }}</span>
          <span class="me-name">{{ session.me.display_name.split(' ')[0] }}</span>
        </RouterLink>
        <button class="btn ghost icon" aria-label="Выйти" title="Выйти" @click="logout"><Icon name="logout" /></button>
      </div>
    </header>
    <div v-if="session.gameOn && game.player" class="res-mobile game-only"><ResourceBar :player="game.player" compact /></div>
    <main id="main"><RouterView /></main>
    <nav class="bottom" aria-label="Разделы">
      <RouterLink v-for="i in mobileNav" :key="i.to" :to="i.to" :aria-current="active(i.to) ? 'page' : undefined">
        <Icon :name="i.icon" :size="20" /><span>{{ i.label }}</span>
      </RouterLink>
    </nav>
  </template>
  <main v-else id="main"><RouterView /></main>
  <Toasts />
</template>

<style scoped>
.skip { position: absolute; left: -999px; top: 8px; z-index: 100; background: var(--color-surface); padding: 8px 12px; border-radius: var(--radius-sm); }
.skip:focus { left: 8px; }
.top { position: sticky; top: 0; z-index: 30; display: flex; align-items: center; gap: var(--space-4); padding: 0 var(--space-5); height: 60px; background: color-mix(in srgb, var(--color-surface) 92%, transparent); backdrop-filter: blur(8px); border-bottom: 1px solid var(--color-border); }
.brand { display: flex; align-items: center; gap: var(--space-2); text-decoration: none; color: var(--color-text); flex: none; }
.brand-name { font-family: var(--font-display); font-weight: 700; font-size: 1.125rem; }
.nav { display: flex; gap: 2px; min-width: 0; overflow-x: auto; scrollbar-width: none; }
.nav-link { display: inline-flex; align-items: center; gap: 6px; padding: 8px 10px; border-radius: var(--radius-sm); color: var(--color-text-muted); text-decoration: none; font-weight: 600; font-size: var(--text-sm); white-space: nowrap; }
.nav-link:hover { color: var(--color-text); background: var(--color-action-secondary-hover); }
.nav-link[aria-current="page"] { color: var(--color-action-primary); background: var(--color-selected); }
.right { margin-left: auto; display: flex; align-items: center; gap: var(--space-2); flex: none; }
.mode { display: inline-flex; align-items: center; gap: 6px; height: 32px; padding: 0 12px; border-radius: var(--radius-pill); border: 1px solid var(--color-border-strong); background: var(--color-surface); font-weight: 650; font-size: var(--text-sm); cursor: pointer; }
.mode:hover { background: var(--color-action-secondary-hover); }
.mode[aria-pressed="true"] { border-color: var(--color-gold); color: var(--color-gold); background: var(--color-gold-bg); }
.me { display: inline-flex; align-items: center; gap: 6px; text-decoration: none; color: var(--color-text); font-weight: 600; font-size: var(--text-sm); padding: 4px 8px 4px 4px; border-radius: var(--radius-pill); }
.me:hover { background: var(--color-action-secondary-hover); }
.avatar { display: grid; place-items: center; width: 30px; height: 30px; border-radius: 50%; background: var(--color-action-primary); color: var(--color-action-primary-text); font-size: var(--text-xs); font-weight: 700; }
.res-mobile { display: none; }
.bottom { display: none; }
@media (max-width: 1180px) { .me-name, .brand-name { display: none; } }
@media (max-width: 960px) { .res-top { display: none !important; } .res-mobile { display: flex; justify-content: center; padding: 8px 16px; border-bottom: 1px solid var(--color-border); background: var(--color-surface); } }
@media (max-width: 767px) {
  .top { padding: 0 var(--space-4); gap: var(--space-2); height: 56px; }
  .brand-name { display: inline; }
  .nav { display: none; }
  .mode-text { display: none; }
  .bottom { display: flex; position: fixed; bottom: 0; left: 0; right: 0; z-index: 30; background: var(--color-surface); border-top: 1px solid var(--color-border); padding: 6px 4px calc(6px + env(safe-area-inset-bottom)); }
  .bottom a { flex: 1; display: flex; flex-direction: column; align-items: center; gap: 2px; padding: 4px 0; min-height: var(--size-touch); color: var(--color-text-muted); text-decoration: none; font-size: 11px; font-weight: 600; }
  .bottom a[aria-current="page"] { color: var(--color-action-primary); }
}
@media (max-width: 380px) { .brand-name { display: none; } }
</style>
