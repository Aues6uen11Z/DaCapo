import os
from pathlib import Path
from typing import Optional
from loguru import logger
from nicegui import ui

from src.core.config import InstanceConfig, TemplateConfig
from src.interface.utils import bind_value, card_title, get_text

_ = get_text()


class Custom:

    def __init__(
            self,
            task_name: str,
            ist_config: InstanceConfig,
            tpl_config: TemplateConfig,
            is_general: Optional[bool] = False
    ):

        self.task_name = task_name  # not translated
        self.ist_config = ist_config
        self.tpl_config = tpl_config
        self.is_general = is_general

        self.language = self.ist_config.language
        self.storage = self.ist_config.storage
        self.group_dict = self.tpl_config.group_dict(self.task_name)

    def tr(self, group: str, argument: str = None, key: str = None) -> str:
        """Translate the group name, argument name, and argument value"""
        if self.language == 'default':
            if not argument:
                return group
            if not key:
                return argument
            return key

        # Translate the group name
        if not argument:
            res = self.tpl_config.translation(self.language).get(group, {}).get('_info', {}).get('name', group)
            return res
        # Translate the argument name
        if not key:
            res = self.tpl_config.translation(self.language).get(group, {}).get(argument, {}).get('name', argument)
            return res
        # Translate the argument value
        res = self.tpl_config.translation(self.language).get(group, {}).get(argument, {}).get(key, key)
        return res

    def help(self, group: str, argument: str = None) -> str:
        """Gray explanatory text for each configuration item."""
        if self.language == 'default':
            return ''

        if not argument:
            res = self.tpl_config.translation(self.language).get(group, {}).get('_info', {}).get('help', '')
            return res
        res = self.tpl_config.translation(self.language).get(group, {}).get(argument, {}).get('help', '')
        return res

    def argument_item(self, group: str, argument: str, args: dict) -> bool:
        """Configuration items for custom pages."""
        with ui.grid(columns='2fr 1fr').classes('w-full gap-0'):
            label = ui.label(f'{self.tr(group, argument)}').classes('text-lg content-center')
            display = {"display": False if args.get('display') else True}
            label.bind_visibility_from(display, 'display')

            value = args.get('value', '')
            if args.get('type') == 'select':
                value_t = self.tr(group, argument, value)
                options = args.get('option', [])
                options_t = [self.tr(group, argument, option) for option in options]
                var2trans = dict(zip(options, options_t))
                trans2var = dict(zip(options_t, options))
                # Display the translated value, but store the original value
                element = ui.select(options_t, value=value_t).props('dense outlined')
                # element.bind_value(self.storage, (self.task_name, group, argument),
                #                    forward=lambda x: trans2var.get(x, x), backward=lambda x: var2trans.get(x, x))
                bind_value(element, self.storage, (self.task_name, group, argument),
                           forward=lambda x: trans2var.get(x, x), backward=lambda x: var2trans.get(x, x))

            elif args.get('type') == 'stored':  # TODO
                element = ui.input(value='{}').props('dense')
                # element.bind_value(self.storage, (self.task_name, group, argument))
                bind_value(element, self.storage, (self.task_name, group, argument))

            elif args.get('type') == 'input' or args.get('type') == 'textarea':
                element = ui.input(value=value).props('dense')
                # element.bind_value(self.storage, (self.task_name, group, argument))
                bind_value(element, self.storage, (self.task_name, group, argument))

            elif args.get('type') == 'checkbox':
                element = ui.checkbox(value=value)
                # element.bind_value(self.storage, (self.task_name, group, argument))
                bind_value(element, self.storage, (self.task_name, group, argument))

            else:
                element = ui.input(value=value).props('dense')
                # element.bind_value(self.storage, (self.task_name, group, argument))
                bind_value(element, self.storage, (self.task_name, group, argument))

            element.classes('justify-center')
            element.bind_visibility_from(display, 'display')

            if self.help(group, argument):
                help = ui.label(f'{self.help(group, argument)}').classes('text-gray-500').style('white-space: pre-wrap')
                help.bind_visibility_from(display, 'display')

        return display['display']  # Return whether the item is displayed

    def group_item(self, group: str):
        """A group contains multiple argument items."""
        argument_count = {'count': 0}
        with ui.card().style('width:90%') as card:
            card_title(self.tr(group), help=self.help(group))
            with ui.column().classes('w-full gap-1'):
                for argument, value in self.group_dict[group].items():
                    if self.argument_item(group, argument, value):
                        argument_count['count'] += 1
        card.bind_visibility_from(argument_count, 'count')

    def task_group(self):
        """Each task has a group for setting the priority and command."""
        with ui.card().style('width:90%'):
            card_title(_('任务设置'))
            with ui.column().classes('w-full gap-1'):
                with ui.grid(columns='2fr 1fr').classes('w-full gap-0'):
                    ui.label(_('默认优先级')).classes('text-lg content-center')
                    priority = ui.number(value=self.ist_config.priority(self.task_name), min=0, max=100)\
                        .props('dense').classes('justify-center')
                    priority.set_enabled(self.ist_config.priority_enabled(self.task_name))
                    # priority.bind_value(self.storage, ('_info', 'tasks', self.task_name, 'priority'))
                    bind_value(priority, self.storage, ('_info', 'tasks', self.task_name, 'priority'))
                    ui.label(_('在等候队列的排序优先级，不同任务可以重复')).classes('text-gray-500').style(
                        'white-space: pre-wrap')
                    ui.space()

                    ui.label(_('运行命令')).classes('text-lg content-center')
                    command = ui.input(value=self.ist_config.command(self.task_name))\
                        .props('dense').classes('justify-center')
                    command.set_enabled(self.ist_config.command_enabled(self.task_name))
                    # command.bind_value(self.storage, ('_info', 'tasks', self.task_name, 'command'))
                    bind_value(command, self.storage, ('_info', 'tasks', self.task_name, 'command'))
                    ui.label(
                        _('执行该任务的命令，请先在命令行尝试是否能正常运行，例如\n'
                          '在设置的工作目录执行“D:\\python\\python.exe test.py -task1”\n若有多个python环境，请注意python解释器路径')
                    ).classes('text-gray-500').style('white-space: pre-wrap')

    def on_config_path_change(self):
        config_path = Path(self.ist_config.config_path).resolve()
        
        if not config_path.parent.exists():
            ui.notify(_('父目录不存在'), type='error', position='top')
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
            if self.is_general:
                self.general_group()
            else:
                self.task_group()
            for group in list(self.group_dict.keys()):
                if group == '_Base':
                    continue
                self.group_item(group)
