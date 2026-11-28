#!/usr/bin/env python3
"""Lightweight tests for images/spellcheck/yaspeller (run with python3)."""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path


def test_typo_detected(script: Path) -> None:
    with tempfile.TemporaryDirectory() as td:
        root = Path(td)
        (root / "a.md").write_text("This has a mispelllll token.\n", encoding="utf-8")
        cfg = {
            "byWords": True,
            "ignoreUrls": True,
            "findRepeatWords": False,
            "ignoreDigits": True,
            "lang": "en",
            "fileExtensions": [".md"],
            "excludeFiles": [],
            "dictionary": [],
        }
        ycfg = root / "yaspeller.json"
        ycfg.write_text(json.dumps(cfg), encoding="utf-8")
        proc = subprocess.run(
            [sys.executable, str(script), "-c", str(ycfg), str(root)],
            capture_output=True,
            text=True,
        )
        assert proc.returncode == 1, (proc.stdout, proc.stderr)
        assert "✗" in proc.stdout, proc.stdout


def test_regex_dictionary_allows_vm(script: Path) -> None:
    with tempfile.TemporaryDirectory() as td:
        root = Path(td)
        (root / "a.md").write_text("VirtualMachine is ok.\n", encoding="utf-8")
        cfg = {
            "byWords": True,
            "ignoreUrls": True,
            "findRepeatWords": False,
            "ignoreDigits": True,
            "lang": "en",
            "fileExtensions": [".md"],
            "excludeFiles": [],
            "dictionary": ["(VM|vm)(s|’s|'s)?"],
        }
        ycfg = root / "yaspeller.json"
        ycfg.write_text(json.dumps(cfg), encoding="utf-8")
        proc = subprocess.run(
            [sys.executable, str(script), "-c", str(ycfg), str(root)],
            capture_output=True,
            text=True,
        )
        assert proc.returncode == 0, (proc.stdout, proc.stderr)


def test_exclude_files(script: Path) -> None:
    with tempfile.TemporaryDirectory() as td:
        root = Path(td)
        excl = root / "excluded"
        excl.mkdir()
        (excl / "x.md").write_text("mispelllll here\n", encoding="utf-8")
        (root / "good.md").write_text("Plain.\n", encoding="utf-8")
        cfg = {
            "byWords": True,
            "ignoreUrls": True,
            "findRepeatWords": False,
            "ignoreDigits": True,
            "lang": "en",
            "fileExtensions": [".md"],
            "excludeFiles": [str(excl)],
            "dictionary": [],
        }
        ycfg = root / "yaspeller.json"
        ycfg.write_text(json.dumps(cfg), encoding="utf-8")
        proc = subprocess.run(
            [sys.executable, str(script), "-c", str(ycfg), str(root)],
            capture_output=True,
            text=True,
        )
        assert proc.returncode == 0, (proc.stdout, proc.stderr)


def test_repeat_word(script: Path) -> None:
    with tempfile.TemporaryDirectory() as td:
        root = Path(td)
        (root / "a.md").write_text("The the sentence.\n", encoding="utf-8")
        cfg = {
            "byWords": True,
            "ignoreUrls": True,
            "findRepeatWords": True,
            "ignoreDigits": True,
            "lang": "en",
            "fileExtensions": [".md"],
            "excludeFiles": [],
            "dictionary": [],
        }
        ycfg = root / "yaspeller.json"
        ycfg.write_text(json.dumps(cfg), encoding="utf-8")
        proc = subprocess.run(
            [sys.executable, str(script), "-c", str(ycfg), str(root)],
            capture_output=True,
            text=True,
        )
        assert proc.returncode == 1, (proc.stdout, proc.stderr)
        assert "Repeated word" in proc.stdout, proc.stdout


def main() -> int:
    repo_root = Path(__file__).resolve().parents[2]
    script = repo_root / "images" / "spellcheck" / "yaspeller"
    if not script.is_file():
        print(f"missing {script}", file=sys.stderr)
        return 2
    if not shutil.which("cspell"):
        print(
            "cspell not on PATH; install with: npm install -g cspell@10.4.0",
            file=sys.stderr,
        )
        return 2
    tests = [test_typo_detected, test_regex_dictionary_allows_vm, test_exclude_files, test_repeat_word]
    for t in tests:
        try:
            t(script)
        except AssertionError as e:
            print(f"FAIL {t.__name__}: {e}", file=sys.stderr)
            return 1
        print(f"ok {t.__name__}")
    print("all tests passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
