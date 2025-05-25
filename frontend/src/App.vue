<template>
  <router-view v-if="!loading" />
  <div v-else class="fullscreen flex flex-center">
    <q-spinner-dots color="primary" size="40" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, onUnmounted } from 'vue';
import {
  useIstStore,
  useSchedulerStore,
  useSettingsStore,
} from './stores/global-store';
import { useI18n } from 'vue-i18n';
import { useQuasar } from 'quasar';
import {
  updateRepo,
  updateSchedulerState,
  sendSchedulerCron,
} from './services/api';

const istStore = useIstStore();
const taskStore = useSchedulerStore();
const settingsStore = useSettingsStore();
const loading = ref(true);
const i18n = useI18n();
const $q = useQuasar();

// Auto-update function for all instances
const autoUpdateInstances = async () => {
  try {
    // Iterate through all instances
    for (const instanceName in istStore.layout) {
      const updateSettings =
        istStore.layout[instanceName]?.Project?.Update?._Base;
      const autoUpdate = updateSettings?.auto_update?.value;

      if (autoUpdate) {
        try {
          const isUpdated = await updateRepo(instanceName);
          if (isUpdated) {
            // If updates are available, reload instance configuration
            await istStore.loadInstance(instanceName);
            $q.notify({
              type: 'positive',
              message: `${instanceName}: Updated successfully`,
            });
          }
        } catch (err) {
          console.error(`Failed to auto update ${instanceName}:`, err);
          $q.notify({
            type: 'negative',
            message: `${instanceName}: Failed to auto update`,
          });
        }
      }
    }
  } catch (err) {
    console.error('Auto update failed:', err);
  }
};

// Initialize WebSocket connection
let unsubscribe: (() => void) | null = null;

onMounted(async () => {
  try {
    // Load settings
    settingsStore.loadSettings(i18n, $q);

    // Load instance data
    await istStore.loadInstance();
    unsubscribe = taskStore.initWebSocket();

    if (settingsStore.runOnStartup) {
      // if runOnStartup is true, start all instances
      await updateSchedulerState('start', '', true);
    } else if (settingsStore.schedulerCron) {
      // if runOnStartup is false, set the cron expression
      await sendSchedulerCron(settingsStore.schedulerCron);
    }
  } catch (err) {
    console.error('Failed to initialize app:', err);
  } finally {
    loading.value = false;

    // Execute auto-update asynchronously after loading is complete
    autoUpdateInstances().catch((err) => {
      console.error('Auto update failed:', err);
    });
  }
});

onUnmounted(() => {
  if (unsubscribe) {
    unsubscribe();
  }
});
</script>
