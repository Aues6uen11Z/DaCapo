# 实例生命周期

> 相关文件：`backend/service/scheduler.go`、`backend/model/scheduler.go`、`backend/model/instance_info.go`、`backend/service/instance.go`

## 状态机

实例状态是字符串常量（`backend/model/scheduler.go`）：

| 状态 | 含义 |
| --- | --- |
| `pending` | 空闲，可启动 |
| `running` | 正在执行任务 |
| `updating` | 仓库/环境更新中，禁止启动 |
| `failed` | 上次执行出错，需手动重启 |

注意：`ready` 不是状态，而是 `InstanceInfo.Ready` 布尔字段（实例是否配置完成、可参与调度）。
`waiting` 也不是状态，而是 `TaskQueue.Waiting` 任务列表。

状态流转：

```
pending --StartOne--> running --正常结束--> pending
running --出错--> failed
failed  --手动启动--> running   (startOne 允许重启 failed 实例)
任意    --触发更新--> updating --更新完成--> pending/failed
任意    --手动停止--> pending
```

## 运行时结构

每个实例有一个 `TaskManager`（`model/scheduler.go`），由全局 `Scheduler.TaskManagers` map 持有：

```go
type TaskManager struct {
    InstanceName string
    Status       string          // pending/running/updating/failed
    Queue        TaskQueue       // Running + Waiting[] + Stopped[]
    Cmd          *exec.Cmd       // 当前子进程，用于手动 kill
    ManualStop   bool            // 手动停止标记
    LastError    string          // 失败原因，用于通知和重启校验
}
```

`Scheduler`（单例，`model.GetScheduler()`）还有 `IsRunning`、`CronExpr`、`AutoClose`、`CloseFunc`。
`SyncWithDatabase()` 会让 `TaskManagers` 与数据库中的实例保持一致（新增/移除）。

## 启动单个实例：StartOne

`SchedulerService.StartOne(name)` -> `startOne(name)`（`scheduler.go`）：

1. `validateTaskManager(name, allowFailed=true)` - 失败实例也允许重启。
2. 若 `Status == updating` 直接返回，不启动。
3. 清空 `LastError`，置 `running`，广播状态与队列。
4. 循环 `tm.SwitchRun()` 从 `Waiting` 取下一个任务：
   - 任务命令若以 `py ` 开头，替换为虚拟环境的 python（`utils.GetVenvPython`）。
   - `RunCommand(tm, cmd, workDir)` 执行，stdout/stderr 流式处理后广播为 `log`。
   - 出错 -> `stopOne(name, err)` 置 `failed`，存 `LastError`，广播错误日志，返回。
   - 手动停止（`ErrManualStop`）-> 直接返回，不改状态（停止状态由 stopOne 的手动分支处理）。
5. 队列清空后置 `pending`。

## 停止：两条路径

- **手动停止**：`UpdateSchedulerState("stop", name)` -> `stopOne(name, nil)`（err 为 nil）。
  `CancelTask` kill 子进程并置 `ManualStop=true`，状态置 `pending`，`RemoveRun` 把当前任务移入 `Stopped`。
- **任务失败**：`startOne` 内 `failWithError` -> `stopOne(name, err)`（err 非 nil）。
  状态置 `failed`，记录 `LastError`，广播日志。

## 全部启动：StartAll

调度级编排（前后台分组、并发限流、结束后动作）见 [scheduler.md](./scheduler.md)。
单实例执行机制（上文 StartOne/stopOne）本文已覆盖，二者不重复。