import asyncio
from asyncio.subprocess import Process
from pathlib import Path
import shlex
import sys
from typing import List, Literal, Optional

from loguru import logger

from src.core.updater import Updater
from src.core.config import InstanceConfig
from src.interface.exclusive.home import Home


class TaskManager:
    """
    Run a command in the background and display the output in the pre-created log view.
    """

    def __init__(self, ist_config: InstanceConfig, gui: Home = None):
        self.ist_config = ist_config
        self.gui = gui

        self.process: Optional[Process] = None
        self.manual_stop: bool = False  # Whether the task is manually stopped
        self.status: Literal['standby', 'updating', 'running', 'error'] = 'standby'

    async def update(self) -> Optional[Exception]:
        updater = Updater(self.ist_config)
        self.status = 'updating'
        error = await updater.update()
        if error:
            self.status = 'error'
            return error
        self.status = 'standby'

    # Inspired by https://github.com/zauberzeug/nicegui/blob/main/examples/script_executor/main.py
    async def run_command(self, task_name: str, command: str) -> Optional[Exception]:
        """Run a command in the background, not wait for stop."""
        work_dir = Path(self.ist_config.work_dir).resolve()
        try:
            if "win" in sys.platform.lower():
                cmd = shlex.split(command, posix=False)
            else:
                cmd = shlex.split(command, posix=True)
            
            if cmd[0] == 'py':
                if not self.ist_config.env_name:
                    raise ValueError('"py" command is only supported when env_name is set')
                python_exec = Path('./envs') / self.ist_config.env_name / 'python.exe'
                if not python_exec.exists():
                    raise FileNotFoundError(f'Python executable not found: {python_exec}')
                cmd[0] = str(python_exec.resolve())
            
            self.process = await asyncio.create_subprocess_exec(
                *cmd,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.STDOUT,
                cwd=work_dir,
                creationflags=0x08000000  # CREATE_NO_WINDOW
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
            error = await self.run_command(task, cmd)
            if error:
                self.gui.set_btn_visibility(running=False)
                self.status = 'error'
                return error
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
                return
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
        while self.foreground_tasks:
            all_updating = True
            for _ in range(len(self.foreground_tasks)):
                if not self.foreground_tasks:
                    return  # Manual stop will call clear_tasks()

                task = self.foreground_tasks.pop(0)
                if task.status == "updating":
                    self.foreground_tasks.append(task)
                elif task.status == "error" and task in self.foreground_tasks:
                    self.foreground_tasks.remove(task)  # Update failed
                else:
                    all_updating = False
                    await task.run()
                    if task in self.foreground_tasks:
                        self.foreground_tasks.remove(task)

            if all_updating:
                await asyncio.sleep(10)

    @staticmethod
    async def run_with_check(task: TaskManager):
        while task.status == "updating":
            await asyncio.sleep(10)
        if task.status == "standby":
            await task.run()

    async def run_background_tasks(self):
        if self.background_tasks:
            await asyncio.gather(*(self.run_with_check(task) for task in self.background_tasks))
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
