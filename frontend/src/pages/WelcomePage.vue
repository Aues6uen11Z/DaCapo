<template>
  <div class="tw-h-full tw-w-full flex flex-center column">
    <div class="text-h3 text-weight-bold q-mb-md">
      {{ t('welcome.title') }}
    </div>
    <div class="text-subtitle1 text-grey-7">
      {{ t('welcome.description') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
</script>
