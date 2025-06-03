import { defineStore } from 'pinia';
import type {
  Layout,
  TaskQueue,
  Translation,
  TranslationMenu,
  TranslationTask,
  TranslationGroup,
  TranslationItem,
} from '../services/response';
import {
  deleteInstance,
  fetchInstance,
  updateInstance,
  connectWebSocket,
  updateTaskQueue,
  fetchSettings,
  updateSettings,
} from '../services/api';
import type { MessageLanguages } from 'src/boot/i18n';
import type { Composer } from 'vue-i18n';
import type { QVueGlobals } from 'quasar';

const isTranslationMenu = (obj: unknown): obj is TranslationMenu => {
  if (!obj || typeof obj !== 'object') return false;
  const candidate = obj as Record<string, unknown>;
  return (
    'name' in candidate &&
    typeof candidate.name === 'string' &&
    'tasks' in candidate &&
    typeof candidate.tasks === 'object'
  );
};

const isTranslationTask = (obj: unknown): obj is TranslationTask => {
  if (!obj || typeof obj !== 'object') return false;
  const candidate = obj as Record<string, unknown>;
  return (
    'name' in candidate &&
    typeof candidate.name === 'string' &&
    'groups' in candidate &&
    typeof candidate.groups === 'object'
  );
};

const isTranslationGroup = (obj: unknown): obj is TranslationGroup => {
  if (!obj || typeof obj !== 'object') return false;
  const candidate = obj as Record<string, unknown>;
  return (
    'name' in candidate &&
    typeof candidate.name === 'string' &&
    'items' in candidate &&
    typeof candidate.items === 'object' &&
    (!('help' in candidate) || typeof candidate.help === 'string')
  );
};

const isTranslationItem = (obj: unknown): obj is TranslationItem => {
  if (!obj || typeof obj !== 'object') return false;
  const candidate = obj as Record<string, unknown>;
  const keys = Object.keys(candidate);
  return (
    keys.length >= 1 &&
    keys.length <= 3 &&
    'name' in candidate &&
    typeof candidate.name === 'string' &&
    keys.every((key) => ['name', 'help', 'options'].includes(key))
  );
};

export const useIstStore = defineStore('instance', {
  state: () => ({
    layout: {} as Layout,
    translation: {} as Record<string, Translation>,
    ready: {} as Record<string, boolean>,
    workingTemplate: [] as string[],
    error: null as string | null,
  }),
  getters: {
    instanceNames: (state) => Object.keys(state.layout),

    getTranslatedText: (state) => (instanceName: string, path: string) => {
      const translation = state.translation[instanceName];
      if (!translation) return null;

      // Parse the path, e.g., "Menu.tasks.Task1.groups.Group2.items.setting1.name"
      const keys = path.split('.');
      let current:
        | Translation
        | TranslationMenu
        | TranslationTask
        | TranslationGroup
        | TranslationItem
        | Record<string, unknown>
        | string
        | undefined = translation;

      try {
        for (const key of keys) {
          if (!current || typeof current === 'string') {
            return null;
          }

          if (isTranslationMenu(current)) {
            if (key === 'name') {
              current = current.name;
            } else if (key === 'tasks') {
              current = current.tasks as Record<string, unknown>;
            } else {
              current = current.tasks[key];
            }
          } else if (isTranslationTask(current)) {
            if (key === 'name') {
              current = current.name;
            } else if (key === 'groups') {
              current = current.groups as Record<string, unknown>;
            } else {
              current = current.groups[key];
            }
          } else if (isTranslationGroup(current)) {
            if (key === 'name') {
              current = current.name;
            } else if (key === 'help') {
              current = current.help;
            } else if (key === 'items') {
              current = current.items as Record<string, unknown>;
            } else {
              current = current.items[key];
            }
          } else if (isTranslationItem(current)) {
            if (key === 'name') {
              current = current.name;
            } else if (key === 'help') {
              current = current.help;
            } else if (key === 'options') {
              current = current.options as Record<string, unknown>;
            } else if (current.options) {
              current = current.options[key];
            }
          } else if (typeof current === 'object' && key in current) {
            current = (current as Record<string, unknown>)[key] as
              | string
              | Record<string, unknown>;
          } else {
            return null;
          }
        }

        return typeof current === 'string' ? current : null;
      } catch {
        return null;
      }
    },
  },
  actions: {
    async loadInstance(instanceName?: string) {
      try {
        if (instanceName) {
          const response = await fetchInstance(instanceName);
          this.layout = { ...this.layout, ...response.layout };
          this.ready = { ...this.ready, ...response.ready };
          this.workingTemplate.push(...response.working_template);
          this.translation = { ...this.translation, ...response.translation };
        } else {
          const response = await fetchInstance();
          this.layout = response.layout;
          this.ready = response.ready;
          this.workingTemplate = response.working_template;
          this.translation = response.translation;
        }
        this.error = null;
      } catch (err) {
        this.error = err instanceof Error ? err.message : 'Unknown error';
        console.error('Failed to load instances:', err);
      }
    },

    async updateInstance(
      istName: string,
      menuName: string,
      taskName: string,
      groupName: string,
      itemName: string,
      value: unknown,
    ) {
      try {
        const response = await updateInstance(istName, {
          menu: menuName,
          task: taskName,
          group: groupName,
          item: itemName,
          value: value,
        });

        // Update the value in the layout
        if (
          this.layout[istName]?.[menuName]?.[taskName]?.[groupName]?.[itemName]
        ) {
          this.layout[istName][menuName][taskName][groupName][itemName].value =
            value;
        }

        // If new translations are returned, update translation data
        if (response) {
          this.translation = {
            ...this.translation,
            [istName]: response,
          };
        }
      } catch (err) {
        this.error = err instanceof Error ? err.message : 'Unknown error';
        throw err;
      }
    },

    async updateReady(value: boolean, instanceName?: string) {
      try {
        if (instanceName) {
          await updateInstance(instanceName, {
            menu: 'Project',
            task: 'General',
            group: '_Base',
            item: 'ready',
            value: value,
          });
          this.ready[instanceName] = value;
        } else {
          await Promise.all(
            Object.keys(this.ready).map(async (name) => {
              await updateInstance(name, {
                menu: 'Project',
                task: 'General',
                group: '_Base',
                item: 'ready',
                value: value,
              });
              this.ready[name] = value;
            }),
          );
        }
      } catch (err) {
        this.error = err instanceof Error ? err.message : 'Unknown error';
        console.error('Failed to update ready status:', err);
      }
    },

    async removeInstance(instanceName: string) {
      try {
        await deleteInstance(instanceName);
        delete this.layout[instanceName];
        delete this.ready[instanceName];
        this.error = null;
      } catch (err) {
        this.error = err instanceof Error ? err.message : 'Unknown error';
        console.error('Failed to delete instance:', err);
      }
    },
  },
});

export const useIstTabStore = defineStore('ist-tab', {
  state: () => ({
    tab: '',
  }),
  actions: {
    setTab(newTab: string) {
      this.tab = newTab;
    },
  },
});

export const useTaskTabStore = defineStore('task-tab', {
  state: () => ({
    // Store independent task tab state for each instance
    tabs: {} as Record<string, string>,
  }),
  actions: {
    setTab(instanceName: string, newTab: string) {
      this.tabs[instanceName] = newTab;
    },
    getTab(instanceName: string) {
      // Return default value 'Home' if instance has no state
      return this.tabs[instanceName] || 'Home';
    },
  },
});

export const useSchedulerStore = defineStore('scheduler', {
  state: () => ({
    queues: {} as Record<string, TaskQueue>,
    states: {} as Record<string, string>, // Store instance states
    logs: {} as Record<string, string[]>,
    schedulerRunning: false, // Store scheduler state
    instanceUpdating: {} as Record<string, boolean>,
    instanceUpdated: {} as Record<string, boolean>,
  }),

  getters: {
    isInstanceRunning: (state) => (instanceName: string) => {
      return state.states[instanceName] === 'running';
    },

    getInstanceState: (state) => (instanceName: string) => {
      return state.states[instanceName];
    },

    isSchedulerRunning: (state) => {
      return state.schedulerRunning;
    },

    hasRunningInstance: (state) => {
      // If scheduler is running, return false to allow stopping
      if (state.schedulerRunning) {
        return false;
      }
      // Otherwise check if there are any running instances
      return Object.values(state.states).some((state) => state === 'running');
    },

    isInstanceUpdating: (state) => (instanceName: string) => {
      return !!state.instanceUpdating[instanceName];
    },

    isInstanceUpdated: (state) => (instanceName: string) => {
      return !!state.instanceUpdated[instanceName];
    },
  },
  actions: {
    // Initialize WebSocket connection
    initWebSocket() {
      return connectWebSocket((data) => {
        if (data.type === 'queue' && data.queue) {
          this.queues[data.instance_name] = data.queue;
        } else if (data.type === 'state' && data.state) {
          if (data.instance_name) {
            this.states[data.instance_name] = data.state;
          } else {
            // Update scheduler state if no instance_name
            this.schedulerRunning = data.state === 'running';
          }
        } else if (data.type === 'log' && data.instance_name && data.content) {
          (this.logs[data.instance_name] ??= []).push(data.content);
        } else if (data.type === 'file_change' && data.instance_name) {
          // Handle file modification notifications
          console.log(
            `Configuration file modified for instance ${data.instance_name}: ${data.filename}`,
          );

          // Trigger a reload of the instance configuration
          this.loadInstance(data.instance_name).catch((err) => {
            console.error(
              `Failed to reload instance ${data.instance_name} after file change:`,
              err,
            );
          });
        }
      });
    },

    // Load instance configuration (delegated to instance store)
    async loadInstance(instanceName: string) {
      const istStore = useIstStore();
      return await istStore.loadInstance(instanceName);
    },

    // Update task queue
    async updateQueue(queues: Record<string, TaskQueue>) {
      try {
        await updateTaskQueue(queues);
        Object.assign(this.queues, queues);
      } catch (err) {
        console.error('Failed to update queue:', err);
      }
    },

    setInstanceUpdating(instanceName: string, updating: boolean) {
      this.instanceUpdating[instanceName] = updating;
    },

    setInstanceUpdated(instanceName: string, updated: boolean) {
      this.instanceUpdated[instanceName] = updated;
    },
  },
});

export const useSettingsStore = defineStore('settings', {
  state: () => ({
    language: 'en-US' as MessageLanguages,
    runOnStartup: false,
    schedulerCron: '',
    autoActionTrigger: 'scheduler_end', // 'scheduler_end' | 'scheduled'
    autoActionCron: '',
    autoActionType: 'none', // 'none' | 'close_app' | 'hibernate' | 'shutdown'
  }),
  actions: {
    async setLanguage(lang: MessageLanguages, i18n: Composer) {
      this.language = lang;
      i18n.locale.value = lang;
      await this.saveSettings();
    },
    async setRunOnStartup(value: boolean) {
      this.runOnStartup = value;
      await this.saveSettings();
    },
    async setSchedulerCron(value: string) {
      this.schedulerCron = value;
      await this.saveSettings();
    },
    async setAutoActionTrigger(value: string) {
      this.autoActionTrigger = value;
      await this.saveSettings();
    },
    async setAutoActionCron(value: string) {
      this.autoActionCron = value;
      await this.saveSettings();
    },
    async setAutoActionType(value: string) {
      this.autoActionType = value;
      await this.saveSettings();
    },
    async loadSettings(i18n: Composer, quasar: QVueGlobals) {
      try {
        const settings = await fetchSettings();

        // If language is empty/null, auto-detect language based on system
        if (!settings.language) {
          const locale = quasar.lang.getLocale() || 'en-US';
          const lang = locale.toLowerCase().startsWith('zh')
            ? 'zh-CN'
            : 'en-US';
          this.language = lang as MessageLanguages;
          // Save the auto-detected language
          await this.setLanguage(this.language, i18n);
        } else {
          // Use saved language setting with type validation
          const validLanguages = ['zh-CN', 'en-US'];
          const savedLang = validLanguages.includes(settings.language)
            ? (settings.language as MessageLanguages)
            : 'en-US';
          this.language = savedLang;
          i18n.locale.value = savedLang;
        }

        // Load other settings with defaults
        this.runOnStartup = settings.runOnStartup ?? false;
        this.schedulerCron = settings.schedulerCron || '';
        this.autoActionTrigger = settings.autoActionTrigger || 'scheduler_end';
        this.autoActionCron = settings.autoActionCron || '';
        this.autoActionType = settings.autoActionType || 'none';

        // Validate settings
        const validTriggers = ['scheduler_end', 'scheduled'];
        const validTypes = ['none', 'close_app', 'hibernate', 'shutdown'];

        if (!validTriggers.includes(this.autoActionTrigger)) {
          await this.setAutoActionTrigger('scheduler_end');
        }

        if (!validTypes.includes(this.autoActionType)) {
          await this.setAutoActionType('none');
        }
      } catch (error) {
        console.error('Failed to load settings:', error);
        // Fallback to auto-detection if loading fails
        const locale = quasar.lang.getLocale() || 'en-US';
        const lang = locale.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en-US';
        this.setLanguage(lang as MessageLanguages, i18n);
      }
    },
    async saveSettings() {
      try {
        await updateSettings({
          language: this.language,
          runOnStartup: this.runOnStartup,
          schedulerCron: this.schedulerCron,
          autoActionTrigger: this.autoActionTrigger,
          autoActionCron: this.autoActionCron,
          autoActionType: this.autoActionType,
        });
      } catch (error) {
        console.error('Failed to save settings:', error);
      }
    },
  },
});
