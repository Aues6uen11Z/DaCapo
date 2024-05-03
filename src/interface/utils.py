import gettext
import locale
from typing import Any, Callable, Optional, Tuple, Union

from nicegui import app, ui
from nicegui.elements.mixins.value_element import ValueElement


def default_ui_lang():
    system_lang = locale.getdefaultlocale()[0]
    if system_lang not in ['zh_CN', 'en_US']:  # To be extended
        return 'en_US'
    else:
        return system_lang


def get_text():
    lang = app.storage.general.get('language', default_ui_lang())
    translate = gettext.translation('dacapo', localedir='locale', languages=[lang])
    return translate.gettext


# Title of the cards in main content(right part)
def card_title(
        name: str,
        icon: Optional[str] = None,
        extra: Optional[Callable[[], ui.element]] = None,
        help: Optional[str] = None
) -> None:
    with ui.row().classes('w-full justify-between'):
        with ui.row().classes('h-full'):
            if icon:
                ui.icon(icon).classes('text-2xl h-full')
            ui.label(name).classes('text-2xl h-full content-center')
        if extra:
            extra()

    if help:
        with ui.row().classes('w-full mt-[-10px]'):
            ui.label(help).classes('text-gray-500').style('white-space: pre-wrap')

    ui.separator()


# A temporary method, https://github.com/zauberzeug/nicegui/discussions/2978
def bind_value(
        obj: ValueElement,
        target_object: Any,
        target_name: Union[str, Tuple[str, ...]] = 'value', *,
        forward: Callable[..., Any] = lambda x: x,
        backward: Callable[..., Any] = lambda x: x,
) -> ValueElement:
    def _convert_target(
            target_object: Any,
            target_name: Union[str, Tuple[str, ...]]
    ) -> Tuple[Any, str]:
        if isinstance(target_name, tuple):
            if isinstance(target_object, dict):
                for key in target_name[:-1]:
                    try:
                        target_object[key]
                    except KeyError:
                        target_object[key] = {}
                    target_object = target_object[key]
                target_name = target_name[-1]
            else:
                raise TypeError

        return target_object, target_name

    target_object, target_name = _convert_target(target_object, target_name)

    return obj.bind_value(target_object, target_name, forward=forward, backward=backward)
