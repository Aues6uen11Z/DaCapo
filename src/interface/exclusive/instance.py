from typing import List, Optional

from nicegui import ui

from src.core.config import InstanceConfig, TemplateConfig
from src.core.scheduler import TaskManager
from src.interface.exclusive.general import General
from src.interface.exclusive.home import Home
from src.interface.exclusive.custom import Custom
from src.interface.exclusive.update import Update
from src.utils import get_text


_ = get_text()


class Instance:

    def __init__(self, name: str):
        self.name = name
        self.ist_config = InstanceConfig(self.name)
        self.tpl_config = TemplateConfig(self.ist_config.template)

        self.nav_tabs: Optional[ui.tabs] = None
        self.home_tab: Optional[ui.tab] = None
        self.custom_tabs: Optional[List[ui.tab]] = None
        self.task_manager: Optional[TaskManager] = None
    
    def navbar(self) -> None:
        """navbar, middle part"""
        language = self.ist_config.language
        with ui.card().classes('w-48 p-0 gap-0'):
            custom_tabs = []
            with ui.tabs().props('vertical dense').classes('w-full') as nav_tabs:
                home_tab = ui.tab(_('主页'), icon='home')

                ori_navbar_list = self.tpl_config.navbar_list('default')    # Get the original navbar list
                for i, (menu, tasks_t) in enumerate(self.tpl_config.navbar_list(language)):
                    with ui.expansion(menu, value=True) \
                            .props('dense expand-separator').classes('w-full font-medium uppercase'):
                        with ui.column().classes('w-full my-[-15px] gap-0'):
                            for j, task_t in enumerate(tasks_t):
                                task = ori_navbar_list[i][1][j]
                                tab = ui.tab(task, label=task_t).classes('justify-start').props('no-caps')
                                custom_tabs.append(tab)
        self.nav_tabs = nav_tabs
        self.home_tab = home_tab
        self.custom_tabs = custom_tabs

    def content(self) -> None:
        """sub content, right part"""
        language = self.ist_config.language
        with ui.tab_panels(self.nav_tabs, value=self.home_tab).classes('w-full h-full'):
            with ui.tab_panel(self.home_tab).classes('p-0.5'):
                task_list = self.tpl_config.task_list('default')
                if language == 'default':
                    task_list_t = task_list
                else:
                    task_list_t = self.tpl_config.task_list(language)    # translated
                home = Home(self.ist_config, task_list, task_list_t)
                self.task_manager = TaskManager(self.ist_config, home)
                home.add_callback(self.task_manager.run, self.task_manager.stop)
                home.show()
            
            for tab in self.custom_tabs:
                with ui.tab_panel(tab).classes('p-0'):
                    tab_name = tab._props['name']   # not translated
                    if tab_name == 'General':
                        General(tab_name, self.ist_config, self.tpl_config).show()
                    elif tab_name == 'Update':
                        Update(self.ist_config, self.task_manager.update).show()
                    else:
                        Custom(tab_name, self.ist_config, self.tpl_config).show()
    
    def show(self) -> TaskManager:
        with ui.tab_panel(self.name).classes('w-full h-full p-2'):
            with ui.row(wrap=False).classes('w-full h-full'):
                self.navbar()
                self.content()
        return self.task_manager


if __name__ == '__main__':
    # from niceguiToolkit.layout import inject_layout_tool
    # inject_layout_tool()

    ui.add_css(".nicegui-log > *{ white-space: normal; word-break: break-all; }")
    Instance('崩铁').show()
    ui.run(window_size=(1200, 800), reload=False)
