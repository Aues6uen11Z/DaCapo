from functools import cached_property, lru_cache
import locale
from pathlib import Path
from typing import Dict, List, Tuple

from nicegui.storage import PersistentDict

from src.core.utils import read_json, write_json


class TemplateConfig:
    """Read-only configuration for a template.

    Hierarchical relationship:
    menu
    ├── task
    │   ├── group
    │   │   ├── argument
    │   │   │   ├── type: "select"
    │   │   │   ├── value: "example"
    │   │   │   ├── option: ["this", "is", "an", "example"]
    │   │   │   └── ...

    NOTE Mandatory requirements:
    1. First menu can not contain actual tasks, only general settings.
    2. First menu must contain a task named "General".
    """

    def __init__(self, name: str):
        self.name = name
        self.args_path = Path(f'./config/templates/{self.name}/args.json')
        self.i18n_path = Path(f'./config/templates/{self.name}/i18n')

    @cached_property
    def args(self) -> dict:
        return read_json(self.args_path)
    
    @cached_property
    def available_languages(self) -> List[str]:
        lang_list = [lang.stem for lang in self.i18n_path.glob('*.json')]
        lang_list.append('default')
        return lang_list
    
    @lru_cache
    def translation(self, language: str) -> dict:
        assert (self.i18n_path / f'{language}.json').exists(), f"Language {language} is not supported."
        return read_json(self.i18n_path / f'{language}.json')

    @lru_cache
    def navbar_list(self, language: str) -> List[Tuple[str, List[str]]]:
        """List of tuples, each tuple contains a menu name and some tasks of this menu."""
        menu_tasks_list = []
        menu_names = list(self.args.keys())
        for menu_name in menu_names:
            tasks = list(self.args[menu_name].keys())
            if language == 'default':
                menu_tasks_list.append((menu_name, tasks))
            else:
                menu_name = self.translation(language)['Menu'][menu_name]['name']
                tasks = [self.translation(language)['Task'].get(task, {}).get('name', task) for task in tasks]
                menu_tasks_list.append((menu_name, tasks))

        return menu_tasks_list
    
    def task_list(self, language: str) -> list[str]:
        """List of task names, excluding the first menu which should be general settings, not tasks."""
        unordered_list = []
        menu_names = list(self.args.keys())
        for menu_name in menu_names[1:]:
            for task, _ in self.args[menu_name].items():
                if language != 'default':
                    task_t = self.translation(language)['Task'][task]['name']
                    unordered_list.append(task_t)
                else:
                    unordered_list.append(task)
        
        return unordered_list
    
    @lru_cache
    def group_dict(self, task_name: str) -> dict:
        """Argument groups included in a task, task_name is not translated."""
        group_dict = {}
        for menu, tasks in self.navbar_list('default'):
            if task_name in tasks:
                group_dict = self.args[menu][task_name]
                break
        return group_dict

    @property
    def default_language(self) -> str:
        """IETF language tag, if not found i18n directory, return 'default'."""
        i18n_path = Path(f'./config/templates/{self.name}/i18n')
        if not i18n_path.exists() or not list(i18n_path.glob('*.json')):
            return 'default'

        system_language = locale.getdefaultlocale()[0].replace('_', '-')  # Get the system language
        if system_language and (i18n_path / f'{system_language}.json').exists():
            return system_language
        else:
            return list(i18n_path.glob('*.json'))[0].stem
    
    @property
    def _work_dir(self) -> Tuple[str, bool]:
        first_menu = list(self.args.keys())[0]
        if '_Base' not in self.args[first_menu]['General']:
            return '', True
        value = self.args[first_menu]['General']['_Base'].get('work_dir', '')
        enabled = self.args[first_menu]['General']['_Base'].get('work_dir_enabled', True)
        return value, enabled
    
    @property
    def _is_background(self) -> Tuple[bool, bool]:
        first_menu = list(self.args.keys())[0]
        if '_Base' not in self.args[first_menu]['General']:
            return False, True
        value = self.args[first_menu]['General']['_Base'].get('is_background', False)
        enabled = self.args[first_menu]['General']['_Base'].get('is_background_enabled', True)
        return value, enabled
    
    @property
    def _config_path(self) -> Tuple[str, bool]:
        first_menu = list(self.args.keys())[0]
        if '_Base' not in self.args[first_menu]['General']:
            return '', True
        value = self.args[first_menu]['General']['_Base'].get('config_path', '')
        enabled = self.args[first_menu]['General']['_Base'].get('config_path_enabled', True)
        return value, enabled
    
    @property
    def _tasks(self) -> Dict:
        tasks_list = dict()
        for menu, tasks in self.navbar_list('default')[1:]:
            for task in tasks:
                priority = self.args[menu][task].get('_Base', {}).get('priority', 1)
                priority_enabled = self.args[menu][task].get('_Base', {}).get('priority_enabled', True)
                command = self.args[menu][task].get('_Base', {}).get('command', '')
                command_enabled = self.args[menu][task].get('_Base', {}).get('command_enabled', True)
                tasks_list[task] = {
                    'priority': priority,
                    'priority_enabled': priority_enabled,
                    'command': command,
                    'command_enabled': command_enabled
                }
        return tasks_list
        
    def add_instance(self, instance_name: str) -> None:
        path = Path(f'./config/{instance_name}.json')
        path.touch()
        init_data = dict()
        init_data['_info'] = {
            'is_ready': True,
            'template': self.name,
            'language': self.default_language,
            'work_dir': self._work_dir[0],
            'work_dir_enabled': self._work_dir[1],
            'is_background': self._is_background[0],
            'is_background_enabled': self._is_background[1],
            'config_path': self._config_path[0],
            'config_path_enabled': self._config_path[1],
            'tasks': self._tasks
            }
        write_json(path, init_data)


class InstanceConfig:
    """Configuration for a specific instance, necessary for running GUI and tasks."""

    def __init__(self, name):
        self.name = name
        self.path = Path(f'./config/{self.name}.json').resolve()
        self.storage = PersistentDict(self.path, encoding='utf-8')

    @property
    def is_ready(self) -> bool:
        return self.storage['_info']['is_ready']
    
    @property
    def template(self) -> str:
        return self.storage['_info']['template']

    @property
    def language(self) -> str:
        return self.storage['_info']['language']
    
    @property
    def work_dir(self) -> str:
        return self.storage['_info']['work_dir']
    
    @property
    def work_dir_enabled(self) -> bool:
        return self.storage['_info']['work_dir_enabled']

    @property
    def is_background(self) -> bool:
        return self.storage['_info']['is_background']
    
    @property
    def is_background_enabled(self) -> bool:
        return self.storage['_info']['is_background_enabled']
    
    @property
    def config_path(self) -> str:
        return self.storage['_info']['config_path']
    
    @property
    def config_path_enabled(self) -> bool:
        return self.storage['_info']['config_path_enabled']
    
    def priority(self, task_name: str) -> int:
        return self.storage['_info']['tasks'][task_name]['priority']
    
    def priority_enabled(self, task_name: str) -> bool:
        return self.storage['_info']['tasks'][task_name]['priority_enabled']
    
    def command(self, task_name: str) -> str:
        return self.storage['_info']['tasks'][task_name]['command']
    
    def command_enabled(self, task_name: str) -> bool:
        return self.storage['_info']['tasks'][task_name]['command_enabled']

    def update_ready_status(self, status: bool) -> None:
        self.storage['_info']['is_ready'] = status
        write_json(self.path, self.storage)
