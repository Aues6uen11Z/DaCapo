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
| `type` | string | 控件类型：`input`/`checkbox`/`dropdown`/`select`/`dir`/`file`/`priority` 等 |
| `value` | any | 默认值 |
| `help` | string | 帮助文本（可选） |
| `option` | []any | 下拉/单选选项（可选） |
| `hidden` | bool | 隐藏不显示 |
| `disabled` | bool | 只读 |

实例配置里 item **只存值**（无 type 等元数据），元数据运行时从模板取。

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
