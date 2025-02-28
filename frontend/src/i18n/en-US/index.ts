export default {
  welcome: {
    title: 'Welcome',
    description:
      'Click the settings button in the bottom left corner to create your first instance',
  },
  tab: {
    home: 'Home',
    project: 'Project',
    general: 'General',
    update: 'Update',
  },
  home: {
    running: 'Running',
    waiting: 'Waiting',
    stopped: 'Stopped',
    logs: 'Logs',
  },
  general: {
    title: 'Basic Settings',
    language: 'Language',
    workDir: 'Working Directory',
    background: 'Background',
    configPath: 'Config Path',
    help: {
      language: 'The language displayed in this instance',
      workDir:
        'The working directory of the program, usually it should be the project root directory',
      background:
        'Whether it is a complete background program, not occupying screen, keyboard, mouse and other devices',
      configPath:
        'Where your program accesses the JSON configuration file, specific to the file name',
    },
  },
  update: {
    title: 'Update Settings',
    checkUpdates: 'Check Updates',
    upToDate: 'Up-To-Date',
    repoUrl: 'Repository URL',
    branch: 'Git branch',
    localPath: 'Local Path',
    templatePath: 'Template Path',
    autoUpdate: 'Auto Update',
    advancedSettings: 'Advanced Settings',
    envName: 'Environment Name',
    depsPath: 'Dependencies Path',
    pythonExec: 'Python Executable',
    help: {
      repoUrl: 'e.g., https://github.com/OwnerName/RepoName',
      branch: 'Leave empty for default branch',
      localPath: 'Path of the local repository directory',
      templatePath:
        'Path of the layout template directory relative to the repository root',
      autoUpdate: 'Changes will take effect after restart',
      envName:
        "Fill this in for the first time, click update to create a new Python virtual environment and install dependencies\nTo use this virtual environment, replace 'python' with 'py' in your commands, e.g.,'py main.py'",
      depsPath:
        'Where the requirements.txt file is located, default: ./requirements.txt',
      pythonExec:
        'Override default Python executable in PATH, leave empty to use the system default',
    },
  },
  custom: {
    title: 'Task Settings',
    active: 'Active',
    priority: 'Priority',
    command: 'Command',
    help: {
      active: 'Whether this task will be added to the task queue',
      priority: '0-31, higher number means higher priority',
      command: 'Command to execute this task',
    },
  },
  settings: {
    instance: 'Instance',
    general: 'General',
    about: 'About',

    createNew: 'Create New',
    fromLocal: 'From local',
    fromTemplate: 'From existing template',
    fromRemote: 'From remote',
    instanceName: 'Instance name',
    templateName: 'Template name',
    cannotDeleteTemplate: 'Cannot delete template in use',
    templatePath: 'Template path',
    repoUrl: 'Repository URL',
    localPath: 'Local path',
    branch: 'Branch',
    optional: 'optional',
    relativePath: 'Relative to repository root',
    create: 'Create',
    manage: 'Manage',
    selectAll: 'Select all',

    language: 'Language',

    homepage: 'Homepage',
    version: 'Version',
    license: 'License',
  },
};
