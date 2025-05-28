<template>
  <template v-if="!itemConf.hidden">
    <div class="tw-col-span-2 tw-text-lg tw-content-center tw-h-full">
      {{ displayLabel }}
    </div>

    <!-- Input component based on type -->
    <template v-if="itemConf.type === 'select'">
      <q-select
        v-model="selectValue"
        :options="translatedOptions"
        :disable="itemConf.disabled"
        dense
        option-value="value"
        option-label="label"
        @blur="update(selectValue)"
      />
    </template>

    <template v-else-if="itemConf.type === 'checkbox'">
      <q-checkbox
        v-model="checkboxValue"
        :disable="itemConf.disabled"
        class="tw-justify-center tw-h-full"
        @update:model-value="update(checkboxValue)"
      />
    </template>

    <template v-else-if="itemConf.type === 'priority'">
      <q-input
        v-model.number="priorityValue"
        type="number"
        :disable="itemConf.disabled"
        :min="0"
        :max="31"
        dense
        @blur="update(priorityValue)"
      />
    </template>

    <template v-else-if="itemConf.type === 'folder'">
      <dir-input
        v-model="dirValue"
        :disabled="Boolean(itemConf.disabled)"
        @blur="update(dirValue)"
      />
    </template>

    <template v-else-if="itemConf.type === 'file'">
      <file-input
        v-model="fileValue"
        :disabled="Boolean(itemConf.disabled)"
        @blur="update(fileValue)"
      />
    </template>

    <template v-else-if="itemConf.type === 'cron'">
      <q-input
        v-model="inputValue"
        :disable="itemConf.disabled"
        dense
        :rules="[(val) => validateCron(val) || t('item.cronHelp')]"
        @blur="handleCronBlur"
      />
    </template>

    <template v-else>
      <q-input
        v-model="inputValue"
        :disable="itemConf.disabled"
        dense
        @blur="update(inputValue)"
      />
    </template>

    <!-- Help text -->
    <template v-if="helpText">
      <div class="tw-col-span-2 tw-text-gray-500 tw-whitespace-pre-wrap">
        {{ helpText }}
      </div>

      <q-space />
    </template>
  </template>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { Item } from '../services/response';
import DirInput from './DirInput.vue';
import FileInput from './FileInput.vue';
import { useIstStore } from '../stores/global-store';
import { useQuasar } from 'quasar';
import { useTranslation } from '../i18n/index';
import { useI18n } from 'vue-i18n';
import { validateCron } from '../utils/cron';

const { t } = useI18n();

const props = defineProps<{
  istName: string;
  menuName: string;
  taskName: string;
  groupName: string;
  itemName: string;

  itemConf: Item;

  displayName?: string;
  help?: string;
}>();

const { getItemOption } = useTranslation(props.istName);

const displayLabel = computed(() => props.displayName || props.itemName);
const helpText = computed(() => props.help || props.itemConf.help);

// Store original value for comparison
const originalValue = ref(props.itemConf.value);

// Store the raw value separately
const rawSelectValue = ref(
  props.itemConf.type === 'select' ? props.itemConf.value : null,
);

// Use computed for selectValue to automatically update with translations
const selectValue = computed({
  get() {
    if (props.itemConf.type !== 'select' || rawSelectValue.value === null) {
      return null;
    }
    // Return the matching translated option
    return (
      translatedOptions.value.find(
        (opt) => opt.value === rawSelectValue.value,
      ) || rawSelectValue.value
    );
  },
  set(newValue) {
    // Extract the original value when setting
    rawSelectValue.value =
      typeof newValue === 'object' && newValue !== null
        ? (newValue as { value: unknown }).value
        : newValue;
  },
});

const checkboxValue = ref(
  props.itemConf.type === 'checkbox' ? Boolean(props.itemConf.value) : false,
);
const priorityValue = ref(
  props.itemConf.type === 'priority' ? Number(props.itemConf.value) : 0,
);
const dirValue = ref(String(props.itemConf.value));
const fileValue = ref(String(props.itemConf.value));
const inputValue = ref(String(props.itemConf.value));

const istStore = useIstStore();
const $q = useQuasar();

// Translate options for select component
const translatedOptions = computed(() => {
  if (!Array.isArray(props.itemConf.option)) return [];

  return props.itemConf.option.map((opt) => ({
    value: opt, // Keep original value
    label: getItemOption(
      props.menuName,
      props.taskName,
      props.groupName,
      props.itemName,
      String(opt),
    ),
  }));
});

// Compare if value has changed
const hasValueChanged = (newValue: unknown) => {
  if (props.itemConf.type === 'checkbox') {
    return Boolean(originalValue.value) !== Boolean(newValue);
  }
  if (props.itemConf.type === 'priority') {
    return Number(originalValue.value) !== Number(newValue);
  }
  return String(originalValue.value) !== String(newValue);
};

const update = async (value: unknown) => {
  // For select type, ensure using the original value
  const actualValue =
    props.itemConf.type === 'select' && typeof value === 'object'
      ? (value as { value: unknown }).value
      : value;

  if (!hasValueChanged(actualValue)) {
    return;
  }

  try {
    await istStore.updateInstance(
      props.istName,
      props.menuName,
      props.taskName,
      props.groupName,
      props.itemName,
      actualValue,
    );
    originalValue.value = actualValue;

    // Update raw value for select type
    if (props.itemConf.type === 'select') {
      rawSelectValue.value = actualValue;
    }

    // If language setting changes, Vue will automatically recompute all translation-dependent computed properties
    if (
      props.menuName === 'Project' &&
      props.taskName === 'General' &&
      props.itemName === 'language'
    ) {
      // No additional operations needed as Vue will handle the reactivity
    }
  } catch (err) {
    console.error(`Failed to update ${props.itemName}:`, err);
    $q.notify({
      type: 'negative',
      html: true,
      message: `Failed to update ${props.itemName}:<br> ${err instanceof Error ? err.message : 'Unknown error'}`,
    });
  }
};

const handleCronBlur = () => {
  if (validateCron(inputValue.value)) {
    update(inputValue.value);
  }
};
</script>
