import os
from pathlib import Path

from loguru import logger
from nicegui import ui

from src.core.config import InstanceConfig, TemplateConfig
from src.interface.exclusive.custom import Custom
from src.interface.utils import get_text, card_title, bind_value

_ = get_text()


class General(Custom):
    def __init__(
            self,
            task_name: str,
            ist_config: InstanceConfig,
            tpl_config: TemplateConfig
    ):

        super().__init__(task_name, ist_config, tpl_config)

    def on_config_path_change(self):
        config_path = Path(self.ist_config.config_path)

        if not config_path.parent.exists():
            ui.notify(_('父目录不存在'), type='negative', position='top')
            logger.warning(f'{self.ist_config.name}: config path {config_path.parent} does not exist')
            return

        # Create a symbolic link pointing to source
        source_path = self.ist_config.path
        if config_path.exists():
            config_path.unlink()
        try:
            os.symlink(source_path, config_path)
        except OSError:
            ui.notify(_('权限不足，请以管理员身份运行本程序\n或手动创建该实例配置文件软链接（快捷方式）到目标位置下'),
                      type='warning', position='top', multi_line=True, classes='multi-line-notification')

    def general_group(self):
        """If this task is "General", show an extra group for language and working directory settings."""
        with ui.card().style('width:90%'):
            card_title(_('基本设置'))
            with ui.column().classes('w-full gap-1'):
                with ui.grid(columns='2fr 1fr').classes('w-full gap-0'):
                    ui.label(_('语言')).classes('text-lg content-center')
                    lang = ui.select(self.tpl_config.available_languages, value=self.language).props(
                        'dense outlined').classes('justify-center')
                    # lang.bind_value(self.storage, ('_info', 'language'))
                    bind_value(lang, self.storage, ('_info', 'language'))
                    lang.on_value_change(lambda: ui.notify(_('修改将在重启后生效'), type='info', position='top'))
                    ui.label(_('本界面显示的语言')).classes('text-gray-500').style('white-space: pre-wrap')
                    ui.space()

                    ui.label(_('工作目录')).classes('text-lg content-center')
                    cwd = ui.input(value=self.ist_config.work_dir).props('dense').classes('justify-center')
                    cwd.set_enabled(self.ist_config.work_dir_enabled)
                    # cwd.bind_value(self.storage, ('_info', 'work_dir'))
                    bind_value(cwd, self.storage, ('_info', 'work_dir'))
                    ui.label(_('程序运行的工作目录，通常应该是项目根目录')).classes('text-gray-500').style(
                        'white-space: pre-wrap')
                    ui.space()

                    ui.label(_('后台任务')).classes('text-lg content-center')
                    is_bg = ui.checkbox(value=self.ist_config.is_background).classes('justify-center')
                    is_bg.set_enabled(self.ist_config.is_background_enabled)
                    # is_bg.bind_value(self.storage, ('_info', 'is_background'))
                    bind_value(is_bg, self.storage, ('_info', 'is_background'))
                    ui.label(_('是否为完全的后台程序，不占用屏幕键鼠等设备')).classes('text-gray-500').style(
                        'white-space: pre-wrap')
                    ui.space()

                    ui.label(_('配置路径')).classes('text-lg content-center')
                    config_path = ui.input(value=self.ist_config.config_path).props('dense').classes('justify-center')
                    config_path.set_enabled(self.ist_config.config_path_enabled)
                    # If the config path is not enabled, call the function to create a symbolic link initiatively.
                    if not config_path.enabled:
                        self.on_config_path_change()
                    # config_path.bind_value(self.storage, ('_info', 'config_path'))
                    bind_value(config_path, self.storage, ('_info', 'config_path'))
                    config_path.on('blur', self.on_config_path_change)
                    ui.label(
                        _('你的程序到何处访问配置文件，具体到文件名\n例如 D:\\MyProject\\config\\template.json')).classes(
                        'text-gray-500').style('white-space: pre-wrap')

    def show(self):
        with ui.scroll_area().classes('h-full').props(
                'content-style="padding:0.125rem;align-items:center" '
                'content-active-style="padding:0.125rem;align-items:center"'):
            self.general_group()
            for group in list(self.group_dict.keys()):
                if group == '_Base':
                    continue
                self.group_item(group)
