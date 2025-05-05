<template>
  <q-scroll-area
    content-style="gap:1rem; padding:0.125rem; align-items:center; display:flex; flex-direction:column"
    content-active-style="gap:1rem; padding:0.125rem; align-items:center; display:flex; flex-direction:column"
    class="tw-h-full"
  >
    <!-- Display _Base group first -->
    <q-card v-if="baseGroup" style="width: 90%" class="tw-mr-8">
      <q-card-section>
        <card-title :name="t('general.title')" />
        <div class="my-column tw-gap-0 tw-grid tw-grid-cols-3 tw-w-full">
          <item-line
            :ist-name="istName"
            menu-name="Project"
            task-name="General"
            group-name="_Base"
            item-name="language"
            :item-conf="baseGroup.language!"
            :display-name="t('general.language')"
            :help="t('general.help.language')"
          />
          <item-line
            :ist-name="istName"
            menu-name="Project"
            task-name="General"
            group-name="_Base"
            item-name="work_dir"
            :item-conf="baseGroup.work_dir!"
            :display-name="t('general.workDir')"
            :help="t('general.help.workDir')"
          />
          <item-line
            :ist-name="istName"
            menu-name="Project"
            task-name="General"
            group-name="_Base"
            item-name="background"
            :item-conf="baseGroup.background!"
            :display-name="t('general.background')"
            :help="t('general.help.background')"
          />
          <item-line
            :ist-name="istName"
            menu-name="Project"
            task-name="General"
            group-name="_Base"
            item-name="config_path"
            :item-conf="baseGroup.config_path!"
            :display-name="t('general.configPath')"
            :help="t('general.help.configPath')"
          />
          <item-line
            :ist-name="istName"
            menu-name="Project"
            task-name="General"
            group-name="_Base"
            item-name="log_path"
            :item-conf="baseGroup.log_path!"
            :display-name="t('general.logPath')"
            :help="t('general.help.logPath')"
          />
        </div>
      </q-card-section>
    </q-card>

    <!-- Display other groups -->
    <group-card
      v-for="(group, groupName) in customGroups"
      :key="groupName"
      :ist-name="istName"
      menu-name="Project"
      task-name="General"
      :group-name="String(groupName)"
      :group-conf="group"
    />
  </q-scroll-area>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useIstStore } from '../stores/global-store';
import GroupCard from '../components/GroupCard.vue';
import CardTitle from '../components/CardTitle.vue';
import ItemLine from '../components/ItemLine.vue';
import type { Task, Group } from '../services/response';
import { useI18n } from 'vue-i18n';

const props = defineProps<{
  istName: string;
}>();

const { t } = useI18n();
const istStore = useIstStore();
const task = computed<Task>(() => {
  return istStore.layout[props.istName]?.Project?.General || {};
});

// 获取 _Base 组
// Get the _Base group
const baseGroup = computed<Group | undefined>(() => {
  return task.value['_Base'];
});

// 获取除 _Base 外的其他组
// Get all groups except _Base
const customGroups = computed(() => {
  return Object.fromEntries(
    Object.entries(task.value).filter(([key]) => key !== '_Base'),
  );
});
</script>
