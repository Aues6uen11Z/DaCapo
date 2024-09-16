import asyncio
import copy
from functools import partial
from pathlib import Path
import shutil
import time
from typing import Optional

import anyconfig
from loguru import logger

from src.core.config import InstanceConfig, TemplateConfig
from src.utils import TMP_FLAG, CmdFailError, LayoutOutdatedError, write_config


class Updater:
    def __init__(self, ist_config: InstanceConfig):
        self.ist_config = ist_config
        # Git repository
        local_git = Path('./tools/Git/cmd/git.exe')
        self.git_exec = str(local_git.resolve()) if local_git.exists() else 'git'
        self.ist_name = ist_config.name
        self.repo_url = ist_config.repo_url
        self.branch = ist_config.branch

        self.local_path = Path(ist_config.local_path).resolve()
        self.repo_name = self.repo_url.rstrip('/').split('/')[-1].replace('.git', '')
        if self.local_path.name != self.repo_name and self.local_path.is_dir():
            self.local_path /= self.repo_name

        self.run_command_p = partial(self.run_command, work_dir=self.local_path)

        # Python virtual environment
        local_python = Path('./tools/Python/python.exe')
        self.python_exec = str(local_python.resolve()) if local_python.exists() else 'python'
        self.env_name = ist_config.env_name
        self.pip_mirror = ist_config.pip_mirror
        self.env_last_update = ist_config.env_last_update

    async def run_command(self, command: list, work_dir: Path) -> str:
        try:
            process = await asyncio.create_subprocess_exec(
                *command,
                cwd=work_dir,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                creationflags=0x08000000
            )
            logger.info(f'{self.ist_name}-Update started with command: {" ".join(command)}')

            stdout, stderr = await process.communicate()
            output = stdout.decode().strip()
            error = stderr.decode().strip()
        except Exception as e:
            logger.error(f'{self.ist_name}: {" ".join(command)}: {e}')
            raise

        if process.returncode == 0:
            return output
        else:
            logger.error(f'{self.ist_name}: {" ".join(command)}: {error}')
            raise CmdFailError(error)

    async def clone(self) -> None:
        """Clone the repository if it does not exist."""

        await self.run_command([self.git_exec, 'clone', self.repo_url], self.local_path.parent)
        # Switch to the specified branch
        if self.branch:
            await self.run_command_p([self.git_exec, 'checkout', self.branch])

    async def pull(self) -> None:
        """Pull the latest changes from the remote repository."""

        # Get the default branch if not specified
        if not self.branch:
            output = await self.run_command_p([self.git_exec, 'remote', 'show', 'origin'])
            for line in output.splitlines():
                if 'HEAD branch' in line:
                    self.branch = line.split(':')[-1].strip()
                    break

        await self.run_command_p([self.git_exec, 'stash'])
        await self.run_command_p([self.git_exec, 'checkout', self.branch])
        await self.run_command_p([self.git_exec, 'fetch'])
        # Check if the current commit is the same as the remote commit
        current_commit = await self.run_command_p([self.git_exec, 'rev-parse', 'HEAD'])
        remote_commit = await self.run_command_p([self.git_exec, 'rev-parse', f'origin/{self.branch}'])
        if current_commit == remote_commit:
            try:
                await self.run_command_p([self.git_exec, 'stash', 'pop'])
            except CmdFailError:
                pass
            logger.info(f'{self.ist_name}: {self.repo_name} already up to date')
            return

        await self.run_command_p([self.git_exec, 'merge', 'origin', self.branch])
        try:
            await self.run_command_p([self.git_exec, 'stash', 'pop'])
        except CmdFailError:
            pass

    async def create_venv(self) -> None:
        env_path = Path('./envs') / self.env_name
        env_path.parent.mkdir(exist_ok=True)
        if not env_path.exists():
            if self.python_exec == 'python':
                await self.run_command([self.python_exec, '-m', 'venv', self.env_name], env_path.parent)
            else:
                # Python embed can not use venv module
                shutil.copytree(Path('./tools/Python'), env_path)
            logger.info(f'{self.ist_name}: Create virtual environment: {self.env_name}')

    async def install_deps(self) -> None:
        requirements_path = self.local_path / 'requirements.txt'
        if not requirements_path.exists():
            logger.warning(f'{self.ist_name}: requirements.txt not found in {self.local_path}')
            raise FileNotFoundError(f'requirements.txt not found in {self.local_path}')

        last_modified = requirements_path.stat().st_mtime
        if last_modified < self.env_last_update:
            logger.info(f'{self.ist_name}: python dependencies already up to date')
            return
        if self.python_exec == 'python':
            venv_python = str((Path('./envs') / self.env_name / 'Scripts/python.exe').resolve())
        else:
            venv_python = str((Path('./envs') / self.env_name / 'python.exe').resolve())
        await self.run_command_p([venv_python, '-m', 'pip', 'install', '-r', 'requirements.txt', '-i', self.pip_mirror])
        self.ist_config.update_config('env_last_update', time.time())

    def load_template(self) -> None:
        """
        Copy template files from the repository and create instance configuration,
        similar to add_new_instance() in Settings.
        """

        source_path = self.local_path / self.ist_config.template_path
        target_path = Path('./config/templates') / self.repo_name
        if target_path.exists():
            shutil.rmtree(target_path)
        target_path.mkdir()

        tpl_config = TemplateConfig(self.repo_name)
        args_path = tpl_config.find_path_by_stem(source_path, 'args')
        if not args_path:
            logger.error(f'{self.ist_name}: Layout args not found in {source_path}')
            raise FileNotFoundError(f'Layout args not found in {source_path}')

        # Copy template files
        for file in [args_path.name, 'i18n']:
            if not (source_path / file).exists():
                continue
            if (source_path / file).is_dir():
                shutil.copytree(source_path / file, target_path / file)
            else:
                shutil.copy2(source_path / file, target_path / file)
        self.ist_config.update_config('layout_last_update', time.time())

        # Merge the newly created instance configuration with the old one
        old_config = dict(copy.deepcopy(self.ist_config.storage))
        del old_config['_info']['template']
        del old_config['_info']['language']
        new_config = tpl_config.add_instance(self.ist_name, True)  # File name has a flag suffix
        anyconfig.merge(new_config, old_config)
        write_config(new_config, self.ist_config.path.with_name(f'{self.ist_name}{TMP_FLAG}.json'))
        
        raise LayoutOutdatedError

    def layout_outdated(self) -> bool:
        """Check if local layout args need to be updated."""
        tpl_path = self.local_path / self.ist_config.template_path
        args_path = TemplateConfig(self.repo_name).find_path_by_stem(tpl_path, 'args')
        if not args_path:
            logger.error(f'{self.ist_name}: Layout args not found in {tpl_path}')
            raise FileNotFoundError(f'Layout args not found in {tpl_path}')

        return args_path.stat().st_mtime > self.ist_config.layout_last_update

    async def update(self) -> Optional[Exception]:
        try:
            if not self.local_path.exists():
                await self.clone()
            else:
                await self.pull()
            logger.info(f'{self.ist_name}: Git repository update completed')

            if self.env_name:
                await self.create_venv()
                await self.install_deps()
                logger.info(f'{self.ist_name}: Python environment update completed')

            if self.ist_config.template == 'Init':
                logger.info(f'{self.ist_name}: Init from empty template')
                self.load_template()
            elif self.layout_outdated():
                logger.info(f'{self.ist_name}: Layout args outdated')
                self.load_template()
        except LayoutOutdatedError as e:
            logger.info(f'{self.ist_name}-Update finished successfully, need to restart')
            return e
        except Exception as e:
            return e

        logger.info(f'{self.ist_name}-Update finished successfully')
        return None
