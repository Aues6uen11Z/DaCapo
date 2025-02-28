<template>
  <div class="card-title my-column">
    <div class="my-row tw-w-full tw-justify-between">
      <div class="my-row tw-h-full">
        <!-- Optional icon -->
        <q-icon v-if="icon" :name="icon" class="tw-text-2xl tw-h-full"></q-icon>
        <!-- Title name -->
        <div class="tw-text-2xl tw-h-full tw-content-center">{{ name }}</div>
      </div>
      <!-- Slot for additional components -->
      <slot></slot>
    </div>
    <!-- Optional help information -->
    <div v-if="help" class="tw-w-full tw-mt-[-10px]">
      <div class="tw-text-gray-500" style="white-space: pre-wrap">
        {{ help }}
      </div>
    </div>
    <!-- Separator line -->
    <q-separator class="tw-w-full tw-mb-4" />
  </div>
</template>

<script setup lang="ts">
interface Props {
  name: string;
  icon?: string;
  help?: string;
}

defineProps<Props>();
</script>
