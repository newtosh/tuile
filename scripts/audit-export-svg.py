#!/usr/bin/env python3
"""Audit a browser-exported SVG for full-bleed raster alignment."""

from __future__ import annotations

import re
import subprocess
import sys
import tempfile
from base64 import b64decode
from pathlib import Path

try:
    from PIL import Image
except ImportError as exc:  # pragma: no cover
    raise SystemExit("Pillow required for svg pixel audit") from exc


def parse_svg(path: Path) -> tuple[int, int, bytes, str]:
    text = path.read_text()
    root = re.search(r'width="(\d+)"[^>]*height="(\d+)"[^>]*viewBox="0 0 (\d+) (\d+)"', text)
    if not root:
        raise SystemExit(f"{path}: missing svg root dimensions")
    width, height, vbw, vbh = map(int, root.groups())
    if (width, height) != (vbw, vbh):
        raise SystemExit(f"{path}: width/height {width}x{height} != viewBox {vbw}x{vbh}")
    image = re.search(r'href="data:image/(png|jpeg);base64,([^"]+)"', text)
    if not image:
        raise SystemExit(f"{path}: missing embedded raster")
    mime, b64 = image.group(1), image.group(2)
    raw = b64decode(b64)
    if len(raw) < 4:
        raise SystemExit(f"{path}: embedded raster too small")
    if mime == "png":
        if raw[1:4] != b"PNG":
            raise SystemExit(f"{path}: invalid png signature")
        pw, ph = int.from_bytes(raw[16:20], "big"), int.from_bytes(raw[20:24], "big")
        if pw < width or ph < height:
            raise SystemExit(f"{path}: embedded png {pw}x{ph} smaller than svg {width}x{height}")
        if pw % width or ph % height:
            raise SystemExit(f"{path}: embedded png {pw}x{ph} not a multiple of svg {width}x{height}")
    return width, height, raw, mime


def sample(img: Image.Image, x: int, y: int) -> tuple[int, int, int, int]:
    x = max(0, min(img.width - 1, x))
    y = max(0, min(img.height - 1, y))
    return img.getpixel((x, y))


def audit(path: Path) -> None:
    width, height, raw, mime = parse_svg(path)
    embedded = Image.open(__import__("io").BytesIO(raw)).convert("RGBA")
    if embedded.size != (width, height):
        raise SystemExit(f"{path}: embedded {mime} is {embedded.size[0]}x{embedded.size[1]}, expected {width}x{height}")

    with tempfile.NamedTemporaryFile(suffix=".svg") as tmp:
        tmp.write(path.read_bytes())
        tmp.flush()
        rendered = subprocess.check_output(
            ["rsvg-convert", "-w", str(width), tmp.name],
            stderr=subprocess.DEVNULL,
        )
    raster = Image.open(__import__("io").BytesIO(rendered)).convert("RGBA")

    if raster.size != (width, height):
        raise SystemExit(f"{path}: rsvg render size {raster.size} != {width}x{height}")

    probes = {
        "top": (width // 2, 2),
        "left": (2, height // 2),
        "right": (width - 3, height // 2),
        "bottom": (width // 2, height - 3),
        "center": (width // 2, height // 2),
    }
    for name, (x, y) in probes.items():
        r, g, b, a = sample(raster, x, y)
        if a < 200:
            raise SystemExit(f"{path}: {name} probe at {(x, y)} is too transparent alpha={a}")
        if max(r, g, b) > 245:
            raise SystemExit(f"{path}: {name} probe at {(x, y)} looks empty/unpainted rgb={(r, g, b)}")

    print(f"OK {path.name}: {width}x{height} full-bleed {mime} raster")


def main(argv: list[str]) -> int:
    if len(argv) < 2:
        print("usage: audit-export-svg.py FILE.svg [FILE2.svg ...]", file=sys.stderr)
        return 2
    for arg in argv[1:]:
        audit(Path(arg))
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
