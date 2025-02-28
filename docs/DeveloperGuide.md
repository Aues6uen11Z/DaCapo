## Table of Contents

- [Quick Start](#quick-start)

  - [Customize Interface](#customize-interface)
  - [Read User Settings](#read-user-settings)

- [Advanced](#advanced)
  - [Multilingual Support](#multilingual-support)
  - [Remote Repository Updates](#remote-repository-updates)
  - [Predefined Basic Setting Groups](#predefined-basic-setting-groups)

## Quick Start

### Customize Interface

You need to provide a configuration template, currently supporting `JSON`/`YAML` formats, and the file name must be "template":

```
----Template/		# This naming is not important
    |----template.yml
```

This file contains all the configuration content that needs to be generated, divided into 4 levels representing task groups (Menu), tasks, setting groups (Group), and settings (Item), where **tasks** are the units of command execution.

<img src="./images/1.png" style="zoom: 50%;" />

The first task group "Project" is special, it has a fixed structure, and its tasks "General" and "Update" will not enter the execution queue. You should put the settings for tasks to be executed in the subsequent task groups. You can add custom option groups to the "General" task, like this:

```yaml
Project:
  General:
    Group1:
      _help: # Special Item, used to display setting group help information
        value: This group shows 4 types of setting
      input:
        type: input
        value: someting
        help: type "input" shows a text box
```

To generate a setting, just fill in its information in the template file, including:

- type: One of the 5 types - input box, select dropdown, checkbox, folder input, file input
- value: Default value
- help: Help information
- option: Options, only effective when type is select
- hidden: Whether to hide this setting item, when all Items in a Group are hidden, the Group will also be hidden
- disabled: Whether it is non-editable

> Note that DaCapo does not provide validation for input content, you need to handle possible exceptions in your own program.

Organize your template freely according to the Menu-Task-Group-Item structure, and you will get the corresponding page. You can refer to [this repository](https://github.com/Aues6uen11Z/DaCapoExample) for details. A simple example is as follows:

```yaml
Menu:
  Task1:
    Group2:
      setting1:
        value: ""
        type: input
        help: settings can be disabled
        disabled: true
```

<img src="./images/2.png" style="zoom:50%;" />

### Read User Settings

To allow users to freely create multiple instances, the modified settings are not saved in `template.xxxx`. The above layout parameters are just a template, and the specific content is derived from the instance configuration.

Your program should accept a **json** configuration file (note that YAML/TOML is not supported), still in a four-layer structure, but with type, option list, and other information removed, which can be directly read as a multi-layer hash table (dictionary):

```json
{
  "Project": {
    "General": {
      "Group1": {
        "input": "someting",
        "select": "option1",
        "checkbox": true,
        "folder": "./repos/DaCapoExample",
        "file": "./repos/DaCapoExample/template/template.yml"
      }
    }
  },
  "Menu": {
    "Task1": {
      "Group2": {
        "setting1": "1",
        "setting2": "2",
        "setting3": "3"
      }
    },
    "Task2": {
      "Group3": {
        "setting1": "4",
        "setting2": "5"
      }
    },
    "Task3": {
      "Group4": {
        "setting1": "6"
      }
    }
  }
}
```

Next, you need to set an execution command for each task, which is also the way DaCapo actually calls programs—command line execution, so be very careful about the safety of the command itself.

## Advanced

### Multilingual Support

If you need to add multilingual support to your program, you need to add an `i18n` directory to store translation json files:

```
----Template/
    |----template.yaml
    |----i18n/
         |----中文.json
         |----English.json
         |----......
```

Translation files are organized like this:

```json
{
  "Menu": {
    "name": "菜单",
    "tasks": {
      "Task1": {
        "groups": {
          "Group2": {
            "help": "",
            "items": {
              "setting1": {
                "help": "设置可以被禁用",
                "name": "设置1"
              },
......
```

Although it looks complicated, don't worry, if you're too lazy to write it manually, you can easily use a [python script](https://github.com/Aues6uen11Z/DaCapoExample/blob/master/gen_i18n.py) to export translation files from the template file.

Finally, seamlessly switch the display language through the language settings on the interface

<img src="./images/3.png" style="zoom:50%;" />

### Remote Repository Updates

If your repository is a public repository hosted on platforms like Github, and the project uses a language like Python that can be updated via source code, you can choose to create an instance from remote, and an update interface will automatically appear after creation.

![image-20250227194918354](./images/4.png)

<img src="./images/5.png" style="zoom:50%;" />

If the project happens to use Python, you can also set the virtual environment name and dependency file location. Clicking update will automatically create a virtual environment and install dependencies. By default, it uses the Python pointed to by system variables; if a different version is needed, you can fill in the specific path to the Python executable.

### Predefined Basic Setting Groups

You may have noticed that there are option groups on the interface that you didn't write in the layout parameter file. These option groups are called "basic option groups." If you need to predefine their content or even make them non-modifiable to prevent novice users from accidentally changing them, you can edit them in the template file.

Currently, the basic options are:

- Project (the first task group)

  - General
    - language: Language
    - work_dir: Working directory
    - background: Whether it is a background task
    - config_path: Configuration file path, the specific settings of the instance are saved in the root directory's instances directory, modifying this item will create a symbolic link to config_path
  - Update
    - auto_update: Whether to enable automatic updates
    - env_name: Virtual environment name
    - deps_path: Path of the Python dependency file relative to the repository root directory
    - python_exec: Python executable file used to create the virtual environment, mainly used when there are requirements for Python version

- Menu (other actual task groups)

  - Task (for all tasks)
    - active: Whether to enable, determines whether this task is added to the waiting queue at startup
    - priority: Task priority, determines the position of this task in the waiting queue at startup
    - command: Command to execute this task

To modify these options, add a "\_Base" option group under the corresponding task, for example:

```yaml
# template.yaml

Project:
  General:
    _Base:
      work_dir:
        value: ./repos/DaCapoExample
        disabled: true
```
