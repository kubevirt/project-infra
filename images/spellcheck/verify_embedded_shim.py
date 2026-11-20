#!/usr/bin/env python3
"""Verify base64-embedded yaspeller shim in Dockerfiles matches images/spellcheck/yaspeller."""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
CANON = REPO / "images/spellcheck/yaspeller"


def extract_b64_from_dockerfile(path: Path) -> str:
    text = path.read_text(encoding="utf-8")
    # Match printf '%s' then one or more single-quoted fragments with backslash continuation
    m = re.search(
        r"printf '%s'[ \t]*\\[ \t]*\n((?:[ \t]*'[^']*'[ \t]*\\\n)+)",
        text,
    )
    if not m:
        raise ValueError(f"no printf block in {path}")
    block = m.group(1)
    parts = re.findall(r"'([^']*)'", block)
    return "".join(parts)


def decode_matches_canonical(path: Path) -> bool:
    b64 = extract_b64_from_dockerfile(path)
    got = subprocess.check_output(
        ["base64", "-d"],
        input=b64.encode("ascii"),
    )
    return got == CANON.read_bytes()


def main() -> int:
    files = [
        REPO / "images/kubevirt-user-guide/Dockerfile",
        REPO / "images/kubevirt-kubevirt.github.io/Dockerfile",
        REPO / "images/yaspeller/Dockerfile",
    ]
    for f in files:
        if not decode_matches_canonical(f):
            print(f"FAIL embedded shim mismatch: {f}", file=sys.stderr)
            return 1
        print(f"ok {f.relative_to(REPO)}")
    print("all embedded shims match canonical images/spellcheck/yaspeller")
    return 0


if __name__ == "__main__":
    sys.exit(main())
