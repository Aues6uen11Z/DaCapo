<template>
  <!-- Group card is only shown if there are visible items -->
  <q-card v-if="!allItemsHidden" style="width: 90%" class="tw-mr-8">
    <q-card-section>
      <card-title
        :name="getGroupName(menuName, taskName, groupName)"
        :help="getGroupHelp(menuName, taskName, groupName)"
      />
      <div class="my-column tw-gap-0">
        <div
          v-for="(item, itemName) in displayItems"
          :key="itemName"
          class="tw-grid tw-grid-cols-3 tw-w-full"
        >
          <item-line
            :ist-name="istName"
            :menu-name="menuName"
            :task-name="taskName"
            :group-name="groupName"
            :item-name="String(itemName)"
            :item-conf="item"
            :display-name="
              getItemName(menuName, taskName, groupName, String(itemName))
            "
            :help="getItemHelp(menuName, taskName, groupName, String(itemName))"
          />
        </div>
      </div>
    </q-card-section>
  </q-card>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import CardTitle from './CardTitle.vue';
import ItemLine from './ItemLine.vue';
import type { Group } from '../services/response';
import { useTranslation } from '../i18n/index';

const props = defineProps<{
  istName: string;
  menuName: string;
  taskName: string;
  groupName: string;

  groupConf: Group;
}>();

const { getGroupName, getGroupHelp, getItemName, getItemHelp } = useTranslation(
  props.istName,
);

// Filter out '_help' field and compute displayable items
const displayItems = computed(() =>
  Object.fromEntries(
    Object.entries(props.groupConf).filter(([key]) => key !== '_help'),
  ),
);

// Check if all items in the group are hidden
const allItemsHidden = computed(() => {
  return Object.values(displayItems.value).every(
    (item) => item.hidden === true,
  );
});
</script>
