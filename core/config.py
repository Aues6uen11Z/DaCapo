from functools import cached_property, lru_cache
import locale
from pathlib import Path
from typing import List, Tuple

from nicegui.storage import PersistentDict

from core.utils import read_json, write_json


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

    # List of tuples, each tuple contains a menu name and some tasks of this menu.
    @lru_cache
    def navbar_list(self, language: str) -> List[Tuple[str, List[str]]]:
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
    
    # List of task names, excluding the first menu which should be general settings, not tasks
    def task_list(self, language: str) -> list[str]:
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
    
    # Argument groups included in a task, task_name is not translated
    @lru_cache
    def group_dict(self, task_name: str) -> dict:
        group_dict = {}
        for menu, tasks in self.navbar_list('default'):
            if task_name in tasks:
                group_dict = self.args[menu][task_name]
                break
        return group_dict

    # IETF language tag, if not found i18n directory, return 'default'
    @property
    def default_language(self) -> str:
        i18n_path = Path(f'./config/templates/{self.name}/i18n')
        if not i18n_path.exists() or not list(i18n_path.glob('*.json')):
            return 'default'

        system_language = locale.getdefaultlocale()[0].replace('_', '-')  # Get the system language
        if system_language and (i18n_path / f'{system_language}.json').exists():
            return system_language
        else:
            return list(i18n_path.glob('*.json'))[0].stem
        
    def add_instance(self, instance_name: str) -> None:
        path = Path(f'./config/{instance_name}.json')
        path.touch()
        init_data = dict()
        tasks = self.task_list('default')
        tasks_dict = {task: {'priority': 1, 'command': ''} for task in tasks}
        init_data['_info'] = {
            'is_ready': True,
            'template': self.name,
            'language': self.default_language,
            'work_dir': '',
            'is_background': False,
            'config_path': '',
            'tasks': tasks_dict
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
    def is_background(self) -> bool:
        return self.storage['_info']['is_background']
    
    @property
    def config_path(self) -> str:
        return self.storage['_info']['config_path']
    
    def priority(self, task_name: str) -> int:
        return self.storage['_info']['tasks'][task_name]['priority']
    
    def command(self, task_name: str) -> str:
        return self.storage['_info']['tasks'][task_name]['command']

    def update_ready_status(self, status: bool) -> None:
        self.storage['_info']['is_ready'] = status
        write_json(self.path, self.storage)
