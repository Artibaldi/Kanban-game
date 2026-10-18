<script setup lang="ts">
// A task card filled with its stage colour. The title button opens the card and stretches over it;
// the «⋯» button sits beside it, so there are no nested interactive elements.
import { computed } from 'vue'
import Icon from '../ui/Icon.vue'
import { date, isOverdue, plural } from '../lib/format'
import { avatarHue, cardReward, initials } from '../lib/work'
import { priorityName } from '../lib/board'
import { useGame } from '../stores/game'
import type { Task } from '../types'

const props = defineProps<{ task: Task; menu?: boolean }>()
const emit = defineEmits<{ open: []; menu: [anchor: HTMLElement] }>()
const game = useGame()

const late = computed(() => props.task.state !== 'done' && isOverdue(props.task.due_date))
const when = computed(() => {
  const t = props.task
  if (t.state === 'done') return t.approved_at ? `сдано ${date(t.approved_at)}` : 'сдано'
  return t.due_date ? `до ${date(t.due_date)}` : 'без срока'
})
const reward = computed(() => cardReward(props.task, game.config?.settings))
</script>

<template>
  <article class="tcard" :class="'s-' + task.state" :data-id="task.id">
    <div class="c-top">
      <span class="c-due" :class="{ late }"><Icon :name="task.state === 'done' ? 'check' : late ? 'alert' : 'calendar'" :size="14" />{{ late ? `просрочено · ${date(task.due_date)}` : when }}</span>
      <span class="prio" :title="`Приоритет: ${priorityName[task.priority]}`">{{ task.priority }}</span>
    </div>
    <button v-if="menu" class="c-more" aria-haspopup="menu" :aria-label="`Действия с ${task.number}`" @click.stop="emit('menu', $event.currentTarget as HTMLElement)">
      <Icon name="more" :size="18" />
    </button>
    <p class="c-key">{{ task.number }} · {{ task.project_name }}</p>
    <h3 class="c-title"><button class="open" @click="emit('open')">{{ task.title }}</button></h3>
    <p class="c-meta">{{ task.assignee_name || 'Без исполнителя' }}</p>
    <div class="c-bot">
      <span class="c-val" :title="task.points_approved ? 'Сложность согласована' : 'Сложность ещё не согласована'">
        <b class="num">{{ task.points }}</b><span>{{ plural(task.points, 'очко', 'очка', 'очков') }}</span>
        <Icon v-if="!task.points_approved" name="alert" :size="14" class="unagreed" label="оценка не согласована" />
      </span>
      <span v-if="reward !== null" class="gold-chip game-only" :title="task.reward ? 'Начислено исполнителю' : 'Награда после подтверждения'">
        <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><circle cx="12" cy="12" r="9" fill="#FFD75A" stroke="#7A4A00" stroke-width="2"/><path d="M12 8v8" stroke="#7A4A00" stroke-width="2" stroke-linecap="round"/></svg>+{{ reward }}
      </span>
      <span v-if="task.assignee_name" class="av" :style="{ '--hue': avatarHue(task.assignee_name) }" aria-hidden="true">{{ initials(task.assignee_name) }}</span>
    </div>
  </article>
</template>

<style scoped>
.tcard { position: relative; background: var(--fill); color: var(--ink); border-radius: 20px; padding: 14px 14px 14px 16px; user-select: none; -webkit-user-select: none; -webkit-touch-callout: none; box-shadow: 0 1px 2px rgba(60, 40, 20, 0.08), 0 6px 16px rgba(60, 40, 20, 0.08); transition: transform 0.2s var(--ease-emphasized), box-shadow 0.2s var(--ease-emphasized); }
.tcard.s-backlog { box-shadow: inset 0 0 0 1px var(--line), var(--shadow-sm); }
@media (hover: hover) { .tcard:hover { transform: translateY(-3px); box-shadow: 0 2px 4px rgba(60, 40, 20, 0.08), 0 16px 30px rgba(60, 40, 20, 0.16); } }
.tcard:has(.open:focus-visible) { outline: 3px solid var(--color-focus); outline-offset: 3px; }
.c-top { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--mute); min-height: 30px; padding-right: 36px; }
.c-due { display: inline-flex; align-items: center; gap: 5px; white-space: nowrap; }
.c-due.late { padding: 3px 8px 3px 6px; border-radius: var(--radius-pill); background: #fff; color: #a8341a; font-weight: 600; }
.prio { font-size: 11px; font-weight: 700; padding: 2px 7px; border-radius: var(--radius-pill); background: var(--soft); color: var(--ink); }
.c-more { position: absolute; z-index: 2; top: 14px; right: 14px; display: grid; place-items: center; width: 30px; height: 30px; border: 0; border-radius: 50%; background: var(--soft); color: var(--ink); cursor: pointer; transition: background 0.2s; }
.c-more:hover { background: var(--line); }
.c-key { font-size: 12px; font-weight: 500; color: var(--mute); margin-top: 10px; }
.c-title { font-size: 16px; line-height: 1.22; font-weight: 700; margin-top: 3px; letter-spacing: -0.015em; text-wrap: balance; }
.open { all: unset; cursor: pointer; }
.open::after { content: ""; position: absolute; inset: 0; border-radius: inherit; z-index: 1; }
.open:focus-visible { outline: none; }
.c-meta { font-size: 12px; color: var(--mute); margin-top: 6px; }
.c-bot { display: flex; align-items: flex-end; gap: 8px; margin-top: 16px; }
.c-val { display: flex; align-items: baseline; gap: 4px; white-space: nowrap; }
.c-val b { font-size: 30px; line-height: 0.85; font-weight: 700; letter-spacing: -0.04em; }
.c-val span { font-size: 12px; color: var(--mute); }
.unagreed { align-self: center; color: var(--ink); opacity: 0.8; }
.gold-chip { align-self: center; }
.av { margin-left: auto; }
</style>
