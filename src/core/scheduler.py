import asyncio
from asyncio.subprocess import Process
import shlex
import sys
from typing import List, Literal, Optional

from loguru import logger

from src.core.config import InstanceConfig
from src.interface.exclusive.home import Home


class TaskManager:
    """
    Run a command in the background and display the output in the pre-created log view.
    """

    def __init__(self, ist_config: InstanceConfig, gui: Home = None):
        self.ist_config = ist_config
        self.gui = gui
        self.is_background = ist_config.is_background

        self.process: Process = None
        self.manual_stop: bool = False    # Whether the task is manually stopped
        self.status: Literal['standby', 'running', 'error'] = 'standby'

    # Inspired by https://github.com/zauberzeug/nicegui/blob/main/examples/script_executor/main.py
    async def run_command(self, task_name: str, command: str) -> Optional[Exception]:
        """Run a command in the background and display the output in the pre-created dialog."""
        try:
            if "win" in sys.platform.lower():
                self.process = await asyncio.create_subprocess_exec(
                    *shlex.split(command, posix=False),
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.STDOUT,
                    cwd=self.ist_config.work_dir,
                    creationflags=0x08000000  # CREATE_NO_WINDOW
                )
            else:
                self.process = await asyncio.create_subprocess_exec(
                    *shlex.split(command, posix=True),
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.STDOUT,
                    cwd=self.ist_config.work_dir
                )
        except Exception as e:
            logger.error(f'{self.ist_config.name}-{task_name}: {e}')
            return e
        
        logger.info(f'{self.ist_config.name}-{task_name} start running with command: {command}')
        return None

    async def output(self):
        # NOTE we need to read the output in chunks, otherwise the process will block
        while True:
            new = await self.process.stdout.read(4096)
            if not new:
                break
            yield new.decode()
        await self.process.wait()
    
    def stop(self):
        self.process.terminate()
        self.manual_stop = True

    async def run(self) -> Optional[Exception]:
        self.gui.set_btn_visibility(running=True)
        self.status = 'running'
        task = self.gui.running_get()
        while task:
            cmd = self.ist_config.command(task)
            res = await self.run_command(task, cmd)
            if res:
                self.gui.set_btn_visibility(running=False)
                self.status = 'standby'
                return res
            self.gui.move2running()
            async for line in self.output():
                self.gui.log.push(line)
            
            if self.process.returncode == 0:
                logger.info(f'{self.ist_config.name}-{task} finished successfully')
                self.gui.move_completed()
                task = self.gui.running_get()
            else:
                if self.manual_stop:
                    logger.info(f'{self.ist_config.name}-{task} stopped manually')
                    self.status = 'standby'
                    self.manual_stop = False
                else:
                    logger.info(f'{self.ist_config.name}-{task} exited with error')
                    self.status = 'error'
                self.gui.move_completed()
                self.gui.set_btn_visibility(running=False)
                break
        # No task in the waiting card
        self.gui.set_btn_visibility(running=False)
        self.status = 'standby'


class Scheduler:
    def __init__(self):
        self.foreground_tasks: List[TaskManager] = []
        self.background_tasks: List[TaskManager] = []

    def add_foreground_task(self, task):
        self.foreground_tasks.append(task)

    def add_background_task(self, task):
        self.background_tasks.append(task)

    def clear_tasks(self):
        self.foreground_tasks = []
        self.background_tasks = []

    async def run_foreground_tasks(self):
        for task in list(self.foreground_tasks):
            if task in self.foreground_tasks:
                await task.run()
                if task in self.foreground_tasks:
                    self.foreground_tasks.remove(task)

    async def run_background_tasks(self):
        if self.background_tasks:
            await asyncio.gather(*(task.run() for task in self.background_tasks))
            self.background_tasks = []

    async def run(self):
        logger.info(f'Scheduler start')
        logger.info(f'Foreground instance tasks:')
        for task in self.foreground_tasks:
            logger.info(f'  {task.ist_config.name}')
        logger.info(f'Background instance tasks:')
        for task in self.background_tasks:
            logger.info(f'  {task.ist_config.name}')

        await asyncio.gather(self.run_foreground_tasks(), self.run_background_tasks())

        logger.info(f'All instance tasks finished, scheduler stopped')
