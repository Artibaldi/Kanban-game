<script setup lang="ts">
import Icon from '../ui/Icon.vue'
import { date, isOverdue } from '../lib/format'
import { priorityName } from '../lib/board'
import type { Task } from '../types'

defineProps<{ task: Task; draggable?: boolean; stepLabel?: string; stepDisabled?: string }>()
defineEmits<{ open: []; step: [] }>()
</script>

<template>
  <article class="tcard" :class="['p-' + task.priority, { done: task.state === 'done' }]" :draggable="draggable" :data-id="task.id">
    <button class="open" @click="$emit('open')" :aria-label="`Открыть ${task.number}: ${task.title}`">
      <span class="head">
        <span class="number">{{ task.number }}</span>
        <span class="prio" :title="`Приоритет: ${priorityName[task.priority]}`">{{ task.priority }}</span>
        <span class="points" :class="{ unagreed: !task.points_approved }" :title="task.points_approved ? 'Сложность согласована' : 'Сложность не согласована'">
          <Icon v-if="!task.points_approved" name="alert" :size="12" />{{ task.points }}
        </span>
      </span>
      <span class="title">{{ task.title }}</span>
      <span class="meta">
        <span v-if="task.assignee_name" class="who"><Icon name="user" :size="13" />{{ task.assignee_name }}</span>
        <span v-else class="who muted">Без исполнителя</span>
        <span v-if="task.due_date" class="due" :class="{ late: task.state !== 'done' && isOverdue(task.due_date) }"><Icon name="clock" :size="13" />{{ date(task.due_date) }}</span>
      </span>
      <span v-if="task.reward" class="reward game-only">
        <span class="g">+{{ task.reward.gold }} золота</span><span class="e">+{{ task.reward.energy }} энергии</span>
      </span>
    </button>
    <button v-if="stepLabel" class="step" :disabled="!!stepDisabled" :title="stepDisabled || stepLabel" @click="$emit('step')">
      {{ stepLabel }}<Icon name="right" :size="14" />
    </button>
  </article>
</template>

<style scoped>
.tcard { position: relative; background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-md); box-shadow: var(--shadow-sm); border-left: 3px solid var(--color-p3); transition: box-shadow var(--duration-fast), transform var(--duration-fast); }
.tcard[draggable="true"] { cursor: grab; }
.tcard:hover { box-shadow: var(--shadow-md); }
.tcard.p-P1 { border-left-color: var(--color-p1); }
.tcard.p-P2 { border-left-color: var(--color-p2); }
.tcard.p-P4 { border-left-color: var(--color-p4); }
.tcard.done .title { color: var(--color-text-muted); }
.open { display: flex; flex-direction: column; gap: 6px; width: 100%; text-align: left; background: none; border: 0; padding: var(--space-3); cursor: pointer; border-radius: inherit; }
.head { display: flex; align-items: center; gap: 6px; }
.number { font-family: var(--font-mono); font-size: var(--text-xs); color: var(--color-text-muted); }
.prio { font-size: 10px; font-weight: 800; padding: 0 5px; border-radius: 4px; background: var(--color-surface-sunken); color: var(--color-text-muted); }
.p-P1 .prio { color: var(--color-p1); }
.p-P2 .prio { color: var(--color-p2); }
.points { margin-left: auto; display: inline-flex; align-items: center; gap: 2px; min-width: 24px; height: 22px; justify-content: center; padding: 0 6px; border-radius: var(--radius-pill); background: var(--color-selected); color: var(--color-action-primary); font-weight: 800; font-size: var(--text-xs); }
.points.unagreed { background: var(--color-warning-bg); color: var(--color-warning); }
.title { font-weight: 600; font-size: var(--text-sm); line-height: 1.35; }
.meta { display: flex; justify-content: space-between; gap: var(--space-2); font-size: var(--text-xs); color: var(--color-text-muted); }
.meta span { display: inline-flex; align-items: center; gap: 4px; }
.due.late { color: var(--color-danger); font-weight: 700; }
.reward { display: flex; gap: var(--space-2); font-size: 11px; font-weight: 700; }
.reward .g { color: var(--color-gold); }
.reward .e { color: var(--color-energy); }
.step { display: flex; align-items: center; justify-content: center; gap: 4px; width: 100%; border: 0; border-top: 1px dashed var(--color-border); background: transparent; padding: 6px; font-size: var(--text-xs); font-weight: 650; color: var(--color-action-primary); cursor: pointer; border-radius: 0 0 var(--radius-md) var(--radius-md); }
.step:hover:not(:disabled) { background: var(--color-selected); }
.step:disabled { color: var(--color-text-muted); cursor: not-allowed; opacity: 0.7; }
</style>
