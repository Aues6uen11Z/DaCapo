import asyncio
from asyncio.subprocess import Process
from functools import partial
from pathlib import Path
import shlex
import sys
from typing import List, Literal, Optional, Tuple

from loguru import logger

from src.core.config import InstanceConfig
from src.interface.exclusive.home import Home
from src.interface.utils import get_text

_ = get_text()


class TaskManager:
    """
    Run a command in the background and display the output in the pre-created log view.
    """

    def __init__(self, ist_config: InstanceConfig, gui: Home = None):
        self.ist_config = ist_config
        self.gui = gui
        self.is_background = ist_config.is_background

        self.process: Optional[Process] = None
        self.manual_stop: bool = False  # Whether the task is manually stopped
        self.status: Literal['standby', 'updating', 'running', 'error'] = 'standby'

    async def update(self) -> Tuple[bool, Optional[Exception]]:
        async def execute_command(command: list, repo_path: Path) -> Tuple[bool, str, Optional[Exception]]:
            try:
                process = await asyncio.create_subprocess_exec(
                    *command,
                    cwd=repo_path,
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.PIPE,
                    creationflags=0x08000000
                )
                logger.info(f'{self.ist_config.name}-Update started with command: {" ".join(command)}')

                stdout, stderr = await process.communicate()
                output = stdout.decode().strip()
                error = stderr.decode().strip()
                if process.returncode == 0:
                    return True, output, None
                else:
                    self.status = 'error'
                    logger.error(f'{self.ist_config.name}-{" ".join(command)}: {error}')
                    return False, output, Exception(error)
            except Exception as e:
                self.status = 'error'
                logger.error(f'{self.ist_config.name}-{" ".join(command)}: {e}')
                return False, '', e

        self.status = 'updating'
        repo_url = self.ist_config.repo_url
        branch = self.ist_config.branch

        local_path = Path(self.ist_config.local_path).resolve()
        repo_name = repo_url.rstrip('/').split('/')[-1].replace('.git', '')
        if local_path.name != repo_name:
            if not local_path.is_dir():
                self.status = 'error'
                global _
                return False, Exception(_('本地路径设置有误'))
            else :
                local_path /= repo_name

        execute_command_p = partial(execute_command, repo_path=local_path)

        if not local_path.exists():
            # Clone the repository if it does not exist
            is_success, _, error = await execute_command(['git', 'clone', repo_url, str(local_path)], local_path.parent)
            if not is_success:
                return False, error
            # Switch to the specified branch
            if branch:
                is_success, _, error = await execute_command_p(['git', 'checkout', branch])
                if not is_success:
                    return False, error
        else:
            # Get the default branch if not specified
            if not branch:
                is_success, output, error = await execute_command_p(['git', 'remote', 'show', 'origin'])
                if not is_success:
                    return False, error
                for line in output.splitlines():
                    if 'HEAD branch' in line:
                        branch = line.split(':')[-1].strip()
                        break

            # Stash local changes
            await execute_command_p(['git', 'stash'])
            # Switch to the specified branch
            is_success, _, error = await execute_command_p(['git', 'checkout', branch])
            if not is_success:
                return False, error

            # Fetch the latest changes from the remote repository
            is_success, _, error = await execute_command_p(['git', 'fetch'])
            if not is_success:
                return False, error
            # Check if the current commit is the same as the remote commit
            _, current_commit, _ = await execute_command_p(['git', 'rev-parse', 'HEAD'])
            _, remote_commit, _ = await execute_command_p(['git', 'rev-parse', f'origin/{branch}'])
            if current_commit == remote_commit:
                await execute_command_p(['git', 'stash', 'pop'])
                self.status = 'standby'
                logger.info(f'{self.ist_config.name}: {repo_name} already up to date')
                return True, None

            # Pull the latest changes
            is_success, _, error = await execute_command_p(['git', 'merge', 'origin', branch])
            if not is_success:
                return False, error
            # Apply stashed changes if any
            await execute_command_p(['git', 'stash', 'pop'])

        self.status = 'standby'
        logger.info(f'{self.ist_config.name}-Update finished successfully')
        return True, None

    # Inspired by https://github.com/zauberzeug/nicegui/blob/main/examples/script_executor/main.py
    async def run_command(self, task_name: str, command: str) -> Optional[Exception]:
        """Run a command in the background, not wait for stop."""
        work_dir = Path(self.ist_config.work_dir).resolve()
        try:
            if "win" in sys.platform.lower():
                self.process = await asyncio.create_subprocess_exec(
                    *shlex.split(command, posix=False),
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.STDOUT,
                    cwd=work_dir,
                    creationflags=0x08000000  # CREATE_NO_WINDOW
                )
            else:
                self.process = await asyncio.create_subprocess_exec(
                    *shlex.split(command, posix=True),
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.STDOUT,
                    cwd=work_dir
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
