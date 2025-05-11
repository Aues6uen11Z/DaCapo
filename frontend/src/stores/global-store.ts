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
} from '../services/api';
import type { MessageLanguages } from 'src/boot/i18n';
import type { Composer } from 'vue-i18n';
import type { QVueGlobals } from 'quasar';

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

          // Type guard functions for translation interfaces
          const isTranslationMenu = (obj: unknown): obj is TranslationMenu =>
            obj !== null && typeof obj === 'object' && 'tasks' in obj;
          const isTranslationTask = (obj: unknown): obj is TranslationTask =>
            obj !== null && typeof obj === 'object' && 'groups' in obj;
          const isTranslationGroup = (obj: unknown): obj is TranslationGroup =>
            obj !== null && typeof obj === 'object' && 'items' in obj;
          const isTranslationItem = (obj: unknown): obj is TranslationItem =>
            obj !== null &&
            typeof obj === 'object' &&
            ('name' in obj || 'options' in obj);

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
        }
      });
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
    runOnStartup: localStorage.getItem('runOnStartup') === 'true',
    schedulerCron: localStorage.getItem('schedulerCron') || '',
  }),
  actions: {
    setLanguage(lang: MessageLanguages, i18n: Composer) {
      this.language = lang;
      i18n.locale.value = lang;
      localStorage.setItem('language', lang);
    },
    setRunOnStartup(value: boolean) {
      this.runOnStartup = value;
      localStorage.setItem('runOnStartup', String(value));
    },
    setSchedulerCron(value: string) {
      this.schedulerCron = value;
      localStorage.setItem('schedulerCron', value);
    },
    loadSettings(i18n: Composer, quasar: QVueGlobals) {
      // Check for saved language settings
      const savedLang = localStorage.getItem(
        'language',
      ) as MessageLanguages | null;

      if (savedLang) {
        // If there are saved settings, use them
        this.setLanguage(savedLang, i18n);
      } else {
        // If no saved settings, choose based on system language
        const locale = quasar.lang.getLocale() || 'en-US';
        const lang = locale.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en-US';
        this.setLanguage(lang as MessageLanguages, i18n);
      }
    },
  },
});
