<template>
  <q-scroll-area
    ref="scrollArea"
    :thumb-style="thumbStyle"
    content-style="max-width: 100%;"
    content-active-style="max-width: 100%;"
    class="log-viewer tw-h-full tw-font-light tw-font-mono tw-text-sm tw-bg-gray-100 tw-px-4"
  >
    <div
      v-for="(log, index) in highlightedLogs"
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
import { highlightLog } from '../utils/logHighlighter';

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

// Apply syntax highlighting to logs
const highlightedLogs = computed(() =>
  logs.value.map((log) => highlightLog(log)),
);

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

// Watch highlighted logs and apply smart scroll behavior
watch(highlightedLogs, () => {
  smartScroll();
});
</script>

<style scoped>
/* Log syntax highlighting styles */
.log-viewer :deep(.log-error) {
  color: #dc2626;
}
.log-viewer :deep(.log-error-type) {
  color: #dc2626;
  font-weight: 600;
}
.log-viewer :deep(.log-warning) {
  color: #d97706;
}
.log-viewer :deep(.log-info) {
  color: #059669;
}
.log-viewer :deep(.log-debug) {
  color: #6b7280;
}
.log-viewer :deep(.log-success) {
  color: #16a34a;
}
.log-viewer :deep(.log-time) {
  color: #0891b2;
}
.log-viewer :deep(.log-path) {
  color: #0891b2;
}
.log-viewer :deep(.log-url) {
  color: #2563eb;
  text-decoration: underline;
}
.log-viewer :deep(.log-number) {
  color: #0284c7;
}
.log-viewer :deep(.log-string) {
  color: #16a34a;
}
.log-viewer :deep(.log-bool-true) {
  color: #16a34a;
  font-style: italic;
}
.log-viewer :deep(.log-bool-false) {
  color: #dc2626;
  font-style: italic;
}
.log-viewer :deep(.log-none) {
  color: #9333ea;
  font-style: italic;
}
.log-viewer :deep(.log-bracket) {
  font-weight: 600;
}
.log-viewer :deep(.log-muted) {
  color: #6b7280;
}
</style>
