#!/usr/bin/env python3
"""Check that code shown in the docs matches the fixtures the examples run.

A code block in README.md or docs/ that follows a comment like

    <!-- fixture: ../fixtures/download/basic.star -->

must have exactly that file's content (the path is relative to the page).
Run from the repository root: python3 docs/check_sources.py
"""

import difflib
import glob
import os
import re
import sys

BLOCK = re.compile(r"<!-- fixture: (\S+) -->\n```[^\n]*\n(.*?)^```$", re.S | re.M)


def pages():
    yield "README.md"
    for path in sorted(glob.glob("docs/**/*.md", recursive=True)):
        if not path.startswith(os.path.join("docs", "fixtures") + os.sep):
            yield path


def main():
    checked, failures = 0, []
    for page in pages():
        text = open(page, encoding="utf-8").read()
        for m in BLOCK.finditer(text):
            line = text.count("\n", 0, m.start()) + 1
            rel, shown = m.group(1), m.group(2)
            fixture = os.path.normpath(os.path.join(os.path.dirname(page), rel))
            checked += 1
            if not os.path.isfile(fixture):
                failures.append(f"{page}:{line}: {fixture} does not exist")
                continue
            actual = open(fixture, encoding="utf-8").read()
            if actual != shown:
                diff = "".join(difflib.unified_diff(
                    actual.splitlines(True), shown.splitlines(True),
                    fixture, f"{page}:{line}"))
                failures.append(f"{page}:{line}: doesn't match {fixture}\n{diff}")
    for f in failures:
        print(f, file=sys.stderr)
    print(f"{checked} fixture blocks checked, {len(failures)} mismatched")
    return 1 if failures or checked == 0 else 0


if __name__ == "__main__":
    sys.exit(main())
