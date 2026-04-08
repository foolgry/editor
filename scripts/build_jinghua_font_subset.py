#!/usr/bin/env python3
"""Build a GB2312-based WOFF2 subset for 京華老宋體.

This keeps the common simplified-Chinese glyph set for web delivery and lets
rare characters fall back to the next font in the CSS stack.
"""

from __future__ import annotations

import subprocess
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
SOURCE_FONT = ROOT / "frontend/fonts/京華老宋体v3.0.ttf"
OUTPUT_FONT = ROOT / "frontend/fonts/京華老宋体v3.0-gb2312.woff2"


def build_gb2312_charset() -> str:
    chars = []
    for high in range(0xA1, 0xF8):
        for low in range(0xA1, 0xFF):
            try:
                chars.append(bytes((high, low)).decode("gb2312"))
            except UnicodeDecodeError:
                continue
    return "".join(chars)


def main() -> int:
    if not SOURCE_FONT.exists():
        print(f"Source font not found: {SOURCE_FONT}", file=sys.stderr)
        return 1

    with tempfile.NamedTemporaryFile(
        mode="w", encoding="utf-8", suffix=".txt", delete=False
    ) as charset_file:
        charset_file.write(build_gb2312_charset())
        charset_path = Path(charset_file.name)

    try:
        cmd = [
            sys.executable,
            "-m",
            "fontTools.subset",
            str(SOURCE_FONT),
            f"--text-file={charset_path}",
            "--flavor=woff2",
            f"--output-file={OUTPUT_FONT}",
        ]
        subprocess.run(cmd, check=True)
    except subprocess.CalledProcessError as exc:
        print(
            "fontTools subsetting failed. Install dependencies with "
            "`python3 -m pip install --user fonttools brotli`.",
            file=sys.stderr,
        )
        return exc.returncode
    finally:
        charset_path.unlink(missing_ok=True)

    size_mb = OUTPUT_FONT.stat().st_size / (1024 * 1024)
    print(f"Built {OUTPUT_FONT} ({size_mb:.2f} MB)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
