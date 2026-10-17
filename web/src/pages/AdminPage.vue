<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { useToasts } from '../stores/toast'
import type { Settings, TeamSettings } from '../types'

interface AdminUser { id: string; login: string; display_name: string; role: string; team_id: string | null; team_name: string; title: string; active: boolean }
const toasts = useToasts()
const users = ref<AdminUser[]>([])
const teams = ref<{ id: string; name: string }[]>([])
const settings = ref<Settings | null>(null)
const defaults = ref<Settings | null>(null)
const saving = ref(false)
const teamSettings = ref<TeamSettings[]>([])
const dayNames = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс']

async function load() {
  try {
    const [u, t, s] = await Promise.all([
      api.get<{ items: AdminUser[] }>('/admin/users'), api.get<{ items: { id: string; name: string }[] }>('/teams'),
      api.get<{ settings: Settings; defaults: Settings }>('/admin/settings'),
    ])
    users.value = u.items; teams.value = t.items; settings.value = s.settings; defaults.value = s.defaults
    teamSettings.value = (await api.get<{ items: TeamSettings[] }>('/admin/teams')).items
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
async function patchTeam(t: TeamSettings, body: Partial<Pick<TeamSettings, 'work_days' | 'wave_weeks'>>) {
  try {
    const next = await api.patch<TeamSettings>(`/admin/teams/${t.id}`, body)
    Object.assign(t, next)
    toasts.push({ kind: 'success', title: `${t.name}: сохранено`, text: body.wave_weeks ? 'Новая длина волны начнёт действовать со следующей волны.' : undefined }, 2500)
  } catch (e) { toasts.error(e); load() }
}
function toggleDay(t: TeamSettings, d: number, on: boolean) {
  patchTeam(t, { work_days: on ? [...t.work_days, d].sort() : t.work_days.filter(x => x !== d) })
}
const fields: { key: keyof Settings; label: string; step: number; hint: string }[] = [
  { key: 'gold_per_point', label: 'Золото за балл', step: 1, hint: 'золото = это × баллы × приоритет × (1 + бонус за срок)' },
  { key: 'energy_per_point', label: 'Энергия за балл', step: 1, hint: 'энергия = это × баллы' },
  { key: 'xp_per_point', label: 'Опыт за балл', step: 1, hint: 'опыт = это × баллы' },
  { key: 'on_time_bonus', label: 'Бонус за срок', step: 0.05, hint: 'доля, 0.1 = +10% золота' },
  { key: 'level_xp_base', label: 'Опыт уровня: база', step: 10, hint: 'до уровня n+1 нужно база × n^степень' },
  { key: 'level_xp_exp', label: 'Опыт уровня: степень', step: 0.1, hint: '' },
  { key: 'random_spread', label: 'Случайный разброс', step: 0.01, hint: 'до 0.3; 0.1 = ±10% урона' },
  { key: 'overdue_damage_per_point', label: 'Урон за балл просрочки, %', step: 0.5, hint: 'в сутки, по рабочим дням' },
  { key: 'fortress_floor', label: 'Минимум прочности, %', step: 1, hint: 'ниже крепость не опускается' },
  { key: 'max_defense_bonus', label: 'Бонус обороны, макс.', step: 0.05, hint: 'доля к общей награде волны' },
  { key: 'defense_score_full', label: 'Оборона для полного бонуса', step: 100, hint: 'средняя оборона участника' },
  { key: 'friendly_xp', label: 'Опыт за дружеский бой', step: 1, hint: '' },
  { key: 'friendly_per_day', label: 'Дружеских боёв в день', step: 1, hint: '' },
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
        <label class="field">Пороги корон
          <span class="row"><input v-model.number="settings.crown_thresholds[0]" type="number" step="0.05" min="0" max="1" class="input" aria-label="Порог 1 короны" />
          <input v-model.number="settings.crown_thresholds[1]" type="number" step="0.05" min="0" max="1" class="input" aria-label="Порог 2 корон" /></span>
          <span class="hint">доля плана для 1 и 2 корон; 3 короны — весь план и ни одной просрочки</span>
        </label>
        <label v-for="(c, i) in [1, 2, 3]" :key="'chest' + c" class="field">Общая награда за {{ c }} {{ c === 1 ? 'корону' : 'короны' }}
          <span class="row"><input v-model.number="settings.wave_chest_gold[i + 1]" type="number" step="10" min="0" class="input" :aria-label="`Золото за ${c}`" />
          <input v-model.number="settings.wave_chest_energy[i + 1]" type="number" step="5" min="0" class="input" :aria-label="`Энергия за ${c}`" /></span>
          <span class="hint">золото и энергия каждому участнику</span>
        </label>
        <label v-for="(c, i) in [1, 2]" :key="'rep' + c" class="field">Ремонт за {{ c }} {{ c === 1 ? 'корону' : 'короны' }}, %
          <input v-model.number="settings.repair_by_crowns[i + 1]" type="number" step="5" min="0" max="100" class="input" />
          <span class="hint">3 короны восстанавливают крепость полностью</span>
        </label>
      </form>
      <div><button class="btn primary" :aria-busy="saving" :disabled="saving" @click="save">Сохранить коэффициенты</button></div>
    </section>

    <section v-if="teamSettings.length" class="card panel stack" style="margin-top: 20px">
      <h2>Команды: рабочие дни и волны</h2>
      <p class="small muted">В нерабочие дни просроченные задачи не бьют по крепости. Длина волны — от 1 до 4 недель.</p>
      <div v-for="t in teamSettings" :key="t.id" class="team-row">
        <strong class="team-name">{{ t.name }}</strong>
        <fieldset class="days">
          <legend class="sr-only">Рабочие дни команды {{ t.name }}</legend>
          <label v-for="(d, i) in dayNames" :key="d" class="day"><input type="checkbox" :checked="t.work_days.includes(i + 1)" @change="toggleDay(t, i + 1, ($event.target as HTMLInputElement).checked)" />{{ d }}</label>
        </fieldset>
        <label class="row small">Волна
          <select class="select" :value="t.wave_weeks" :aria-label="`Длина волны: ${t.name}`" @change="patchTeam(t, { wave_weeks: +($event.target as HTMLSelectElement).value })">
            <option v-for="n in 4" :key="n" :value="n">{{ n }} {{ n === 1 ? 'неделя' : 'недели' }}</option>
          </select>
        </label>
      </div>
    </section>
  </div>
</template>

<style scoped>
.form { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: var(--space-4); }
.mono { font-family: var(--font-mono); }
.table .select { min-width: 150px; }
.team-row { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-4); padding: var(--space-2) 0; border-top: 1px solid var(--color-border); }
.team-name { min-width: 120px; }
.days { display: flex; flex-wrap: wrap; gap: var(--space-2); border: 0; margin: 0; padding: 0; }
.day { display: inline-flex; align-items: center; gap: 4px; font-size: var(--text-sm); }
</style>
