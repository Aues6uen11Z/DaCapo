# 配置文件参考

> 相关文件：`backend/model/template_conf.go`、`backend/model/instance_conf.go`、`backend/model/instance_info.go`、`backend/model/template_info.go`
>
> 本文是内部结构参考。第三方作者如何编写模板，见 [`开发者指南.md`](../开发者指南.md)。

## 概念：模板 vs 实例

- **模板**（`template.{yml,yaml,json}`）：定义配置**结构**（有哪些项、类型、默认值）。
  存于模板目录，路径记录在 `TemplateInfo.Path`。
- **实例配置**（`instances/<name>.json`）：只存**值**，结构由模板派生。
- **数据库**（`InstanceInfo`/`TaskInfo`）：存 DaCapo 内置设置（与具体模板无关），如运行顺序、cron、环境名等。

## 四层有序结构

模板与实例配置都是 **Menu -> Task -> Group -> Item** 四层有序 map（`orderedmap`，保插入顺序）：

```
Menu(菜单)   例如 General / Update / 自定义菜单
 └ Task(任务) 例如 task1 / task2
    └ Group(分组) 例如 _Base / 自定义分组
       └ Item(配置项) 例如 api_key / work_dir
```

## ItemConf 字段

模板里每个 item 是 `ItemConf`（`template_conf.go`）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `type` | string | 控件类型，见下表 |
| `value` | any | 默认值 |
| `help` | string | 帮助文本（可选） |
| `option` | []any | 下拉/单选选项（可选） |
| `hidden` | bool | 隐藏不显示 |
| `disabled` | bool | 只读 |

实例配置里 item **只存值**（无 type 等元数据），元数据运行时从模板取。

## 控件类型

`type` 决定前端渲染的控件（`frontend/src/components/ItemLine.vue`）：

| type | 控件 | 说明 |
| --- | --- | --- |
| `input` | 文本框 | 未识别的 type 也回退到此 |
| `select` | 下拉框 | 需配合 `option` |
| `checkbox` | 复选框 | 布尔值 |
| `priority` | 数字框 | 0-31，任务优先级专用 |
| `folder` | 目录选择框 | 文本框 + 选目录按钮 |
| `file` | 文件选择框 | 文本框 + 选文件按钮 |
| `cron` | cron 输入框 | 失焦校验表达式 |

`priority`、`cron` 为内置项类型（分别用于 `_Base.priority`、`General.cron_expr`），自定义项一般使用其余 5 种。

## 保留名称

| 名称 | 层级 | 含义 |
| --- | --- | --- |
| `Project` | Menu | 布局构建时跳过（`instance.go` GetLayout） |
| `General` | Menu | 内置菜单，项由 `InstanceInfo` 字段承载 |
| `Update` | Menu | 内置菜单，项由 `InstanceInfo` 字段承载 |
| `_Base` | Group | 每任务内置分组（active/priority/command），由 DB `TaskInfo` 注入；模板里的 `_Base` 被忽略 |
| `_help` | Item | 帮助文本项，存值时跳过 |

## 内置项

**General 菜单**（映射到 `InstanceInfo`）：`language`、`work_dir`、`background`、
`config_path`、`log_path`、`cron_expr`。

**Update 菜单**（映射到 `InstanceInfo`）：`branch`、`auto_update`、`env_name`、
`deps_path`、`python_version`；另有两个只读项 `local_path`、`template_rel_path`。

**任务 `_Base` 分组**（映射到 `TaskInfo`）：`active`、`priority`、`command`。
任务命令以 `py ` 开头时，运行时替换为虚拟环境里的 python（见 [instance.md](./backend/instance.md)）。

## 配置流转

1. 模板解析为 `TemplateConf`（`TemplateConf.Load`，按扩展名选 yaml/json）。
2. 创建实例：`InstanceConf.Create` 从模板生成 `instances/<name>.json`，item 取模板默认值（跳过 `_Base`/`_help`）。
3. 用户在前端改值 -> `InstanceConf.SetValue` 写回 json（经符号链接也写到仓库内 `config_path`，供脚本读取）。
4. 模板变更（更新流程）：`InstanceConf.Update(tpl)` 合并——**保留已有值**，新字段用模板默认。

> 符号链接：`instances/<name>.json` -> 仓库内 `config_path`，让脚本直接读自己仓库里的配置文件。
> 更新流程会先 `RemoveLink` 再 `CheckLink` 重建（见 [updater.md](./backend/updater.md)）。

### 模板/配置损坏时的兜底

`GetSingleInstance` 加载模板或实例配置失败时**不报错**，而是用空模板返回仅含 `Project` 菜单的兜底布局
（General/Update 项全部来自数据库，不依赖模板解析），实例保持可见、更新入口可用；
错误原因通过响应 `errors`（实例名 -> 原因）透出，前端在实例页顶部显示横幅。
模板修复后重新拉取更新（`UpdatePage` 成功后重载布局）即可恢复完整页面。
