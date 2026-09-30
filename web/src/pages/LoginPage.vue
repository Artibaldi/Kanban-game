<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from '../ui/Icon.vue'
import { api } from '../api'
import { useSession } from '../stores/session'

interface DemoUser { login: string; display_name: string; role: string; team: string; title: string }
const users = ref<DemoUser[]>([])
const error = ref('')
const busy = ref('')
const session = useSession()
const router = useRouter()
const route = useRoute()
const roleName: Record<string, string> = { admin: 'Администратор', lead: 'Руководитель', user: 'Пользователь' }

onMounted(async () => {
  try { users.value = (await api.get<{ items: DemoUser[] }>('/demo-users')).items } catch (e) { error.value = (e as Error).message }
})

async function enter(login: string) {
  busy.value = login
  error.value = ''
  try {
    await session.login(login)
    router.replace(typeof route.query.next === 'string' ? route.query.next : '/')
  } catch (e) { error.value = (e as Error).message } finally { busy.value = '' }
}
const teams = () => [...new Set(users.value.map(u => u.team))]
</script>

<template>
  <div class="login">
    <section class="hero">
      <svg class="skyline" viewBox="0 0 400 140" aria-hidden="true">
        <path d="M0 140V96h30V80h14v16h20V60l22-18 22 18v36h18V70h12V56h10v14h12v26h26V40l28-24 28 24v56h20V76h14v20h30V88h16v8h20v44z" fill="var(--art-stone)" opacity="0.5"/>
        <path d="M168 40l28-24 28 24" fill="none" stroke="var(--art-gold)" stroke-width="3"/>
        <rect x="192" y="0" width="3" height="18" fill="var(--art-wood-dark)"/><path d="M195 1h16l-5 5 5 5h-16z" fill="var(--art-banner)"/>
      </svg>
      <p class="eyebrow">Канбан и стратегия</p>
      <h1>Бастион задач</h1>
      <p class="lead">Выполняйте задачи команды, получайте золото, энергию и опыт, стройте город и защищайте его.
        Рабочий рейтинг считается только по подтверждённым задачам.</p>
    </section>
    <section class="card panel pick" aria-labelledby="pick-title">
      <div class="notice warning row"><Icon name="alert" /><span><strong>Демо-вход без пароля.</strong> Только для локального прототипа — это не производственная авторизация.</span></div>
      <p v-if="route.query.expired" class="notice info">Сессия закончилась. Войдите снова.</p>
      <h2 id="pick-title">Войти как</h2>
      <p v-if="error" class="notice danger" role="alert">{{ error }}</p>
      <div v-for="team in teams()" :key="team" class="group">
        <p class="eyebrow">{{ team || 'Без команды' }}</p>
        <div class="grid">
          <button v-for="u in users.filter(x => x.team === team)" :key="u.login" class="user" :aria-busy="busy === u.login" :disabled="!!busy" @click="enter(u.login)">
            <span class="avatar" aria-hidden="true">{{ u.display_name.split(' ').map(p => p[0]).join('').slice(0, 2) }}</span>
            <span class="grow">
              <strong>{{ u.display_name }}</strong>
              <span class="small muted">{{ roleName[u.role] }} · {{ u.title }}</span>
            </span>
            <span class="login-id xs muted">{{ u.login }}</span>
          </button>
        </div>
      </div>
      <p class="small muted">Сценарий для проверки: <strong>alexey</strong> берёт задачу и отправляет на проверку, <strong>irina</strong> подтверждает, затем <strong>alexey</strong> строит город и атакует.</p>
    </section>
  </div>
</template>

<style scoped>
.login { min-height: 100vh; display: grid; grid-template-columns: 1fr minmax(0, 520px); gap: var(--space-10); align-items: center; max-width: 1120px; margin: 0 auto; padding: var(--space-8) var(--space-6); }
.hero h1 { font-size: 3.25rem; margin: var(--space-2) 0 var(--space-4); }
.hero .lead { font-size: var(--text-lg); color: var(--color-text-muted); max-width: 46ch; }
.skyline { width: 100%; max-width: 420px; margin-bottom: var(--space-6); }
.pick { display: flex; flex-direction: column; gap: var(--space-4); }
.notice.row { align-items: flex-start; }
.group { display: flex; flex-direction: column; gap: var(--space-2); }
.grid { display: flex; flex-direction: column; gap: var(--space-2); }
.user { display: flex; align-items: center; gap: var(--space-3); text-align: left; padding: var(--space-2) var(--space-3); min-height: var(--size-touch); border: 1px solid var(--color-border); border-radius: var(--radius-md); background: var(--color-surface); cursor: pointer; }
.user:hover:not(:disabled) { border-color: var(--color-action-primary); background: var(--color-selected); }
.user:disabled { opacity: 0.6; cursor: wait; }
.user .grow { display: flex; flex-direction: column; }
.avatar { display: grid; place-items: center; width: 36px; height: 36px; border-radius: 50%; background: var(--color-action-primary); color: var(--color-action-primary-text); font-weight: 700; font-size: var(--text-sm); flex: none; }
.login-id { font-family: var(--font-mono); }
@media (max-width: 900px) {
  .login { grid-template-columns: 1fr; gap: var(--space-5); padding: var(--space-5) var(--space-4); }
  .hero h1 { font-size: 2.375rem; }
  .skyline { max-width: 260px; margin-bottom: var(--space-3); }
}
</style>
