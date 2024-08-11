from typing import List, Optional
from loguru import logger
from nicegui import context, ui

from src.core.config import InstanceConfig
from src.core.scheduler import Scheduler, TaskManager
from src.core.utils import instance_list
from src.interface.common.settings import Settings
from src.interface.exclusive.instance import Instance
from src.interface.utils import get_text

_ = get_text()


class DaCapoUI:

    def __init__(self):
        self.HEADER_HEIGHT = 0
        # self.header = Header(self.HEADER_HEIGHT)
        self.settings_page = Settings([self.sidebar, self.content])
        self.scheduler = Scheduler()
        self.settings_dialog: Optional[ui.dialog] = None
        self.drawer_tabs: Optional[ui.tabs] = None
        self.tab_list: Optional[List[ui.tab]] = None
        self.ist_tasks: List[TaskManager] = []
        self.start_btn: Optional[ui.button] = None
        self.stop_btn: Optional[ui.button] = None

        context.client.content.classes('py-0')
        ui.scroll_area.default_props('content-style="padding:0; gap:6px" content-active-style="padding:0; gap:6px"')
        ui.add_css(".nicegui-log > *{ white-space: normal; word-break: break-all; }")
        ui.add_css(".multi-line-notification { white-space: pre-line; }")

    async def on_start(self):
        for ist_task in self.ist_tasks:
            if ist_task.ist_config.is_ready and ist_task.status == 'standby':
                if ist_task.ist_config.is_background:
                    self.scheduler.add_background_task(ist_task)
                else:
                    self.scheduler.add_foreground_task(ist_task)
        await self.scheduler.run()

    def on_stop(self):
        # Cancel the task to be executed
        self.scheduler.clear_tasks()
        # Stop the running task
        for ist_task in self.ist_tasks:
            if ist_task.status == 'running':
                ist_task.stop()
        logger.info(f'Scheduler stopped manually')

    @ui.refreshable
    def sidebar(self) -> None:
        """Select different instances"""
        tab_list = []
        with ui.tabs().props('vertical indicator-color="transparent" active-color="primary"') \
                .classes('w-full h-4/5 text-violet-400') as drawer_tabs:

            for instance in instance_list():
                ready = InstanceConfig(instance).is_ready
                if ready:
                    tab = ui.tab(instance, icon='rocket_launch').props('no-caps')
                else:
                    tab = ui.tab(instance, icon='rocket').props('no-caps')
                tab_list.append(tab)

        with ui.column().classes('w-full'):
            self.start_btn = ui.button(icon='play_arrow').props(
                'push color="white" text-color="primary" round').classes('self-center')
            self.start_btn.on_click(self.on_start)
            self.stop_btn = ui.button(icon='stop').props('push color="white" text-color="primary" round').classes(
                'self-center')
            self.stop_btn.set_visibility(False)
            self.stop_btn.on_click(self.on_stop)
            ui.button(icon='settings', on_click=self.settings_dialog.open) \
                .props('push color="white" text-color="primary" round').classes('self-center mb-3')

        self.drawer_tabs = drawer_tabs
        self.tab_list = tab_list

    def set_status(self, tab: ui.tab, status: str):
        # left drawer tabs
        if status == 'running':
            tab.props('alert="green"')
        elif status == 'error':
            tab.props('alert="red"')
        else:
            tab.props(remove='alert')

        # start/stop button, under the tabs
        if any(task.status == 'running' for task in self.ist_tasks):
            self.start_btn.set_visibility(False)
            self.stop_btn.set_visibility(True)
            self.settings_page.apply_btn.set_enabled(False)
        else:
            self.start_btn.set_visibility(True)
            self.stop_btn.set_visibility(False)
            self.settings_page.apply_btn.set_enabled(True)

    @ui.refreshable
    def content(self):
        self.ist_tasks = []
        if self.tab_list:
            with ui.tab_panels(self.drawer_tabs, value=self.tab_list[0]).classes(
                    f'w-full h-[calc(100vh-{self.HEADER_HEIGHT}px)]'):
                for tab in self.tab_list:
                    ist_task = Instance(tab._props['name']).show()
                    self.ist_tasks.append(ist_task)

                    # Bind the status of the instance to the tab and start/stop button, using a hidden input element
                    proxy_element = ui.input().bind_value_from(ist_task, 'status')
                    proxy_element.on_value_change(lambda e, tab=tab: self.set_status(tab, e.value))
        else:
            ui.label(_('点击左下角添加配置')).classes(
                f'h-[calc(100vh-{self.HEADER_HEIGHT}px)] content-center self-center text-5xl')

    def show(self):
        ui.colors(primary='#711FCE')
        self.settings_dialog = self.settings_page.show()
        # self.header.show()
        with ui.left_drawer().props('width=75').classes('p-0 justify-between bg-gradient-to-t from-fuchsia-100'):
            self.sidebar()
        self.content()


if __name__ == '__main__':
    DaCapoUI().show()
    ui.run(window_size=(1200, 800), reload=False)
