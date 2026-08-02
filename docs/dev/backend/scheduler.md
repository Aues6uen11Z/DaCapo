# 调度器与定时

> 相关文件：`backend/service/scheduler.go`、`backend/model/scheduler.go`、`backend/app/app.go`（`DomReady`）、`backend/model/settings.go`

本文讲「何时触发执行」与「结束后做什么」。单实例的执行机制（状态机、`StartOne`/`stopOne`）见
[instance.md](./instance.md)，本文不重复，只覆盖调度编排层。

## Scheduler 单例

`model.GetScheduler()` 返回全局 `Scheduler`（`model/scheduler.go`）：

```go
type Scheduler struct {
    TaskManagers map[string]*TaskManager  // 每实例一个
    IsRunning    bool                      // 全局调度是否在跑
    CronExpr     string                    // 调度器级 cron（settings.scheduler_cron）
    AutoClose    bool                      // 本次运行是否允许结束后动作
    CloseFunc    func()                    // scheduler_end 触发的动作
    mu           sync.Mutex
}
```

`SyncWithDatabase()` 让 `TaskManagers` 与数据库实例保持一致：新实例建 `TaskManager`（初始 `pending`，队列来自 `GetTaskQueue`），已删除实例移除。`GetScheduler()` 每次调用都会同步。

## cron 注册（app.DomReady）

启动时在 `app.DomReady` 里创建**一个** `cron.New()`，注册三类任务后 `c.Start()`：

1. **实例级 cron**：每个 `Ready` 且 `CronExpr != ""` 的实例注册一个 job，到点直接
   `SchedulerService().StartOne(name)`。注意：走的是单实例启动，**不**置 `AutoClose`，
   因此不会触发结束后动作。
2. **启动即运行 / 调度器 cron**（二选一）：
   - `settings.RunOnStartup == true`：置 `AutoClose=true`，`go StartAll()`。
   - 否则 `settings.SchedulerCron != ""`：注册 cron job，到点置 `AutoClose=true` 再 `StartAll()`。
3. **结束后动作**（见下）。

注册前有 `time.Sleep(3 * time.Second)`，给前端就绪留时间。

## 全部启动：StartAll

`StartAll()`（`scheduler.go`）是调度级执行入口：

1. `scheduler.Start()`（`IsRunning=true`），广播全局 `running`。
2. 取所有 `Ready` 实例，跳过 `failed`/缺失的，按 `Background` 字段分前台/后台两组。
3. **后台组**（`Background=true`）：`runBackgroundTasks` 为每个实例起 goroutine 并发执行，
   信号量限流 `settings.max_bg_concurrent`（0=不限）。每个经 `runWithCheck` 循环等待
   `updating` 解除、校验非 `failed` 后调用 `startOne`（单实例执行机制见 [instance.md](./instance.md)）。
4. **前台组**（`Background=false`）：`runForegroundTasks` 顺序执行；
   遇 `updating` 的实例重新排队等待，`failed` 的跳过。两组之间并行。
5. 全部完成：汇总 `SchedulerResult` -> `NotificationService` 通知 ->
   `scheduler.Stop()` -> `TriggerCloseFunc()` -> 广播全局 `pending`。

> 手动启动（`UpdateSchedulerState("start", "")`）会显式置 `AutoClose=false`，
> 所以手动跑完不会触发关机/休眠。只有「启动即运行 / 调度器 cron」自动触发才允许结束后动作。

## 结束后动作（auto-action）

由 `settings` 的三个字段控制：

| 字段 | 取值 | 含义 |
| --- | --- | --- |
| `auto_action_trigger` | `scheduler_end` / `scheduled` | 动作时机 |
| `auto_action_type` | `close_app` / `hibernate` / `shutdown` / `none` | 动作类型 |
| `auto_action_cron` | cron 表达式 | `scheduled` 模式下到点执行 |

`DomReady` 据 `auto_action_type` 选 `closeFunc`（`utils.CloseApp`/`Hibernate`/`Shutdown`，`none` 则为 nil）：

- `trigger == "scheduled"` 且 cron 与 closeFunc 都非空：注册 cron job，到点时**若 `AutoClose` 为真**才执行 `closeFunc()`。
- `trigger == "scheduler_end"`：把 `closeFunc` 赋给 `scheduler.CloseFunc`，
  `StartAll` 结束时 `TriggerCloseFunc()` 在 `AutoClose` 为真时调用它。

关键约束：所有自动动作都受 `AutoClose` 门控，而 `AutoClose` 仅在「启动即运行 / 调度器 cron」自动触发时被置真；手动启动和实例级 cron 不置真，故不触发关机类动作。
