import sys
import os
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SRC = ROOT / "src"

os.environ.setdefault(
    "DB_FILE",
    str(Path(tempfile.gettempdir()) / f"comics_test_{os.getpid()}.db"),
)

for path in (ROOT, SRC):
    path_str = str(path)
    if path_str not in sys.path:
        sys.path.insert(0, path_str)
