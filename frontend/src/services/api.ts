import axios from 'axios';
import type {
  TaskQueue,
  RspApi,
  RspGetInstance,
  RspWSMessage,
  RspUpdateRepo,
  Translation,
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

// GET /api/instance
export async function fetchInstance(
  instanceName?: string,
): Promise<RspGetInstance> {
  const url = instanceName ? `/instance/${instanceName}` : '/instance';
  const response = await api.get<RspApi & RspGetInstance>(url);

  if (response.data.code === 0) {
    return {
      working_template: response.data.working_template,
      layout: response.data.layout,
      ready: response.data.ready,
      translation: response.data.translation,
    };
  }

  console.error(response.data.detail);
  throw new Error(response.data.detail);
}

// POST /api/instance/local
export async function createLocalInstance(data: ReqFromLocal) {
  const response = await api.post<RspApi>('/instance/local', data);

  if (response.data.code !== 0) {
    console.error(response.data.detail);
    throw new Error(response.data.detail);
  }
}

// POST /api/instance/template
export async function createTemplateInstance(data: ReqFromTemplate) {
  const response = await api.post<RspApi>('/instance/template', data);

  if (response.data.code !== 0) {
    console.error(response.data.detail);
    throw new Error(response.data.detail);
  }
}

// POST /api/instance/remote
export async function createRemoteInstance(data: ReqFromRemote) {
  const response = await api.post<RspApi>('/instance/remote', data);

  if (response.data.code !== 0) {
    console.error(response.data.detail);
    throw new Error(response.data.detail);
  }
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

  if (response.data.code !== 0) {
    console.error(response.data.detail);
    throw new Error(response.data.detail);
  }

  return response.data.translation;
}

// DELETE /api/instance/{instanceName}
export async function deleteInstance(instanceName: string) {
  const response = await api.delete<RspApi>(`/instance/${instanceName}`);

  if (response.data.code !== 0) {
    console.error(response.data.detail);
    throw new Error(response.data.detail);
  }
}

// GET /api/template
export async function fetchTemplates(): Promise<string[]> {
  const response = await api.get<RspApi & { templates: string[] }>('/template');

  if (response.data.code === 0) {
    return response.data.templates;
  }

  console.error(response.data.detail);
  throw new Error(response.data.detail);
}

// DELETE /api/template/{templateName}
export async function deleteTemplate(templateName: string): Promise<void> {
  const response = await api.delete<RspApi>(`/template/${templateName}`);

  if (response.data.code !== 0) {
    console.error(response.data.detail);
    throw new Error(response.data.detail);
  }
}

let ws: WebSocket | null = null;
const wsCallbacks: ((data: RspWSMessage) => void)[] = [];

export function connectWebSocket(callback: (data: RspWSMessage) => void) {
  if (!ws || ws.readyState === WebSocket.CLOSED) {
    ws = new WebSocket('ws://localhost:48596/api/scheduler/ws');

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      wsCallbacks.forEach((cb) => cb(data));
    };

    ws.onclose = () => {
      console.log('WebSocket connection closed');
      setTimeout(() => connectWebSocket(callback), 5000);
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

// PATCH /api/scheduler/queue
export async function updateTaskQueue(queues: Record<string, TaskQueue>) {
  const response = await api.patch<RspApi>('/scheduler/queue', { queues });

  if (response.data.code !== 0) {
    console.error(response.data.detail);
    throw new Error(response.data.detail);
  }
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

  if (response.data.code !== 0) {
    console.error(response.data.detail);
    throw new Error(response.data.detail);
  }
}

// POST /api/scheduler/cron
export async function sendSchedulerCron(cronExpr: string) {
  const response = await api.post<RspApi>('/scheduler/cron', {
    cron_expr: cronExpr,
  });

  if (response.data.code !== 0) {
    console.error(response.data.detail);
    throw new Error(response.data.detail);
  }
}

// GET /api/updater/{instanceName}
export async function updateRepo(instanceName: string): Promise<boolean> {
  const response = await api.get<RspApi & RspUpdateRepo>(
    `/updater/${instanceName}`,
  );

  if (response.data.code === 0) {
    return response.data.is_updated;
  }

  console.error(response.data.detail);
  throw new Error(response.data.detail);
}
