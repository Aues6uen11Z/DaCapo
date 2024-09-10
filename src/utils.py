import gettext
import json
import locale
from os import PathLike
from pathlib import Path
from typing import Any, Callable, Optional, Tuple, Union, List

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


def card_title(
        name: str,
        icon: Optional[str] = None,
        extra: Optional[Callable[[], Any]] = None,
        help: Optional[str] = None
) -> None:
    """Title of the cards in main content(right part)"""
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


def bind_value(
        obj: ValueElement,
        target_object: Any,
        target_name: Union[str, Tuple[str, ...]] = 'value', *,
        forward: Callable[..., Any] = lambda x: x,
        backward: Callable[..., Any] = lambda x: x,
) -> ValueElement:
    """A temporary method, https://github.com/zauberzeug/nicegui/discussions/2978"""

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


def read_json(path: PathLike):
    with open(path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    return data


def write_json(path: PathLike, data: dict):
    with open(path, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False)


def instance_list() -> List[str]:
    path = Path('./config')
    return [p.stem for p in path.glob('*.json')]


def venv_list() -> List[str]:
    path = Path('./envs')
    if not path.exists():
        return []
    return [p.name for p in path.iterdir()]
