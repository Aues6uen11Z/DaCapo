<template>
  <div class="my-row tw-h-full">
    <div class="my-column tw-w-80 tw-h-full">
      <q-card class="tw-w-full tw-h-36">
        <q-card-section class="tw-h-full">
          <card-title :name="t('home.running')" icon="cached">
            <q-btn
              :icon="isRunning ? 'stop' : 'play_arrow'"
              flat
              round
              text-color="primary"
              @click="toggleInstance"
            />
          </card-title>

          <div v-if="currentQueue?.running">
            <task-item
              :instance-name="instanceName"
              :task-name="currentQueue.running"
              type="running"
            />
          </div>
        </q-card-section>
      </q-card>

      <q-card class="tw-w-full tw-h-2/5">
        <q-card-section class="tw-h-full tw-flex tw-flex-col">
          <card-title :name="t('home.waiting')" icon="hourglass_top">
            <q-btn
              icon="south"
              flat
              round
              text-color="primary"
              @click="moveAllToStopped"
            />
          </card-title>

          <q-scroll-area class="tw-grow">
            <div ref="waitingEl" class="my-column tw-gap-1">
              <task-item
                v-for="taskName in localWaitingTasks"
                :key="taskName"
                :instance-name="instanceName"
                :task-name="taskName"
                type="waiting"
              />
            </div>
          </q-scroll-area>
        </q-card-section>
      </q-card>

      <q-card class="tw-w-full tw-grow">
        <q-card-section class="tw-h-full tw-flex tw-flex-col">
          <card-title :name="t('home.stopped')" icon="block">
            <q-btn
              icon="north"
              flat
              round
              text-color="primary"
              @click="moveAllToWaiting"
            />
          </card-title>

          <q-scroll-area class="tw-grow">
            <div ref="stoppedEl" class="my-column tw-gap-1">
              <task-item
                v-for="taskName in localStoppedTasks"
                :key="taskName"
                :instance-name="instanceName"
                :task-name="taskName"
                type="stopped"
              />
            </div>
          </q-scroll-area>
        </q-card-section>
      </q-card>
    </div>

    <q-card class="tw-w-full tw-h-full">
      <q-card-section class="tw-h-full tw-flex tw-flex-col">
        <card-title :name="t('home.logs')" icon="description">
          <q-btn
              icon="open_in_new"
              flat
              round
              text-color="primary"
              @click="openLog"
            />
        </card-title>

        <log-viewer :instance-name="instanceName" />
      </q-card-section>
    </q-card>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useIstStore, useSchedulerStore } from '../stores/global-store';
import { updateSchedulerState } from '../services/api';
import { useDraggable } from 'vue-draggable-plus';
import { useI18n } from 'vue-i18n';
import CardTitle from '../components/CardTitle.vue';
import TaskItem from '../components/TaskItem.vue';
import LogViewer from '../components/LogViewer.vue';
import { OpenFileExplorer } from 'app/wailsjs/go/app/App';

const props = defineProps<{
  instanceName: string;
}>();

const { t } = useI18n();
const taskStore = useSchedulerStore();
const waitingEl = ref();
const stoppedEl = ref();

const isRunning = computed(() =>
  taskStore.isInstanceRunning(props.instanceName),
);
const currentQueue = computed(() => taskStore.queues[props.instanceName]);
const waitingTasks = computed(() => currentQueue.value?.waiting || []);
const stoppedTasks = computed(() => currentQueue.value?.stopped || []);

// Local data for drag and drop functionality
const localWaitingTasks = ref<string[]>([]);
const localStoppedTasks = ref<string[]>([]);

// Watch for queue changes and update local data
watch(
  waitingTasks,
  (newVal) => {
    localWaitingTasks.value = [...newVal];
  },
  { immediate: true },
);

watch(
  stoppedTasks,
  (newVal) => {
    localStoppedTasks.value = [...newVal];
  },
  { immediate: true },
);

// Initialize drag and drop functionality
useDraggable(waitingEl, localWaitingTasks, {
  animation: 150,
  group: 'tasks',
  onEnd: () => {
    const queue = taskStore.queues[props.instanceName];
    if (!queue) return;

    taskStore.updateQueue({
      [props.instanceName]: {
        ...queue,
        waiting: localWaitingTasks.value,
        stopped: localStoppedTasks.value,
      },
    });
  },
});

useDraggable(stoppedEl, localStoppedTasks, {
  animation: 150,
  group: 'tasks',
  onEnd: () => {
    const queue = taskStore.queues[props.instanceName];
    if (!queue) return;

    taskStore.updateQueue({
      [props.instanceName]: {
        ...queue,
        waiting: localWaitingTasks.value,
        stopped: localStoppedTasks.value,
      },
    });
  },
});

const toggleInstance = async () => {
  try {
    await updateSchedulerState(
      isRunning.value ? 'stop' : 'start',
      props.instanceName,
    );
  } catch (err) {
    console.error('Failed to toggle instance:', err);
  }
};

const moveAllToWaiting = () => {
  const queue = taskStore.queues[props.instanceName];
  if (!queue) return;
  taskStore.updateQueue({
    [props.instanceName]: {
      ...queue,
      waiting: [...queue.waiting, ...queue.stopped],
      stopped: [],
    },
  });
};

const moveAllToStopped = () => {
  const queue = taskStore.queues[props.instanceName];
  if (!queue) return;
  taskStore.updateQueue({
    [props.instanceName]: {
      ...queue,
      waiting: [],
      stopped: [...queue.stopped, ...queue.waiting],
    },
  });
};

const istStore = useIstStore();
const workDir = computed(() => {
  return istStore.layout[props.instanceName]?.Project?.General?._Base?.work_dir?.value as string || '';
});
const logPath = computed(() => {
  return istStore.layout[props.instanceName]?.Project?.General?._Base?.log_path?.value as string || '';
});
const openLog = async () => {
  await OpenFileExplorer(workDir.value, logPath.value)
}
</script>
