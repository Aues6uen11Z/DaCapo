from threading import Thread

from loguru import logger
from nicegui import app, ui
from PIL import Image
from pystray import Icon, Menu, MenuItem

from src.interface.gui import DaCapoUI
from src.interface.utils import get_text

# from niceguiToolkit.layout import inject_layout_tool
# inject_layout_tool()


_ = get_text()

logger.add('dacapo.log', format="<green>{time:YYYY-MM-DD HH:mm:ss}</green> | "
                                "<level>{level: <7}</level> | "
                                "<level>{message}</level>"
           )


def on_open():
    app.native.main_window.show()


def on_hide():
    app.native.main_window.hide()


def on_exit():
    icon.stop()
    app.shutdown()


def show_exception(e):
    logger.exception(e)
    ui.notify(str(e), position='top', type='negative')


if __name__ == "__main__":
    image = Image.open('static/logo/logo.ico')
    menu = Menu(MenuItem(_('打开'), on_open, default=True), MenuItem(_('隐藏'), on_hide), MenuItem(_('退出'), on_exit))
    icon = Icon('DaCapo', image, menu=menu)
    Thread(target=icon.run, daemon=True).start()

    app.on_exception(lambda e: show_exception(e))
    DaCapoUI().show()
    ui.run(title='DaCapo', window_size=(1200, 800), reload=False)
    icon.stop()
