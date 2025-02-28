<template>
  <!-- File input field with file selection button -->
  <q-input
    v-model="filePath"
    :disable="disabled"
    dense
    @update:model-value="emit('update:modelValue', $event?.toString() || '')"
  >
    <template v-slot:append>
      <!-- File selection button -->
      <q-btn
        flat
        dense
        color="primary"
        icon="folder_open"
        @click="selectFile"
      />
    </template>
  </q-input>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { SelectFile } from 'app/wailsjs/go/app/App';

const props = defineProps<{
  modelValue: string;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void;
}>();

const filePath = ref(props.modelValue);

// Handler for file selection dialog
const selectFile = async () => {
  try {
    const path = await SelectFile();
    filePath.value = path;
    emit('update:modelValue', path);
  } catch (err) {
    console.error('Failed to select file:', err);
  }
};
</script>
