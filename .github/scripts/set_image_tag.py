#!/usr/bin/env python3
"""Set image.tag in a chart's values.yaml, preserving comments and alignment.

Called by image.yml's build-and-push-image job after a successful build, to
bump this repo's own chart/values.yaml in the same commit -- the same-repo
equivalent of vikingvault-infrastructure's copy of this script, which the
old cross-repo bump used before the chart moved into this repo
(2026-08-24, journal-1 cloud-native migration).

Only the FIRST `tag:` inside the top-level `image:` block is touched. A plain
sed would also be safe on today's file, but not once the chart grows a
second `tag:` key -- it already carries an unrelated
`timescaledb: image: timescale/timescaledb:...` further down.

    set_image_tag.py chart/values.yaml priority-channel-4ef51c7
    -> prints "changed priority-channel-7da14d6 priority-channel-4ef51c7"
       or "unchanged priority-channel-4ef51c7"
"""
from __future__ import annotations

import re
import sys
from pathlib import Path


def main() -> int:
    if len(sys.argv) != 3:
        print(__doc__, file=sys.stderr)
        return 2
    path, new = Path(sys.argv[1]), sys.argv[2]

    text = path.read_text()
    out, in_image, done, old = [], False, False, None

    for line in text.splitlines(keepends=True):
        stripped = line.rstrip("\n")
        if re.match(r"^image:\s*$", stripped):
            in_image = True
        elif in_image and stripped and not stripped[0].isspace():
            in_image = False          # a non-indented line ends the block
        if in_image and not done:
            if m := re.match(r"^(\s+tag:\s*)(\S+)(.*)$", stripped):
                old = m.group(2)
                line = f"{m.group(1)}{new}{m.group(3)}\n"
                done = True
        out.append(line)

    if not done:
        print(f"no image.tag found in {path}", file=sys.stderr)
        return 1
    if old == new:
        print(f"unchanged {old}")
        return 0

    path.write_text("".join(out))
    print(f"changed {old} {new}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
