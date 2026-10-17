<script setup lang="ts">
// Shown once, the first time a player opens the game: how the co-op works and that it never judges work.
import Modal from '../ui/Modal.vue'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'

const session = useSession()
const toasts = useToasts()
async function ack() { try { await session.ackGameRules() } catch (e) { toasts.error(e) } }
</script>

<template>
  <Modal :open="true" title="Как устроена игра" @close="ack">
    <div class="stack">
      <p>Команда вместе защищает крепость от своего бэклога. Задачи, взятые в план, — враги; подтверждённая задача — побеждённый враг.</p>
      <p>Раз в неделю волна подводит итог, и вся команда получает общую награду.</p>
      <p><strong>Игровые данные не используются для оценки работы.</strong></p>
      <div><button class="btn primary" @click="ack">Понятно</button></div>
    </div>
  </Modal>
</template>
