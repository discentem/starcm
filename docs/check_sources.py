#!/usr/bin/env python3
"""Check that code shown in the docs matches the files the examples run.

A code block in README.md, docs/ or integration/ whose first line names a
file in the repository shows that file, and must match it exactly:

    ```python
    # examples/download/basic.star
    ...the content of examples/download/basic.star...
    ```

For templates the first line is a Jinja comment, {# examples/... #}. For
files that can't hold a comment (JSON), a label line right before the block
names the file instead: `examples/read_file/settings.json`:

Paths are relative to the repository root. Run from there:
python3 docs/check_sources.py
"""

import difflib
import glob
import os
import re
import sys

# A path the docs show files from.
PATH = r"(?:examples|integration)/\S+"
# ```lang, then a first line naming the file, then the content.
COMMENT_BLOCK = re.compile(r"^```[^\n]*\n(?:# (" + PATH + r")|\{# (" + PATH + r") #\})\n(.*?)^```$", re.S | re.M)
# `path`: on its own line, a blank line, then the block.
LABEL_BLOCK = re.compile(r"^`(" + PATH + r")`:\n\n```[^\n]*\n(.*?)^```$", re.S | re.M)


def pages():
    yield "README.md"
    for top in ("docs", "integration"):
        for path in sorted(glob.glob(top + "/**/*.md", recursive=True)):
            if os.sep + "fixtures" + os.sep not in path:
                yield path


def blocks(text):
    """Yields (offset, path, shown content) for every block naming a file."""
    for m in COMMENT_BLOCK.finditer(text):
        yield m.start(), m.group(1) or m.group(2), m.group(3)
    for m in LABEL_BLOCK.finditer(text):
        yield m.start(), m.group(1), m.group(2)


def main():
    checked, failures = 0, []
    for page in pages():
        text = open(page, encoding="utf-8").read()
        for offset, path, shown in blocks(text):
            line = text.count("\n", 0, offset) + 1
            checked += 1
            if not os.path.isfile(path):
                failures.append(f"{page}:{line}: {path} does not exist")
                continue
            actual = open(path, encoding="utf-8").read()
            if actual != shown:
                diff = "".join(difflib.unified_diff(
                    actual.splitlines(True), shown.splitlines(True),
                    path, f"{page}:{line}"))
                failures.append(f"{page}:{line}: doesn't match {path}\n{diff}")
    for f in failures:
        print(f, file=sys.stderr)
    print(f"{checked} example blocks checked, {len(failures)} mismatched")
    return 1 if failures or checked == 0 else 0


if __name__ == "__main__":
    sys.exit(main())
