<script setup lang="ts">
// The Arena: challenges to friendly duels, opened from the fortress. The challenge is the consent:
// a colleague accepts it, or posts an open challenge for the team. «Не сейчас» is quiet.
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import type { MemberAction } from '../lib/arena'
import { useRouter } from 'vue-router'
import Icon from '../ui/Icon.vue'
import { api } from '../api'
import { hoursLeft, memberAction, myOpen } from '../lib/arena'
import { ago } from '../lib/format'
import { avatarHue, initials } from '../lib/work'
import { useToasts } from '../stores/toast'
import type { ArenaMember, ArenaState } from '../types'

const props = defineProps<{ state: ArenaState }>()
const emit = defineEmits<{ close: []; changed: [] }>()
const router = useRouter()
const toasts = useToasts()
const busy = ref('')
const root = ref<HTMLElement>()

// The clock behind «ещё N ч», refreshed every minute while the Arena is open.
const now = ref(new Date())
let tick = 0
const pending = computed(() => props.state.incoming.filter(c => c.status === 'pending'))
const awaiting = computed(() => props.state.incoming.filter(c => c.status === 'accepted'))
const ready = computed(() => props.state.outgoing.filter(c => c.status === 'accepted'))
const open = computed(() => myOpen(props.state.outgoing))
const live = (c: { status: string }) => c.status === 'pending' || c.status === 'accepted'
// Closed challenges worth a word, unless a newer one to the same colleague is already waiting.
const recent = computed(() => props.state.outgoing.filter(c => !c.open && (c.status === 'declined' || c.status === 'closed') && !props.state.outgoing.some(x => x.to_id === c.to_id && live(x))).slice(0, 3))
const actions = computed(() => Object.fromEntries(props.state.team.map(m => [m.user_id, memberAction(m, props.state.outgoing)])) as Record<string, MemberAction>)
const limitLeft = computed(() => Math.max(0, props.state.per_day - props.state.today))

async function run(key: string, fn: () => Promise<unknown>, ok: string) {
  busy.value = key
  try { await fn(); toasts.push({ kind: 'success', title: ok }, 3000); emit('changed') } catch (e) { toasts.error(e) } finally { busy.value = '' }
}
const challenge = (m: { user_id: string; display_name: string }) =>
  run('c:' + m.user_id, () => api.post('/challenges', { to_id: m.user_id }), `Вызов отправлен: ${m.display_name}`)
const openChallenge = () => run('open', () => api.post('/challenges', {}), 'Ваш город открыт для боёв команды на сутки')
const cancel = (id: string) => run('x:' + id, () => api.post(`/challenges/${id}/cancel`), 'Вызов отменён')
const accept = (id: string, name: string) => run('a:' + id, () => api.post(`/challenges/${id}/accept`), `Вызов принят — ${name} может выходить в бой`)
const decline = (id: string) => run('d:' + id, () => api.post(`/challenges/${id}/decline`), 'Хорошо, в другой раз')
const fight = (id: string) => router.push(`/fortress/duels?challenge=${id}`)
const duel = (m: ArenaMember) => router.push(`/fortress/duels?opponent=${m.user_id}`)
const watchBattle = (id: string) => router.push(`/battles/${id}`)

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') { emit('close'); return }
  // Keep Tab inside the Arena.
  if (e.key !== 'Tab') return
  const f = [...(root.value?.querySelectorAll<HTMLElement>('button:not([disabled])') || [])]
  if (!f.length) return
  const i = f.indexOf(document.activeElement as HTMLElement)
  if (e.shiftKey && i <= 0) { e.preventDefault(); f[f.length - 1].focus() }
  else if (!e.shiftKey && (i === -1 || i === f.length - 1)) { e.preventDefault(); f[0].focus() }
}
onMounted(() => {
  addEventListener('keydown', onKey)
  tick = window.setInterval(() => (now.value = new Date()), 60000)
  nextTick(() => root.value?.querySelector<HTMLElement>('.g-x')?.focus())
})
onUnmounted(() => { removeEventListener('keydown', onKey); clearInterval(tick) })
</script>

<template>
  <div class="backdrop" @click.self="emit('close')">
    <div ref="root" class="g-panel pop arena-panel" role="dialog" aria-modal="true" aria-label="Арена: вызовы на дружеский бой">
      <span class="g-ribbon gold"><b>Арена</b></span>
      <button class="g-x" aria-label="Закрыть арену" @click="emit('close')"><Icon name="x" :size="18" /></button>
      <div class="scroll">

      <div class="top">
        <span class="g-chip">Боёв сегодня: {{ state.today }} из {{ state.per_day }}</span>
        <span class="hint">Бой бесплатный: войска не гибнут, ничего не теряется. +{{ state.friendly_xp }} опыта.</span>
      </div>

      <section v-if="pending.length || awaiting.length" class="block">
        <h3>Вас вызывают</h3>
        <div v-for="c in pending" :key="c.id" class="row-card hot">
          <span class="av" :style="{ '--hue': avatarHue(c.from_name) }" aria-hidden="true">{{ initials(c.from_name) }}</span>
          <span class="grow"><b>{{ c.from_name }}</b> вызывает вас на дружеский бой<small>ещё {{ hoursLeft(c.expires_at, now) }}</small></span>
          <button class="g-btn small" :disabled="!!busy" @click="accept(c.id, c.from_name)"><span>Принять</span></button>
          <button class="g-btn blue small" :disabled="!!busy" @click="decline(c.id)"><span>Не сейчас</span></button>
        </div>
        <div v-for="c in awaiting" :key="c.id" class="row-card">
          <span class="av" :style="{ '--hue': avatarHue(c.from_name) }" aria-hidden="true">{{ initials(c.from_name) }}</span>
          <span class="grow">Вы приняли вызов: <b>{{ c.from_name }}</b> выйдет на ваш город<small>ещё {{ hoursLeft(c.expires_at, now) }}</small></span>
        </div>
      </section>

      <section v-if="ready.length" class="block">
        <h3>Вызов принят — в бой!</h3>
        <div v-for="c in ready" :key="c.id" class="row-card hot">
          <span class="av" :style="{ '--hue': avatarHue(c.to_name || '') }" aria-hidden="true">{{ initials(c.to_name) }}</span>
          <span class="grow"><b>{{ c.to_name }}</b> принял(а) ваш вызов<small>ещё {{ hoursLeft(c.expires_at, now) }}</small></span>
          <button class="g-btn yellow small" :disabled="!limitLeft" @click="fight(c.id)"><span>В бой</span></button>
        </div>
      </section>

      <section v-if="state.against_me.length" class="block">
        <h3>Ваш город атаковали</h3>
        <div v-for="b in state.against_me" :key="b.id" class="row-card">
          <span class="av" :style="{ '--hue': avatarHue(b.attacker) }" aria-hidden="true">{{ initials(b.attacker) }}</span>
          <span class="grow"><b>{{ b.attacker }}</b>: {{ b.destruction }}% разрушений
            <span class="stars" :aria-label="`${b.stars} из 3 звёзд`"><Icon v-for="i in 3" :key="i" name="star" :size="13" :class="{ lit: i <= b.stars }" /></span>
            <small>{{ ago(b.created_at) }} · ничего не потеряно</small></span>
          <button class="g-btn blue small" @click="watchBattle(b.id)"><span>Смотреть</span></button>
          <button class="g-btn small" :disabled="!!busy" @click="challenge({ user_id: b.attacker_id, display_name: b.attacker })"><span>Ответный вызов</span></button>
        </div>
      </section>

      <section v-if="state.open.length" class="block">
        <h3>Открытые вызовы команды</h3>
        <div v-for="c in state.open" :key="c.id" class="row-card" :class="{ hot: !c.attacked }">
          <span class="av" :style="{ '--hue': avatarHue(c.from_name) }" aria-hidden="true">{{ initials(c.from_name) }}</span>
          <span class="grow">Город <b>{{ c.from_name }}</b> открыт для боя<small>ещё {{ hoursLeft(c.expires_at, now) }}</small></span>
          <span v-if="c.attacked" class="done">Вы ответили</span>
          <button v-else class="g-btn yellow small" :disabled="!limitLeft" @click="fight(c.id)"><span>В бой</span></button>
        </div>
      </section>

      <section class="block">
        <h3>Команда</h3>
        <div class="open-me">
          <template v-if="open">
            <span class="grow">Ваш город открыт для команды ещё {{ hoursLeft(open.expires_at, now) }}</span>
            <button class="g-btn red small" :disabled="!!busy" @click="cancel(open.id)"><span>Закрыть</span></button>
          </template>
          <template v-else>
            <span class="grow">Бросьте открытый вызов — любой коллега сможет напасть на ваш город один раз за сутки.</span>
            <button class="g-btn yellow small" :disabled="!!busy" :aria-busy="busy === 'open'" @click="openChallenge"><span>Открытый вызов</span></button>
          </template>
        </div>
        <p v-if="!state.team.length" class="empty">В команде пока нет других игроков.</p>
        <div class="team">
          <article v-for="m in state.team" :key="m.user_id" class="member">
            <span class="av big" :style="{ '--hue': avatarHue(m.display_name) }" aria-hidden="true">{{ initials(m.display_name) }}</span>
            <span class="th g-stroke s2" :title="`Ратуша ${m.townhall}`">{{ m.townhall }}</span>
            <strong>{{ m.display_name }}</strong>
            <span class="meta">башни {{ m.defense.towers }} · стены {{ m.defense.walls }}</span>
            <button v-if="actions[m.user_id].kind === 'challenge'" class="g-btn small" :disabled="!!busy" :aria-busy="busy === 'c:' + m.user_id" @click="challenge(m)"><span>Вызвать</span></button>
            <button v-else-if="actions[m.user_id].kind === 'duel'" class="g-btn yellow small" :disabled="!limitLeft" @click="duel(m)"><span>В бой</span></button>
            <button v-else-if="actions[m.user_id].kind === 'fight'" class="g-btn yellow small" :disabled="!limitLeft" @click="fight((actions[m.user_id] as { id: string }).id)"><span>Вызов принят — в бой</span></button>
            <span v-else class="wait">Ждёт ответа <button class="link" :disabled="!!busy" @click="cancel((actions[m.user_id] as { id: string }).id)">отменить</button></span>
          </article>
        </div>
        <p v-for="c in recent" :key="c.id" class="note">{{ c.to_name }}: {{ c.status === 'declined' ? 'не сейчас' : 'вызов истёк' }} — можно вызвать снова.</p>
        <p v-if="!limitLeft" class="note warn">Лимит боёв на сегодня исчерпан. Таверна в городе добавляет бои.</p>
      </section>
      </div>
    </div>
  </div>
</template>

<style scoped>
.backdrop { position: fixed; inset: 0; z-index: 210; display: grid; place-items: center; padding: 40px 12px 12px; background: rgba(10, 12, 34, 0.55); overflow-y: auto; }
/* The ribbon and the close button sit on the frame, so only the inside scrolls. */
.arena-panel { width: min(780px, 100%); max-height: calc(100dvh - 60px); padding-top: 46px; display: flex; flex-direction: column; }
.scroll { flex: 1; min-height: 0; overflow-y: auto; display: flex; flex-direction: column; gap: 14px; padding: 2px 4px 4px; scrollbar-width: thin; }
.top { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; justify-content: center; }
.hint { font: 600 13px var(--font-sans); color: #7a5530; }
.block h3 { font: 900 17px var(--font-game); color: #3a2410; margin-bottom: 8px; }
.row-card { display: flex; align-items: center; gap: 10px; padding: 8px 10px; margin-bottom: 8px; border: 2px solid var(--g-panel-b); border-radius: 14px; background: #fff8e6; font: 600 14px var(--font-sans); color: #3a2410; flex-wrap: wrap; }
.row-card.hot { background: linear-gradient(#fff4c4, #ffe98a); border-color: var(--g-ink); }
.row-card .g-btn { margin-bottom: 6px; }
.row-card small { display: block; font: 600 12px var(--font-sans); color: #7a5530; }
.av { display: grid; place-items: center; width: 34px; height: 34px; border-radius: 50%; flex: none; border: 2px solid var(--g-ink); background: hsl(var(--hue, 40) 70% 80%); font: 800 12px var(--font-sans); color: #1e1e1e; }
.av.big { width: 52px; height: 52px; font-size: 16px; }
.stars { display: inline-flex; gap: 1px; vertical-align: middle; color: #b9ae97; margin-left: 4px; }
.stars .lit { color: #e3a800; fill: #ffd323; }
.done, .wait { font: 700 13px var(--font-sans); color: #7a5530; }
.link { border: 0; background: none; padding: 0; color: #2459a8; font: inherit; text-decoration: underline; cursor: pointer; }
.open-me { display: flex; align-items: center; gap: 10px; padding: 10px 12px; margin-bottom: 10px; border-radius: 14px; background: rgba(107, 70, 38, 0.1); font: 600 14px var(--font-sans); color: #3a2410; flex-wrap: wrap; }
.open-me .g-btn { margin-bottom: 6px; }
.team { display: grid; grid-template-columns: repeat(auto-fill, minmax(170px, 1fr)); gap: 10px; }
.member { position: relative; display: flex; flex-direction: column; align-items: center; gap: 4px; padding: 12px 8px 6px; border: 2px solid var(--g-panel-b); border-radius: 16px; background: #fff8e6; color: #3a2410; text-align: center; }
.member strong { font: 800 15px var(--font-game); }
.member .g-btn { margin-top: 4px; }
.th { position: absolute; top: 8px; right: 8px; display: grid; place-items: center; width: 28px; height: 28px; border: 2px solid var(--g-ink); border-radius: 8px; background: linear-gradient(#7db8ff, #3e8ef0 55%, #2f78da); font-size: 14px; }
.meta { font: 600 12px var(--font-sans); color: #7a5530; }
.note { font: 600 12px var(--font-sans); color: #7a5530; margin-top: 6px; }
.note.warn { color: #9b1f1f; }
.empty { font: 600 14px var(--font-sans); color: #7a5530; text-align: center; }
@media (max-width: 600px) { .row-card .grow { flex-basis: 100%; } }
</style>
