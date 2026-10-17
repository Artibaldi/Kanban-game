<script setup lang="ts">
import { onMounted, ref } from 'vue'
import Icon from '../ui/Icon.vue'
import { api } from '../api'
import { applyTheme, loadTheme, type ThemeChoice } from '../ui/theme'
import { dateTime, signed } from '../lib/format'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import type { LedgerItem } from '../types'

const session = useSession()
const toasts = useToasts()
const theme = ref<ThemeChoice>(loadTheme())
const ledger = ref<LedgerItem[]>([])
const kind: Record<string, string> = {
  seed: 'Стартовый запас', task_reward: 'Награда за задачу', task_reward_reversal: 'Отмена награды', build: 'Постройка', upgrade: 'Улучшение',
  train: 'Обучение войск', attack: 'Атака', battle_reward: 'Награда за бой', admin: 'Корректировка',
}
onMounted(async () => {
  if (!session.plays) return
  try { ledger.value = (await api.get<{ items: LedgerItem[] }>('/ledger')).items } catch (e) { toasts.error(e) }
})
function setTheme(t: ThemeChoice) { theme.value = t; applyTheme(t) }
async function setMode(on: boolean) { try { await session.setGameMode(on) } catch (e) { toasts.error(e) } }
async function setFriendly(on: boolean) { try { await session.setAcceptFriendly(on) } catch (e) { toasts.error(e) } }
</script>

<template>
  <div class="page">
    <div class="page-head"><div><p class="eyebrow">{{ session.me?.display_name }} · {{ session.me?.login }}</p><h1>Настройки</h1></div></div>
    <div class="grid">
      <section class="card panel stack">
        <h2>Режим</h2>
        <div class="choice" role="radiogroup" aria-label="Режим интерфейса">
          <button role="radio" :aria-checked="!session.me?.game_mode" :disabled="!session.plays" @click="setMode(false)">
            <Icon name="briefcase" /><span><strong>Рабочий</strong><span class="small muted">Только канбан: без города, ресурсов, анимаций и игровых уведомлений. Награды копятся.</span></span>
          </button>
          <button role="radio" :aria-checked="!!session.me?.game_mode" :disabled="!session.plays" @click="setMode(true)">
            <Icon name="gamepad" /><span><strong>Игровой</strong><span class="small muted">Город, армия, бои и награды на карточках.</span></span>
          </button>
        </div>
        <p v-if="!session.plays" class="small muted">У администратора без команды нет города.</p>
        <p class="small muted">Игровые данные не используются для оценки работы. Режим «Работа» убирает игру, но командные награды продолжают копиться.</p>
        <label v-if="session.plays" class="row small check">
          <input type="checkbox" :checked="!!session.me?.accept_friendly" @change="setFriendly(($event.target as HTMLInputElement).checked)" />
          <span><strong>Принимать дружеские бои</strong><span class="muted"> — коллеги смогут потренироваться на вашем городе. Вы ничего не теряете.</span></span>
        </label>
        <h2>Тема</h2>
        <div class="tabs" role="tablist" aria-label="Тема">
          <button role="tab" :aria-selected="theme === 'system'" @click="setTheme('system')">Как в системе</button>
          <button role="tab" :aria-selected="theme === 'light'" @click="setTheme('light')"><Icon name="sun" :size="14" /> Светлая</button>
          <button role="tab" :aria-selected="theme === 'dark'" @click="setTheme('dark')"><Icon name="moon" :size="14" /> Тёмная</button>
        </div>
        <h2>Вход</h2>
        <p class="notice warning small">Демо-вход без пароля: только для локального прототипа. Производственная авторизация (пароли, SSO, MFA) не реализована.</p>
      </section>

      <section v-if="session.plays" class="card panel stack">
        <h2>Журнал операций</h2>
        <p class="small muted">Каждое изменение золота, энергии и опыта — отдельная запись. Награда связана с задачей; отмена награды — новая запись, исходная не удаляется.</p>
        <div class="table-scroll">
          <table class="table">
            <thead><tr><th>№</th><th>Операция</th><th class="r">Золото</th><th class="r">Энергия</th><th class="r">Опыт</th><th>Когда</th></tr></thead>
            <tbody>
              <tr v-for="l in ledger" :key="l.id" :class="{ rev: l.reversed }">
                <td class="num">{{ l.id }}</td>
                <td><strong>{{ kind[l.kind] || l.kind }}</strong><span v-if="l.task_number" class="badge" style="margin-left: 6px">{{ l.task_number }}</span>
                  <span v-if="l.reversed" class="badge">отменена</span><div class="xs muted">{{ l.note }}</div></td>
                <td class="r num" :class="{ pos: l.gold > 0, neg: l.gold < 0 }">{{ l.gold ? signed(l.gold) : '' }}</td>
                <td class="r num" :class="{ pos: l.energy > 0, neg: l.energy < 0 }">{{ l.energy ? signed(l.energy) : '' }}</td>
                <td class="r num" :class="{ pos: l.xp > 0, neg: l.xp < 0 }">{{ l.xp ? signed(l.xp) : '' }}</td>
                <td class="small">{{ dateTime(l.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.grid { display: grid; grid-template-columns: 400px minmax(0, 1fr); gap: var(--space-4); align-items: start; }
.choice { display: flex; flex-direction: column; gap: var(--space-2); }
.choice button { display: flex; gap: var(--space-3); align-items: flex-start; text-align: left; padding: var(--space-3); border-radius: var(--radius-md); border: 1px solid var(--color-border); background: var(--color-surface); cursor: pointer; }
.choice button span { display: flex; flex-direction: column; }
.choice button:hover:not(:disabled) { border-color: var(--color-action-primary); }
.choice button[aria-checked="true"] { border: 2px solid var(--color-action-primary); background: var(--color-selected); }
.choice button:disabled { opacity: 0.6; cursor: not-allowed; }
.r { text-align: right; }
.pos { color: var(--color-success); font-weight: 700; }
.neg { color: var(--color-danger); font-weight: 700; }
tr.rev td { opacity: 0.65; }
@media (max-width: 960px) { .grid { grid-template-columns: 1fr; } }
</style>
