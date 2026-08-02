# 前端概览

> 相关文件：`frontend/src/services/api.ts`、`frontend/src/stores/global-store.ts`、`frontend/src/router/`、`frontend/src/boot/`、`frontend/quasar.config.ts`

Vue 3 + TypeScript + Quasar + Vite 的单页应用，构建产物内嵌进 Wails 桌面窗口。
开发服务器 `http://localhost:33204`（`npm run dev`）。

## 目录结构

```
src/
  pages/        路由页面
  components/   可复用组件
  stores/       Pinia store（global-store.ts）
  services/     后端通信（api.ts / request.ts / response.ts）
  router/       路由（index.ts / routes.ts）
  boot/         Quasar 启动插件（axios.ts / i18n.ts）
  i18n/         国际化（zh-CN 默认 / en-US）
  css/          样式
  wailsjs/      Wails 生成的 Go 绑定（app/wailsjs/go/app/App）
```

## 与后端通信

三条通道（详见 [architecture.md](../architecture.md)）：

- **HTTP**：`services/api.ts` 里 axios 实例，`baseURL = http://localhost:48596/api`。
  统一用 `handleApiResponse` 校验：`response.data.code !== 0` 即抛错（`detail` 为信息）。
  请求/响应类型分别在 `request.ts` / `response.ts`。
- **WebSocket**：`api.ts` 维护**单条** `ws` 连接（`ws://localhost:48596/api/ws`）。
  收到的消息分两路：调度类（`state`/`queue`/`log`）分发给 `wsCallbacks`；
  应用更新类（`update_*`/`restart`/`upgrade`）按 `type` 分发给 `updateCallbacks`。
  `ws.onclose` 后 **5 秒**重连并保留回调。
- **Wails 绑定**：`app/wailsjs/go/app/App`，用于 `GetVersion`、`SelectDir`/`SelectFile`、`OpenFileExplorer` 等原生能力。

## 状态管理

`stores/global-store.ts`（Pinia）是全局状态中枢，持有：

- 设置镜像（`language`/`runOnStartup`/`schedulerCron`/`autoAction*`/`maxBgConcurrent`/`serverChanSendkey`），`saveSettings()` 回写后端。
- 调度器与实例状态（由 WS `state`/`queue` 消息驱动）。
- 应用更新状态机（`unknown`/`available`/`up-to-date`/`error`）+ 版本信息。
- 关键 action：`initializeWebSocket`（建连 + 注册更新事件监听）、`checkForAppUpdates`、`setLanguage`。

## 页面与组件

- `MainPage.vue` 主布局；`InstancePage`/`GeneralPage`/`UpdatePage`/`CustomPage` 是实例配置的标签页；
  `HomePage`、`SettingsPage`、`WelcomePage`、`ErrorNotFound` 为其余路由。
- 组件：`GroupCard`/`ItemLine`/`TaskItem` 渲染配置表单，`LogViewer` 展示实时日志，
  `DirInput`/`FileInput` 调 Wails 选目录/文件，`AppUpdateDialog` 处理自更新交互。

## i18n

`zh-CN` 为默认语言。新增界面文案需同时更新 `zh-CN` 与 `en-US`，否则 fallback 到 key。
