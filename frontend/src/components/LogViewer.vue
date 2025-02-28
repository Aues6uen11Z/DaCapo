<template>
  <q-scroll-area
    ref="scrollArea"
    :thumb-style="thumbStyle"
    content-style="max-width: 100%;"
    content-active-style="max-width: 100%;"
    class="tw-h-full tw-font-light tw-font-mono tw-text-sm tw-bg-gray-100 tw-px-4"
  >
    <div
      v-for="(log, index) in parsedLogs"
      :key="index"
      class="tw-whitespace-pre-wrap"
      v-html="log"
    />
  </q-scroll-area>
</template>

<script setup lang="ts">
import { ref, watch, computed, nextTick, onMounted } from 'vue';
import { useSchedulerStore } from '../stores/global-store';
import { QScrollArea } from 'quasar';
import Convert from 'ansi-to-html';

const props = defineProps<{
  instanceName: string;
}>();

const thumbStyle = {
  borderRadius: '6px',
  width: '6px',
};

const taskStore = useSchedulerStore();
const scrollArea = ref<InstanceType<typeof QScrollArea> | null>(null);
const logs = ref<string[]>([]);

// Create ANSI converter instance
const convert = new Convert({
  fg: '#000',
  bg: '#fff',
  newline: false,
  escapeXML: true,
  stream: false,
  colors: {
    0: '#000000', // Black
    1: '#cd3131', // Red
    2: '#0dbc79', // Green
    3: '#e5e510', // Yellow
    4: '#2472c8', // Blue
    5: '#bc3fbc', // Magenta
    6: '#11a8cd', // Cyan
    7: '#e5e5e5', // White
    8: '#666666', // Bright Black (Gray)
    9: '#f14c4c', // Bright Red
    10: '#23d18b', // Bright Green
    11: '#f5f543', // Bright Yellow
    12: '#3b8eea', // Bright Blue
    13: '#d670d6', // Bright Magenta
    14: '#29b8db', // Bright Cyan
    15: '#ffffff', // Bright White
  },
});

// Convert logs to HTML format
const parsedLogs = computed(() => logs.value.map((log) => convert.toHtml(log)));

// Scroll to the bottom of the log viewer
const scrollToBottom = () => {
  nextTick(() => {
    const scroll = scrollArea.value;
    if (!scroll) return;

    const scrollTarget = scroll.getScrollTarget();
    scrollTarget.scrollTop = scrollTarget.scrollHeight;
  });
};

// Smart scroll implementation:
// Only auto-scrolls if the user is already near the bottom
const smartScroll = () => {
  nextTick(() => {
    const scroll = scrollArea.value;
    if (!scroll) return;

    const scrollTarget = scroll.getScrollTarget();
    const threshold = 100;
    const isNearBottom =
      scrollTarget.scrollTop + scrollTarget.clientHeight >=
      scrollTarget.scrollHeight - threshold;
    // Only auto-scroll if near the bottom
    if (isNearBottom) {
      scrollTarget.scrollTop = scrollTarget.scrollHeight;
    }
  });
};

// Scroll to bottom when component is mounted
onMounted(() => {
  scrollToBottom();
});

// Watch for changes in the logs for this instance
watch(
  () => taskStore.logs[props.instanceName],
  (newLogs) => {
    if (!newLogs) return;
    logs.value = newLogs;
  },
  { immediate: true },
);

// Watch parsed logs and apply smart scroll behavior
watch(parsedLogs, () => {
  smartScroll();
});
</script>
