<script setup lang="ts">
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
const session = useSession()
const toasts = useToasts()
async function on() { try { await session.setGameMode(true) } catch (e) { toasts.error(e) } }
</script>

<template>
  <div class="page">
    <div class="card panel stack off">
      <h1>Игра выключена</h1>
      <p v-if="session.plays" class="muted">Сейчас включён рабочий режим: только канбан, без игровых панелей и анимаций. Награды за задачи продолжают начисляться.</p>
      <p v-else class="muted">Город и бои доступны участникам команд. У администратора нет своего города.</p>
      <div class="row">
        <button v-if="session.plays" class="btn primary" @click="on">Включить игровой режим</button>
        <RouterLink class="btn" to="/board">К доске</RouterLink>
      </div>
    </div>
  </div>
</template>

<style scoped>
.off { max-width: 560px; }
.off h1 { font-size: var(--text-xl); font-family: var(--font-sans); }
.btn { text-decoration: none; }
</style>
