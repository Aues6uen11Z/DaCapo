<template>
  <q-layout view="hHh LpR lFr">
    <!-- Left sidebar with instance list -->
    <q-drawer
      :width="75"
      show-if-above
      behavior="desktop"
      bordered
      class="my-column tw-p-0 tw-justify-between tw-bg-gradient-to-t tw-from-fuchsia-100"
    >
      <q-scroll-area class="tw-w-full tw-h-4/5" :thumb-style="{ width: '4px' }">
        <div
          ref="tabsEl"
          class="tw-flex tw-flex-col tw-text-violet-400"
          role="tablist"
        >
          <div
            v-for="instance in draggableInstances"
            :key="instance"
            class="tw-relative"
          >
            <q-btn
              :class="[
                'tw-w-full tw-flex tw-flex-col tw-items-center tw-justify-center tw-normal-case tw-py-2',
                activeInstance === instance ? 'text-primary' : '',
              ]"
              :aria-selected="activeInstance === instance"
              role="tab"
              flat
              no-caps
              stack
              @click="activeInstance = instance"
            >
              <q-icon
                :name="istStore.ready[instance] ? 'rocket_launch' : 'rocket'"
                size="sm"
              />
              <div class="tw-text-xs tw-mt-1">{{ instance }}</div>
            </q-btn>
            <q-badge
              v-if="taskStore.getInstanceState(instance) === 'running'"
              color="positive"
              floating
              transparent
              class="tw-w-2 tw-h-2 tw-rounded-full"
              style="top: 3px; right: 3px"
            />
            <q-badge
              v-if="taskStore.getInstanceState(instance) === 'updating'"
              color="info"
              floating
              transparent
              class="tw-w-2 tw-h-2 tw-rounded-full"
              style="top: 3px; right: 3px"
            />
            <q-badge
              v-if="taskStore.getInstanceState(instance) === 'failed'"
              color="negative"
              floating
              transparent
              class="tw-w-2 tw-h-2 tw-rounded-full"
              style="top: 3px; right: 3px"
            />
          </div>
        </div>
      </q-scroll-area>

      <div class="my-column tw-w-full tw-items-center">
        <q-btn
          :icon="taskStore.isSchedulerRunning ? 'stop' : 'play_arrow'"
          :disabled="taskStore.hasRunningInstance"
          color="white"
          text-color="primary"
          push
          round
          @click="toggleScheduler"
        />
        <q-btn
          icon="settings"
          color="white"
          text-color="primary"
          push
          round
          class="tw-mb-3"
          @click="showSettings = true"
        />
      </div>
    </q-drawer>

    <!-- Main content area -->
    <q-page-container>
      <div v-if="instances.length === 0" class="tw-h-screen flex flex-center">
        <WelcomePage />
      </div>
      <q-tab-panels
        v-else
        v-model="activeInstance"
        animated
        vertical
        class="tw-h-screen"
      >
        <q-tab-panel
          v-for="instance in instances"
          :key="instance"
          :name="instance"
          class="tw-p-2 tw-h-full"
        >
          <InstancePage :instance-name="instance" />
        </q-tab-panel>
      </q-tab-panels>
    </q-page-container>

    <!-- Settings dialog -->
    <SettingsPage v-model="showSettings" />
  </q-layout>
</template>

<script setup lang="ts">
// Import required components and stores
import { computed, ref, watch } from 'vue';
import { useDraggable } from 'vue-draggable-plus';
import { useIstStore, useSchedulerStore } from '../stores/global-store';
import { updateSchedulerState, updateInstanceOrder } from '../services/api';
import WelcomePage from './WelcomePage.vue';
import InstancePage from './InstancePage.vue';
import SettingsPage from './SettingsPage.vue';

// Initialize stores and reactive variables
const istStore = useIstStore();
const taskStore = useSchedulerStore();
const instances = computed(() => istStore.instanceNames);
const draggableInstances = ref<string[]>([]);
const activeInstance = ref<string>('');
const showSettings = ref(false);
const tabsEl = ref();

// Toggle scheduler state
const toggleScheduler = async () => {
  try {
    await updateSchedulerState(taskStore.isSchedulerRunning ? 'stop' : 'start');
  } catch (err) {
    console.error('Failed to toggle scheduler:', err);
  }
};

// Initialize drag and drop functionality
useDraggable(tabsEl, draggableInstances, {
  animation: 150,
  onEnd: async () => {
    try {
      await updateInstanceOrder(draggableInstances.value);
      await istStore.loadInstance();
    } catch (err) {
      console.error('Failed to update instance order:', err);
    }
  },
});

// Watch for instance list changes
watch(
  instances,
  (newInstances) => {
    draggableInstances.value = [...newInstances];
    const firstInstance = newInstances[0];
    if (firstInstance && !newInstances.includes(activeInstance.value)) {
      activeInstance.value = firstInstance;
    }
  },
  { immediate: true },
);
</script>
