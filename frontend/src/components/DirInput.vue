<template>
  <!-- Directory input field with directory selection button -->
  <q-input
    v-model="dirPath"
    :disable="disabled"
    dense
    @update:model-value="emit('update:modelValue', $event?.toString() || '')"
  >
    <template v-slot:append>
      <!-- Directory selection button -->
      <q-btn flat dense color="primary" icon="folder_open" @click="selectDir" />
    </template>
  </q-input>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { SelectDir } from 'app/wailsjs/go/app/App';

const props = defineProps<{
  modelValue: string;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void;
}>();

const dirPath = ref(props.modelValue);

// Handler for directory selection dialog
const selectDir = async () => {
  try {
    const path = await SelectDir();
    dirPath.value = path;
    emit('update:modelValue', path);
  } catch (err) {
    console.error('Failed to select directory:', err);
  }
};
</script>
