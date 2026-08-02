# DaCapo 开发文档索引

本目录是 DaCapo 内部架构与实现说明，面向开发者和 AI Agent。
每篇文档小而自洽，按需阅读即可。先读本索引定位，再打开对应文件。

> 第三方脚本作者（如何用 `template.yaml` 适配 DaCapo）请看
> [`开发者指南.md`](../开发者指南.md) / [`DeveloperGuide.md`](../DeveloperGuide.md)，
> 不在本目录范围内。

## 路线图：问题 -> 文档

| 想了解 | 读这个 |
| --- | --- |
| 整体架构、技术栈、一次请求的链路 | [architecture.md](./architecture.md) |
| ServiceManager 依赖注入、各 Service 职责 | [architecture.md](./architecture.md#服务层与依赖注入) |
| WebSocket 实时通信、消息类型、广播路径 | [architecture.md](./architecture.md#websocket-实时通信) |
| 实例状态机、启动/停止/重启 | [backend/instance.md](./backend/instance.md) |
| 调度器、cron、全部启动、结束后动作 | [backend/scheduler.md](./backend/scheduler.md) |
| 实例更新：git pull / uv venv / 依赖安装 / 模板重解析 | [backend/updater.md](./backend/updater.md) |
| `template.yaml` / `instance.json` 字段结构 | [config-schema.md](./config-schema.md) |
| 前端结构、stores/services、与后端通信 | [frontend/overview.md](./frontend/overview.md) |

## 文档清单

```
docs/dev/
  README.md              本索引
  architecture.md        架构、技术栈、分层、依赖注入、路由、WebSocket
  config-schema.md       模板/实例配置的四层结构与保留名称
  backend/
    instance.md          实例状态机、TaskManager、StartOne/stopOne
    scheduler.md         Scheduler 单例、cron 注册、StartAll 编排、auto-action
    updater.md           仓库拉取、Python 环境、依赖安装、模板重解析
  frontend/
    overview.md          目录结构、HTTP/WS/Wails 三通道、Pinia store、页面组件
```

## 约定

- 每篇文档开头列「相关文件」，锚定真实代码路径，便于核对与发现漂移。
- 稳定信息（架构、技术栈）与易变信息（模块内部逻辑）分文件存放。
- 改了某模块的代码，顺手更新对应文档；发现文档与代码不符，以代码为准并修正文档。
