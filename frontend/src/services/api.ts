import axios from 'axios';
import type {
  TaskQueue,
  RspApi,
  RspGetInstance,
  RspWSMessage,
  RspUpdateRepo,
  Translation,
  RspSettings,
  UpdateMessage,
} from './response';
import type {
  ReqFromLocal,
  ReqFromRemote,
  ReqFromTemplate,
  ReqUpdateInstance,
} from './request';

const api = axios.create({
  baseURL: 'http://localhost:48596/api',
});

// Helper function for consistent error handling
function handleApiResponse<T>(response: { data: RspApi & T }): T {
  if (response.data.code !== 0) {
    console.error(response.data.detail);
    throw new Error(response.data.detail);
  }
  return response.data as T;
}

// GET /api/instance
export async function fetchInstance(
  instanceName?: string,
): Promise<RspGetInstance> {
  const url = instanceName ? `/instance/${instanceName}` : '/instance';
  const response = await api.get<RspApi & RspGetInstance>(url);
  return handleApiResponse(response);
}

// POST /api/instance/local
export async function createLocalInstance(data: ReqFromLocal) {
  const response = await api.post<RspApi>('/instance/local', data);
  handleApiResponse(response);
}

// POST /api/instance/template
export async function createTemplateInstance(data: ReqFromTemplate) {
  const response = await api.post<RspApi>('/instance/template', data);
  handleApiResponse(response);
}

// POST /api/instance/remote
export async function createRemoteInstance(data: ReqFromRemote) {
  const response = await api.post<RspApi>('/instance/remote', data);
  handleApiResponse(response);
}

// PATCH /api/instance/{instanceName}
export async function updateInstance(
  instanceName: string,
  data: ReqUpdateInstance,
): Promise<Translation | undefined> {
  const response = await api.patch<RspApi & { translation?: Translation }>(
    `/instance/${instanceName}`,
    data,
  );
  const result = handleApiResponse(response);
  return result.translation;
}

// DELETE /api/instance/{instanceName}
export async function deleteInstance(instanceName: string) {
  const response = await api.delete<RspApi>(`/instance/${instanceName}`);
  handleApiResponse(response);
}

// PATCH /api/instance/order
export async function updateInstanceOrder(names: string[]) {
  const response = await api.patch<RspApi>('/instance/order', { names });
  handleApiResponse(response);
}

// GET /api/template
export async function fetchTemplates(): Promise<string[]> {
  const response = await api.get<RspApi & { templates: string[] }>('/template');
  const result = handleApiResponse(response);
  return result.templates;
}

// DELETE /api/template/{templateName}
export async function deleteTemplate(templateName: string): Promise<void> {
  const response = await api.delete<RspApi>(`/template/${templateName}`);
  handleApiResponse(response);
}

// Unified WebSocket management
let ws: WebSocket | null = null;
const wsCallbacks: ((data: RspWSMessage) => void)[] = [];
const updateCallbacks: Map<string, ((data: unknown) => void)[]> = new Map();

// Helper function to check if message is an app update message
function isUpdateMessage(data: unknown): data is UpdateMessage {
  if (!data || typeof data !== 'object' || data === null) {
    return false;
  }

  const obj = data as Record<string, unknown>;
  if (!('type' in obj) || typeof obj.type !== 'string') {
    return false;
  }

  const type = obj.type;
  return (
    type.startsWith('update_') ||
    type.includes('restart') ||
    type.includes('upgrade')
  );
}

// Unified WebSocket functions
export function connectWebSocket(callback: (data: RspWSMessage) => void) {
  if (!ws || ws.readyState === WebSocket.CLOSED) {
    ws = new WebSocket('ws://localhost:48596/api/ws');
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);

      // Handle app update messages
      if (isUpdateMessage(data)) {
        const listeners = updateCallbacks.get(data.type);
        if (listeners) {
          listeners.forEach((listener) => listener(data.data));
        }
      } else {
        // Handle scheduler messages
        wsCallbacks.forEach((cb) => cb(data));
      }
    };
    ws.onclose = () => {
      // Reconnect and preserve all existing callbacks
      if (wsCallbacks.length > 0) {
        setTimeout(() => {
          // Reconnect with the first callback (any callback will do since they're all preserved)
          const firstCallback = wsCallbacks[0];
          if (firstCallback) {
            connectWebSocket(firstCallback);
          }
        }, 5000);
      }
    };
  }

  wsCallbacks.push(callback);
  return () => {
    const index = wsCallbacks.indexOf(callback);
    if (index > -1) {
      wsCallbacks.splice(index, 1);
    }
  };
}

export function disconnectWebSocket() {
  if (ws) {
    ws.close();
    ws = null;
  }
  wsCallbacks.length = 0;
  updateCallbacks.clear();
}

export function isWebSocketConnected(): boolean {
  return ws !== null && ws.readyState === WebSocket.OPEN;
}

export function onUpdateMessage(
  messageType: string,
  callback: (data: unknown) => void,
) {
  if (!updateCallbacks.has(messageType)) {
    updateCallbacks.set(messageType, []);
  }
  updateCallbacks.get(messageType)!.push(callback);
}

export function offUpdateMessage(
  messageType: string,
  callback: (data: unknown) => void,
) {
  const callbacks = updateCallbacks.get(messageType);
  if (callbacks) {
    const index = callbacks.indexOf(callback);
    if (index > -1) {
      callbacks.splice(index, 1);
    }
  }
}

export function sendUpdateMessage(messageType: string, data: unknown) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    const message = {
      type: messageType,
      data: data,
    };
    ws.send(JSON.stringify(message));
  } else {
    console.error('WebSocket is not connected');
  }
}

// PATCH /api/scheduler/queue
export async function updateTaskQueue(queues: Record<string, TaskQueue>) {
  const response = await api.patch<RspApi>('/scheduler/queue', { queues });
  handleApiResponse(response);
}

// GET /api/scheduler/queue/{instanceName}
export async function getTaskQueue(instanceName: string) {
  await api.get<RspApi>(`/scheduler/queue/${instanceName}`);
}

// PATCH /api/scheduler/state
export async function updateSchedulerState(
  type: 'start' | 'stop',
  instanceName?: string,
) {
  const response = await api.patch<RspApi>('/scheduler/state', {
    type,
    instance_name: instanceName,
  });
  handleApiResponse(response);
}

// GET /api/updater/{instanceName}
export async function updateRepo(instanceName: string): Promise<boolean> {
  const response = await api.get<RspApi & RspUpdateRepo>(
    `/updater/${instanceName}`,
  );
  const result = handleApiResponse(response);
  return result.is_updated;
}

// GET /api/settings
export async function fetchSettings() {
  const response = await api.get<RspSettings>('/settings');
  return response.data;
}

// PATCH /api/settings
export async function updateSettings(settings: {
  language?: string;
  runOnStartup?: boolean;
  schedulerCron?: string;
  autoActionTrigger?: string;
  autoActionCron?: string;
  autoActionType?: string;
  maxBgConcurrent?: number;
  serverChanSendKey?: string;
}) {
  const response = await api.patch<RspApi>('/settings', settings);
  handleApiResponse(response);
}

// POST /api/app/check-update
export async function startAppUpdate(manual: boolean = false): Promise<void> {
  const params = manual ? { manual: 'true' } : {};
  const response = await api.post<RspApi>('/app/check-update', {}, { params });
  handleApiResponse(response);
}
