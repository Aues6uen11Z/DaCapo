# 架构概览

> 相关文件：`main.go`、`backend/router/router.go`、`backend/service/service_manager.go`、`backend/service/websocket.go`、`backend/utils/ws_manager.go`、`backend/app/app.go`

## 技术栈

| 层 | 技术 |
| --- | --- |
| 桌面框架 | Wails v2（Go 后端 + 内嵌 web 前端） |
| 后端语言 | Go 1.23 |
| Web 框架 | Gin，监听 `:48596` |
| 数据库 | SQLite，经 `gormlite` + GORM |
| 定时任务 | `robfig/cron/v3` |
| WebSocket | `gorilla/websocket` |
| 自更新 | `fynelabs/selfupdate` |
| 前端 | Vue 3 + TypeScript + Quasar + Vite |
| 状态管理 | Pinia |

## 进程结构

`main.go` 做三件事，顺序固定：

1. `model.InitDB()` — 初始化 SQLite/GORM。
2. `router.SetupRouter()` — 启动 Gin 服务（`go r.Run(":48596")`，后台 goroutine）。
3. `wails.Run(...)` — 阻塞，创建桌面窗口并内嵌 `frontend/dist`。

前端同时通过两条通道与后端通信：

- **HTTP**：axios 请求 `http://localhost:48596/api/*`（见 `frontend/src/services/`）。
- **WebSocket**：连接 `ws://localhost:48596/api/ws`，接收实时状态/队列/日志推送。

> 注意：Wails 绑定（`backend/app`）目前主要用于版本查询和生命周期回调，
> 业务交互基本走 Gin HTTP + WS，而非 Wails 的方法绑定。

## 后端分层

```
controller/  HTTP 处理器，薄层，解析请求并调用 service
service/     业务逻辑
model/       GORM 模型 + 数据库访问；全局 scheduler 单例
router/      路由注册 + Gin 中间件（CORS、日志、recovery）
app/         Wails 绑定与生命周期回调（Startup/DomReady/Shutdown）
utils/       日志(zap)、WSManager、版本、自更新、symlink 等工具
```

请求链路：`HTTP 请求 → router → controller → service → model`，结果原路返回；
副作用（状态变化、日志）通过 WebSocket 异步推给前端。

## 服务层与依赖注入

`service.ServiceManager` 是单例（`service.GetServiceManager()`），
在 `init()` 中按依赖顺序构造各服务：

```
WebSocketService          （无依赖，最先建）
  └─ NotificationService  （可选，需 settings.ServerChanSendKey）
       └─ SchedulerService  （依赖 ws + notification）
InstanceService           （无依赖）
InstanceUpdaterService    （依赖 scheduler + ws）
```

controller 通过 `controller.Services`（即该单例）或直接 `service.GetServiceManager()`
获取服务。`ReloadNotificationService()` 在设置变更后重建通知服务并回填给 scheduler。

> 关键点：服务在启动时一次性注入。若新增需要通知/WS 的服务，
> 在 `ServiceManager.init()` 里加依赖并暴露 getter。

## 路由总览

所有业务路由挂在 `/api` 下（`backend/router/router.go`）：

| 分组 | 示例 | 说明 |
| --- | --- | --- |
| `/api/instance` | `GET/PATCH/DELETE /:name`、`POST /local|template|remote` | 实例 CRUD 与创建来源 |
| `/api/template` | `GET`、`DELETE /:name` | 模板读取/删除 |
| `/api/scheduler` | `PATCH /queue`、`PATCH /state`、`POST /cron` | 任务队列、启停、调度 cron |
| `/api/updater/:name` | `GET` | 触发实例仓库更新 |
| `/api/settings` | `GET`、`PATCH` | 全局设置 |
| `/api/app/check-update` | `POST` | 应用自更新检查 |
| `/api/ws` | `GET`（升级） | WebSocket 入口 |

## WebSocket 实时通信

- 连接：前端 `new WebSocket("ws://localhost:48596/api/ws")`；
  后端 `WebSocketService.HandleConnection` 把连接注册进 `utils.WSManager`（单例）。
- 广播：service 调 `wsService.BroadcastState/Queue/Log` → `WSManager.BroadcastJSON` → 遍历所有连接 `WriteJSON`。
- 消息 `type` 字段取值：

| `type` | 含义 | 触发方 |
| --- | --- | --- |
| `state` | 实例状态或全局调度器状态 | `BroadcastState(name, status)`；`name` 为空表示全局 |
| `queue` | 实例任务队列（Running/Waiting/Stopped） | `BroadcastQueue(name)` |
| `log` | 任务执行日志行 | `BroadcastLog(name, content)`（同时落盘） |

前端收到后更新 Pinia store，由 Vue 响应式刷新界面；`ws.onclose` 后 5 秒重连并保留已注册的回调。
