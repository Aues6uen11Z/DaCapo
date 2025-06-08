<template>
  <q-dialog v-model="show" class="tw-w-full">
    <q-card class="tw-w-full tw-h-full">
      <q-card-section class="tw-h-[95%]">
        <q-tabs v-model="tab" class="tw-self-center">
          <q-tab name="instance" :label="t('settings.instance')" />
          <q-tab name="general" :label="t('settings.general')" />
          <q-tab name="about" :label="t('settings.about')" />
        </q-tabs>

        <q-tab-panels v-model="tab" animated class="tw-w-full tw-h-full">
          <!-- Instance Settings Panel -->
          <q-tab-panel name="instance">
            <!-- Create new instance -->
            <div class="my-column tw-gap-3 tw-h-full">
              <div class="tw-text-xl tw-font-bold">
                {{ t('settings.createNew') }}
              </div>
              <div class="my-row">
                <q-radio
                  v-model="instanceType"
                  val="local"
                  :label="t('settings.fromLocal')"
                />
                <q-radio
                  v-model="instanceType"
                  val="template"
                  :label="t('settings.fromTemplate')"
                />
                <q-radio
                  v-model="instanceType"
                  val="remote"
                  :label="t('settings.fromRemote')"
                />
              </div>

              <div class="my-row tw-w-full">
                <q-input
                  dense
                  outlined
                  v-model="instanceName"
                  class="tw-w-1/2"
                  :label="t('settings.instanceName')"
                />
                <q-input
                  v-if="instanceType !== 'template'"
                  dense
                  outlined
                  v-model="templateName"
                  class="tw-w-1/2"
                  :label="t('settings.templateName')"
                />

                <q-select
                  v-if="instanceType === 'template'"
                  dense
                  outlined
                  v-model="templateName"
                  class="tw-w-1/2"
                  :options="templateOptions"
                  :label="t('settings.templateName')"
                >
                  <template v-slot:option="scope">
                    <q-item v-bind="scope.itemProps">
                      <q-item-section
                        class="my-row tw-items-center tw-justify-between"
                      >
                        <q-item-label>{{ scope.opt }}</q-item-label>
                        <q-btn
                          flat
                          dense
                          icon="clear"
                          :disabled="workingTpl.includes(scope.opt)"
                          @click.stop="removeTemplate(scope.opt)"
                        >
                          <q-tooltip v-if="workingTpl.includes(scope.opt)">
                            {{ t('settings.cannotDeleteTemplate') }}
                          </q-tooltip>
                        </q-btn>
                      </q-item-section>
                    </q-item>
                  </template>
                </q-select>
              </div>

              <dir-input
                v-if="instanceType === 'local'"
                v-model="templatePath"
                outlined
                class="tw-w-full"
                :label="t('settings.templatePath')"
              />

              <q-input
                v-if="instanceType === 'remote'"
                dense
                outlined
                v-model="repoUrl"
                class="tw-w-full"
                :label="t('settings.repoUrl')"
              />
              <dir-input
                v-if="instanceType === 'remote'"
                v-model="localPath"
                outlined
                class="tw-w-full"
                :label="t('settings.localPath')"
              />
              <div v-if="instanceType === 'remote'" class="my-row tw-w-full">
                <q-input
                  dense
                  outlined
                  v-model="branch"
                  class="tw-w-1/2"
                  :label="t('settings.branch')"
                  :placeholder="t('settings.optional')"
                />
                <q-input
                  dense
                  outlined
                  v-model="templateRelPath"
                  class="tw-w-1/2"
                  :label="t('settings.templatePath')"
                  :placeholder="t('settings.relativePath')"
                />
              </div>
              <div class="tw-self-end">
                <q-btn
                  outline
                  color="primary"
                  :label="t('settings.create')"
                  no-caps
                  @click="createInstance"
                />
              </div>

              <!-- Manage instances -->
              <div class="tw-text-xl tw-font-bold tw--mt-8">
                {{ t('settings.manage') }}
              </div>

              <!-- Auto Action Settings -->
              <div class="my-row tw-w-full tw-justify-between tw--mb-4">
                <q-select
                  dense
                  v-model="autoActionTrigger"
                  :options="autoActionTriggerOptions"
                  option-value="value"
                  option-label="label"
                  emit-value
                  map-options
                  class="tw-w-1/3"
                />
                <q-input
                  dense
                  v-model="autoActionCron"
                  label="Cron"
                  :disable="autoActionTrigger !== 'scheduled'"
                  :rules="[
                    (val) => !val || validateCron(val) || t('item.cronHelp'),
                  ]"
                  @blur="handleAutoActionCronBlur"
                  class="tw-w-1/3"
                />
                <q-select
                  dense
                  v-model="autoActionType"
                  :options="autoActionTypeOptions"
                  :label="t('settings.autoAction')"
                  option-value="value"
                  option-label="label"
                  emit-value
                  map-options
                  class="tw-w-1/3"
                />
              </div>

              <div class="my-row tw-w-full tw-justify-between tw--mb-4">
                <q-toggle
                  v-model="selectAll"
                  :label="t('settings.selectAll')"
                  @update:model-value="toggleAll"
                />

                <q-checkbox
                  v-model="runOnStartup"
                  :label="t('settings.runOnStartup')"
                />

                <q-input
                  dense
                  v-model="schedulerCron"
                  label="Cron"
                  :disable="runOnStartup"
                  :rules="[(val) => validateCron(val) || t('item.cronHelp')]"
                  @blur="handleCronBlur"
                />
              </div>

              <q-scroll-area class="tw-w-full tw-h-2/5 tw-border">
                <div class="tw-grid tw-grid-cols-2 tw-gap-4">
                  <div v-for="instance in instances" :key="instance">
                    <div class="my-row tw-justify-between">
                      <q-toggle
                        v-model="readyInstances[instance]"
                        @update:model-value="
                          (value) => istStore.updateReady(value, instance)
                        "
                        :label="instance"
                      />
                      <q-btn
                        flat
                        icon="delete"
                        color="primary"
                        @click="istStore.removeInstance(instance)"
                      />
                    </div>
                  </div>
                </div>
              </q-scroll-area>
            </div>
          </q-tab-panel>

          <!-- General Settings Panel -->
          <q-tab-panel name="general">
            <div class="my-column tw-gap-4">
              <div
                class="tw-grid tw-grid-cols-2 tw-gap-4 tw-place-items-center tw-w-full"
              >
                <div class="tw-flex tw-items-center">
                  <q-icon name="g_translate" size="md" class="tw-mr-2" />
                  <span class="tw-text-xl tw-font-bold">{{
                    t('settings.language')
                  }}</span>
                </div>
                <q-select
                  dense
                  outlined
                  v-model="language"
                  :options="['zh-CN', 'en-US']"
                  @update:model-value="onLanguageChange"
                />
              </div>

              <div
                class="tw-grid tw-grid-cols-2 tw-gap-4 tw-place-items-center tw-w-full"
              >
                <div class="tw-flex tw-items-center">
                  <q-icon name="article" size="md" class="tw-mr-2" />
                  <span class="tw-text-xl tw-font-bold">{{
                    t('settings.log')
                  }}</span>
                </div>
                <q-btn
                  outline
                  color="primary"
                  :label="t('settings.openLog')"
                  no-caps
                  @click="openLog"
                />
              </div>
            </div>
          </q-tab-panel>
          <!-- About Panel -->
          <q-tab-panel name="about">
            <div class="tw-flex tw-flex-col tw-items-center tw-h-full">
              <img src="logo.png" />
              <div class="tw-text-4xl tw-font-bold tw-mt-2">DaCapo</div>
              <div class="tw-text-lg tw-mt-4">
                <p>
                  <strong>{{ t('settings.homepage') }}</strong
                  >:
                  <a
                    href="https://github.com/Aues6uen11Z/DaCapo"
                    target="_blank"
                    class="hover:tw-underline"
                  >
                    https://github.com/Aues6uen11Z/DaCapo
                  </a>
                </p>
                <p>
                  <strong>{{ t('settings.version') }}</strong
                  >: {{ version }}
                </p>
                <p>
                  <strong>{{ t('settings.license') }}</strong
                  >: GPL-3.0
                </p>
              </div>
              <!-- Update status section -->
              <div class="tw-mt-8 tw-p-4 tw-bg-gray-50 tw-rounded-lg">
                <div class="tw-text-center tw-mb-4">
                  <!-- Display current update status -->
                  <div
                    v-if="settingsStore.isUpToDate"
                    class="tw-flex tw-items-center tw-justify-center tw-text-green-600 tw-mb-2"
                  >
                    <q-icon name="check_circle" size="md" class="tw-mr-2" />
                    <span class="tw-text-base tw-font-medium">{{
                      t('settings.upToDate')
                    }}</span>
                  </div>
                  <div
                    v-else-if="settingsStore.isUpdateAvailable"
                    class="tw-flex tw-items-center tw-justify-center tw-text-orange-600 tw-mb-2"
                  >
                    <q-icon name="update" size="md" class="tw-mr-2" />
                    <span class="tw-text-base tw-font-medium">{{
                      t('settings.updateAvailable')
                    }}</span>
                  </div>
                  <div
                    v-else-if="settingsStore.hasUpdateError"
                    class="tw-flex tw-items-center tw-justify-center tw-text-red-600 tw-mb-2"
                  >
                    <q-icon name="error" size="md" class="tw-mr-2" />
                    <span class="tw-text-base">{{
                      settingsStore.updateStatusMessage
                    }}</span>
                  </div>
                  <div
                    v-else
                    class="tw-flex tw-items-center tw-justify-center tw-text-gray-500 tw-mb-2"
                  >
                    <q-icon name="help" size="md" class="tw-mr-2" />
                    <span class="tw-text-base">{{
                      t('settings.updateStatusUnknown')
                    }}</span>
                  </div>
                </div>

                <!-- Check for updates button or update action -->
                <div class="tw-flex tw-justify-center">
                  <q-btn
                    v-if="settingsStore.canCheckForUpdates"
                    outline
                    color="primary"
                    :label="t('settings.checkUpdates')"
                    @click="() => settingsStore.checkForAppUpdates(true)"
                    icon="system_update"
                    no-caps
                  />
                  <q-btn
                    v-else-if="settingsStore.isUpdateAvailable"
                    outline
                    color="positive"
                    :label="t('appUpdate.updateNow')"
                    @click="() => settingsStore.checkForAppUpdates(true)"
                    icon="download"
                    no-caps
                  />
                </div>
              </div>
            </div>
          </q-tab-panel>
        </q-tab-panels>
      </q-card-section>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useIstStore, useSettingsStore } from '../stores/global-store';
import { useQuasar } from 'quasar';
import DirInput from '../components/DirInput.vue';
import {
  createLocalInstance,
  createRemoteInstance,
  createTemplateInstance,
  fetchTemplates,
  deleteTemplate,
  getTaskQueue,
} from 'src/services/api';
import { useI18n } from 'vue-i18n';
import type { MessageLanguages } from 'src/boot/i18n';
import { GetVersion } from 'app/wailsjs/go/app/App';
import { OpenFileExplorer } from 'app/wailsjs/go/app/App';
import { validateCron } from '../utils/cron';

const $q = useQuasar();
const istStore = useIstStore();
const settingsStore = useSettingsStore();
const i18n = useI18n();
const { t } = i18n;

// Settings dialog properties and emits
const props = defineProps<{
  modelValue: boolean;
}>();
const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
}>();
const show = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
});
const tab = ref('instance');

// Instance creation form data
const instanceType = ref('local');
const instanceName = ref('');
const templateName = ref('');
const templatePath = ref('');
const repoUrl = ref('');
const branch = ref('');
const localPath = ref('');
const templateRelPath = ref('');
const templateOptions = ref<string[]>([]);
const workingTpl = computed(() => istStore.workingTemplate);

// Version number
const version = ref('');

onMounted(async () => {
  loadTemplates();
  try {
    version.value = await GetVersion();
  } catch (err) {
    console.error('Failed to get version:', err);
    version.value = '*';
  }
});

// Load available templates from backend
const loadTemplates = async () => {
  try {
    templateOptions.value = await fetchTemplates();
  } catch (err) {
    console.error('Failed to load templates:', err);
    $q.notify({
      type: 'negative',
      html: true,
      message: `Failed to load templates: <br> ${err instanceof Error ? err.message : 'Unknown error'}`,
    });
  }
};

// Delete a template from the system
const removeTemplate = async (templateName: string) => {
  try {
    await deleteTemplate(templateName);
    // 删除成功后更新模板列表
    templateOptions.value = templateOptions.value.filter(
      (t) => t !== templateName,
    );
  } catch (err) {
    console.error('Failed to delete template:', err);
    $q.notify({
      type: 'negative',
      html: true,
      message: `Failed to delete template: <br> ${err instanceof Error ? err.message : 'Unknown error'}`,
    });
  }
};

// Validate required fields
const check = (name: string, value: string) => {
  if (!value) {
    $q.notify({
      type: 'negative',
      message: `${name} is required`,
    });
    return false;
  }
  return true;
};

// Clear form data after instance creation
const clear = () => {
  instanceName.value = '';
  templateName.value = '';
  templatePath.value = '';
  repoUrl.value = '';
  branch.value = '';
  localPath.value = '';
  templateRelPath.value = '';
};

// Show notification during instance creation
const createNotify = () => {
  return $q.notify({
    group: false,
    timeout: 0,
    spinner: true,
    type: 'info',
    message: 'Creating new instance...',
  });
};

// Handle instance creation based on selected type
const createInstance = async () => {
  if (!check('Instance name', instanceName.value)) return;
  if (!check('Template name', templateName.value)) return;
  if (instances.value.includes(instanceName.value)) {
    $q.notify({
      type: 'negative',
      message: 'Instance name already exists',
    });
    return;
  }
  if (
    instanceType.value != 'template' &&
    templateOptions.value.includes(templateName.value)
  ) {
    $q.notify({
      type: 'negative',
      message: 'Template name already exists',
    });
    return;
  }

  let notify;
  try {
    switch (instanceType.value) {
      case 'local': {
        if (!check('Template path', templatePath.value)) return;
        show.value = false;
        notify = createNotify();
        await createLocalInstance({
          instance_name: instanceName.value,
          template_name: templateName.value,
          template_path: templatePath.value,
        });
        break;
      }
      case 'template': {
        show.value = false;
        notify = createNotify();
        await createTemplateInstance({
          instance_name: instanceName.value,
          template_name: templateName.value,
        });
        break;
      }
      case 'remote': {
        if (
          !check('Repository URL', repoUrl.value) ||
          !check('Local path', localPath.value) ||
          !check('Template path', templateRelPath.value)
        )
          return;
        show.value = false;
        notify = createNotify();
        await createRemoteInstance({
          instance_name: instanceName.value,
          template_name: templateName.value,
          url: repoUrl.value,
          local_path: localPath.value,
          template_rel_path: templateRelPath.value,
          branch: branch.value,
        });
        break;
      }
    }

    await istStore.loadInstance(instanceName.value);
    if (!templateOptions.value.includes(templateName.value)) {
      templateOptions.value.push(templateName.value);
    }
    await getTaskQueue(instanceName.value);
    clear();
    notify?.({
      spinner: false,
      type: 'positive',
      message: 'Instance created successfully',
      timeout: 3000,
    });
  } catch (err) {
    console.error('Failed to create instance:', err);
    notify?.({
      spinner: false,
      type: 'negative',
      html: true,
      message: `Failed to create instance: <br> ${err instanceof Error ? err.message : 'Unknown error'}`,
      timeout: 3000,
    });
  }
};

// Auto action settings
const autoActionTrigger = computed({
  get: () => settingsStore.autoActionTrigger,
  set: (value) => settingsStore.setAutoActionTrigger(value),
});

const autoActionType = computed({
  get: () => settingsStore.autoActionType,
  set: (value) => settingsStore.setAutoActionType(value),
});

const autoActionTriggerOptions = computed(() => [
  { label: t('settings.schedulerEnd'), value: 'scheduler_end' },
  { label: t('settings.scheduled'), value: 'scheduled' },
]);

const autoActionTypeOptions = computed(() => [
  { label: t('settings.noAction'), value: 'none' },
  { label: t('settings.closeApp'), value: 'close_app' },
  { label: t('settings.hibernate'), value: 'hibernate' },
  { label: t('settings.shutdown'), value: 'shutdown' },
]);

const autoActionCron = ref(settingsStore.autoActionCron);
const handleAutoActionCronBlur = () => {
  if (validateCron(autoActionCron.value)) {
    settingsStore.setAutoActionCron(autoActionCron.value);
  }
};

const runOnStartup = computed({
  get: () => settingsStore.runOnStartup,
  set: (value) => settingsStore.setRunOnStartup(value),
});

const schedulerCron = ref(settingsStore.schedulerCron);
const handleCronBlur = () => {
  if (validateCron(schedulerCron.value)) {
    settingsStore.setSchedulerCron(schedulerCron.value);
  }
};

// Instance management
const instances = computed(() => istStore.instanceNames);
const selectAll = ref(false);
const readyInstances = computed(() => istStore.ready);

// Toggle all instances' ready state
const toggleAll = async () => {
  await istStore.updateReady(selectAll.value);
};

// General settings
const language = ref(settingsStore.language);
const onLanguageChange = (newLang: MessageLanguages) => {
  settingsStore.setLanguage(newLang, i18n);
};
const openLog = async () => {
  await OpenFileExplorer('.', 'logs');
};
</script>
