export interface Item {
  type: string;
  value: unknown;
  help?: string;
  option?: unknown[];
  hidden?: boolean;
  disabled?: boolean;
}

export interface Group {
  [argumentName: string]: Item;
}

export interface Task {
  [groupName: string]: Group;
}

export interface Menu {
  [taskName: string]: Task;
}

export interface Instance {
  [menuName: string]: Menu;
}

export interface Layout {
  [instanceName: string]: Instance;
}

export interface TaskQueue {
  running: string;
  waiting: string[];
  stopped: string[];
}

export interface RspApi {
  code: number;
  message: string;
  detail: string;
}

export interface TranslationItem {
  name: string;
  help?: string;
  options?: Record<string, string>;
}

export interface TranslationGroup {
  name: string;
  help?: string;
  items: Record<string, TranslationItem>;
}

export interface TranslationTask {
  name: string;
  groups: Record<string, TranslationGroup>;
}

export interface TranslationMenu {
  name: string;
  tasks: Record<string, TranslationTask>;
}

export interface Translation {
  [menuName: string]: TranslationMenu;
}

export interface RspGetInstance {
  working_template: string[];
  ready: Record<string, boolean>;
  layout: Layout;
  translation: Record<string, Translation>;
}

export interface RspUpdateRepo {
  is_updated: boolean;
}

export interface RspWSMessage {
  type: 'queue' | 'log' | 'state';
  instance_name: string;
  content?: string;
  queue?: TaskQueue;
  state?: string;
}
