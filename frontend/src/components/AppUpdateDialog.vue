<template>
  <!-- Update confirmation dialog -->
  <q-dialog v-model="showDialog" persistent>
    <q-card style="min-width: 400px" class="q-pa-md">
      <q-card-section class="row items-center q-pb-sm">
        <q-icon
          name="system_update"
          color="primary"
          size="md"
          class="q-mr-sm"
        />
        <span class="text-h6 text-weight-medium">{{
          t('appUpdate.title')
        }}</span>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <p class="text-body1 q-mb-sm">{{ t('appUpdate.message') }}</p>
      </q-card-section>

      <q-card-actions align="right" class="q-pt-md">
        <q-btn
          flat
          :label="t('appUpdate.later')"
          @click="onCancel"
          :disable="isUpdating"
          class="text-grey-7"
        />
        <q-btn
          outline
          color="primary"
          :label="t('appUpdate.updateNow')"
          @click="onConfirm"
          :loading="isUpdating"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
  <!-- Progress dialog -->
  <q-dialog v-model="showProgressDialog" persistent>
    <q-card style="min-width: 400px" class="q-pa-md">
      <q-card-section class="row items-center q-pb-sm">
        <q-icon name="download" color="info" size="md" class="q-mr-sm" />
        <span class="text-h6 text-weight-medium">{{
          t('appUpdate.downloading')
        }}</span>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <q-linear-progress
          :value="downloadProgress"
          color="primary"
          size="8px"
          rounded
          class="q-mb-md"
        />
        <p class="text-body2 text-grey-7 q-mb-none">
          {{ progressDescription }}
        </p>
      </q-card-section>
    </q-card>
  </q-dialog>
  <!-- Upgrade confirmation dialog -->
  <q-dialog v-model="showUpgradeConfirmDialog" persistent>
    <q-card style="min-width: 400px" class="q-pa-md">
      <q-card-section class="row items-center q-pb-sm">
        <q-icon name="upgrade" color="warning" size="md" class="q-mr-sm" />
        <span class="text-h6 text-weight-medium">{{
          t('appUpdate.confirmUpgrade')
        }}</span>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <p class="text-body1 q-mb-none">{{ upgradeConfirmMessage }}</p>
      </q-card-section>

      <q-card-actions align="right" class="q-pt-md">
        <q-btn
          flat
          :label="t('appUpdate.cancel')"
          @click="onUpgradeCancel"
          class="text-grey-7"
        />
        <q-btn
          outline
          color="warning"
          :label="t('appUpdate.confirm')"
          @click="onUpgradeConfirm"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
  <!-- Restart confirmation dialog -->
  <q-dialog v-model="showRestartDialog" persistent>
    <q-card style="min-width: 400px" class="q-pa-md">
      <q-card-section class="row items-center q-pb-sm">
        <q-icon name="restart_alt" color="positive" size="md" class="q-mr-sm" />
        <span class="text-h6 text-weight-medium">{{
          t('appUpdate.restartTitle')
        }}</span>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <p class="text-body1 q-mb-sm">{{ t('appUpdate.restartMessage') }}</p>
        <p class="text-caption text-grey-6 q-mb-none">
          {{ t('appUpdate.restartNote') }}
        </p>
      </q-card-section>

      <q-card-actions align="right" class="q-pt-md">
        <q-btn
          flat
          :label="t('appUpdate.restartLater')"
          @click="onRestartLater"
          class="text-grey-7"
        />
        <q-btn
          outline
          color="positive"
          :label="t('appUpdate.restartNow')"
          @click="onRestartNow"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useQuasar } from 'quasar';
import {
  onUpdateMessage,
  offUpdateMessage,
  sendUpdateMessage,
} from '../services/api';
import type { UpdateProgressData } from '../services/response';

// Props to control dialog behavior
interface Props {
  isManualUpdate?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  isManualUpdate: false,
});

const emit = defineEmits<{
  cancel: [];
  confirm: [];
  complete: [];
}>();

const { t } = useI18n();
const $q = useQuasar();

const showDialog = ref(!props.isManualUpdate);
const showProgressDialog = ref(false);
const showUpgradeConfirmDialog = ref(false);
const showRestartDialog = ref(false);
const isUpdating = ref(false);
const downloadProgress = ref(0);
const progressDescription = ref('');
const upgradeConfirmMessage = ref('');

let confirmDialogTimer: NodeJS.Timeout | null = null;

const onCancel = () => {
  if (confirmDialogTimer) {
    clearTimeout(confirmDialogTimer);
    confirmDialogTimer = null;
  }

  sendUpdateMessage('update_confirm_response', { confirmed: false });
  showDialog.value = false;
  emit('cancel');
};

const onConfirm = () => {
  if (confirmDialogTimer) {
    clearTimeout(confirmDialogTimer);
    confirmDialogTimer = null;
  }

  isUpdating.value = true;
  showDialog.value = false;

  sendUpdateMessage('update_confirm_response', { confirmed: true });

  $q.notify({
    type: 'info',
    message: t('appUpdate.updateStarted'),
  });
};

const onUpgradeConfirm = () => {
  sendUpdateMessage('update_confirm_response', { confirmed: true });
  showUpgradeConfirmDialog.value = false;
};

const onUpgradeCancel = () => {
  sendUpdateMessage('update_confirm_response', { confirmed: false });
  showUpgradeConfirmDialog.value = false;
  isUpdating.value = false;
  emit('cancel');
};

const onRestartNow = () => {
  sendUpdateMessage('restart_confirm_response', { confirmed: true });
  showRestartDialog.value = false;
  emit('complete');
};

const onRestartLater = () => {
  sendUpdateMessage('restart_confirm_response', { confirmed: false });
  showRestartDialog.value = false;
  $q.notify({
    type: 'positive',
    message: t('appUpdate.updateComplete'),
  });
  emit('complete');
};

const handleProgress = (data: unknown) => {
  const progressData = data as UpdateProgressData;
  downloadProgress.value = progressData.progress;
  progressDescription.value = progressData.description;
  showProgressDialog.value = true;
};

const handleConfirmUpgrade = (data: unknown) => {
  upgradeConfirmMessage.value =
    String(data) || t('appUpdate.confirmUpgradeMessage');
  showUpgradeConfirmDialog.value = true;
};

const handleConfirmRestart = () => {
  showProgressDialog.value = false;
  showRestartDialog.value = true;
};

const handleComplete = () => {
  showProgressDialog.value = false;
  isUpdating.value = false;
  $q.notify({
    type: 'positive',
    message: t('appUpdate.updateComplete'),
  });
  emit('complete');
};

const handleError = (data: unknown) => {
  const errorMessage = String(data) || 'Unknown error';
  showProgressDialog.value = false;
  showUpgradeConfirmDialog.value = false;
  isUpdating.value = false;
  $q.notify({
    type: 'negative',
    message: t('appUpdate.error', { error: errorMessage }),
  });
  emit('cancel');
};

const handleRestartStarted = () => {
  $q.notify({
    type: 'info',
    message: t('appUpdate.restarting'),
  });
};

onMounted(() => {
  onUpdateMessage('update_progress', handleProgress);
  onUpdateMessage('update_confirm_upgrade', handleConfirmUpgrade);
  onUpdateMessage('update_confirm_restart', handleConfirmRestart);
  onUpdateMessage('update_complete', handleComplete);
  onUpdateMessage('update_error', handleError);
  onUpdateMessage('update_restart_started', handleRestartStarted);

  if (showDialog.value && !props.isManualUpdate) {
    confirmDialogTimer = setTimeout(() => {
      if (showDialog.value) {
        showDialog.value = false;
        emit('cancel');
      }
      confirmDialogTimer = null;
    }, 30000);
  }
});

onUnmounted(() => {
  if (confirmDialogTimer) {
    clearTimeout(confirmDialogTimer);
    confirmDialogTimer = null;
  }

  offUpdateMessage('update_progress', handleProgress);
  offUpdateMessage('update_confirm_upgrade', handleConfirmUpgrade);
  offUpdateMessage('update_confirm_restart', handleConfirmRestart);
  offUpdateMessage('update_complete', handleComplete);
  offUpdateMessage('update_error', handleError);
  offUpdateMessage('update_restart_started', handleRestartStarted);
});
</script>
