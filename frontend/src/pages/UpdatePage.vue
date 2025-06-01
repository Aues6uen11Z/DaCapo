<template>
  <q-scroll-area
    content-style="gap:1rem; padding:0.125rem; align-items:center; display:flex; flex-direction:column"
    content-active-style="gap:1rem; padding:0.125rem; align-items:center; display:flex; flex-direction:column"
    class="tw-h-full"
  >
    <q-card style="width: 90%" class="tw-mr-8">
      <q-card-section>
        <card-title :name="t('update.title')">
          <q-btn
            icon="update"
            :label="updateBtnLabel"
            outline
            no-caps
            text-color="primary"
            :loading="taskStore.isInstanceUpdating(istName)"
            :disable="taskStore.isInstanceUpdated(istName)"
            @click="handleUpdate"
          />
        </card-title>
        <div class="my-column tw-gap-0 tw-grid tw-grid-cols-3 tw-w-full">
          <item-line
            :ist-name="istName"
            menu-name="Project"
            task-name="Update"
            group-name="_Base"
            item-name="Repository URL"
            :item-conf="updateGroup.repo_url!"
            :display-name="t('update.repoUrl')"
            :help="t('update.help.repoUrl')"
          />
          <item-line
            :ist-name="istName"
            menu-name="Project"
            task-name="Update"
            group-name="_Base"
            item-name="branch"
            :item-conf="updateGroup.branch!"
            :display-name="t('update.branch')"
            :help="t('update.help.branch')"
          />
          <item-line
            :ist-name="istName"
            menu-name="Project"
            task-name="Update"
            group-name="_Base"
            item-name="local_path"
            :item-conf="updateGroup.local_path!"
            :display-name="t('update.localPath')"
            :help="t('update.help.localPath')"
          />
          <item-line
            :ist-name="istName"
            menu-name="Project"
            task-name="Update"
            group-name="_Base"
            item-name="template_rel_path"
            :item-conf="updateGroup.template_rel_path!"
            :display-name="t('update.templatePath')"
            :help="t('update.help.templatePath')"
          />
          <item-line
            :ist-name="istName"
            menu-name="Project"
            task-name="Update"
            group-name="_Base"
            item-name="auto_update"
            :item-conf="updateGroup.auto_update!"
            :display-name="t('update.autoUpdate')"
            :help="t('update.help.autoUpdate')"
          />
        </div>
        <q-expansion-item
          icon="fa-brands fa-python"
          :label="t('update.advancedSettings')"
          dense
          :default-opened="isPythonConfigured"
          class="tw-w-[calc(100%+32px)] tw--ml-4 tw-pt-4"
        >
          <div
            class="my-column tw-gap-0 tw-grid tw-grid-cols-3 tw-w-full tw-px-4"
          >
            <item-line
              :ist-name="istName"
              menu-name="Project"
              task-name="Update"
              group-name="_Base"
              item-name="env_name"
              :item-conf="updateGroup.env_name!"
              :display-name="t('update.envName')"
              :help="t('update.help.envName')"
            />
            <item-line
              :ist-name="istName"
              menu-name="Project"
              task-name="Update"
              group-name="_Base"
              item-name="deps_path"
              :item-conf="updateGroup.deps_path!"
              :display-name="t('update.depsPath')"
              :help="t('update.help.depsPath')"
            />
            <item-line
              :ist-name="istName"
              menu-name="Project"
              task-name="Update"
              group-name="_Base"
              item-name="python_version"
              :item-conf="updateGroup.python_version!"
              :display-name="t('update.pythonVersion')"
              :help="t('update.help.pythonVersion')"
            />
          </div>
        </q-expansion-item>
      </q-card-section>
    </q-card>
  </q-scroll-area>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useIstStore, useSchedulerStore } from '../stores/global-store';
import { updateRepo } from '../services/api';
import CardTitle from '../components/CardTitle.vue';
import ItemLine from '../components/ItemLine.vue';
import { useQuasar } from 'quasar';
import { useI18n } from 'vue-i18n';

const props = defineProps<{
  istName: string;
}>();

const { t } = useI18n();
const istStore = useIstStore();
const taskStore = useSchedulerStore();
const updateGroup = computed(() => {
  return istStore.layout[props.istName]?.Project?.Update?._Base || {};
});

// Check if Python environment is configured
const isPythonConfigured = computed(() =>
  Boolean(updateGroup.value.env_name?.value),
);

const $q = useQuasar();

// Computed button label based on update status
const updateBtnLabel = computed(() => {
  if (taskStore.isInstanceUpdated(props.istName)) return t('update.upToDate');
  return t('update.checkUpdates');
});

// Handle repository update process
const handleUpdate = async () => {
  if (
    taskStore.isInstanceUpdating(props.istName) ||
    taskStore.isInstanceUpdated(props.istName)
  )
    return;

  taskStore.setInstanceUpdating(props.istName, true);
  try {
    const response = await updateRepo(props.istName);

    // Update successful
    taskStore.setInstanceUpdated(props.istName, true);

    // If there's a new layout, update the interface
    if (response) {
      await istStore.loadInstance(props.istName);
    }
  } catch (err) {
    console.error('Failed to update repository:', err);
    $q.notify({
      type: 'negative',
      html: true,
      message: `Failed to update:<br> ${err instanceof Error ? err.message : 'Unknown error'}`,
    });
    // Reset status to allow retry on failure
    taskStore.setInstanceUpdated(props.istName, false);
  } finally {
    taskStore.setInstanceUpdating(props.istName, false);
  }
};
</script>
