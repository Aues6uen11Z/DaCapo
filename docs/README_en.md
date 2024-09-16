<div align="center">

  [简体中文](../README.md) | English

  <a><img src="../static/logo/logo.png" alt="Logo" width="150" height="150"></a>

  <h3 align="center">DaCapo</h3>

  <p align="center">
    A graphical script manager driven by configuration files
  </p>
</div>

## Introduction

This project aims to provide a graphical interface for programs with complex user configurations. Developers do not need to write additional code, but only need to provide and use configuration files according to specific requirements to create a GUI for their programs. At the same time, for users, it allows for centralized management of multiple program scripts, which is very suitable for tasks that require periodic execution.

<img src="../static/images/show_en.png" style="zoom:50%;" />

## Highlights

- Generate GUI from configuration files in JSON/YAML/TOML formats
- Manage and run multiple task instances with one click
- Automatically pull code from remote repositories and create interfaces
- Automatically manage Python virtual environments and update dependencies
- Support for multiple languages

## Guide

How to make my program compatible with DaCapo? 👉 [Developer Guide](./DeveloperGuide.md)

How to use DaCapo? 👉 [User Guide](./UserGuide.md)

**Examples:**

1. [SimpleScript](./examples/SimpleScript): As the name suggests, a simple introductory example
2. [HonkaiHelper](https://github.com/Aues6uen11Z/HonkaiHelper): Automation script for Honkai Impact 3rd

## Installation

#### Obtain the Release Version

You can download the latest release version [here](https://github.com/Aues6uen11Z/DaCapo/releases). Currently, only Windows systems are supported. After extracting the files, click on DaCapo.exe to run the application.

#### Building from Source Code
Create a Python virtual environment with version 3.6 or higher, install the dependencies listed in `requirements.txt`, and then execute `main.py`. There shouldn’t be any major issues.

## Acknowledgements

[NiceGUI](https://github.com/zauberzeug/nicegui)：The GUI library used in this project, which is versatile and powerful. The maintainers are very friendly and respond quickly, and the community is also very active.

[niceguiToolkit](https://github.com/CrystalWindSnake/nicegui-toolkit)：An auxiliary tool for NiceGUI, the author of which is also very helpful. His tutorials and answers to questions have been very beneficial to me.

[SRC](https://github.com/LmeSzinc/StarRailCopilot)/[ALAS](https://github.com/LmeSzinc/AzurLaneAutoScript)：Where it all began, this project imitates their page layout and configuration file approach.

[Nuitka](https://github.com/Nuitka/Nuitka)：The packaging tool used in this project.