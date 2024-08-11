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
    ):

        self.task_name = task_name  # not translated
        self.ist_config = ist_config
        self.tpl_config = tpl_config

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
                    priority = ui.number(value=self.ist_config.priority(self.task_name), min=0, max=100) \
                        .props('dense').classes('justify-center')
                    priority.set_enabled(self.ist_config.priority_enabled(self.task_name))
                    # priority.bind_value(self.storage, ('_info', 'tasks', self.task_name, 'priority'))
                    bind_value(priority, self.storage, ('_info', 'tasks', self.task_name, 'priority'))
                    ui.label(_('在等候队列的排序优先级，不同任务可以重复')).classes('text-gray-500').style(
                        'white-space: pre-wrap')
                    ui.space()

                    ui.label(_('运行命令')).classes('text-lg content-center')
                    command = ui.input(value=self.ist_config.command(self.task_name)) \
                        .props('dense').classes('justify-center')
                    command.set_enabled(self.ist_config.command_enabled(self.task_name))
                    # command.bind_value(self.storage, ('_info', 'tasks', self.task_name, 'command'))
                    bind_value(command, self.storage, ('_info', 'tasks', self.task_name, 'command'))
                    ui.label(
                        _('执行该任务的命令，请先在命令行尝试是否能正常运行，例如\n'
                          '在设置的工作目录执行“D:\\python\\python.exe test.py -task1”\n若有多个python环境，请注意python解释器路径')
                    ).classes('text-gray-500').style('white-space: pre-wrap')

    def show(self):
        with ui.scroll_area().classes('h-full').props(
                'content-style="padding:0.125rem;align-items:center" '
                'content-active-style="padding:0.125rem;align-items:center"'):
            self.task_group()
            for group in list(self.group_dict.keys()):
                if group == '_Base':
                    continue
                self.group_item(group)
