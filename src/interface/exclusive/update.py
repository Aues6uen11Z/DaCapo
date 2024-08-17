from typing import Callable, Optional

from nicegui import ui

from src.core.config import InstanceConfig
from src.interface.utils import get_text, bind_value

_ = get_text()


class Update:

    def __init__(
            self,
            ist_config: InstanceConfig,
            update_start: Callable
    ):
        self.ist_config = ist_config
        self.update_start = update_start
        self.storage = self.ist_config.storage

        self.update_btn: Optional[ui.button] = None

    async def on_update(self):
        self.update_btn.set_text(_('更新中...'))
        self.update_btn.set_enabled(False)

        is_success, error = await self.update_start()

        if is_success:
            self.update_btn.set_text(_('已是最新版本'))
        else:
            ui.notify(_('更新出错：\n{0}').format(error),
                      type='negative', position='top', multi_line=True, classes='multi-line-notification')
            self.update_btn.set_enabled(True)
            self.update_btn.set_text(_('检查更新'))

    def update_group(self):
        with ui.card().style('width:90%'):
            with ui.row().classes('w-full justify-between'):
                with ui.row().classes('h-full'):
                    ui.label(_('更新设置')).classes('text-2xl h-full content-center')
                self.update_btn = ui.button(_('检查更新'), icon='update', on_click=self.on_update).props('outline')
            ui.separator()

            with ui.column().classes('w-full gap-1'):
                with ui.grid(columns='2fr 1fr').classes('w-full gap-0'):
                    ui.label(_('Git仓库地址')).classes('text-lg content-center')
                    repo_url = ui.input(value=self.ist_config.repo_url).props('dense').classes('justify-center')
                    repo_url.set_enabled(self.ist_config.repo_url_enabled)
                    bind_value(repo_url, self.storage, ('_info', 'repo_url'))

                    ui.label(_('Git分支')).classes('text-lg content-center')
                    branch = ui.input(value=self.ist_config.branch).props('dense').classes('justify-center')
                    branch.set_enabled(self.ist_config.branch_enabled)
                    bind_value(branch, self.storage, ('_info', 'branch'))
                    ui.label(_('留空则为默认分支')).classes('text-gray-500').style('white-space: pre-wrap')
                    ui.space()

                    ui.label(_('本地路径')).classes('text-lg content-center')
                    local_path = ui.input(value=self.ist_config.local_path).props('dense').classes('justify-center')
                    local_path.set_enabled(self.ist_config.local_path_enabled)
                    bind_value(local_path, self.storage, ('_info', 'local_path'))
                    ui.label(_('仓库在本地存储路径')).classes('text-gray-500').style('white-space: pre-wrap')
                    ui.space()

                    ui.label(_('自动更新')).classes('text-lg content-center')
                    auto_update = ui.checkbox().classes('justify-center')
                    bind_value(auto_update, self.storage, ('_info', 'auto_update'))
                    ui.label(_('修改将在下次启动生效')).classes('text-gray-500').style('white-space: pre-wrap')
                    ui.space()
                    if auto_update.value:
                        ui.timer(0.1, callback=self.on_update, once=True)

    def show(self):
        with ui.scroll_area().classes('h-full').props(
                'content-style="padding:0.125rem;align-items:center" '
                'content-active-style="padding:0.125rem;align-items:center"'):
            self.update_group()
