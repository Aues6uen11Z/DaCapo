<template>
  <q-scroll-area
    content-style="gap:1rem; padding:0.125rem; align-items:center; display:flex; flex-direction:column"
    content-active-style="gap:1rem; padding:0.125rem; align-items:center; display:flex; flex-direction:column"
    class="tw-h-full"
  >
    <!-- Display _Base group first -->
    <q-card v-if="baseGroup" style="width: 90%" class="tw-mr-8">
      <q-card-section>
        <card-title :name="t('custom.title')" />
        <div class="my-column tw-gap-0 tw-grid tw-grid-cols-3 tw-w-full">
          <item-line
            :ist-name="istName"
            :menu-name="menuName"
            :task-name="taskName"
            group-name="_Base"
            item-name="active"
            :item-conf="baseGroup.active!"
            :display-name="t('custom.active')"
            :help="t('custom.help.active')"
          />
          <item-line
            :ist-name="istName"
            :menu-name="menuName"
            :task-name="taskName"
            group-name="_Base"
            item-name="priority"
            :item-conf="baseGroup.priority!"
            :display-name="t('custom.priority')"
            :help="t('custom.help.priority')"
          />
          <item-line
            :ist-name="istName"
            :menu-name="menuName"
            :task-name="taskName"
            group-name="_Base"
            item-name="command"
            :item-conf="baseGroup.command!"
            :display-name="t('custom.command')"
            :help="t('custom.help.command')"
          />
        </div>
      </q-card-section>
    </q-card>

    <!-- Display other groups -->
    <group-card
      v-for="(group, groupName) in customGroups"
      :key="groupName"
      :ist-name="istName"
      :menu-name="menuName"
      :task-name="taskName"
      :group-name="String(groupName)"
      :group-conf="group"
    />
  </q-scroll-area>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import GroupCard from '../components/GroupCard.vue';
import CardTitle from '../components/CardTitle.vue';
import ItemLine from '../components/ItemLine.vue';
import type { Group, Task } from '../services/response';
import { useI18n } from 'vue-i18n';

const props = defineProps<{
  istName: string;
  menuName: string;
  taskName: string;

  taskConf: Task;
}>();

const { t } = useI18n();

// Get the _Base group
const baseGroup = computed<Group | undefined>(() => {
  return props.taskConf['_Base'];
});

// Get all groups except _Base
const customGroups = computed(() => {
  return Object.fromEntries(
    Object.entries(props.taskConf).filter(([key]) => key !== '_Base'),
  );
});
</script>
