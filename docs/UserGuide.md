## Table of Contents
- [Create Instance](#create-instance)
  - [Create from Existing Template](#create-from-existing-template)
  - [Import from Local](#import-from-local)
  - [Pull from Remote Repository](#pull-from-remote-repository)
- [Global Settings](#global-settings)
- [Task Settings](#task-settings)
- [Start Running](#start-running)
- [Notes](#notes)

## Create Instance

### Create from Existing Template

1. Click the gear icon in the bottom left corner to enter the settings page.
2. In the "Add New Instance" section, select "From template" and choose the corresponding template.
3. Enter the instance name.
4. Click "Apply" in the bottom right corner.

### Import from Local

1. Click the gear icon in the bottom left corner to enter the settings page.
2. In the "Add New Instance" section, select "Import" and browse local files to select the directory containing the layout parameter file.
   > Note: Do not click into the directory. Currently, the file browser cannot go back to the previous level. If you go too deep, you will need to reselect.

3. Enter the instance name@template name.
4. Click "Apply" in the bottom right corner.

<img src="../static/images/guide1.gif" style="zoom:50%;" />

### Pull from Remote Repository

1. Follow the instructions in [Create from Existing Template](#create-from-existing-template) to create an instance of the "Init" template.
2. Fill in the settings on the "Update" page. If the target is a compliant Python project, you can fill in the Python-related settings.
3. Click "Check for Updates" in the top right corner.
4. Restart DaCapo after the update is complete.

## Global Settings

Enter the first task group's first task page, which corresponds to the "General" task. Pay special attention to the "Basic Settings" group.

<img src="../static/images/guide2.png" style="zoom:50%;" />

## Task Settings

Then, starting from the second task group, set all task items, paying special attention to the "Task Settings" group. The default priority is higher for smaller numbers, and changes will take effect on the next startup.

<img src="../static/images/guide3.png" style="zoom:50%;" />

## Start Running

After all settings are completed, return to the homepage and check if the order of tasks in the waiting queue is appropriate. If you want to adjust, you can manually click the task to move it to the termination queue. Tasks in the termination queue will no longer be executed.

Once everything is ready, click the start button on the right side of the "Run" card to start a single instance. If there are multiple instances, you can execute them all at once using the start button in the left column. In this case, all foreground instances will be executed sequentially, and background instances will be executed concurrently. Foreground and background refer to whether the tasks in the instance occupy screen, keyboard, mouse, and other devices, and whether they can be fully executed in the background.

You can observe the task execution status through the log panel, and the `dacapo.log` file will also record some coarse-grained information.

## Notes

1. Although the images shown here are in Chinese, don't worry, English is supported.

2. Do not create or delete instances, modify readiness status, or update from remote repositories while tasks are running. These operations will refresh the interface or require a restart, causing task execution to be interrupted.