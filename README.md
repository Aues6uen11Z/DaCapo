<div align="center">

  简体中文 | [English](docs/README_en.md)

  <a><img src="static/logo/logo.png" alt="Logo" width="150" height="150"></a>

  <h3 align="center">DaCapo</h3>

  <p align="center">
    一个配置文件为驱动的图形化脚本管理器
  </p>
</div>

<details>
  <summary>目录</summary>
  <ol>
    <li><a href="#简介">简介</a></li>
    <li><a href="#执行策略">执行策略</a></li>
    <li>
      <a href="#使用指南">使用指南</a>
      <ul>
        <li><a href="#前置准备">前置准备</a></li>
        <li><a href="#安装">安装</a></li>
        <li><a href="#添加配置">添加配置</a></li>
        <li><a href="#全局设置">全局设置</a></li>
        <li><a href="#任务设置">任务设置</a></li>
        <li><a href="#开始运行">开始运行</a></li>
      </ul>
    </li>
    <li><a href="#使用示例">使用示例</a></li>
    <li><a href="#致谢">致谢</a></li>
  </ol>
</details>

## 简介

本项目旨在为有繁杂用户配置的程序提供图形化界面，开发者无需编写额外代码，只需按特定要求提供和使用JSON配置文件，即可为自己的程序创建GUI。同时对于用户而言，可以将多个程序脚本集中管理，很适合一些需要周期性执行的任务。

<img src="static/images/show.png" style="zoom:50%;" />

## 执行策略

每个左侧栏的选项被称为一个实例，一个实例内部可以包含若干个任务，目前这些任务根据优先级顺序执行，没有自动管理调度。

有多个实例时可以一键执行，此时所有前台实例顺序执行，后台实例则并发执行。所谓前后台实例指的是这个实例包含的任务是否占用屏幕键鼠等设备，能否完全后台执行。

## 使用指南

### 前置准备

在开始使用前，你需要提供一个配置模板，里面包含了需要生成的内容。在一个配置目录中，必须包含`args.json`，`i18n`则可以省略。具体来说，它应该是这样的结构：

```
----Template/		# 该命名不重要
    |----args.json
    |----i18n/
         |----zh-CN.json
         |----en-US.json
         |----......
```

#### args.json

该文件包含所有需要生成的配置内容，一共分为4级，分别代表任务组、任务、选项组、选项，其中**任务**是命令执行的单位。

第一个任务组用来做一些总体设置，其中**必须**包含一个"General"任务，而第一个任务组的所有任务都不会进入执行队列，你应该把要执行的任务的设置项放到后面的任务组中。

选项`argument`目前支持3种类型，分别是`input`输入框，`select`下拉框以及`checkbox`复选框。

> 注意，DaCapo不提供对input输入框内容的数据校验，你需要在自己的程序中处理可能的异常。

```
menu
├── task
│   ├── group
│   │   ├── argument
│   │   │   ├── type: "select"
│   │   │   ├── value: "example"
│   │   │   ├── option: ["this", "is", "an", "example"]
│   │   │   └── ...
```

<img src="static/images/architecture.png" style="zoom: 50%;" />

#### i18n

这个目录下包含参数翻译及帮助信息，如果你只提供一种语言，但也想写一些帮助信息，可以创建一个该类json文件，把`name`留空只填写`help`。每个语种的json文件内是这样组织的：

- 第一层分别是Menu任务组、Task任务、以及所有Group选项组。

<img src="static/images/trans1.png" style="zoom:50%;" />

- 第二层，对于Menu和Task来说是是任务组名、任务名的翻译。

<img src="static/images/trans2.png" style="zoom:50%;" />

而对于Group来说既包含了该Group的翻译（_info）也包含了其下所有设置项的翻译，若设置项为下拉框，则直接在name和help同级翻译所有选项。

<img src="static/images/trans3.png" style="zoom:50%;" />



然后， 你的程序应该接收一个json配置文件，这次省略了任务组，层级为`Task`任务-`Group`选项组-`Argument`选项，形式如下：

<img src="static/images/config.png" style="zoom:50%;" />



### 安装

#### 获取发布版

你可以到[这里](https://github.com/Aues6uen11Z/DaCapo/releases)下载最新的发布版，目前仅支持Windows系统，解压后点击DaCapo.exe即可运行。

#### 从源码构建

新建Python3.6以上版本虚拟环境，安装`requirements.txt`的依赖包，执行`main.py`即可，应该没什么坑。



### 添加配置

打开程序后，首先点击齿轮图标进入设置页面，在“添加新实例”处选择”导入“，选择[前置准备](#前置准备)一节提到的模板目录。

> 注意不要点进该目录，目前文件浏览器无法返回上一级，点过头了只能重新选

然后输入新配置名，格式为实例名@模板名，最后点击应用。

> 同一个模板可以创建多个实例，已经导入过的模板下次可以选择“从已有模板创建”

<img src="static/images/guide1.gif" style="zoom:50%;" />



### 全局设置

进入全局设置页面，也就是“General”任务对应的页面，重点注意“基本设置”一组。

<img src="static/images/guide2.png" style="zoom:50%;" />



### 任务设置

随后从第二个任务组开始，设置所有任务项，重点注意“任务设置”组。其中默认优先级数字越小优先级越高，修改将在下次启动时生效。

<img src="static/images/guide3.png" style="zoom:50%;" />



### 开始运行

所有设置完成后回到主页，检查等待队列中任务的顺序是否合适，若还想调整可以手动点击任务将其移动到终止队列，终止队列的任务将不再执行。

一切就绪后点击“运行”卡片右侧的开始按钮将开始单个实例，点击左栏开始按钮将开始所有实例任务。

任务执行情况可以通过日志面板观察，`dacapo.log`文件也会记录一些粗粒度信息。

## 使用示例

三言两语可能难以表达清楚，建议结合[示例](https://github.com/Aues6uen11Z/DaCapo/tree/master/examples/SimpleScript)理解，若仍有疑问欢迎在Issue中提问，也欢迎PR来补充文档。

## 致谢

[NiceGUI](https://github.com/zauberzeug/nicegui)：本项目使用的GUI库，功能灵活强大，维护者十分友善且回复极快，社区也非常活跃。

[niceguiToolkit](https://github.com/CrystalWindSnake/nicegui-toolkit)：一个NiceGUI辅助工具，作者也是很乐于助人，其教程和答疑让我受益匪浅，B站/微信公众号同名：数据大宇宙，学NiceGUI找他就对了！

[SRC](https://github.com/LmeSzinc/StarRailCopilot)/[ALAS](https://github.com/LmeSzinc/AzurLaneAutoScript)：一切的开始，本项目仿照了其页面布局和配置文件方式。

[Nuitka](https://github.com/Nuitka/Nuitka)：本项目使用的打包工具。