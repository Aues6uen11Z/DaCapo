# 实例更新流程

> 相关文件：`backend/service/instance_updater.go`、`backend/controller/instance_updater_ctrl.go`、`backend/utils/tools.go`（`GitPull`/`GetUVExecutable`/`GetVenvPython`）、`backend/utils/symlink.go`、`backend/model/template_conf.go`、`backend/model/instance_conf.go`

入口：`GET /api/updater/:instance_name` -> `controller.UpdateRepo` ->
`InstanceUpdaterService.UpdateRepo(name)`。返回 `RspUpdateRepo{Code, Message, Detail, IsUpdated}`。

## 前置检查

取实例与 `TaskManager`。若 `Status == running` 直接拒绝（`StatusBusy`，code 1007）。
否则置 `StatusUpdating` 并广播。

## 1. 拉取代码

1. `utils.RemoveLink(configPath)` - 先移除配置文件符号链接，避免 git 操作误伤。
2. `utils.GitPull(localPath)`：
   - `git pull origin`；若 "Already up to date" 直接返回。
   - 冲突时 `git merge --abort` -> `git fetch origin` -> `git reset --hard @{u}`（强制对齐远程分支）。
3. `utils.CheckLink(instances/<name>.json -> configPath)` - 重建符号链接。
4. 拉取失败 -> `StatusFailed`，返回 `StatusGit`（code 1004）。

## 2. Python 环境（仅 EnvName 非空）

`envPath = ./envs/<EnvName>`。

- `createEnv(tm, envPath, pythonVersion)`：venv 不存在才建。
  `GetUVExecutable()` 取 uv（系统 -> 打包 `tools/uv` -> 联网下载，仅 Windows 自动下载），
  有则 `uv venv --python <ver> <envPath>`；否则回退 `python -m venv <envPath>`。
  通过 `schedulerService.RunCommand` 执行，输出实时广播为日志。
- 环境失败 -> `StatusFailed`，返回 `StatusPython`（code 1005）。

## 3. 安装依赖（仅 DepsPath 非空）

`depsPath = localPath/<DepsPath>`（默认 `requirements.txt`）。

- **跳过条件**：`EnvLastUpdate` 非零且 `requirements` 文件 mtime 早于它 -> 视为未变更，跳过。
- `getPyPIMirror()`：并发请求清华/阿里/USTC/PyPI 四个镜像，最先完整响应的胜出。
- 安装命令：有 uv 则 `uv pip install -r <deps> --python <env> -i <mirror>`；
  否则回退 venv 内 `python -m pip install -r <deps> -i <mirror>`。
- 成功后 `UpdateField("env_last_update", now)`。
- 失败 -> `StatusFailed`，返回 `StatusPython`（code 1005）。

## 4. 模板重解析

取 `TemplateInfo`，定位 `template.{yml,yaml,json}`，`os.Stat` 取 mtime：

- 若 `tplFile.ModTime().After(LayoutLastUpdate)`：模板变了。
  `UpdateField("layout_last_update", now)`，`syncIstConf`（重载实例配置 + 模板，
  `InstanceConf.Update(tpl)` 合并：保留已有值、新字段用模板默认），
  置 `StatusPending`，返回 `IsUpdated=true`。
- 否则无变化，置 `StatusPending`，返回成功。

## 状态码

| Code | 常量 | 含义 |
| --- | --- | --- |
| 0 | `StatusSuccess` | 成功 |
| 1001 | `StatusFile` | 文件操作失败 |
| 1004 | `StatusGit` | Git 操作失败 |
| 1005 | `StatusPython` | Python 环境/依赖失败 |
| 1007 | `StatusBusy` | 实例运行中，拒绝更新 |

> 任何步骤出错都先置 `StatusFailed` 再返回对应码，前端据 `code`/`detail` 展示。
