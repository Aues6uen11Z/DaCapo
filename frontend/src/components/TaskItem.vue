<template>
  <div class="my-row tw-w-full tw-gap-0 tw-border tw-border-black tw-rounded">
    <q-btn
      no-caps
      flat
      align="left"
      class="tw-self-stretch tw-grow"
      :label="translatedTaskName"
      @click="toggleTask"
    />
    <q-btn
      icon="settings"
      flat
      round
      text-color="primary"
      class="tw-self-center"
      @click="navigateToTask"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import {
  useSchedulerStore,
  useTaskTabStore,
  useIstStore,
} from '../stores/global-store';
import { useTranslation } from '../i18n/index';

const props = defineProps<{
  instanceName: string;
  taskName: string;
  type: 'waiting' | 'stopped' | 'running';
}>();

const taskStore = useSchedulerStore();
const taskTabStore = useTaskTabStore();
const istStore = useIstStore();
const { getTaskName } = useTranslation(props.instanceName);

// Get translated task name
const translatedTaskName = computed(() => {
  // Search through Layout to find the menu containing the task
  const layout = istStore.layout[props.instanceName];
  if (!layout) return props.taskName;

  // Find the menu name that contains this task
  const menuName = Object.entries(layout).find(
    ([name, menu]) => name !== 'Project' && props.taskName in menu,
  )?.[0];

  return menuName ? getTaskName(menuName, props.taskName) : props.taskName;
});

const toggleTask = () => {
  if (props.type === 'running') return;
  const targetType = props.type === 'waiting' ? 'stopped' : 'waiting';
  const queue = taskStore.queues[props.instanceName];
  if (!queue) return;

  // Update using original task name
  const updatedQueue = {
    ...queue,
    [props.type]: queue[props.type].filter((name) => name !== props.taskName),
    [targetType]: [...queue[targetType], props.taskName],
  };

  taskStore.updateQueue({ [props.instanceName]: updatedQueue });
};

const navigateToTask = () => {
  // Navigate using original task name
  taskTabStore.setTab(props.instanceName, props.taskName);
};
</script>
