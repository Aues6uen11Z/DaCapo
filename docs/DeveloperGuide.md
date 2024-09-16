## Table of Contents
- [Quick Start](#quick-start)
  - [Customize Interface](#customize-interface)
  - [Load User Settings](#load-user-settings)

- [Advanced](#advanced)
  - [Multilingual and Help](#multilingual-and-help)
  - [Enable Updates](#enable-updates)
  - [Predefine Basic Option Groups](#predefine-basic-option-groups)

## Quick Start

### Custom Interface

You need to provide a configuration template, currently supporting configuration files in `JSON`/`YAML`/`TOML` formats (the latter two must be fully convertible to JSON). As of now, it should have the following structure, with only one layout parameter file:

```
----Template/		# Dirname is not important
    |----args.yaml
```

This file contains all the configuration content that needs to be generated, divided into four levels representing task groups, tasks, option groups, and options, respectively. Among them, **tasks** are the units of command execution.

The first task group is used for some overall settings, and it **must** include a "General" task. None of the tasks in the first task group will enter the execution queue; you should place the settings for the tasks to be executed in the subsequent task groups.

The `argument` option currently supports three types: `input` text box, `select` dropdown box, and `checkbox` checkbox.

> Note that DaCapo does not provide data validation for the content of the input text box; you need to handle possible exceptions in your own program.

<img src="../static/images/architecture.png" style="zoom: 50%;" />

After understanding the meaning of each level of elements, you can write your own layout arguments, where `type` and `value` refer to the type and default value, respectively.

```yaml
# args.yaml

Project:    # menu
    General:    # task
        Group1:     # group
            setting1:    # argument
                type: input
                value: something
            setting2:
                option:
                - option1
                - option2
                - option3
                type: select
                value: option1
            setting3:
                type: checkbox
                value: true
```

### Load User Settings

To allow users to freely create multiple instances, the modified settings are not saved in args.xxxx. The above layout arguments are just a template, and the specific content is derived from the instance configuration.

Your program should accept a **json** configuration file (note that YAML/TOML is not supported). This time, the task group is omitted, and the hierarchy is `Task` - `Group` - `Argument`, as follows:

```json
{
    "_info": {...},
    "General": {
        "Group1": {
            "setting1": "someting",
            "setting2": "option1",
            "setting3": true
        },
        "Group2": {
            "setting1": "someting",
        }
    },
    "Task1": {
        "Group3": {
            "setting1": "someting",
        }
    }
}
```

Here, the type, option list, and other information are removed, and it can be directly read as a multi-layer hash table (dictionary).

Next, you need to set an execution command for each task, which is also the way DaCapo actually calls the program: command line execution.

## Advanced

### Multilingual and Help

If you need to add multilingual support to your program or just write some explanations for the options, you need to add an i18n directory:

```
----Template/
    |----args.yaml
    |----i18n/
         |----zh-CN.yaml
         |----en-US.yaml
         |----......
```

Although it is strange, the translation files are organized as follows:

- The first layer includes `Menu`, `Task`, and all `Group`.

```yaml
# i18n/zh-CN.yaml

Menu: ...
Task: ...
Group1: ...
Group2: ...
Group3: ...
```

- The second layer, for Menu and Task, is the translation of task group names and task names.

```yaml
# i18n/zh-CN.yaml

Menu:
    Project:
        name: 总览
    Type1:
        name: 类型1
Task:
    General:
        name: 全局设置
    Task1:
        name: 任务1
```

For Group, it includes the translation of the Group (_info) and the translation of all its settings. If the setting is a dropdown box, all options are translated at the same level as name and help.

```yaml
# i18n/zh-CN.yaml

Group1:
    _info:
        help: 组1的帮助信息
        name: 组1
    setting1:
        help: 设置项1的帮助信息，input类型只需填写name和help，help也可省略
        name: 设置项1
    setting2:
        help: 设置项2的帮助信息，select类型需填写所有下拉框选项翻译
        name: 设置项2
        option1: 选项1
        option2: 选项2
        option3: 选项3
    setting3:
        help: 设置项3的帮助信息，checkbox类型只需填写name和help，help也可省略
        name: 设置项3
```

### Enable Updates

If your repository is a public repository on platforms like GitHub, and the project uses a language like Python that can be updated via source code, you can enable the update settings page. It's very simple, just add a line in the layout arguments file:

```yaml
# args.yaml

Project:
    General: {}
    Update: {}
```

If it happens to be a Python project, you can also let DaCapo manage the Python virtual environment. All you need to do is place a requirements.txt file specifying the list of dependencies in the root directory of the repository.

### Predefine Basic Option Groups

You may have noticed that some option groups appear on the interface that you didn't write in the layout parameter file. These option groups are called "basic option groups." If you need to predefine their content or even set them as unmodifiable to prevent novice users from accidentally changing them, you can edit them in args.xxxx.

The current basic options are:

- Project (the first task group)

  - General
      - language
      - work_dir: Working directory
      - work_dir_enabled: Set the editability of the above item, same below
      - is_background: Is it a background task
      - is_background_enabled
      - config_path: Configuration file path
      - config_path_enabled
  - Update
      - repo_url: Remote repository URL
      - repo_url_enabled
      - branch: Git repository branch
      - branch_enabled
      - local_path: Local storage path of the repository
      - local_path_enabled
      - template_path: Path of the layout arguments file relative to the root directory of the repository
      - template_path_enabled
      - auto_update: Enable automatic updates
      - env_name: Virtual environment name
      - pip_mirror: PyPI mirror source

  - Menu (other actual task groups)

    - Task (for all tasks)

        - priority: Task priority

        - priority_enabled

        - command: Command to execute the task

        - command_enabled

To modify these options, add a "_Base" option group under the corresponding task and directly fill in the values, for example:

```yaml
# args.yaml

Project:
    General:
        _Base:
            is_background: false
            is_background_enabled: false
```