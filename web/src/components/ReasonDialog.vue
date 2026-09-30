<script setup lang="ts">
import { ref, watch } from 'vue'
import Modal from '../ui/Modal.vue'
const props = defineProps<{ open: boolean; title: string; text: string; action: string; destructive?: boolean }>()
const emit = defineEmits<{ close: []; confirm: [comment: string] }>()
const comment = ref('')
const touched = ref(false)
watch(() => props.open, v => { if (v) { comment.value = ''; touched.value = false } })
function submit() {
  touched.value = true
  if (comment.value.trim()) emit('confirm', comment.value.trim())
}
</script>

<template>
  <Modal :open="open" :title="title" @close="emit('close')">
    <form class="stack" @submit.prevent="submit">
      <p class="small">{{ text }}</p>
      <label class="field">Причина (обязательно)
        <textarea v-model="comment" class="textarea" :aria-invalid="touched && !comment.trim()" required maxlength="1000" />
        <span v-if="touched && !comment.trim()" class="hint" style="color: var(--color-danger)">Укажите причину — исполнитель увидит её в карточке.</span>
      </label>
    </form>
    <template #footer>
      <button class="btn" @click="emit('close')">Отмена</button>
      <button class="btn" :class="destructive ? 'danger' : 'primary'" @click="submit">{{ action }}</button>
    </template>
  </Modal>
</template>
