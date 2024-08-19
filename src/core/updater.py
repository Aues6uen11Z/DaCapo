import asyncio
from functools import partial
from pathlib import Path
from typing import Optional

from loguru import logger

from src.core.config import InstanceConfig


class CmdFailError(Exception):
    pass


class Updater:
    def __init__(self, ist_config: InstanceConfig):
        self.ist_name = ist_config.name
        self.repo_url = ist_config.repo_url
        self.branch = ist_config.branch

        self.local_path = Path(ist_config.local_path).resolve()
        self.repo_name = self.repo_url.rstrip('/').split('/')[-1].replace('.git', '')
        if self.local_path.name != self.repo_name and self.local_path.is_dir():
            self.local_path /= self.repo_name

        self.run_command_p = partial(self.run_command, work_dir=self.local_path)

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
            logger.error(f'{self.ist_name}-{" ".join(command)}: {e}')
            raise

        if process.returncode == 0:
            return output
        else:
            logger.error(f'{self.ist_name}-{" ".join(command)}: {error}')
            raise CmdFailError(error)

    async def clone(self) -> None:
        """Clone the repository if it does not exist."""

        await self.run_command(['git', 'clone', self.repo_url, str(self.local_path)], self.local_path.parent)
        # Switch to the specified branch
        if self.branch:
            await self.run_command_p(['git', 'checkout', self.branch])

    async def pull(self) -> None:
        """Pull the latest changes from the remote repository."""

        # Get the default branch if not specified
        if not self.branch:
            output = await self.run_command_p(['git', 'remote', 'show', 'origin'])
            for line in output.splitlines():
                if 'HEAD branch' in line:
                    self.branch = line.split(':')[-1].strip()
                    break

        await self.run_command_p(['git', 'stash'])
        await self.run_command_p(['git', 'checkout', self.branch])
        await self.run_command_p(['git', 'fetch'])
        # Check if the current commit is the same as the remote commit
        current_commit = await self.run_command_p(['git', 'rev-parse', 'HEAD'])
        remote_commit = await self.run_command_p(['git', 'rev-parse', f'origin/{self.branch}'])
        if current_commit == remote_commit:
            try:
                await self.run_command_p(['git', 'stash', 'pop'])
            except CmdFailError:
                pass
            logger.info(f'{self.ist_name}: {self.repo_name} already up to date')
            return

        await self.run_command_p(['git', 'merge', 'origin', self.branch])
        try:
            await self.run_command_p(['git', 'stash', 'pop'])
        except CmdFailError:
            pass

    async def update_repo(self) -> Optional[Exception]:
        try:
            if not self.local_path.exists():
                await self.clone()
            else:
                await self.pull()
        except Exception as e:
            return e

        logger.info(f'{self.ist_name}-Update finished successfully')
        return None
