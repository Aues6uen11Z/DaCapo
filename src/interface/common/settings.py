from pathlib import Path
import shutil
from typing import List, Tuple, Optional
from nicegui import app, ui

from src.core.config import InstanceConfig, TemplateConfig
from src.interface.common.file_picker import local_file_picker
from src.utils import default_ui_lang, get_text, instance_list

_ = get_text()


class Settings:

    def __init__(self, refreshables: List[ui.refreshable] = None):
        self.refreshables = refreshables
        self.get_new_method = [_('从已有模板创建'), _('导入')]
        self.apply_btn: Optional[ui.button] = None
        self.new_instance_dir = None

    async def pick_file(self) -> None:
        """Pick file from local file system."""
        self.new_instance_dir = await local_file_picker('~', multiple=False)
        ui.notify(_('你选择了 {0}').format(self.new_instance_dir), position='top', type='info')

    def add_new_instance(self, new_name: str, choice: str, tpl_name: str) -> None:
        """Copy new instance config to the config directory."""

        # validation check
        assert choice in self.get_new_method, f"Invalid choice: {choice}"
        if choice == self.get_new_method[0]:
            new_instance_name = new_name
        else:
            try:
                new_instance_name, new_tpl_name = new_name.rsplit('@', 1)
            except ValueError:
                new_instance_name = new_tpl_name = None
            if not new_instance_name or not new_tpl_name:
                ui.notify(_('请填写 [实例名@模板名]'), position='top', type='warning')
                return
        for instance_name in instance_list():
            if instance_name == new_instance_name:
                ui.notify(_("新名字与已有实例名冲突"), position='top', type='warning')
                return

        # Import new instance, create new template first
        if choice == self.get_new_method[1]:
            warning = _('请选择有效目录, 当前为 ')
            if self.new_instance_dir:
                source_path = Path(self.new_instance_dir[0])
            else:
                ui.notify(warning + f"{self.new_instance_dir}", position='top', type='warning')
                return

            if not source_path.is_dir():
                ui.notify(warning + f"{source_path}", position='top', type='warning')
                return

            files_to_check = ['args.json']
            fail_flag = False  # Remind users of all missing files
            for file in files_to_check:
                if not (source_path / file).exists():
                    ui.notify(_("缺少必要配置文件 {0}").format(file), position='top', type='warning')
                    fail_flag = True
            if fail_flag:
                return

            # Copy files to config/templates
            target_path = Path('./config/templates') / new_tpl_name
            if target_path.exists():
                shutil.rmtree(target_path)
            target_path.mkdir()
            for file in files_to_check + ['i18n']:
                if not (source_path / file).exists():
                    continue
                if (source_path / file).is_dir():
                    shutil.copytree(source_path / file, target_path / file)
                else:
                    shutil.copy2(source_path / file, target_path / file)
            tpl_name = new_tpl_name

        TemplateConfig(tpl_name).add_instance(new_instance_name)

    def refresh_ui(self) -> None:
        self.instance_panel.refresh()
        if self.refreshables:
            for refreshable in self.refreshables:
                refreshable.refresh()

    def on_apply(self, inputs: List[ui.input], radio: ui.radio, select: ui.select, switches: List[ui.switch]) -> None:
        """Check all things and apply modifications."""
        choice = radio.value
        tpl_name = select.value
        if choice == self.get_new_method[0]:
            new_name = inputs[0].value
        else:
            new_name = inputs[1].value

        if new_name:
            self.add_new_instance(new_name, choice, tpl_name)

        # Update instance "is_ready" status
        for switch in switches:
            ist_config = InstanceConfig(switch.text)
            ist_config.update_ready_status(switch.value)

        self.refresh_ui()

    def on_delete(self, instance_name: str) -> None:
        """Show dialog to confirm deletion of an instance."""
        def delete_instance(instance_name: str) -> None:
            path = Path(f'./config/{instance_name}.json')
            path.unlink()
            dialog.clear()
            self.refresh_ui()

        def cancel() -> None:
            dialog.close()
            dialog.clear()

        with ui.dialog() as dialog, ui.card():
            ui.label(_('确定删除配置<{0}>吗?').format(instance_name))
            with ui.row().classes('w-full justify-end'):
                ui.button(_('确定'), on_click=lambda: delete_instance(instance_name)).props('outline')
                ui.button(_('取消'), on_click=lambda: cancel()).props('outline')
        dialog.open()

    def instance_switches(self) -> List[ui.switch]:
        """Show all instances with switches"""
        with ui.scroll_area().classes('w-full h-72 border'):
            switches = []
            for instance_name in instance_list():
                with ui.row().classes('w-full justify-between'):
                    switch = ui.switch(instance_name, value=InstanceConfig(instance_name).is_ready)
                    ui.button(icon='delete_outline',
                              on_click=lambda instance_name=instance_name: self.on_delete(instance_name)
                              ).props('flat round')
                switches.append(switch)
        return switches

    def linkage_options(self) -> Tuple[List[ui.input], ui.radio, ui.select]:
        """Display different effects based on radio"""
        with ui.row().classes('items-end w-full gap-0'):
            with ui.row().classes('w-full h-10 items-center justify-between'):
                selected = {'value': self.get_new_method[0]}
                radio = ui.radio(self.get_new_method, value=self.get_new_method[0]) \
                    .props('dense inline size="sm"').bind_value_to(selected, 'value')

                tpl_path = Path('./config/templates')
                tpl_list = [p.name for p in tpl_path.iterdir() if p.is_dir()]
                if not tpl_list:
                    tpl_list = ['']
                select = ui.select(tpl_list, value=tpl_list[0]).props('dense').classes('grow') \
                    .bind_visibility_from(radio, 'value', value=self.get_new_method[0])

                ui.button(_('选择文件'), on_click=lambda: self.pick_file(), icon='folder_open') \
                    .props('outline color="primary"') \
                    .bind_visibility_from(radio, 'value', value=self.get_new_method[1])

        ui.space()

        input1 = ui.input(_('新配置名'), placeholder=_('格式：实例名')).props('dense') \
            .classes('w-full').bind_visibility_from(radio, 'value', value=self.get_new_method[0])
        input2 = ui.input(_('新配置名'), placeholder=_('格式：实例名@模板名')).props('dense') \
            .classes('w-full').bind_visibility_from(radio, 'value', value=self.get_new_method[1])

        return [input1, input2], radio, select

    # Add new instance and configure
    @ui.refreshable
    def instance_panel(self) -> None:
        with ui.grid(columns='1fr 3fr').classes('w-full items-center gap-1'):
            ui.label(_('添加新实例')).classes('text-xl font-bold')

            inputs, radio, select = self.linkage_options()

            ui.label(_('管理实例')).classes('text-xl font-bold')
            select_all = ui.switch(_('全选/取消全选'))
            select_all.on_value_change(lambda e: [switch.set_value(e.value) for switch in switches])
            ui.space()
            switches = self.instance_switches()

        self.apply_btn = ui.button(_('应用'), on_click=lambda: self.on_apply(inputs, radio, select, switches)) \
            .props('outline color="primary"').classes('place-self-end')

    @staticmethod
    def general_panel() -> None:
        with ui.grid(columns='1fr 1fr 1fr').classes('self-center'):
            with ui.row(wrap=False).classes('items-center'):
                ui.icon('g_translate').classes('text-xl font-bold')
                ui.label(_('语言')).classes('text-xl font-bold')
            ui.space()
            lang = ui.select(['zh_CN', 'en_US'], value=default_ui_lang()).props('dense')
            lang.bind_value(app.storage.general, 'language')
            lang.on_value_change(lambda: ui.notify(_('修改将在重启后生效'), type='info', position='top'))

    # Setting page, accessed by clicking button on the bottom left
    def show(self, open: bool = False) -> ui.dialog:
        with ui.dialog(value=open) as dialog:
            with ui.card().classes('w-full h-5/6'):
                with ui.tabs().classes('self-center') as setting_tabs:
                    instance_tab = ui.tab(_('实例'))
                    general_tab = ui.tab(_('通用'))
                    about_tab = ui.tab(_('关于'))

                with ui.tab_panels(setting_tabs, value=instance_tab).classes('w-full h-full'):
                    with ui.tab_panel(instance_tab):
                        self.instance_panel()
                    with ui.tab_panel(general_tab):
                        self.general_panel()

        return dialog


if __name__ == '__main__':
    Settings().show()
    ui.run(window_size=(1200, 800), reload=False)
