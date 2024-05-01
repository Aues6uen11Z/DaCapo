import json
from os import PathLike
from pathlib import Path


def read_json(path: PathLike):
    with open(path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    return data


def write_json(path: PathLike, data: dict):
    with open(path, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False)


def instance_list() -> list[str]:
    path = Path('./config')
    return [p.stem for p in path.glob('*.json')]
