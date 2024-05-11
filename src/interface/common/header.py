from nicegui import app, ui

class Header:
    """Title and window-control buttons."""
    def __init__(self, height: int):
        self.name = 'DaCapo'
        self.height = height
        self.window_state = {'max': False}

    def maximize(self):
        app.native.main_window.maximize()
        self.window_state['max'] = True
    
    def restore(self):
        app.native.main_window.restore()
        self.window_state['max'] = False

    def show(self):
        with ui.header().style(f'height: {self.height}px').classes('items-center justify-between p-0'):
            ui.label(f'{self.name}').classes('text-2xl ml-2')
            with ui.row().classes('gap-0 h-full'):
                ui.button(icon='horizontal_rule', on_click=lambda: app.native.main_window.minimize())\
                    .classes('h-full').props('flat color="white"')
                ui.button(icon='fullscreen', on_click=self.maximize).bind_visibility_from(self.window_state, 'max', lambda x: not x)\
                    .classes('h-full').props('flat color="white"')
                ui.button(icon='fullscreen_exit', on_click=self.restore).bind_visibility_from(self.window_state, 'max')\
                    .classes('h-full').props('flat color="white"')
                ui.button(icon='close', on_click=lambda: app.native.main_window.hide())\
                    .classes('h-full').props('flat color="white"')


if __name__ == '__main__':
    header = Header(60)
    header.show()
    ui.run(native=True, window_size=(1200, 800), frameless=True, reload=False)
