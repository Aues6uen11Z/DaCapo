<template>
  <div class="my-row tw-h-full tw-px-4">
    <!-- Left sidebar with instance navigation -->
    <q-card class="tw-w-48">
      <q-card-section class="tw-p-0">
        <q-tabs v-model="taskTab" vertical dense>
          <q-tab name="Home" icon="home" :label="t('tab.home')" />
          <template
            v-for="(menu, menuName) in istStore.layout[instanceName]"
            :key="menuName"
          >
            <q-expansion-item
              :label="
                menuName === 'Project'
                  ? t('tab.project')
                  : getMenuName(String(menuName))
              "
              dense
              expand-separator
              default-opened
              class="tw-font-medium tw-uppercase"
            >
              <div class="my-colmun tw-gap-0 tw-px-4">
                <template v-if="menuName === 'Project'">
                  <q-tab
                    no-caps
                    name="General"
                    :label="t('tab.general')"
                    class="tw-justify-start"
                  />
                  <q-tab
                    v-if="hasUpdateConfig"
                    no-caps
                    name="Update"
                    :label="t('tab.update')"
                    class="tw-justify-start"
                  />
                </template>
                <template v-else>
                  <template v-for="(_, taskName) in menu" :key="taskName">
                    <q-tab
                      no-caps
                      :name="String(taskName)"
                      :label="getTaskName(String(menuName), String(taskName))"
                      class="tw-justify-start"
                    />
                  </template>
                </template>
              </div>
            </q-expansion-item>
          </template>
        </q-tabs>
      </q-card-section>
    </q-card>

    <!-- Main content area with tab panels -->
    <q-tab-panels
      v-model="taskTab"
      animated
      vertical
      class="tw-w-full tw-h-full"
    >
      <q-tab-panel name="Home" class="tw-p-0.5">
        <home-page :instance-name="instanceName" />
      </q-tab-panel>
      <q-tab-panel name="General" class="tw-p-0.5">
        <general-page :ist-name="instanceName" />
      </q-tab-panel>
      <q-tab-panel v-if="hasUpdateConfig" name="Update" class="tw-p-0.5">
        <update-page :ist-name="instanceName" />
      </q-tab-panel>
      <template
        v-for="(menu, menuName) in istStore.layout[instanceName]"
        :key="menuName"
      >
        <template v-if="menuName !== 'Project'">
          <q-tab-panel
            v-for="(task, taskName) in menu"
            :key="taskName"
            :name="String(taskName)"
            class="tw-p-0.5"
          >
            <custom-page
              :ist-name="instanceName"
              :menu-name="String(menuName)"
              :task-name="String(taskName)"
              :task-conf="task"
            />
          </q-tab-panel>
        </template>
      </template>
    </q-tab-panels>
  </div>
</template>

<script setup lang="ts">
// Import required components and stores
import { useTaskTabStore, useIstStore } from '../stores/global-store';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import HomePage from './HomePage.vue';
import GeneralPage from './GeneralPage.vue';
import UpdatePage from './UpdatePage.vue';
import CustomPage from './CustomPage.vue';
import { useTranslation } from '../i18n/index';

// Props definition
const props = defineProps<{
  instanceName: string;
}>();

// Initialize i18n and stores
const { t } = useI18n();
const istStore = useIstStore();
const taskTabStore = useTaskTabStore();
const { getMenuName, getTaskName } = useTranslation(props.instanceName);

// Computed property for task tab state management
const taskTab = computed({
  get: () => taskTabStore.getTab(props.instanceName),
  set: (value: string) => {
    taskTabStore.setTab(props.instanceName, value);
  },
});

// Check if instance has update configuration available
const hasUpdateConfig = computed(() => {
  return !!istStore.layout[props.instanceName]?.Project?.Update;
});
</script>
