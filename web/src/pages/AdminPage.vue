<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { useToasts } from '../stores/toast'
import type { Settings } from '../types'

interface AdminUser { id: string; login: string; display_name: string; role: string; team_id: string | null; team_name: string; title: string; active: boolean }
const toasts = useToasts()
const users = ref<AdminUser[]>([])
const teams = ref<{ id: string; name: string }[]>([])
const settings = ref<Settings | null>(null)
const defaults = ref<Settings | null>(null)
const saving = ref(false)

async function load() {
  try {
    const [u, t, s] = await Promise.all([
      api.get<{ items: AdminUser[] }>('/admin/users'), api.get<{ items: { id: string; name: string }[] }>('/teams'),
      api.get<{ settings: Settings; defaults: Settings }>('/admin/settings'),
    ])
    users.value = u.items; teams.value = t.items; settings.value = s.settings; defaults.value = s.defaults
  } catch (e) { toasts.error(e) }
}
onMounted(load)
async function patchUser(u: AdminUser, body: Record<string, unknown>) {
  try { await api.patch(`/admin/users/${u.id}`, body); toasts.push({ kind: 'success', title: `${u.display_name}: сохранено` }, 2000) } catch (e) { toasts.error(e) }
  load()
}
async function save() {
  if (!settings.value) return
  saving.value = true
  try {
    settings.value = (await api.put<{ settings: Settings }>('/admin/settings', settings.value)).settings
    toasts.push({ kind: 'success', title: 'Коэффициенты сохранены', text: 'Новые значения действуют для следующих подтверждений и боёв.' })
  } catch (e) { toasts.error(e) } finally { saving.value = false }
}
const fields: { key: keyof Settings; label: string; step: number; hint: string }[] = [
  { key: 'gold_per_point', label: 'Золото за балл', step: 1, hint: 'золото = это × баллы × приоритет × (1 + бонус за срок)' },
  { key: 'energy_per_point', label: 'Энергия за балл', step: 1, hint: 'энергия = это × баллы' },
  { key: 'xp_per_point', label: 'Опыт за балл', step: 1, hint: 'опыт = это × баллы' },
  { key: 'on_time_bonus', label: 'Бонус за срок', step: 0.05, hint: 'доля, 0.1 = +10% золота' },
  { key: 'level_xp_base', label: 'Опыт уровня: база', step: 10, hint: 'до уровня n+1 нужно база × n^степень' },
  { key: 'level_xp_exp', label: 'Опыт уровня: степень', step: 0.1, hint: '' },
  { key: 'attack_energy', label: 'Энергия на атаку', step: 1, hint: '' },
  { key: 'shield_minutes', label: 'Щит после атаки, мин', step: 1, hint: '' },
  { key: 'rematch_minutes', label: 'Повторная атака, мин', step: 1, hint: 'пауза перед атакой того же игрока' },
  { key: 'random_spread', label: 'Случайный разброс', step: 0.01, hint: 'до 0.3; 0.1 = ±10% урона' },
  { key: 'battle_gold_base', label: 'Золото за бой: база', step: 5, hint: '' },
  { key: 'battle_gold_per_percent', label: 'Золото за % разрушений', step: 0.5, hint: '' },
  { key: 'battle_xp_per_star', label: 'Опыт за звезду', step: 1, hint: '' },
  { key: 'time_scale', label: 'Множитель таймеров', step: 0.1, hint: '1 = короткие демо-таймеры' },
]
</script>

<template>
  <div class="page">
    <div class="page-head"><div><p class="eyebrow">Администрирование</p><h1>Пользователи и экономика</h1></div></div>
    <section class="card panel stack">
      <h2>Пользователи</h2>
      <div class="table-scroll">
        <table class="table">
          <thead><tr><th>Пользователь</th><th>Логин</th><th>Роль</th><th>Команда</th><th>Активен</th></tr></thead>
          <tbody>
            <tr v-for="u in users" :key="u.id">
              <td><strong>{{ u.display_name }}</strong><div class="xs muted">{{ u.title }}</div></td>
              <td class="mono">{{ u.login }}</td>
              <td><select class="select" :value="u.role" :aria-label="`Роль: ${u.display_name}`" @change="patchUser(u, { role: ($event.target as HTMLSelectElement).value })">
                <option value="user">Пользователь</option><option value="lead">Руководитель</option><option value="admin">Администратор</option></select></td>
              <td><select class="select" :value="u.team_id || ''" :aria-label="`Команда: ${u.display_name}`" @change="patchUser(u, { team_id: ($event.target as HTMLSelectElement).value || null })">
                <option value="">Без команды</option><option v-for="t in teams" :key="t.id" :value="t.id">{{ t.name }}</option></select></td>
              <td><input type="checkbox" :checked="u.active" :aria-label="`Активен: ${u.display_name}`" @change="patchUser(u, { active: ($event.target as HTMLInputElement).checked })" /></td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section v-if="settings" class="card panel stack" style="margin-top: 20px">
      <h2>Коэффициенты экономики</h2>
      <form class="form" @submit.prevent="save">
        <label v-for="f in fields" :key="f.key" class="field">{{ f.label }}
          <input v-model.number="(settings as any)[f.key]" type="number" :step="f.step" min="0" class="input" />
          <span class="hint">{{ f.hint }}{{ f.hint ? ' · ' : '' }}по умолчанию {{ (defaults as any)?.[f.key] }}</span>
        </label>
        <label v-for="p in ['P1', 'P2', 'P3', 'P4']" :key="p" class="field">Множитель {{ p }}
          <input v-model.number="settings.priority_mult[p]" type="number" step="0.05" min="0" class="input" />
        </label>
        <label class="field">Соперники
          <select v-model="settings.battle_scope" class="select"><option value="team">Только своя команда</option><option value="all">Все команды</option></select>
          <span class="hint">Из чужой команды видны только город и трофеи, не задачи.</span>
        </label>
      </form>
      <div><button class="btn primary" :aria-busy="saving" :disabled="saving" @click="save">Сохранить коэффициенты</button></div>
    </section>
  </div>
</template>

<style scoped>
.form { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: var(--space-4); }
.mono { font-family: var(--font-mono); }
.table .select { min-width: 150px; }
</style>
