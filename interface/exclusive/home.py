from typing import Callable, List
from nicegui import ui

from core.config import InstanceConfig
from interface.utils import card_title, get_text

_ = get_text()


class Home:

    def __init__(self, ist_config: InstanceConfig, task_list: List[str], task_list_t: List[str]):
        self.ist_config = ist_config
        self.task_list = task_list
        self.task_list_t = task_list_t
        self.var2trans = dict(zip(task_list, task_list_t))
        self.trans2var = dict(zip(task_list_t, task_list))
        self.sort_tasks()

        self.running_card = None
        self.waiting_card = None
        self.terminated_card = None
        self.log = None
        self.start_callback = None
        self.stop_callback = None
        self.start_btn = None
        self.stop_btn = None

    def sort_tasks(self):
        self.task_list = sorted(self.task_list, key=lambda task: self.ist_config.priority(task))
        self.task_list_t = [self.var2trans[task] for task in self.task_list]

    # Move running task item to terminated card
    def move_completed(self) -> None:
        completed_task = list(self.running_card)[0]
        completed_task.move(self.terminated_card)

    def move2running(self) -> None:
        try:
            task_item = list(self.waiting_card)[0]
        except IndexError:
            return None
        task_item.move(self.running_card)

    # Get first task in the waiting card
    def running_get(self) -> str:
        try:
            task_item = list(self.waiting_card)[0]
        except IndexError:
            return ''
        task_button = list(list(task_item)[0])[0]
        task_name = self.trans2var[task_button.text]
        return task_name

    # Redirect to task config page by clicking the setting button in task item
    def goto_task_panel(self, setting_button: ui.button) -> None:
        task_button = list(setting_button.parent_slot.parent)[0]
        task_name = task_button.text
        # button.text is translated, but need the original task name to navigate to the task panel
        task_name = self.trans2var[task_name]

        element = task_button
        while not isinstance(element, ui.tab_panels):
            element = element.parent_slot.parent
        element.set_value(task_name)

    # Move all task items to waiting card
    def move2waiting(self) -> None:
        tasks = list(self.terminated_card)
        for task in tasks:
            task.move(self.waiting_card)

    # Move all task items to terminated card
    def move2terminated(self) -> None:
        tasks = list(self.waiting_card)
        for task in tasks:
            task.move(self.terminated_card)

    # Task item on click method
    def move_task(self, button: ui.button) -> None:
        button = button.parent_slot.parent.parent_slot.parent  # ui.button -> ui.row -> ui.row, the "button"
        parent = button.parent_slot.parent  # ui.row -> ui.scroll_area
        if parent == self.running_card:
            return
        elif parent == self.waiting_card:
            button.move(self.terminated_card)
        elif parent == self.terminated_card:
            button.move(self.waiting_card)

    # Task item in running, waiting, and terminated cards
    def task_item(self, name: str) -> None:
        with ui.row().classes('w-full'):  # https://github.com/zauberzeug/nicegui/pull/2301
            with ui.row(wrap=False).classes('w-full gap-0 border border-black rounded'):
                ui.button(name, color=None, on_click=lambda e: self.move_task(e.sender)).classes(
                    'self-stretch grow').props('no-caps flat align=left')
                ui.button(icon='settings', on_click=lambda e: self.goto_task_panel(e.sender)).classes(
                    'self-center').props('flat round')

    # Garbage code, needs improvement, used in instance.content
    def add_callback(self, start_callback: Callable, stop_callback: Callable):
        self.start_callback = start_callback
        self.stop_callback = stop_callback

    def set_btn_visibility(self, running: bool):
        if running:
            self.start_btn.set_visibility(False)
            self.stop_btn.set_visibility(True)
        else:
            self.start_btn.set_visibility(True)
            self.stop_btn.set_visibility(False)

    def execute_button(self) -> None:
        async def on_start():
            res = await self.start_callback()
            if res:
                ui.notify(_('请检查工作目录和任务运行命令设置\n报错内容：{0}').format(res),
                          position='top', type='negative', multi_line=True, classes='multi-line-notification')

        def on_stop():
            self.stop_callback()

        self.start_btn = ui.button(icon='play_arrow').props('flat round')
        self.start_btn.on_click(on_start)

        self.stop_btn = ui.button(icon='stop').props('flat round')
        self.stop_btn.set_visibility(False)
        self.stop_btn.on_click(on_stop)

    def show(self):
        with ui.row(wrap=False).classes('w-full h-full'):
            with ui.column().classes('w-80 h-full'):
                # running card
                with ui.card().classes('w-full h-36 pb-0'):
                    card_title(_('运行'), 'cached', self.execute_button)
                    running_card = ui.scroll_area().classes('h-full')

                # waiting card
                with ui.card().classes('w-full h-2/5'):
                    card_title(_('等待'), 'hourglass_top',
                               lambda: ui.button(icon='south', on_click=self.move2terminated).props('flat round'))
                    with ui.scroll_area().classes('h-full') as waiting_card:
                        for task in self.task_list_t:
                            self.task_item(task)

                # terminated card
                with ui.card().classes('w-full grow'):
                    card_title(_('终止'), 'block',
                               lambda: ui.button(icon='north', on_click=self.move2waiting).props('flat round'))
                    terminated_card = ui.scroll_area().classes('h-full')

            # log card
            with ui.card().classes('w-full h-full'):
                card_title(_('日志'), 'description')
                self.log = ui.log().classes('w-full h-full')

        self.running_card = running_card
        self.waiting_card = waiting_card
        self.terminated_card = terminated_card
